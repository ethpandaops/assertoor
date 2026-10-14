package logger

import (
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type logDBWriter struct {
	logger *LogScope

	bufferSize uint64
	flushDelay time.Duration

	lastIdx        uint64
	bufIdx         uint64
	bufMtx         sync.Mutex
	buf            []*db.TaskLog
	flushScheduled bool
}

func newLogDBWriter(logger *LogScope, bufferSize uint64, flushDelay time.Duration) *logDBWriter {
	return &logDBWriter{
		logger:     logger,
		bufferSize: bufferSize,
		flushDelay: flushDelay,
		buf:        make([]*db.TaskLog, 0, bufferSize),
	}
}

func (lh *logDBWriter) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (lh *logDBWriter) Fire(entry *logrus.Entry) error {
	lh.bufMtx.Lock()
	defer lh.bufMtx.Unlock()

	if lh.bufIdx >= lh.bufferSize {
		lh.flushToDBLocked()
	}

	logIdx := lh.lastIdx + 1
	lh.lastIdx = logIdx

	taskLog := &db.TaskLog{
		RunID:      lh.logger.options.TestRunID,
		TaskID:     lh.logger.options.TaskID,
		LogIndex:   logIdx,
		LogTime:    entry.Time.UnixMilli(),
		LogLevel:   uint32(entry.Level),
		LogMessage: entry.Message,
	}

	if len(entry.Data) > 0 {
		fieldsYaml, err := yaml.Marshal(map[string]interface{}(entry.Data))
		if err == nil {
			taskLog.LogFields = string(fieldsYaml)
		}
	}

	lh.buf = append(lh.buf, taskLog)
	lh.bufIdx++

	lh.scheduleFlushLocked()

	return nil
}

// scheduleFlushLocked arranges for the buffer to be flushed after flushDelay.
// The caller must hold bufMtx. At most one delayed flush is pending at a time;
// a flush triggered earlier by the buffer filling up simply leaves the pending
// timer to find an empty buffer and do nothing.
func (lh *logDBWriter) scheduleFlushLocked() {
	if lh.flushScheduled {
		return
	}

	lh.flushScheduled = true

	delay := lh.flushDelay
	if delay <= 0 {
		delay = 2 * time.Second
	}

	go func() {
		time.Sleep(delay)

		lh.bufMtx.Lock()
		defer lh.bufMtx.Unlock()

		lh.flushScheduled = false
		lh.flushToDBLocked()
	}()
}

// flush writes any buffered entries to the database.
func (lh *logDBWriter) flush() {
	lh.bufMtx.Lock()
	defer lh.bufMtx.Unlock()

	lh.flushToDBLocked()
}

// flushToDBLocked writes the buffered entries to the database. The caller must
// hold bufMtx. The buffer is always cleared afterwards, even on error, so a
// persistent database failure cannot grow it without bound. Errors are reported
// through the parent logger, not this writer's own logger, whose db hook would
// re-enter Fire and deadlock on bufMtx.
func (lh *logDBWriter) flushToDBLocked() {
	if len(lh.buf) == 0 {
		return
	}

	err := lh.logger.options.Database.RunTransaction(func(tx *sqlx.Tx) error {
		for _, entry := range lh.buf {
			if err := lh.logger.options.Database.InsertTaskLog(tx, entry); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		lh.logFlushError(err)
	}

	lh.buf = lh.buf[:0]
	lh.bufIdx = 0
}

func (lh *logDBWriter) logFlushError(err error) {
	if lh.logger.parentLogger != nil {
		lh.logger.parentLogger.WithError(err).Error("failed to write task log entries to db")
		return
	}

	logrus.WithError(err).Error("failed to write task log entries to db")
}

func (lh *logDBWriter) getBufferEntries() []*db.TaskLog {
	lh.bufMtx.Lock()
	defer lh.bufMtx.Unlock()

	entries := make([]*db.TaskLog, lh.bufIdx)
	copy(entries, lh.buf)

	return entries
}

func (lh *logDBWriter) GetLogEntryCount() uint64 {
	return lh.lastIdx
}

func (lh *logDBWriter) GetLogEntries(from, limit uint64) []*db.TaskLog {
	bufEntries := lh.getBufferEntries()

	// If buffer covers the requested start position, serve directly from buffer
	if len(bufEntries) > 0 && bufEntries[0].LogIndex <= from {
		offset := from - bufEntries[0].LogIndex
		if offset >= uint64(len(bufEntries)) {
			return nil
		}

		bufEntries = bufEntries[offset:]

		if limit > 0 && uint64(len(bufEntries)) > limit {
			return bufEntries[:limit]
		}

		return bufEntries
	}

	// Need to query DB (buffer is empty or starts after `from`)
	dbEntries, err := lh.logger.options.Database.GetTaskLogs(
		lh.logger.options.TestRunID, lh.logger.options.TaskID, from, limit,
	)
	if err != nil {
		return nil
	}

	// If DB returned the full limit or no buffer entries to merge, return DB results
	if (limit > 0 && uint64(len(dbEntries)) >= limit) || len(bufEntries) == 0 {
		return dbEntries
	}

	// Merge DB and buffer entries, removing any overlap
	if len(dbEntries) > 0 && dbEntries[len(dbEntries)-1].LogIndex >= bufEntries[0].LogIndex {
		lastDBIndex := dbEntries[len(dbEntries)-1].LogIndex
		firstBufIndex := bufEntries[0].LogIndex
		overlap := lastDBIndex - firstBufIndex + 1

		if overlap >= uint64(len(bufEntries)) {
			bufEntries = nil
		} else {
			bufEntries = bufEntries[overlap:]
		}
	}

	if len(bufEntries) > 0 {
		dbEntries = append(dbEntries, bufEntries...)
	}

	if limit > 0 && uint64(len(dbEntries)) > limit {
		return dbEntries[:limit]
	}

	return dbEntries
}

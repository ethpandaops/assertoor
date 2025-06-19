package logger

import (
	"sync"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/db"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type logDBWriter struct {
	logger *LogScope

	bufferSize uint64
	flushDelay time.Duration

	lastIdx      uint64
	flushIdx     uint64
	bufIdx       uint64
	bufMtx       sync.Mutex
	buf          []*db.TaskLog
	flushMtx     sync.Mutex
	flushing     bool
	flushingChan chan bool
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
		lh.flushToDB()
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

	lh.flushDelayed()

	return nil
}

func (lh *logDBWriter) flushDelayed() {
	if lh.flushing {
		return
	}

	lh.flushing = true
	lh.flushingChan = make(chan bool)

	go func() {
		defer func() {
			lh.flushing = false
		}()

		select {
		case <-lh.flushingChan:
			lh.flushingChan = nil
			return
		case <-time.After(2 * time.Second):
		}

		lh.flushingChan = nil

		lh.bufMtx.Lock()
		defer lh.bufMtx.Unlock()
		lh.flushToDB()
	}()
}

func (lh *logDBWriter) flushToDB() {
	lh.flushMtx.Lock()
	defer func() {
		lh.flushMtx.Unlock()
	}()

	if flushingChan := lh.flushingChan; flushingChan != nil {
		close(flushingChan)
	}

	if len(lh.buf) == 0 {
		return
	}

	err := lh.logger.options.Database.RunTransaction(func(tx *sqlx.Tx) error {
		for _, entry := range lh.buf {
			err := lh.logger.options.Database.InsertTaskLog(tx, entry)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		lh.logger.logger.Errorf("failed to write log entries to db: %v", err)
		return
	}

	lh.buf = lh.buf[:0]
	lh.bufIdx = 0
	lh.flushIdx = lh.lastIdx
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

	if len(bufEntries) > 0 && bufEntries[0].LogIndex >= from {
		firstIdx := bufEntries[0].LogIndex
		if firstIdx > from {
			if uint64(len(bufEntries)) <= firstIdx-from {
				return nil
			}

			bufEntries = bufEntries[firstIdx-from:]
		}

		if limit == 0 || uint64(len(bufEntries)) <= limit {
			return bufEntries
		}

		return bufEntries[0:limit]
	}

	dbEntries, err := lh.logger.options.Database.GetTaskLogs(lh.logger.options.TestRunID, lh.logger.options.TaskID, from, limit)
	if err != nil {
		return nil
	}

	if uint64(len(dbEntries)) == limit || len(bufEntries) == 0 {
		return dbEntries
	}

	if len(dbEntries) > 0 && dbEntries[len(dbEntries)-1].LogIndex >= bufEntries[0].LogIndex {
		// remove overlapping entries
		lastDBIndex := dbEntries[len(dbEntries)-1].LogIndex
		firstBufIndex := bufEntries[0].LogIndex
		bufEntries = bufEntries[lastDBIndex-firstBufIndex+1:]
	}

	dbEntries = append(dbEntries, bufEntries...)
	if uint64(len(dbEntries)) > limit {
		return dbEntries[:limit]
	}

	return dbEntries
}

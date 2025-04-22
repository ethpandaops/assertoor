package logger

import (
	"sync"

	"github.com/noku-team/assertoor/pkg/coordinator/db"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type logMemBuffer struct {
	logger     *LogScope
	bufferSize uint64
	bufIdx     uint64
	bufMtx     sync.Mutex
	buf        []*db.TaskLog
	lastIdx    uint64
}

func newLogMemBuffer(logger *LogScope, bufferSize uint64) *logMemBuffer {
	return &logMemBuffer{
		logger:     logger,
		bufferSize: bufferSize,
		buf:        make([]*db.TaskLog, 0, bufferSize),
	}
}

func (lmb *logMemBuffer) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (lmb *logMemBuffer) Fire(entry *logrus.Entry) error {
	lmb.bufMtx.Lock()
	defer lmb.bufMtx.Unlock()

	logIdx := lmb.lastIdx + 1
	lmb.lastIdx = logIdx

	taskLog := &db.TaskLog{
		RunID:      lmb.logger.options.TestRunID,
		TaskID:     lmb.logger.options.TaskID,
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

	if lmb.bufIdx >= lmb.bufferSize {
		bufIdx := lmb.bufIdx % lmb.bufferSize
		lmb.buf[bufIdx] = taskLog
	} else {
		lmb.buf = append(lmb.buf, taskLog)
	}

	lmb.bufIdx++

	return nil
}

func (lmb *logMemBuffer) GetLogEntryCount() uint64 {
	return lmb.lastIdx
}

func (lmb *logMemBuffer) GetLogEntries(from, limit uint64) []*db.TaskLog {
	lmb.bufMtx.Lock()
	defer lmb.bufMtx.Unlock()

	var entries []*db.TaskLog

	if lmb.bufIdx >= lmb.bufferSize {
		entries = make([]*db.TaskLog, lmb.bufferSize)
		firstIdx := lmb.bufIdx % lmb.bufferSize

		copy(entries, lmb.buf[firstIdx:])
		copy(entries[lmb.bufferSize-firstIdx:], lmb.buf[0:firstIdx])
	} else {
		entries = make([]*db.TaskLog, lmb.bufIdx)
		copy(entries, lmb.buf)
	}

	if len(entries) == 0 {
		return entries
	}

	if entries[0].LogIndex < from {
		indexDiff := from - entries[0].LogIndex
		if indexDiff >= uint64(len(entries)) {
			return nil
		}

		entries = entries[indexDiff:]
	}

	if limit != 0 && uint64(len(entries)) > limit {
		entries = entries[0:limit]
	}

	return entries
}

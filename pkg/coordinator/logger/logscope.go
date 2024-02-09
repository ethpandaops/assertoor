package logger

import (
	"io"
	"sync"

	"github.com/sirupsen/logrus"
)

type LogScope struct {
	options      *ScopeOptions
	logger       *logrus.Logger
	parentLogger *logrus.Logger
	parentFields logrus.Fields

	bufIdx uint64
	bufMtx sync.Mutex
	buf    []*logrus.Entry
}

type ScopeOptions struct {
	Parent      logrus.FieldLogger
	HistorySize uint64
}

type logForwarder struct {
	logger *LogScope
}

type logHistory struct {
	logger *LogScope
}

func NewLogger(options *ScopeOptions) *LogScope {
	if options == nil {
		options = &ScopeOptions{}
	}

	logger := &LogScope{
		options: options,
		logger:  logrus.New(),
		buf:     []*logrus.Entry{},
	}

	logger.logger.SetOutput(io.Discard)

	if options.Parent != nil {
		tmpEntry := options.Parent.WithFields(logrus.Fields{})
		logger.parentLogger = tmpEntry.Logger
		logger.parentFields = tmpEntry.Data
		logger.logger.AddHook(&logForwarder{
			logger: logger,
		})
	}

	if options.HistorySize > 0 {
		logger.logger.AddHook(&logHistory{
			logger: logger,
		})
	}

	return logger
}

func (lf *logForwarder) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (lf *logForwarder) Fire(entry *logrus.Entry) error {
	entry2 := entry.Dup()
	entry2.Logger = lf.logger.parentLogger

	entry2.WithFields(lf.logger.parentFields).Log(entry.Level, entry.Message)

	return nil
}

func (lh *logHistory) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (lh *logHistory) Fire(entry *logrus.Entry) error {
	lh.logger.bufMtx.Lock()
	defer lh.logger.bufMtx.Unlock()

	if lh.logger.bufIdx >= lh.logger.options.HistorySize {
		bufIdx := lh.logger.bufIdx % lh.logger.options.HistorySize
		lh.logger.buf[bufIdx] = entry
	} else {
		lh.logger.buf = append(lh.logger.buf, entry)
	}

	lh.logger.bufIdx++

	return nil
}

func (ls *LogScope) GetLogger() *logrus.Logger {
	return ls.logger
}

func (ls *LogScope) GetLogEntries() []*logrus.Entry {
	ls.bufMtx.Lock()
	defer ls.bufMtx.Unlock()

	var entries []*logrus.Entry

	if ls.bufIdx >= ls.options.HistorySize {
		entries = make([]*logrus.Entry, ls.options.HistorySize)
		firstIdx := ls.bufIdx % ls.options.HistorySize

		copy(entries, ls.buf[firstIdx:])
		copy(entries[ls.options.HistorySize-firstIdx:], ls.buf[0:firstIdx])
	} else {
		entries = make([]*logrus.Entry, ls.bufIdx)
		copy(entries, ls.buf)
	}

	return entries
}

func (ls *LogScope) GetLogEntriesSince(since int64) []*logrus.Entry {
	entries := ls.GetLogEntries()

	for idx, entry := range entries {
		if entry.Time.UnixNano() > since {
			return entries[idx:]
		}
	}

	return []*logrus.Entry{}
}

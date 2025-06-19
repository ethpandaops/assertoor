package logger

import "github.com/erigontech/assertoor/pkg/coordinator/db"

type LogReader interface {
	GetLogEntries(from, limit uint64) []*db.TaskLog
	GetLogEntryCount() uint64
}

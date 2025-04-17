package logger

import "github.com/ethpandaops/assertoor/pkg/coordinator/db"

type LogReader interface {
	GetLogEntries(from, limit uint64) []*db.TaskLog
	GetLogEntryCount() uint64
}

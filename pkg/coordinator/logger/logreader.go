package logger

import "github.com/noku-team/assertoor/pkg/coordinator/db"

type LogReader interface {
	GetLogEntries(from, limit int) []*db.TaskLog
	GetLogEntryCount() int
}

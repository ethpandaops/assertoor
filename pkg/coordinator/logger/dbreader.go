package logger

import "github.com/erigontech/assertoor/pkg/coordinator/db"

type logDBReader struct {
	database  *db.Database
	testRunID uint64
	taskID    uint64
	lastIdx   *uint64
}

func NewLogDBReader(database *db.Database, testRunID, taskID uint64) LogReader {
	return &logDBReader{
		database:  database,
		testRunID: testRunID,
		taskID:    taskID,
	}
}

func (ls *logDBReader) GetLogEntryCount() uint64 {
	if ls.lastIdx == nil {
		lastIdx, err := ls.database.GetLastLogIndex(ls.testRunID, ls.taskID)
		if err != nil {
			return 0
		}

		ls.lastIdx = &lastIdx
	}

	if ls.lastIdx == nil {
		return 0
	}

	return *ls.lastIdx
}

func (ls *logDBReader) GetLogEntries(from, limit uint64) []*db.TaskLog {
	dbEntries, err := ls.database.GetTaskLogs(ls.testRunID, ls.taskID, from, limit)
	if err != nil {
		return nil
	}

	return dbEntries
}

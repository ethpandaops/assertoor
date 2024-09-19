package logger

import "github.com/ethpandaops/assertoor/pkg/coordinator/db"

type LogDBReader struct {
	database  *db.Database
	testRunID uint64
	taskID    uint64
	lastIdx   *int
}

func NewLogDBReader(database *db.Database, testRunID, taskID uint64) *LogDBReader {
	return &LogDBReader{
		database:  database,
		testRunID: testRunID,
		taskID:    taskID,
	}
}

func (ls *LogDBReader) GetLogEntryCount() int {
	if ls.lastIdx == nil {
		lastIdx, err := ls.database.GetLastLogIndex(int(ls.testRunID), int(ls.taskID))
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

func (ls *LogDBReader) GetLogEntries(from, limit int) []*db.TaskLog {
	dbEntries, err := ls.database.GetTaskLogs(int(ls.testRunID), int(ls.taskID), from, limit)
	if err != nil {
		return nil
	}

	return dbEntries
}

package logger

import "github.com/noku-team/assertoor/pkg/coordinator/db"

type logDBReader struct {
	database  *db.Database
	testRunID int
	taskID    int
	lastIdx   *int
}

func NewLogDBReader(database *db.Database, testRunID, taskID int) LogReader {
	return &logDBReader{
		database:  database,
		testRunID: testRunID,
		taskID:    taskID,
	}
}

func (ls *logDBReader) GetLogEntryCount() int {
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

func (ls *logDBReader) GetLogEntries(from, limit int) []*db.TaskLog {
	dbEntries, err := ls.database.GetTaskLogs(ls.testRunID, ls.taskID, from, limit)
	if err != nil {
		return nil
	}

	return dbEntries
}

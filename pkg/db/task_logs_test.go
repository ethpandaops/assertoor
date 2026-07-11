package db

import (
	"io"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func newSqliteTestDB(t *testing.T) *Database {
	t.Helper()

	quiet := logrus.New()
	quiet.SetOutput(io.Discard)

	database := NewDatabase(quiet)
	if err := database.InitDB(&DatabaseConfig{
		Engine: "sqlite",
		Sqlite: &SqliteDatabaseConfig{File: t.TempDir() + "/test.sqlite"},
	}); err != nil {
		t.Fatalf("init db: %v", err)
	}

	if err := database.ApplySchema(-2); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	return database
}

// TestInsertTaskLogPersistsAndUpserts verifies that a task log is persisted and
// that re-inserting the same primary key (run_id, task_id, log_idx) updates the
// row in place rather than duplicating or failing. The upsert conflict target
// must be the primary key; any other target is rejected by Postgres at execution
// time, which previously made every insert fail on that engine.
func TestInsertTaskLogPersistsAndUpserts(t *testing.T) {
	database := newSqliteTestDB(t)

	insert := func(l *TaskLog) {
		if err := database.RunTransaction(func(tx *sqlx.Tx) error {
			return database.InsertTaskLog(tx, l)
		}); err != nil {
			t.Fatalf("insert task log: %v", err)
		}
	}

	insert(&TaskLog{RunID: 1, TaskID: 1, LogIndex: 1, LogTime: 100, LogLevel: 1, LogMessage: "hello"})
	insert(&TaskLog{RunID: 1, TaskID: 1, LogIndex: 2, LogTime: 110, LogLevel: 1, LogMessage: "world"})

	logs, err := database.GetTaskLogs(1, 1, 0, 100)
	if err != nil {
		t.Fatalf("get task logs: %v", err)
	}

	if len(logs) != 2 {
		t.Fatalf("expected 2 persisted logs, got %d", len(logs))
	}

	// re-insert log index 1 with new content
	insert(&TaskLog{RunID: 1, TaskID: 1, LogIndex: 1, LogTime: 200, LogLevel: 2, LogMessage: "updated"})

	logs, err = database.GetTaskLogs(1, 1, 0, 100)
	if err != nil {
		t.Fatalf("get task logs: %v", err)
	}

	if len(logs) != 2 {
		t.Fatalf("after upsert: expected 2 logs, got %d", len(logs))
	}

	if logs[0].LogMessage != "updated" {
		t.Fatalf("after upsert: expected log index 1 to read \"updated\", got %q", logs[0].LogMessage)
	}
}

package logger

import (
	"io"
	"testing"
	"time"

	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/sirupsen/logrus"
)

func newSqliteDB(t *testing.T, applySchema bool) *db.Database {
	t.Helper()

	quiet := logrus.New()
	quiet.SetOutput(io.Discard)

	database := db.NewDatabase(quiet)
	if err := database.InitDB(&db.DatabaseConfig{
		Engine: "sqlite",
		Sqlite: &db.SqliteDatabaseConfig{File: t.TempDir() + "/logger.sqlite"},
	}); err != nil {
		t.Fatalf("init db: %v", err)
	}

	if applySchema {
		if err := database.ApplySchema(-2); err != nil {
			t.Fatalf("apply schema: %v", err)
		}
	}

	return database
}

// TestFlushDoesNotDeadlockOrGrowOnDBError drives the real logger with a database
// whose task_logs table does not exist, so every flush fails. Previously the flush
// error was logged through the writer's own logger, whose db hook re-entered Fire
// and deadlocked on the buffer mutex, and the buffer was never trimmed on error so
// it grew without bound. Both must be gone: logging must make progress and the
// buffer must stay bounded.
func TestFlushDoesNotDeadlockOrGrowOnDBError(t *testing.T) {
	quiet := logrus.New()
	quiet.SetOutput(io.Discard)

	database := newSqliteDB(t, false) // no schema -> every InsertTaskLog fails

	ls := NewLogger(&ScopeOptions{
		Parent:     quiet,
		Database:   database,
		BufferSize: 4,
		TestRunID:  1,
		TaskID:     1,
	})
	log := ls.GetLogger()

	done := make(chan struct{})

	go func() {
		for i := 0; i < 1000; i++ {
			log.Infof("entry %d", i)
		}

		close(done)
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("logging deadlocked against a failing database")
	}

	ls.dbWriter.bufMtx.Lock()
	buffered := len(ls.dbWriter.buf)
	ls.dbWriter.bufMtx.Unlock()

	if uint64(buffered) > ls.dbWriter.bufferSize {
		t.Fatalf("buffer grew to %d entries on persistent db error (bufferSize %d)", buffered, ls.dbWriter.bufferSize)
	}
}

// TestLogsPersistThroughWriter verifies the happy path still works after the flush
// rewrite: entries logged through the scope are written to the database on flush.
func TestLogsPersistThroughWriter(t *testing.T) {
	quiet := logrus.New()
	quiet.SetOutput(io.Discard)

	database := newSqliteDB(t, true)

	ls := NewLogger(&ScopeOptions{
		Parent:    quiet,
		Database:  database,
		TestRunID: 7,
		TaskID:    3,
	})
	log := ls.GetLogger()

	log.Info("first")
	log.Info("second")
	log.Info("third")

	ls.Flush()

	logs, err := database.GetTaskLogs(7, 3, 0, 100)
	if err != nil {
		t.Fatalf("get task logs: %v", err)
	}

	if len(logs) != 3 {
		t.Fatalf("expected 3 persisted logs, got %d", len(logs))
	}
}

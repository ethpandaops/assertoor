package db

import (
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func TestSqliteSharesDataAcrossConnections(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{name: "bare memory", file: ":memory:"},
		{name: "default memory", file: ":memory:?cache=shared"},
		{name: "memory URI", file: "file::memory:?cache=shared"},
		{name: "named memory URI", file: "file:assertoor-test?mode=memory&cache=shared"},
		{name: "file", file: filepath.Join(t.TempDir(), "assertoor.db")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := NewDatabase(logrus.New())
			if err := database.InitDB(&DatabaseConfig{
				Engine: "sqlite",
				Sqlite: &SqliteDatabaseConfig{File: test.file},
			}); err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				if err := database.CloseDB(); err != nil {
					t.Error(err)
				}
			})

			if err := database.ApplySchema(-2); err != nil {
				t.Fatal(err)
			}

			want := &TaskLog{RunID: 1, TaskID: 1, LogIndex: 1, LogMessage: "shared log"}

			if err := database.RunTransaction(func(tx *sqlx.Tx) error {
				return database.InsertTaskLog(tx, want)
			}); err != nil {
				t.Fatal(err)
			}

			conn, err := database.reader.Conn(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				if closeErr := conn.Close(); closeErr != nil {
					t.Error(closeErr)
				}
			})

			logs, err := database.GetTaskLogs(want.RunID, want.TaskID, 0, 1)
			if err != nil {
				t.Fatalf("read task logs on a second connection: %v", err)
			}

			if len(logs) != 1 || *logs[0] != *want {
				t.Fatalf("task logs = %v, want %v", logs, want)
			}
		})
	}
}

package db

import (
	"github.com/jmoiron/sqlx"
)

type TestRun struct {
	RunID     int    `db:"run_id"`
	TestID    string `db:"test_id"`
	Name      string `db:"name"`
	Source    string `db:"source"`
	Config    string `db:"config"`
	StartTime int64  `db:"start_time"`
	StopTime  int64  `db:"stop_time"`
	Status    string `db:"status"`
}

func (db *Database) InsertTestRun(tx *sqlx.Tx, run *TestRun) error {
	_, err := tx.Exec(db.EngineQuery(map[EngineType]string{
		EnginePgsql: `
			INSERT INTO test_runs (
				run_id, test_id, name, source, config, start_time, stop_time, status
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (run_id) DO UPDATE SET
				test_id = excluded.test_id,
				name = excluded.name,
				source = excluded.source,
				start_time = excluded.start_time,
				stop_time = excluded.stop_time,
				status = excluded.status`,
		EngineSqlite: `
			INSERT OR REPLACE INTO test_runs (
				run_id, test_id, name, source, config, start_time, stop_time, status
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
	}),
		run.RunID, run.TestID, run.Name, run.Source, run.Config, run.StartTime, run.StopTime, run.Status)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) UpdateTestRunStatus(tx *sqlx.Tx, run *TestRun) error {
	_, err := tx.Exec(`
			UPDATE test_runs
			SET status = $1, start_time = $2, stop_time = $3
			WHERE run_id = $4 AND test_id = $5`,
		run.Status, run.StartTime, run.StopTime, run.RunID, run.TestID)
	if err != nil {
		return err
	}

	return nil
}
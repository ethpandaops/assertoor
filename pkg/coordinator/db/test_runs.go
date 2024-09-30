package db

import (
	"fmt"
	"strings"

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
	Timeout   int32  `db:"timeout"`
	Status    string `db:"status"`
}

// InsertTestRun inserts a test run into the database.
func (db *Database) InsertTestRun(tx *sqlx.Tx, run *TestRun) error {
	_, err := tx.Exec(db.EngineQuery(map[EngineType]string{
		EnginePgsql: `
			INSERT INTO test_runs (
				run_id, test_id, name, source, config, start_time, stop_time, timeout, status
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (run_id) DO UPDATE SET
				test_id = excluded.test_id,
				name = excluded.name,
				source = excluded.source,
				start_time = excluded.start_time,
				stop_time = excluded.stop_time,
				timeout = excluded.timeout,
				status = excluded.status`,
		EngineSqlite: `
			INSERT OR REPLACE INTO test_runs (
				run_id, test_id, name, source, config, start_time, stop_time, timeout, status
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
	}),
		run.RunID, run.TestID, run.Name, run.Source, run.Config, run.StartTime, run.StopTime, run.Timeout, run.Status)
	if err != nil {
		return err
	}

	return nil
}

// UpdateTestRunStatus updates the status fields of a test run.
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

// GetTestRunByRunID returns a test run by run ID.
func (db *Database) GetTestRunByRunID(runID int) (*TestRun, error) {
	var run TestRun

	err := db.reader.Get(&run, `
		SELECT * FROM test_runs
		WHERE run_id = $1`,
		runID)
	if err != nil {
		return nil, err
	}

	return &run, nil
}

// GetTestRunRange returns a range of test runs.
func (db *Database) GetTestRunRange(testID string, firstRunID, offset, limit int) ([]*TestRun, int, error) {
	var runs []*TestRun

	var sql strings.Builder

	fmt.Fprint(&sql, `
	WITH cte AS (
		SELECT
			run_id, test_id, name, source, config, start_time, stop_time, timeout, status
		FROM test_runs
	`)

	args := []any{}
	whereGlue := "WHERE"

	if testID != "" {
		fmt.Fprintf(&sql, `%v test_id = $%v `, whereGlue, len(args)+1)
		args = append(args, testID)
		whereGlue = "AND"
	}

	if firstRunID > 0 {
		fmt.Fprintf(&sql, `%v run_id <= $%v `, whereGlue, len(args)+1)
		args = append(args, firstRunID)
	}

	fmt.Fprintf(&sql, `) 
	SELECT 
		count(*) AS run_id, 
		"" AS test_id, 
		"" AS name, 
		"" AS source, 
		"" AS config, 
		0 AS start_time, 
		0 AS stop_time, 
		0 AS timeout, 
		"" AS status
	FROM cte
	UNION ALL SELECT * FROM (
	SELECT * FROM cte
	ORDER BY run_id DESC 
	`)

	if limit > 0 {
		fmt.Fprintf(&sql, ` LIMIT $%v`, len(args)+1)
		args = append(args, limit)
	}

	if offset > 0 {
		fmt.Fprintf(&sql, ` OFFSET $%v`, len(args)+1)
		args = append(args, offset)
	}

	fmt.Fprintf(&sql, `)`)

	err := db.reader.Select(&runs, sql.String(), args...)
	if err != nil {
		return nil, 0, err
	}

	return runs[1:], runs[0].RunID, nil
}

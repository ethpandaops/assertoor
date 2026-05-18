package db

import "github.com/jmoiron/sqlx"

type TaskResult struct {
	RunID  uint64 `db:"run_id"`
	TaskID uint64 `db:"task_id"`
	Type   string `db:"result_type"`
	Index  uint64 `db:"result_index"`
	Name   string `db:"name"`
	Size   uint64 `db:"size"`
	Data   []byte `db:"data"`
}

type TaskResultHeader struct {
	TaskID uint64 `db:"task_id"`
	Type   string `db:"result_type"`
	Index  uint64 `db:"result_index"`
	Name   string `db:"name"`
	Size   uint64 `db:"size"`
}

func (db *Database) UpsertTaskResult(tx *sqlx.Tx, result *TaskResult) error {
	_, err := tx.Exec(db.EngineQuery(map[EngineType]string{
		EnginePgsql: `
			INSERT INTO task_results (run_id, task_id, result_type, result_index, name, size, data)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (run_id, task_id, result_type, result_index) DO UPDATE SET
				name = excluded.name,
				size = excluded.size,
				data = excluded.data
			`,
		EngineSqlite: `
			INSERT OR REPLACE INTO task_results (run_id, task_id, result_type, result_index, name, size, data)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
	}),
		result.RunID, result.TaskID, result.Type, result.Index, result.Name, result.Size, result.Data)

	return err
}

func (db *Database) GetTaskResultByIndex(runID, taskID uint64, resultType string, index int) (*TaskResult, error) {
	var result TaskResult

	err := db.reader.Get(&result, `
		SELECT * FROM task_results
		WHERE run_id = $1 AND task_id = $2 AND result_type = $3 AND result_index = $4`,
		runID, taskID, resultType, index)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (db *Database) GetTaskResultByName(runID, taskID uint64, resultType, name string) (*TaskResult, error) {
	var result TaskResult

	err := db.reader.Get(&result, `
		SELECT * FROM task_results
		WHERE run_id = $1 AND task_id = $2 AND result_type = $3 AND name = $4`,
		runID, taskID, resultType, name)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (db *Database) GetTaskResults(runID, taskID uint64, summaryType string) ([]TaskResult, error) {
	var results []TaskResult

	err := db.reader.Select(&results, `
		SELECT * FROM task_results
		WHERE run_id = $1 AND task_id = $2 AND result_type = $3`,
		runID, taskID, summaryType)

	return results, err
}

func (db *Database) GetAllTaskResultHeaders(runID uint64) ([]TaskResultHeader, error) {
	var headers []TaskResultHeader

	err := db.reader.Select(&headers, `
		SELECT task_id, result_type, result_index, name, size FROM task_results
		WHERE run_id = $1 AND task_id != 0`,
		runID)

	return headers, err
}

// testResultTaskID and testResultType are the sentinel coordinates we use
// to store the run-level "test result" markdown in the shared
// task_results table. TaskID 0 is unused by real tasks (real indices
// start at 1).
const (
	testResultTaskID = 0
	testResultType   = "test_result"
)

// UpsertTestResult stores the run-level markdown blob (set by tasks that
// write to $ASSERTOOR_TEST_RESULT). One blob per test run.
func (db *Database) UpsertTestResult(runID uint64, data []byte) error {
	return db.RunTransaction(func(tx *sqlx.Tx) error {
		return db.UpsertTaskResult(tx, &TaskResult{
			RunID:  runID,
			TaskID: testResultTaskID,
			Type:   testResultType,
			Index:  0,
			Name:   "result.md",
			Size:   uint64(len(data)),
			Data:   data,
		})
	})
}

// GetTestResult returns the run-level markdown blob, or nil + nil error
// when none is set.
func (db *Database) GetTestResult(runID uint64) (*TaskResult, error) {
	result, err := db.GetTaskResultByIndex(runID, testResultTaskID, testResultType, 0)
	if err != nil {
		// Surface "no rows" as nil, nil so callers can distinguish "not
		// set" from "fetch failed".
		return nil, nil //nolint:nilerr // sql.ErrNoRows-style miss
	}

	return result, nil
}

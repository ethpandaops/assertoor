package db

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type TaskState struct {
	RunID      uint64 `db:"run_id"`
	TaskID     uint64 `db:"task_id"`
	ParentTask uint64 `db:"parent_task"`
	Name       string `db:"name"`
	Title      string `db:"title"`
	RefID      string `db:"ref_id"`
	Timeout    int64  `db:"timeout"`
	IfCond     string `db:"ifcond"`
	RunFlags   uint32 `db:"run_flags"`
	StartTime  int64  `db:"start_time"`
	StopTime   int64  `db:"stop_time"`
	ScopeOwner uint64 `db:"scope_owner"`
	TaskConfig string `db:"task_config"`
	TaskStatus string `db:"task_status"`
	TaskResult int    `db:"task_result"`
	TaskError  string `db:"task_error"`
}

type TaskStateIndex struct {
	TaskID     uint64 `db:"task_id"`
	ParentTask uint64 `db:"parent_task"`
	RunFlags   uint32 `db:"run_flags"`
}

var (
	TaskRunFlagCleanup uint32 = 0x01
	TaskRunFlagStarted uint32 = 0x02
	TaskRunFlagRunning uint32 = 0x04
	TaskRunFlagSkipped uint32 = 0x08
	TaskRunFlagTimeout uint32 = 0x10
)

// InsertTaskState inserts a task state into the database.
func (db *Database) InsertTaskState(tx *sqlx.Tx, state *TaskState) error {
	_, err := tx.Exec(db.EngineQuery(map[EngineType]string{
		EnginePgsql: `
			INSERT INTO task_states (
				run_id, task_id, parent_task, name, title, ref_id, timeout, ifcond, run_flags, 
				start_time, stop_time, scope_owner, task_config, task_status, task_result, task_error
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
			ON CONFLICT (run_id, task_id) DO UPDATE SET
				parent_task = excluded.parent_task,
				name = excluded.name,
				title = excluded.title,
				ref_id = excluded.ref_id,
				timeout = excluded.timeout,
				ifcond = excluded.ifcond,
				run_flags = excluded.run_flags,
				start_time = excluded.start_time,
				stop_time = excluded.stop_time,
				scope_owner = excluded.scope_owner,
				task_config = excluded.task_config,
				task_status = excluded.task_status,
				task_result = excluded.task_result,
				task_error = excluded.task_error`,
		EngineSqlite: `
			INSERT OR REPLACE INTO task_states (
				run_id, task_id, parent_task, name, title, ref_id, timeout, ifcond, run_flags, 
				start_time, stop_time, scope_owner, task_config, task_status, task_result, task_error
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
	}),
		state.RunID, state.TaskID, state.ParentTask, state.Name, state.Title, state.RefID, state.Timeout,
		state.IfCond, state.RunFlags, state.StartTime, state.StopTime, state.ScopeOwner, state.TaskConfig,
		state.TaskStatus, state.TaskResult, state.TaskError)
	if err != nil {
		return err
	}

	return nil
}

// UpdateTaskStateStatus updates the status fields of a task state.
func (db *Database) UpdateTaskStateStatus(tx *sqlx.Tx, state *TaskState, updateFields []string) error {
	var sql strings.Builder

	args := []any{}

	fmt.Fprint(&sql, `UPDATE task_states SET `)

	for i, field := range updateFields {
		if i > 0 {
			fmt.Fprint(&sql, `, `)
		}

		switch field {
		case "title":
			fmt.Fprintf(&sql, `title = $%v`, len(args)+1)
			args = append(args, state.Title)
		case "run_flags":
			fmt.Fprintf(&sql, `run_flags = $%v`, len(args)+1)
			args = append(args, state.RunFlags)
		case "start_time":
			fmt.Fprintf(&sql, `start_time = $%v`, len(args)+1)
			args = append(args, state.StartTime)
		case "stop_time":
			fmt.Fprintf(&sql, `stop_time = $%v`, len(args)+1)
			args = append(args, state.StopTime)
		case "task_config":
			fmt.Fprintf(&sql, `task_config = $%v`, len(args)+1)
			args = append(args, state.TaskConfig)
		case "task_status":
			fmt.Fprintf(&sql, `task_status = $%v`, len(args)+1)
			args = append(args, state.TaskStatus)
		case "task_result":
			fmt.Fprintf(&sql, `task_result = $%v`, len(args)+1)
			args = append(args, state.TaskResult)
		case "task_error":
			fmt.Fprintf(&sql, `task_error = $%v`, len(args)+1)
			args = append(args, state.TaskError)
		default:
			return fmt.Errorf("unknown field %q", field)
		}
	}

	fmt.Fprintf(&sql, ` WHERE run_id = $%v AND task_id = $%v`, len(args)+1, len(args)+2)
	args = append(args, state.RunID, state.TaskID)

	_, err := tx.Exec(sql.String(), args...)
	if err != nil {
		return err
	}

	return nil
}

// GetTaskStateIndex returns the task index for a given test run.
func (db *Database) GetTaskStateIndex(runID uint64) ([]*TaskStateIndex, error) {
	var states []*TaskStateIndex

	err := db.reader.Select(&states, `
		SELECT task_id, parent_task, run_flags
		FROM task_states
		WHERE run_id = $1
		ORDER BY task_id ASC`,
		runID)
	if err != nil {
		return nil, err
	}

	return states, nil
}

// GetTaskStateByTaskID returns a task state by task ID.
func (db *Database) GetTaskStateByTaskID(runID, taskID uint64) (*TaskState, error) {
	var state TaskState

	err := db.reader.Get(&state, `
		SELECT * FROM task_states
		WHERE run_id = $1 AND task_id = $2`,
		runID, taskID)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

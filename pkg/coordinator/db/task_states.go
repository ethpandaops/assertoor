package db

import "github.com/jmoiron/sqlx"

/*
CREATE TABLE IF NOT EXISTS public."task_states"
(

	"run_id" INTEGER NOT NULL,
	"task_id" INTEGER NOT NULL,
	"options" TEXT NOT NULL,
	"parent_task" INTEGER NOT NULL,
	"is_cleanup" BOOLEAN NOT NULL,
	"is_started" BOOLEAN NOT NULL,
	"is_running" BOOLEAN NOT NULL,
	"is_skipped" BOOLEAN NOT NULL,
	"is_timeout" BOOLEAN NOT NULL,
	"start_time" BIGINT NOT NULL,
	"stop_time" BIGINT NOT NULL,
	"task_config" TEXT NOT NULL,
	"task_status" TEXT NOT NULL,
	"task_result" INTEGER NOT NULL,
	CONSTRAINT "task_states_pkey" PRIMARY KEY ("run_id", "task_id")

);
*/
type TaskState struct {
	RunID      int    `db:"run_id"`
	TaskID     int    `db:"task_id"`
	ParentTask int    `db:"parent_task"`
	Name       string `db:"name"`
	Title      string `db:"title"`
	Timeout    int    `db:"timeout"`
	IfCond     string `db:"ifcond"`
	IsCleanup  bool   `db:"is_cleanup"`
	IsStarted  bool   `db:"is_started"`
	IsRunning  bool   `db:"is_running"`
	IsSkipped  bool   `db:"is_skipped"`
	IsTimeout  bool   `db:"is_timeout"`
	StartTime  int64  `db:"start_time"`
	StopTime   int64  `db:"stop_time"`
	TaskConfig string `db:"task_config"`
	TaskStatus string `db:"task_status"`
	TaskResult int    `db:"task_result"`
}

func (db *Database) InsertTaskState(tx *sqlx.Tx, state *TaskState) error {
	_, err := tx.Exec(db.EngineQuery(map[EngineType]string{
		EnginePgsql: `
			INSERT INTO task_states (
				run_id, task_id, parent_task, name, title, timeout, ifcond, is_cleanup, is_started, is_running, is_skipped, is_timeout, 
				start_time, stop_time, task_config, task_status, task_result
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
			ON CONFLICT (run_id, task_id) DO UPDATE SET
				parent_task = excluded.parent_task,
				name = excluded.name,
				title = excluded.title,
				timeout = excluded.timeout,
				ifcond = excluded.ifcond,
				is_cleanup = excluded.is_cleanup,
				is_started = excluded.is_started,
				is_running = excluded.is_running,
				is_skipped = excluded.is_skipped,
				is_timeout = excluded.is_timeout,
				start_time = excluded.start_time,
				stop_time = excluded.stop_time,
				task_config = excluded.task_config,
				task_status = excluded.task_status,
				task_result = excluded.task_result`,
		EngineSqlite: `
			INSERT OR REPLACE INTO task_states (
				run_id, task_id, parent_task, name, title, timeout, ifcond, is_cleanup, is_started, is_running, is_skipped, is_timeout, 
				start_time, stop_time, task_config, task_status, task_result
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
	}),
		state.RunID, state.TaskID, state.ParentTask, state.Name, state.Title, state.Timeout, state.IfCond, state.IsCleanup, state.IsStarted, state.IsRunning,
		state.IsSkipped, state.IsTimeout, state.StartTime, state.StopTime, state.TaskConfig, state.TaskStatus,
		state.TaskResult)
	if err != nil {
		return err
	}

	return nil
}

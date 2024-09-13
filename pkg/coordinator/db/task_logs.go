package db

import "github.com/jmoiron/sqlx"

type TaskLog struct {
	RunID      int    `db:"run_id"`
	TaskID     int    `db:"task_id"`
	LogTime    int64  `db:"log_time"`
	LogLevel   string `db:"log_level"`
	LogFields  string `db:"log_fields"`
	LogMessage string `db:"log_message"`
}

func (db *Database) InsertTaskLog(tx *sqlx.Tx, log *TaskLog) error {
	_, err := tx.Exec(db.EngineQuery(map[EngineType]string{
		EnginePgsql: `
			INSERT INTO task_logs (
				run_id, task_id, log_time, log_level, log_fields, log_message
			) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (run_id, task_id, log_time) DO UPDATE SET
				log_level = excluded.log_level,
				log_fields = excluded.log_fields,
				log_message = excluded.log_message`,
		EngineSqlite: `
			INSERT OR REPLACE INTO task_logs (
				run_id, task_id, log_time, log_level, log_fields, log_message
			) VALUES ($1, $2, $3, $4, $5, $6)`,
	}),
		log.RunID, log.TaskID, log.LogTime, log.LogLevel, log.LogFields, log.LogMessage)
	if err != nil {
		return err
	}

	return nil
}

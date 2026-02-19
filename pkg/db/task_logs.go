package db

import "github.com/jmoiron/sqlx"

type TaskLog struct {
	RunID      uint64 `db:"run_id"`
	TaskID     uint64 `db:"task_id"`
	LogIndex   uint64 `db:"log_idx"`
	LogTime    int64  `db:"log_time"`
	LogLevel   uint32 `db:"log_level"`
	LogFields  string `db:"log_fields"`
	LogMessage string `db:"log_message"`
}

func (db *Database) InsertTaskLog(tx *sqlx.Tx, log *TaskLog) error {
	_, err := tx.Exec(db.EngineQuery(map[EngineType]string{
		EnginePgsql: `
			INSERT INTO task_logs (
				run_id, task_id, log_idx, log_time, log_level, log_fields, log_message
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (run_id, task_id, log_time) DO UPDATE SET
				log_time = excluded.log_time,
				log_level = excluded.log_level,
				log_fields = excluded.log_fields,
				log_message = excluded.log_message`,
		EngineSqlite: `
			INSERT OR REPLACE INTO task_logs (
				run_id, task_id, log_idx, log_time, log_level, log_fields, log_message
			) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
	}),
		log.RunID, log.TaskID, log.LogIndex, log.LogTime, log.LogLevel, log.LogFields, log.LogMessage)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) GetTaskLogs(runID, taskID, fromIdx, limit uint64) ([]*TaskLog, error) {
	var logs []*TaskLog

	err := db.reader.Select(&logs, `
		SELECT * FROM task_logs
		WHERE run_id = $1 AND task_id = $2 AND log_idx >= $3
		ORDER BY log_idx ASC
		LIMIT $4`,
		runID, taskID, fromIdx, limit)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func (db *Database) GetLastLogIndex(runID, taskID uint64) (uint64, error) {
	var logIdx uint64

	err := db.reader.Get(&logIdx, `
		SELECT COALESCE(MAX(log_idx), 0) FROM task_logs
		WHERE run_id = $1 AND task_id = $2`,
		runID, taskID)
	if err != nil {
		return 0, err
	}

	return logIdx, nil
}

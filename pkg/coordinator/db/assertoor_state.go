package db

import (
	"encoding/json"

	"github.com/jmoiron/sqlx"
)

type AssertoorState struct {
	Key   string `db:"key"`
	Value string `db:"value"`
}

func (db *Database) GetAssertoorState(key string, returnValue interface{}) (interface{}, error) {
	entry := AssertoorState{}

	err := db.reader.Get(&entry, `SELECT key, value FROM assertoor_state WHERE key = $1`, key)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(entry.Value), returnValue)
	if err != nil {
		return nil, err
	}

	return returnValue, nil
}

func (db *Database) SetExplorerState(tx *sqlx.Tx, key string, value interface{}) error {
	valueMarshal, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = tx.Exec(db.EngineQuery(map[EngineType]string{
		EnginePgsql: `
			INSERT INTO assertoor_state (key, value)
			VALUES ($1, $2)
			ON CONFLICT (key) DO UPDATE SET
				value = excluded.value`,
		EngineSqlite: `
			INSERT OR REPLACE INTO assertoor_state (key, value)
			VALUES ($1, $2)`,
	}), key, valueMarshal)
	if err != nil {
		return err
	}

	return nil
}

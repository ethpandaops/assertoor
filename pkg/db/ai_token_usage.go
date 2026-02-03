package db

import (
	"time"

	"github.com/jmoiron/sqlx"
)

// AITokenUsage represents a row in the ai_token_usage table.
type AITokenUsage struct {
	ID               int64     `db:"id"`
	Date             time.Time `db:"date"`
	TestName         string    `db:"test_name"`
	Model            string    `db:"model"`
	PromptTokens     int64     `db:"prompt_tokens"`
	CompletionTokens int64     `db:"completion_tokens"`
	TotalTokens      int64     `db:"total_tokens"`
	Requests         int64     `db:"requests"`
	CreatedAt        time.Time `db:"created_at"`
}

// AIUsageStats holds aggregated usage statistics.
type AIUsageStats struct {
	TotalPromptTokens     int64 `json:"totalPromptTokens" db:"total_prompt_tokens"`
	TotalCompletionTokens int64 `json:"totalCompletionTokens" db:"total_completion_tokens"`
	TotalTokens           int64 `json:"totalTokens" db:"total_tokens"`
	TotalRequests         int64 `json:"totalRequests" db:"total_requests"`
}

// RecordTokenUsage inserts or updates token usage for a day/test/model combo.
func (db *Database) RecordTokenUsage(
	tx *sqlx.Tx,
	date time.Time,
	testName, model string,
	promptTokens, completionTokens int64,
) error {
	totalTokens := promptTokens + completionTokens
	dateOnly := date.Format("2006-01-02")

	_, err := tx.Exec(db.EngineQuery(map[EngineType]string{
		EnginePgsql: `
			INSERT INTO ai_token_usage (
				date, test_name, model, prompt_tokens, completion_tokens, total_tokens, requests
			) VALUES ($1, $2, $3, $4, $5, $6, 1)
			ON CONFLICT (date, test_name, model) DO UPDATE SET
				prompt_tokens = ai_token_usage.prompt_tokens + excluded.prompt_tokens,
				completion_tokens = ai_token_usage.completion_tokens + excluded.completion_tokens,
				total_tokens = ai_token_usage.total_tokens + excluded.total_tokens,
				requests = ai_token_usage.requests + 1`,
		EngineSqlite: `
			INSERT INTO ai_token_usage (
				date, test_name, model, prompt_tokens, completion_tokens, total_tokens, requests
			) VALUES ($1, $2, $3, $4, $5, $6, 1)
			ON CONFLICT (date, test_name, model) DO UPDATE SET
				prompt_tokens = ai_token_usage.prompt_tokens + excluded.prompt_tokens,
				completion_tokens = ai_token_usage.completion_tokens + excluded.completion_tokens,
				total_tokens = ai_token_usage.total_tokens + excluded.total_tokens,
				requests = ai_token_usage.requests + 1`,
	}),
		dateOnly, testName, model, promptTokens, completionTokens, totalTokens)

	return err
}

// GetUsageByDateRange returns aggregated usage stats for a date range.
func (db *Database) GetUsageByDateRange(startDate, endDate time.Time) (*AIUsageStats, error) {
	var stats AIUsageStats

	startDateOnly := startDate.Format("2006-01-02")
	endDateOnly := endDate.Format("2006-01-02")

	err := db.reader.Get(&stats, `
		SELECT
			COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as total_completion_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(requests), 0) as total_requests
		FROM ai_token_usage
		WHERE date >= $1 AND date <= $2`,
		startDateOnly, endDateOnly)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetUsageByTestName returns usage stats for a specific test within a date range.
func (db *Database) GetUsageByTestName(
	testName string,
	startDate, endDate time.Time,
) (*AIUsageStats, error) {
	var stats AIUsageStats

	startDateOnly := startDate.Format("2006-01-02")
	endDateOnly := endDate.Format("2006-01-02")

	err := db.reader.Get(&stats, `
		SELECT
			COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as total_completion_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(requests), 0) as total_requests
		FROM ai_token_usage
		WHERE test_name = $1 AND date >= $2 AND date <= $3`,
		testName, startDateOnly, endDateOnly)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

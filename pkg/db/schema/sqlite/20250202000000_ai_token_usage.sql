-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS "ai_token_usage"
(
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "date" DATE NOT NULL,
    "test_name" TEXT NOT NULL,
    "model" TEXT NOT NULL,
    "prompt_tokens" INTEGER NOT NULL,
    "completion_tokens" INTEGER NOT NULL,
    "total_tokens" INTEGER NOT NULL,
    "requests" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS "ai_token_usage_date_idx" ON "ai_token_usage" ("date");
CREATE INDEX IF NOT EXISTS "ai_token_usage_test_name_idx" ON "ai_token_usage" ("test_name");
CREATE UNIQUE INDEX IF NOT EXISTS "ai_token_usage_unique_idx" ON "ai_token_usage" ("date", "test_name", "model");

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "ai_token_usage";
-- +goose StatementEnd

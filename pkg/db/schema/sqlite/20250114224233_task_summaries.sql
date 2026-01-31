-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS "task_results"
(
    "run_id" INTEGER NOT NULL,
    "task_id" INTEGER NOT NULL,
    "result_type" TEXT NOT NULL,
    "result_index" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "size" INTEGER NOT NULL,
    "data" BLOB NOT NULL,
    CONSTRAINT "task_results_pkey" PRIMARY KEY ("run_id", "task_id", "result_type", "result_index")
);

CREATE INDEX IF NOT EXISTS "task_results_name_idx" ON "task_results" ("run_id", "task_id", "name");

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
SELECT 'NOT SUPPORTED';
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS "assertoor_state"
(
    "key" TEXT NOT NULL UNIQUE,
    "value" TEXT,
    CONSTRAINT "assertoor_state_pkey" PRIMARY KEY ("key")
);

CREATE TABLE IF NOT EXISTS "test_configs"
(
    "test_id" TEXT NOT NULL,
    "source" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "timeout" INTEGER NOT NULL,
    "config" TEXT NOT NULL,
    "config_vars" TEXT NOT NULL,
    "schedule_startup" BOOLEAN NOT NULL,
    "schedule_cron_yaml" TEXT NOT NULL,
    CONSTRAINT "test_configs_pkey" PRIMARY KEY ("test_id")
);

CREATE TABLE IF NOT EXISTS "test_runs"
(
    "run_id" INTEGER NOT NULL,
    "test_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "source" TEXT NOT NULL,
    "config" TEXT NOT NULL,
    "start_time" INTEGER NOT NULL,
    "stop_time" INTEGER NOT NULL,
    "status" TEXT NOT NULL,
    CONSTRAINT "test_runs_pkey" PRIMARY KEY ("run_id")
);

CREATE TABLE IF NOT EXISTS "task_states"
(
    "run_id" INTEGER NOT NULL,
    "task_id" INTEGER NOT NULL,
    "parent_task" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "timeout" INTEGER NOT NULL,
    "ifcond" TEXT NOT NULL,
    "is_cleanup" BOOLEAN NOT NULL,
    "is_started" BOOLEAN NOT NULL,
    "is_running" BOOLEAN NOT NULL,
    "is_skipped" BOOLEAN NOT NULL,
    "is_timeout" BOOLEAN NOT NULL,
    "start_time" INTEGER NOT NULL,
    "stop_time" INTEGER NOT NULL,
    "task_config" TEXT NOT NULL,
    "task_status" TEXT NOT NULL,
    "task_result" INTEGER NOT NULL,
    CONSTRAINT "task_states_pkey" PRIMARY KEY ("run_id", "task_id")
);

CREATE TABLE IF NOT EXISTS "task_logs"
(
    "run_id" INTEGER NOT NULL,
    "task_id" INTEGER NOT NULL,
    "log_time" INTEGER NOT NULL,
    "log_level" TEXT NOT NULL,
    "log_fields" TEXT NOT NULL,
    "log_message" TEXT NOT NULL,
    CONSTRAINT "task_logs_pkey" PRIMARY KEY ("run_id", "task_id", "log_time")
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
SELECT 'NOT SUPPORTED';
-- +goose StatementEnd

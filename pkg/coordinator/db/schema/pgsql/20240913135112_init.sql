-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS public."assertoor_state"
(
    "key" character varying(150) COLLATE pg_catalog."default" NOT NULL,
    "value" text COLLATE pg_catalog."default",
    CONSTRAINT "assertoor_state_pkey" PRIMARY KEY ("key")
);

CREATE TABLE IF NOT EXISTS public."test_configs"
(
    "test_id" VARCHAR(128) NOT NULL,
    "source" VARCHAR(256) NOT NULL,
    "name" VARCHAR(256) NOT NULL,
    "timeout" INTEGER NOT NULL,
    "config" TEXT NOT NULL,
    "config_vars" TEXT NOT NULL,
    "schedule_startup" boolean NOT NULL,
    "schedule_cron_yaml" TEXT NOT NULL,
    CONSTRAINT "test_configs_pkey" PRIMARY KEY ("test_id")
);

CREATE TABLE IF NOT EXISTS public."test_runs"
(
    "run_id" INTEGER NOT NULL,
    "test_id" VARCHAR(256) NOT NULL,
    "name" VARCHAR(256) NOT NULL,
    "source" VARCHAR(256) NOT NULL,
    "config" TEXT NOT NULL,
    "start_time" BIGINT NOT NULL,
    "stop_time" BIGINT NOT NULL,
    "status" VARCHAR(16) NOT NULL,
    CONSTRAINT "test_runs_pkey" PRIMARY KEY ("run_id")
);

CREATE TABLE IF NOT EXISTS public."task_states"
(
    "run_id" INTEGER NOT NULL,
    "task_id" INTEGER NOT NULL,
    "parent_task" INTEGER NOT NULL,
    "name" VARCHAR(128) NOT NULL,
    "title" TEXT NOT NULL,
    "timeout" INTEGER NOT NULL,
    "ifcond" TEXT NOT NULL,
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

CREATE TABLE IF NOT EXISTS public."task_logs"
(
    "run_id" INTEGER NOT NULL,
    "task_id" INTEGER NOT NULL,
    "log_time" BIGINT NOT NULL,
    "log_level" VARCHAR(16) NOT NULL,
    "log_fields" TEXT NOT NULL,
    "log_message" TEXT NOT NULL,
    CONSTRAINT "task_logs_pkey" PRIMARY KEY ("run_id", "task_id", "log_time")
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
SELECT 'NOT SUPPORTED';
-- +goose StatementEnd

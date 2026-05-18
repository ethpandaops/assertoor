-- +goose Up
-- +goose StatementBegin

CREATE TABLE "dashboard_config" (
    "key" TEXT PRIMARY KEY,
    "data" BYTEA NOT NULL,
    "updated_at" BIGINT NOT NULL
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
SELECT 'NOT SUPPORTED';
-- +goose StatementEnd

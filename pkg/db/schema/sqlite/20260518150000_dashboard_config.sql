-- +goose Up
-- +goose StatementBegin

CREATE TABLE "dashboard_config" (
    "key" TEXT PRIMARY KEY,
    "data" BLOB NOT NULL,
    "updated_at" INTEGER NOT NULL
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
SELECT 'NOT SUPPORTED';
-- +goose StatementEnd

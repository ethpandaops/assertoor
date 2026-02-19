-- +goose Up
-- +goose StatementBegin

ALTER TABLE "test_configs" ADD COLUMN "yaml_source" TEXT NOT NULL DEFAULT '';

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
SELECT 'NOT SUPPORTED';
-- +goose StatementEnd

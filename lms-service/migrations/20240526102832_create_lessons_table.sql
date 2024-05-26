-- +goose Up
-- +goose StatementBegin
CREATE TABLE lessons (
                         id SERIAL PRIMARY KEY,
                         created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                         updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                         title TEXT NOT NULL,
                         link TEXT,
                         conspect TEXT,
                         module_id INTEGER REFERENCES modules (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE lessons;
-- +goose StatementEnd

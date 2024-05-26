-- +goose Up
-- +goose StatementBegin
CREATE TABLE modules (
                         id SERIAL PRIMARY KEY,
                         created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                         updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                         title TEXT NOT NULL,
                         course_id INTEGER REFERENCES courses (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE modules;
-- +goose StatementEnd

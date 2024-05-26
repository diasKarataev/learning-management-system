-- +goose Up
-- +goose StatementBegin
CREATE TABLE courses (
                         id SERIAL PRIMARY KEY,
                         created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                         updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                         title TEXT NOT NULL,
                         description TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE courses;
-- +goose StatementEnd

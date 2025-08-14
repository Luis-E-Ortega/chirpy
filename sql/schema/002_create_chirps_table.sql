-- +goose Up
CREATE TABLE chirps (
    id UUID primary key,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    body TEXT NOT NULL,
    user_id UUID,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE chirps;
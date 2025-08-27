-- +goose Up
CREATE TABLE refresh_tokens (
    token TEXT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    user_id UUID,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP,
    revoked_at TIMESTAMP
);

-- Function to update 'updated_at' column (single line body). Environment wouldn't load it properly with multiple lines, had to make return statement in one.
CREATE OR REPLACE FUNCTION set_updated_at_timestamp()
RETURNS TRIGGER AS ' BEGIN NEW.updated_at = NOW(); RETURN NEW; END; ' LANGUAGE plpgsql;

-- Trigger to call the function before each update on 'refresh_tokens'
CREATE TRIGGER trigger_set_updated_at
BEFORE UPDATE ON refresh_tokens
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_timestamp();

-- +goose Down
DROP TABLE refresh_tokens;
DROP FUNCTION IF EXISTS set_updated_at_timestamp(); -- Clean up the function on Down
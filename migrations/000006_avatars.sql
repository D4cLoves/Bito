-- +migrate Up
CREATE TABLE IF NOT EXISTS user_avatars (
    user_id    UUID        PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    avatar     VARCHAR(32) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +migrate Down
DROP TABLE IF EXISTS user_avatars;

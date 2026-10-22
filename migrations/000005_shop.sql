-- +migrate Up
CREATE TABLE IF NOT EXISTS user_items (
    user_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item      VARCHAR(32) NOT NULL,
    price     BIGINT      NOT NULL CHECK (price > 0),
    bought_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, item)
);

CREATE TABLE IF NOT EXISTS user_looks (
    user_id    UUID        PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme      VARCHAR(32) NOT NULL,
    felt       VARCHAR(32) NOT NULL,
    back       VARCHAR(32) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +migrate Down
DROP TABLE IF EXISTS user_looks;
DROP TABLE IF EXISTS user_items;

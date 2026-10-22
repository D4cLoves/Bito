-- +migrate Up
CREATE TABLE IF NOT EXISTS friendships (
    user_low     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_high    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    requested_by UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       VARCHAR(16) NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accepted_at  TIMESTAMPTZ,
    PRIMARY KEY (user_low, user_high),

    CONSTRAINT friendships_ordered CHECK (user_low < user_high),
    CONSTRAINT friendships_requester_in_pair CHECK (requested_by IN (user_low, user_high)),
    CONSTRAINT friendships_status_valid CHECK (status IN ('pending', 'accepted')),
    CONSTRAINT friendships_accepted_at_matches_status CHECK ((status = 'accepted') = (accepted_at IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS idx_friendships_high ON friendships(user_high);

-- +migrate Down
DROP TABLE IF EXISTS friendships;

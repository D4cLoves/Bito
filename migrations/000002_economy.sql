-- +migrate Up
CREATE TABLE IF NOT EXISTS games (
    id          UUID PRIMARY KEY,
    mode        VARCHAR(16) NOT NULL,
    deck        SMALLINT    NOT NULL,
    stake       BIGINT      NOT NULL CHECK (stake > 0),
    fee         BIGINT      NOT NULL DEFAULT 0 CHECK (fee >= 0),
    status      VARCHAR(16) NOT NULL DEFAULT 'playing',
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,

    CONSTRAINT games_status_valid CHECK (status IN ('playing', 'finished', 'draw', 'refunded'))
);

CREATE TABLE IF NOT EXISTS game_players (
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    place   SMALLINT,
    outcome VARCHAR(8),
    payout  BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (game_id, user_id),

    CONSTRAINT game_players_outcome_valid CHECK (outcome IS NULL OR outcome IN ('win', 'lose', 'draw'))
);
CREATE INDEX IF NOT EXISTS idx_game_players_user ON game_players(user_id);

CREATE TABLE IF NOT EXISTS wallet_transactions (
    id            BIGSERIAL PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    amount        BIGINT NOT NULL CHECK (amount <> 0),
    kind          VARCHAR(16) NOT NULL,
    game_id       UUID REFERENCES games(id) ON DELETE SET NULL,
    balance_after BIGINT NOT NULL CHECK (balance_after >= 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_user ON wallet_transactions(user_id, created_at DESC);

ALTER TABLE user_wallets ADD COLUMN IF NOT EXISTS last_top_up_at TIMESTAMPTZ;

INSERT INTO wallet_transactions (user_id, amount, kind, balance_after)
SELECT w.user_id, w.balance, 'opening', w.balance
FROM user_wallets w
WHERE w.balance > 0
  AND NOT EXISTS (SELECT 1 FROM wallet_transactions t WHERE t.user_id = w.user_id);

-- +migrate Down
ALTER TABLE user_wallets DROP COLUMN IF EXISTS last_top_up_at;
DROP TABLE IF EXISTS wallet_transactions;
DROP TABLE IF EXISTS game_players;
DROP TABLE IF EXISTS games;

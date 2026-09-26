-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied

-- Включаем расширение для генерации UUID v4 (если вдруг база без gen_random_uuid)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255),
    name          VARCHAR(64) NOT NULL,
    password_hash VARCHAR(255),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Уникальный функциональный индекс на email (регистронезависимый)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower ON users (LOWER(email)) WHERE email IS NOT NULL;

-- Уникальный функциональный индекс на никнейм (регистронезависимый)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_name_lower ON users (LOWER(name));

-- 2. Таблица игровой статистики (Связь 1-к-1 с каскадным удалением)
CREATE TABLE IF NOT EXISTS user_stats (
    user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    elo         INTEGER NOT NULL DEFAULT 1000,
    wins        INTEGER NOT NULL DEFAULT 0,
    losses      INTEGER NOT NULL DEFAULT 0,
    total_games INTEGER NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индекс для таблицы лидеров (топ игроков по рейтингу Elo)
CREATE INDEX IF NOT EXISTS idx_user_stats_elo ON user_stats(elo DESC);

-- 3. Таблица кошелька и баланса (Связь 1-к-1 с каскадным удалением)
CREATE TABLE IF NOT EXISTS user_wallets (
    user_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    balance    BIGINT NOT NULL DEFAULT 2500 CHECK (balance >= 0),
    bonus      BIGINT NOT NULL DEFAULT 0 CHECK (bonus >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Таблица привязок сторонних аккаунтов (OAuth: Google, VK, Telegram)
CREATE TABLE IF NOT EXISTS oauth_accounts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         VARCHAR(32) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Один и тот же аккаунт соцсети нельзя привязать дважды
    CONSTRAINT uq_oauth_provider_user UNIQUE (provider, provider_user_id)
);

-- Индекс для выборки всех привязанных соцсетей конкретного пользователя
CREATE INDEX IF NOT EXISTS idx_oauth_accounts_user_id ON oauth_accounts(user_id);

-- +migrate Down
-- SQL in section 'Down' is executed when this migration is rolled back

DROP TABLE IF EXISTS oauth_accounts;
DROP TABLE IF EXISTS user_wallets;
DROP TABLE IF EXISTS user_stats;
DROP TABLE IF EXISTS users;

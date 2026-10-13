---
title: Bito — план экономики (бэкенд)
tags: [bito, plan, economy, backend]
created: 2026-10-13
---

# Экономика: план реализации

> [!abstract] Цель
> Ставка списывается при старте партии, по итогам партии деньги распределяются по местам, всё пишется в журнал `wallet_transactions`. Баланс и статистика наконец начинают меняться после игры.

Ветка: `feature/economy` от `main`.

## 0. Что уже есть в коде

| Что | Где | Состояние |
|---|---|---|
| Порядок мест | `game.go`, поле `places`, метод `Places()` | **готово**: вышедшие по порядку → дурак → сдавшиеся (последний сдавшийся раньше) |
| Несколько вышли за один ход | `checkGameOver` идёт по `g.players` | **готово**: порядок по месту за столом |
| Кошелёк | таблица `user_wallets`, `balance BIGINT CHECK (balance >= 0)`, по умолчанию 2500 | есть, но никто не меняет |
| Статистика | таблица `user_stats` (wins, losses, total_games) | есть, но никто не меняет |
| Баланс в API | `GET /api/v1/users/me` → `wallet.balance` | готово |
| Ставки на фронте | `web/src/constants/economy.ts`: `STAKES`, `TABLE_FEE`, выбор ставки в лобби | UI готов, на бэк не отправляется |
| Комната ↔ БД | `ws.Room` | **связи нет**, комната ничего не знает о деньгах |

> [!note] Пункт P0 «порядок выхода игроков» из `features.md` фактически сделан. Не хватает только метода, который отдаёт **всех проигравших** (см. шаг 2).

## 1. Решения (по умолчанию так, скажи, если иначе)

| # | Вопрос | Решение | Почему |
|---|---|---|---|
| 1 | Когда списывать ставку | **при старте партии**, всем сразу, одной транзакцией | Не нужен возврат при уходе из ожидающего стола или закрытии стола до старта. Минус целый класс багов. |
| 2 | Кто платит | **дурак и все сдавшиеся** | Сдача и автосдача по таймауту — это проигрыш (так в `features.md`). |
| 3 | Что делится | пул = сумма ставок всех проигравших, минус комиссия 5% | Формула долей из `features.md` не меняется, просто пул может быть больше одной ставки. |
| 4 | Округление | комиссия `pool * 5 / 100` (вниз), доли вниз, остаток → 1 месту | Целые числа, деньги не теряются. |
| 5 | Ничья | всем вернуть ставку, комиссии нет | Как в `features.md`. |
| 6 | Стартовый баланс | **2500** | Уже стоит в БД по умолчанию и на фронте (`JoinCard.vue`). 1000 из `features.md` устарело. |
| 7 | Допустимые ставки | только из списка `100, 250, 500, 1000, 2500, 5000` | Тот же список, что `STAKES` на фронте. |
| 8 | ID партии | **= ID комнаты** | Комната живёт ровно одну партию и закрывается через 30 сек после конца. |
| 9 | Подачка | до 500, если баланс < 100, не чаще раза в час | Как в `features.md`. |

## 2. Шаги

Каждый шаг — отдельный коммит, после каждого `go test ./...` зелёный.

### Шаг 1. Миграция `migrations/000002_economy.sql`

```sql
-- +migrate Up
CREATE TABLE IF NOT EXISTS games (
    id          UUID PRIMARY KEY,
    mode        VARCHAR(16) NOT NULL,
    deck        SMALLINT    NOT NULL,
    stake       BIGINT      NOT NULL CHECK (stake > 0),
    fee         BIGINT      NOT NULL DEFAULT 0,
    status      VARCHAR(16) NOT NULL DEFAULT 'playing', -- playing | finished | draw | refunded
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS game_players (
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    place   SMALLINT,              -- 1..N, NULL пока идёт партия
    outcome VARCHAR(8),            -- win | lose | draw
    payout  BIGINT NOT NULL DEFAULT 0, -- итог для игрока: +выигрыш, -ставка, 0
    PRIMARY KEY (game_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_game_players_user ON game_players(user_id);

CREATE TABLE IF NOT EXISTS wallet_transactions (
    id            BIGSERIAL PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount        BIGINT NOT NULL,          -- со знаком: -100 списание, +237 зачисление
    kind          VARCHAR(16) NOT NULL,     -- signup_bonus | stake_lock | stake_refund | win_payout | top_up | opening
    game_id       UUID REFERENCES games(id) ON DELETE SET NULL,
    balance_after BIGINT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_user ON wallet_transactions(user_id, created_at DESC);

ALTER TABLE user_wallets ADD COLUMN IF NOT EXISTS last_top_up_at TIMESTAMPTZ;

-- Входящий остаток для уже существующих игроков, чтобы баланс = сумма журнала
INSERT INTO wallet_transactions (user_id, amount, kind, balance_after)
SELECT user_id, balance, 'opening', balance FROM user_wallets;

-- +migrate Down
ALTER TABLE user_wallets DROP COLUMN IF EXISTS last_top_up_at;
DROP TABLE IF EXISTS wallet_transactions;
DROP TABLE IF EXISTS game_players;
DROP TABLE IF EXISTS games;
```

> [!question] Как ты применяешь миграции?
> В репозитории нет инструмента миграций, в `docker-compose.yml` их тоже нет. Если руками через `psql`, то так же. Если хочешь, добавим `make migrate`.

**Почему `balance_after`:** при разборе «куда делись деньги» сразу видно баланс после каждой операции, не надо суммировать.

### Шаг 2. Чистая логика: пакет `internal/economy`

Без БД и без сети, только математика. Здесь главные тесты.

```go
package economy

const (
    StartBalance   int64 = 2500
    MinStake       int64 = 100
    FeePercent     int64 = 5
    TopUpThreshold int64 = 100
    TopUpTo        int64 = 500
    TopUpCooldown        = time.Hour
)

var AllowedStakes = []int64{100, 250, 500, 1000, 2500, 5000}

func IsAllowedStake(s int64) bool

type Result struct {
    Places []string        // из game.Places()
    Losers map[string]bool // дурак + сдавшиеся
    IsDraw bool
}

type Payout struct {
    UserID string
    Place  int   // 1..N
    Credit int64 // сколько ВЕРНУТЬ на баланс при расчёте (ставка уже списана)
    Net    int64 // итог партии для игрока: Credit - stake
    Win    bool
}

// Settle считает выплаты. Возвращает выплаты по местам и комиссию.
func Settle(stake int64, r Result) (payouts []Payout, fee int64)
```

Логика `Settle`:
1. `IsDraw` → всем `Credit = stake`, `Net = 0`, `fee = 0`.
2. Победители = `Places` без `Losers`, по порядку. `m = len(winners)`.
3. `pool = stake * len(Losers)`, `fee = pool * FeePercent / 100`, `dist = pool - fee`.
4. Доля места `i` (с 1): `dist * (m - i + 1) / (m * (m + 1) / 2)`. Остаток `dist - сумма долей` → первому месту.
5. Победитель: `Credit = stake + доля`, `Net = доля`. Проигравший: `Credit = 0`, `Net = -stake`.

В пакет `game` добавить:

```go
// Losers возвращает дурака и всех сдавшихся.
func (g *Game) Losers() []PlayerID
```

Дурак — `loserID`, сдавшиеся — `g.surrendered`. Без дублей: при сдаче последнего соперника `loserID` уже равен сдавшемуся.

**Тесты (`economy_test.go`):**
- Пример из `features.md` §3.3: 4 игрока, ставка 500 → `+238 / +158 / +79 / −500`, комиссия 25.
- 2 игрока, ставка 100 → `+95 / −100`, комиссия 5.
- Двое проигравших (дурак + сдавшийся), 4 игрока.
- Ничья → все `Net = 0`, комиссия 0.
- **Инвариант**: для всех ставок из `AllowedStakes` и от 2 до 6 игроков, при любом числе проигравших от 1 до N−1: `сумма Net == −fee` и `fee >= 0`. Это тест «деньги не появляются и не исчезают».

### Шаг 3. Репозиторий `internal/repository/postgres/wallet.go`

```go
var ErrInsufficientFunds = errors.New("insufficient funds") // лучше в model/errors.go
var ErrTopUpNotAllowed  = errors.New("top-up not allowed")

type WalletRepository struct{ pool *pgxpool.Pool }

func (r *WalletRepository) Balance(ctx, userID uuid.UUID) (int64, error)

// LockStakes: одна транзакция на всю партию.
func (r *WalletRepository) LockStakes(ctx, g GameRecord, userIDs []uuid.UUID) error

// Settle: одна транзакция, идемпотентна.
func (r *WalletRepository) Settle(ctx, gameID uuid.UUID, fee int64, status string, payouts []economy.Payout) error

func (r *WalletRepository) TopUp(ctx, userID uuid.UUID) (newBalance int64, err error)

// RefundUnfinished: при старте сервера вернуть ставки партий, оборванных падением сервера.
func (r *WalletRepository) RefundUnfinished(ctx) (int, error)
```

**`LockStakes`**, всё в одном `tx`:
1. `INSERT INTO games (id, mode, deck, stake)`.
2. Отсортировать `userIDs` (одинаковый порядок блокировок → нет дедлоков между столами).
3. Для каждого игрока:
   ```sql
   UPDATE user_wallets SET balance = balance - $2, updated_at = NOW()
   WHERE user_id = $1 AND balance >= $2
   RETURNING balance
   ```
   Нет строки (`pgx.ErrNoRows`) → откат и `ErrInsufficientFunds` с ID игрока (обернуть: `fmt.Errorf("%w: %s", ErrInsufficientFunds, id)` или свой тип ошибки).
4. `INSERT INTO wallet_transactions (... kind 'stake_lock', amount -stake, balance_after)`.
5. `INSERT INTO game_players (game_id, user_id)`.

**`Settle`**, всё в одном `tx`:
1. Защита от двойной выплаты, первым запросом:
   ```sql
   UPDATE games SET status = $2, fee = $3, finished_at = NOW()
   WHERE id = $1 AND status = 'playing'
   ```
   `RowsAffected() == 0` → уже рассчитано, выйти без ошибки.
2. Для каждой выплаты с `Credit > 0`: `UPDATE user_wallets SET balance = balance + credit RETURNING balance` и строка в журнал (`stake_refund` на ставку, `win_payout` на выигрыш; либо одна строка `win_payout` на всю сумму, выбери сам).
3. `UPDATE game_players SET place, outcome, payout = Net`.
4. `UPDATE user_stats SET wins = wins + ..., losses = losses + ..., total_games = total_games + 1`. При ничьей только `total_games`.

**`TopUp`**, атомарно одним запросом:
```sql
WITH old AS (SELECT balance FROM user_wallets WHERE user_id = $1 FOR UPDATE)
UPDATE user_wallets w SET balance = $2, last_top_up_at = NOW(), updated_at = NOW()
FROM old
WHERE w.user_id = $1 AND old.balance < $3
  AND (w.last_top_up_at IS NULL OR w.last_top_up_at < NOW() - INTERVAL '1 hour')
RETURNING w.balance, $2 - old.balance
```
Нет строки → `ErrTopUpNotAllowed`. Есть → строка в журнал `top_up`.

**Регистрация.** В `UserRepository.CreateUser` вставлять кошелёк с явным `economy.StartBalance` и в той же транзакции писать в журнал `signup_bonus`.

### Шаг 4. Сервис `internal/service/economy.go`

Тонкий слой: собирает `economy.Settle` и репозиторий, переводит `string` ID комнаты в `uuid`.

```go
type EconomyService struct{ repo *postgres.WalletRepository }

func (s *EconomyService) Balance(ctx, userID string) (int64, error)
func (s *EconomyService) LockStakes(ctx, gameID string, mode string, deck int, stake int64, players []string) error
func (s *EconomyService) Settle(ctx, gameID string, stake int64, r economy.Result) ([]economy.Payout, error)
func (s *EconomyService) TopUp(ctx, userID string) (int64, error)
```

### Шаг 5. Подключить к комнате (`internal/delivery/ws`)

Интерфейс объявляем **в пакете `ws`** (там, где его используют), реализует его `EconomyService`:

```go
type Bank interface {
    Balance(ctx context.Context, userID string) (int64, error)
    LockStakes(ctx context.Context, gameID, mode string, deck int, stake int64, players []string) error
    Settle(ctx context.Context, gameID string, stake int64, r economy.Result) ([]economy.Payout, error)
}
```

1. `Hub` получает `bank Bank` в `NewHub`, передаёт в `NewRoom`. `Room` получает поля `bank` и `stake int64`.
2. `RoomInfo` + поле `Stake int64 \`json:"stake"\`` (фронт уже читает `stake` из списка столов).
3. **Посадка.** В `Hub.ServeWS` до `Upgrade`: `bank.Balance(playerID) < room.Stake()` → `402` или `400` с текстом «Не хватает фишек на ставку».
4. **Старт.** В `startGame()` до `game.NewGame`: `ctx` с таймаутом 5 сек, `bank.LockStakes(ctx, r.id, ...)`.
   - `ErrInsufficientFunds` для игрока X → отправить X ошибку, закрыть его клиента, удалить из `r.clients`, игру **не** стартовать, ждать следующего.
   - Другая ошибка → лог, ошибка всем, `r.Close()`.
5. **Конец.** В `onGameStateChanged` при `PhaseFinished` до `broadcastGameOver`: собрать `economy.Result` из `Places()`, `Losers()`, `IsDraw()` и вызвать `bank.Settle`. Ошибку записать в лог (расчёт идемпотентный, можно повторить).
6. `GameOverPayload` + `Payouts map[string]int64 \`json:"payouts,omitempty"\`` с `Net` каждого игрока, чтобы фронт показал «+238» и «−500».
7. **Тесты комнаты:** в `room_test.go` все `NewRoom(...)` получат фейковый банк: `fakeBank` с балансом «бесконечность» и записью вызовов. Добавить тест: конец партии → `Settle` вызван один раз с правильными местами.

> [!warning] БД в цикле комнаты
> `LockStakes` и `Settle` блокируют горутину `Room.Run` на время запроса (миллисекунды). Для диплома нормально. Обязательно с таймаутом в `ctx`, чтобы зависшая БД не повесила стол навсегда.

### Шаг 6. Один аккаунт — один стол

1. В `Room`: множество `seated` (ID игроков) под `r.mu`. Обновляется в `handleRegister` и `handleUnregister` до старта, после старта фиксируется на игроках партии. Метод `Has(playerID) bool`.
2. В `Hub.ServeWS`: если игрок сидит в **другой** комнате, которая не закрыта и партия в ней не окончена → `409` «Вы уже за другим столом». Реконнект в свою комнату разрешён.

### Шаг 7. REST

| Метод | Ответ | Ошибки |
|---|---|---|
| `POST /api/v1/tables` + поле `stake` | `RoomInfo` со `stake` | `400` ставка не из списка; `400` баланс меньше ставки |
| `GET /api/v1/tables` | у каждого стола есть `stake` | |
| `POST /api/v1/users/me/top-up` | `{ "balance": 500 }` | `409` «Подачка доступна при балансе меньше 100 и раз в час» |
| `GET /api/v1/users/me/transactions?limit=50` | журнал (необязательно, но полезно для защиты) | |

### Шаг 8. Запуск

В `app.go`:
1. `walletRepo := postgres.NewWalletRepository(pool)`, `economySvc := service.NewEconomyService(walletRepo)`.
2. **До** старта сервера: `walletRepo.RefundUnfinished(ctx)` возвращает ставки партий в статусе `playing` (сервер упал посреди игры), ставит им `refunded`.
3. `ws.NewHub(tokenManager, cfg.AllowedOrigins, economySvc)`, `NewTableHandler(hub, economySvc)`, `NewUserHandler(..., economySvc)`.

## 3. Как проверить руками

1. Два аккаунта по 2500. Стол на 2 игрока, ставка 500.
2. После старта: у обоих 2000, в `wallet_transactions` две строки `stake_lock`.
3. Доиграть. Победитель 2975 (2000 + 500 + 475), проигравший 2000.
4. Проверка суммы:
   ```sql
   SELECT w.user_id, w.balance, SUM(t.amount) AS journal
   FROM user_wallets w JOIN wallet_transactions t USING (user_id)
   GROUP BY w.user_id, w.balance
   HAVING w.balance <> SUM(t.amount);  -- должно вернуть 0 строк
   ```
5. Сдаться посреди партии → сдавшийся платит. Закрыть вкладку на 30 сек → то же самое.
6. Убить сервер посреди партии и поднять → ставки вернулись, `games.status = 'refunded'`.

## 4. Фронт (сделаю я, когда бэк будет готов)

- Отправлять `stake` в `createTable` и `quickMatch`.
- Показывать ставку стола в списке, прятать или блокировать столы дороже баланса.
- В окне результата партии: `+238` / `−500` из `payouts`.
- Обновлять баланс после партии и после подачки.
- Кнопка «Подачка» при балансе меньше 100.

## 5. Порядок коммитов

1. `feat(db): economy tables and wallet journal`
2. `feat(economy): settlement math with money invariant tests`
3. `feat(game): expose all losers`
4. `feat(repo): wallet repository — lock, settle, top-up, refund`
5. `feat(ws): lock stakes on start, settle on game over`
6. `feat(ws): one account, one active table`
7. `feat(http): stake on tables, top-up and transactions endpoints`

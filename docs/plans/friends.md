---
title: Bito — друзья (учебный план)
tags: [bito, plan, friends, sql, go, vue, learning]
created: 2026-10-17
---

# Друзья: учебный план

> [!abstract] Что строим
> Настоящие друзья вместо заглушек в лобби. Игрок добавляет друга по имени, тот видит заявку и принимает её. В списке видно, кто **в лобби**, кто **за столом**, а кто **не в сети**. Заявка приходит другу сразу, без перезагрузки страницы, через уже готовый лобби-сокет.
> Связанные заметки: [[features]] (пункт 7, P1), [[lobby-socket]] (на нём держится онлайн-статус), [[lobby]] (как устроено лобби).

> [!info] Как работать с этой заметкой
> Каждый шаг идёт по одной схеме:
> 1. **Зачем** — какую проблему решаем.
> 2. **Теория** — приёмы, которые понадобятся. Читай до кода.
> 3. **Практика** — что написать, с подсказками.
> 4. **Ловушки** — где обычно ошибаются.
> 5. **Проверь себя** — вопросы. Ответ свёрнут, сначала подумай.
> 6. **Решение** — готовый код, тоже свёрнут. Открывай, когда написал своё или застрял больше чем на 20 минут.
> 7. **Готово, когда** — чек-лист.
>
> Решение — это один из правильных вариантов, а не единственный. Если твой код отличается, но тесты проходят и ты можешь объяснить, почему он так написан, всё в порядке.

> [!success] Статус: готово (20.10.2026), кроме бонусов из шага 9
> Шаги 1–5 написаны тобой. При разборе нашлась одна ошибка: `FriendAccepted` был `"accept"` вместо `"accepted"`, из-за чего принятые друзья попадали бы в заявки, а повторная заявка другу отвечала бы «заявка уже отправлена». Шаги 6–7 и тесты доделаны по этому плану. В проекте файлы называются в единственном числе: `friend.go`, `friend_test.go`, `friend_integration_test.go`.
> Проверено: юнит-тесты и интеграционные на Postgres, `npm run build`, в браузере. Заявка, принятие и встречная заявка приходят второму игроку без перезагрузки, ошибки формы показываются под полем.
> Не проверено вживую: статусы «в лобби» и «за столом» (второй игрок был только через API). Они покрыты тестом `TestHub_PresenceLobbyTableOffline`.

---

## 0. Карта

### Что есть сейчас

| Где | Что там | Чего не хватает |
|---|---|---|
| `web/src/mocks/lobby.ts` | `MOCK_FRIENDS`, `MOCK_REQUESTS` | всё выдумано |
| `components/client/FriendsList.vue` | панель друзей: поиск, фильтр «В сети / Все», заявки | кнопка «Добавить друга» показывает тост «скоро» |
| `components/client/FriendsRail.vue` | рейка справа: аватарки, счётчик в сети | данные из моков |
| `views/lobby/LobbyView.vue` | `acceptRequest` и `declineRequest` правят массивы в памяти | после перезагрузки всё возвращается |
| `ws/lobby.go` | лобби знает, кто подключён | наружу это знание не отдаётся |
| `ws/hub.go` | хаб знает, кто за каким столом | то же самое |

### Что будет

```
 Алиса (лобби)                         Go-сервер                              Боб (лобби)
┌──────────────┐  POST /friends/requests  ┌───────────────┐                 ┌──────────────┐
│ «Добавить    │ ───────────────────────► │ FriendHandler │                 │              │
│  друга: Боб» │                          │       │       │                 │              │
└──────────────┘                          │ FriendService │                 │              │
                                          │   │       │   │                 │              │
                                          │   ▼       ▼   │  friends_changed│              │
                                          │ Postgres  Hub ─┼───────────────► │ useLobbySocket│
                                          │ friendships   │   (/ws/lobby)   │      │       │
                                          └───────────────┘                 │      ▼       │
                                                  ▲          GET /friends   │ useFriends   │
                                                  └──────────────────────── │ .refresh()   │
                                                                            └──────────────┘
```

Сервер не присылает Бобу саму заявку. Он присылает **сигнал** «у тебя изменились друзья», а клиент сам перезапрашивает список. Почему так — в шаге 6.

### API

| Метод | Путь | Тело | Ответ | Что делает |
|---|---|---|---|---|
| `GET` | `/api/v1/friends` | — | `200 {friends, incoming, outgoing}` | всё про друзей одним запросом |
| `POST` | `/api/v1/friends/requests` | `{"username":"Боб"}` | `201 {"status":"pending"}` или `200 {"status":"accepted"}` | отправить заявку; встречная заявка сразу делает друзьями |
| `POST` | `/api/v1/friends/requests/{id}/accept` | — | `204` | принять заявку от игрока `{id}` |
| `DELETE` | `/api/v1/friends/{id}` | — | `204` | удалить друга, отклонить или отменить заявку |

Сообщение лобби-сокета: `{"type":"friends_changed"}` без полезной нагрузки.

### Шаги

| # | Шаг | Главный приём | Сложность | Файлы |
|---|---|---|---|---|
| 1 | Таблица `friendships` | каноническая пара, `CHECK`, составной ключ | ★★☆ | `migrations/000003_friends.sql` |
| 2 | Модель и ошибки | именованные типы, ошибки-метки | ★☆☆ | `model/friend.go` |
| 3 | Репозиторий | upsert с условием, гонки, `RowsAffected` | ★★★ | `repository/postgres/friends.go` |
| 4 | Сервис | маленькие интерфейсы, инверсия зависимостей | ★★☆ | `service/friends.go` |
| 5 | HTTP | REST, коды ответов, ошибки → статусы | ★★☆ | `delivery/http/friend_handler.go`, `server.go`, `app.go` |
| 6 | Статусы и уведомления | присутствие в памяти, «сигнал, а не данные» | ★★☆ | `ws/lobby.go`, `ws/hub.go`, `ws/message.go` |
| 7 | Фронт | перезапрос по сигналу, склейка запросов | ★★☆ | `api/friends.ts`, `useFriends.ts`, `FriendsList.vue`, `LobbyView.vue` |
| 8 | Проверка руками | два браузера, SQL, curl | ★☆☆ | — |
| 9 | Бонусы | «был в сети», лимиты, приглашения | ★★★ | — |

Ветка уже создана: `feature/friends` от `feature/lobby-socket`. Если её нет:

```bash
git switch -c feature/friends
```

---

## Шаг 1. Таблица friendships

### Зачем

Дружба — связь пользователя с пользователем. Нужно хранить её так, чтобы **сама база** не позволила записать ерунду: дружбу с собой, две записи на одну пару, неизвестный статус.

### Теория

> [!note] Связь «многие ко многим» с самим собой
> `users` ↔ `users`. У одного игрока много друзей, и каждый из них тоже игрок. Обычное решение для M:N — отдельная таблица связей. Особенность в том, что дружба **симметрична**: если Алиса дружит с Бобом, то и Боб дружит с Алисой.

> [!note] Два способа хранить симметричную связь
>
> | | Две строки на пару (A→B и B→A) | Одна строка, упорядоченная пара |
> |---|---|---|
> | Как | при дружбе пишем обе стороны | всегда `user_low < user_high` |
> | «Мои друзья» | `WHERE user_id = $1`, просто | `WHERE user_low = $1 OR user_high = $1` |
> | Целостность | легко сломать: одна строка записалась, вторая нет | дубль невозможен, его запрещает первичный ключ |
> | Заявки | неудобно: у заявки есть направление | `requested_by` хранит, кто начал |
>
> Берём **одну строку**. Чуть сложнее запрос списка, зато база сама гарантирует: на пару ровно одна запись, а встречная заявка не создаст вторую.

> [!note] Каноническая пара
> «Упорядочить» значит договориться: в `user_low` всегда меньший id, в `user_high` больший. Тогда пара (Алиса, Боб) и пара (Боб, Алиса) превращаются в одну и ту же строку. Договорённость закрепляем `CHECK (user_low < user_high)`, иначе однажды кто-то запишет наоборот.
> Заодно это запрещает дружбу с собой: `x < x` ложно.

> [!note] Индекс и правило левого префикса
> Первичный ключ `(user_low, user_high)` создаёт индекс по двум колонкам. Он ускоряет поиск по `user_low` и по паре, но **не** по одному `user_high`, как телефонная книга, отсортированная по фамилии, не помогает искать по имени. Запрос «мои друзья» ищет и там и там, поэтому нужен второй индекс на `user_high`.

> [!note] Удаление пользователя
> В проекте удаление мягкое: `users.deleted_at` (см. `UserRepository.Delete`). Строка пользователя остаётся, значит, `ON DELETE CASCADE` не сработает. Удалённых надо **отфильтровать в запросе** (`u.deleted_at IS NULL`). Каскад всё равно ставим: если когда-нибудь пользователей начнут удалять по-настоящему, дружбы уйдут вместе с ними.

> [!note] Статус через CHECK
> `VARCHAR` плюс `CHECK (status IN (...))` — тот же приём, что у `games.status`. Можно было завести `ENUM`-тип Postgres, но добавлять в него значения неудобнее, а в проекте уже принят `CHECK`.

### Практика

Файл `migrations/000003_friends.sql`, формат как у `000002_economy.sql`: секции `-- +migrate Up` и `-- +migrate Down`. Миграция применится сама при старте сервера (коммит `7481d3f`).

Колонки:
- `user_low`, `user_high` — ссылки на `users(id)`;
- `requested_by` — кто отправил заявку;
- `status` — `pending` или `accepted`, по умолчанию `pending`;
- `created_at`, `accepted_at`.

Ограничения:
- первичный ключ на пару;
- `user_low < user_high`;
- `requested_by` — один из двух участников;
- статус только из списка;
- **бонус:** `accepted_at` заполнено тогда и только тогда, когда статус `accepted`.

Индекс на `user_high`.

> [!warning] Ловушки
> - Секция `Down` удаляет таблицу. Без неё миграцию не откатить.
> - `requested_by` не обязана ссылаться на `users`: она всегда равна одному из участников, а их ссылки уже проверены.
> - Ограничение `CHECK (requested_by IN (user_low, user_high))` пишется именно так, через `IN` по колонкам.

### Проверь себя

> [!question]- Почему не хватает `UNIQUE (user_low, user_high)` без упорядочивания?
> Без упорядочивания (Алиса, Боб) и (Боб, Алиса) — разные пары, `UNIQUE` пропустит обе. Встречные заявки создадут две строки, и дальше непонятно, какая главная.

> [!question]- Сравнивает ли Postgres UUID так же, как Go-код будет их упорядочивать?
> Postgres сравнивает `uuid` побайтно, как `memcmp` 16 байт. В Go `uuid.UUID` — это `[16]byte`, и `bytes.Compare(a[:], b[:])` сравнивает так же. Порядок совпадает, поэтому Go может заранее посчитать, кто `low`, а `CHECK` в базе это подтвердит.

> [!question]- Зачем индекс на `user_high`, если есть первичный ключ?
> Индекс ключа отсортирован сначала по `user_low`. Условие `user_high = $1` по нему не найти быстро (правило левого префикса), пришлось бы читать всю таблицу.

> [!question]- Что даёт ограничение на `accepted_at`?
> Нельзя случайно получить «принятую» дружбу без даты или «ожидающую» с датой. Это мелочь, но такие инварианты в базе ловят ошибки в коде, которые тесты могли пропустить.

> [!example]- Решение: миграция
> ```sql
> -- +migrate Up
> CREATE TABLE IF NOT EXISTS friendships (
>     user_low     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
>     user_high    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
>     requested_by UUID        NOT NULL,
>     status       VARCHAR(16) NOT NULL DEFAULT 'pending',
>     created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
>     accepted_at  TIMESTAMPTZ,
>     PRIMARY KEY (user_low, user_high),
>
>     CONSTRAINT friendships_ordered CHECK (user_low < user_high),
>     CONSTRAINT friendships_requester CHECK (requested_by IN (user_low, user_high)),
>     CONSTRAINT friendships_status_valid CHECK (status IN ('pending', 'accepted')),
>     CONSTRAINT friendships_accepted_at CHECK ((status = 'accepted') = (accepted_at IS NOT NULL))
> );
> CREATE INDEX IF NOT EXISTS idx_friendships_high ON friendships(user_high);
>
> -- +migrate Down
> DROP TABLE IF EXISTS friendships;
> ```

### Готово, когда

- [ ] сервер стартует, в логах нет ошибок миграции
- [ ] `\d friendships` в `psql` показывает ключ, четыре `CHECK` и индекс
- [ ] вставка «наоборот», «сам с собой» и со статусом `blocked` падает с ошибкой

---

## Шаг 2. Модель и ошибки

### Зачем

Слоям нужен общий язык: как называется связь, какие у неё состояния и какие бывают ошибки. Всё это живёт в `internal/model`, который не зависит ни от базы, ни от HTTP.

### Теория

> [!note] Именованные строковые типы
> `type FriendState string` с константами вместо голых строк. Компилятор не даст передать `Presence` туда, где ждут `FriendState`, а IDE подскажет все допустимые значения. В JSON и в базе это всё ещё обычная строка.

> [!note] Ошибки-метки (sentinel errors)
> `var ErrAlreadyFriends = errors.New(...)`. Нижние слои возвращают такую ошибку, верхние проверяют её через `errors.Is` и решают, какой HTTP-статус отдать. Так уже устроены `ErrUserNotFound` и `ErrInsufficientFunds` в `model/errors.go`.
> Разница с `fmt.Errorf("...: %w", err)`: обёртка добавляет контекст, но `errors.Is` всё равно находит метку внутри.

> [!note] Почему `Presence` в модели, а не в `ws`
> Статус «в лобби / за столом» вычисляет `ws`, а использует сервис друзей. Если положить тип в `ws`, сервис начнёт импортировать пакет сокетов. Модель — нейтральная территория, её импортируют все.

### Практика

Файл `internal/model/friend.go`:
- ошибки: дружба с собой, уже друзья, заявка уже отправлена, заявка не найдена;
- `FriendState` с `pending` и `accepted`;
- `Presence` с `offline`, `online`, `playing`;
- `FriendLink` — одна связь с точки зрения конкретного игрока: id и имя **другого** игрока, состояние, «исходящая ли заявка», дата, статус присутствия;
- `FriendsOverview` — три списка: друзья, входящие, исходящие.

### Проверь себя

> [!question]- Почему `FriendLink` хранит «другого» игрока, а не пару `low/high`?
> Пара — деталь хранения. Сервису и экрану нужно «кто этот человек для меня»: его имя и кто кому писал. Репозиторий переводит язык таблицы на язык предметной области.

> [!example]- Решение: model/friend.go
> ```go
> package model
>
> import (
> 	"errors"
> 	"time"
>
> 	"github.com/google/uuid"
> )
>
> var (
> 	ErrFriendSelf            = errors.New("cannot befriend yourself")
> 	ErrAlreadyFriends        = errors.New("already friends")
> 	ErrFriendRequestExists   = errors.New("friend request already sent")
> 	ErrFriendRequestNotFound = errors.New("friend request not found")
> )
>
> type FriendState string
>
> const (
> 	FriendPending  FriendState = "pending"
> 	FriendAccepted FriendState = "accepted"
> )
>
> type Presence string
>
> const (
> 	PresenceOffline Presence = "offline"
> 	PresenceOnline  Presence = "online"
> 	PresencePlaying Presence = "playing"
> )
>
> type FriendLink struct {
> 	UserID   uuid.UUID
> 	Username string
> 	State    FriendState
> 	Outgoing bool
> 	Since    time.Time
> 	Presence Presence
> }
>
> type FriendsOverview struct {
> 	Friends  []FriendLink
> 	Incoming []FriendLink
> 	Outgoing []FriendLink
> }
> ```

---

## Шаг 3. Репозиторий

### Зачем

SQL для четырёх операций: заявка, принять, удалить, список. Самое интересное здесь — **гонки**: двое могут нажать «Добавить» друг на друга в одну и ту же секунду.

### Теория

> [!note] Проверить, потом записать: почему это ловушка (TOCTOU)
> Наивно: `SELECT` пару → если нет, `INSERT`; если есть встречная заявка, `UPDATE`. Между `SELECT` и `INSERT` проходит время. Два запроса одновременно увидят «пусто», оба сделают `INSERT`, один упадёт на первичном ключе, и пользователь увидит 500.
> Название: *time of check to time of use*. Лечится тем, что проверка и запись делаются **одной командой**, которую база выполняет атомарно.

> [!note] Upsert с условием
> ```sql
> INSERT ... VALUES (...)
> ON CONFLICT (user_low, user_high) DO UPDATE
> SET status = 'accepted', accepted_at = NOW()
> WHERE friendships.status = 'pending'
>   AND friendships.requested_by <> EXCLUDED.requested_by
> RETURNING status
> ```
> - Пары нет → вставка, `RETURNING` вернёт `pending`.
> - Есть **встречная** заявка → обновление, вернёт `accepted`. Двое захотели дружить, значит, они друзья.
> - Есть **моя же** заявка или уже друзья → условие `WHERE` ложно, ничего не меняется, `RETURNING` **не вернёт строк**. Тогда отдельным `SELECT` выясняем, какая именно ошибка.
>
> `EXCLUDED` — строка, которую мы пытались вставить. `friendships.` — строка, которая уже лежит в таблице.
> При одновременных запросах второй ждёт блокировку строки, которую вставил первый, а потом проверяет `WHERE` уже по свежим данным. Поэтому встречные заявки в одну секунду дают одну строку `accepted`.

> [!note] `RowsAffected` как «сравнить и записать»
> `UPDATE ... WHERE ... AND status = 'pending' AND requested_by = $3` обновит строку, только если всё ещё есть входящая заявка от этого игрока. `RowsAffected() == 0` значит: заявки нет, её уже отменили или принимает не тот. Отдельный `SELECT` не нужен, и гонки нет.

> [!note] «Другой» участник через CASE
> ```sql
> JOIN users u ON u.id = CASE WHEN f.user_low = $1 THEN f.user_high ELSE f.user_low END
> ```
> Для каждой строки берём того, кто **не я**. Флаг «исходящая» считаем прямо в запросе: `f.requested_by = $1`.

> [!note] Идемпотентный DELETE
> Повторный `DELETE` той же пары ничего не находит. Ошибка ли это? Пользователь дважды нажал «Удалить», или заявку уже отклонили с другой вкладки: результат один и тот же, связи нет. Поэтому `Remove` молча возвращает `nil`, а HTTP отвечает `204` в обоих случаях.

> [!note] Интеграционные тесты
> В проекте они помечены `//go:build integration` и поднимают отдельную базу `bito_test` (`setupTestDB`, `createUsers` в `wallet_integration_test.go`). Обычный `go test` их пропускает. Запуск:
> ```bash
> go test -tags integration ./internal/repository/postgres/ -run Friends -v
> ```
> Нужен запущенный `docker compose`: Postgres на порту 5433.

### Практика

Файл `internal/repository/postgres/friends.go`:

1. `orderedPair(a, b uuid.UUID) (low, high uuid.UUID)` через `bytes.Compare`.
2. `Request(ctx, from, to) (model.FriendState, error)` — upsert из теории, плюс разбор случая «строк не вернулось».
3. `Accept(ctx, me, from) error` — условный `UPDATE` и `RowsAffected`.
4. `Remove(ctx, me, other) error` — `DELETE` пары.
5. `List(ctx, me) ([]model.FriendLink, error)` — один запрос с `CASE`, без удалённых, по алфавиту без учёта регистра.

Интеграционные тесты `friends_integration_test.go`:
- полный путь: заявка → повтор (ошибка) → отправитель пытается принять сам (ошибка) → получатель принимает → заявка другу (ошибка) → удаление → повторное удаление (не ошибка);
- встречная заявка сразу даёт `accepted`;
- **одновременные встречные заявки** из двух горутин дают одну строку `accepted`;
- удалённый пользователь пропадает из списка;
- база сама отвергает «плохие» строки.

> [!warning] Ловушки
> - `pgx.ErrNoRows` при `QueryRow(...).Scan` — не ошибка базы, а «строк нет». Проверяй через `errors.Is`.
> - Забыть `orderedPair` в одном из методов: всё работает, пока id случайно упорядочены, и ломается в половине случаев.
> - `Scan` в `model.FriendState`: pgx умеет сканировать текст в именованный строковый тип, отдельная конвертация не нужна.
> - Пустой список должен быть `[]`, а не `nil`. Начни срез с `make([]model.FriendLink, 0)`.

### Проверь себя

> [!question]- Что вернёт `Request`, если Алиса шлёт заявку Бобу второй раз?
> Upsert упрётся в конфликт, условие `WHERE` ложно (`requested_by` совпадает), строк не вернётся. Дальше `SELECT` покажет `pending`, значит, `ErrFriendRequestExists`.

> [!question]- Почему тест на одновременные заявки не нуждается в `time.Sleep`?
> Он не ждёт «пока что-то случится». Две горутины стартуют одновременно, `WaitGroup` ждёт обе, и проверяется итог в базе. Если бы гонка была, хотя бы иногда получились бы ошибка или две строки. Запусти его с `-count=20`.

> [!question]- Почему `Accept` проверяет `requested_by = $3`, а не просто пару?
> Иначе отправитель смог бы «принять» собственную заявку и стать другом без согласия второго.

> [!question]- Чем плох `OR` в `WHERE` списка, и что можно сделать?
> `OR` по двум колонкам Postgres обычно выполняет через два индекса и объединение (BitmapOr), и для наших объёмов это нормально. Альтернатива — `UNION ALL` двух запросов, по одному на колонку. Это быстрее на огромных таблицах, но длиннее и сложнее читать.

> [!example]- Решение: repository/postgres/friends.go
> ```go
> package postgres
>
> import (
> 	"bytes"
> 	"context"
> 	"errors"
> 	"fmt"
>
> 	"bito/internal/model"
>
> 	"github.com/google/uuid"
> 	"github.com/jackc/pgx/v5"
> 	"github.com/jackc/pgx/v5/pgxpool"
> )
>
> type FriendRepository struct {
> 	pool *pgxpool.Pool
> }
>
> func NewFriendRepository(pool *pgxpool.Pool) *FriendRepository {
> 	return &FriendRepository{pool: pool}
> }
>
> func orderedPair(a, b uuid.UUID) (low, high uuid.UUID) {
> 	if bytes.Compare(a[:], b[:]) < 0 {
> 		return a, b
> 	}
> 	return b, a
> }
>
> func (r *FriendRepository) Request(ctx context.Context, from, to uuid.UUID) (model.FriendState, error) {
> 	low, high := orderedPair(from, to)
> 	const upsertSQL = `
> 		INSERT INTO friendships (user_low, user_high, requested_by)
> 		VALUES ($1, $2, $3)
> 		ON CONFLICT (user_low, user_high) DO UPDATE
> 		SET status = 'accepted', accepted_at = NOW()
> 		WHERE friendships.status = 'pending'
> 		  AND friendships.requested_by <> EXCLUDED.requested_by
> 		RETURNING status
> 	`
> 	var state model.FriendState
> 	err := r.pool.QueryRow(ctx, upsertSQL, low, high, from).Scan(&state)
> 	if err == nil {
> 		return state, nil
> 	}
> 	if !errors.Is(err, pgx.ErrNoRows) {
> 		return "", fmt.Errorf("friend repo: request: %w", err)
> 	}
>
> 	const currentSQL = `SELECT status FROM friendships WHERE user_low = $1 AND user_high = $2`
> 	if err = r.pool.QueryRow(ctx, currentSQL, low, high).Scan(&state); err != nil {
> 		return "", fmt.Errorf("friend repo: read pair: %w", err)
> 	}
> 	if state == model.FriendAccepted {
> 		return "", model.ErrAlreadyFriends
> 	}
> 	return "", model.ErrFriendRequestExists
> }
>
> func (r *FriendRepository) Accept(ctx context.Context, me, from uuid.UUID) error {
> 	low, high := orderedPair(me, from)
> 	const acceptSQL = `
> 		UPDATE friendships
> 		SET status = 'accepted', accepted_at = NOW()
> 		WHERE user_low = $1 AND user_high = $2 AND status = 'pending' AND requested_by = $3
> 	`
> 	tag, err := r.pool.Exec(ctx, acceptSQL, low, high, from)
> 	if err != nil {
> 		return fmt.Errorf("friend repo: accept: %w", err)
> 	}
> 	if tag.RowsAffected() == 0 {
> 		return model.ErrFriendRequestNotFound
> 	}
> 	return nil
> }
>
> func (r *FriendRepository) Remove(ctx context.Context, me, other uuid.UUID) error {
> 	low, high := orderedPair(me, other)
> 	const removeSQL = `DELETE FROM friendships WHERE user_low = $1 AND user_high = $2`
> 	if _, err := r.pool.Exec(ctx, removeSQL, low, high); err != nil {
> 		return fmt.Errorf("friend repo: remove: %w", err)
> 	}
> 	return nil
> }
>
> func (r *FriendRepository) List(ctx context.Context, me uuid.UUID) ([]model.FriendLink, error) {
> 	const listSQL = `
> 		SELECT u.id, u.name, f.status, f.requested_by = $1, COALESCE(f.accepted_at, f.created_at)
> 		FROM friendships f
> 		JOIN users u ON u.id = CASE WHEN f.user_low = $1 THEN f.user_high ELSE f.user_low END
> 		WHERE (f.user_low = $1 OR f.user_high = $1) AND u.deleted_at IS NULL
> 		ORDER BY LOWER(u.name)
> 	`
> 	rows, err := r.pool.Query(ctx, listSQL, me)
> 	if err != nil {
> 		return nil, fmt.Errorf("friend repo: list: %w", err)
> 	}
> 	defer rows.Close()
>
> 	links := make([]model.FriendLink, 0)
> 	for rows.Next() {
> 		var l model.FriendLink
> 		if err = rows.Scan(&l.UserID, &l.Username, &l.State, &l.Outgoing, &l.Since); err != nil {
> 			return nil, fmt.Errorf("friend repo: scan: %w", err)
> 		}
> 		links = append(links, l)
> 	}
> 	if err = rows.Err(); err != nil {
> 		return nil, fmt.Errorf("friend repo: read list: %w", err)
> 	}
> 	return links, nil
> }
> ```

> [!example]- Решение: интеграционные тесты
> ```go
> //go:build integration
>
> package postgres
>
> import (
> 	"context"
> 	"errors"
> 	"sync"
> 	"testing"
>
> 	"bito/internal/model"
> )
>
> func TestFriends_RequestAcceptRemove(t *testing.T) {
> 	pool := setupTestDB(t)
> 	repo := NewFriendRepository(pool)
> 	ctx := context.Background()
> 	ids := createUsers(t, pool, 2)
> 	alice, bob := ids[0], ids[1]
>
> 	state, err := repo.Request(ctx, alice, bob)
> 	if err != nil || state != model.FriendPending {
> 		t.Fatalf("request = %q, %v; want pending", state, err)
> 	}
> 	if _, err = repo.Request(ctx, alice, bob); !errors.Is(err, model.ErrFriendRequestExists) {
> 		t.Fatalf("repeat request err = %v, want ErrFriendRequestExists", err)
> 	}
> 	if err = repo.Accept(ctx, alice, bob); !errors.Is(err, model.ErrFriendRequestNotFound) {
> 		t.Fatalf("sender accepted own request: err = %v", err)
> 	}
>
> 	aliceView, err := repo.List(ctx, alice)
> 	if err != nil || len(aliceView) != 1 || !aliceView[0].Outgoing || aliceView[0].UserID != bob {
> 		t.Fatalf("alice list = %+v, %v; want one outgoing to bob", aliceView, err)
> 	}
> 	bobView, err := repo.List(ctx, bob)
> 	if err != nil || len(bobView) != 1 || bobView[0].Outgoing || bobView[0].UserID != alice {
> 		t.Fatalf("bob list = %+v, %v; want one incoming from alice", bobView, err)
> 	}
>
> 	if err = repo.Accept(ctx, bob, alice); err != nil {
> 		t.Fatalf("accept: %v", err)
> 	}
> 	if _, err = repo.Request(ctx, bob, alice); !errors.Is(err, model.ErrAlreadyFriends) {
> 		t.Fatalf("request to a friend err = %v, want ErrAlreadyFriends", err)
> 	}
> 	bobView, _ = repo.List(ctx, bob)
> 	if len(bobView) != 1 || bobView[0].State != model.FriendAccepted {
> 		t.Fatalf("bob list after accept = %+v", bobView)
> 	}
>
> 	if err = repo.Remove(ctx, bob, alice); err != nil {
> 		t.Fatalf("remove: %v", err)
> 	}
> 	if err = repo.Remove(ctx, bob, alice); err != nil {
> 		t.Fatalf("second remove should be a no-op: %v", err)
> 	}
> 	if list, _ := repo.List(ctx, alice); len(list) != 0 {
> 		t.Fatalf("alice list after remove = %+v, want empty", list)
> 	}
> }
>
> func TestFriends_CounterRequestBecomesFriendship(t *testing.T) {
> 	pool := setupTestDB(t)
> 	repo := NewFriendRepository(pool)
> 	ctx := context.Background()
> 	ids := createUsers(t, pool, 2)
>
> 	if _, err := repo.Request(ctx, ids[0], ids[1]); err != nil {
> 		t.Fatalf("request: %v", err)
> 	}
> 	state, err := repo.Request(ctx, ids[1], ids[0])
> 	if err != nil || state != model.FriendAccepted {
> 		t.Fatalf("counter request = %q, %v; want accepted", state, err)
> 	}
> }
>
> func TestFriends_SimultaneousRequestsMakeOneRow(t *testing.T) {
> 	pool := setupTestDB(t)
> 	repo := NewFriendRepository(pool)
> 	ctx := context.Background()
> 	ids := createUsers(t, pool, 2)
>
> 	var wg sync.WaitGroup
> 	errs := make([]error, 2)
> 	for i := range 2 {
> 		wg.Add(1)
> 		go func() {
> 			defer wg.Done()
> 			_, errs[i] = repo.Request(ctx, ids[i], ids[1-i])
> 		}()
> 	}
> 	wg.Wait()
> 	for _, err := range errs {
> 		if err != nil {
> 			t.Fatalf("simultaneous request failed: %v", err)
> 		}
> 	}
>
> 	var rows int
> 	var status string
> 	if err := pool.QueryRow(ctx, `SELECT COUNT(*), MAX(status) FROM friendships`).Scan(&rows, &status); err != nil {
> 		t.Fatal(err)
> 	}
> 	if rows != 1 || status != "accepted" {
> 		t.Fatalf("rows = %d, status = %q; want one accepted row", rows, status)
> 	}
> }
>
> func TestFriends_DeletedUserDisappears(t *testing.T) {
> 	pool := setupTestDB(t)
> 	repo := NewFriendRepository(pool)
> 	ctx := context.Background()
> 	ids := createUsers(t, pool, 2)
>
> 	if _, err := repo.Request(ctx, ids[0], ids[1]); err != nil {
> 		t.Fatal(err)
> 	}
> 	if err := NewUserRepository(pool).Delete(ctx, ids[1]); err != nil {
> 		t.Fatal(err)
> 	}
> 	if list, _ := repo.List(ctx, ids[0]); len(list) != 0 {
> 		t.Fatalf("deleted user still listed: %+v", list)
> 	}
> }
>
> func TestFriends_DatabaseRejectsBrokenRows(t *testing.T) {
> 	pool := setupTestDB(t)
> 	ctx := context.Background()
> 	ids := createUsers(t, pool, 2)
> 	low, high := orderedPair(ids[0], ids[1])
>
> 	cases := map[string][]any{
> 		"reversed pair":      {high, low, high, "pending"},
> 		"self":               {low, low, low, "pending"},
> 		"stranger requested": {low, high, createUsers(t, pool, 1)[0], "pending"},
> 		"unknown status":     {low, high, low, "blocked"},
> 	}
> 	for name, args := range cases {
> 		_, err := pool.Exec(ctx, `INSERT INTO friendships (user_low, user_high, requested_by, status) VALUES ($1, $2, $3, $4)`, args...)
> 		if err == nil {
> 			t.Errorf("%s: insert succeeded, want a constraint violation", name)
> 		}
> 	}
> }
> ```

### Готово, когда

- [ ] `go test -tags integration ./internal/repository/postgres/ -run Friends -count=10` зелёный
- [ ] можешь объяснить upsert с `WHERE` и чем он лучше «SELECT, потом INSERT»

---

## Шаг 4. Сервис

### Зачем

Правила, которые не про SQL: найти игрока по имени, запретить дружбу с собой, разложить связи на три списка, добавить онлайн-статусы и **сообщить второму игроку**, что у него что-то изменилось.

### Теория

> [!note] Инверсия зависимостей
> Сервису нужны четыре вещи: хранилище дружб, поиск игрока по имени, статусы присутствия и способ уведомить. Он **не** импортирует `postgres` и `ws`. Вместо этого объявляет четыре маленьких интерфейса, а в `app.go` в них подставляются настоящие реализации: `FriendRepository`, `UserRepository`, `Hub`.
> Зависимость направлена «внутрь»: `ws` знает про `model`, `service` знает про `model`, но `service` ничего не знает про сокеты. Поэтому сервис тестируется фейками за миллисекунды.

> [!note] Один объект — два интерфейса
> `Hub` будет реализовывать и `PresenceSource`, и `FriendNotifier`. В `app.go` он передаётся дважды: `NewFriendService(repo, userRepo, hub, hub)`. Сервис видит две разные роли, и если когда-нибудь уведомления уедут, например, в Redis pub/sub, поменяется одна строка.

> [!note] Уведомлять только после успеха
> Сначала запись в базу, потом уведомление. Если запись упала, уведомлять не о чем. Порядок важен и в тестах: проверь, что при ошибке репозитория уведомлений нет.

> [!note] Нормализация ввода
> Имя из формы может прийти с пробелами по краям. Поиск по имени в `GetByName` уже регистронезависимый (`LOWER(u.name) = LOWER($1)`), осталось убрать пробелы. Пустое имя сразу считаем «не найден», в базу не ходим.

### Практика

Файл `internal/service/friends.go`:
- интерфейсы `FriendRepository`, `UserFinder` (только `GetByName`), `PresenceSource`, `FriendNotifier`;
- `FriendService` и конструктор;
- `Overview(ctx, me)`: список из репозитория → разложить на `Friends`, `Incoming`, `Outgoing` → статусы только для друзей (заявкам онлайн не показываем) → тем, кого нет в ответе, поставить `offline`;
- `Request(ctx, me, username)`: обрезать пробелы → найти → не себя → репозиторий → уведомить получателя;
- `Accept(ctx, me, from)` → уведомить `from`;
- `Remove(ctx, me, other)` → уведомить `other`.

Тесты `friends_test.go` на фейках: нашёл по имени с пробелами и другим регистром; себя и неизвестного отверг, не трогая репозиторий; при ошибке не уведомил; `Overview` правильно разложил и проставил статусы.

> [!warning] Ловушки
> - Три списка должны быть `[]`, а не `nil`, иначе в JSON уйдёт `null` и фронт упадёт на `.length`.
> - Показывать онлайн-статус людям, которые ещё не друзья, — утечка приватности. Статусы только для `accepted`.

### Проверь себя

> [!question]- Почему `UserFinder` объявлен с одним методом, хотя у `UserRepository` их много?
> Сервису друзей нужен только поиск по имени. Маленький интерфейс проще подменить фейком и честно показывает, от чего сервис зависит. Это «принцип разделения интерфейсов».

> [!question]- Почему не уведомлять и самого себя?
> Свой экран клиент обновит сам, сразу после ответа на запрос. Уведомление нужно второй стороне, у которой страница ничего не знает.

> [!question]- Где проверять «не себя»: в сервисе или в базе?
> В обоих. `CHECK (user_low < user_high)` — последняя линия обороны. Сервис проверяет раньше, чтобы вернуть понятную ошибку, а не нарушение ограничения базы.

> [!example]- Решение: service/friends.go
> ```go
> package service
>
> import (
> 	"context"
> 	"strings"
>
> 	"bito/internal/model"
>
> 	"github.com/google/uuid"
> )
>
> type FriendRepository interface {
> 	Request(ctx context.Context, from, to uuid.UUID) (model.FriendState, error)
> 	Accept(ctx context.Context, me, from uuid.UUID) error
> 	Remove(ctx context.Context, me, other uuid.UUID) error
> 	List(ctx context.Context, me uuid.UUID) ([]model.FriendLink, error)
> }
>
> type UserFinder interface {
> 	GetByName(ctx context.Context, name string) (*model.User, error)
> }
>
> type PresenceSource interface {
> 	Presence(ids []uuid.UUID) map[uuid.UUID]model.Presence
> }
>
> type FriendNotifier interface {
> 	FriendsChanged(ids ...uuid.UUID)
> }
>
> type FriendService struct {
> 	repo     FriendRepository
> 	users    UserFinder
> 	presence PresenceSource
> 	notify   FriendNotifier
> }
>
> func NewFriendService(repo FriendRepository, users UserFinder, presence PresenceSource, notify FriendNotifier) *FriendService {
> 	return &FriendService{repo: repo, users: users, presence: presence, notify: notify}
> }
>
> func (s *FriendService) Overview(ctx context.Context, me uuid.UUID) (model.FriendsOverview, error) {
> 	links, err := s.repo.List(ctx, me)
> 	if err != nil {
> 		return model.FriendsOverview{}, err
> 	}
>
> 	out := model.FriendsOverview{
> 		Friends:  []model.FriendLink{},
> 		Incoming: []model.FriendLink{},
> 		Outgoing: []model.FriendLink{},
> 	}
> 	ids := make([]uuid.UUID, 0, len(links))
> 	for _, l := range links {
> 		switch {
> 		case l.State == model.FriendAccepted:
> 			out.Friends = append(out.Friends, l)
> 			ids = append(ids, l.UserID)
> 		case l.Outgoing:
> 			out.Outgoing = append(out.Outgoing, l)
> 		default:
> 			out.Incoming = append(out.Incoming, l)
> 		}
> 	}
>
> 	statuses := s.presence.Presence(ids)
> 	for i := range out.Friends {
> 		out.Friends[i].Presence = statuses[out.Friends[i].UserID]
> 		if out.Friends[i].Presence == "" {
> 			out.Friends[i].Presence = model.PresenceOffline
> 		}
> 	}
> 	return out, nil
> }
>
> func (s *FriendService) Request(ctx context.Context, me uuid.UUID, username string) (model.FriendState, error) {
> 	name := strings.TrimSpace(username)
> 	if name == "" {
> 		return "", model.ErrUserNotFound
> 	}
> 	target, err := s.users.GetByName(ctx, name)
> 	if err != nil {
> 		return "", err
> 	}
> 	if target.ID == me {
> 		return "", model.ErrFriendSelf
> 	}
>
> 	state, err := s.repo.Request(ctx, me, target.ID)
> 	if err != nil {
> 		return "", err
> 	}
> 	s.notify.FriendsChanged(target.ID)
> 	return state, nil
> }
>
> func (s *FriendService) Accept(ctx context.Context, me, from uuid.UUID) error {
> 	if err := s.repo.Accept(ctx, me, from); err != nil {
> 		return err
> 	}
> 	s.notify.FriendsChanged(from)
> 	return nil
> }
>
> func (s *FriendService) Remove(ctx context.Context, me, other uuid.UUID) error {
> 	if err := s.repo.Remove(ctx, me, other); err != nil {
> 		return err
> 	}
> 	s.notify.FriendsChanged(other)
> 	return nil
> }
> ```

> [!example]- Решение: service/friends_test.go
> ```go
> package service
>
> import (
> 	"context"
> 	"errors"
> 	"strings"
> 	"testing"
> 	"time"
>
> 	"bito/internal/model"
>
> 	"github.com/google/uuid"
> )
>
> type fakeFriendRepo struct {
> 	links     []model.FriendLink
> 	requested [][2]uuid.UUID
> 	state     model.FriendState
> 	err       error
> }
>
> func (f *fakeFriendRepo) Request(_ context.Context, from, to uuid.UUID) (model.FriendState, error) {
> 	f.requested = append(f.requested, [2]uuid.UUID{from, to})
> 	return f.state, f.err
> }
>
> func (f *fakeFriendRepo) Accept(context.Context, uuid.UUID, uuid.UUID) error { return f.err }
> func (f *fakeFriendRepo) Remove(context.Context, uuid.UUID, uuid.UUID) error { return f.err }
> func (f *fakeFriendRepo) List(context.Context, uuid.UUID) ([]model.FriendLink, error) {
> 	return f.links, f.err
> }
>
> type fakeUsers map[string]*model.User
>
> func (f fakeUsers) GetByName(_ context.Context, name string) (*model.User, error) {
> 	if u, ok := f[strings.ToLower(name)]; ok {
> 		return u, nil
> 	}
> 	return nil, model.ErrUserNotFound
> }
>
> type fakePresence map[uuid.UUID]model.Presence
>
> func (f fakePresence) Presence([]uuid.UUID) map[uuid.UUID]model.Presence { return f }
>
> type fakeNotifier struct{ got []uuid.UUID }
>
> func (f *fakeNotifier) FriendsChanged(ids ...uuid.UUID) { f.got = append(f.got, ids...) }
>
> func TestFriendService_RequestFindsUserByName(t *testing.T) {
> 	me, bob := uuid.New(), uuid.New()
> 	repo := &fakeFriendRepo{state: model.FriendPending}
> 	notify := &fakeNotifier{}
> 	s := NewFriendService(repo, fakeUsers{"bob": {ID: bob, Name: "Bob"}}, fakePresence{}, notify)
>
> 	state, err := s.Request(context.Background(), me, "  BOB ")
> 	if err != nil || state != model.FriendPending {
> 		t.Fatalf("request = %q, %v", state, err)
> 	}
> 	if len(repo.requested) != 1 || repo.requested[0] != [2]uuid.UUID{me, bob} {
> 		t.Fatalf("repo got %v, want me→bob", repo.requested)
> 	}
> 	if len(notify.got) != 1 || notify.got[0] != bob {
> 		t.Fatalf("notified %v, want bob", notify.got)
> 	}
> }
>
> func TestFriendService_RequestRejectsSelfAndUnknown(t *testing.T) {
> 	me := uuid.New()
> 	repo := &fakeFriendRepo{}
> 	notify := &fakeNotifier{}
> 	s := NewFriendService(repo, fakeUsers{"me": {ID: me, Name: "Me"}}, fakePresence{}, notify)
>
> 	if _, err := s.Request(context.Background(), me, "me"); !errors.Is(err, model.ErrFriendSelf) {
> 		t.Fatalf("self err = %v", err)
> 	}
> 	if _, err := s.Request(context.Background(), me, "ghost"); !errors.Is(err, model.ErrUserNotFound) {
> 		t.Fatalf("unknown err = %v", err)
> 	}
> 	if _, err := s.Request(context.Background(), me, "   "); !errors.Is(err, model.ErrUserNotFound) {
> 		t.Fatalf("blank err = %v", err)
> 	}
> 	if len(repo.requested) != 0 || len(notify.got) != 0 {
> 		t.Fatalf("repo or notifier touched on a rejected request")
> 	}
> }
>
> func TestFriendService_FailedRequestDoesNotNotify(t *testing.T) {
> 	bob := uuid.New()
> 	notify := &fakeNotifier{}
> 	repo := &fakeFriendRepo{err: model.ErrFriendRequestExists}
> 	s := NewFriendService(repo, fakeUsers{"bob": {ID: bob}}, fakePresence{}, notify)
>
> 	if _, err := s.Request(context.Background(), uuid.New(), "bob"); !errors.Is(err, model.ErrFriendRequestExists) {
> 		t.Fatalf("err = %v", err)
> 	}
> 	if len(notify.got) != 0 {
> 		t.Fatalf("notified on failure: %v", notify.got)
> 	}
> }
>
> func TestFriendService_OverviewSplitsAndAddsPresence(t *testing.T) {
> 	friend, gone, in, out := uuid.New(), uuid.New(), uuid.New(), uuid.New()
> 	repo := &fakeFriendRepo{links: []model.FriendLink{
> 		{UserID: friend, State: model.FriendAccepted, Since: time.Now()},
> 		{UserID: gone, State: model.FriendAccepted},
> 		{UserID: in, State: model.FriendPending},
> 		{UserID: out, State: model.FriendPending, Outgoing: true},
> 	}}
> 	s := NewFriendService(repo, fakeUsers{}, fakePresence{friend: model.PresencePlaying}, &fakeNotifier{})
>
> 	o, err := s.Overview(context.Background(), uuid.New())
> 	if err != nil {
> 		t.Fatal(err)
> 	}
> 	if len(o.Friends) != 2 || len(o.Incoming) != 1 || len(o.Outgoing) != 1 {
> 		t.Fatalf("overview = %+v", o)
> 	}
> 	if o.Friends[0].Presence != model.PresencePlaying || o.Friends[1].Presence != model.PresenceOffline {
> 		t.Fatalf("presence = %q, %q", o.Friends[0].Presence, o.Friends[1].Presence)
> 	}
> 	if o.Incoming[0].UserID != in || o.Outgoing[0].UserID != out {
> 		t.Fatalf("requests split wrong: %+v", o)
> 	}
> }
> ```

### Готово, когда

- [ ] `go test -race ./internal/service/ -run Friend` зелёный
- [ ] сервис не импортирует ни `postgres`, ни `ws`

---

## Шаг 5. HTTP

### Зачем

Превратить сервис в API: разобрать запрос, достать игрока из токена, перевести ошибки в понятные статусы и русские сообщения.

### Теория

> [!note] REST: ресурсы и глаголы
> - `GET /friends` — прочитать, ничего не меняет.
> - `POST /friends/requests` — создать заявку. По имени, потому что id друга игрок не знает.
> - `POST /friends/requests/{id}/accept` — действие над заявкой. Это не «создать ресурс», а команда, поэтому `POST`.
> - `DELETE /friends/{id}` — удалить связь, какой бы она ни была.
>
> В путях `{id}` — это id **другого игрока**, а не id строки в таблице. У строки и нет своего id: её ключ — пара. Удобно и фронту, ведь id игрока он уже знает.

> [!note] Коды ответов
>
> | Ситуация | Код | Почему |
> |---|---|---|
> | заявка создана | `201 Created` | появилось новое |
> | встречная заявка, стали друзьями | `200 OK` | новое не создано, обновилось существующее |
> | принял, удалил | `204 No Content` | всё хорошо, тело не нужно |
> | нет такого игрока, нет такой заявки | `404 Not Found` | |
> | дружба с собой, кривой id | `400 Bad Request` | запрос неверен сам по себе |
> | уже друзья, заявка уже есть | `409 Conflict` | запрос нормальный, но противоречит текущему состоянию |
> | без токена | `401 Unauthorized` | |

> [!note] Ошибки → статусы в одном месте
> Одна функция `respondFriendError` с `switch` по `errors.Is`. Обработчики не думают о статусах, а неизвестная ошибка уходит в лог и отвечает 500 без подробностей. Подробности базы клиенту не показываем.

> [!note] DTO отдельно от модели
> `FriendResponse` — то, что видит фронт: строки, `camelCase`, дата в RFC 3339. `model.FriendLink` — внутреннее представление. Если модель поменяется, API останется прежним.

> [!note] Тесты обработчика без базы
> Обработчик зависит от интерфейса `FriendService`, значит, в тесте подставляем заглушку, которая возвращает нужную ошибку. Табличный тест проверяет все коды ответов за раз. Роутер и авторизация настоящие, как в `user_handler_test.go`.

### Практика

1. `internal/delivery/http/friend_handler.go`: интерфейс `FriendService`, DTO, `FriendHandler` с методами `List`, `Request`, `Accept`, `Remove`, общий `pair()` для разбора `{id}`, `respondFriendError`, конвертеры.
2. `server.go`: поле `friendHandler`, поле `Friend` в `Handlers`, группа маршрутов `/api/v1/friends` с `AuthMiddleware`.
3. `app.go`: собрать репозиторий, сервис и обработчик.
4. `friend_handler_test.go`: без токена → 401; форма ответа списка, пустые списки `[]`; табличный тест кодов для заявки; `{id}` из пути доходит до сервиса; кривой id → 400.

> [!warning] Ловушки
> - `chi.URLParam(r, "id")` работает, только если имя в шаблоне маршрута тоже `{id}`.
> - `decodeJSON` в проекте запрещает лишние поля (`DisallowUnknownFields`). Тело `{"name":"Боб"}` вместо `{"username":...}` получит 400.
> - `hub` в `app.go` создаётся **после** экономики, а обработчики собираются ещё позже. Сервис друзей собирай там же, где остальные обработчики.

### Проверь себя

> [!question]- Почему встречная заявка отвечает 200, а не 201?
> Новый ресурс не появился: существующая заявка стала дружбой. Фронт по статусу в теле (`accepted`) покажет «теперь вы друзья» вместо «заявка отправлена».

> [!question]- Не опасно ли, что 404 «игрок не найден» выдаёт, существует ли имя?
> Имена в игре публичные: они видны за столом, в чате и в списке столов. Скрывать их существование бессмысленно. С почтой всё было бы иначе, её мы не раскрываем.

> [!question]- Зачем отдельный `pair()`?
> `Accept` и `Remove` одинаково достают меня из токена и другого игрока из пути. Общий помощник убирает повтор и делает так, что ошибка разбора id везде одна и та же.

> [!example]- Решение: delivery/http/friend_handler.go
> ```go
> package http
>
> import (
> 	"context"
> 	"errors"
> 	"log"
> 	"net/http"
> 	"time"
>
> 	"bito/internal/model"
>
> 	"github.com/go-chi/chi/v5"
> 	"github.com/google/uuid"
> )
>
> type FriendService interface {
> 	Overview(ctx context.Context, me uuid.UUID) (model.FriendsOverview, error)
> 	Request(ctx context.Context, me uuid.UUID, username string) (model.FriendState, error)
> 	Accept(ctx context.Context, me, from uuid.UUID) error
> 	Remove(ctx context.Context, me, other uuid.UUID) error
> }
>
> type FriendResponse struct {
> 	ID       string `json:"id"`
> 	Username string `json:"username"`
> 	Status   string `json:"status"`
> 	Since    string `json:"since"`
> }
>
> type FriendRequestResponse struct {
> 	ID       string `json:"id"`
> 	Username string `json:"username"`
> 	At       string `json:"at"`
> }
>
> type FriendsResponse struct {
> 	Friends  []FriendResponse        `json:"friends"`
> 	Incoming []FriendRequestResponse `json:"incoming"`
> 	Outgoing []FriendRequestResponse `json:"outgoing"`
> }
>
> type AddFriendRequest struct {
> 	Username string `json:"username"`
> }
>
> type AddFriendResponse struct {
> 	Status string `json:"status"`
> }
>
> type FriendHandler struct {
> 	friends FriendService
> }
>
> func NewFriendHandler(friends FriendService) *FriendHandler {
> 	return &FriendHandler{friends: friends}
> }
>
> func (fh *FriendHandler) List(w http.ResponseWriter, r *http.Request) {
> 	me, ok := UserIDFromContext(r.Context())
> 	if !ok {
> 		respondError(w, http.StatusUnauthorized, "unauthorized")
> 		return
> 	}
>
> 	overview, err := fh.friends.Overview(r.Context(), me)
> 	if err != nil {
> 		respondFriendError(w, me, err)
> 		return
> 	}
> 	respondJSON(w, http.StatusOK, toFriendsResponse(overview))
> }
>
> func (fh *FriendHandler) Request(w http.ResponseWriter, r *http.Request) {
> 	me, ok := UserIDFromContext(r.Context())
> 	if !ok {
> 		respondError(w, http.StatusUnauthorized, "unauthorized")
> 		return
> 	}
>
> 	var req AddFriendRequest
> 	if !decodeJSON(w, r, &req) {
> 		return
> 	}
>
> 	state, err := fh.friends.Request(r.Context(), me, req.Username)
> 	if err != nil {
> 		respondFriendError(w, me, err)
> 		return
> 	}
>
> 	status := http.StatusCreated
> 	if state == model.FriendAccepted {
> 		status = http.StatusOK
> 	}
> 	respondJSON(w, status, AddFriendResponse{Status: string(state)})
> }
>
> func (fh *FriendHandler) Accept(w http.ResponseWriter, r *http.Request) {
> 	me, other, ok := fh.pair(w, r)
> 	if !ok {
> 		return
> 	}
> 	if err := fh.friends.Accept(r.Context(), me, other); err != nil {
> 		respondFriendError(w, me, err)
> 		return
> 	}
> 	w.WriteHeader(http.StatusNoContent)
> }
>
> func (fh *FriendHandler) Remove(w http.ResponseWriter, r *http.Request) {
> 	me, other, ok := fh.pair(w, r)
> 	if !ok {
> 		return
> 	}
> 	if err := fh.friends.Remove(r.Context(), me, other); err != nil {
> 		respondFriendError(w, me, err)
> 		return
> 	}
> 	w.WriteHeader(http.StatusNoContent)
> }
>
> func (fh *FriendHandler) pair(w http.ResponseWriter, r *http.Request) (me, other uuid.UUID, ok bool) {
> 	me, ok = UserIDFromContext(r.Context())
> 	if !ok {
> 		respondError(w, http.StatusUnauthorized, "unauthorized")
> 		return uuid.Nil, uuid.Nil, false
> 	}
> 	other, err := uuid.Parse(chi.URLParam(r, "id"))
> 	if err != nil {
> 		respondError(w, http.StatusBadRequest, "неверный id игрока")
> 		return uuid.Nil, uuid.Nil, false
> 	}
> 	return me, other, true
> }
>
> func respondFriendError(w http.ResponseWriter, me uuid.UUID, err error) {
> 	switch {
> 	case errors.Is(err, model.ErrUserNotFound):
> 		respondError(w, http.StatusNotFound, "игрок с таким именем не найден")
> 	case errors.Is(err, model.ErrFriendSelf):
> 		respondError(w, http.StatusBadRequest, "нельзя добавить в друзья самого себя")
> 	case errors.Is(err, model.ErrAlreadyFriends):
> 		respondError(w, http.StatusConflict, "вы уже друзья")
> 	case errors.Is(err, model.ErrFriendRequestExists):
> 		respondError(w, http.StatusConflict, "заявка уже отправлена")
> 	case errors.Is(err, model.ErrFriendRequestNotFound):
> 		respondError(w, http.StatusNotFound, "заявка не найдена")
> 	default:
> 		log.Printf("friends: %s: %v", me, err)
> 		respondError(w, http.StatusInternalServerError, "internal server error")
> 	}
> }
>
> func toFriendsResponse(o model.FriendsOverview) FriendsResponse {
> 	resp := FriendsResponse{
> 		Friends:  make([]FriendResponse, 0, len(o.Friends)),
> 		Incoming: make([]FriendRequestResponse, 0, len(o.Incoming)),
> 		Outgoing: make([]FriendRequestResponse, 0, len(o.Outgoing)),
> 	}
> 	for _, f := range o.Friends {
> 		resp.Friends = append(resp.Friends, FriendResponse{
> 			ID:       f.UserID.String(),
> 			Username: f.Username,
> 			Status:   string(f.Presence),
> 			Since:    f.Since.Format(time.RFC3339),
> 		})
> 	}
> 	for _, l := range o.Incoming {
> 		resp.Incoming = append(resp.Incoming, toFriendRequestResponse(l))
> 	}
> 	for _, l := range o.Outgoing {
> 		resp.Outgoing = append(resp.Outgoing, toFriendRequestResponse(l))
> 	}
> 	return resp
> }
>
> func toFriendRequestResponse(l model.FriendLink) FriendRequestResponse {
> 	return FriendRequestResponse{ID: l.UserID.String(), Username: l.Username, At: l.Since.Format(time.RFC3339)}
> }
> ```

> [!example]- Решение: маршруты и сборка
> В `server.go` рядом с остальными группами:
> ```go
> if s.friendHandler != nil {
> 	s.router.Route("/api/v1/friends", func(r chi.Router) {
> 		r.Use(AuthMiddleware(s.tokens))
> 		r.Get("/", s.friendHandler.List)
> 		r.Post("/requests", s.friendHandler.Request)
> 		r.Post("/requests/{id}/accept", s.friendHandler.Accept)
> 		r.Delete("/{id}", s.friendHandler.Remove)
> 	})
> }
> ```
> Плюс поле `friendHandler *FriendHandler` в `Server`, поле `Friend *FriendHandler` в `Handlers` и `friendHandler: h.Friend` в `NewServer`.
>
> В `app.go`, в списке обработчиков:
> ```go
> Friend: deliveryHTTP.NewFriendHandler(service.NewFriendService(postgres.NewFriendRepository(pool), userRepo, hub, hub)),
> ```

> [!example]- Решение: delivery/http/friend_handler_test.go
> ```go
> package http
>
> import (
> 	"context"
> 	"encoding/json"
> 	"net/http"
> 	"testing"
> 	"time"
>
> 	"bito/internal/delivery/ws"
> 	"bito/internal/model"
> 	"bito/internal/pkg/jwt"
>
> 	"github.com/google/uuid"
> )
>
> type stubFriends struct {
> 	overview model.FriendsOverview
> 	state    model.FriendState
> 	err      error
> 	lastName string
> 	lastID   uuid.UUID
> }
>
> func (s *stubFriends) Overview(context.Context, uuid.UUID) (model.FriendsOverview, error) {
> 	return s.overview, s.err
> }
>
> func (s *stubFriends) Request(_ context.Context, _ uuid.UUID, username string) (model.FriendState, error) {
> 	s.lastName = username
> 	return s.state, s.err
> }
>
> func (s *stubFriends) Accept(_ context.Context, _, from uuid.UUID) error {
> 	s.lastID = from
> 	return s.err
> }
>
> func (s *stubFriends) Remove(_ context.Context, _, other uuid.UUID) error {
> 	s.lastID = other
> 	return s.err
> }
>
> func newFriendsRouter(t *testing.T, stub *stubFriends) (http.Handler, string) {
> 	t.Helper()
> 	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)
> 	hub := ws.NewHub(tokenManager, nil, nil)
> 	server := NewServer(":0", hub, Handlers{Friend: NewFriendHandler(stub)}, tokenManager)
> 	token, err := tokenManager.GenerateAccessToken(uuid.New(), "Alice")
> 	if err != nil {
> 		t.Fatal(err)
> 	}
> 	return server.httpServer.Handler, token
> }
>
> func TestFriends_RequireAuth(t *testing.T) {
> 	router, _ := newFriendsRouter(t, &stubFriends{})
> 	if w := doUsersRequest(router, http.MethodGet, "/api/v1/friends", "", ""); w.Code != http.StatusUnauthorized {
> 		t.Fatalf("status = %d, want 401", w.Code)
> 	}
> }
>
> func TestFriends_ListShape(t *testing.T) {
> 	bob := uuid.New()
> 	stub := &stubFriends{overview: model.FriendsOverview{
> 		Friends: []model.FriendLink{{UserID: bob, Username: "Bob", Presence: model.PresenceOnline}},
> 	}}
> 	router, token := newFriendsRouter(t, stub)
>
> 	w := doUsersRequest(router, http.MethodGet, "/api/v1/friends", token, "")
> 	if w.Code != http.StatusOK {
> 		t.Fatalf("status = %d", w.Code)
> 	}
> 	var resp FriendsResponse
> 	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
> 		t.Fatal(err)
> 	}
> 	if len(resp.Friends) != 1 || resp.Friends[0].ID != bob.String() || resp.Friends[0].Status != "online" {
> 		t.Fatalf("friends = %+v", resp.Friends)
> 	}
> 	if resp.Incoming == nil || resp.Outgoing == nil {
> 		t.Fatalf("empty lists must be [] not null: %s", w.Body.String())
> 	}
> }
>
> func TestFriends_RequestStatusCodes(t *testing.T) {
> 	cases := []struct {
> 		name  string
> 		state model.FriendState
> 		err   error
> 		want  int
> 	}{
> 		{"new request", model.FriendPending, nil, http.StatusCreated},
> 		{"counter request", model.FriendAccepted, nil, http.StatusOK},
> 		{"unknown user", "", model.ErrUserNotFound, http.StatusNotFound},
> 		{"self", "", model.ErrFriendSelf, http.StatusBadRequest},
> 		{"already friends", "", model.ErrAlreadyFriends, http.StatusConflict},
> 		{"already sent", "", model.ErrFriendRequestExists, http.StatusConflict},
> 	}
> 	for _, tc := range cases {
> 		t.Run(tc.name, func(t *testing.T) {
> 			stub := &stubFriends{state: tc.state, err: tc.err}
> 			router, token := newFriendsRouter(t, stub)
> 			w := doUsersRequest(router, http.MethodPost, "/api/v1/friends/requests", token, `{"username":"Bob"}`)
> 			if w.Code != tc.want {
> 				t.Fatalf("status = %d, want %d (%s)", w.Code, tc.want, w.Body.String())
> 			}
> 			if stub.lastName != "Bob" {
> 				t.Fatalf("service got username %q", stub.lastName)
> 			}
> 		})
> 	}
> }
>
> func TestFriends_AcceptAndRemoveUseIDFromPath(t *testing.T) {
> 	stub := &stubFriends{}
> 	router, token := newFriendsRouter(t, stub)
> 	bob := uuid.New()
>
> 	if w := doUsersRequest(router, http.MethodPost, "/api/v1/friends/requests/"+bob.String()+"/accept", token, ""); w.Code != http.StatusNoContent {
> 		t.Fatalf("accept status = %d", w.Code)
> 	}
> 	if stub.lastID != bob {
> 		t.Fatalf("accept got %s, want %s", stub.lastID, bob)
> 	}
> 	if w := doUsersRequest(router, http.MethodDelete, "/api/v1/friends/"+bob.String(), token, ""); w.Code != http.StatusNoContent {
> 		t.Fatalf("remove status = %d", w.Code)
> 	}
> 	if w := doUsersRequest(router, http.MethodDelete, "/api/v1/friends/not-a-uuid", token, ""); w.Code != http.StatusBadRequest {
> 		t.Fatalf("bad id status = %d, want 400", w.Code)
> 	}
>
> 	stub.err = model.ErrFriendRequestNotFound
> 	if w := doUsersRequest(router, http.MethodPost, "/api/v1/friends/requests/"+bob.String()+"/accept", token, ""); w.Code != http.StatusNotFound {
> 		t.Fatalf("missing request status = %d, want 404", w.Code)
> 	}
> }
> ```

### Готово, когда

- [ ] `go test -race ./internal/delivery/http/ -run Friends` зелёный
- [ ] `curl` по шагу 8 отвечает ожидаемыми кодами

---

## Шаг 6. Статусы и уведомления

### Зачем

Две вещи из лобби-сокета, которые пока заперты внутри: **кто где сейчас** и **как достучаться до конкретного игрока**.

### Теория

> [!note] Присутствие живёт в памяти, а не в базе
> «Сейчас в лобби» меняется каждую секунду и теряет смысл после перезапуска сервера: все соединения всё равно оборвутся. Писать это в Postgres — лишняя нагрузка ради данных, которые никто не будет читать через минуту. Источник правды — карта `clients` в горутине `Lobby` и комнаты в `Hub`.

> [!note] Два источника статуса
> - **За столом:** `Hub` знает комнаты, `room.Holds(id)` — сидит ли игрок. Это и ожидающий стол, и идущая партия.
> - **В лобби:** только горутина `Lobby` знает свою карту. Спрашиваем её тем же приёмом, что и `Online()`: канал-запрос с ответом, только теперь в запросе список id.
>
> Если игрок и за столом, и в лобби (две вкладки), показываем «за столом»: это важнее.

> [!note] Не держать блокировку дольше нужного
> `Hub.Presence` копирует список комнат под `h.mu.RLock`, отпускает блокировку и только потом опрашивает каждую комнату. Так хаб не стоит заблокированным, пока перебираются чужие блокировки комнат. Приём тот же, что «не ждать канал, держа мьютекс» из лобби-сокета.

> [!note] Сигнал, а не данные
> Можно было бы прислать Бобу саму заявку: `{"type":"friend_request","from":"Алиса"}`. Но тогда:
> - на каждое событие (заявка, принятие, удаление, отмена) нужен свой формат и свой обработчик на фронте;
> - если сообщение потерялось, данные на экране навсегда разошлись с базой.
>
> Вместо этого шлём одно `friends_changed` без данных, а клиент перезапрашивает `GET /friends`. Источник правды один — база. Потерянный сигнал исправится при следующем обновлении. Цена — лишний HTTP-запрос, для друзей это копейки.

> [!note] Адресная доставка
> У лобби уже есть `clients[playerID]` — все вкладки игрока. Уведомление — просьба в канал `notices`, а `Run` рассылает её по этим вкладкам через тот же `sendTo`, что отключает медленных клиентов. Если игрока нет в лобби, ничего не происходит: увидит изменения, когда зайдёт.

> [!note] Как обновятся статусы друзей
> Друг зашёл в лобби → у всех изменился счётчик онлайна → лобби разослало `lobby_state` → фронт видит новое `online` и перезапрашивает друзей. Отдельный механизм не нужен. Цена: при большом онлайне каждый вход вызовет волну `GET /friends`. Для проекта это нормально, для тысяч игроков понадобилась бы адресная рассылка только друзьям.

### Практика

1. `ws/message.go`: константа `TypeFriendsChanged = "friends_changed"`.
2. `ws/lobby.go`:
   - тип запроса `presenceQuery{ids, reply}` и канал `presence`;
   - тип `notice{playerID, data}` и канал `notices`;
   - две новые ветки в `Run`;
   - методы `OnlineOf(ids []string) map[string]bool` и `Notify(playerID string, data []byte)`, оба не виснут после `Close`.
3. `ws/hub.go`:
   - `Presence(ids []uuid.UUID) map[uuid.UUID]model.Presence`;
   - `FriendsChanged(ids ...uuid.UUID)`.
4. Тесты: `OnlineOf` видит только подключённых; `Notify` доходит до всех вкладок одного игрока и не доходит до других; `Hub.Presence` различает лобби, стол и «не в сети».

> [!warning] Ловушки
> - Отдать наружу саму карту `l.clients` вместо копии ответа. Тогда её начнут читать из чужой горутины, и `-race` это поймает.
> - Новый клиент получает `lobby_state`, а регистрация следующего снова рассылает `lobby_state` всем. В тесте `Notify` очисти буферы **всех** клиентов после регистрации, иначе первым прочитаешь не то сообщение.
> - `NewMessage(TypeFriendsChanged, nil)` даёт `{"type":"friends_changed"}`: поле `payload` с `omitempty` пропадает. Так и задумано.

### Проверь себя

> [!question]- Почему `Presence` спрашивает лобби одним запросом со списком, а не по одному id?
> Каждый запрос — поход в горутину `Run` через канал. У игрока 30 друзей — 30 походов вместо одного. Пакетный запрос дешевле и даёт согласованный снимок на один момент.

> [!question]- Что если `Notify` вызвали, когда лобби уже остановлено?
> `select` со `stop`: отправка не случится, функция сразу вернётся. HTTP-обработчик не повиснет.

> [!question]- Почему статус «за столом» берётся у `Hub`, а не у лобби?
> Со страницы стола игрок не подключён к лобби-сокету: `LobbyView` размонтирован, и его сокет закрыт. Про столы знает только хаб.

> [!example]- Решение: ws/lobby.go (что добавить)
> Поля и типы:
> ```go
> 	online     chan chan int
> 	presence   chan presenceQuery
> 	notices    chan notice
> }
>
> type presenceQuery struct {
> 	ids   []string
> 	reply chan map[string]bool
> }
>
> type notice struct {
> 	playerID string
> 	data     []byte
> }
> ```
> В `NewLobby`:
> ```go
> 		presence:   make(chan presenceQuery),
> 		notices:    make(chan notice),
> ```
> Новые ветки в `select` внутри `Run`:
> ```go
> 		case q := <-l.presence:
> 			in := make(map[string]bool, len(q.ids))
> 			for _, id := range q.ids {
> 				if _, ok := l.clients[id]; ok {
> 					in[id] = true
> 				}
> 			}
> 			q.reply <- in
> 		case n := <-l.notices:
> 			for c := range l.clients[n.playerID] {
> 				l.sendTo(c, n.data)
> 			}
> ```
> Методы:
> ```go
> func (l *Lobby) OnlineOf(ids []string) map[string]bool {
> 	q := presenceQuery{ids: ids, reply: make(chan map[string]bool, 1)}
> 	select {
> 	case l.presence <- q:
> 	case <-l.stop:
> 		return map[string]bool{}
> 	}
> 	select {
> 	case in := <-q.reply:
> 		return in
> 	case <-l.stop:
> 		return map[string]bool{}
> 	}
> }
>
> func (l *Lobby) Notify(playerID string, data []byte) {
> 	select {
> 	case l.notices <- notice{playerID: playerID, data: data}:
> 	case <-l.stop:
> 	}
> }
> ```

> [!example]- Решение: ws/hub.go и ws/message.go
> В `message.go`, в блок серверных типов:
> ```go
> 	TypeFriendsChanged     = "friends_changed"
> ```
> В `hub.go` (в импорты добавить `"bito/internal/model"`):
> ```go
> func (h *Hub) Presence(ids []uuid.UUID) map[uuid.UUID]model.Presence {
> 	keys := make([]string, len(ids))
> 	for i, id := range ids {
> 		keys[i] = id.String()
> 	}
> 	inLobby := h.lobby.OnlineOf(keys)
>
> 	h.mu.RLock()
> 	rooms := make([]*Room, 0, len(h.rooms))
> 	for _, room := range h.rooms {
> 		rooms = append(rooms, room)
> 	}
> 	h.mu.RUnlock()
>
> 	out := make(map[uuid.UUID]model.Presence, len(ids))
> 	for i, id := range ids {
> 		out[id] = model.PresenceOffline
> 		if inLobby[keys[i]] {
> 			out[id] = model.PresenceOnline
> 		}
> 		for _, room := range rooms {
> 			if room.Holds(keys[i]) {
> 				out[id] = model.PresencePlaying
> 				break
> 			}
> 		}
> 	}
> 	return out
> }
>
> func (h *Hub) FriendsChanged(ids ...uuid.UUID) {
> 	data, err := NewMessage(TypeFriendsChanged, nil)
> 	if err != nil {
> 		log.Printf("[Hub] friends_changed: %v", err)
> 		return
> 	}
> 	for _, id := range ids {
> 		h.lobby.Notify(id.String(), data)
> 	}
> }
> ```

> [!example]- Решение: тесты лобби и хаба
> В `lobby_test.go` (в импорты добавить `bito/internal/game`, `bito/internal/model`, `github.com/google/uuid`):
> ```go
> func TestLobby_OnlineOfReportsOnlyConnectedPlayers(t *testing.T) {
> 	l, _ := startLobby(t)
> 	l.Register(lobbyClient("alice", l))
>
> 	in := l.OnlineOf([]string{"alice", "bob"})
> 	if !in["alice"] || in["bob"] {
> 		t.Fatalf("online of = %v, want only alice", in)
> 	}
> }
>
> func TestLobby_NotifyReachesEveryTabOfOnePlayer(t *testing.T) {
> 	l, _ := startLobby(t)
> 	tab1, tab2 := lobbyClient("alice", l), lobbyClient("alice", l)
> 	bob := lobbyClient("bob", l)
> 	for _, c := range []*Client{tab1, tab2, bob} {
> 		l.Register(c)
> 	}
> 	l.Online()
> 	for _, c := range []*Client{tab1, tab2, bob} {
> 		for len(c.send) > 0 {
> 			<-c.send
> 		}
> 	}
>
> 	l.Notify("alice", []byte(`{"type":"friends_changed"}`))
> 	l.Online()
>
> 	for _, c := range []*Client{tab1, tab2} {
> 		select {
> 		case data := <-c.send:
> 			if string(data) != `{"type":"friends_changed"}` {
> 				t.Fatalf("got %s", data)
> 			}
> 		default:
> 			t.Fatal("a tab of alice got no notice")
> 		}
> 	}
> 	if len(bob.send) != 0 {
> 		t.Fatal("bob got alice's notice")
> 	}
> }
>
> func TestHub_PresenceLobbyTableOffline(t *testing.T) {
> 	h := NewHub(nil, nil, nil)
> 	inLobby, atTable, away := uuid.New(), uuid.New(), uuid.New()
>
> 	h.lobby.Register(lobbyClient(inLobby.String(), h.lobby))
> 	room := h.CreateRoom(RoomSettings{Mode: game.ModePodkidnoy, Deck: game.Deck36, MaxPlayers: 2, CreatedBy: "x"})
> 	defer room.Close()
> 	room.RegisterClient(createTestClient(atTable.String(), "T", room))
> 	deadline := time.Now().Add(time.Second)
> 	for !room.Holds(atTable.String()) && time.Now().Before(deadline) {
> 		time.Sleep(5 * time.Millisecond)
> 	}
>
> 	got := h.Presence([]uuid.UUID{inLobby, atTable, away})
> 	want := map[uuid.UUID]model.Presence{inLobby: model.PresenceOnline, atTable: model.PresencePlaying, away: model.PresenceOffline}
> 	for id, p := range want {
> 		if got[id] != p {
> 			t.Errorf("presence of %s = %q, want %q", id, got[id], p)
> 		}
> 	}
> }
> ```

### Готово, когда

- [ ] `go test -race ./internal/delivery/ws/ -count=20` зелёный
- [ ] `Hub` реализует оба интерфейса сервиса, и `app.go` собирается

---

## Шаг 7. Фронт

### Зачем

Убрать моки и подключить настоящие данные: список, заявки, форму добавления, отправленные заявки, удаление. Плюс обновление по сигналу из сокета.

### Теория

> [!note] Один источник правды
> Данные о друзьях живут в одном месте — composable `useFriends`. Компоненты только показывают и сообщают о действиях. После каждого действия список перезапрашивается с сервера, а не правится вручную в трёх массивах. Так экран никогда не разойдётся с базой.

> [!note] Пессимистичный и оптимистичный интерфейс
> **Оптимистичный:** сразу убрать заявку с экрана, а запрос отправить в фоне; если он упал — вернуть как было. Быстро на вид, но сложно откатывать.
> **Пессимистичный:** дождаться ответа и перезагрузить список. На локальном сервере разницы не видно. Берём его: проще и честнее.

> [!note] Склейка запросов
> `refresh()` могут вызвать одновременно: пришёл `friends_changed`, изменился онлайн, игрок нажал «Принять». Три одинаковых запроса подряд бессмысленны, а ответы могут прийти не по порядку. Приём: если запрос уже идёт, запоминаем «нужно ещё раз» (`again = true`) и возвращаем тот же промис. Когда текущий закончится, сделаем ровно один повтор.

> [!note] Функция-проп вместо события, когда нужен результат
> `emit` в Vue ничего не возвращает: родитель не может сказать компоненту «получилось» или «ошибка, вот текст». Для формы добавления передаём **функцию** пропом: `add: (username) => Promise<string>`. Компонент ждёт её, показывает текст результата или текст ошибки прямо под полем.

> [!note] `defineModel`
> Открыта ли форма добавления, знают двое: `FriendsList` (кнопка «Добавить друга») и `LobbyView` (плюсик на рейке открывает панель сразу с формой). `defineModel('adding')` даёт двустороннюю привязку `v-model:adding` без ручной пары «проп + событие».

### Практика

1. `types/social.ts`: у `Friend` поле `since?`, у `FriendRequest` поле `at?`, тип `FriendsOverview`.
2. `api/friends.ts`: четыре функции по таблице API. `encodeURIComponent` для id в пути.
3. `composables/useFriends.ts`: `friends`, `incoming`, `outgoing`, `loaded`, `error`, `refresh` со склейкой, `add`, `accept`, `remove`.
4. `useLobbySocket.ts`: счётчик `friendsVersion`, который растёт на каждое `friends_changed`.
5. `FriendsList.vue`: форма добавления (поле, «Отправить заявку», «Отмена», текст результата), раздел «Отправленные» с отменой, кнопка «Удалить из друзей» у друга.
6. `LobbyView.vue`: убрать моки, подключить `useFriends`, перезапрашивать при изменении `online` и `friendsVersion`, при открытии и в запасном опросе. Значок на шапке — число входящих.
7. `mocks/lobby.ts`: удалить `MOCK_FRIENDS` и `MOCK_REQUESTS`.

> [!warning] Ловушки
> - `watch([online, friendsVersion], ...)` сработает и при первом подключении сокета (`null` → число). Это полезно, не бойся.
> - Удаление друга — действие с последствиями. Нужно подтверждение, хотя бы `window.confirm`.
> - Форма остаётся открытой после закрытия панели. Сбрасывай `adding`, когда панель друзей закрывается.
> - Кнопка «Позвать за стол» пока остаётся заглушкой: приглашения — это бонус, шаг 9.

### Проверь себя

> [!question]- Зачем `friendsVersion` — число, а не булев флаг?
> `watch` реагирует на **изменение** значения. Флаг `true` → `true` изменением не считается, и второй сигнал потеряется. Счётчик меняется каждый раз.

> [!question]- Что будет, если сокет отвалился?
> Сигналы перестанут приходить, но `refreshAll` в запасном режиме (`status !== 'open'`) перезапрашивает и столы, и друзей по таймеру. Обновления станут медленнее, но не пропадут.

> [!question]- Почему после `add` не нужно руками добавлять заявку в `outgoing`?
> `add` дожидается ответа и вызывает `refresh()`: сервер вернёт уже правильные списки, в том числе случай «встречная заявка, сразу друзья».

> [!example]- Решение: api/friends.ts
> ```ts
> import { authedFetch } from './http'
> import type { FriendsOverview } from '../types/social'
>
> const BASE = '/api/v1/friends'
>
> export function getFriends(): Promise<FriendsOverview> {
>   return authedFetch<FriendsOverview>(BASE)
> }
>
> export function addFriend(username: string): Promise<{ status: 'pending' | 'accepted' }> {
>   return authedFetch(`${BASE}/requests`, {
>     method: 'POST',
>     body: JSON.stringify({ username }),
>   })
> }
>
> export async function acceptFriend(id: string): Promise<void> {
>   await authedFetch<null>(`${BASE}/requests/${encodeURIComponent(id)}/accept`, { method: 'POST' })
> }
>
> export async function removeFriend(id: string): Promise<void> {
>   await authedFetch<null>(`${BASE}/${encodeURIComponent(id)}`, { method: 'DELETE' })
> }
> ```

> [!example]- Решение: composables/useFriends.ts
> ```ts
> import { ref } from 'vue'
> import { acceptFriend, addFriend, getFriends, removeFriend } from '../api/friends'
> import type { Friend, FriendRequest } from '../types/social'
>
> export function useFriends() {
>   const friends = ref<Friend[]>([])
>   const incoming = ref<FriendRequest[]>([])
>   const outgoing = ref<FriendRequest[]>([])
>   const loaded = ref(false)
>   const error = ref('')
>
>   let running: Promise<void> | null = null
>   let again = false
>
>   async function load() {
>     const data = await getFriends()
>     friends.value = data.friends
>     incoming.value = data.incoming
>     outgoing.value = data.outgoing
>     loaded.value = true
>   }
>
>   function refresh(): Promise<void> {
>     if (running) {
>       again = true
>       return running
>     }
>     running = (async () => {
>       try {
>         do {
>           again = false
>           await load()
>         } while (again)
>         error.value = ''
>       } catch (err) {
>         error.value = err instanceof Error ? err.message : 'Не удалось загрузить друзей'
>       } finally {
>         running = null
>       }
>     })()
>     return running
>   }
>
>   async function add(username: string) {
>     const { status } = await addFriend(username.trim())
>     await refresh()
>     return status
>   }
>
>   async function accept(id: string) {
>     await acceptFriend(id)
>     await refresh()
>   }
>
>   async function remove(id: string) {
>     await removeFriend(id)
>     await refresh()
>   }
>
>   return { friends, incoming, outgoing, loaded, error, refresh, add, accept, remove }
> }
> ```

> [!example]- Решение: types/social.ts и useLobbySocket.ts
> В `types/social.ts`:
> ```ts
> export interface Friend {
>   id: string
>   username: string
>   status: FriendStatus
>   since?: string
>   lastSeen?: string
> }
>
> export interface FriendRequest {
>   id: string
>   username: string
>   at?: string
> }
>
> export interface FriendsOverview {
>   friends: Friend[]
>   incoming: FriendRequest[]
>   outgoing: FriendRequest[]
> }
> ```
> В `useLobbySocket.ts`: `const friendsVersion = ref(0)`, в начало `onmessage` после разбора JSON:
> ```ts
>       if (msg.type === 'friends_changed') {
>         friendsVersion.value++
>         return
>       }
> ```
> и `friendsVersion` в возвращаемом объекте.

> [!example]- Решение: FriendsList.vue
> Скрипт:
> ```ts
> import { computed, nextTick, ref, watch } from 'vue'
> import { PhUserPlus, PhUserMinus, PhPaperPlaneTilt, PhCheck, PhX, PhMagnifyingGlass } from '@phosphor-icons/vue'
> import type { Friend, FriendRequest, FriendStatus } from '../../types/social'
> import { avatarHue, avatarInitial } from '../../composables/avatar'
>
> const props = defineProps<{
>   friends: Friend[]
>   requests: FriendRequest[]
>   outgoing: FriendRequest[]
>   add: (username: string) => Promise<string>
> }>()
> defineEmits<{
>   (e: 'invite', friend: Friend): void
>   (e: 'accept', req: FriendRequest): void
>   (e: 'decline', req: FriendRequest): void
>   (e: 'cancel', req: FriendRequest): void
>   (e: 'remove', friend: Friend): void
> }>()
>
> const adding = defineModel<boolean>('adding', { default: false })
> const newName = ref('')
> const addBusy = ref(false)
> const addResult = ref<{ ok: boolean; text: string } | null>(null)
> const addInput = ref<HTMLInputElement | null>(null)
>
> watch(
>   adding,
>   async (open) => {
>     addResult.value = null
>     if (!open) return
>     await nextTick()
>     addInput.value?.focus()
>   },
>   { immediate: true },
> )
>
> async function submitAdd() {
>   const name = newName.value.trim()
>   if (!name || addBusy.value) return
>   addBusy.value = true
>   try {
>     addResult.value = { ok: true, text: await props.add(name) }
>     newName.value = ''
>   } catch (err) {
>     addResult.value = { ok: false, text: err instanceof Error ? err.message : 'Не получилось' }
>   } finally {
>     addBusy.value = false
>   }
> }
> ```
> (остальной скрипт — `ORDER`, `query`, `filter`, `shown`, `statusText` — без изменений)
>
> В шаблоне у строки друга, после кнопки «Позвать за стол»:
> ```html
>           <button
>             type="button"
>             class="mini invite"
>             :aria-label="`Удалить ${f.username} из друзей`"
>             title="Удалить из друзей"
>             @click="$emit('remove', f)"
>           >
>             <PhUserMinus :size="15" />
>           </button>
> ```
> После списка друзей, вместо старой кнопки «Добавить друга»:
> ```html
>     <section v-if="outgoing.length" class="group" aria-labelledby="outgoing-title">
>       <h3 id="outgoing-title">Отправленные <span>{{ outgoing.length }}</span></h3>
>       <ul class="rows">
>         <li v-for="r in outgoing" :key="r.id" class="row friend offline">
>           <span class="avatar" :style="{ '--h': avatarHue(r.username) }">{{ avatarInitial(r.username) }}</span>
>           <span class="who">
>             <b>{{ r.username }}</b>
>             <small>ждёт ответа</small>
>           </span>
>           <button type="button" class="mini" :aria-label="`Отменить заявку для ${r.username}`" title="Отменить" @click="$emit('cancel', r)">
>             <PhX :size="15" weight="bold" />
>           </button>
>         </li>
>       </ul>
>     </section>
>
>     <form v-if="adding" class="add-form" @submit.prevent="submitAdd" @keydown.esc="adding = false">
>       <label class="search">
>         <PhUserPlus :size="17" aria-hidden="true" />
>         <input
>           ref="addInput"
>           v-model="newName"
>           type="text"
>           maxlength="64"
>           autocomplete="off"
>           placeholder="Имя игрока"
>           aria-label="Имя игрока"
>         />
>       </label>
>       <div class="add-actions">
>         <button type="button" class="ghost" @click="adding = false">Отмена</button>
>         <button type="submit" class="send" :disabled="!newName.trim() || addBusy">
>           {{ addBusy ? 'Отправляем…' : 'Отправить заявку' }}
>         </button>
>       </div>
>       <p v-if="addResult" class="add-result" :class="{ bad: !addResult.ok }" role="status">{{ addResult.text }}</p>
>     </form>
>     <button v-else type="button" class="add" @click="adding = true">
>       <PhUserPlus :size="18" aria-hidden="true" />
>       Добавить друга
>     </button>
> ```
> Стили в конец `<style scoped>`:
> ```css
> .add-form {
>   display: flex;
>   flex-direction: column;
>   gap: 10px;
> }
>
> .add-actions {
>   display: flex;
>   gap: 8px;
> }
>
> .add-actions button {
>   flex: 1;
>   height: 40px;
>   border: 0;
>   border-radius: 999px;
>   font: inherit;
>   font-size: 14px;
>   cursor: pointer;
>   transition: opacity 0.2s, background-color 0.2s;
> }
>
> .ghost {
>   background: var(--color-paper-sunken);
>   color: var(--color-ink-muted);
> }
>
> .send {
>   background: var(--color-gold);
>   color: var(--color-ink);
> }
>
> .send:disabled {
>   opacity: 0.5;
>   cursor: default;
> }
>
> .add-result {
>   margin: 0;
>   padding: 0 8px;
>   color: var(--color-green);
>   font-size: 13px;
> }
>
> .add-result.bad {
>   color: var(--color-red);
> }
> ```

> [!example]- Решение: LobbyView.vue
> Импорты: убрать `MOCK_FRIENDS, MOCK_REQUESTS`, добавить:
> ```ts
> import { useFriends } from '../../composables/useFriends'
> ```
> Вместо двух `ref` с моками:
> ```ts
> const { status: lobbyStatus, tables: liveTables, online, friendsVersion } = useLobbySocket()
>
> const { friends, incoming, outgoing, refresh: refreshFriends, add: addFriendByName, accept: acceptFriend, remove: removeFriend } = useFriends()
> const addingFriend = ref(false)
>
> watch([online, friendsVersion], () => refreshFriends())
> watch(
>   () => panels.panel.value,
>   (panel) => {
>     if (panel !== 'friends') addingFriend.value = false
>   },
> )
> ```
> Вместо старых `acceptRequest` и `declineRequest`:
> ```ts
> function friendError(err: unknown) {
>   sound.playError()
>   toast.show(err instanceof ApiError ? err.message : 'Не получилось, попробуйте ещё раз', 'error')
> }
>
> async function addFriend(username: string) {
>   const status = await addFriendByName(username)
>   sound.playSuccess()
>   return status === 'accepted' ? `${username} теперь в друзьях` : `Заявка отправлена: ${username}`
> }
>
> async function acceptRequest(req: FriendRequest) {
>   try {
>     await acceptFriend(req.id)
>     sound.playSuccess()
>     toast.show(`${req.username} теперь в друзьях`)
>   } catch (err) {
>     friendError(err)
>   }
> }
>
> async function dropLink(id: string, done: string) {
>   try {
>     await removeFriend(id)
>     sound.playClick()
>     toast.show(done)
>   } catch (err) {
>     friendError(err)
>   }
> }
>
> function declineRequest(req: FriendRequest) {
>   dropLink(req.id, `Заявка от ${req.username} отклонена`)
> }
>
> function cancelRequest(req: FriendRequest) {
>   dropLink(req.id, `Заявка для ${req.username} отменена`)
> }
>
> function removeFromFriends(friend: Friend) {
>   if (!window.confirm(`Удалить ${friend.username} из друзей?`)) return
>   dropLink(friend.id, `${friend.username} больше не в друзьях`)
> }
>
> function openAddFriend() {
>   addingFriend.value = true
>   panels.open('friends')
> }
> ```
> Запасной опрос и первая загрузка:
> ```ts
> function refreshAll() {
>   if (lobbyStatus.value !== 'open') {
>     refreshTables()
>     refreshFriends()
>   }
>   refreshActiveTable()
> }
> ```
> и в `onMounted` после `refreshAll()` добавить `refreshFriends()`.
>
> В шаблоне:
> - `TopBar`: `:friends-badge="incoming.length"`;
> - `FriendsRail`: `:requests="incoming"` и `@add="openAddFriend"`;
> - `FriendsList`:
> ```html
>         <FriendsList
>           class="friends-sheet"
>           v-model:adding="addingFriend"
>           :friends="friends"
>           :requests="incoming"
>           :outgoing="outgoing"
>           :add="addFriend"
>           @invite="invite"
>           @accept="acceptRequest"
>           @decline="declineRequest"
>           @cancel="cancelRequest"
>           @remove="removeFromFriends"
>         />
> ```

### Готово, когда

- [ ] `npm run build` в `web/` проходит
- [ ] в коде лобби нет `MOCK_FRIENDS` и `MOCK_REQUESTS`
- [ ] весь сценарий из шага 8 проходит в браузере

---

## Шаг 8. Проверка руками

### Сценарий в двух браузерах

Обычное окно — Алиса, инкогнито — Боб. Оба в лобби.

1. Алиса: «Добавить друга» → «Боб» → «Заявка отправлена». У неё раздел «Отправленные».
2. Боб **без перезагрузки** видит значок на иконке друзей и заявку в панели.
3. Боб принимает → у обоих друг в списке со статусом «в лобби».
4. Боб садится за стол → у Алисы через пару секунд «за столом».
5. Боб закрывает вкладку → у Алисы «не в сети».
6. Ошибки: добавить себя, несуществующее имя, Боба повторно — под полем понятный текст.
7. Встречная заявка: оба удаляют друг друга, Алиса шлёт заявку, Боб шлёт заявку Алисе → сразу «теперь в друзьях».

### Заглянуть в базу

```bash
docker exec -it bito_postgres psql -U postgres -d bito_db
```

```sql
SELECT f.status, a.name AS low, b.name AS high, r.name AS requested_by, f.created_at, f.accepted_at
FROM friendships f
JOIN users a ON a.id = f.user_low
JOIN users b ON b.id = f.user_high
JOIN users r ON r.id = f.requested_by;
```

### API через curl

```bash
curl -s -c alice.jar -X POST localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d "{\"email\":\"<почта>\",\"password\":\"<пароль>\"}"
```

```bash
curl -s -b alice.jar -X POST localhost:8080/api/v1/friends/requests -H "Content-Type: application/json" -d "{\"username\":\"Боб\"}" -w "\n%{http_code}\n"
```

```bash
curl -s -b alice.jar localhost:8080/api/v1/friends
```

### Сообщение в сокете

DevTools → Network → WS → `lobby` → Messages: у Боба после заявки Алисы появляется `{"type":"friends_changed"}`.

---

## Шаг 9. Бонусы

Делай, если основное готово и есть время. Решения здесь — набросок, а не готовый код.

### 9.1. «Был в сети N минут назад» ★★☆

Экран уже умеет показывать `lastSeen` (`statusText` в `FriendsList`). Не хватает данных.

> [!example]- Набросок
> - Миграция: `ALTER TABLE users ADD COLUMN last_seen_at TIMESTAMPTZ`.
> - Когда у игрока в лобби закрывается **последняя** вкладка (`remove` вернул `wentOffline = true`), записать время. Но горутине `Lobby` нельзя ходить в базу. Варианты: отдельная горутина-писатель с буферизованным каналом или карта `lastSeen` в памяти лобби, которую раз в минуту сбрасывают в базу.
> - `List` возвращает `u.last_seen_at`, сервис кладёт его только офлайн-друзьям.
> - Подумай: что будет с «был в сети» после перезапуска сервера в каждом из вариантов?

### 9.2. Лимиты ★☆☆

> [!example]- Набросок
> - Не больше 100 друзей и 20 исходящих заявок. Проверка в сервисе перед `Request`: посчитать в репозитории `COUNT(*)`. Ошибка `ErrFriendLimit` → `409`.
> - Подумай, почему проверка «посчитать, потом вставить» снова допускает гонку и почему здесь это не страшно (в худшем случае 101 друг).
> - Частота заявок: лимитер в Redis уже есть (`redisRepo.NewRateLimiter`), его можно повесить и на `POST /friends/requests`.

### 9.3. Приглашение за стол ★★★

Последний этап из [[features]]. Кнопка «Позвать за стол» уже нарисована.

> [!example]- Набросок
> - `POST /api/v1/friends/{id}/invite` с `{ "tableId": "..." }`. Проверки: вы друзья; стол существует и не начался; у друга хватает на ставку; друг не сидит за другим столом.
> - Забронировать место: `room.Reserve(friendID)` уже есть, бронь живёт 15 секунд. Для приглашения нужна бронь подольше или отдельное поле.
> - Доставка: `lobby.Notify(friendID, {"type":"table_invite","payload":{"from":"Алиса","tableId":"...","stake":100}})`. Здесь как раз **данные, а не сигнал**: приглашение — разовое событие, его не перезапросить из базы.
> - Фронт: тост с кнопками «Сесть» и «Нет», таймер. «Сесть» → `router.push` на стол.
> - Вопросы для размышления: что, если друг за 30 секунд не ответил? если стол заполнился? если пригласили двое сразу?

---

## Вопросы комиссии

> [!question]- Как устроена таблица друзей и почему так?
> Одна строка на пару игроков, id упорядочены: `user_low < user_high`. Первичный ключ на паре гарантирует, что дубль невозможен, а `CHECK` запрещает дружбу с собой и неизвестные статусы. Кто отправил заявку, хранится в `requested_by`.

> [!question]- Что будет, если двое одновременно отправят заявки друг другу?
> Заявка записывается одним SQL-запросом `INSERT ... ON CONFLICT DO UPDATE ... WHERE`. Второй запрос ждёт блокировку строки первого, видит встречную заявку и делает её дружбой. Итог — одна строка `accepted`, это проверено тестом с двумя горутинами.

> [!question]- Откуда сервер знает, что друг в сети?
> Из памяти, а не из базы: лобби-сокет знает, кто в лобби, а хаб — кто за столом. Это данные, которые живут секунды и бессмысленны после перезапуска, поэтому в базу их не пишем.

> [!question]- Как друг узнаёт о заявке без перезагрузки?
> Сервис после записи в базу просит хаб уведомить второго игрока. Лобби отправляет во все его вкладки сигнал `friends_changed`, и фронт перезапрашивает список. В сигнале нет данных: источник правды один — база.

> [!question]- Почему сервис не зависит от WebSocket?
> Он объявляет интерфейсы «источник статусов» и «уведомитель», а хаб их реализует. Сервис тестируется на фейках, а способ доставки можно заменить, не трогая логику.

> [!question]- Что с приватностью?
> Онлайн-статус видят только друзья, не заявки и не посторонние. Почта нигде не раскрывается, поиск идёт по публичному имени.

## Шпаргалка

| Что | Где |
|---|---|
| Таблица | `migrations/000003_friends.sql` |
| Модель и ошибки | `internal/model/friend.go` |
| SQL | `internal/repository/postgres/friends.go` |
| Правила | `internal/service/friends.go` |
| API | `internal/delivery/http/friend_handler.go`, маршруты в `server.go` |
| Присутствие и уведомления | `ws/lobby.go` (`OnlineOf`, `Notify`), `ws/hub.go` (`Presence`, `FriendsChanged`) |
| Фронт | `api/friends.ts`, `composables/useFriends.ts`, `FriendsList.vue`, `LobbyView.vue` |

```bash
go test -race ./internal/...
```

```bash
go test -tags integration ./internal/repository/postgres/ -run Friends -v
```

```bash
npm --prefix web run build
```

## Словарь

| Термин | Коротко |
|---|---|
| Каноническая пара | договорённость хранить пару в одном порядке, чтобы (A, B) и (B, A) были одной строкой |
| Правило левого префикса | составной индекс помогает искать по первой колонке, но не по второй отдельно |
| Upsert | вставить или обновить одной командой (`ON CONFLICT DO UPDATE`) |
| `EXCLUDED` | строка, которую пытались вставить, внутри `ON CONFLICT` |
| TOCTOU | гонка «проверил, а пока записывал, всё изменилось» |
| Compare-and-set | обновить, только если состояние всё ещё то, что ожидали; проверка через `RowsAffected` |
| Идемпотентность | повтор действия даёт тот же результат, что и первый раз |
| Инверсия зависимостей | бизнес-логика объявляет интерфейсы, а детали (база, сокеты) их реализуют |
| Присутствие (presence) | кто сейчас в сети; живёт в памяти |
| Сигнал, а не данные | уведомление «что-то изменилось» без самих изменений; клиент перезапрашивает |
| Склейка запросов | несколько одновременных `refresh()` превращаются в один запрос и максимум один повтор |
| Пессимистичный интерфейс | экран меняется после ответа сервера, а не до |

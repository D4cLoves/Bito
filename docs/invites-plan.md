---
title: Bito — приглашения за стол (учебный план)
tags: [bito, plan, invites, go, websocket, learning]
created: 2026-10-20
---

# Приглашения за стол: учебный план

> [!abstract] Что строим
> Алиса сидит за своим столом и зовёт друга Боба. Сервер **держит Бобу место** 45 секунд и присылает ему в лобби приглашение: кто зовёт, ставка, режим, колода. Боб либо садится, либо отказывается, и тогда Алиса за столом видит «Боб отказался». Посторонний это место занять не может, а сам Боб, пока думает, может спокойно играть в другом месте.
> Связанные заметки: [[friends-plan]] (друзья и статус «в лобби»), [[lobby-socket-plan]] (адресная доставка через лобби), [[features]] (пункт 7, этап 3).

> [!info] Как работать с этой заметкой
> Как и в прошлых планах, каждый шаг устроен так: **зачем → теория → практика → ловушки → проверь себя → решение → готово, когда**. Ответы и решения свёрнуты, сначала подумай и напиши своё.
> Здесь только **бэкенд**. Фронт (карточка приглашения в лобби, режим «позвать» в диалоге стола, строчка «отказался» в чате стола) делается отдельно; что ему понадобится от сервера, описано в шаге 8.

> [!success] Статус: готово (20.10.2026)
> Шаги 1–3 и 5 написаны тобой, причём `freeSeatsFor` у тебя лучше эталона: игрок с бронью и приглашением одновременно считается один раз, а подключённых ты смотришь по `clients`. Тесты (шаги 4–5), HTTP и стол на двоих (шаг 6) доделаны по плану. Фронт из шага 8 сделан: «Позвать за стол» у друга открывает диалог стола в режиме приглашения, у приглашённого в лобби появляется карточка с отсчётом и кнопками «Сесть» и «Нет», за столом в чате видно отказ.
> Проверено в браузере: Алиса позвала Боба (Боб в лобби через тестовый клиент), Боб получил `table_invite`, отказ пришёл Алисе в чат стола; в обратную сторону карточка появилась у Алисы, «Сесть» посадила её за стол, «Нет» отправила отказ.

> [!success] Что проверено
> Все решения взяты из эталона, который собирался и проходил тесты: `go vet`, `go test -race ./internal/...`, тесты комнаты и хаба 10 прогонов подряд. Код вставлен в заметку скриптом, а не переписан руками.

---

## 0. Карта

### Сценарий

```
 Алиса (за своим столом)              Go-сервер                              Боб (в лобби)
┌───────────────────────┐ POST /tables/{id}/invites ┌──────────────────┐
│ «Позвать Боба»        │ ─────────────────────────► │ InviteHandler    │
└───────────────────────┘   {"userId": "<Боб>"}      │  InviteService   │
                                                      │   друзья? в лобби?│
                                                      │   хватит фишек?  │
                                                      │  Hub.Invite      │──┐ держит место
                                                      │   (под h.mu)     │  │ 45 секунд
                                                      │  Hub.SendInvite ─┼──┼─► table_invite ─► карточка
                                                      └──────────────────┘  │   (/ws/lobby)    «Сесть / Нет»
                                                                            │
   «Сесть» ── обычный вход /ws?room_id=… ──► место из приглашения ◄─────────┘
   «Нет»   ── POST /tables/{id}/invites/decline ──► место освобождается
                                                 └─► invite_declined ─► Алисе за столом
```

### Почему приглашение «к столу», а не «к другу»

Можно было бы сделать `POST /friends/{id}/invite`, который сам создаёт стол. Но тогда одно действие делает две вещи, и непонятно, как позвать второго друга за тот же стол. Мы разделяем:

1. стол создаётся как обычно (`POST /api/v1/tables`);
2. приглашение — отдельное действие **над столом**: `POST /api/v1/tables/{id}/invites`.

В лобби кнопка «Позвать за стол» на фронте просто делает оба запроса подряд. А позже из экрана стола можно будет звать ещё друзей тем же эндпоинтом.

### API

| Метод | Путь | Тело | Ответ |
|---|---|---|---|
| `POST` | `/api/v1/tables` | как раньше, но теперь можно `maxPlayers: 2` | `201` стол |
| `POST` | `/api/v1/tables/{id}/invites` | `{"userId":"<id друга>"}` | `201 {"tableId","expiresAt"}` |
| `POST` | `/api/v1/tables/{id}/invites/decline` | — | `204` |

### Сообщения сокетов

| Кому | Канал | Сообщение |
|---|---|---|
| Приглашённому | `/ws/lobby` | `{"type":"table_invite","payload":{"tableId","fromId","fromName","mode","deck","maxPlayers","stake","expiresAt"}}` |
| Всем за столом | `/ws?room_id=` | `{"type":"invite_declined","payload":{"username"}}` |

### Шаги

| # | Шаг | Главный приём | Сложность | Файлы |
|---|---|---|---|---|
| 1 | Модель и ошибки | словарь ошибок предметной области | ★☆☆ | `model/invite.go` |
| 2 | Комната: место под приглашение | два вида брони, проверка места в горутине-владельце | ★★★ | `ws/room.go` |
| 3 | Хаб: пригласить, отказаться, доставить | проверка и запись под одной блокировкой | ★★☆ | `ws/hub.go`, `ws/message.go` |
| 4 | Тесты комнаты и хаба | табличные тесты, ожидание без `Sleep`-гаданий | ★★☆ | `ws/invite_test.go` |
| 5 | Сервис | порядок проверок, только друзья и только в лобби | ★★☆ | `service/invite.go` |
| 6 | HTTP и стол на двоих | 403 / 404 / 409, осознанная смена старого правила | ★★☆ | `delivery/http/invite_handler.go`, `table_handler.go`, `server.go`, `app.go` |
| 7 | Проверка руками | браузер + curl + DevTools | ★☆☆ | — |
| 8 | Что нужно фронту | контракт | — | — |
| 9 | Бонусы | отмена приглашения, «Боб принял», звать со стола | ★★★ | — |

Ветка: продолжай в `feature/friends` или создай новую от неё:

```bash
git switch -c feature/invites
```

---

## Шаг 1. Модель и ошибки

### Зачем

Приглашение может не состояться по десятку причин, и каждую фронт должен показать понятным текстом. Значит, у каждой причины должна быть своя ошибка-метка, которую верхние слои отличают через `errors.Is`.

### Теория

> [!note] Ошибки предметной области — это словарь
> Сначала выпиши все «нельзя», потом пиши код. Для приглашений:
>
> | Ошибка | Когда |
> |---|---|
> | `ErrTableNotFound` | стол уже закрылся |
> | `ErrNotAtTable` | зовёшь не за свой стол |
> | `ErrTableStarted` | партия уже идёт |
> | `ErrTableFull` | свободных мест нет (с учётом чужих броней) |
> | `ErrAlreadyInvited` | этого друга уже позвали или он уже сидит |
> | `ErrFriendBusy` | друг сам сидит или держит другой стол |
> | `ErrNotFriends` | это не друг |
> | `ErrFriendNotInLobby` | друг не в лобби, приглашение некуда доставить |
> | `ErrFriendNoFunds` | у друга не хватит фишек на ставку |
> | `ErrInviteNotFound` | отказываются от приглашения, которого уже нет |

> [!note] Почему типы стола в `model`, а не в `ws`
> Сервис должен знать ставку стола и отдавать её фронту, но импортировать пакет сокетов ему нельзя (инверсия зависимостей, как в [[friends-plan]]). Поэтому хаб отдаёт наружу нейтральный `model.InviteTable`, а не `*ws.Room`.

### Практика

Файл `internal/model/invite.go`: ошибки из таблицы, `InviteTable` (id, режим, колода, мест, ставка) и `TableInvite` (стол, кто зовёт: id и имя, когда истекает).

> [!example]- Решение: model/invite.go
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
> 	ErrTableNotFound    = errors.New("table not found")
> 	ErrNotAtTable       = errors.New("inviter does not sit at this table")
> 	ErrTableStarted     = errors.New("game already started")
> 	ErrTableFull        = errors.New("no free seat at the table")
> 	ErrAlreadyInvited   = errors.New("player already invited to this table")
> 	ErrFriendBusy       = errors.New("friend sits at another table")
> 	ErrNotFriends       = errors.New("players are not friends")
> 	ErrFriendNotInLobby = errors.New("friend is not in the lobby")
> 	ErrFriendNoFunds    = errors.New("friend cannot afford the stake")
> 	ErrInviteNotFound   = errors.New("invite not found")
> )
>
> type InviteTable struct {
> 	ID         string
> 	Mode       string
> 	Deck       int
> 	MaxPlayers int
> 	Stake      int64
> }
>
> type TableInvite struct {
> 	Table     InviteTable
> 	FromID    uuid.UUID
> 	FromName  string
> 	ExpiresAt time.Time
> }
> ```

### Готово, когда

- [ ] `go build ./...` проходит
- [ ] можешь назвать, какую ошибку получит каждый случай из таблицы

---

## Шаг 2. Комната: место под приглашение

### Зачем

Сейчас бронь в комнате одна (`reservations`), и у неё два смысла сразу: «место занято» и «игрок занят этим столом». Для создателя стола это правильно: так работает правило «один игрок — один открытый стол». Но для приглашённого второй смысл вреден: пока Боб думает 45 секунд, он не сможет нажать «Быстрая игра», сервер ответит «Вы уже играете за другим столом».

Есть и вторая дыра: при входе за стол сервер считает только подключённых (`len(r.clients) >= r.maxUsers`). Если Боб думает, а за стол заходит посторонний, он займёт место Боба.

### Теория

> [!note] Два вида брони
>
> | | `reservations` (уже есть) | `invites` (новое) |
> |---|---|---|
> | Держит место | да | да |
> | Делает игрока «занятым» (`Claims`) | да, у обычных столов | **нет** |
> | Срок | 15 секунд | 45 секунд |
> | Кто ставит | создание стола, быстрая игра | приглашение |
>
> `Claims` не трогаем: он смотрит только на `reservations`. Поэтому приглашение само собой не мешает Бобу играть в другом месте.

> [!note] Срок без таймеров
> Бронь — это просто «до какого времени». Никаких `time.AfterFunc` на каждое приглашение: при каждом чтении сравниваем `time.Now().Before(until)`. Просроченная бронь сама перестаёт считаться. Так уже устроены `reservations`, и это дешевле и проще, чем горутина или таймер на каждую бронь.

> [!note] Где проверять свободное место
> Проверку делаем **в горутине комнаты**, в `handleRegister`, в момент посадки. Регистрации обрабатываются по одной (горутина-владелец из [[lobby-socket-plan]]), поэтому двое не сядут на последнее место одновременно.
> Свободно = `maxUsers` − подключённые − **чужие** действующие брони обоих видов. Своя бронь не мешает своему входу, поэтому её не считаем.

> [!note] Объявление в комнату
> Отказ Боба надо показать всем за столом, а карта `clients` принадлежит горутине комнаты. Решение то же, что в лобби: канал `announce` и ветка в `Run`, которая рассылает сообщение.
> Отправка в канал **не блокирует** HTTP-обработчик: канал с буфером, а если он полон, сообщение отбрасывается (`default`). Потерять строчку «Боб отказался» не страшно, а подвесить обработчик запроса — страшно.

### Практика

В `internal/delivery/ws/room.go`:

1. Константа `inviteTTL = 45 * time.Second` рядом с `reservationTTL`.
2. Поля `invites map[string]time.Time` и `announce chan []byte` (буфер 8), инициализация в `NewRoom`.
3. Методы `Invite(playerID, ttl)`, `Uninvite(playerID) bool` (вернуть, было ли **действующее** приглашение), `Invited(playerID) bool`.
4. `Occupancy` считает и `invites`; `seat` удаляет запись из `invites`.
5. `freeSeatsFor(playerID) int` и проверка в `handleRegister` вместо `len(r.clients) >= r.maxUsers`.
6. `Announce(data []byte)` и ветка `case data := <-r.announce` в `Run`.

> [!warning] Ловушки
> - `freeSeatsFor` читает `r.clients`. Это безопасно, только потому что её зовут из `Run`. Не вызывай её из хаба.
> - Не добавляй `invites` в `Claims`, иначе вернётся проблема «Боб занят, пока думает».
> - Старые тесты должны остаться зелёными. Если упал тест быстрой игры, проверь, что своя бронь не считается против себя.

### Проверь себя

> [!question]- Почему посторонний раньше мог занять место, хотя в списке стол выглядел полным?
> Список считает `Occupancy`, а там брони учитывались. А вход за стол проверял только `len(r.clients)`. Два места в коде отвечали на один вопрос по-разному. Теперь вход тоже учитывает брони.

> [!question]- Что будет, если Боб нажмёт «Сесть» через минуту?
> Приглашение просрочено и не держит место. Если место свободно, Боб сядет как обычный игрок. Если занято, получит «стол уже заполнен». Отдельной ошибки «приглашение истекло» при входе не нужно.

> [!question]- Почему `Announce` не ждёт, пока сообщение заберут?
> Его вызывает HTTP-обработчик отказа. Если горутина комнаты занята или комната закрывается, ожидание подвесило бы запрос. Буфер плюс `default` делают отправку мгновенной ценой возможной потери уведомления.

> [!example]- Решение: изменения в ws/room.go
> Константа и поля:
> ```go
> 	reservationTTL  = 15 * time.Second
> 	inviteTTL       = 45 * time.Second
> ```
> ```go
> 	reservations      map[string]time.Time
> 	invites           map[string]time.Time
> ```
> ```go
> 	incoming          chan IncomingMessage
> 	announce          chan []byte
> ```
> В `NewRoom`:
> ```go
> 		invites:           make(map[string]time.Time),
> ```
> ```go
> 		announce:          make(chan []byte, 8),
> ```
> В `Occupancy` после цикла по `reservations`:
> ```go
> 	for id, until := range r.invites {
> 		if !r.seated[id] && now.Before(until) {
> 			n++
> 		}
> 	}
> ```
> В `seat` после `delete(r.reservations, playerID)`:
> ```go
> 	delete(r.invites, playerID)
> ```
> В `Run`, новая ветка `select`:
> ```go
> 		case data := <-r.announce:
> 			for _, c := range r.clients {
> 				c.Send(data)
> 			}
> ```
> В `handleRegister` вместо блока `if len(r.clients) >= r.maxUsers { ... }`:
> ```go
> 	if _, alreadyPresent := r.clients[playerID]; !alreadyPresent && r.freeSeatsFor(playerID) <= 0 {
> 		r.sendError(client, "стол уже заполнен")
> 		client.Close()
> 		return
> 	}
> ```
> Новые методы в конце файла:
> ```go
> func (r *Room) Invite(playerID string, ttl time.Duration) {
> 	r.mu.Lock()
> 	r.invites[playerID] = time.Now().Add(ttl)
> 	r.mu.Unlock()
> }
>
> func (r *Room) Uninvite(playerID string) bool {
> 	r.mu.Lock()
> 	defer r.mu.Unlock()
> 	until, ok := r.invites[playerID]
> 	delete(r.invites, playerID)
> 	return ok && time.Now().Before(until)
> }
>
> func (r *Room) Invited(playerID string) bool {
> 	r.mu.RLock()
> 	defer r.mu.RUnlock()
> 	until, ok := r.invites[playerID]
> 	return ok && time.Now().Before(until)
> }
>
> func (r *Room) freeSeatsFor(playerID string) int {
> 	r.mu.RLock()
> 	defer r.mu.RUnlock()
>
> 	now := time.Now()
> 	held := len(r.clients)
> 	for _, booked := range []map[string]time.Time{r.reservations, r.invites} {
> 		for id, until := range booked {
> 			if id != playerID && !r.seated[id] && now.Before(until) {
> 				held++
> 			}
> 		}
> 	}
> 	return r.maxUsers - held
> }
>
> func (r *Room) Announce(data []byte) {
> 	select {
> 	case r.announce <- data:
> 	case <-r.stop:
> 	default:
> 	}
> }
> ```

### Готово, когда

- [ ] `go test -race ./internal/delivery/...` зелёный, старые тесты не правились
- [ ] можешь объяснить разницу между `reservations` и `invites`

---

## Шаг 3. Хаб: пригласить, отказаться, доставить

### Зачем

Хаб знает все столы и умеет проверять «игрок занят где-то ещё» (`busyLocked`). Значит, приглашение со всеми проверками стола делается здесь.

### Теория

> [!note] Проверить и записать под одной блокировкой
> Все проверки (стол есть, ты за ним, партия не началась, друга ещё не звали, друг не занят, есть место) и сама бронь делаются под `h.mu.Lock()`. Иначе между «место есть» и «держу место» другой запрос успел бы занять последнее место. Это тот же TOCTOU, что в [[friends-plan]], только в памяти, а не в базе.

> [!note] «За своим столом» — это `Holds` или `Reserved`
> Создатель стола сначала только бронирует место (15 секунд), а садится, когда страница стола подключит сокет. Звать можно в обоих состояниях. А столы быстрой игры (`matchmaking`) не для приглашений: это дуэль с незнакомцем.

> [!note] Здесь шлём данные, а не сигнал
> Для друзей мы слали пустое `friends_changed`, и клиент перезапрашивал список из базы. Приглашение в базе не хранится, перезапросить его неоткуда, это разовое событие. Поэтому `table_invite` несёт всё, что нужно для карточки: кто зовёт, ставку, режим, колоду, до какого времени.

### Практика

1. `ws/message.go`: константы `TypeTableInvite = "table_invite"`, `TypeInviteDeclined = "invite_declined"`, структуры `TableInvitePayload` и `InviteDeclinedPayload`.
2. `ws/hub.go`:
   - `inviteTable(room) model.InviteTable` — помощник;
   - `InviteTable(roomID) (model.InviteTable, bool)` — сервису нужна ставка до брони;
   - `Invite(roomID, from, to string) (model.InviteTable, time.Time, error)` — проверки и бронь под `h.mu`;
   - `Decline(roomID, playerID, username string) bool` — снять приглашение и объявить за столом;
   - `SendInvite(to uuid.UUID, inv model.TableInvite)` — доставка в лобби через `lobby.Notify`.

> [!warning] Ловушки
> - Порядок проверок в `Invite` важен для понятных ошибок: сначала «стол есть», потом «ты за ним», потом всё остальное.
> - `room.Occupancy() >= room.maxUsers` проверяй **после** «друга уже звали», иначе повторное приглашение на последнее место ответит «нет мест» вместо «уже приглашён».
> - `Decline` не должен отвечать «ок», если приглашения уже нет: `Uninvite` возвращает `false` для просроченного.

### Проверь себя

> [!question]- Почему `ErrFriendBusy` проверяется через `busyLocked`, а не через статус «за столом»?
> Статус говорит, на какой странице друг. А `busyLocked` говорит, держит ли он место за другим столом, в том числе свежесозданным, на который ещё не зашёл. Для правила «один игрок — один стол» важно второе.

> [!question]- Зачем `Decline` объявляет за столом, а не шлёт Алисе в лобби?
> Алиса в этот момент на странице стола, её лобби-сокет закрыт. Её соединение — сокет комнаты, туда и шлём.

> [!example]- Решение: ws/message.go (что добавить)
> В блок серверных типов:
> ```go
> 	TypeTableInvite        = "table_invite"
> 	TypeInviteDeclined     = "invite_declined"
> ```
> В конец файла:
> ```go
> type TableInvitePayload struct {
> 	TableID    string `json:"tableId"`
> 	FromID     string `json:"fromId"`
> 	FromName   string `json:"fromName"`
> 	Mode       string `json:"mode"`
> 	Deck       int    `json:"deck"`
> 	MaxPlayers int    `json:"maxPlayers"`
> 	Stake      int64  `json:"stake"`
> 	ExpiresAt  string `json:"expiresAt"`
> }
>
> type InviteDeclinedPayload struct {
> 	Username string `json:"username"`
> }
> ```

> [!example]- Решение: ws/hub.go (что добавить)
> ```go
> func inviteTable(room *Room) model.InviteTable {
> 	return model.InviteTable{
> 		ID:         room.id,
> 		Mode:       room.gameMode.String(),
> 		Deck:       int(room.deckType),
> 		MaxPlayers: room.maxUsers,
> 		Stake:      room.stake,
> 	}
> }
>
> func (h *Hub) InviteTable(roomID string) (model.InviteTable, bool) {
> 	room, ok := h.GetRoom(roomID)
> 	if !ok {
> 		return model.InviteTable{}, false
> 	}
> 	return inviteTable(room), true
> }
>
> func (h *Hub) Invite(roomID, from, to string) (model.InviteTable, time.Time, error) {
> 	h.mu.Lock()
> 	defer h.mu.Unlock()
>
> 	room, ok := h.rooms[roomID]
> 	switch {
> 	case !ok:
> 		return model.InviteTable{}, time.Time{}, model.ErrTableNotFound
> 	case room.matchmaking || !(room.Holds(from) || room.Reserved(from)):
> 		return model.InviteTable{}, time.Time{}, model.ErrNotAtTable
> 	case room.Started():
> 		return model.InviteTable{}, time.Time{}, model.ErrTableStarted
> 	case room.Holds(to) || room.Invited(to):
> 		return model.InviteTable{}, time.Time{}, model.ErrAlreadyInvited
> 	case h.busyLocked(to, room):
> 		return model.InviteTable{}, time.Time{}, model.ErrFriendBusy
> 	case room.Occupancy() >= room.maxUsers:
> 		return model.InviteTable{}, time.Time{}, model.ErrTableFull
> 	}
>
> 	room.Invite(to, inviteTTL)
> 	return inviteTable(room), time.Now().Add(inviteTTL), nil
> }
>
> func (h *Hub) Decline(roomID, playerID, username string) bool {
> 	room, ok := h.GetRoom(roomID)
> 	if !ok || !room.Uninvite(playerID) {
> 		return false
> 	}
> 	if data, err := NewMessage(TypeInviteDeclined, InviteDeclinedPayload{Username: username}); err == nil {
> 		room.Announce(data)
> 	}
> 	return true
> }
>
> func (h *Hub) SendInvite(to uuid.UUID, inv model.TableInvite) {
> 	data, err := NewMessage(TypeTableInvite, TableInvitePayload{
> 		TableID:    inv.Table.ID,
> 		FromID:     inv.FromID.String(),
> 		FromName:   inv.FromName,
> 		Mode:       inv.Table.Mode,
> 		Deck:       inv.Table.Deck,
> 		MaxPlayers: inv.Table.MaxPlayers,
> 		Stake:      inv.Table.Stake,
> 		ExpiresAt:  inv.ExpiresAt.Format(time.RFC3339),
> 	})
> 	if err != nil {
> 		log.Printf("[Hub] table_invite: %v", err)
> 		return
> 	}
> 	h.lobby.Notify(to.String(), data)
> }
> ```

### Готово, когда

- [ ] `go build ./...` проходит
- [ ] в `Invite` нет ни одного обращения к комнате вне `h.mu`

---

## Шаг 4. Тесты комнаты и хаба

### Зачем

Здесь самая хитрая логика проекта после экономики: брони двух видов, места, гонки. Без тестов её не проверить руками — нужно три браузера и секундомер.

### Теория

> [!note] Что именно доказывать
> 1. Приглашение держит место, но не делает друга «занятым».
> 2. Посторонний не садится на место приглашённого, а сам приглашённый садится.
> 3. Все отказы `Invite` — одной табличкой.
> 4. Отказ освобождает место, сообщает столу, а второй отказ ничего не находит.
> 5. Приглашение доходит до лобби.

> [!note] Ждать события, а не время
> Регистрация в комнате асинхронная. Вместо `time.Sleep(100ms)` и надежды крутим короткий цикл: «уже сел?» или «уже пришла ошибка?», с общим дедлайном в секунду. Тест быстрый, когда всё хорошо, и не падает случайно на медленной машине.

### Практика

Файл `internal/delivery/ws/invite_test.go`, тесты из списка. Помощники `createTestClient`, `lobbyClient` и `lastOfType` уже есть в пакете.

```bash
go test -race ./internal/delivery/ws/ -run "Invite|Decline|Stranger" -count=10
```

> [!example]- Решение: ws/invite_test.go
> ```go
> package ws
>
> import (
> 	"encoding/json"
> 	"errors"
> 	"testing"
> 	"time"
>
> 	"bito/internal/game"
> 	"bito/internal/model"
>
> 	"github.com/google/uuid"
> )
>
> func waitHolds(t *testing.T, room *Room, id string) {
> 	t.Helper()
> 	deadline := time.Now().Add(time.Second)
> 	for !room.Holds(id) {
> 		if time.Now().After(deadline) {
> 			t.Fatalf("%s never took a seat", id)
> 		}
> 		time.Sleep(5 * time.Millisecond)
> 	}
> }
>
> func companyTable(t *testing.T, h *Hub, creator string, seats int) *Room {
> 	t.Helper()
> 	room, err := h.CreateTable(RoomSettings{Mode: game.ModePodkidnoy, Deck: game.Deck36, MaxPlayers: seats, Stake: 100, CreatedBy: creator})
> 	if err != nil {
> 		t.Fatal(err)
> 	}
> 	t.Cleanup(room.Close)
> 	return room
> }
>
> func TestRoom_InviteHoldsSeatWithoutClaimingPlayer(t *testing.T) {
> 	h := NewHub(nil, nil, nil)
> 	room := companyTable(t, h, "alice", 2)
>
> 	if _, _, err := h.Invite(room.ID(), "alice", "bob"); err != nil {
> 		t.Fatalf("invite: %v", err)
> 	}
> 	if room.Occupancy() != 2 {
> 		t.Fatalf("occupancy = %d, want 2 (alice reserved, bob invited)", room.Occupancy())
> 	}
> 	if h.SeatedElsewhere("bob", "") {
> 		t.Fatal("an invite must not block bob from other tables")
> 	}
> }
>
> func TestRoom_StrangerCannotTakeInvitedSeat(t *testing.T) {
> 	h := NewHub(nil, nil, nil)
> 	room := companyTable(t, h, "alice", 2)
> 	if _, _, err := h.Invite(room.ID(), "alice", "bob"); err != nil {
> 		t.Fatal(err)
> 	}
>
> 	alice := createTestClient("alice", "Alice", room)
> 	room.RegisterClient(alice)
> 	waitHolds(t, room, "alice")
>
> 	eve := createTestClient("eve", "Eve", room)
> 	room.RegisterClient(eve)
> 	deadline := time.Now().Add(time.Second)
> 	for {
> 		if _, ok := lastOfType[ErrorPayload](t, eve, TypeError); ok {
> 			break
> 		}
> 		if room.Holds("eve") || time.Now().After(deadline) {
> 			t.Fatal("eve took the seat kept for bob")
> 		}
> 		time.Sleep(5 * time.Millisecond)
> 	}
>
> 	bob := createTestClient("bob", "Bob", room)
> 	room.RegisterClient(bob)
> 	waitHolds(t, room, "bob")
> }
>
> func TestHub_InviteRules(t *testing.T) {
> 	h := NewHub(nil, nil, nil)
> 	room := companyTable(t, h, "alice", 3)
> 	other := companyTable(t, h, "carol", 3)
>
> 	cases := []struct {
> 		name   string
> 		roomID string
> 		from   string
> 		to     string
> 		want   error
> 	}{
> 		{"unknown table", "nope", "alice", "bob", model.ErrTableNotFound},
> 		{"not my table", room.ID(), "mallory", "bob", model.ErrNotAtTable},
> 		{"friend owns another table", room.ID(), "alice", "carol", model.ErrFriendBusy},
> 		{"ok", room.ID(), "alice", "bob", nil},
> 		{"twice", room.ID(), "alice", "bob", model.ErrAlreadyInvited},
> 		{"last seat", room.ID(), "alice", "dave", nil},
> 		{"no seats left", room.ID(), "alice", "erin", model.ErrTableFull},
> 	}
> 	for _, tc := range cases {
> 		_, _, err := h.Invite(tc.roomID, tc.from, tc.to)
> 		if !errors.Is(err, tc.want) {
> 			t.Errorf("%s: err = %v, want %v", tc.name, err, tc.want)
> 		}
> 	}
> 	_ = other
> }
>
> func TestHub_DeclineFreesSeatAndTellsTheTable(t *testing.T) {
> 	h := NewHub(nil, nil, nil)
> 	room := companyTable(t, h, "alice", 2)
> 	alice := createTestClient("alice", "Alice", room)
> 	room.RegisterClient(alice)
> 	waitHolds(t, room, "alice")
>
> 	if _, _, err := h.Invite(room.ID(), "alice", "bob"); err != nil {
> 		t.Fatal(err)
> 	}
> 	if !h.Decline(room.ID(), "bob", "Bob") {
> 		t.Fatal("decline of a live invite returned false")
> 	}
> 	if h.Decline(room.ID(), "bob", "Bob") {
> 		t.Fatal("second decline should find nothing")
> 	}
> 	if room.Occupancy() != 1 {
> 		t.Fatalf("occupancy after decline = %d, want 1", room.Occupancy())
> 	}
>
> 	deadline := time.Now().Add(time.Second)
> 	for {
> 		if msg, ok := lastOfType[InviteDeclinedPayload](t, alice, TypeInviteDeclined); ok {
> 			if msg.Payload.Username != "Bob" {
> 				t.Fatalf("declined by %q", msg.Payload.Username)
> 			}
> 			return
> 		}
> 		if time.Now().After(deadline) {
> 			t.Fatal("alice was not told about the decline")
> 		}
> 		time.Sleep(5 * time.Millisecond)
> 	}
> }
>
> func TestHub_SendInviteReachesLobby(t *testing.T) {
> 	h := NewHub(nil, nil, nil)
> 	bobID := uuid.New()
> 	tab := lobbyClient(bobID.String(), h.lobby)
> 	h.lobby.Register(tab)
> 	h.lobby.Online()
> 	for len(tab.send) > 0 {
> 		<-tab.send
> 	}
>
> 	h.SendInvite(bobID, model.TableInvite{Table: model.InviteTable{ID: "t1", Stake: 100}, FromName: "Alice", ExpiresAt: time.Now()})
> 	h.lobby.Online()
>
> 	select {
> 	case data := <-tab.send:
> 		var msg Message[TableInvitePayload]
> 		if err := json.Unmarshal(data, &msg); err != nil || msg.Type != TypeTableInvite || msg.Payload.TableID != "t1" || msg.Payload.FromName != "Alice" {
> 			t.Fatalf("got %s (%v)", data, err)
> 		}
> 	default:
> 		t.Fatal("no table_invite in the lobby")
> 	}
> }
> ```

### Готово, когда

- [ ] тесты выше зелёные 10 раз подряд с `-race`

---

## Шаг 5. Сервис

### Зачем

Правила, которые не про столы: звать можно только **друга**, только того, кто **в лобби** (иначе приглашение некуда доставить), и только если у него **хватит фишек**. Плюс имя приглашающего для карточки.

### Теория

> [!note] Порядок проверок
> 1. Дешёвые и «про права»: друг ли это, в лобби ли он.
> 2. Стол и ставка (`InviteTable`), потом баланс друга.
> 3. Имя приглашающего.
> 4. Только потом бронь места (`tables.Invite`) и доставка.
>
> Ставку читаем **до** брони, чтобы при нехватке фишек не пришлось бронь откатывать. Мелкая гонка («ставка поменялась между чтением и бронью») невозможна: ставка стола не меняется.

> [!note] Переиспользуем интерфейсы
> `RelationSource`, `PresenceSource` и `PlayerUsers` уже объявлены для друзей и профиля. Новые нужны только для столов (`InviteTables`), баланса (`BalanceSource`) и доставки (`InviteNotifier`). `Hub` реализует три из них, `EconomyService` — баланс.

### Практика

1. `internal/service/invite.go`: `InviteService` с `Invite(ctx, me, roomID, friend)` и `Decline(ctx, me, roomID)`.
2. `internal/service/invite_test.go`: удачное приглашение доставлено; табличка отказов (не друг, друг за столом, не в сети, нет стола, мало фишек, стол отказал), и ни один отказ не доставлен; отказ без приглашения — `ErrInviteNotFound`.

> [!warning] Ловушки
> - Ни одна ошибка не должна приводить к `SendInvite`.
> - Статус друга бери из `Presence`, сравнивай именно с `PresenceOnline`: «за столом» тоже не подходит.

### Проверь себя

> [!question]- Почему приглашать можно только тех, кто в лобби?
> Доставка идёт через лобби-сокет. Если друг за столом или закрыл вкладку, сообщение никуда не придёт, а место будет 45 секунд держаться впустую. Лучше сразу честно ответить «друг сейчас не в лобби».

> [!question]- Зачем проверять баланс друга при приглашении, если он всё равно проверяется при входе за стол?
> При входе проверка тоже есть, она последняя линия обороны. Но тогда Боб нажал бы «Сесть» и получил отказ, а место 45 секунд держалось бы зря. Ранняя проверка даёт Алисе понятную ошибку сразу.

> [!example]- Решение: service/invite.go
> ```go
> package service
>
> import (
> 	"context"
> 	"time"
>
> 	"bito/internal/model"
>
> 	"github.com/google/uuid"
> )
>
> type InviteTables interface {
> 	InviteTable(roomID string) (model.InviteTable, bool)
> 	Invite(roomID, from, to string) (model.InviteTable, time.Time, error)
> 	Decline(roomID, playerID, username string) bool
> }
>
> type InviteNotifier interface {
> 	SendInvite(to uuid.UUID, inv model.TableInvite)
> }
>
> type BalanceSource interface {
> 	Balance(ctx context.Context, userID string) (int64, error)
> }
>
> type InviteService struct {
> 	tables    InviteTables
> 	relations RelationSource
> 	presence  PresenceSource
> 	users     PlayerUsers
> 	balances  BalanceSource
> 	notify    InviteNotifier
> }
>
> func NewInviteService(tables InviteTables, relations RelationSource, presence PresenceSource, users PlayerUsers, balances BalanceSource, notify InviteNotifier) *InviteService {
> 	return &InviteService{tables: tables, relations: relations, presence: presence, users: users, balances: balances, notify: notify}
> }
>
> func (s *InviteService) Invite(ctx context.Context, me uuid.UUID, roomID string, friend uuid.UUID) (model.TableInvite, error) {
> 	relation, err := s.relations.Between(ctx, me, friend)
> 	if err != nil {
> 		return model.TableInvite{}, err
> 	}
> 	if relation != model.RelationFriend {
> 		return model.TableInvite{}, model.ErrNotFriends
> 	}
> 	if s.presence.Presence([]uuid.UUID{friend})[friend] != model.PresenceOnline {
> 		return model.TableInvite{}, model.ErrFriendNotInLobby
> 	}
>
> 	table, ok := s.tables.InviteTable(roomID)
> 	if !ok {
> 		return model.TableInvite{}, model.ErrTableNotFound
> 	}
> 	balance, err := s.balances.Balance(ctx, friend.String())
> 	if err != nil {
> 		return model.TableInvite{}, err
> 	}
> 	if balance < table.Stake {
> 		return model.TableInvite{}, model.ErrFriendNoFunds
> 	}
>
> 	inviter, err := s.users.GetByID(ctx, me)
> 	if err != nil {
> 		return model.TableInvite{}, err
> 	}
>
> 	table, expiresAt, err := s.tables.Invite(roomID, me.String(), friend.String())
> 	if err != nil {
> 		return model.TableInvite{}, err
> 	}
>
> 	inv := model.TableInvite{Table: table, FromID: me, FromName: inviter.Name, ExpiresAt: expiresAt}
> 	s.notify.SendInvite(friend, inv)
> 	return inv, nil
> }
>
> func (s *InviteService) Decline(ctx context.Context, me uuid.UUID, roomID string) error {
> 	user, err := s.users.GetByID(ctx, me)
> 	if err != nil {
> 		return err
> 	}
> 	if !s.tables.Decline(roomID, me.String(), user.Name) {
> 		return model.ErrInviteNotFound
> 	}
> 	return nil
> }
> ```

> [!example]- Решение: service/invite_test.go
> ```go
> package service
>
> import (
> 	"context"
> 	"errors"
> 	"testing"
> 	"time"
>
> 	"bito/internal/model"
>
> 	"github.com/google/uuid"
> )
>
> type fakeTables struct {
> 	table    model.InviteTable
> 	found    bool
> 	err      error
> 	invited  []string
> 	declined bool
> }
>
> func (f *fakeTables) InviteTable(string) (model.InviteTable, bool) { return f.table, f.found }
>
> func (f *fakeTables) Invite(_, _, to string) (model.InviteTable, time.Time, error) {
> 	if f.err != nil {
> 		return model.InviteTable{}, time.Time{}, f.err
> 	}
> 	f.invited = append(f.invited, to)
> 	return f.table, time.Now().Add(time.Minute), nil
> }
>
> func (f *fakeTables) Decline(string, string, string) bool { return f.declined }
>
> type fixedBalance int64
>
> func (b fixedBalance) Balance(context.Context, string) (int64, error) { return int64(b), nil }
>
> type fakeInviteNotifier struct{ sent []model.TableInvite }
>
> func (f *fakeInviteNotifier) SendInvite(_ uuid.UUID, inv model.TableInvite) {
> 	f.sent = append(f.sent, inv)
> }
>
> type inviteSetup struct {
> 	me, bob  uuid.UUID
> 	tables   *fakeTables
> 	relation *fixedRelation
> 	presence fakePresence
> 	balance  fixedBalance
> 	notify   *fakeInviteNotifier
> }
>
> func newInviteSetup() *inviteSetup {
> 	me, bob := uuid.New(), uuid.New()
> 	return &inviteSetup{
> 		me:       me,
> 		bob:      bob,
> 		tables:   &fakeTables{table: model.InviteTable{ID: "t1", Stake: 500}, found: true},
> 		relation: &fixedRelation{relation: model.RelationFriend},
> 		presence: fakePresence{bob: model.PresenceOnline},
> 		balance:  2500,
> 		notify:   &fakeInviteNotifier{},
> 	}
> }
>
> func (s *inviteSetup) service() *InviteService {
> 	users := usersByID{s.me: {ID: s.me, Name: "Alice"}, s.bob: {ID: s.bob, Name: "Bob"}}
> 	return NewInviteService(s.tables, s.relation, s.presence, users, s.balance, s.notify)
> }
>
> func TestInviteService_SendsInviteToFriendInLobby(t *testing.T) {
> 	s := newInviteSetup()
> 	inv, err := s.service().Invite(context.Background(), s.me, "t1", s.bob)
> 	if err != nil {
> 		t.Fatal(err)
> 	}
> 	if inv.FromName != "Alice" || inv.Table.ID != "t1" {
> 		t.Fatalf("invite = %+v", inv)
> 	}
> 	if len(s.tables.invited) != 1 || s.tables.invited[0] != s.bob.String() {
> 		t.Fatalf("seat kept for %v", s.tables.invited)
> 	}
> 	if len(s.notify.sent) != 1 {
> 		t.Fatalf("notices sent = %d, want 1", len(s.notify.sent))
> 	}
> }
>
> func TestInviteService_Refusals(t *testing.T) {
> 	cases := []struct {
> 		name  string
> 		setup func(*inviteSetup)
> 		want  error
> 	}{
> 		{"not friends", func(s *inviteSetup) { s.relation.relation = model.RelationOutgoing }, model.ErrNotFriends},
> 		{"friend at a table", func(s *inviteSetup) { s.presence[s.bob] = model.PresencePlaying }, model.ErrFriendNotInLobby},
> 		{"friend offline", func(s *inviteSetup) { delete(s.presence, s.bob) }, model.ErrFriendNotInLobby},
> 		{"no table", func(s *inviteSetup) { s.tables.found = false }, model.ErrTableNotFound},
> 		{"poor friend", func(s *inviteSetup) { s.balance = 100 }, model.ErrFriendNoFunds},
> 		{"table refuses", func(s *inviteSetup) { s.tables.err = model.ErrTableFull }, model.ErrTableFull},
> 	}
> 	for _, tc := range cases {
> 		t.Run(tc.name, func(t *testing.T) {
> 			s := newInviteSetup()
> 			tc.setup(s)
> 			if _, err := s.service().Invite(context.Background(), s.me, "t1", s.bob); !errors.Is(err, tc.want) {
> 				t.Fatalf("err = %v, want %v", err, tc.want)
> 			}
> 			if len(s.notify.sent) != 0 {
> 				t.Fatal("refused invite was still delivered")
> 			}
> 		})
> 	}
> }
>
> func TestInviteService_Decline(t *testing.T) {
> 	s := newInviteSetup()
> 	if err := s.service().Decline(context.Background(), s.bob, "t1"); !errors.Is(err, model.ErrInviteNotFound) {
> 		t.Fatalf("decline without invite: err = %v", err)
> 	}
> 	s.tables.declined = true
> 	if err := s.service().Decline(context.Background(), s.bob, "t1"); err != nil {
> 		t.Fatalf("decline: %v", err)
> 	}
> }
> ```

### Готово, когда

- [ ] `go test -race ./internal/service/ -run Invite` зелёный

---

## Шаг 6. HTTP и стол на двоих

### Зачем

Эндпоинты, перевод ошибок в статусы и одно осознанное изменение старого правила.

### Теория

> [!note] 403, 404 или 409
>
> | Код | Смысл | Ошибки |
> |---|---|---|
> | `404` | того, о чём речь, уже нет | стол закрыт, приглашение недействительно |
> | `403` | тебе так нельзя в принципе | не друг, не твой стол |
> | `409` | можно, но не сейчас: мешает текущее состояние | партия началась, мест нет, уже позвали, друг занят, не в лобби, мало фишек |
>
> Фронту это помогает: на 409 можно предложить «попробовать позже», на 403 — нет.

> [!note] Стол на двоих: меняем правило осознанно
> В `table_handler.go` стоит `minCompanySize = 3`, а тест `TestTables_CreateIsForCompaniesOnly` это правило закрепляет: дуэли шли только через быструю игру с незнакомцем. Для приглашений нужна дуэль с другом, иначе для живой демонстрации понадобится третий аккаунт.
> Тест здесь — запись старого решения. Мы меняем решение, поэтому **меняем и тест**, а не удаляем его: теперь он проверяет, что стол на двоих создаётся. Посторонний за такой стол не сядет: второе место держит приглашение (шаг 2).

### Практика

1. `internal/delivery/http/invite_handler.go`: интерфейс `InviteService`, `InviteRequest` (`userId`), `InviteResponse` (`tableId`, `expiresAt`), обработчики `Invite` и `Decline`, `respondInviteError`.
2. `server.go`: поле `inviteHandler`, поле `Invite` в `Handlers`, два маршрута внутри группы `/api/v1/tables`.
3. `app.go`: собрать сервис из `hub`, `FriendRepository`, `userRepo`, `economyService`.
4. `table_handler.go`: `minTableSize = 2` вместо `minCompanySize = 3`, текст ошибки «between 2 and 6».
5. `economy_handler_test.go`: тест про «только компании» превратить в «стол на двоих создаётся».
6. `invite_handler_test.go`: id стола и друга доходят до сервиса, кривой `userId` → 400, табличка статусов, отказ 204 и 404.

> [!warning] Ловушки
> - Маршруты приглашений должны быть **внутри** группы `/api/v1/tables`, под её `AuthMiddleware`.
> - `decodeJSON` запрещает лишние поля: тело `{"friendId": ...}` получит 400.

> [!example]- Решение: delivery/http/invite_handler.go
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
> type InviteService interface {
> 	Invite(ctx context.Context, me uuid.UUID, roomID string, friend uuid.UUID) (model.TableInvite, error)
> 	Decline(ctx context.Context, me uuid.UUID, roomID string) error
> }
>
> type InviteRequest struct {
> 	UserID string `json:"userId"`
> }
>
> type InviteResponse struct {
> 	TableID   string `json:"tableId"`
> 	ExpiresAt string `json:"expiresAt"`
> }
>
> type InviteHandler struct {
> 	invites InviteService
> }
>
> func NewInviteHandler(invites InviteService) *InviteHandler {
> 	return &InviteHandler{invites: invites}
> }
>
> func (ih *InviteHandler) Invite(w http.ResponseWriter, r *http.Request) {
> 	me, ok := UserIDFromContext(r.Context())
> 	if !ok {
> 		respondError(w, http.StatusUnauthorized, "unauthorized")
> 		return
> 	}
>
> 	var req InviteRequest
> 	if !decodeJSON(w, r, &req) {
> 		return
> 	}
> 	friend, err := uuid.Parse(req.UserID)
> 	if err != nil {
> 		respondError(w, http.StatusBadRequest, "неверный id игрока")
> 		return
> 	}
>
> 	inv, err := ih.invites.Invite(r.Context(), me, chi.URLParam(r, "id"), friend)
> 	if err != nil {
> 		respondInviteError(w, me, err)
> 		return
> 	}
> 	respondJSON(w, http.StatusCreated, InviteResponse{TableID: inv.Table.ID, ExpiresAt: inv.ExpiresAt.Format(time.RFC3339)})
> }
>
> func (ih *InviteHandler) Decline(w http.ResponseWriter, r *http.Request) {
> 	me, ok := UserIDFromContext(r.Context())
> 	if !ok {
> 		respondError(w, http.StatusUnauthorized, "unauthorized")
> 		return
> 	}
> 	if err := ih.invites.Decline(r.Context(), me, chi.URLParam(r, "id")); err != nil {
> 		respondInviteError(w, me, err)
> 		return
> 	}
> 	w.WriteHeader(http.StatusNoContent)
> }
>
> func respondInviteError(w http.ResponseWriter, me uuid.UUID, err error) {
> 	switch {
> 	case errors.Is(err, model.ErrTableNotFound):
> 		respondError(w, http.StatusNotFound, "стол уже закрыт")
> 	case errors.Is(err, model.ErrInviteNotFound):
> 		respondError(w, http.StatusNotFound, "приглашение уже недействительно")
> 	case errors.Is(err, model.ErrNotFriends):
> 		respondError(w, http.StatusForbidden, "звать за стол можно только друзей")
> 	case errors.Is(err, model.ErrNotAtTable):
> 		respondError(w, http.StatusForbidden, "звать можно только за свой стол")
> 	case errors.Is(err, model.ErrTableStarted):
> 		respondError(w, http.StatusConflict, "партия уже началась")
> 	case errors.Is(err, model.ErrTableFull):
> 		respondError(w, http.StatusConflict, "за столом нет свободных мест")
> 	case errors.Is(err, model.ErrAlreadyInvited):
> 		respondError(w, http.StatusConflict, "друг уже приглашён")
> 	case errors.Is(err, model.ErrFriendBusy):
> 		respondError(w, http.StatusConflict, "друг сейчас за другим столом")
> 	case errors.Is(err, model.ErrFriendNotInLobby):
> 		respondError(w, http.StatusConflict, "друг сейчас не в лобби")
> 	case errors.Is(err, model.ErrFriendNoFunds):
> 		respondError(w, http.StatusConflict, "у друга не хватает фишек на эту ставку")
> 	default:
> 		log.Printf("invites: %s: %v", me, err)
> 		respondError(w, http.StatusInternalServerError, "internal server error")
> 	}
> }
> ```

> [!example]- Решение: маршруты, сборка, стол на двоих
> В `server.go`, внутри `s.router.Route("/api/v1/tables", ...)` после `r.Get("/{id}", ...)`:
> ```go
> 			if s.inviteHandler != nil {
> 				r.Post("/{id}/invites", s.inviteHandler.Invite)
> 				r.Post("/{id}/invites/decline", s.inviteHandler.Decline)
> 			}
> ```
> Плюс поле `inviteHandler *InviteHandler` в `Server`, `Invite *InviteHandler` в `Handlers` и `inviteHandler: h.Invite` в `NewServer`.
>
> В `app.go`, в списке обработчиков:
> ```go
> 		Invite: deliveryHTTP.NewInviteHandler(service.NewInviteService(hub, postgres.NewFriendRepository(pool), hub, userRepo, economyService, hub)),
> ```
> В `table_handler.go`:
> ```go
> 	minTableSize   = 2
> ```
> ```go
> 	ErrInvalidMaxPlayers = errors.New("maxPlayers must be between 2 and 6")
> ```
> ```go
> 	if req.MaxPlayers < minTableSize || req.MaxPlayers > maxCompanySize {
> ```
> В `economy_handler_test.go` вместо `TestTables_CreateIsForCompaniesOnly`:
> ```go
> func TestTables_CreateAllowsTwoSeatsForInvites(t *testing.T) {
> 	router, _, token, _ := newEconomyRouter(t, &stubWallet{balance: 2500})
> 	if w := doRequest(router, http.MethodPost, "/api/v1/tables", token, `{"mode":"podkidnoy","deck":36,"maxPlayers":2}`); w.Code != http.StatusCreated {
> 		t.Errorf("two-player table via create: status = %d, want 201", w.Code)
> 	}
> }
> ```

> [!example]- Решение: delivery/http/invite_handler_test.go
> ```go
> package http
>
> import (
> 	"context"
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
> type stubInvites struct {
> 	err      error
> 	gotRoom  string
> 	gotGuest uuid.UUID
> }
>
> func (s *stubInvites) Invite(_ context.Context, _ uuid.UUID, roomID string, friend uuid.UUID) (model.TableInvite, error) {
> 	s.gotRoom, s.gotGuest = roomID, friend
> 	if s.err != nil {
> 		return model.TableInvite{}, s.err
> 	}
> 	return model.TableInvite{Table: model.InviteTable{ID: roomID}, ExpiresAt: time.Now()}, nil
> }
>
> func (s *stubInvites) Decline(_ context.Context, _ uuid.UUID, roomID string) error {
> 	s.gotRoom = roomID
> 	return s.err
> }
>
> func newInvitesRouter(t *testing.T, stub *stubInvites) (http.Handler, string) {
> 	t.Helper()
> 	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)
> 	hub := ws.NewHub(tokenManager, nil, nil)
> 	server := NewServer(":0", hub, Handlers{Table: NewTableHandler(hub, nil), Invite: NewInviteHandler(stub)}, tokenManager)
> 	token, err := tokenManager.GenerateAccessToken(uuid.New(), "Alice")
> 	if err != nil {
> 		t.Fatal(err)
> 	}
> 	return server.httpServer.Handler, token
> }
>
> func TestInvites_PassTableAndGuest(t *testing.T) {
> 	stub := &stubInvites{}
> 	router, token := newInvitesRouter(t, stub)
> 	bob := uuid.New()
>
> 	w := doUsersRequest(router, http.MethodPost, "/api/v1/tables/t1/invites", token, `{"userId":"`+bob.String()+`"}`)
> 	if w.Code != http.StatusCreated {
> 		t.Fatalf("status = %d (%s)", w.Code, w.Body.String())
> 	}
> 	if stub.gotRoom != "t1" || stub.gotGuest != bob {
> 		t.Fatalf("service got %q, %s", stub.gotRoom, stub.gotGuest)
> 	}
> 	if w := doUsersRequest(router, http.MethodPost, "/api/v1/tables/t1/invites", token, `{"userId":"nope"}`); w.Code != http.StatusBadRequest {
> 		t.Fatalf("bad user id status = %d, want 400", w.Code)
> 	}
> }
>
> func TestInvites_ErrorStatuses(t *testing.T) {
> 	cases := []struct {
> 		err  error
> 		want int
> 	}{
> 		{model.ErrTableNotFound, http.StatusNotFound},
> 		{model.ErrNotFriends, http.StatusForbidden},
> 		{model.ErrNotAtTable, http.StatusForbidden},
> 		{model.ErrTableStarted, http.StatusConflict},
> 		{model.ErrTableFull, http.StatusConflict},
> 		{model.ErrAlreadyInvited, http.StatusConflict},
> 		{model.ErrFriendBusy, http.StatusConflict},
> 		{model.ErrFriendNotInLobby, http.StatusConflict},
> 		{model.ErrFriendNoFunds, http.StatusConflict},
> 	}
> 	for _, tc := range cases {
> 		router, token := newInvitesRouter(t, &stubInvites{err: tc.err})
> 		w := doUsersRequest(router, http.MethodPost, "/api/v1/tables/t1/invites", token, `{"userId":"`+uuid.NewString()+`"}`)
> 		if w.Code != tc.want {
> 			t.Errorf("%v: status = %d, want %d", tc.err, w.Code, tc.want)
> 		}
> 	}
> }
>
> func TestInvites_Decline(t *testing.T) {
> 	stub := &stubInvites{}
> 	router, token := newInvitesRouter(t, stub)
> 	if w := doUsersRequest(router, http.MethodPost, "/api/v1/tables/t1/invites/decline", token, ""); w.Code != http.StatusNoContent || stub.gotRoom != "t1" {
> 		t.Fatalf("decline status = %d, room = %q", w.Code, stub.gotRoom)
> 	}
> 	stub.err = model.ErrInviteNotFound
> 	if w := doUsersRequest(router, http.MethodPost, "/api/v1/tables/t1/invites/decline", token, ""); w.Code != http.StatusNotFound {
> 		t.Fatalf("stale decline status = %d, want 404", w.Code)
> 	}
> }
> ```

### Готово, когда

- [ ] `go test -race ./internal/...` целиком зелёный
- [ ] пересобранный сервер стартует: `docker compose up -d --build server`

---

## Шаг 7. Проверка руками

Без фронта приглашение можно проверить так: **Боб** в браузере в лобби (ему нужен лобби-сокет), **Алиса** — через curl.

1. Боб входит в лобби в браузере и открывает DevTools → Network → WS → `lobby` → Messages.
2. Алиса логинится и сразу создаёт стол на двоих. Звать нужно в течение 15 секунд, пока держится её бронь создателя, иначе ответ будет «звать можно только за свой стол».

```bash
curl -s -c alice.jar -X POST localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d "{\"email\":\"<почта Алисы>\",\"password\":\"<пароль>\"}"
```

```bash
curl -s -b alice.jar -X POST localhost:8080/api/v1/tables -H "Content-Type: application/json" -d "{\"mode\":\"podkidnoy\",\"deck\":36,\"maxPlayers\":2,\"stake\":100}"
```

```bash
curl -s -b alice.jar -X POST localhost:8080/api/v1/tables/<id стола>/invites -H "Content-Type: application/json" -d "{\"userId\":\"<id Боба>\"}" -w "\n%{http_code}\n"
```

3. У Боба в сообщениях сокета появился `table_invite` с именем Алисы и ставкой.
4. Повтори приглашение — `409 друг уже приглашён`. Позови не друга — `403`. Позови друга, который на странице стола, — `409 друг сейчас не в лобби`.
5. Боб отказывается (его cookie из браузера проще всего взять, повторив вход через curl в `bob.jar`):

```bash
curl -s -b bob.jar -X POST localhost:8080/api/v1/tables/<id стола>/invites/decline -w "%{http_code}\n"
```

Первый раз — `204`, второй — `404`.

Посмотреть стол и брони: `GET /api/v1/tables/<id>` — поле `players` показывает занятые места вместе с бронями.

---

## Шаг 8. Что нужно фронту

Фронт делается отдельно, вот контракт:

| Где | Что делать |
|---|---|
| Лобби, кнопка «Позвать за стол» у друга в сети | открыть диалог создания стола в режиме «позвать» (можно 2 места), затем `POST /tables` → `POST /tables/{id}/invites` → перейти за стол |
| `useLobbySocket`, сообщение `table_invite` | хранить список приглашений по `tableId`, удалять по `expiresAt` |
| Карточка приглашения в лобби | «Алиса зовёт: подкидной, 36 карт, ставка 100», обратный отсчёт, кнопки «Сесть» (переход на `/table/{tableId}`) и «Нет» (`POST .../invites/decline`) |
| `useGameSocket`, сообщение `invite_declined` | системная строчка в чате стола: «Боб не принял приглашение» |
| Ошибки | текст из поля `error` ответа уже на русском, его можно показывать как есть |

---

## Шаг 9. Бонусы

### 9.1. Алиса отменяет приглашение ★☆☆

> [!example]- Набросок
> `DELETE /api/v1/tables/{id}/invites/{userId}`: проверить, что Алиса за этим столом, `room.Uninvite(userId)`, и отправить Бобу в лобби `invite_cancelled {tableId}`, чтобы карточка исчезла.

### 9.2. «Боб принял» ★★☆

> [!example]- Набросок
> В `handleRegister`, если входящий игрок был в `invites`, разослать столу `invite_accepted {username}`. Подумай, где узнать «был приглашён»: `seat` удаляет запись, значит, проверить надо до `r.seat(playerID)`.

### 9.3. Звать со страницы стола ★★☆

> [!example]- Набросок
> Эндпоинт уже готов, нужен только интерфейс в ожидании партии: список друзей в сети и кнопка «Позвать». Список даёт `GET /api/v1/friends`. Друзья видят Алису «за столом», это нормально, звать она может.

### 9.4. Один стол — несколько приглашений одному другу ★★★

> [!example]- Набросок
> Сейчас двое разных друзей могут позвать Боба за разные столы, и у него будут две карточки. Хватит ли этого? Если нет, придётся хранить «последнее приглашение» на стороне Боба и заменять старое.

---

## Вопросы комиссии

> [!question]- Как сервер держит место для приглашённого?
> В комнате два вида брони. Обычная (создатель, быстрая игра) держит место и делает игрока «занятым». Приглашение держит место 45 секунд, но не мешает другу играть в другом месте. При входе за стол сервер считает свободные места с учётом чужих броней, так что посторонний место не займёт.

> [!question]- Что если двое одновременно займут последнее место?
> Все проверки и бронь приглашения делаются под одной блокировкой хаба, а посадка — в горутине комнаты, по одной регистрации за раз. Двойной посадки не бывает.

> [!question]- Почему приглашение шлёт данные, а «друзья изменились» — нет?
> Друзья лежат в базе, их можно перезапросить, поэтому достаточно сигнала. Приглашение в базе не хранится, это разовое событие, поэтому все данные идут прямо в сообщении.

> [!question]- Почему приглашения не хранятся в базе?
> Они живут 45 секунд и теряют смысл после перезапуска сервера: стола, за который звали, тоже уже нет. Писать их в Postgres — лишняя работа без пользы.

> [!question]- Что если друг закрыл вкладку?
> Приглашать можно только тех, кто сейчас в лобби. Если друг ушёл после приглашения, место просто освободится через 45 секунд.

## Шпаргалка

| Что | Где |
|---|---|
| Ошибки и типы | `internal/model/invite.go` |
| Место под приглашение, объявление | `ws/room.go`: `invites`, `freeSeatsFor`, `Announce` |
| Проверки стола, доставка | `ws/hub.go`: `Invite`, `Decline`, `SendInvite` |
| Правила «друг, в лобби, фишки» | `service/invite.go` |
| API | `delivery/http/invite_handler.go`, маршруты в `server.go` |

```bash
go test -race ./internal/...
```

```bash
go test -race ./internal/delivery/ws/ -run "Invite|Decline|Stranger" -count=10
```

## Словарь

| Термин | Коротко |
|---|---|
| Бронь (reservation) | место держится за игроком, и игрок считается занятым этим столом |
| Приглашение (invite) | место держится за игроком, но игрок свободен |
| Ленивое истечение | срок проверяется при чтении, без таймеров |
| Объявление в комнату | сообщение всем за столом через канал горутины комнаты |
| Данные, а не сигнал | сообщение несёт всё нужное, потому что перезапросить неоткуда |
| 403 / 404 / 409 | нельзя в принципе / уже нет / не сейчас |

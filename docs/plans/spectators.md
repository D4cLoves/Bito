---
title: Bito — зрители за столом (учебный план)
tags: [bito, plan, spectators, go, websocket, learning]
created: 2026-10-21
---

# Зрители: учебный план

> [!abstract] Что строим
> Ева открывает идущую партию и **смотрит**: видит стол, карты в бою, сколько карт у каждого, козырь, таймер, чат и итог. Место она не занимает и ходить не может, а **чужих карт на руках не видит**. Игроки видят «👁 3» — сколько человек смотрит. В лобби появляется список «Идут сейчас»; закрытый стол можно смотреть только по ссылке с ключом.
> Связанные заметки: [[lobby-socket]] (горутина-владелец), [[invites]] (брони и `announce`), [[features]].

> [!info] Как работать с этой заметкой
> Каждый шаг: **зачем → теория → практика → ловушки → проверь себя → решение → готово, когда**. Решения свёрнуты: сначала подумай и напиши своё.
> Здесь только **бэкенд**. Фронт (режим зрителя на странице стола, список «Идут сейчас», счётчик зрителей) делаю я, а что ему нужно от сервера, описано в шаге 7.

> [!success] Что проверено
> Решения взяты из эталона, который собирался и проходил `go vet` и `go test -race -count=5 ./internal/delivery/...`. Код вставлен в заметку скриптом, руками его не переписывали.

---

## 0. Карта

### Сценарий

```
 Ева (лобби)                         Go-сервер                              Алиса и Боб (играют)
┌──────────────────┐ GET /tables/live ┌──────────────────┐
│ «Идут сейчас»    │ ───────────────► │ Hub.ListLiveRooms│  начатые, не закончены, не закрытые
└──────────────────┘                  └──────────────────┘
        │ «Смотреть»
        ▼
 /ws?room_id=…&watch=1[&key=…] ──► ServeWS ── Holds(Ева)? нет ──► Hub.watch
                                              │                       │ MayEnter (ключ закрытого стола)
                                              │ да: обычный вход       ▼
                                              ▼               Room.addWatcher (в горутине комнаты)
                                         admit / seat          ├─ game_state без руки
                                                               ├─ game_over, если уже кончилась
                                                               └─ watchers {count} ──► всем ──► «👁 1»
```

### Что видит зритель, а что нет

| | Игрок | Зритель |
|---|---|---|
| Стол (атака и защита), козырь, колода, отбой | да | да |
| Сколько карт у каждого | да | да |
| **Карты на руке** | свои | **ничьи** |
| Таймер хода, чат, итог, «потерял связь» | да | да |
| Ходить, писать в чат, голосовать за реванш | да | нет |
| Занимает место, ставка, «один стол на аккаунт» | да | нет |

> [!tip] Почему не нужна задержка трансляции
> На турнирах по покеру трансляцию задерживают, потому что зритель видит **закрытые** карты и может подсказать игроку. У нас зритель видит только то, что и так видят все игроки за столом. Подсказать ему нечего, значит задержка не нужна. Если когда-нибудь появится режим «смотреть с картами игрока», задержка станет обязательной (бонус 9.2).

### API

| Что | Где | Изменение |
|---|---|---|
| Смотреть стол | `GET /ws?room_id=…&watch=1` (+ `&key=…` для закрытого) | новый параметр `watch` |
| Идущие партии | `GET /api/v1/tables/live` | новый эндпоинт, массив `RoomInfo` |
| Сколько смотрят | `RoomInfo.watchers` | новое поле |

### Сообщения сокета

| Тип | Кому | Payload |
|---|---|---|
| `watchers` | всем за столом и зрителям | `{"count": 3}` |
| `error` | зрителю, который пытается ходить или писать | `{"message": "зрители не ходят и не пишут в чат"}` |
| `game_state`, `game_over`, `chat`, `player_disconnected`, `player_reconnected`, `rematch_*` | теперь и зрителям | как раньше |

### Шаги

1. Клиент-зритель — флаг в `Client`.
2. Комната: отдельная карта `watchers` и рассылка «всем».
3. Вход и выход зрителя, лимит, счётчик.
4. Хаб: `watch=1` в `ServeWS`, список идущих партий.
5. HTTP: `GET /tables/live`.
6. Тесты.
7. Что нужно фронту.
8. Проверка руками.
9. Бонусы.

---

## Шаг 1. Клиент-зритель

### Зачем

Комнате надо отличать зрителя от игрока прямо в момент регистрации, ещё до того, как она решит, куда его положить. Самое простое место для такой метки — сам `Client`: он создаётся в `ServeWS`, где мы уже знаем, что пришёл `watch=1`.

### Теория

> [!note] Метка неизменяемая
> Флаг `watch` ставится один раз в конструкторе и больше не меняется. Поэтому его можно читать из любой горутины без мьютекса: гонки бывают только там, где кто-то пишет одновременно с чтением.

> [!note] Почему не отдельный тип `Watcher`
> Можно было бы сделать новый тип, но `ReadPump`/`WritePump`, отправка, закрытие у зрителя точно такие же. Один тип с флагом — меньше кода, и `endpoint` (`deliver`/`leave`) работает как раньше.

### Практика

В `internal/delivery/ws/client.go`:

1. Поле `watch bool` в `Client`.
2. Конструктор `NewWatcher(id, name, conn, ep) *Client` — вызывает `NewClient` и ставит флаг.
3. Метод `Watching() bool`.

> [!example]- Решение: client.go
> ```go
> type Client struct {
> 	id     string
> 	name   string
> 	conn   *websocket.Conn
> 	owner  endpoint
> 	watch  bool
> 	send   chan []byte
> 	mu     sync.Mutex
> 	closed bool
> }
> ```
> ```go
> func NewWatcher(id, name string, conn *websocket.Conn, ep endpoint) *Client {
> 	c := NewClient(id, name, conn, ep)
> 	c.watch = true
> 	return c
> }
>
> func (c *Client) Watching() bool {
> 	return c.watch
> }
> ```

### Готово, когда

- [ ] `go build ./...` проходит
- [ ] можешь объяснить, почему `Watching()` не нужен мьютекс

---

## Шаг 2. Комната: отдельная карта зрителей

### Зачем

Главное решение всей фичи: **где хранить зрителей**. Соблазн — положить их в ту же `clients` с флагом. Посмотри, сколько мест в `room.go` считает `len(r.clients)` или ходит по `r.clients`:

- старт партии: `len(r.clients) == r.maxUsers`;
- свободные места: `r.maxUsers - len(r.clients) - …`;
- `startGame` собирает игроков из `r.clients`;
- реванш голосует по `r.clients`;
- `takeOver` закрывает «вторую вкладку» по id.

Каждое из этих мест сломалось бы от зрителя: партия стартовала бы со зрителем вместо игрока, зритель занимал бы место, голосовал бы за реванш. Пришлось бы везде дописывать `if c.Watching()`, и одно забытое место — баг.

### Теория

> [!note] Отдельная карта — инвариант бесплатно
> `clients` — **только игроки**, `watchers` — **только зрители**. Тогда весь старый код про места, старт и реванш остаётся правильным без единой правки. Меняются лишь места, где мы **рассылаем** сообщения: им нужно «всем», то есть обе карты.

> [!note] Один помощник для рассылки
> Заведи `broadcast(data []byte)`: обходит `clients`, потом `watchers`. И переведи на него уже существующий `sendAll` из `rematch.go`. Дальше везде, где был цикл `for _, c := range r.clients { c.Send(data) }`, пиши `r.broadcast(data)` или `r.sendAll(тип, payload)`.

> [!note] Кто владеет картой
> `watchers`, как и `clients`, читает и пишет **только горутина комнаты** (`Run` и то, что он вызывает). Хабу и HTTP нужно лишь число зрителей для `RoomInfo` — для этого счётчик `watcherCount atomic.Int32`, точно как `playersCount`.

> [!note] Состояние партии для зрителя уже готово
> Посмотри на `sendGameStateTo`: рука кладётся в `MyHand`, только если `p.ID() == client.ID()`. У зрителя id не совпадает ни с одним игроком этой партии (шаг 4 гарантирует, что игрок этой партии всегда входит как игрок), значит `MyHand` будет пустой. Отдельный «вид для зрителя» писать не надо.

### Практика

В `room.go`:

1. Поле `watchers map[string]*Client`, инициализация в `NewRoom`; поле `watcherCount atomic.Int32`.
2. Поле `Watchers int \`json:"watchers"\`` в `RoomInfo`, заполнить в `Info()` через метод `Watchers()` (его напишешь в шаге 3).
3. `broadcast(data)` — новый помощник; `sendAll` в `rematch.go` вызывает его.
4. Перевести на рассылку всем: ветку `announce` в `Run`, `handleChat`, `broadcastGameOver`, уведомления `player_disconnected` и `player_reconnected`.
5. `broadcastGameState` — второй цикл по `watchers` с `sendGameStateTo`.
6. `cleanup` — закрыть и зрителей.
7. Вынести сборку `GameOverPayload` в `gameOverPayload()`: она понадобится, чтобы показать итог опоздавшему зрителю.

Ошибки `errNoFunds`, `errStakesRefunded`, `errStartFailed` **не** трогай: они про деньги игроков, зрителю они ни к чему.

> [!warning] Ловушки
> - **Порядок в переподключении.** Раньше `player_reconnected` слали всем, кроме вернувшегося (`if id != playerID`). Если просто заменить на `sendAll` **после** `takeOver`, вернувшийся получит уведомление о себе первым, ещё до `game_state` — тест `TestRoom_GracePeriodReconnect` упадёт. Подсказка: подумай, есть ли вернувшийся игрок в `clients` до `takeOver`.
> - Не забудь `cleanup`: иначе после закрытия стола у зрителей останутся висеть открытые сокеты.
> - `sendAll` уже есть в `rematch.go`. Не заводи второй с тем же смыслом.

### Проверь себя

> [!question]- Почему отдельная карта лучше, чем флаг в общей карте?
> Потому что весь код про места, старт, реванш и вторую вкладку опирается на «в `clients` только игроки». С отдельной картой этот инвариант сохраняется без правок. С флагом пришлось бы помнить про зрителя в каждом таком месте, а забытое место дало бы тихий баг: зритель занял бы место или проголосовал.

> [!question]- Как отправить «вернулся» всем, кроме вернувшегося, без `if id != playerID`?
> Отправить **до** `takeOver`. Пока соединение не вернули в `clients`, вернувшегося там нет (его удалили в `handleUnregister`), и `sendAll` до него не дойдёт.

> [!question]- Почему зрителю нельзя просто отдавать то же `game_state`, что и игроку Алисе?
> В `game_state` Алисы лежит её рука. Состояние строится **на каждого получателя отдельно**: сервер решает, что можно знать этому человеку. Это главное правило сетевых карточных игр: клиенту никогда не отправляют то, чего ему видеть нельзя, даже если интерфейс это «спрячет».

> [!example]- Решение: изменения в room.go
> Поля и `RoomInfo`:
> ```go
> 	Private     bool      `json:"private"`
> 	Watchers    int       `json:"watchers"`
> ```
> ```go
> 	clients           map[string]*Client
> 	watchers          map[string]*Client
> ```
> ```go
> 	playersCount      atomic.Int32
> 	watcherCount      atomic.Int32
> ```
> В `NewRoom` и `Info()`:
> ```go
> 		watchers:          make(map[string]*Client),
> ```
> ```go
> 		Watchers:    r.Watchers(),
> ```
> В `Run`:
> ```go
> 		case data := <-r.announce:
> 			r.broadcast(data)
> ```
> В `handleRegister`, ветка переподключения во время партии:
> ```go
> 			r.sendAll(TypePlayerReconnected, PlayerStatusPayload{PlayerID: playerID})
> 			r.takeOver(playerID, client)
>
> 			r.sendGameStateTo(client)
> 			return
> ```
> В `handleUnregister` вместо цикла с `notifyMsg`:
> ```go
> 	r.sendAll(TypePlayerDisconnected, PlayerStatusPayload{PlayerID: playerID})
> ```
> Рассылка состояния и итога:
> ```go
> func (r *Room) broadcastGameState() {
> 	for _, client := range r.clients {
> 		r.sendGameStateTo(client)
> 	}
> 	for _, client := range r.watchers {
> 		r.sendGameStateTo(client)
> 	}
> }
> ```
> ```go
> func (r *Room) broadcastGameOver() {
> 	r.sendAll(TypeGameOver, r.gameOverPayload())
> }
>
> func (r *Room) gameOverPayload() GameOverPayload {
> 	places := make([]string, 0, len(r.game.Places()))
> 	for _, id := range r.game.Places() {
> 		places = append(places, string(id))
> 	}
> 	return GameOverPayload{
> 		WinnerID: string(r.game.WinnerID()),
> 		LoserID:  string(r.game.LoserID()),
> 		Places:   places,
> 		Payouts:  r.payouts,
> 		Stake:    r.stake,
> 		IsDraw:   r.draw,
> 	}
> }
> ```
>
> Конец `handleChat` и `cleanup`:
> ```go
> 	data, err := json.Marshal(chatMsg)
> 	if err != nil {
> 		return
> 	}
> 	r.broadcast(data)
> }
> ```
> ```go
> 	for _, c := range r.clients {
> 		c.Close()
> 	}
> 	for _, c := range r.watchers {
> 		c.Close()
> 	}
> ```
> `sendAll` в `rematch.go`:
> ```go
> func (r *Room) sendAll(msgType string, payload any) {
> 	data, err := NewMessage(msgType, payload)
> 	if err != nil {
> 		return
> 	}
> 	r.broadcast(data)
> }
> ```


### Готово, когда

- [ ] старые тесты зелёные без правок (`go test -race ./internal/delivery/ws/`)
- [ ] в `room.go` не осталось циклов `range r.clients` с `c.Send`, кроме денежных ошибок игрокам

---

## Шаг 3. Вход и выход зрителя

### Зачем

Зрителя надо положить в карту, сразу показать ему стол (он мог прийти посреди партии или после неё), сообщить всем новое число зрителей — и всё это в горутине комнаты, как и регистрацию игрока.

### Теория

> [!note] Развилка в самом начале
> `handleRegister`, `handleUnregister` и `handleIncoming` первым делом смотрят `client.Watching()` и уводят зрителя в свои функции. Так старая логика игроков вообще не видит зрителей.

> [!note] Опоздавший зритель
> Партия могла начаться час назад. Новому зрителю сразу шлём `game_state` (если `r.game != nil`), а если партия уже закончилась — ещё и `game_over`. Иначе он увидит пустой стол до следующего хода.

> [!note] Лимит
> Каждый зритель — это горутины `ReadPump`/`WritePump` и копия каждого сообщения. Без лимита один стол можно завалить тысячей соединений. Константа `maxWatchers = 30` и ошибка `errTooManyWatchers`.

> [!note] Вторая вкладка зрителя
> Как у игроков: если Ева открыла стол во второй вкладке, старое соединение получает `errOtherTab` и закрывается, новое занимает её место. Лимит при этом не проверяем — число зрителей не растёт.

> [!note] Чужое «отключение»
> `removeWatcher` проверяет `r.watchers[c.ID()] == c`. Когда вторая вкладка вытеснила первую, первая всё равно придёт в `unregister` (её `ReadPump` закроется). Без проверки она удалила бы из карты **новую** вкладку. Такая же защита уже стоит в `handleUnregister` у игроков.

> [!note] Зритель шлёт сообщения
> `handleIncoming` у игроков начинается с `if r.clients[id] != msg.Client { return }` — зритель бы тихо проигнорировался. Лучше честно ответить ошибкой `errWatcherAction`, чтобы фронт показал понятный тост, а не «ничего не произошло».

### Практика

1. В `bank.go` рядом с остальными текстами ошибок: `errTooManyWatchers = "у стола уже слишком много зрителей"`, `errWatcherAction = "зрители не ходят и не пишут в чат"`.
2. В `message.go`: `TypeWatchers = "watchers"`.
3. Новый файл `internal/delivery/ws/spectate.go`: `maxWatchers`, `WatchersPayload`, `Watchers()`, `addWatcher`, `removeWatcher`, `broadcast`.
4. Развилки в начале `handleRegister`, `handleUnregister`, `handleIncoming`.

> [!warning] Ловушки
> - `watcherCount.Store(…)` после **каждого** изменения карты, иначе `RoomInfo.watchers` соврёт.
> - Уведомление `watchers` отправляй **после** того, как положил зрителя в карту — тогда и он сам получит актуальное число.
> - Стол закрывается, когда ушёл последний **игрок** (`len(r.clients) == 0`). Зрители стол не держат — так и задумано; `cleanup` их закроет.

### Проверь себя

> [!question]- Почему `addWatcher` вызывается из горутины комнаты, а не прямо из `ServeWS`?
> Карта `watchers` принадлежит горутине комнаты. Если писать в неё из горутины HTTP-запроса, будет гонка с `Run`, который в это время по ней рассылает. Регистрация через канал `register` даёт порядок и безопасность без мьютекса.

> [!question]- Что сломается без проверки `r.watchers[c.ID()] == c` в `removeWatcher`?
> Ева открыла вторую вкладку → первая закрыта и потом приходит в `unregister`. Без проверки она удалит из карты вторую, живую вкладку: Ева перестанет получать ходы, а счётчик покажет 0.

> [!question]- Зачем отдельно слать `game_over` опоздавшему, если `game_state` уже пришёл?
> `game_state` показывает только расклад. Кто выиграл, места и выплаты есть только в `game_over`. Без него зритель, зашедший после конца, увидит застывший стол без итога.

> [!example]- Решение: развилки в room.go
> ```go
> func (r *Room) handleRegister(client *Client) {
> 	if client.Watching() {
> 		r.addWatcher(client)
> 		return
> 	}
> 	playerID := client.ID()
> ```
> ```go
> func (r *Room) handleUnregister(client *Client) {
> 	if client.Watching() {
> 		r.removeWatcher(client)
> 		return
> 	}
> 	playerID := client.ID()
> ```
> ```go
> func (r *Room) handleIncoming(msg IncomingMessage) {
> 	if msg.Client.Watching() {
> 		if r.watchers[msg.Client.ID()] == msg.Client {
> 			r.sendError(msg.Client, errWatcherAction)
> 		}
> 		return
> 	}
> 	if r.clients[msg.Client.ID()] != msg.Client {
> 		return
> 	}
> ```
> Ошибки и тип сообщения:
> ```go
> 	errPrivateTable    = "стол закрытый: нужно приглашение"
> 	errTooManyWatchers = "у стола уже слишком много зрителей"
> 	errWatcherAction   = "зрители не ходят и не пишут в чат"
> ```
> ```go
> 	TypeRematchStart     = "rematch_start"
> 	TypeWatchers         = "watchers"
> ```

> [!example]- Решение: spectate.go целиком (с хабом из шага 4)
> ```go
> package ws
>
> import (
> 	"sort"
>
> 	"github.com/gorilla/websocket"
> )
>
> const maxWatchers = 30
>
> type WatchersPayload struct {
> 	Count int `json:"count"`
> }
>
> func (r *Room) Watchers() int {
> 	return int(r.watcherCount.Load())
> }
>
> func (r *Room) addWatcher(c *Client) {
> 	old, again := r.watchers[c.ID()]
> 	if !again && len(r.watchers) >= maxWatchers {
> 		r.sendError(c, errTooManyWatchers)
> 		c.Close()
> 		return
> 	}
> 	if again {
> 		r.sendError(old, errOtherTab)
> 		old.Close()
> 	}
>
> 	r.watchers[c.ID()] = c
> 	r.watcherCount.Store(int32(len(r.watchers)))
>
> 	r.sendGameStateTo(c)
> 	if r.game != nil && r.finished.Load() {
> 		if data, err := NewMessage(TypeGameOver, r.gameOverPayload()); err == nil {
> 			c.Send(data)
> 		}
> 	}
> 	r.sendAll(TypeWatchers, WatchersPayload{Count: len(r.watchers)})
> }
>
> func (r *Room) removeWatcher(c *Client) {
> 	if r.watchers[c.ID()] != c {
> 		return
> 	}
> 	delete(r.watchers, c.ID())
> 	c.Close()
> 	r.watcherCount.Store(int32(len(r.watchers)))
> 	r.sendAll(TypeWatchers, WatchersPayload{Count: len(r.watchers)})
> }
>
> func (r *Room) broadcast(data []byte) {
> 	for _, c := range r.clients {
> 		c.Send(data)
> 	}
> 	for _, c := range r.watchers {
> 		c.Send(data)
> 	}
> }
>
> func (h *Hub) watch(conn *websocket.Conn, room *Room, playerID, playerName, key string) {
> 	if !room.MayEnter(playerID, key) {
> 		rejectConn(conn, errPrivateTable)
> 		return
> 	}
> 	client := NewWatcher(playerID, playerName, conn, room)
> 	if !room.RegisterClient(client) {
> 		client.Close()
> 		_ = conn.Close()
> 		return
> 	}
> 	go client.WritePump()
> 	go client.ReadPump()
> }
>
> func (h *Hub) ListLiveRooms() []RoomInfo {
> 	h.mu.RLock()
> 	rooms := make([]RoomInfo, 0)
> 	for _, room := range h.rooms {
> 		if room.Started() && !room.finished.Load() && !room.private {
> 			rooms = append(rooms, room.Info())
> 		}
> 	}
> 	h.mu.RUnlock()
>
> 	sort.Slice(rooms, func(i, j int) bool {
> 		return rooms[i].Watchers > rooms[j].Watchers || rooms[i].Watchers == rooms[j].Watchers && rooms[i].CreatedAt.After(rooms[j].CreatedAt)
> 	})
> 	return rooms
> }
> ```

### Готово, когда

- [ ] зритель, пришедший посреди партии, сразу получает `game_state` без руки
- [ ] после ухода зрителя игроки получают `watchers` с меньшим числом

---

## Шаг 4. Хаб: вход по `watch=1` и список идущих партий

### Зачем

`ServeWS` сейчас каждого пропускает через `admit`, а `admit` **занимает место** (`claimSeat`) и проверяет баланс. Зрителю не нужно ни то ни другое: смотреть можно без фишек и даже сидя за другим столом. Но закрытый стол должен оставаться закрытым.

### Теория

> [!note] Порядок проверок в ServeWS
> 1. Пришёл `watch=1` **и** человек не сидит за этим столом (`!room.Holds(id)`) → путь зрителя.
> 2. Иначе — обычный вход игрока, как раньше.
>
> Второе условие важно: Алиса, которая играет за этим столом, открыла ссылку «смотреть» (или фронт ошибся). Если пустить её зрителем, у неё будет **два** соединения — игрока и зрителя, а её карты зритель-вкладка не покажет. Игрок за своим столом всегда входит как игрок.

> [!note] Закрытый стол
> Для зрителя нужна только проверка `room.MayEnter(id, key)` из приватных столов: участник или правильный ключ. Никакого `claimSeat`, никакой проверки баланса.

> [!note] Список идущих партий
> `ListLiveRooms`: стол начат (`Started`), не закончен (`finished`), не закрытый. Дуэли из быстрой игры (`matchmaking`) тоже показываем — это как раз самые частые партии. Сортировка: сначала где больше зрителей, потом новые.
> Пустой список должен уходить в JSON как `[]`, а не `null`: создавай срез через `make(…, 0)`.

### Практика

1. В `spectate.go`: метод хаба `watch(conn, room, id, name, key)` — проверка `MayEnter`, `NewWatcher`, `RegisterClient`, запуск помп.
2. В `ServeWS` после `Upgrade`, **до** `admit`: развилка на `watch`.
3. `ListLiveRooms() []RoomInfo` в `spectate.go`.

> [!warning] Ловушки
> - Развилку ставь **до** `admit`. Если после — `admit` успеет занять зрителю место и проверить баланс, и Ева без фишек не сможет смотреть.
> - `finished` и `private` — поля комнаты; хаб читает их под `h.mu.RLock`, а сами они `atomic` или неизменяемые, так что гонки нет. Не читай из хаба `r.clients` или `r.watchers` — только `Watchers()`.
> - Не возвращай `nil`-срез из `ListLiveRooms`, иначе фронт получит `null` и упадёт на `.length`.

### Проверь себя

> [!question]- Почему проверка ключа у зрителя в хабе, а лимит зрителей — в комнате?
> Ключ — неизменяемое поле комнаты, его безопасно проверить где угодно, и лучше раньше: чужой даже не дойдёт до комнаты. Лимит зависит от карты `watchers`, а её знает только горутина комнаты; проверка в хабе была бы гонкой «проверил 29 → двое одновременно добавились → 31».

> [!question]- Можно ли смотреть стол, сидя за другим столом?
> Да. Правило «один аккаунт — один стол» про места и ставки. Зритель места не занимает, поэтому `claimSeat` не вызываем.

> [!example]- Решение: ServeWS
> ```go
> 	conn, err := h.upgrader.Upgrade(w, r, nil)
> 	if err != nil {
> 		log.Printf("Upgrade error: %v", err)
> 		return
> 	}
>
> 	if query.Get("watch") == "1" && !room.Holds(playerID) {
> 		h.watch(conn, room, playerID, playerName, query.Get("key"))
> 		return
> 	}
>
> 	if reason := h.admit(room, playerID, query.Get("key")); reason != "" {
> ```
> Сами `watch` и `ListLiveRooms` — в решении шага 3 (файл `spectate.go`).

### Готово, когда

- [ ] без ключа закрытый стол отвечает зрителю `стол закрытый: нужно приглашение`
- [ ] зритель без фишек может смотреть стол со ставкой

---

## Шаг 5. HTTP: `GET /api/v1/tables/live`

### Зачем

Лобби нужен список того, что можно посмотреть. Это обычный эндпоинт рядом с `List`.

### Практика

1. Метод `Live` в `table_handler.go` — `respondJSON(w, 200, th.hub.ListLiveRooms())`.
2. Маршрут `r.Get("/live", s.tableHandler.Live)` в группе `/api/v1/tables` в `server.go`.

> [!question]- Не перехватит ли `/{id}` запрос `/live`?
> Нет: в chi статический сегмент побеждает параметр, `/live` выберется раньше `/{id}`. Но это стоит проверить тестом — он ниже.

> [!example]- Решение
> ```go
> func (th *TableHandler) Live(w http.ResponseWriter, r *http.Request) {
> 	respondJSON(w, http.StatusOK, th.hub.ListLiveRooms())
> }
> ```
> ```go
> 			r.Get("/", s.tableHandler.List)
> 			r.Get("/live", s.tableHandler.Live)
> 			r.Get("/{id}", s.tableHandler.Get)
> ```

---

## Шаг 6. Тесты

### Теория

> [!note] Что именно доказываем
> | Тест | Что ловит |
> |---|---|
> | зритель видит стол, но не руку | утечку карт — самый опасный баг фичи |
> | зритель не ходит и не пишет | что ход зрителя не меняет партию и не доходит до игроков |
> | счётчик растёт и падает, место не занято | `watcherCount` и `Occupancy` |
> | лимит | защиту от завала соединениями |
> | вторая вкладка | вытеснение и проверку `== c` |
> | опоздавший видит итог | `game_over` после конца |
> | зрители не голосуют за реванш | что отдельная карта не сломала реванш |
> | закрытие стола закрывает зрителей | `cleanup` |
> | список идущих | начатые, публичные |

> [!tip] Помощники уже есть
> `createTestClient`, `send`, `drain`, `waitFor`, `lastOfType`, `finishedDuel`, `fakeStarter`, `richBank`, `isClosed` — всё в существующих тестах пакета. Зрителя делаем так же, только с `c.watch = true`.
> Помни: `lastOfType` **вычитывает** весь буфер клиента. Если ждёшь два разных сообщения — проверяй их в одном `waitFor` или по очереди с пониманием, что первое съест второе.

### Практика

Создай `internal/delivery/ws/spectate_test.go` и добавь HTTP-тест в `table_handler_test.go`. Попробуй сначала написать сам по таблице выше.

> [!example]- Решение: spectate_test.go
> ```go
> package ws
>
> import (
> 	"encoding/json"
> 	"testing"
> )
>
> func watcherOf(id string, room *Room) *Client {
> 	c := createTestClient(id, "Watcher "+id, room)
> 	c.watch = true
> 	return c
> }
>
> func TestWatch_SeesTheTableButNoHands(t *testing.T) {
> 	room := NewRoom("room-watch", testSettings(2), nil)
> 	go room.Run()
> 	defer room.Close()
>
> 	eve := watcherOf("w1", room)
> 	room.RegisterClient(eve)
> 	room.RegisterClient(createTestClient("p1", "Alice", room))
> 	room.RegisterClient(createTestClient("p2", "Bob", room))
> 	waitFor(t, func() bool { return room.Started() })
>
> 	var st Message[GameStatePayload]
> 	waitFor(t, func() bool {
> 		got, ok := lastOfType[GameStatePayload](t, eve, TypeGameState)
> 		st = got
> 		return ok
> 	})
> 	if len(st.Payload.MyHand) != 0 {
> 		t.Fatalf("watcher got a hand: %v", st.Payload.MyHand)
> 	}
> 	if len(st.Payload.Players) != 2 || st.Payload.Players[0].HandCount == 0 {
> 		t.Fatalf("watcher does not see the players: %+v", st.Payload.Players)
> 	}
> 	if room.PlayersCount() != 2 || room.Watchers() != 1 {
> 		t.Fatalf("players = %d, watchers = %d", room.PlayersCount(), room.Watchers())
> 	}
> }
>
> func TestWatch_CannotMoveOrChat(t *testing.T) {
> 	room := NewRoom("room-watch-act", testSettings(2), nil)
> 	go room.Run()
> 	defer room.Close()
>
> 	alice := createTestClient("p1", "Alice", room)
> 	eve := watcherOf("w1", room)
> 	room.RegisterClient(alice)
> 	room.RegisterClient(createTestClient("p2", "Bob", room))
> 	room.RegisterClient(eve)
> 	waitFor(t, func() bool { return room.Watchers() == 1 })
> 	drain(alice)
>
> 	send(room, eve, TypeTake)
> 	chat, _ := json.Marshal(map[string]any{"type": TypeChat, "payload": map[string]string{"message": "бей пиковой"}})
> 	room.incoming <- IncomingMessage{Client: eve, Data: chat}
>
> 	waitFor(t, func() bool {
> 		e, ok := lastOfType[ErrorPayload](t, eve, TypeError)
> 		return ok && e.Payload.Message == errWatcherAction
> 	})
> 	if _, ok := lastOfType[ChatPayload](t, alice, TypeChat); ok {
> 		t.Fatal("a watcher's message reached the players")
> 	}
> }
>
> func TestWatch_CountFollowsComingAndGoing(t *testing.T) {
> 	room := NewRoom("room-watch-count", testSettings(3), nil)
> 	go room.Run()
> 	defer room.Close()
>
> 	alice := createTestClient("p1", "Alice", room)
> 	room.RegisterClient(alice)
> 	eve := watcherOf("w1", room)
> 	room.RegisterClient(eve)
> 	waitFor(t, func() bool {
> 		w, ok := lastOfType[WatchersPayload](t, alice, TypeWatchers)
> 		return ok && w.Payload.Count == 1
> 	})
> 	if room.Info().Watchers != 1 {
> 		t.Fatalf("info watchers = %d", room.Info().Watchers)
> 	}
>
> 	room.unregister <- eve
> 	waitFor(t, func() bool {
> 		w, ok := lastOfType[WatchersPayload](t, alice, TypeWatchers)
> 		return ok && w.Payload.Count == 0
> 	})
> 	if room.Occupancy() != 1 {
> 		t.Fatalf("occupancy = %d, a watcher took a seat", room.Occupancy())
> 	}
> }
>
> func TestWatch_Limit(t *testing.T) {
> 	room := NewRoom("room-watch-limit", testSettings(2), nil)
> 	go room.Run()
> 	defer room.Close()
>
> 	for i := range maxWatchers {
> 		room.RegisterClient(watcherOf(string(rune('a'+i)), room))
> 	}
> 	late := watcherOf("late", room)
> 	room.RegisterClient(late)
> 	waitFor(t, func() bool {
> 		e, ok := lastOfType[ErrorPayload](t, late, TypeError)
> 		return ok && e.Payload.Message == errTooManyWatchers
> 	})
> 	if room.Watchers() != maxWatchers {
> 		t.Fatalf("watchers = %d", room.Watchers())
> 	}
> }
>
> func TestWatch_SecondTabReplacesTheFirst(t *testing.T) {
> 	room := NewRoom("room-watch-tabs", testSettings(2), nil)
> 	go room.Run()
> 	defer room.Close()
>
> 	first := watcherOf("w1", room)
> 	second := watcherOf("w1", room)
> 	room.RegisterClient(first)
> 	room.RegisterClient(second)
> 	waitFor(t, func() bool { return isClosed(first) })
> 	if room.Watchers() != 1 {
> 		t.Fatalf("watchers = %d", room.Watchers())
> 	}
> }
>
> func TestWatch_LateWatcherSeesTheResult(t *testing.T) {
> 	starter := &fakeStarter{}
> 	room, _, _ := finishedDuel(t, richBank(), starter)
>
> 	eve := watcherOf("w1", room)
> 	room.RegisterClient(eve)
> 	waitFor(t, func() bool {
> 		_, ok := lastOfType[GameOverPayload](t, eve, TypeGameOver)
> 		return ok
> 	})
> }
>
> func TestWatch_WatchersDoNotVoteForRematch(t *testing.T) {
> 	starter := &fakeStarter{}
> 	room, alice, bob := finishedDuel(t, richBank(), starter)
> 	room.RegisterClient(watcherOf("w1", room))
> 	waitFor(t, func() bool { return room.Watchers() == 1 })
>
> 	send(room, alice, TypeRematch)
> 	send(room, bob, TypeRematch)
> 	waitFor(t, func() bool { return starter.count() == 1 })
> 	if got := starter.calls[0].players; len(got) != 2 {
> 		t.Fatalf("rematch players = %v", got)
> 	}
> }
>
> func TestWatch_ClosingTheTableClosesWatchers(t *testing.T) {
> 	room := NewRoom("room-watch-close", testSettings(2), nil)
> 	go room.Run()
>
> 	alice := createTestClient("p1", "Alice", room)
> 	eve := watcherOf("w1", room)
> 	room.RegisterClient(alice)
> 	room.RegisterClient(eve)
> 	waitFor(t, func() bool { return room.Watchers() == 1 })
>
> 	room.unregister <- alice
> 	waitFor(t, func() bool { return isClosed(eve) })
> }
>
> func TestHub_LiveTablesAreStartedAndPublic(t *testing.T) {
> 	h := NewHub(nil, nil, nil)
> 	start := func(private bool, players ...string) *Room {
> 		room, err := h.CreateTable(RoomSettings{Mode: defaultDuelMode, Deck: defaultDuelDeck, MaxPlayers: 2, CreatedBy: players[0], Private: private})
> 		if err != nil {
> 			t.Fatal(err)
> 		}
> 		t.Cleanup(room.Close)
> 		for _, id := range players {
> 			room.RegisterClient(createTestClient(id, id, room))
> 		}
> 		return room
> 	}
>
> 	live := start(false, "a", "b")
> 	hidden := start(true, "c", "d")
> 	waiting := start(false, "e")
> 	waitFor(t, func() bool { return live.Started() && hidden.Started() })
>
> 	list := h.ListLiveRooms()
> 	if len(list) != 1 || list[0].ID != live.ID() {
> 		t.Fatalf("live = %+v, want only %s (not private %s, not waiting %s)", list, live.ID(), hidden.ID(), waiting.ID())
> 	}
> }
> ```

> [!example]- Решение: HTTP-тест
> ```go
> func TestTables_LiveListsOnlyStartedGames(t *testing.T) {
> 	router, token, _ := newTestTablesRouter(t)
> 	if w := doTablesRequest(router, http.MethodPost, token, `{"mode":"podkidnoy","deck":36,"maxPlayers":3}`); w.Code != http.StatusCreated {
> 		t.Fatalf("create status = %d", w.Code)
> 	}
>
> 	req := httptest.NewRequest(http.MethodGet, "/api/v1/tables/live", nil)
> 	req.Header.Set("Authorization", "Bearer "+token)
> 	w := httptest.NewRecorder()
> 	router.ServeHTTP(w, req)
>
> 	if w.Code != http.StatusOK {
> 		t.Fatalf("status = %d, want 200", w.Code)
> 	}
> 	if body := strings.TrimSpace(w.Body.String()); body != "[]" {
> 		t.Fatalf("live = %s, want [] while nobody plays", body)
> 	}
> }
> ```

### Готово, когда

- [ ] `go test -race -count=5 ./internal/delivery/...` зелёный
- [ ] `go vet ./...` чистый

---

## Шаг 7. Что нужно фронту

Это делаю я, тебе только знать контракт:

- `GET /api/v1/tables/live` → `RoomInfo[]` с полем `watchers`.
- Сокет: `/ws?room_id=…&watch=1`, для закрытого стола ещё `&key=…`.
- Новое сообщение `watchers {count}` — для «👁 3» в верхней панели стола.
- Ошибки, после которых сокет закрыт: `у стола уже слишком много зрителей`, `стол закрытый: нужно приглашение`.
- Зрителю `game_state` приходит с пустой `myHand`; по ней и по `?watch=1` в адресе фронт включает режим без руки и без кнопок.

---

## Шаг 8. Проверка руками

1. Пересобери сервер: `docker compose up -d --build server`.
2. Двумя аккаунтами начни партию (или аккаунт + бот).
3. Третьим аккаунтом: `GET /api/v1/tables/live` — стол есть, `watchers: 0`.
4. Открой `/table/<id>?watch=1` (после фронта) или подключись `wscat`-ом к `ws://localhost:8080/ws?room_id=<id>&watch=1` с кукой — приходит `game_state` с пустой `myHand`, игроки получают `watchers {count:1}`.
5. Отправь зрителем `{"type":"take"}` — ошибка, партия не изменилась.
6. Закрытый стол без `key` — отказ; с `key` — пускает.

---

## Шаг 9. Бонусы

### 9.1. Смотреть за другом ★★☆
В списке друзей у статуса «играет» кнопка «Смотреть». Нужно, чтобы `Presence` отдавала ещё и id стола (только для публичных столов!), а фронт открывал `/table/<id>?watch=1`. Подумай, как не раскрыть id закрытого стола.

### 9.2. Режим «через плечо» ★★★
Друг смотрит **с картами** одного игрока, который его позвал. Тут уже нужна задержка трансляции (например, 30 секунд буфер сообщений на зрителя), иначе это готовая схема для подсказок.

### 9.3. Зрители в чате ★☆☆
Разрешить зрителям писать, но помечать сообщения `spectator: true` и, например, не показывать их игрокам во время партии.

### 9.4. Список зрителей с именами ★☆☆
Вместо числа слать `[{id, name}]`. Подумай, нужно ли это игрокам и не мешает ли приватности.

---

## Вопросы комиссии

> [!question]- Как вы гарантируете, что зритель не видит карты игроков?
> Состояние партии собирается на сервере отдельно для каждого получателя. Рука попадает в сообщение, только если id получателя совпадает с id игрока, а игрок этого стола всегда входит как игрок, не как зритель. Клиенту вообще не отправляется то, что ему нельзя видеть.

> [!question]- Почему зрители хранятся отдельно от игроков?
> Весь код про места, старт партии, реванш и вторые вкладки опирается на то, что в карте игроков только игроки. Отдельная карта сохраняет этот инвариант без правок старого кода, а рассылка «всем» сделана одним помощником.

> [!question]- Что мешает открыть тысячу вкладок-зрителей и положить сервер?
> Лимит 30 зрителей на стол, проверяется в горутине комнаты, поэтому без гонок. Плюс вторая вкладка того же аккаунта вытесняет первую, а не добавляется.

> [!question]- Можно ли подсмотреть закрытый стол?
> Нет: зритель проходит ту же проверку `MayEnter`, что и игрок, — участник стола или правильный секретный ключ, сравнение за постоянное время. В списке идущих партий закрытых столов нет.

## Шпаргалка

| Где | Что |
|---|---|
| `client.go` | `watch`, `NewWatcher`, `Watching` |
| `room.go` | `watchers`, `watcherCount`, развилки в `handleRegister/Unregister/Incoming`, рассылки через `broadcast`/`sendAll`, `gameOverPayload`, `cleanup` |
| `spectate.go` | `maxWatchers`, `addWatcher`, `removeWatcher`, `broadcast`, `Hub.watch`, `ListLiveRooms` |
| `hub.go` | развилка `watch=1` до `admit` |
| `table_handler.go`, `server.go` | `Live`, `GET /tables/live` |

## Словарь

- **Зритель (spectator, watcher)** — соединение со столом без места и без права ходить.
- **Инвариант** — правило, которое всегда верно в коде («в `clients` только игроки»); на него опирается остальной код.
- **Задержка трансляции** — показ событий зрителям с опозданием, чтобы они не могли подсказывать; нужна, только когда зритель видит закрытую информацию.
- **Горутина-владелец** — единственная горутина, которая трогает данные; остальные общаются с ней через каналы.

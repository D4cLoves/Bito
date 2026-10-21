---
title: Bito — метрики и нагрузочный тест (учебный план)
tags: [bito, plan, metrics, prometheus, grafana, load-testing, go, learning]
created: 2026-10-21
---

# Метрики и нагрузочный тест: учебный план

> [!abstract] Что строим
> 1. **Метрики**: сервер считает, что с ним происходит — сколько столов и игроков, сколько партий в минуту, как быстро комната обрабатывает ход, сколько времени уходит на кассу в базе, теряются ли сообщения. **Prometheus** каждые 5 секунд забирает эти числа, **Grafana** рисует из них дашборд.
> 2. **Нагрузочный тест** `cmd/loadtest`: сотни виртуальных игроков жмут «Быстрая игра», садятся за столы и **по-настоящему играют** партии через сокет. В конце — отчёт: сколько партий, задержки p50/p95/p99, ошибки.
>
> Вместе это даёт картину «нагрузка → как отвечает сервер → где узкое место», и её можно показать на защите живьём.
> Связанные заметки: [[ci-plan]], [[lobby-socket-plan]] (горутина-владелец), [[economy-plan]] (касса).

> [!info] Как работать с этой заметкой
> Каждый шаг: **зачем → теория → практика → ловушки → проверь себя → решение → готово, когда**. Решения свёрнуты.
> Порядок важен: сначала метрики (часть A), потом нагрузка (часть B) — иначе нагрузка покажет только «упало / не упало».

> [!success] Что проверено
> Эталон собран и запущен на этом проекте: `go vet`, `go test -race`, тест меток метрик, стек `server + prometheus + grafana` в docker compose, нагрузочный прогон с цифрами в шаге B5 — они настоящие, не придуманные.

---

## 0. Карта

```
   cmd/loadtest (N горутин-игроков)
        │  POST /api/v1/tables/quick       ┌───────────────────────────────┐
        │  WS  /ws?room_id=…  ход ⇄ state  │ Bito server :8080             │
        └────────────────────────────────► │   Hub, Room-горутины, касса   │
                                           │   metrics: счётчики, гистограммы
                                           │ :9091/metrics  ◄── только внутри
                                           └──────────────┬────────────────┘
                                                          │ scrape каждые 5 с
                                                 ┌────────▼────────┐
                                                 │ Prometheus :9090│ хранит ряды
                                                 └────────┬────────┘
                                                          │ PromQL
                                                 ┌────────▼────────┐
                                                 │ Grafana :3000   │ дашборд «Bito — сервер»
                                                 └─────────────────┘
```

### Шаги

**Часть A. Метрики**
- A1. Теория: типы метрик, метки, кардинальность.
- A2. Пакет `internal/metrics`.
- A3. HTTP-middleware с шаблоном маршрута.
- A4. Метрики комнат и хаба.
- A5. Отдельный порт `/metrics`.
- A6. Prometheus и Grafana в docker compose, дашборд.

**Часть B. Нагрузка**
- B1. Теория: что и как мерить, перцентили, open/closed модель.
- B2. Посев игроков без почты.
- B3. Виртуальный игрок.
- B4. Статистика и отчёт.
- B5. Прогон, чтение результатов, поиск узкого места.
- B6. Бонусы.

---

# Часть A. Метрики

## A1. Теория: что такое метрика

### Зачем

Логи отвечают на вопрос «что случилось с этим запросом». Метрики — «что происходит со всем сервером прямо сейчас и как это менялось». Во время нагрузки логов будут десятки тысяч строк, а нужно одно число: «p95 обработки хода 3 мс».

### Теория

> [!note] Четыре типа метрик Prometheus
> | Тип | Что это | Пример у нас |
> |---|---|---|
> | **Counter** | только растёт (сбрасывается при рестарте) | `bito_games_started_total` |
> | **Gauge** | текущее значение, может падать | `bito_players_connected` |
> | **Histogram** | раскладывает наблюдения по «корзинам» (`le` — less or equal) | `bito_room_handle_seconds` |
> | **Summary** | считает квантили на стороне сервера | не используем: их нельзя складывать между экземплярами |
>
> Counter сам по себе неинтересен (вечно растёт). Смотрят его **скорость**: `rate(bito_games_finished_total[1m])` — партий в секунду за последнюю минуту.

> [!note] Pull-модель
> Сервер **не отправляет** метрики. Он держит в памяти числа и отдаёт их текстом на `GET /metrics`. Prometheus сам приходит каждые 5 секунд (scrape). Плюс: сервер не зависит от Prometheus — упал Prometheus, игра работает.

> [!note] Метки и кардинальность — главная ловушка метрик
> Метка — измерение: `bito_http_requests_total{method="GET", route="/api/v1/tables/{id}", code="200"}`. Каждая **уникальная комбинация** меток — отдельный временной ряд в памяти Prometheus.
> Если в метку положить `id` стола или `user_id`, рядов станет столько, сколько столов и игроков, — память Prometheus растёт без предела. Это называется **взрыв кардинальности**.
> Правило: в метках только значения из **маленького заранее известного** набора: метод, шаблон маршрута, код ответа, тип сообщения из списка.

> [!note] Гистограмма и перцентили
> Среднее время врёт: 99 быстрых ходов и один на 2 секунды дадут «в среднем 20 мс», а игрок, у которого завис ход, этого не почувствует. Смотрят **перцентили**: p95 — «95% ходов быстрее этого». Prometheus считает их по корзинам гистограммы: `histogram_quantile(0.95, sum by (le) (rate(..._bucket[1m])))`. Точность — с точностью до границ корзин, поэтому корзины выбирают под ожидаемый диапазон.

> [!note] RED и USE
> Два чек-листа, что мерить:
> - **RED** для запросов: **R**ate (сколько), **E**rrors (сколько с ошибкой), **D**uration (как долго).
> - **USE** для ресурсов: **U**tilization, **S**aturation (очередь, переполнение), **E**rrors.
> У нас: RED — HTTP и ходы в комнатах; USE — горутины, память, CPU и `ws_send_dropped` (насыщение буфера отправки).

### Проверь себя

> [!question]- Почему нельзя сделать метку `table_id` у счётчика ходов?
> Каждый стол даст новый ряд, а столы создаются постоянно. Через сутки — сотни тысяч рядов, которые Prometheus держит в памяти и индексах. Метка должна иметь ограниченное число значений.

> [!question]- Чем плохо среднее время ответа?
> Оно прячет «хвост». Пользователи запоминают редкие долгие зависания, а среднее их размазывает. Поэтому p95/p99.

---

## A2. Пакет internal/metrics

### Зачем

Одно место, где объявлены все метрики проекта с именами и описаниями. Остальной код просто вызывает `metrics.GamesStarted.Inc()`.

### Теория

> [!note] promauto и глобальный реестр
> `promauto.NewCounter(...)` создаёт метрику и сразу регистрирует её в **глобальном** реестре Prometheus. Это проще всего: переменные пакета, импортируй и пользуйся. Минус — глобальное состояние: в тестах счётчики общие для всех тестов, поэтому тест смотрит **прирост**, а не абсолютное значение (шаг A3).
> Глобальный реестр также сам отдаёт метрики Go-рантайма: `go_goroutines`, `go_memstats_*`, `process_cpu_seconds_total` — бесплатно.

> [!note] Имена
> Формат `namespace_name_unit`: `bito_room_handle_seconds`, `bito_http_requests_total`. Время — всегда в **секундах** (дробных), счётчики — с суффиксом `_total`. Так Grafana и PromQL понимают единицы.

> [!note] GaugeFunc для состояния хаба
> Число столов и игроков уже **известно** хабу. Не надо делать `Inc/Dec` при каждом входе и выходе (легко ошибиться и разъехаться). `GaugeFunc` вызывает функцию **в момент scrape**: раз в 5 секунд хаб под `RLock` пробегает по столам и считает. Дёшево и всегда правда.
> Чтобы пакет `metrics` не импортировал `ws` (и не было цикла импортов), хаб отдаёт структуру `metrics.HubStats`, а `metrics.WatchHub(hub.Stats)` регистрирует gauges.

### Практика

1. `go get github.com/prometheus/client_golang@v1.22.0`, потом `go mod tidy`.
2. Создай `internal/metrics/metrics.go`: метрики из таблицы ниже, `HubStats`, `WatchHub`, `Handler()`.

| Метрика | Тип | Метки | Где обновляется |
|---|---|---|---|
| `bito_http_requests_total` | Counter | method, route, code | middleware |
| `bito_http_request_duration_seconds` | Histogram | method, route | middleware |
| `bito_games_started_total` | Counter | — | `startGame` |
| `bito_games_finished_total` | Counter | result (`loser`/`draw`) | конец партии |
| `bito_game_duration_seconds` | Histogram | — | конец партии |
| `bito_room_messages_total` | Counter | type | `handleIncoming` |
| `bito_room_handle_seconds` | Histogram | — | `Run` вокруг `handleIncoming` |
| `bito_bank_duration_seconds` | Histogram | op (`lock`/`settle`) | касса |
| `bito_ws_send_dropped_total` | Counter | — | `Client.Send`, ветка `default` |
| `bito_rooms`, `bito_rooms_playing`, `bito_players_connected`, `bito_watchers_connected`, `bito_lobby_online` | GaugeFunc | — | `Hub.Stats` при scrape |

> [!warning] Ловушки
> - `go mod tidy` в первый раз качает много — запускай заранее.
> - Регистрация одной и той же метрики дважды паникует (`duplicate metrics collector registration`). `WatchHub` вызывай ровно один раз — в `app.Run`, не в `NewHub` (иначе упадут тесты, где хабов много).
> - Корзины гистограмм подбирай под диапазон: обработка хода — доли миллисекунды, поэтому `ExponentialBuckets(0.0001, 2, 14)` (0.1 мс … ~0.8 с), а не стандартные (5 мс … 10 с) — в стандартных все ходы упали бы в первую корзину, и p95 был бы бессмысленным.

> [!example]- Решение: internal/metrics/metrics.go
> ```go
> package metrics
>
> import (
> 	"net/http"
>
> 	"github.com/prometheus/client_golang/prometheus"
> 	"github.com/prometheus/client_golang/prometheus/promauto"
> 	"github.com/prometheus/client_golang/prometheus/promhttp"
> )
>
> const namespace = "bito"
>
> var (
> 	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
> 		Namespace: namespace,
> 		Name:      "http_requests_total",
> 		Help:      "HTTP requests by method, route pattern and status code.",
> 	}, []string{"method", "route", "code"})
>
> 	HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
> 		Namespace: namespace,
> 		Name:      "http_request_duration_seconds",
> 		Help:      "Time to answer an HTTP request.",
> 		Buckets:   prometheus.DefBuckets,
> 	}, []string{"method", "route"})
>
> 	GamesStarted = promauto.NewCounter(prometheus.CounterOpts{
> 		Namespace: namespace,
> 		Name:      "games_started_total",
> 		Help:      "Games that locked stakes and started.",
> 	})
>
> 	GamesFinished = promauto.NewCounterVec(prometheus.CounterOpts{
> 		Namespace: namespace,
> 		Name:      "games_finished_total",
> 		Help:      "Finished games by outcome.",
> 	}, []string{"result"})
>
> 	GameDuration = promauto.NewHistogram(prometheus.HistogramOpts{
> 		Namespace: namespace,
> 		Name:      "game_duration_seconds",
> 		Help:      "From the deal to the last card.",
> 		Buckets:   []float64{15, 30, 60, 120, 180, 300, 600, 900},
> 	})
>
> 	RoomMessages = promauto.NewCounterVec(prometheus.CounterOpts{
> 		Namespace: namespace,
> 		Name:      "room_messages_total",
> 		Help:      "Messages players sent to tables, by type.",
> 	}, []string{"type"})
>
> 	RoomHandle = promauto.NewHistogram(prometheus.HistogramOpts{
> 		Namespace: namespace,
> 		Name:      "room_handle_seconds",
> 		Help:      "Time a table goroutine spends on one incoming message.",
> 		Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 14),
> 	})
>
> 	BankDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
> 		Namespace: namespace,
> 		Name:      "bank_duration_seconds",
> 		Help:      "Database time to lock stakes or settle a game.",
> 		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 12),
> 	}, []string{"op"})
>
> 	SendDropped = promauto.NewCounter(prometheus.CounterOpts{
> 		Namespace: namespace,
> 		Name:      "ws_send_dropped_total",
> 		Help:      "Messages dropped because a client's send buffer was full.",
> 	})
> )
>
> type HubStats struct {
> 	Rooms    int
> 	Playing  int
> 	Players  int
> 	Watchers int
> 	Lobby    int
> }
>
> func WatchHub(stats func() HubStats) {
> 	gauge := func(name, help string, pick func(HubStats) int) {
> 		promauto.NewGaugeFunc(prometheus.GaugeOpts{Namespace: namespace, Name: name, Help: help}, func() float64 {
> 			return float64(pick(stats()))
> 		})
> 	}
> 	gauge("rooms", "Tables in memory.", func(s HubStats) int { return s.Rooms })
> 	gauge("rooms_playing", "Tables with a game in progress.", func(s HubStats) int { return s.Playing })
> 	gauge("players_connected", "Players connected to tables.", func(s HubStats) int { return s.Players })
> 	gauge("watchers_connected", "Spectators connected to tables.", func(s HubStats) int { return s.Watchers })
> 	gauge("lobby_online", "Players connected to the lobby socket.", func(s HubStats) int { return s.Lobby })
> }
>
> func Handler() http.Handler {
> 	return promhttp.Handler()
> }
> ```

### Готово, когда

- [ ] `go build ./...` проходит
- [ ] можешь объяснить, почему `rooms` — GaugeFunc, а не Gauge с `Inc/Dec`

---

## A3. HTTP-middleware

### Зачем

RED для всех REST-запросов одним куском кода: сколько, с каким кодом, как долго — по каждому маршруту.

### Теория

> [!note] Шаблон маршрута, а не путь
> `/api/v1/tables/0a2aa05a-…` и `/api/v1/tables/62edc6f4-…` — это один маршрут `/api/v1/tables/{id}`. chi знает шаблон: `chi.RouteContext(r.Context()).RoutePattern()`. Он заполняется **во время** маршрутизации, поэтому читаем его **после** `next.ServeHTTP`.
> Несуществующий путь даёт пустой шаблон — пишем `unmatched`, иначе сканер, перебирающий `/wp-admin`, `/.env`, `/x1`, `/x2`…, взорвёт кардинальность через сырой путь.

> [!note] Код ответа
> Обработчик пишет код в `ResponseWriter`, middleware его не видит. Оборачиваем: `middleware.NewWrapResponseWriter(w, r.ProtoMajor)` из chi запоминает статус и умеет `Hijack` (нужно для WebSocket).
> У апгрейда в WebSocket gorilla пишет `101` сама в «угнанное» соединение, обёртка его не видит и отдаёт `0`. Подставляем `101 Switching Protocols`.

> [!note] Длительность WebSocket
> `ServeWS` возвращается сразу после запуска помп, поэтому `duration` для `/ws` — это время **апгрейда**, а не жизни соединения. Это и нужно. Если бы обработчик блокировался до закрытия сокета, гистограмма показывала бы минуты и портила p95 (бонус-вопрос ниже).

### Практика

1. `internal/delivery/http/metrics.go` — функция `Metrics(next http.Handler) http.Handler`.
2. В `setupMiddlewares` подключи её сразу после `middleware.RealIP`.
3. Тест: три запроса на разные `id` дают **+3** у шаблона `{id}`, неизвестный путь — **+1** у `unmatched`.

> [!example]- Решение: middleware
> ```go
> package http
>
> import (
> 	"net/http"
> 	"strconv"
> 	"time"
>
> 	"bito/internal/metrics"
>
> 	"github.com/go-chi/chi/v5"
> 	"github.com/go-chi/chi/v5/middleware"
> )
>
> func Metrics(next http.Handler) http.Handler {
> 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
> 		start := time.Now()
> 		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
> 		next.ServeHTTP(ww, r)
>
> 		route := chi.RouteContext(r.Context()).RoutePattern()
> 		if route == "" {
> 			route = "unmatched"
> 		}
> 		code := ww.Status()
> 		if code == 0 {
> 			code = http.StatusSwitchingProtocols
> 		}
> 		metrics.HTTPRequests.WithLabelValues(r.Method, route, strconv.Itoa(code)).Inc()
> 		metrics.HTTPDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
> 	})
> }
> ```

> [!example]- Решение: тест
> ```go
> package http
>
> import (
> 	"net/http"
> 	"net/http/httptest"
> 	"testing"
>
> 	"bito/internal/metrics"
>
> 	"github.com/prometheus/client_golang/prometheus/testutil"
> )
>
> func TestMetrics_LabelsUseRoutePatterns(t *testing.T) {
> 	router, token, _ := newTestTablesRouter(t)
> 	counter := func(route, code string) float64 {
> 		return testutil.ToFloat64(metrics.HTTPRequests.WithLabelValues(http.MethodGet, route, code))
> 	}
> 	before := counter("/api/v1/tables/{id}", "404")
> 	unmatched := counter("unmatched", "404")
>
> 	for _, id := range []string{"a", "b", "c"} {
> 		req := httptest.NewRequest(http.MethodGet, "/api/v1/tables/"+id, nil)
> 		req.Header.Set("Authorization", "Bearer "+token)
> 		router.ServeHTTP(httptest.NewRecorder(), req)
> 	}
> 	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/no/such/page", nil))
>
> 	if got := counter("/api/v1/tables/{id}", "404") - before; got != 3 {
> 		t.Fatalf("three table ids counted as %v requests of the pattern, want 3", got)
> 	}
> 	if got := counter("unmatched", "404") - unmatched; got != 1 {
> 		t.Fatalf("unknown path counted as %v, want 1", got)
> 	}
> }
> ```

> [!question]- Почему тест проверяет прирост, а не значение счётчика?
> Реестр глобальный: другие тесты пакета тоже шлют запросы и увеличивают те же счётчики. Абсолютное значение зависит от порядка тестов, прирост — нет.

### Готово, когда

- [ ] тест зелёный с `-race`
- [ ] на `/metrics` видны ряды `route="/api/v1/tables/{id}"`, и нет рядов с конкретными id

---

## A4. Метрики комнат и хаба

### Зачем

HTTP у нас — это вход и лобби. Сама игра идёт в сокетах и горутинах комнат, и именно там узкие места под нагрузкой.

### Теория

> [!note] Где мерить обработку хода
> Комната обрабатывает сообщения **по одному** в своей горутине (`Run`). Если обработка одного хода долгая, все остальные ходы этого стола ждут в канале `incoming`. Поэтому меряем ровно вокруг `r.handleIncoming(msg)` в `Run` — это «сколько стол занят одним ходом».

> [!note] Касса отдельно
> Блокировка ставок и расчёт ходят в Postgres — это самые медленные операции партии и единственное место, где столы **конкурируют** между собой (пул соединений, блокировки строк кошельков). Отдельная гистограмма `bank_duration_seconds{op}` покажет, когда база становится узким местом.

> [!note] Тип сообщения — только из списка
> `raw.Type` присылает клиент. Если положить его в метку как есть, злоумышленник отправит миллион разных типов — миллион рядов. `messageLabel` пропускает только известные типы, остальное — `other`.

> [!note] Потерянные сообщения — сигнал насыщения
> `Client.Send` при полном буфере (256 сообщений) **молча** выбрасывает сообщение (`default`). Под нагрузкой это значит: клиент не успевает читать, или сеть забита. Счётчик `ws_send_dropped_total` превращает тихую потерю в видимый график.

### Практика

1. Поле `startedAt time.Time` в `Room`; в `startGame` — `startedAt = time.Now()` и `GamesStarted.Inc()`.
2. Вокруг `LockStakes` и `settleWithRetry` — `BankDuration.WithLabelValues("lock"/"settle").Observe(...)`.
3. После `r.finished.Store(true)` — `countFinished()`: результат и длительность партии.
4. В `Run` — замер вокруг `handleIncoming`.
5. В `handleIncoming` после разбора типа — `RoomMessages.WithLabelValues(messageLabel(raw.Type)).Inc()`.
6. В `Client.Send`, ветка `default` — `SendDropped.Inc()`.
7. `Hub.Stats()` — обход столов под `h.mu.RLock()`.

> [!warning] Ловушки
> - `Hub.Stats` читает только атомики и неизменяемые поля комнат (`Started`, `finished`, `PlayersCount`, `Watchers`). **Не** читай `r.clients` из хаба — это гонка с горутиной комнаты.
> - Партия может закончиться ничьей — метка `result="draw"`, не забудь.

> [!example]- Решение: internal/delivery/ws/stats.go
> ```go
> package ws
>
> import (
> 	"time"
>
> 	"bito/internal/metrics"
> )
>
> func (h *Hub) Stats() metrics.HubStats {
> 	h.mu.RLock()
> 	defer h.mu.RUnlock()
>
> 	s := metrics.HubStats{Rooms: len(h.rooms), Lobby: h.lobby.Online()}
> 	for _, room := range h.rooms {
> 		if room.Started() && !room.finished.Load() {
> 			s.Playing++
> 		}
> 		s.Players += room.PlayersCount()
> 		s.Watchers += room.Watchers()
> 	}
> 	return s
> }
>
> var roomMessageTypes = map[string]bool{
> 	TypeAttack: true, TypeDefend: true, TypeTransfer: true, TypeTake: true, TypePass: true,
> 	TypeSurrender: true, TypeChat: true, TypeRematch: true, TypeRematchNo: true,
> }
>
> func messageLabel(msgType string) string {
> 	if roomMessageTypes[msgType] {
> 		return msgType
> 	}
> 	return "other"
> }
>
> func (r *Room) countFinished() {
> 	result := "loser"
> 	if r.draw {
> 		result = "draw"
> 	}
> 	metrics.GamesFinished.WithLabelValues(result).Inc()
> 	metrics.GameDuration.Observe(time.Since(r.startedAt).Seconds())
> }
> ```

> [!example]- Решение: вставки в room.go и client.go
> В `Run`:
> ```go
> 		case msg := <-r.incoming:
> 			start := time.Now()
> 			r.handleIncoming(msg)
> 			metrics.RoomHandle.Observe(time.Since(start).Seconds())
> ```
> В `startGame`:
> ```go
> 	ctx, cancel := bankContext()
> 	lockStart := time.Now()
> 	err = r.bank.LockStakes(ctx, r.id, r.gameMode.String(), int(r.deckType), r.stake, ids)
> 	metrics.BankDuration.WithLabelValues("lock").Observe(time.Since(lockStart).Seconds())
> 	cancel()
> ```
> ```go
> 	r.game = newGame
> 	r.started.Store(true)
> 	r.startedAt = time.Now()
> 	metrics.GamesStarted.Inc()
> ```
> В `settle`:
> ```go
> 	settleStart := time.Now()
> 	s, err := r.settleWithRetry(economy.Result{Places: places, Losers: losers})
> 	metrics.BankDuration.WithLabelValues("settle").Observe(time.Since(settleStart).Seconds())
> ```
> В `onGameStateChanged`:
> ```go
> 		r.settle()
> 		r.finished.Store(true)
> 		r.countFinished()
> ```
> В `handleIncoming` перед `if r.game == nil`:
> ```go
> 	metrics.RoomMessages.WithLabelValues(messageLabel(raw.Type)).Inc()
> ```
> В `Client.Send`:
> ```go
> 	select {
> 	case c.send <- data:
> 		return true
> 	default:
> 		metrics.SendDropped.Inc()
> 		return false
> 	}
> ```

### Готово, когда

- [ ] старые тесты `ws` зелёные с `-race`
- [ ] после одной партии в `/metrics` видно `bito_games_finished_total{result="loser"} 1`

---

## A5. Отдельный порт для /metrics

### Зачем

`/metrics` показывает внутренности: сколько игроков, какие маршруты, версии. Наружу это отдавать незачем.

### Теория

> [!note] Два сервера в одном процессе
> Основной сервер слушает `:8080` (публичный), второй — `:9091` только с `/metrics`. В docker compose порт `9091` проброшен как `127.0.0.1:9091` — доступен с твоей машины, но не из сети. Prometheus внутри compose ходит на `server:9091` по внутренней сети.

### Практика

1. В `config.Config`: `MetricsPort string \`env:"METRICS_PORT" env-default:"9091"\``.
2. В `app.Run` после создания хаба: `metrics.WatchHub(hub.Stats)` и `go serveMetrics(cfg.MetricsPort)`.
3. `serveMetrics` — свой `http.NewServeMux()` с `/metrics`.

> [!example]- Решение
> ```go
> 	hub := ws.NewHub(tokenManager, cfg.AllowedOrigins, economyService)
> 	metrics.WatchHub(hub.Stats)
> 	go serveMetrics(cfg.MetricsPort)
> ```
> ```go
> func serveMetrics(port string) {
> 	mux := http.NewServeMux()
> 	mux.Handle("/metrics", metrics.Handler())
> 	if err := http.ListenAndServe(":"+port, mux); err != nil {
> 		log.Printf("metrics server stopped: %v", err)
> 	}
> }
> ```

> [!question]- Почему не повесить `/metrics` на основной роутер с проверкой токена?
> Prometheus пришлось бы выдавать JWT, а сам эндпоинт попал бы под CORS, логгер и лимиты публичного API. Отдельный порт проще и безопаснее: снаружи его просто нет.

### Готово, когда

- [ ] `curl http://127.0.0.1:9091/metrics` отдаёт текст с `bito_` и `go_` рядами
- [ ] `curl http://localhost:8080/metrics` — 404

---

## A6. Prometheus и Grafana в docker compose

### Зачем

Числа на `/metrics` — это моментальный снимок. Prometheus хранит их **во времени**, Grafana рисует графики. Это и есть то, что показывают на защите.

### Теория

> [!note] Provisioning
> Grafana можно настроить файлами при старте: источник данных (`datasources/*.yml`) и дашборды (`dashboards/*.yml` + JSON). Тогда после `docker compose up` всё уже готово — никаких ручных кликов, и настройка лежит в git.

> [!note] Анонимный вход — только локально
> Для учебного стенда включаем `GF_AUTH_ANONYMOUS_ENABLED` и пробрасываем порт только на `127.0.0.1`. В продакшене так нельзя.

> [!note] PromQL, который нужен на дашборде
> - Скорость счётчика: `rate(bito_games_finished_total[1m]) * 60` — партий в минуту.
> - Разбивка: `sum by (type) (rate(bito_room_messages_total[1m]))`.
> - Перцентиль: `histogram_quantile(0.95, sum by (le) (rate(bito_room_handle_seconds_bucket[1m])))`.
> - Перцентиль по маршрутам: `sum by (le, route)` — `le` обязательно остаётся в `by`, иначе `histogram_quantile` не из чего считать.

### Практика

1. `deploy/prometheus/prometheus.yml` — один job, `targets: ["server:9091"]`, `scrape_interval: 5s`.
2. `deploy/grafana/provisioning/datasources/prometheus.yml` и `dashboards/bito.yml`.
3. `deploy/grafana/dashboards/bito.json` — дашборд (готовый — в решении; можно собрать свой в интерфейсе и **экспортировать JSON**).
4. В `docker-compose.yml`: у `server` — `METRICS_PORT` и порт `127.0.0.1:9091:9091`; сервисы `prometheus` (`prom/prometheus:v3.1.0`) и `grafana` (`grafana/grafana:11.4.0`), тома для данных.
5. `docker compose up -d --build` → `http://localhost:9090/targets` (цель `UP`) → `http://localhost:3000` (дашборд «Bito — сервер»).

> [!warning] Ловушки
> - Версии образов закрепляй (`v3.1.0`, а не `latest`): иначе через месяц новый релиз поменяет формат и стенд не поднимется перед защитой.
> - В `prometheus.yml` цель — **имя сервиса** `server`, не `localhost`: внутри контейнера Prometheus `localhost` — это он сам.
> - Если на `/targets` цель `DOWN` с `connection refused` — сервер не слушает `9091` или переменная `METRICS_PORT` не дошла.

> [!example]- Решение: prometheus.yml
> ```yaml
> global:
>   scrape_interval: 5s
>
> scrape_configs:
>   - job_name: bito
>     static_configs:
>       - targets: ["server:9091"]
> ```

> [!example]- Решение: provisioning Grafana
> ```yaml
> apiVersion: 1
>
> datasources:
>   - name: Prometheus
>     uid: prometheus
>     type: prometheus
>     access: proxy
>     url: http://prometheus:9090
>     isDefault: true
> ```
> ```yaml
> apiVersion: 1
>
> providers:
>   - name: bito
>     folder: Bito
>     type: file
>     options:
>       path: /var/lib/grafana/dashboards
> ```

> [!example]- Решение: новые сервисы docker-compose.yml
> ```yaml
>   server:
>     ports:
>       - "8080:8080"
>       - "127.0.0.1:9091:9091"
>     environment:
>       METRICS_PORT: "9091"
> ```
> ```yaml
>   prometheus:
>     image: prom/prometheus:v3.1.0
>     container_name: bito_prometheus
>     restart: always
>     command:
>       - --config.file=/etc/prometheus/prometheus.yml
>       - --storage.tsdb.retention.time=7d
>     ports:
>       - "127.0.0.1:9090:9090"
>     volumes:
>       - ./deploy/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
>       - prometheus_data:/prometheus
>     depends_on:
>       - server
>
>   grafana:
>     image: grafana/grafana:11.4.0
>     container_name: bito_grafana
>     restart: always
>     environment:
>       GF_AUTH_ANONYMOUS_ENABLED: "true"
>       GF_AUTH_ANONYMOUS_ORG_ROLE: "Admin"
>       GF_AUTH_DISABLE_LOGIN_FORM: "true"
>       GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_PATH: /var/lib/grafana/dashboards/bito.json
>     ports:
>       - "127.0.0.1:3000:3000"
>     volumes:
>       - ./deploy/grafana/provisioning:/etc/grafana/provisioning:ro
>       - ./deploy/grafana/dashboards:/var/lib/grafana/dashboards:ro
>       - grafana_data:/var/lib/grafana
>     depends_on:
>       - prometheus
>
> volumes:
>   postgres_data:
>   redis_data:
>   prometheus_data:
>   grafana_data:
> ```

> [!example]- Решение: дашборд deploy/grafana/dashboards/bito.json
> 12 панелей: столы, игроки, лобби, горутины, партии в минуту, ходы по типам, обработка хода p50/p95/p99, HTTP p95 по маршрутам, касса, ошибки и потери, память, CPU. Можно собрать свой в интерфейсе Grafana и сохранить через **Export → JSON**.
> ```json
> {
>   "uid": "bito-overview",
>   "title": "Bito — сервер",
>   "timezone": "browser",
>   "refresh": "5s",
>   "time": {
>     "from": "now-15m",
>     "to": "now"
>   },
>   "schemaVersion": 39,
>   "panels": [
>     {
>       "id": 1,
>       "title": "Столов с партией",
>       "type": "stat",
>       "datasource": {
>         "type": "prometheus",
>         "uid": "prometheus"
>       },
>       "gridPos": {
>         "x": 0,
>         "y": 0,
>         "w": 6,
>         "h": 4
>       },
>       "targets": [
>         {
>           "refId": "A",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "bito_rooms_playing",
>           "legendFormat": ""
>         }
>       ],
>       "fieldConfig": {
>         "defaults": {
>           "unit": "short"
>         },
>         "overrides": []
>       },
>       "options": {
>         "reduceOptions": {
>           "calcs": [
>             "lastNotNull"
>           ]
>         },
>         "graphMode": "area",
>         "colorMode": "none"
>       }
>     },
>     {
>       "id": 2,
>       "title": "Игроков за столами",
>       "type": "stat",
>       "datasource": {
>         "type": "prometheus",
>         "uid": "prometheus"
>       },
>       "gridPos": {
>         "x": 6,
>         "y": 0,
>         "w": 6,
>         "h": 4
>       },
>       "targets": [
>         {
>           "refId": "A",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "bito_players_connected",
>           "legendFormat": ""
>         }
>       ],
>       "fieldConfig": {
>         "defaults": {
>           "unit": "short"
>         },
>         "overrides": []
>       },
>       "options": {
>         "reduceOptions": {
>           "calcs": [
>             "lastNotNull"
>           ]
>         },
>         "graphMode": "area",
>         "colorMode": "none"
>       }
>     },
>     {
>       "id": 3,
>       "title": "В лобби",
>       "type": "stat",
>       "datasource": {
>         "type": "prometheus",
>         "uid": "prometheus"
>       },
>       "gridPos": {
>         "x": 12,
>         "y": 0,
>         "w": 6,
>         "h": 4
>       },
>       "targets": [
>         {
>           "refId": "A",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "bito_lobby_online",
>           "legendFormat": ""
>         }
>       ],
>       "fieldConfig": {
>         "defaults": {
>           "unit": "short"
>         },
>         "overrides": []
>       },
>       "options": {
>         "reduceOptions": {
>           "calcs": [
>             "lastNotNull"
>           ]
>         },
>         "graphMode": "area",
>         "colorMode": "none"
>       }
>     },
>     {
>       "id": 4,
>       "title": "Горутин",
>       "type": "stat",
>       "datasource": {
>         "type": "prometheus",
>         "uid": "prometheus"
>       },
>       "gridPos": {
>         "x": 18,
>         "y": 0,
>         "w": 6,
>         "h": 4
>       },
>       "targets": [
>         {
>           "refId": "A",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "go_goroutines{job=\"bito\"}",
>           "legendFormat": ""
>         }
>       ],
>       "fieldConfig": {
>         "defaults": {
>           "unit": "short"
>         },
>         "overrides": []
>       },
>       "options": {
>         "reduceOptions": {
>           "calcs": [
>             "lastNotNull"
>           ]
>         },
>         "graphMode": "area",
>         "colorMode": "none"
>       }
>     },
>     {
>       "id": 5,
>       "title": "Партий в минуту",
>       "type": "timeseries",
>       "datasource": {
>         "type": "prometheus",
>         "uid": "prometheus"
>       },
>       "gridPos": {
>         "x": 0,
>         "y": 4,
>         "w": 12,
>         "h": 8
>       },
>       "targets": [
>         {
>           "refId": "A",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "sum(rate(bito_games_finished_total[1m])) * 60",
>           "legendFormat": "закончено"
>         },
>         {
>           "refId": "B",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "rate(bito_games_started_total[1m]) * 60",
>           "legendFormat": "начато"
>         }
>       ],
>       "fieldConfig": {
>         "defaults": {
>           "unit": "short",
>           "custom": {
>             "lineWidth": 2,
>             "fillOpacity": 10
>           }
>         },
>         "overrides": []
>       },
>       "options": {
>         "legend": {
>           "displayMode": "list",
>           "placement": "bottom"
>         },
>         "tooltip": {
>           "mode": "multi"
>         }
>       }
>     },
>     {
>       "id": 6,
>       "title": "Ходы в секунду по типам",
>       "type": "timeseries",
>       "datasource": {
>         "type": "prometheus",
>         "uid": "prometheus"
>       },
>       "gridPos": {
>         "x": 12,
>         "y": 4,
>         "w": 12,
>         "h": 8
>       },
>       "targets": [
>         {
>           "refId": "A",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "sum by (type) (rate(bito_room_messages_total[1m]))",
>           "legendFormat": "{{type}}"
>         }
>       ],
>       "fieldConfig": {
>         "defaults": {
>           "unit": "reqps",
>           "custom": {
>             "lineWidth": 2,
>             "fillOpacity": 10
>           }
>         },
>         "overrides": []
>       },
>       "options": {
>         "legend": {
>           "displayMode": "list",
>           "placement": "bottom"
>         },
>         "tooltip": {
>           "mode": "multi"
>         }
>       }
>     },
>     {
>       "id": 7,
>       "title": "Обработка хода в комнате",
>       "type": "timeseries",
>       "datasource": {
>         "type": "prometheus",
>         "uid": "prometheus"
>       },
>       "gridPos": {
>         "x": 0,
>         "y": 12,
>         "w": 12,
>         "h": 8
>       },
>       "targets": [
>         {
>           "refId": "A",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "histogram_quantile(0.5, sum by (le) (rate(bito_room_handle_seconds_bucket[1m])))",
>           "legendFormat": "p50"
>         },
>         {
>           "refId": "B",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "histogram_quantile(0.95, sum by (le) (rate(bito_room_handle_seconds_bucket[1m])))",
>           "legendFormat": "p95"
>         },
>         {
>           "refId": "C",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "histogram_quantile(0.99, sum by (le) (rate(bito_room_handle_seconds_bucket[1m])))",
>           "legendFormat": "p99"
>         }
>       ],
>       "fieldConfig": {
>         "defaults": {
>           "unit": "s",
>           "custom": {
>             "lineWidth": 2,
>             "fillOpacity": 10
>           }
>         },
>         "overrides": []
>       },
>       "options": {
>         "legend": {
>           "displayMode": "list",
>           "placement": "bottom"
>         },
>         "tooltip": {
>           "mode": "multi"
>         }
>       }
>     },
>     {
>       "id": 8,
>       "title": "HTTP p95 по маршрутам",
>       "type": "timeseries",
>       "datasource": {
>         "type": "prometheus",
>         "uid": "prometheus"
>       },
>       "gridPos": {
>         "x": 12,
>         "y": 12,
>         "w": 12,
>         "h": 8
>       },
>       "targets": [
>         {
>           "refId": "A",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "histogram_quantile(0.95, sum by (le, route) (rate(bito_http_request_duration_seconds_bucket[1m])))",
>           "legendFormat": "{{route}}"
>         }
>       ],
>       "fieldConfig": {
>         "defaults": {
>           "unit": "s",
>           "custom": {
>             "lineWidth": 2,
>             "fillOpacity": 10
>           }
>         },
>         "overrides": []
>       },
>       "options": {
>         "legend": {
>           "displayMode": "list",
>           "placement": "bottom"
>         },
>         "tooltip": {
>           "mode": "multi"
>         }
>       }
>     },
>     {
>       "id": 9,
>       "title": "Касса (база) p95",
>       "type": "timeseries",
>       "datasource": {
>         "type": "prometheus",
>         "uid": "prometheus"
>       },
>       "gridPos": {
>         "x": 0,
>         "y": 20,
>         "w": 12,
>         "h": 8
>       },
>       "targets": [
>         {
>           "refId": "A",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "histogram_quantile(0.95, sum by (le, op) (rate(bito_bank_duration_seconds_bucket[1m])))",
>           "legendFormat": "{{op}}"
>         }
>       ],
>       "fieldConfig": {
>         "defaults": {
>           "unit": "s",
>           "custom": {
>             "lineWidth": 2,
>             "fillOpacity": 10
>           }
>         },
>         "overrides": []
>       },
>       "options": {
>         "legend": {
>           "displayMode": "list",
>           "placement": "bottom"
>         },
>         "tooltip": {
>           "mode": "multi"
>         }
>       }
>     },
>     {
>       "id": 10,
>       "title": "Ошибки и потери",
>       "type": "timeseries",
>       "datasource": {
>         "type": "prometheus",
>         "uid": "prometheus"
>       },
>       "gridPos": {
>         "x": 12,
>         "y": 20,
>         "w": 12,
>         "h": 8
>       },
>       "targets": [
>         {
>           "refId": "A",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "sum by (code) (rate(bito_http_requests_total{code=~\"5..\"}[1m]))",
>           "legendFormat": "HTTP {{code}}"
>         },
>         {
>           "refId": "B",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "rate(bito_ws_send_dropped_total[1m])",
>           "legendFormat": "сокет: потеряно"
>         }
>       ],
>       "fieldConfig": {
>         "defaults": {
>           "unit": "reqps",
>           "custom": {
>             "lineWidth": 2,
>             "fillOpacity": 10
>           }
>         },
>         "overrides": []
>       },
>       "options": {
>         "legend": {
>           "displayMode": "list",
>           "placement": "bottom"
>         },
>         "tooltip": {
>           "mode": "multi"
>         }
>       }
>     },
>     {
>       "id": 11,
>       "title": "Память кучи",
>       "type": "timeseries",
>       "datasource": {
>         "type": "prometheus",
>         "uid": "prometheus"
>       },
>       "gridPos": {
>         "x": 0,
>         "y": 28,
>         "w": 12,
>         "h": 8
>       },
>       "targets": [
>         {
>           "refId": "A",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "go_memstats_heap_inuse_bytes{job=\"bito\"}",
>           "legendFormat": "heap in use"
>         }
>       ],
>       "fieldConfig": {
>         "defaults": {
>           "unit": "bytes",
>           "custom": {
>             "lineWidth": 2,
>             "fillOpacity": 10
>           }
>         },
>         "overrides": []
>       },
>       "options": {
>         "legend": {
>           "displayMode": "list",
>           "placement": "bottom"
>         },
>         "tooltip": {
>           "mode": "multi"
>         }
>       }
>     },
>     {
>       "id": 12,
>       "title": "CPU процесса",
>       "type": "timeseries",
>       "datasource": {
>         "type": "prometheus",
>         "uid": "prometheus"
>       },
>       "gridPos": {
>         "x": 12,
>         "y": 28,
>         "w": 12,
>         "h": 8
>       },
>       "targets": [
>         {
>           "refId": "A",
>           "datasource": {
>             "type": "prometheus",
>             "uid": "prometheus"
>           },
>           "expr": "rate(process_cpu_seconds_total{job=\"bito\"}[1m])",
>           "legendFormat": "ядер"
>         }
>       ],
>       "fieldConfig": {
>         "defaults": {
>           "unit": "short",
>           "custom": {
>             "lineWidth": 2,
>             "fillOpacity": 10
>           }
>         },
>         "overrides": []
>       },
>       "options": {
>         "legend": {
>           "displayMode": "list",
>           "placement": "bottom"
>         },
>         "tooltip": {
>           "mode": "multi"
>         }
>       }
>     }
>   ]
> }
> ```

### Готово, когда

- [ ] в Prometheus цель `bito` в состоянии `UP`
- [ ] в Grafana дашборд открывается и графики живые, когда играешь в браузере

---

# Часть B. Нагрузочный тест

## B1. Теория нагрузки

### Зачем

Вопрос комиссии «а сколько игроков выдержит сервер?» требует числа, а не «наверное много». Нагрузочный тест даёт это число и показывает, **что** сломается первым.

### Теория

> [!note] Виды тестов
> | Вид | Вопрос | Как у нас |
> |---|---|---|
> | **Smoke** | работает ли вообще под минимальной нагрузкой | 10 игроков, 30 с |
> | **Load** | держит ли ожидаемую нагрузку | 100–200 игроков, 2 мин |
> | **Stress** | где предел | увеличиваем игроков, пока p95 или ошибки не поползут |
> | **Soak** | нет ли утечек за долгое время | 50 игроков, 30+ мин, смотрим память и горутины |

> [!note] Что значит «игрок» в тесте
> Важно мерить **реальный сценарий**, а не «сколько раз в секунду можно дёрнуть `/health`». Наш виртуальный игрок делает то же, что человек: быстрая игра → сокет → ждёт соперника → ходит по правилам → итог → снова. Только думает 300 мс вместо 5 секунд.

> [!note] Закрытая модель и «думание»
> У нас **закрытая модель**: N игроков, каждый делает следующий ход только после ответа. Нагрузка задаётся числом игроков и паузой `think`. Без паузы 100 ботов создают нагрузку как тысячи живых людей — это стресс, а не реализм. С паузой 300 мс один бот ≈ 10–15 живых игроков по темпу ходов.

> [!note] Что мерить на стороне клиента
> - `quick match` — HTTP: быстрая игра (с проверкой баланса в базе);
> - `socket connect` — апгрейд в WebSocket;
> - `wait for opponent` — от посадки до первой раздачи (сюда входит блокировка ставок в базе);
> - `move round trip` — **главная**: от отправки хода до прихода нового `game_state`. Это то, что игрок ощущает как «лаг».
>
> На стороне сервера (Grafana) — то же изнутри: `room_handle_seconds`, `bank_duration_seconds`, горутины, CPU. Сравнение клиентских и серверных цифр показывает, где теряется время: в сервере, в базе или в сети.

> [!note] Генератор тоже может быть узким местом
> Если машина с нагрузкой загружена на 100% CPU, задержки растут **у неё**, а не у сервера. На одном ноутбуке сервер и генератор делят CPU — это нормально для учебного теста, но цифры надо читать с этой поправкой.

### Проверь себя

> [!question]- Почему 100 ботов без паузы — плохой тест «на 100 игроков»?
> Живой игрок ходит раз в несколько секунд, бот без паузы — сотни раз в секунду. Получится тест на пропускную способность комнат, а не на 100 человек. Это полезно (stress), но называть его надо честно.

> [!question]- Зачем p99, если есть p95?
> p95 показывает типичный «плохой» случай, p99 — редкие подвисания, например из-за сборки мусора или ожидания соединения с базой. Один ход из ста — это заметно для игрока за партию из 40 ходов.

---

## B2. Посев игроков без почты

### Зачем

Регистрация требует кода из письма и упирается в лимит запросов на вход по IP (Redis rate limiter). Нам нужно 200 игроков за секунды, а тест — **про игру**, не про логин.

### Теория

> [!note] Два обхода — и почему они честные
> 1. **Игроки создаются прямо в базе** тем же репозиторием, что и при регистрации: `UserRepository.CreateUser` пишет `users`, `user_stats`, `user_wallets` и стартовый бонус в журнал. Значит инвариант «баланс = сумма журнала» не ломается.
> 2. **JWT подписывается тем же секретом**, что у сервера (`JWT_SECRET` из compose): `jwt.NewTokenManager(secret, ...).GenerateAccessToken(id, name)`. Сервер принимает такой токен, как выданный при входе.
>
> Логин и регистрацию при этом не тестируем — это отдельный сценарий (бонус B6.2).

> [!note] Повторный запуск
> Игроки `load0000…loadNNNN` с почтой `@load.test` ищутся по email; если уже есть — берём существующего. Тест можно запускать много раз.

> [!warning] Ловушки
> - Секрет должен **совпадать** с сервером. Иначе все сокеты получат `401`.
> - Не запускай против продакшена: тест создаёт пользователей и двигает фишки.
> - Проигрывающие боты теряют по 100 фишек за партию. После ~25 поражений у бота не хватит на ставку — в отчёте появится `HTTP 402`/ошибка фишек. Это не баг сервера, а честная экономика (бонус B6.1 — доливать через `/wallet/top-up`).

> [!example]- Решение: cmd/loadtest/seed.go
> ```go
> package main
>
> import (
> 	"context"
> 	"errors"
> 	"fmt"
> 	"time"
>
> 	"bito/internal/model"
> 	"bito/internal/pkg/jwt"
> 	"bito/internal/repository/postgres"
>
> 	"github.com/google/uuid"
> 	"github.com/jackc/pgx/v5/pgxpool"
> )
>
> type user struct {
> 	id    uuid.UUID
> 	name  string
> 	token string
> }
>
> func seed(ctx context.Context, o options) ([]user, error) {
> 	pool, err := pgxpool.New(ctx, o.dsn)
> 	if err != nil {
> 		return nil, err
> 	}
> 	defer pool.Close()
>
> 	repo := postgres.NewUserRepository(pool)
> 	tokens := jwt.NewTokenManager(o.secret, o.duration+time.Hour, time.Hour)
>
> 	users := make([]user, 0, o.players)
> 	for i := range o.players {
> 		name := fmt.Sprintf("%s%04d", o.prefix, i)
> 		email := name + "@load.test"
>
> 		u, err := repo.GetByEmail(ctx, email)
> 		if errors.Is(err, model.ErrUserNotFound) {
> 			u = &model.User{Email: email, Name: name}
> 			err = repo.CreateUser(ctx, u, &model.UserStats{}, &model.UserWallet{})
> 		}
> 		if err != nil {
> 			return nil, fmt.Errorf("%s: %w", name, err)
> 		}
>
> 		token, err := tokens.GenerateAccessToken(u.ID, u.Name)
> 		if err != nil {
> 			return nil, err
> 		}
> 		users = append(users, user{id: u.ID, name: u.Name, token: token})
> 	}
> 	return users, nil
> }
> ```

---

## B3. Виртуальный игрок

### Зачем

Сердце теста: горутина, которая ведёт себя как игрок и меряет задержки.

### Теория

> [!note] Цикл игрока
> ```
> пока не вышло время:
>   POST /tables/quick  → id стола         (мерим)
>   WS  /ws?room_id=id  → соединение       (мерим)
>   читаем сообщения:
>     game_state → если мой ход: подумать, отправить, запомнить время
>                  если ждали ответа на ход: мерим round trip
>     game_over  → +1 партия, следующий круг
>     error      → считаем ошибку
> ```

> [!note] Кто я в партии
> Ход определяется по `attackerId`/`defenderId`, а свой id бот знает из посева. Не по имени: имена могут совпасть или отличаться регистром — на этом мы уже обжигались с тестовым ботом.

> [!note] Стратегия — простая, но по правилам
> Защитник бьёт первую небитую карту самой младшей подходящей, иначе берёт. Атакующий ходит младшей картой; когда всё побито или защитник берёт — «Бито»/«Хватит». Этого достаточно, чтобы партии доигрывались до конца и сервер проходил все фазы: бой, подкидывание, взятие, расчёт.

> [!note] Окончание теста
> По дедлайну новые партии не начинаются. Если бот сидит в поиске соперника (пары не нашлось), чтение прерывается по дедлайну и бот уходит. Начатую партию доигрываем (с потолком 5 минут), иначе соперник останется один, а в отчёте будут ложные ошибки.

> [!warning] Ловушки
> - **Origin.** Сервер проверяет `Origin` у сокетов (`CheckOrigin`). Без заголовка `Origin: http://localhost:5173` будет `403`.
> - Токен для сокета — в cookie `accessToken` (так делает браузер), для REST — `Authorization: Bearer`.
> - `ReadJSON` блокируется навсегда, если сервер замолчал. Всегда ставь `SetReadDeadline`.
> - Ошибка от сервера (`error`) не всегда фатальна: «не ваш ход» после гонки с таймером — норма. Считаем её, но продолжаем.
> - **Пул соединений.** `http.DefaultClient` держит только 2 свободных соединения на хост. Сотни горутин будут открывать новые TCP-соединения на каждый запрос и выжгут локальные порты (шаг B5). Нужен свой `http.Client` с `MaxIdleConnsPerHost` под число игроков.
> - **TIME_WAIT.** Закрытое соединение ещё пару минут держит порт. Между тяжёлыми прогонами проверяй `netstat -an | grep -c TIME_WAIT` и давай портам освободиться.

> [!example]- Решение: cmd/loadtest/player.go
> ```go
> package main
>
> import (
> 	"bytes"
> 	"context"
> 	"encoding/json"
> 	"errors"
> 	"fmt"
> 	"net/http"
> 	"net/url"
> 	"sort"
> 	"strings"
> 	"time"
>
> 	"github.com/gorilla/websocket"
> )
>
> const gameLimit = 5 * time.Minute
>
> var httpClient = &http.Client{
> 	Timeout: 10 * time.Second,
> 	Transport: &http.Transport{
> 		MaxIdleConns:        2048,
> 		MaxIdleConnsPerHost: 2048,
> 		IdleConnTimeout:     90 * time.Second,
> 	},
> }
>
> type card struct {
> 	Suit int `json:"suit"`
> 	Rank int `json:"rank"`
> }
>
> type pair struct {
> 	Attack card  `json:"attack"`
> 	Defend *card `json:"defend"`
> }
>
> type gameState struct {
> 	Phase      string `json:"phase"`
> 	TrumpSuit  int    `json:"trumpSuit"`
> 	Table      []pair `json:"table"`
> 	AttackerID string `json:"attackerId"`
> 	DefenderID string `json:"defenderId"`
> 	MyHand     []card `json:"myHand"`
> 	YouPassed  bool   `json:"youPassed"`
> }
>
> type envelope struct {
> 	Type    string          `json:"type"`
> 	Payload json.RawMessage `json:"payload"`
> }
>
> type player struct {
> 	user  user
> 	opts  options
> 	stats *stats
> 	conn  *websocket.Conn
> 	sent  time.Time
> }
>
> var errStop = errors.New("stop")
>
> func (p *player) run(ctx context.Context, deadline time.Time) {
> 	p.stats.active(1)
> 	defer p.stats.active(-1)
>
> 	for ctx.Err() == nil && time.Now().Before(deadline) {
> 		err := p.playOne(ctx, deadline)
> 		if errors.Is(err, errStop) {
> 			return
> 		}
> 		if err != nil {
> 			p.stats.fail(err.Error())
> 			select {
> 			case <-time.After(time.Second):
> 			case <-ctx.Done():
> 				return
> 			}
> 		}
> 	}
> }
>
> func (p *player) playOne(ctx context.Context, deadline time.Time) error {
> 	start := time.Now()
> 	tableID, err := p.quickMatch(ctx)
> 	if err != nil {
> 		return err
> 	}
> 	p.stats.observe("quick match", time.Since(start))
>
> 	start = time.Now()
> 	if err = p.dial(ctx, tableID); err != nil {
> 		return err
> 	}
> 	defer p.conn.Close()
> 	p.stats.observe("socket connect", time.Since(start))
>
> 	stopWatch := context.AfterFunc(ctx, func() { _ = p.conn.Close() })
> 	defer stopWatch()
>
> 	seated := time.Now()
> 	started := false
> 	for {
> 		limit := time.Now().Add(gameLimit)
> 		if !started && deadline.Before(limit) {
> 			limit = deadline
> 		}
> 		_ = p.conn.SetReadDeadline(limit)
>
> 		var msg envelope
> 		if err = p.conn.ReadJSON(&msg); err != nil {
> 			if !started {
> 				return errStop
> 			}
> 			return fmt.Errorf("read: %w", err)
> 		}
>
> 		switch msg.Type {
> 		case "game_state":
> 			if !started {
> 				started = true
> 				p.stats.observe("wait for opponent", time.Since(seated))
> 			}
> 			if !p.sent.IsZero() {
> 				p.stats.observe("move round trip", time.Since(p.sent))
> 				p.sent = time.Time{}
> 			}
> 			var s gameState
> 			if err = json.Unmarshal(msg.Payload, &s); err != nil {
> 				return err
> 			}
> 			if err = p.move(s); err != nil {
> 				return err
> 			}
> 		case "game_over":
> 			p.stats.gameDone(tableID)
> 			return nil
> 		case "error":
> 			var e struct {
> 				Message string `json:"message"`
> 			}
> 			_ = json.Unmarshal(msg.Payload, &e)
> 			p.sent = time.Time{}
> 			p.stats.fail("server: " + e.Message)
> 		}
> 	}
> }
>
> func (p *player) quickMatch(ctx context.Context) (string, error) {
> 	body, _ := json.Marshal(map[string]any{"mode": "any", "deck": 0, "stake": p.opts.stake})
> 	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.opts.base+"/api/v1/tables/quick", bytes.NewReader(body))
> 	if err != nil {
> 		return "", err
> 	}
> 	req.Header.Set("Content-Type", "application/json")
> 	req.Header.Set("Authorization", "Bearer "+p.user.token)
>
> 	resp, err := httpClient.Do(req)
> 	if err != nil {
> 		return "", fmt.Errorf("quick match: %w", err)
> 	}
> 	defer resp.Body.Close()
> 	if resp.StatusCode >= 300 {
> 		return "", fmt.Errorf("quick match: HTTP %d", resp.StatusCode)
> 	}
>
> 	var table struct {
> 		ID string `json:"id"`
> 	}
> 	if err = json.NewDecoder(resp.Body).Decode(&table); err != nil {
> 		return "", err
> 	}
> 	return table.ID, nil
> }
>
> func (p *player) dial(ctx context.Context, tableID string) error {
> 	u := strings.Replace(p.opts.base, "http", "ws", 1) + "/ws?room_id=" + url.QueryEscape(tableID)
> 	h := http.Header{}
> 	h.Set("Origin", p.opts.origin)
> 	h.Set("Cookie", "accessToken="+p.user.token)
>
> 	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u, h)
> 	if err != nil {
> 		return fmt.Errorf("socket: %w", err)
> 	}
> 	p.conn = conn
> 	return nil
> }
>
> func (p *player) move(s gameState) error {
> 	me := p.user.id.String()
> 	hand := s.MyHand
> 	sort.Slice(hand, func(i, j int) bool {
> 		ti, tj := hand[i].Suit == s.TrumpSuit, hand[j].Suit == s.TrumpSuit
> 		if ti != tj {
> 			return !ti
> 		}
> 		return hand[i].Rank < hand[j].Rank
> 	})
>
> 	switch {
> 	case s.Phase != "battle" && s.Phase != "giving_more":
> 		return nil
> 	case me == s.DefenderID && s.Phase == "battle":
> 		for _, pr := range s.Table {
> 			if pr.Defend != nil {
> 				continue
> 			}
> 			for _, c := range hand {
> 				if beats(pr.Attack, c, s.TrumpSuit) {
> 					return p.send("defend", map[string]any{"attackCard": pr.Attack, "defendCard": c})
> 				}
> 			}
> 			return p.send("take", nil)
> 		}
> 	case me == s.AttackerID && len(s.Table) == 0 && len(hand) > 0:
> 		return p.send("attack", map[string]any{"attackCard": hand[0]})
> 	case me == s.AttackerID && !s.YouPassed && len(s.Table) > 0:
> 		for _, pr := range s.Table {
> 			if pr.Defend == nil && s.Phase == "battle" {
> 				return nil
> 			}
> 		}
> 		return p.send("pass", nil)
> 	}
> 	return nil
> }
>
> func (p *player) send(msgType string, payload any) error {
> 	time.Sleep(p.opts.think)
> 	p.sent = time.Now()
> 	return p.conn.WriteJSON(map[string]any{"type": msgType, "payload": payload})
> }
>
> func beats(a, d card, trump int) bool {
> 	if d.Suit == trump && a.Suit != trump {
> 		return true
> 	}
> 	return d.Suit == a.Suit && d.Rank > a.Rank
> }
> ```

---

## B4. Статистика и отчёт

### Теория

> [!note] Сырые замеры и перцентиль
> Для учебного теста храним **все** замеры в срезе и в конце сортируем: p95 = элемент с индексом `95% от длины`. Для миллионов замеров так не делают (память) — используют гистограммы, как Prometheus, или HDR Histogram (бонус B6.3).

> [!note] Конкурентный сбор
> Сотни горутин пишут в одну структуру. Срезы и карту защищаем мьютексом, простые счётчики — `atomic.Int64`.

> [!example]- Решение: cmd/loadtest/stats.go
> ```go
> package main
>
> import (
> 	"context"
> 	"fmt"
> 	"log"
> 	"sort"
> 	"strings"
> 	"sync"
> 	"sync/atomic"
> 	"time"
> )
>
> type stats struct {
> 	mu      sync.Mutex
> 	samples map[string][]time.Duration
> 	errors  map[string]int
> 	tables  map[string]bool
> 	online  atomic.Int64
> 	began   time.Time
> }
>
> func newStats() *stats {
> 	return &stats{samples: map[string][]time.Duration{}, errors: map[string]int{}, tables: map[string]bool{}, began: time.Now()}
> }
>
> func (s *stats) observe(name string, d time.Duration) {
> 	s.mu.Lock()
> 	s.samples[name] = append(s.samples[name], d)
> 	s.mu.Unlock()
> }
>
> func (s *stats) fail(reason string) {
> 	s.mu.Lock()
> 	s.errors[reason]++
> 	s.mu.Unlock()
> }
>
> func (s *stats) gameDone(tableID string) {
> 	s.mu.Lock()
> 	s.tables[tableID] = true
> 	s.mu.Unlock()
> }
>
> func (s *stats) active(delta int64) {
> 	s.online.Add(delta)
> }
>
> func (s *stats) progress(ctx context.Context, every time.Duration) {
> 	t := time.NewTicker(every)
> 	defer t.Stop()
> 	for {
> 		select {
> 		case <-ctx.Done():
> 			return
> 		case <-t.C:
> 			s.mu.Lock()
> 			failed := 0
> 			for _, n := range s.errors {
> 				failed += n
> 			}
> 			games := len(s.tables)
> 			s.mu.Unlock()
> 			log.Printf("players %d · finished games %d · errors %d", s.online.Load(), games, failed)
> 		}
> 	}
> }
>
> func (s *stats) report(o options) string {
> 	s.mu.Lock()
> 	defer s.mu.Unlock()
>
> 	var b strings.Builder
> 	elapsed := time.Since(s.began)
> 	fmt.Fprintf(&b, "\n%d players, %s, think %s\n", o.players, elapsed.Round(time.Second), o.think)
> 	fmt.Fprintf(&b, "finished games: %d (%.1f per minute)\n\n", len(s.tables), float64(len(s.tables))/elapsed.Minutes())
>
> 	names := make([]string, 0, len(s.samples))
> 	for name := range s.samples {
> 		names = append(names, name)
> 	}
> 	sort.Strings(names)
> 	fmt.Fprintf(&b, "%-18s %8s %10s %10s %10s %10s\n", "", "count", "p50", "p95", "p99", "max")
> 	for _, name := range names {
> 		d := s.samples[name]
> 		sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
> 		fmt.Fprintf(&b, "%-18s %8d %10s %10s %10s %10s\n", name, len(d), pct(d, 50), pct(d, 95), pct(d, 99), d[len(d)-1].Round(time.Microsecond))
> 	}
>
> 	if len(s.errors) > 0 {
> 		b.WriteString("\nerrors:\n")
> 		for reason, n := range s.errors {
> 			fmt.Fprintf(&b, "  %5d  %s\n", n, reason)
> 		}
> 	}
> 	return b.String()
> }
>
> func pct(sorted []time.Duration, p int) time.Duration {
> 	return sorted[(len(sorted)-1)*p/100].Round(time.Microsecond)
> }
> ```

> [!example]- Решение: cmd/loadtest/main.go
> ```go
> package main
>
> import (
> 	"context"
> 	"flag"
> 	"fmt"
> 	"log"
> 	"os"
> 	"os/signal"
> 	"sync"
> 	"time"
> )
>
> type options struct {
> 	base     string
> 	origin   string
> 	dsn      string
> 	secret   string
> 	prefix   string
> 	players  int
> 	ramp     time.Duration
> 	duration time.Duration
> 	think    time.Duration
> 	stake    int64
> }
>
> func main() {
> 	var o options
> 	flag.StringVar(&o.base, "base", "http://localhost:8080", "server address")
> 	flag.StringVar(&o.origin, "origin", "http://localhost:5173", "Origin header the server allows for sockets")
> 	flag.StringVar(&o.dsn, "dsn", "postgres://postgres:1221@localhost:5433/bito_db?sslmode=disable", "database for seeding players")
> 	flag.StringVar(&o.secret, "secret", envOr("JWT_SECRET", "super-secret-jwt-key-change-in-production"), "JWT secret of the server")
> 	flag.StringVar(&o.prefix, "prefix", "load", "name prefix of test players")
> 	flag.IntVar(&o.players, "players", 50, "virtual players")
> 	flag.DurationVar(&o.ramp, "ramp", 10*time.Second, "time to bring every player in")
> 	flag.DurationVar(&o.duration, "duration", time.Minute, "how long players keep starting new games")
> 	flag.DurationVar(&o.think, "think", 300*time.Millisecond, "pause before each move")
> 	flag.Int64Var(&o.stake, "stake", 100, "stake of every game")
> 	flag.Parse()
>
> 	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
> 	defer stop()
>
> 	users, err := seed(ctx, o)
> 	if err != nil {
> 		log.Fatalf("seed: %v", err)
> 	}
> 	log.Printf("seeded %d players, starting load for %s", len(users), o.duration)
>
> 	stats := newStats()
> 	deadline := time.Now().Add(o.duration)
> 	go stats.progress(ctx, 5*time.Second)
>
> 	var wg sync.WaitGroup
> 	for i, u := range users {
> 		wg.Add(1)
> 		go func() {
> 			defer wg.Done()
> 			delay := o.ramp * time.Duration(i) / time.Duration(len(users))
> 			select {
> 			case <-time.After(delay):
> 			case <-ctx.Done():
> 				return
> 			}
> 			p := &player{user: u, opts: o, stats: stats}
> 			p.run(ctx, deadline)
> 		}()
> 	}
> 	wg.Wait()
> 	fmt.Println(stats.report(o))
> }
>
> func envOr(key, fallback string) string {
> 	if v := os.Getenv(key); v != "" {
> 		return v
> 	}
> 	return fallback
> }
> ```

---

## B5. Прогон и чтение результатов

### Практика

1. `docker compose up -d --build` — сервер, база, Prometheus, Grafana.
2. Открой Grafana `http://localhost:3000`, дашборд «Bito — сервер», период 15 минут.
3. Smoke: `go run ./cmd/loadtest -players 10 -duration 30s`.
4. Load: `go run ./cmd/loadtest -players 200 -ramp 20s -duration 2m`.
5. Stress: увеличивай `-players` (400, 800…) или ставь `-think 0`, пока p95 хода или ошибки не поползут. Смотри, какой график в Grafana меняется первым.

### Что получилось у нас (эталонный прогон)

Машина: один ноутбук на Windows, 16 логических ядер; сервер, Postgres и Redis — в Docker Desktop, генератор — `go run` на той же машине. Это **не** продакшен-цифры, но порядок величин и поведение — настоящие.

> [!example] Load: 200 игроков, пауза 300 мс, 2 минуты
> ```
> 200 players, 2m20s, think 300ms
> finished games: 708 (302.8 per minute)
>
>                       count        p50        p95        p99        max
> move round trip       39581      587µs    1.364ms   12.088ms   44.659ms
> quick match            1416    1.389ms    2.161ms     2.87ms    9.833ms
> socket connect         1416    3.007ms    4.899ms   10.473ms   11.759ms
> wait for opponent      1416    3.904ms  103.448ms  104.393ms  111.663ms
> ```
> На сервере в это время: 100 столов с партией, 200 игроков, ~515 горутин, CPU ~0.1 ядра, обработка хода p50 0.05 мс / p95 0.14 мс / p99 3.3 мс, касса p95 ~4–6 мс, потерь сообщений 0.

> [!example] Stress: 400 игроков, пауза 50 мс, 1 минута
> ```
> 400 players, 1m4s, think 50ms
> finished games: 3908 (3690.2 per minute)
>
>                       count        p50        p95        p99        max
> move round trip      218023    1.069ms    2.235ms   14.019ms   43.193ms
> quick match            7816    1.635ms    2.779ms    3.931ms   16.584ms
> socket connect         7816    2.138ms    3.697ms    4.983ms   20.155ms
> wait for opponent      7816    4.846ms    9.007ms   28.259ms   32.814ms
> ```
> ~3 400 ходов в секунду, ошибок 0. Сервер на пике: CPU ~1.3 ядра из 16, обработка хода p99 ~4.7 мс, касса p95 ~7.7 мс (`lock` и `settle`).
> После прогона проверили базу: у всех 600 ботов **баланс = сумма журнала**, незакрытых партий 0. Экономика под нагрузкой не разъехалась.

> [!danger] Stress: 600 игроков без паузы — и упал **генератор**, а не сервер
> Первая попытка `-players 600 -think 0`:
> ```
> errors:
>   13827  socket: dial tcp [::1]:8080: connectex: Only one usage of each socket address
>          (protocol/network address/port) is normally permitted.
> quick match p95 1.198s, socket connect p50 1.13s, move round trip p99 1.2s
> ```
> При этом сервер спокойно работал: CPU ~3.2 ядра из 16, обработка хода p99 ~12 мс, `ws_send_dropped` = 0.
> Причина — **кончились локальные порты Windows** на машине генератора:
> 1. `http.DefaultClient` держит всего **2** свободных соединения на хост (`MaxIdleConnsPerHost` по умолчанию). 600 горутин постоянно открывали новые TCP-соединения для быстрой игры.
> 2. Каждая партия — новый WebSocket, а закрытое соединение ещё ~2 минуты висит в `TIME_WAIT` и держит порт. `netstat -an | grep -c TIME_WAIT` показал **15 000+**.
>
> Что сделали: свой `http.Client` с `MaxIdleConnsPerHost: 2048` (быстрая игра p95 упала с 1.2 с до 46 мс), дождались, пока `TIME_WAIT` рассосётся, и взяли реалистичную паузу 50 мс — результат выше, без ошибок.
> **Урок:** прежде чем сказать «сервер не держит», проверь, держит ли генератор. Ошибки `connectex`/`dial` — почти всегда сторона клиента.

### Что мы из этого поняли

> [!question]- Почему у «wait for opponent» при 200 игроках p95 ровно ~100 мс, а при 400 — 9 мс?
> Это артефакт **разгона**, а не сервера. `-ramp 20s` на 200 игроков — новый игрок раз в 100 мс. Первый из пары садится и ждёт, пока появится следующий. У 400 игроков за 10 с шаг 25 мс, и пары после первой партии уже собираются мгновенно. Мораль: прежде чем винить сервер, проверь, не сам ли тест создал эту задержку.

> [!question]- Почему обработка хода p95 — 0.14 мс, а p99 — 3 мс?
> 99 ходов из 100 — это чистая логика в памяти. А **последний** ход партии внутри той же горутины вызывает `settle` — расчёт в Postgres (~4–8 мс). Он и даёт хвост. Это видно, если сравнить панель «Обработка хода» с «Касса (база)». Идея на будущее: выносить расчёт из горутины комнаты (бонус B6.7).

> [!question]- Какое узкое место у нас первое?
> Не логика комнат (доли миллисекунды) и не CPU сервера. Первой упирается **касса**: две транзакции в базе на партию, и они стоят в горутине комнаты. На нашей машине раньше неё сдался генератор (порты), поэтому честный предел сервера мы пока не нашли — и это тоже результат.

### Как читать

> [!note] Порядок разбора
> 1. **Ошибки** в отчёте и на панели «Ошибки и потери». Ненулевые — сначала разобраться с ними, задержки потом.
> 2. **move round trip p95/p99** у клиента против **room_handle p95** на сервере. Если сервер обрабатывает ход за доли миллисекунды, а клиент ждёт 50 мс — время уходит не в логику, а в сеть, планировщик или CPU генератора.
> 3. **Касса p95** растёт с числом столов → узкое место — база: пул соединений, блокировки кошельков.
> 4. **Горутины** растут линейно с игроками (по 2 на сокет + 1 на стол) и **возвращаются** после теста. Если не вернулись — утечка.
> 5. **CPU** около числа ядер — предел машины.

### Проверь себя

> [!question]- p95 хода у клиента 40 мс, а `room_handle` p95 — 0.2 мс. Где время?
> Не в логике комнаты. Кандидаты: пауза перед ходом не входит (мы ставим время после неё), значит — доставка: сеть/loopback, планировщик Go при сотнях горутин, загруженный CPU (генератор и сервер на одной машине), запись в сокет у соседей. Дальше смотреть CPU и горутины.

> [!question]- Горутины после теста не вернулись к исходным. Что проверить?
> Комнаты, которые не закрылись (`emptyRoomTTL`, 30 с после конца партии), и клиентов, у которых не сработал `unregister`. Подождать минуту: часть закрывается по таймеру. Если число стоит — снимать `pprof` горутин (бонус B6.4).

---

## B6. Бонусы

### B6.1. Долив фишек ботам ★☆☆
Перед быстрой игрой проверять баланс и звать `POST /api/v1/wallet/top-up`, когда меньше ставки. Тогда долгий soak-тест не упрётся в экономику.

### B6.2. Сценарий входа ★★☆
Отдельный режим: N логинов через `/auth/login`. Сразу упрёшься в rate limiter — и это правильный результат: лимитер работает. Для теста лимита нужен разброс IP или отключение лимита в dev-конфиге.

### B6.3. HDR Histogram ★★☆
Заменить сырые срезы на гистограмму с фиксированной памятью — так тест сможет идти часами.

### B6.4. pprof ★★★
Подключить `net/http/pprof` на порт метрик и снять CPU-профиль сервера во время stress-теста: `go tool pprof http://127.0.0.1:9091/debug/pprof/profile?seconds=20`. Увидишь, на что реально уходит CPU: JSON, запись в сокеты, сборка мусора.

### B6.5. Зрители под нагрузкой ★★☆
Добавить ботов-зрителей (`watch=1`) по 10 на стол: проверка, что рассылка `broadcast` не становится узким местом и `ws_send_dropped` остаётся нулём.

### B6.6. k6 для сравнения ★★☆
Тот же сценарий на [k6](https://k6.io) (JavaScript, есть модуль WebSocket). Хорошо для доклада: «написали свой генератор на Go и сверили с индустриальным инструментом».

### B6.7. Касса вне горутины комнаты ★★★
Расчёт `settle` идёт в базе внутри горутины комнаты и даёт хвост p99 обработки хода. Вынести его в отдельную горутину с ответом через канал (как `rematch`), не потеряв гарантию «расчёт ровно один раз». Сравнить p99 до и после.

### B6.8. Генератор внутри Docker-сети ★★☆
На Windows сокеты к контейнеру идут через прокси Docker Desktop. Собрать `cmd/loadtest` в образ и запустить в той же сети compose (`-base http://server:8080`) — уйдут задержки прокси и проблемы с портами хоста.

---

## Вопросы комиссии

> [!question]- Сколько игроков выдерживает сервер?
> На учебной машине — 400 одновременных ботов, ~3 400 ходов в секунду (по темпу это тысячи живых игроков), p95 хода 2 мс, ноль ошибок, сервер занимал ~1.3 ядра. Предел упёрся в генератор (порты Windows), а не в сервер. Первое узкое место сервера по графикам — касса в базе.

> [!question]- Как вы наблюдаете за сервером?
> Сервер отдаёт метрики Prometheus на отдельном внутреннем порту: столы, игроки, партии, время обработки хода, время кассы, HTTP по маршрутам, потери сообщений, рантайм Go. Prometheus хранит их, Grafana показывает дашборд. Конфигурация лежит в репозитории и поднимается одной командой.

> [!question]- Почему у метрик нет метки с id стола или игрока?
> Это бесконечное множество значений — каждое создаёт отдельный ряд в памяти Prometheus. В метках только ограниченные наборы: маршрут-шаблон, код ответа, тип сообщения из белого списка.

> [!question]- Как нагрузочный тест обходит регистрацию по почте и честно ли это?
> Игроки создаются тем же репозиторием, что и при регистрации, с записью в журнал, а токены подписываются секретом сервера. Мы тестируем игру, не вход; для входа — отдельный сценарий, и там упираемся в лимитер, что и требуется.

## Шпаргалка

| Что | Команда / адрес |
|---|---|
| Метрики сервера | `curl http://127.0.0.1:9091/metrics` |
| Prometheus, цели | `http://localhost:9090/targets` |
| Grafana | `http://localhost:3000` → «Bito — сервер» |
| Smoke | `go run ./cmd/loadtest -players 10 -duration 30s` |
| Load | `go run ./cmd/loadtest -players 200 -ramp 20s -duration 2m` |
| Stress | `go run ./cmd/loadtest -players 400 -think 0 -duration 1m` |

## Словарь

- **Метрика** — именованное число во времени.
- **Scrape** — Prometheus забирает `/metrics`.
- **Кардинальность** — число уникальных рядов (комбинаций меток).
- **Перцентиль p95** — значение, быстрее которого 95% наблюдений.
- **RED / USE** — чек-листы: запросы (rate, errors, duration) и ресурсы (utilization, saturation, errors).
- **Закрытая модель нагрузки** — фиксированное число пользователей, каждый ждёт ответа перед следующим действием.
- **Think time** — пауза пользователя между действиями.

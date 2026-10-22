---
title: Bito — CI на GitHub Actions (учебный план)
tags: [bito, plan, ci, github-actions, devops, learning]
created: 2026-10-21
---

# CI на GitHub Actions: учебный план

> [!abstract] Что строим
> На каждый push в `main` и на каждый pull request GitHub сам поднимает чистую машину и проверяет проект: форматирование Go, `go vet`, юнит-тесты с детектором гонок, **интеграционные тесты на настоящих Postgres и Redis**, сборку фронта с проверкой типов и сборку Docker-образа. Если что-то упало, у PR красный крестик, и вливать его нельзя.
> Связанные заметки: [[features]].

> [!info] Как работать с этой заметкой
> Каждый шаг: **зачем → теория → практика → ловушки → проверь себя → решение → готово, когда**. Решения свёрнуты: сначала напиши сам.
> Ты пишешь всё сам, фронт здесь не нужен — CI только собирает его.

> [!success] Что проверено
> Все команды из workflow прогнаны локально на этом проекте: `gofmt`, `go vet` (с тегом `integration` и без), `go test -race ./...`, интеграционные тесты на Postgres и Redis из docker compose, `npm run build`, сборка образа. Сам файл workflow проверен линтером `actionlint`. Две ловушки из шагов 3 и 4 найдены именно этим прогоном, а не придуманы.

---

## 0. Карта

### Что такое CI и зачем он дипломному проекту

**CI (continuous integration)** — правило «каждое изменение автоматически собирается и тестируется в чистом окружении». Не «у меня на ноуте работает», а «на свежей Ubuntu, из одного только git, всё собралось и прошло».

Что это даёт тебе прямо сейчас:
- нельзя забыть прогнать тесты перед слиянием ветки — их прогонит GitHub;
- ловятся вещи, которые на Windows не видны (переводы строк, регистр в именах файлов, тесты, которые тихо пропускаются);
- на защите можно показать зелёную галочку у PR и историю прогонов — это сильный аргумент «проект поддерживаемый».

### Как устроен GitHub Actions

```
 push / pull_request
        │
        ▼
 .github/workflows/ci.yml          ← workflow: когда запускать и что делать
        │
        ├── job: backend   (своя виртуалка ubuntu-latest)
        │     services: postgres, redis   ← контейнеры рядом с job
        │     steps: checkout → setup-go → gofmt → vet → tests → build
        │
        ├── job: frontend  (своя виртуалка, параллельно)
        │     steps: checkout → setup-node → npm ci → npm run build
        │
        └── job: docker    (ждёт backend: needs)
              steps: checkout → docker build
```

| Понятие | Что это |
|---|---|
| **workflow** | YAML-файл в `.github/workflows/`. Описывает триггеры (`on`) и задачи (`jobs`). |
| **job** | Отдельная чистая машина. Jobs идут **параллельно**, если не связаны `needs`. Файлы между ними не передаются. |
| **step** | Шаг внутри job: либо `run:` (команда shell), либо `uses:` (готовое действие из маркетплейса). |
| **action** | Переиспользуемый шаг, например `actions/checkout@v4` — скачать код, `actions/setup-go@v5` — поставить Go. |
| **service container** | Docker-контейнер, который GitHub поднимает рядом с job: так у тестов появляется настоящий Postgres. |
| **runner** | Машина, на которой выполняется job. `ubuntu-latest` — бесплатная от GitHub. |

### Шаги

1. Workflow-скелет: триггеры, права, отмена старых прогонов.
2. Job `backend`: Go, форматирование, `vet`, юнит-тесты.
3. Интеграционные тесты на Postgres и Redis (и ловушка «тихий skip»).
4. Первый красный прогон: gofmt и перевод строки.
5. Job `frontend`: Node, кэш, `npm ci`, сборка.
6. Job `docker`: образ собирается.
7. Защита ветки `main` и бейдж в README.
8. Бонусы.

---

## Шаг 1. Скелет workflow

### Зачем

Сначала договоримся, **когда** запускать проверки и с какими правами, — это общая шапка для всех jobs.

### Теория

> [!note] Триггеры
> - `push` в `main` — чтобы `main` всегда был проверен.
> - `pull_request` — на каждый PR (в любую ветку). Так проверяется именно то, что ты собираешься влить.
>
> Почему не `push` во все ветки: тогда PR из ветки проверялся бы **дважды** — за push и за pull_request. Это двойная трата минут.

> [!note] concurrency
> Ты запушил, через минуту запушил исправление. Старый прогон уже не нужен. Блок `concurrency` с `cancel-in-progress: true` отменяет прогоны той же ветки, которые ещё идут. Ключ группы — `ci-${{ github.ref }}`: своя очередь у каждой ветки и PR.

> [!note] permissions
> По умолчанию у токена workflow бывают широкие права на запись в репозиторий. Нам нужно только читать код: `permissions: contents: read`. Принцип минимальных прав: если какое-то стороннее действие окажется вредным, оно ничего не сможет испортить.

> [!note] Версии действий
> `actions/checkout@v4` — мажорная версия. Для своих проектов это нормально. В очень строгих командах действия закрепляют по полному SHA коммита, чтобы чужой релиз не мог подменить код (бонус 8.3).

### Практика

Создай `.github/workflows/ci.yml` с `name`, `on`, `concurrency`, `permissions` и пустым `jobs:`.

> [!warning] Ловушки
> - YAML чувствителен к отступам: только пробелы, никаких табов.
> - Папка именно `.github/workflows`, с точкой и во множественном числе.
> - Workflow запускается из **той ветки**, куда пушишь. Пока файл не в ветке, GitHub о нём не знает.

### Проверь себя

> [!question]- Почему `pull_request`, а не `push` для всех веток?
> Push в ветку с открытым PR запустил бы два одинаковых прогона. `pull_request` проверяет ровно то, что вливается, и показывает результат прямо в PR.

> [!question]- Что будет без `concurrency`, если запушить три коммита подряд?
> Три полных прогона, два из которых уже никому не нужны. Минуты тратятся, а результат нужного прогона приходит позже.

### Готово, когда

- [ ] файл лежит в `.github/workflows/ci.yml`
- [ ] можешь объяснить, чем job отличается от step

---

## Шаг 2. Job backend: Go, формат, vet, юнит-тесты

### Зачем

Главная проверка: код компилируется, отформатирован, не содержит подозрительных конструкций и проходит тесты, **в том числе на гонки данных** — у нас горутины комнат, лобби и хаба, гонки там вполне реальны.

### Теория

> [!note] setup-go и версия
> `actions/setup-go@v5` с `go-version-file: go.mod` берёт версию Go прямо из `go.mod`. Одна правда о версии: поменял в `go.mod` — CI подхватил сам. Действие ещё и кэширует модули и сборку по `go.sum`, повторные прогоны быстрее.

> [!note] gofmt как проверка
> `gofmt -l` печатает файлы, которые **не** отформатированы, и ничего не меняет. Пустой вывод — всё хорошо. Если вывод есть, шаг завершаем с `exit 1`, чтобы job упал. Именно так ловят «забыл gofmt».

> [!note] go vet и теги сборки
> `go vet` — статический анализ из коробки: неверные форматы в `Printf`, копирование мьютексов, недостижимый код. Файлы с `//go:build integration` по умолчанию **не компилируются**, поэтому vet их не видит. Отдельный запуск с `-tags integration` проверяет и их.

> [!note] -race и -count=1
> - `-race` включает детектор гонок. Тесты медленнее в 2–10 раз, но ловят одновременный доступ к памяти без синхронизации. На Linux раннере он работает из коробки.
> - `-count=1` отключает кэш результатов тестов. В CI кэша и так нет, но флаг делает намерение явным и одинаковым с локальным запуском.

### Практика

Job `backend` на `ubuntu-latest` с `timeout-minutes: 15`. Шаги:
1. `actions/checkout@v4`
2. `actions/setup-go@v5` с `go-version-file: go.mod`
3. `gofmt` по `./cmd ./internal ./migrations`
4. `go vet ./... && go vet -tags integration ./...`
5. `go test -race -count=1 ./...`
6. `go build -o /dev/null ./cmd/server`

> [!warning] Ловушки
> - `timeout-minutes` обязателен: зависший тест без него будет крутиться до 6 часов и съест минуты.
> - Не пиши `gofmt -l .` — попадёт и `web/node_modules`, если там вдруг окажутся `.go`-файлы, и вообще всё лишнее. Перечисли свои папки.

### Проверь себя

> [!question]- Почему отдельный шаг сборки, если тесты всё равно компилируют код?
> Тесты компилируют пакеты с тестами. `cmd/server` и `cmd/loadtest` тестов не имеют, ошибка в `main` прошла бы незамеченной. Сборка бинаря — дешёвая страховка.

> [!question]- Что найдёт `-race`, чего не найдут обычные тесты?
> Два горутины пишут и читают одно поле без мьютекса или канала. Обычный тест чаще всего проходит «случайно», а детектор видит сам факт одновременного доступа и падает с трассой обоих мест.

### Готово, когда

- [ ] job `backend` описан до шага сборки
- [ ] можешь объяснить, зачем два запуска `go vet`

---

## Шаг 3. Интеграционные тесты на настоящих Postgres и Redis

### Зачем

Самая ценная часть проекта — экономика: блокировка ставок, расчёт, журнал `wallet_transactions`, покупки. Её проверяют интеграционные тесты (`//go:build integration`) на живой базе. Если CI их не гоняет, он проверяет только половину.

### Теория

> [!note] services
> В job можно объявить `services:` — GitHub поднимет контейнеры `postgres:16-alpine` и `redis:7-alpine` и пробросит порты на `localhost` раннера. С `options: --health-cmd ...` job **ждёт**, пока база ответит на `pg_isready`, и только потом начинает шаги. Без healthcheck тесты могут стартовать раньше, чем Postgres готов принимать соединения.

> [!note] Как тесты находят базу
> Тесты уже читают адреса из окружения: `BITO_TEST_ADMIN_DSN` (Postgres, по умолчанию `localhost:5433` из docker compose) и `BITO_TEST_REDIS_ADDR`. В CI порт Postgres — `5432`, пароль — тот, что задан в `services`. Значит, достаточно задать `env:` на уровне job.

> [!danger] Ловушка «тихий skip» — самая важная в этой заметке
> Посмотри на `setupTestDB` и `setupRedis`: если база недоступна, они делают `t.Skipf(...)`. Локально это удобно — забыл поднять docker, тесты не падают, а пропускаются.
> Но в CI это значит: опечатка в DSN, сервис не поднялся, порт не тот — и **все интеграционные тесты молча пропускаются, а job зелёный**. Мы это проверили на нашем проекте:
> ```
> BITO_TEST_ADMIN_DSN=...:5999... go test -tags integration ./internal/repository/postgres/
> ok  bito/internal/repository/postgres  1.082s     ← зелёный, хотя ничего не проверено
> ```
> Решение: переменная `BITO_REQUIRE_INTEGRATION=1`. Если она стоит, недоступная база — это **провал** (`t.Fatalf`), а не пропуск. Локально её нет — поведение прежнее. В CI она есть — зелёный значит «правда проверено».

### Практика

1. В `wallet_integration_test.go` (`setupTestDB`) и `auth_integration_test.go` (`setupRedis`) перед `t.Skipf` добавь проверку `os.Getenv("BITO_REQUIRE_INTEGRATION") == "1"` → `t.Fatalf`.
2. В job `backend`: блок `services` с `postgres` и `redis` и healthcheck'ами.
3. `env` на уровне job: `BITO_TEST_ADMIN_DSN`, `BITO_TEST_REDIS_ADDR`, `BITO_REQUIRE_INTEGRATION: "1"`.
4. Шаг `go test -race -count=1 -tags integration ./internal/repository/...`.

> [!warning] Ловушки
> - Значения в `env` YAML лучше брать в кавычки: `"1"`, иначе YAML может понять их как число или булево.
> - DSN к базе `postgres`, а не `bito_db`: `setupTestDB` подключается как админ, **создаёт** отдельную тестовую базу и удаляет её. Так тесты не трогают рабочие данные.
> - Пароль в `services.postgres.env` и в DSN должны совпадать. Это не секрет — контейнер живёт пару минут и недоступен снаружи, поэтому GitHub Secrets здесь не нужны.

### Проверь себя

> [!question]- Почему просто не убрать `t.Skipf` и всегда падать без базы?
> Тогда `go test -tags integration ./...` на ноутбуке без docker падал бы стеной ошибок, хотя ты, может, правишь фронт. Переменная разделяет два режима: дома мягко, в CI строго.

> [!question]- Что будет, если убрать healthcheck у Postgres?
> Контейнер стартует, но Postgres внутри ещё инициализируется несколько секунд. Первый тест получит «connection refused» — а с `BITO_REQUIRE_INTEGRATION=1` это провал. Будут «плавающие» красные прогоны.

> [!example]- Решение: проверка в тестах
> В `internal/repository/postgres/wallet_integration_test.go`:
> ```go
> 	admin, err := pgx.Connect(ctx, adminDSN())
> 	if err != nil {
> 		if os.Getenv("BITO_REQUIRE_INTEGRATION") == "1" {
> 			t.Fatalf("postgres is not reachable: %v", err)
> 		}
> 		t.Skipf("postgres is not reachable (%v); start docker compose to run integration tests", err)
> 	}
> ```
> В `internal/repository/redis/auth_integration_test.go`:
> ```go
> 	if err := client.Ping(ctx).Err(); err != nil {
> 		_ = client.Close()
> 		if os.Getenv("BITO_REQUIRE_INTEGRATION") == "1" {
> 			t.Fatalf("redis is not reachable: %v", err)
> 		}
> 		t.Skipf("redis is not reachable (%v); start docker compose to run integration tests", err)
> 	}
> ```

### Готово, когда

- [ ] локально без docker: тесты пропускаются; с `BITO_REQUIRE_INTEGRATION=1` — падают
- [ ] локально с docker: `BITO_REQUIRE_INTEGRATION=1 go test -tags integration ./internal/repository/...` зелёный

---

## Шаг 4. Первый красный прогон: gofmt и переводы строк

### Зачем

Первый прогон CI почти всегда красный. Это нормально и полезно — значит, проверка работает. У нас он гарантированно упадёт на `gofmt`, и вот почему.

### Теория

> [!note] Что нашёл прогон
> `gofmt -l` выдал `cmd/server/main.go`. Причина — в конце файла нет перевода строки, а gofmt требует его. Локально никто не замечал, потому что никто не запускал `gofmt -l` по всему проекту.

> [!note] CRLF и LF
> На Windows git с `core.autocrlf=true` при checkout превращает `\n` в `\r\n`. Если запустить `gofmt -l` прямо в рабочей папке на Windows, он может ругаться на **все** файлы. В репозитории у нас `.gitattributes` с `* text=auto eol=lf`, значит в git лежит LF, и на Linux-раннере всё честно. Вывод: проверку форматирования доверяй CI (или WSL), а локально просто делай `gofmt -w` перед коммитом.

### Практика

1. Запусти `gofmt -l ./cmd ./internal ./migrations` — посмотри, что выдаст.
2. `gofmt -w cmd/server/main.go` и закоммить.
3. Запушь ветку, открой PR — смотри вкладку **Actions**, разверни упавший шаг, прочитай лог.

> [!question]- Почему лучше, чтобы CI падал на формате, чем молча форматировал сам и коммитил?
> Коммит от бота — это изменения, которых ты не видел, и права на запись для workflow. Проще и безопаснее: CI говорит «вот файлы», ты запускаешь `gofmt -w` у себя.

### Готово, когда

- [ ] видел красный прогон и нашёл в логе, какой шаг и почему упал
- [ ] после исправления все jobs зелёные

---

## Шаг 5. Job frontend: Node, кэш, `npm ci`, сборка

### Зачем

`npm run build` у нас — это `vue-tsc -b && vite build`: сначала **проверка типов TypeScript**, потом сборка. Ошибка типов в компоненте ломает билд — CI поймает её до слияния.

### Теория

> [!note] npm ci против npm install
> `npm ci` ставит **ровно** то, что записано в `package-lock.json`, и падает, если lock расходится с `package.json`. `npm install` может молча обновить lock. В CI нужна воспроизводимость — значит `npm ci`.

> [!note] Кэш
> `actions/setup-node@v4` с `cache: npm` и `cache-dependency-path: web/package-lock.json` кэширует скачанные пакеты между прогонами. Путь к lock обязателен: он лежит не в корне, а в `web/`.

> [!note] working-directory
> `defaults.run.working-directory: web` — все `run:` шаги этой job выполняются в `web/`, не надо писать `cd web &&` в каждой команде.

### Практика

Job `frontend` на `ubuntu-latest`, `timeout-minutes: 10`, `defaults.run.working-directory: web`. Шаги: checkout, setup-node (Node 22, кэш npm), `npm ci`, `npm run build`.

> [!warning] Ловушки
> - `working-directory` действует только на `run:`, не на `uses:` — поэтому путь к lock в `setup-node` пишем полностью.
> - Linux чувствителен к регистру имён файлов. `import Foo from './foo.vue'` при файле `Foo.vue` на Windows работает, а в CI упадёт. Если сборка падает только в CI — первым делом проверь регистр импортов.

### Готово, когда

- [ ] job `frontend` зелёная
- [ ] можешь объяснить разницу `npm ci` и `npm install`

---

## Шаг 6. Job docker: образ собирается

### Зачем

Сервер на защите (и в `docker compose`) запускается из образа. Сломанный `Dockerfile` — сломанный запуск, хотя все тесты зелёные.

### Теория

> [!note] needs
> `needs: [backend]` — job `docker` стартует только после зелёного `backend`. Нет смысла собирать образ из кода, который не прошёл тесты. А `frontend` идёт параллельно с `backend` — так общий прогон быстрее.

> [!note] Только собрать
> Мы **не публикуем** образ (это CD — бонус 8.2). Цель — убедиться, что `docker build .` проходит на чистой машине.

### Практика

Job `docker`, `needs: [backend]`, шаги: checkout и `docker build --tag bito-server:ci .`.

> [!example]- Решение: весь `.github/workflows/ci.yml`
> ```yaml
> name: CI
>
> on:
>   push:
>     branches: [main]
>   pull_request:
>
> concurrency:
>   group: ci-${{ github.ref }}
>   cancel-in-progress: true
>
> permissions:
>   contents: read
>
> jobs:
>   backend:
>     name: Go — vet, unit and integration tests
>     runs-on: ubuntu-latest
>     timeout-minutes: 15
>
>     services:
>       postgres:
>         image: postgres:16-alpine
>         env:
>           POSTGRES_USER: postgres
>           POSTGRES_PASSWORD: ci
>           POSTGRES_DB: bito_db
>         ports:
>           - 5432:5432
>         options: >-
>           --health-cmd "pg_isready -U postgres"
>           --health-interval 5s
>           --health-timeout 5s
>           --health-retries 10
>       redis:
>         image: redis:7-alpine
>         ports:
>           - 6379:6379
>         options: >-
>           --health-cmd "redis-cli ping"
>           --health-interval 5s
>           --health-timeout 5s
>           --health-retries 10
>
>     env:
>       BITO_TEST_ADMIN_DSN: postgres://postgres:ci@localhost:5432/postgres?sslmode=disable
>       BITO_TEST_REDIS_ADDR: localhost:6379
>       BITO_REQUIRE_INTEGRATION: "1"
>
>     steps:
>       - uses: actions/checkout@v4
>
>       - uses: actions/setup-go@v5
>         with:
>           go-version-file: go.mod
>
>       - name: gofmt
>         run: |
>           unformatted=$(gofmt -l ./cmd ./internal ./migrations)
>           if [ -n "$unformatted" ]; then
>             echo "Run gofmt -w on:"
>             echo "$unformatted"
>             exit 1
>           fi
>
>       - name: go vet
>         run: go vet ./... && go vet -tags integration ./...
>
>       - name: Unit tests
>         run: go test -race -count=1 ./...
>
>       - name: Integration tests
>         run: go test -race -count=1 -tags integration ./internal/repository/...
>
>       - name: Build server
>         run: go build -o /dev/null ./cmd/server
>
>   frontend:
>     name: Web — types and build
>     runs-on: ubuntu-latest
>     timeout-minutes: 10
>     defaults:
>       run:
>         working-directory: web
>
>     steps:
>       - uses: actions/checkout@v4
>
>       - uses: actions/setup-node@v4
>         with:
>           node-version: 22
>           cache: npm
>           cache-dependency-path: web/package-lock.json
>
>       - run: npm ci
>
>       - run: npm run build
>
>   docker:
>     name: Docker image builds
>     runs-on: ubuntu-latest
>     timeout-minutes: 15
>     needs: [backend]
>
>     steps:
>       - uses: actions/checkout@v4
>
>       - name: Build image
>         run: docker build --tag bito-server:ci .
> ```

### Готово, когда

- [ ] в PR три зелёные проверки: backend, frontend, docker
- [ ] `actionlint` (бонус 8.1) не находит ошибок

---

## Шаг 7. Защита ветки main и бейдж

### Зачем

CI бесполезен, если красный PR всё равно можно влить. Защита ветки делает зелёные проверки **обязательными**.

### Практика

1. GitHub → репозиторий → **Settings → Branches → Add branch ruleset** (или Branch protection rule) для `main`.
2. Включи **Require status checks to pass** и выбери три проверки: `Go — vet, unit and integration tests`, `Web — types and build`, `Docker image builds`. Они появятся в списке после первого прогона.
3. Включи **Require a pull request before merging** — в `main` только через PR.
4. Бейдж в `README.md`:
   ```markdown
   ![CI](https://github.com/D4cLoves/Bito/actions/workflows/ci.yml/badge.svg)
   ```

> [!question]- Почему проверки нельзя выбрать в настройках сразу после создания файла?
> GitHub узнаёт имена проверок только после того, как workflow хотя бы раз отработал. Сначала прогон, потом настройка защиты.

### Готово, когда

- [ ] красный PR нельзя влить кнопкой Merge
- [ ] бейдж в README показывает статус `main`

---

## Шаг 8. Бонусы

### 8.1. actionlint ★☆☆
Линтер для workflow-файлов: ловит опечатки в ключах, неверные выражения `${{ }}`, ошибки в shell. Локально: `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7`. Можно добавить отдельным шагом в CI.

### 8.2. Публикация образа (CD) ★★☆
Отдельный workflow на push в `main`: логин в GitHub Container Registry (`ghcr.io`) через `GITHUB_TOKEN`, `docker/build-push-action`, тег — SHA коммита. Понадобится `permissions: packages: write` — только для этой job.

### 8.3. Закрепить действия по SHA ★☆☆
`uses: actions/checkout@<полный sha> # v4`. Защищает от подмены релиза действия. Обновлять удобно через Dependabot.

### 8.4. golangci-lint ★★☆
Набор линтеров (`errcheck`, `staticcheck`, `ineffassign`…). Начни с небольшого набора в `.golangci.yml`, иначе первый прогон выдаст сотни замечаний.

### 8.5. Матрица ★☆☆
`strategy.matrix.go: ['1.27.x', 'stable']` — прогон на двух версиях Go. Для диплома не обязательно, но полезно знать, как это выглядит.

### 8.6. Нагрузочный smoke в CI ★★★
Идея на будущее: короткий прогон `cmd/loadtest` на 10 игроков и 30 секунд в отдельной job с `docker compose up`. Падать, если p95 хода выше порога или есть ошибки.

---

## Вопросы комиссии

> [!question]- Что проверяется автоматически при каждом изменении?
> Форматирование и статический анализ Go, юнит-тесты с детектором гонок, интеграционные тесты экономики на настоящих Postgres и Redis, проверка типов и сборка фронта, сборка Docker-образа. Влить в `main` можно только при зелёных проверках.

> [!question]- Как вы убедились, что интеграционные тесты действительно запускаются в CI, а не пропускаются?
> Тесты пропускаются без базы только локально. В CI стоит `BITO_REQUIRE_INTEGRATION=1`, и недоступная база роняет прогон. Мы проверили оба режима руками.

> [!question]- Почему секреты не нужны в этом workflow?
> База и Redis — временные контейнеры внутри прогона, недоступные снаружи. Пароль у них одноразовый. Секреты понадобятся только для публикации образа или деплоя.

## Шпаргалка

| Что | Где |
|---|---|
| Workflow | `.github/workflows/ci.yml` |
| Строгий режим интеграционных тестов | `BITO_REQUIRE_INTEGRATION=1` |
| Локально как в CI | `gofmt -l ./cmd ./internal ./migrations`, `go vet ./... && go vet -tags integration ./...`, `go test -race -count=1 ./...`, `BITO_REQUIRE_INTEGRATION=1 go test -race -tags integration ./internal/repository/...`, `cd web && npm ci && npm run build`, `docker build .` |
| Логи прогона | вкладка **Actions** или **Checks** в PR |

## Словарь

- **CI** — автоматическая сборка и тесты на каждое изменение.
- **CD** — автоматическая доставка: публикация образа, деплой.
- **Runner** — машина, на которой GitHub выполняет job.
- **Service container** — вспомогательный контейнер (база, кэш) рядом с job.
- **Status check** — результат job, который GitHub показывает в PR и может требовать для слияния.
- **Flaky test** — тест, который то проходит, то падает без изменений в коде.

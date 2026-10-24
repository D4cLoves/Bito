<div align="center">

<img src="docs/banner.svg" alt="Bito — онлайн-дурак в реальном времени на виртуальные фишки" width="100%">

<br>
<br>

[![CI](https://img.shields.io/github/actions/workflow/status/D4cLoves/Bito/ci.yml?branch=main&style=flat-square&label=CI)](https://github.com/D4cLoves/Bito/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/D4cLoves/Bito?style=flat-square&color=d4a85a&label=версия)](https://github.com/D4cLoves/Bito/releases/latest)
![Go](https://img.shields.io/badge/Go-1.27-00ADD8?style=flat-square&logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3-42b883?style=flat-square&logo=vuedotjs&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-3178C6?style=flat-square&logo=typescript&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=flat-square&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat-square&logo=docker&logoColor=white)

**[Возможности](#возможности)** · **[Скриншоты](#скриншоты)** · **[Как устроено](#как-устроено)** · **[Запуск](#запуск)** · **[Тесты](#тесты)**

<br>

<img src="docs/screenshots/table.jpg" alt="Партия в переводного: рука игрока, колода с козырем, таймер хода и чат стола" width="100%">

<sub>Партия в переводного: своя рука, колода с козырем, таймер хода и чат стола</sub>

<br>
<br>

<table>
  <tr>
    <td align="center"><h3>&nbsp;&nbsp;2–6&nbsp;&nbsp;</h3><sub>игроков&nbsp;за&nbsp;столом</sub></td>
    <td align="center"><h3>&nbsp;&nbsp;24&nbsp;·&nbsp;36&nbsp;·&nbsp;52&nbsp;&nbsp;</h3><sub>карты&nbsp;в&nbsp;колоде</sub></td>
    <td align="center"><h3>&nbsp;&nbsp;20&nbsp;с&nbsp;&nbsp;</h3><sub>на&nbsp;ход,&nbsp;потом&nbsp;ходит&nbsp;сервер</sub></td>
    <td align="center"><h3>&nbsp;&nbsp;51&nbsp;&nbsp;</h3><sub>предмет&nbsp;в&nbsp;магазине</sub></td>
    <td align="center"><h3>&nbsp;&nbsp;246&nbsp;&nbsp;</h3><sub>тестов&nbsp;на&nbsp;Go</sub></td>
  </tr>
</table>

</div>

## Возможности

<table>
  <tr>
    <th width="33%">🃏 Игра</th>
    <th width="33%">🪙 Вокруг стола</th>
    <th width="33%">🎨 Оформление</th>
  </tr>
  <tr valign="top">
    <td>

- Подкидной и переводной дурак
- Колоды на 24, 36 и 52 карты, тасовка через `crypto/rand`
- Быстрая игра 1 на 1 с подбором соперника
- Столы-компании на 3–6 игроков
- Таймер хода: если игрок молчит, сервер ходит за него
- 30 секунд на возвращение при обрыве связи

</td>
    <td>

- Ставки в виртуальных фишках, выплаты по местам, комиссия 5%
- Кошелёк с графиком баланса и журналом операций
- Друзья со статусом «в лобби / за столом / не в сети»
- Закрытые столы по приглашению или ссылке, реванш одной кнопкой
- Режим зрителя без доступа к рукам
- Чат за столом с быстрыми фразами и эмодзи

</td>
    <td>

- Магазин за фишки: 7 тем, 9 сукон, 9 рубашек, 26 аватаров
- Звуки карт и музыка
- Сигнал «ваш ход» в свёрнутой вкладке
- Мобильная вёрстка с нижним меню
- Регистрация по коду из письма
- История партий и профили игроков

</td>
  </tr>
</table>

## Скриншоты

<table>
  <tr>
    <td width="50%"><img src="docs/screenshots/landing.jpg" alt="Лендинг с формой входа"></td>
    <td width="50%"><img src="docs/screenshots/lobby.jpg" alt="Лобби: быстрая игра один на один, карточка игрока и столы-компании"></td>
  </tr>
  <tr>
    <td align="center"><sub><b>Лендинг</b> · правила, режимы и вход</sub></td>
    <td align="center"><sub><b>Лобби</b> · быстрая игра, профиль, столы</sub></td>
  </tr>
  <tr>
    <td width="50%"><img src="docs/screenshots/wallet.jpg" alt="Кошелёк: график баланса и журнал операций"></td>
    <td width="50%"><img src="docs/screenshots/shop.jpg" alt="Магазин аватаров, рубашек, сукон и тем"></td>
  </tr>
  <tr>
    <td align="center"><sub><b>Кошелёк</b> · график баланса и журнал</sub></td>
    <td align="center"><sub><b>Магазин</b> · оформление за фишки</sub></td>
  </tr>
</table>

<p align="center">
  <img src="docs/screenshots/mobile.jpg" alt="Лобби на телефоне" width="280"><br>
  <sub><b>На телефоне</b> · то же лобби с нижним меню</sub>
</p>

## Как устроено

```mermaid
flowchart LR
    B["Браузер<br/>Vue 3 + TypeScript"]
    subgraph S["Go-сервер"]
        H["HTTP · chi"]
        HUB["Hub"]
        R["Room<br/>горутина на стол"]
        L["Lobby<br/>горутина лобби"]
        G["game · economy<br/>чистая логика"]
    end
    PG[("PostgreSQL<br/>игроки, партии, деньги")]
    RD[("Redis<br/>коды, сессии, лимиты")]

    B -- "REST /api/v1" --> H
    B <-- "WebSocket /ws, /ws/lobby" --> HUB
    HUB --> R
    HUB --> L
    R --> G
    H --> PG
    H --> RD
    R --> PG
```

Так проходит один ход:

```mermaid
sequenceDiagram
    autonumber
    participant A as Атакующий
    participant R as Room (горутина стола)
    participant G as game
    participant D as Защитник

    A->>R: attack, карта 6♦
    R->>G: проверить и применить ход
    G-->>R: новое состояние стола
    R-->>A: game_state со своей рукой
    R-->>D: game_state со своей рукой
    Note over R: таймер защитника 20 с
```

| Принцип | Как сделано |
|---|---|
| **Один стол — одна горутина** | Сообщения игроков приходят в `Room` через канал, состоянием партии владеет только она, поэтому правилам не нужны блокировки |
| **Сервер главный** | Клиент шлёт только намерение (`attack`, `defend`, `take`, `pass`, `transfer`, `surrender`), допустимость проверяет сервер. Каждый игрок получает только свою руку |
| **Деньги в журнале** | Любое движение фишек — строка в `wallet_transactions` с `balance_after`. `CHECK (balance >= 0)`, атомарное списание ставки, расчёт партии ровно один раз |
| **Две базы** | PostgreSQL для того, что нельзя терять, Redis для того, что должно истечь само |

## Стек

| | |
|---|---|
| **Сервер** | Go 1.27, chi, gorilla/websocket, pgx v5, go-redis v9, golang-jwt v5, bcrypt, cleanenv |
| **Клиент** | Vue 3, TypeScript, Vite, Pinia, Vue Router, Tailwind CSS 4, anime.js, motion-v, Web Audio API |
| **Данные** | PostgreSQL 16 (11 таблиц, миграции встроены в бинарник), Redis 7 |
| **Инфраструктура** | Docker Compose, Mailpit, GitHub Actions, образ сервера в ghcr.io |

## Запуск

Понадобятся Docker и Node.js 22, для запуска сервера без Docker ещё Go 1.27.

### Быстрый старт

Сервер, PostgreSQL, Redis и Mailpit поднимаются в Docker, клиент — dev-сервером Vite:

```bash
git clone https://github.com/D4cLoves/Bito.git
cd Bito
docker compose up -d --build
npm --prefix web ci
npm --prefix web run dev
```

Откройте http://localhost:5173 и зарегистрируйтесь. Письмо с кодом придёт в Mailpit: http://localhost:8025

### Для разработки

Инфраструктура в Docker, сервер запускается из исходников:

```bash
cp .env.example .env
make up        # PostgreSQL, Redis и Mailpit
make run       # сервер на :8080, миграции применяются при старте
make web       # клиент на http://localhost:5173
```

| Порт | Что там |
|---|---|
| `5173` | клиент |
| `8080` | REST API и WebSocket |
| `8025` | Mailpit, письма с кодами |
| `5433` | PostgreSQL |
| `6379` | Redis |

<details>
<summary><b>Настоящая почта вместо Mailpit</b></summary>

<br>

Укажите в `.env` SMTP своего ящика и пароль приложения. Например, для mail.ru:

```env
SMTP_HOST=smtp.mail.ru
SMTP_PORT=465
SMTP_USER=you@mail.ru
SMTP_PASSWORD=пароль-для-внешнего-приложения
SMTP_FROM=you@mail.ru
```

Порт 465 шифрует соединение сразу, 587 переходит на TLS через STARTTLS. Эти настройки подхватывают и `make run`, и `docker compose`.

</details>

## Тесты

```bash
make test                     # все тесты на Go
make test-race                # то же с детектором гонок
npm --prefix web run build    # проверка типов и сборка клиента
```

<details>
<summary><b>Интеграционные тесты на настоящих PostgreSQL и Redis</b></summary>

<br>

Поднимите инфраструктуру через `make up` и запустите:

```bash
BITO_TEST_ADMIN_DSN="postgres://postgres:1221@localhost:5433/postgres?sslmode=disable" \
BITO_TEST_REDIS_ADDR="localhost:6379" \
go test -tags integration ./...
```

</details>

> [!NOTE]
> CI на каждый pull request и push в `main` проверяет форматирование, гоняет тесты с детектором гонок на настоящих PostgreSQL и Redis, собирает клиент и Docker-образ. После слияния в `main` образ сервера публикуется в `ghcr.io/d4cloves/bito`.

## Структура

```
cmd/
  server/              точка входа
  cli/                 консольная партия для ручной проверки движка
internal/
  game/                правила дурака: колода, стол, ходы, без сети и БД
  economy/             расчёт выплат и каталог магазина
  delivery/http/       REST API, авторизация, лимиты
  delivery/ws/         Hub, комнаты, лобби, реванш, зрители
  service/             регистрация, профиль, кошелёк, друзья
  repository/          PostgreSQL и Redis
  pkg/                 JWT и почта
migrations/            SQL-миграции
web/                   клиент на Vue 3
docs/                  описание фич и планы реализации
```

## Документация

- [docs/features.md](docs/features.md) — все фичи, приоритеты и что сделано
- [docs/plans](docs/plans) — как устроены экономика, лобби, друзья, приглашения, зрители и CI

<br>

<div align="center">
<sub>Дипломный проект · Компьютерная академия STEP · 2026</sub>
</div>

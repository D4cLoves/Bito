# Bito: Durak Online 🃏

Real-time multiplayer card game «Durak» (Fool Online) with ELO rating, in-game economy, lobbies, chat, and friend system.

## 🚀 Tech Stack

- **Backend:** Go 1.27, WebSockets, In-Memory Game Engine (goroutines, channels, sync)
- **Database & Cache:** PostgreSQL, Redis
- **Mail:** Mailpit (local SMTP)
- **Frontend:** Vue 3 (Composition API), TypeScript, Pinia, Vite, Tailwind CSS
- **Infrastructure:** Docker & Docker Compose

## 🛠 Project Structure

```
bito/
├── cmd/server/            # Application entrypoint
├── internal/
│   ├── config/            # Environment & application configuration
│   ├── game/              # Isolated Durak game engine & state machine
│   ├── lobby/             # Room management and matchmaking
│   ├── transport/         # WebSockets and HTTP REST API
│   ├── service/           # Business logic (wallet, ELO, stats)
│   ├── repository/        # Database storage layer (PostgreSQL)
│   └── models/            # Domain entities
├── migrations/            # Database migrations
├── web/                   # Vue 3 frontend
└── docker-compose.yml     # Local services (PostgreSQL, Redis, Mailpit)
```

## 🏁 Quick Start

### 1. Run infrastructure
```bash
docker-compose up -d
```

### 2. Run backend server
```bash
go run cmd/server/main.go
```

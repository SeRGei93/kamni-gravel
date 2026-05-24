# AGENTS.md

> Project map for AI agents. Keep this file up-to-date as the project evolves.

## Project Overview

Gravel Bot is a Telegram bot and admin dashboard for organizing cycling races and gravel events. The backend is a Go DDD/Clean Architecture service with PostgreSQL persistence; the frontend is a Next.js TailAdmin-style dashboard.

## Canonical Rules

Use these project architecture and workflow boundaries:

- Domain imports nothing from application or infrastructure.
- Application imports domain only.
- Infrastructure owns adapters for PostgreSQL, HTTP, Telegram, and migrations.
- Use native SQL, not an ORM.
- Keep handlers thin and put business logic in commands/queries.
- Inject dependencies through constructors.
- Put `context.Context` first for request-scoped and IO functions.
- Do not add ad hoc Markdown reports or instruction files unless the user explicitly asks.
- Communicate with the user in Russian unless they explicitly ask for another language.

## Tech Stack

- **Backend:** Go 1.24, Chi, Telegram Bot API
- **Database:** PostgreSQL 16
- **Migrations:** SQL files in `backend/internal/infrastructure/migrations`
- **Frontend:** Next.js 16, React 19, TypeScript, Tailwind CSS v4
- **Runtime:** Docker Compose

## Project Structure

```text
.
├── backend/                         # Go backend, Telegram bot, API, migrations
│   ├── cmd/
│   │   ├── api/                     # REST API entry point
│   │   ├── bot/                     # Telegram bot entry point
│   │   └── migrate/                 # migration entry point
│   ├── docs/                        # Swagger UI/static API docs
│   └── internal/
│       ├── application/             # commands, queries, DTOs
│       ├── config/                  # environment-backed config
│       ├── domain/                  # entities, value objects, repository interfaces
│       ├── infrastructure/          # HTTP, Telegram, Postgres, migrations
│       └── pkg/                     # internal shared packages
├── frontend/                        # Next.js admin dashboard
│   ├── public/                      # static assets
│   └── src/
│       ├── api/                     # frontend API clients
│       ├── app/                     # Next.js App Router pages/layouts
│       ├── components/              # dashboard and form components
│       ├── context/                 # auth, sidebar, theme contexts
│       ├── hooks/                   # frontend hooks
│       ├── layout/                  # app shell components
│       ├── types/                   # TypeScript types
│       └── utils/                   # frontend helpers
├── .ai-factory/                     # AI Factory project context
├── docker-compose.yml               # local full-system runtime
├── env.example                      # environment template
├── Makefile                         # common development commands
└── README.md                        # project landing page
```

## Key Entry Points

| File | Purpose |
|------|---------|
| `backend/cmd/api/main.go` | Starts the HTTP API server and wires repositories. |
| `backend/cmd/bot/main.go` | Starts the Telegram bot and wires repositories. |
| `backend/cmd/migrate/main.go` | Runs database migrations. |
| `backend/internal/config/main.go` | Loads and validates environment configuration. |
| `backend/internal/infrastructure/http/server.go` | Builds the HTTP server. |
| `backend/internal/infrastructure/telegram/bot.go` | Builds the Telegram bot adapter. |
| `frontend/src/app/layout.tsx` | Root Next.js layout. |
| `frontend/src/app/(dashboard)/layout.tsx` | Dashboard layout. |
| `frontend/src/api/client.ts` | Frontend API client foundation. |
| `docker-compose.yml` | PostgreSQL, migration, API, bot, and frontend services. |

## Documentation

| Document | Path | Description |
|----------|------|-------------|
| README | `README.md` | Project overview, runtime, commands, ports. |
| API docs | `backend/docs/swagger.yaml` | Swagger/OpenAPI specification. |
| Description | `.ai-factory/DESCRIPTION.md` | AI Factory project specification. |
| Architecture | `.ai-factory/ARCHITECTURE.md` | Codex-readable architecture rules. |

## Verification Commands

| Command | Purpose |
|---------|---------|
| `make test` | Run Go backend tests. |
| `cd backend && go test ./...` | Backend test command. |
| `cd frontend && npm run lint` | Frontend lint command. |
| `cd frontend && npm run build` | Frontend production build. |
| `make docker-up` | Start the Docker Compose runtime. |
| `make docker-down` | Stop Docker Compose services. |
| `make docker-logs` | Follow Docker Compose logs. |

## AI Context Files

| File | Purpose |
|------|---------|
| `AGENTS.md` | This file: project structure and operating map. |
| `.ai-factory/DESCRIPTION.md` | Project specification and stack summary. |
| `.ai-factory/ARCHITECTURE.md` | Architecture boundaries and workflow guidance. |
| `.agents/skills/` | Project skills installed through `skills.sh`. |
| `skills-lock.json` | Installed external skill lock file. |
| `.mcp.json` | Project MCP server configuration. |

## Installed Project Skills

| Skill | Purpose | Security scan |
|-------|---------|---------------|
| `golang-pro` | Go implementation, testing, interfaces, concurrency, project structure. | Clean |
| `nextjs-app-router-patterns` | Next.js App Router and React Server Component patterns. | Clean |

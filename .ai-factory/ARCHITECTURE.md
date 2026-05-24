# Gravel Bot Architecture

## Source Of Truth

The original Cursor rule file, `.cursor/rules/rules.mdc`, is the canonical architecture rule source. This document mirrors those rules for Codex and AI Factory workflows.

## Backend Layers

```text
backend/internal/
├── domain/            # entities, value objects, repository interfaces
├── application/       # use cases: commands, queries, dto
└── infrastructure/    # adapters: persistence, HTTP, Telegram, migrations
```

Dependency direction:

```text
Domain <- Application <- Infrastructure
```

Rules:

- `domain` imports no application or infrastructure packages.
- `application` may import `domain`, but not `infrastructure`.
- `infrastructure` may import `domain` and `application`.
- Keep business rules out of HTTP and Telegram handlers.
- Use native SQL in repositories; do not add an ORM.
- Inject dependencies through constructors.
- Put `context.Context` first in IO and request-scoped functions.
- Validate value object invariants in constructors.
- Keep domain entities free from transport and persistence tags.

## Application Pattern

The application layer uses CQRS-style packages:

- `backend/internal/application/command` for write operations.
- `backend/internal/application/query` for read operations.
- `backend/internal/application/dto` for API-facing and handler-facing data transfer objects.

When adding behavior:

- Define or reuse domain types first.
- Add application command/query logic next.
- Wire infrastructure adapters through constructors.
- Keep handlers as request parsing, auth/session extraction, command/query calls, and response mapping.

## Infrastructure

Infrastructure owns external systems:

- `backend/internal/infrastructure/persistence/postgres` owns SQL and database mapping.
- `backend/internal/infrastructure/http` owns API transport, middleware, and responses.
- `backend/internal/infrastructure/telegram` owns Telegram bot adapters, handlers, keyboards, and sessions.
- `backend/internal/infrastructure/migrations` owns database schema migration files.

## Frontend

```text
frontend/src/
├── app/          # Next.js App Router pages and layouts
├── api/          # API client wrappers
├── components/   # React components
├── context/      # React contexts such as AuthContext
├── hooks/        # shared hooks
├── layout/       # dashboard shell
├── types/        # TypeScript types
└── utils/        # frontend utilities
```

Current main routes:

- `/login`
- `/`
- `/participants`
- `/participants/[id]`
- `/gifts`
- `/events`
- `/events/[id]`
- `/criteria`
- `/nominations`
- `/prize-distribution`

Frontend rules:

- Keep API access in `frontend/src/api`.
- Keep auth state in the existing auth context/hooks pattern.
- Preserve the TailAdmin dashboard structure unless a task explicitly asks for a redesign.
- Prefer typed request/response contracts over inline `any`.

## Runtime And Verification

Default runtime:

- `docker-compose up -d`
- `docker-compose run --rm migrate up`
- `make docker-up`
- `make docker-down`
- `make docker-logs`

Focused local checks:

- `make test`
- `cd backend && go test ./...`
- `cd frontend && npm run lint`
- `cd frontend && npm run build`

Use Docker Compose when verifying end-to-end behavior involving PostgreSQL, migrations, backend services, frontend, or Telegram runtime wiring.

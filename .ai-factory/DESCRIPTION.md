# Project: Gravel Bot

## Overview

Gravel Bot is a Telegram bot and web admin panel for organizing cycling races and gravel events. The backend exposes a REST API, runs the Telegram bot, applies PostgreSQL migrations, and follows DDD/Clean Architecture boundaries. The frontend is a TailAdmin-based Next.js dashboard for administrators.

## Core Features

- Telegram participant registration and event interaction.
- Admin authentication with JWT.
- Event, participant, gift, criteria, nomination, results, and prize distribution management.
- PostgreSQL persistence with SQL migrations.
- Swagger documentation served by the backend.
- Docker Compose runtime for PostgreSQL, migrations, backend services, and frontend.

## Tech Stack

- **Backend language:** Go 1.24
- **Backend routing:** Chi
- **Bot integration:** Telegram Bot API
- **Database:** PostgreSQL 16
- **Persistence:** Native SQL via repository implementations, no ORM
- **Migrations:** SQL migrations under `backend/internal/infrastructure/migrations`
- **Frontend:** Next.js 16, React 19, TypeScript, Tailwind CSS v4
- **Frontend base:** TailAdmin-style dashboard structure
- **Runtime:** Docker Compose

## Architecture Notes

- `.cursor/rules/rules.mdc` is the canonical source of project architecture rules.
- Backend code is organized as `domain`, `application`, and `infrastructure` layers.
- Dependency direction is `Domain <- Application <- Infrastructure`.
- Domain entities must remain pure and must not carry `json` or `db` tags.
- Application logic belongs in commands and queries; HTTP and Telegram handlers stay thin.
- Repository interfaces live in the domain layer; concrete PostgreSQL repositories live in infrastructure.
- Dependencies are passed through constructors.
- `context.Context` is the first parameter for operations that perform IO or request-scoped work.
- Value objects validate their invariants in constructors.

## Non-Functional Requirements

- Configuration comes from environment variables; `.env` is local and ignored.
- Docker Compose is the default full-system runtime.
- Logging should remain structured enough to diagnose API, bot, migration, and DB failures.
- Security-sensitive values include `BOT_TOKEN`, `JWT_SECRET`, database credentials, and Telegram chat IDs.
- Do not introduce ORM usage unless the project direction is explicitly changed.
- Do not create ad hoc Markdown reports or instruction files unless the user explicitly asks; AI Factory context files are allowed when `$aif` is invoked.

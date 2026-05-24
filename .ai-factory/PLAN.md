# Replace Telegram Bot Library With go-telegram/bot

Created: 2026-05-24
Mode: fast
Branch: main

## Goal

Replace the deprecated Telegram library `github.com/go-telegram-bot-api/telegram-bot-api/v5` with `github.com/go-telegram/bot` in the Go backend while preserving current bot behavior:

- `/start` command and main menu
- registration flow
- gift submission flow, including photo `file_id` persistence
- result submission flow
- callback cancellation and message cleanup
- admin API endpoints for Telegram file URL/info

The migration must stay inside infrastructure adapters. Domain and application layers must not import Telegram packages.

## Settings

- Testing: yes, include focused backend tests and run `cd backend && go test ./...`
- Logging: verbose during migration; keep existing `log.Printf` style and add useful logs around bot startup, handler errors, unsupported updates, and Telegram API call failures
- Docs: no standalone docs; update README/API docs only if a public command, endpoint, or runtime behavior changes

## Current Findings

- Current dependency: `backend/go.mod` uses `github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1`.
- Telegram types are confined to infrastructure:
  - `backend/internal/infrastructure/telegram/bot.go`
  - `backend/internal/infrastructure/telegram/handlers.go`
  - `backend/internal/infrastructure/telegram/handler/start.go`
  - `backend/internal/infrastructure/telegram/handler/registration.go`
  - `backend/internal/infrastructure/telegram/handler/gift.go`
  - `backend/internal/infrastructure/telegram/handler/result.go`
  - `backend/internal/infrastructure/telegram/keyboard/builder.go`
  - `backend/internal/infrastructure/http/handler/telegram.go`
- The bot currently owns a manual polling loop via `GetUpdatesChan`, dispatches each update in a goroutine, and uses `api.Send` / `api.Request` helpers.
- The new library exposes `bot.New`, `Start(ctx)`, `WithDefaultHandler`, typed `models.Update`, handler registration helpers, and methods with `(ctx context.Context, params *XParams)` signatures.
- As of 2026-05-24, `github.com/go-telegram/bot` latest pkg.go.dev version is `v1.21.0`, published May 22, 2026, and the GitHub README says it supports Telegram Bot API 10.0 from May 8, 2026.
- The new models are not a pure package rename: `models.Message` uses `ID` instead of `MessageID`, `Message.From` is nullable, `models.CallbackQuery.Message` is `models.MaybeInaccessibleMessage`, and old helpers like `Message.IsCommand()` / `Message.Command()` are not available.
- Architecture rules are available in `.ai-factory/ARCHITECTURE.md` and `AGENTS.md`.

## Tasks

### Phase 1 - Dependency And API Boundary

- [x] 1. Swap backend Telegram dependency.
   - Files: `backend/go.mod`, `backend/go.sum`
   - Deliverable: remove `github.com/go-telegram-bot-api/telegram-bot-api/v5`, add `github.com/go-telegram/bot v1.21.0` or the latest stable version confirmed at implementation time, then run `go mod tidy` from `backend`.
   - Logging: no runtime logging in this task; record the selected version in commit message or implementation notes.
   - Dependencies: none.

- [x] 2. Define the migration boundary for Telegram transport types.
   - Files: `backend/internal/infrastructure/telegram/keyboard/builder.go`, possibly new `backend/internal/infrastructure/telegram/telegramtypes` or helper file if needed
   - Deliverable: decide whether infrastructure handlers use `github.com/go-telegram/bot/models` directly or via a tiny local helper layer. Do not introduce Telegram imports into `backend/internal/domain` or `backend/internal/application`.
   - Logging: no runtime logging; this is type structure only.
   - Dependencies: Task 1.

### Phase 2 - Bot Runtime Adapter

- [x] 3. Migrate bot construction and update processing.
   - Files: `backend/internal/infrastructure/telegram/bot.go`, `backend/cmd/bot/main.go`
   - Deliverable: replace `*tgbotapi.BotAPI` with `*bot.Bot`; construct it with `bot.New(cfg.Token, ...)`; use `bot.WithDefaultHandler` or explicit registered handlers to route updates into existing `Bot.handleUpdate`.
   - Deliverable: choose command routing explicitly with either `WithMessageTextHandler(..., bot.MatchTypeCommand, ...)` / `WithCallbackQueryDataHandler(...)` or local parsing of `models.Message.Text`; do not rely on old `Message.IsCommand()` / `Message.Command()` helpers.
   - Deliverable: replace the manual `GetUpdatesChan` loop with `b.api.Start(ctx)` while preserving graceful shutdown behavior in `cmd/bot/main.go`.
   - Deliverable: support debug mode with `bot.WithDebug()` or equivalent option only when `cfg.Debug` is true.
   - Logging: log successful bot initialization, start, stop, and initialization failures; add DEBUG-style logs for unsupported update kinds only if debug is enabled.
   - Dependencies: Tasks 1-2.

- [x] 4. Migrate low-level Telegram send helpers to context-first calls.
   - Files: `backend/internal/infrastructure/telegram/bot.go`, `backend/internal/infrastructure/telegram/handlers.go`
   - Deliverable: update helper methods to accept `context.Context` first:
     - `SendMessage(ctx, chatID, text)`
     - `SendMessageWithKeyboard(ctx, chatID, text, keyboard)`
     - `EditMessage(ctx, chatID, messageID, text)`
     - `AnswerCallback(ctx, callbackID, text)`
     - `DeleteMessage(ctx, chatID, messageID)`
   - Deliverable: return `*models.Message` from send helpers where callers need the sent message ID for later deletion.
   - Deliverable: remove direct `b.api.Send(...)` and `b.api.Request(...)` calls from `handlers.go` in favor of the helpers.
   - Logging: log Telegram API call failures with operation name, chat ID when available, and callback/message ID when relevant; do not log bot token or user private text.
   - Dependencies: Task 3.

### Phase 3 - Handlers, Keyboards, And File API

- [x] 5. Add local helpers for the new Telegram model shape.
   - Files: `backend/internal/infrastructure/telegram/handlers.go`, new helper file if useful, `backend/internal/infrastructure/telegram/handler/start.go`
   - Deliverable: add small infrastructure-only helper functions for:
     - extracting chat ID and message ID from `models.CallbackQuery.Message` only when it contains an accessible `models.Message`
     - returning/logging a safe fallback when callback messages are inaccessible or absent
     - checking `models.Message.From` before user/session logic
     - mapping sent/edited message IDs through `models.Message.ID`
   - Deliverable: update handler call sites that currently assume `callback.Message.Chat.ID`, `callback.Message.MessageID`, and `msg.From.ID` always exist.
   - Logging: log inaccessible callback messages, missing senders, and unsupported updates without logging bot token or user private text.
   - Dependencies: Tasks 3-4.

- [x] 6. Migrate Telegram models in bot handlers and keyboards.
   - Files:
     - `backend/internal/infrastructure/telegram/handlers.go`
     - `backend/internal/infrastructure/telegram/handler/start.go`
     - `backend/internal/infrastructure/telegram/handler/registration.go`
     - `backend/internal/infrastructure/telegram/handler/gift.go`
     - `backend/internal/infrastructure/telegram/handler/result.go`
     - `backend/internal/infrastructure/telegram/keyboard/builder.go`
   - Deliverable: replace `tgbotapi.Message`, `CallbackQuery`, `InlineKeyboardMarkup`, `InlineKeyboardButton`, and `PhotoSize` usage with `github.com/go-telegram/bot/models` equivalents.
   - Deliverable: replace old constructor helpers such as `NewInlineKeyboardMarkup`, `NewInlineKeyboardRow`, and `NewInlineKeyboardButtonData` with local builder methods that create `models.InlineKeyboardMarkup` / `models.InlineKeyboardButton` structs.
   - Deliverable: preserve callback data values exactly: `register`, `add_gift`, `submit_result`, `info`, `cancel`, `bike_*`, `gender_*`, `gift_gender_*`, `gift_bike_*`, `finish_gift`, `skip_photos`.
   - Logging: keep existing business-error logs; add logs for nil message/callback edge cases introduced by the new model shape.
   - Dependencies: Tasks 2, 4, and 5.

- [x] 7. Migrate Telegram file endpoints used by the admin dashboard.
   - Files: `backend/internal/infrastructure/http/handler/telegram.go`, possibly `backend/internal/infrastructure/http/server.go`
   - Deliverable: replace `tgbotapi.NewBotAPI`, `GetFile`, and `file.Link(token)` with `go-telegram/bot` equivalents: `bot.New`, `GetFile(ctx, &bot.GetFileParams{FileID: fileID})`, and `FileDownloadLink`.
   - Deliverable: avoid making API server startup or ordinary file requests fail only because Telegram `getMe` is unavailable; if the client is created during HTTP handler construction or per request, use `bot.WithSkipGetMe()` or a deliberately short `bot.WithCheckInitTimeout(...)`.
   - Deliverable: prefer a small cached/injected Telegram file client interface if it keeps handler tests from calling the real Telegram API.
   - Deliverable: preserve existing JSON response contracts for `/api/telegram/files/{fileId}` and `/api/telegram/files/{fileId}/info`.
   - Logging: log Telegram connection/get-file failures with file ID only; do not log token.
   - Dependencies: Task 1.

### Phase 4 - Tests And Verification

- [x] 8. Add focused tests for the migrated infrastructure behavior.
   - Files:
     - `backend/internal/infrastructure/telegram/keyboard/builder_test.go`
     - `backend/internal/infrastructure/telegram/handler/*_test.go` where fake repositories are practical
     - optional `backend/internal/infrastructure/http/handler/telegram_test.go` if a small Telegram client interface is introduced
   - Deliverable: table-driven tests for keyboard rows/buttons and callback data preservation.
   - Deliverable: focused tests for handler return values and session state transitions without calling the real Telegram API.
   - Deliverable: cover the risky model-shape cases: inaccessible callback message, missing `Message.From`, command routing for `/start`, and sent-message ID tracking for gift cleanup.
   - Logging: tests should not assert log text unless a log branch is itself the behavior.
   - Dependencies: Tasks 5-7.

- [x] 9. Run backend verification and cleanup.
   - Files: all changed backend files
   - Deliverable: run `gofmt` on changed Go files, then run `cd backend && go test ./...`.
   - Deliverable: run `rg "go-telegram-bot-api|tgbotapi" backend` and ensure the old library is gone.
   - Deliverable: optionally run `make build` to catch all command entry points.
   - Logging: no new runtime logs; verify added logs are actionable and do not expose secrets.
   - Dependencies: Tasks 1-8.

## Risks And Notes

- `go-telegram/bot.New` calls `getMe` by default; this can change startup behavior if used in the HTTP API process for file endpoints. Handle this explicitly.
- The old library provided many convenience constructors. The new library is more parameter/model oriented, so a local keyboard builder is the lowest-risk way to avoid noisy repeated struct literals.
- Current send helper methods do not accept `context.Context`; migration should fix that because project rules require context first for IO.
- The current bot dispatches updates with one goroutine per update. `go-telegram/bot` has its own worker/handler model; preserve concurrency intentionally rather than blindly nesting goroutines.
- Callback message access is a migration risk because new callbacks can contain an inaccessible message; do not blindly dereference callback message/chat fields.
- Message sender access is a migration risk because `models.Message.From` is a pointer; guard it before user/session operations.
- There are currently no backend test files, so this migration should introduce the first focused tests without broad unrelated coverage work.

## Commit Plan

1. `chore(backend): replace telegram bot dependency`
   - Tasks 1-2

2. `refactor(backend): migrate telegram bot runtime to go-telegram`
   - Tasks 3-7

3. `test(backend): cover telegram keyboard and handler migration`
   - Tasks 8-9

## References

- https://github.com/go-telegram/bot
- https://pkg.go.dev/github.com/go-telegram/bot

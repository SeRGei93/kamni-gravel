# Implementation Plan: Telegram Public Chat Welcome And Participant Menu

Branch: feature/telegram-public-chat-menu
Created: 2026-05-27

## Settings

- Testing: yes
- Logging: standard
- Docs: no, except existing env/API docs if an implementation task changes a public contract

## Goal

When a Telegram user joins the configured public chat, the bot should greet them in that chat with the active event name and a participant-aware inline menu.

The same menu must be available in private chat, with registration-aware actions:

- before registration: `✅ Принять участие`
- after registration: `😢 Отказаться от участия`
- `‼️ Условия участия` from event description
- existing gift and result flow untouched
- prize fund via `MINIAPP_URL`
- after gift confirmation, notify/forward admin review context to `ADMIN_CHAT_ID`.

## Requirements

- Use `PUBLIC_CHAT_ID` from `.env` to detect/route public chat welcomes.
- Use `ADMIN_CHAT_ID` from `.env` for gift moderation notifications; do not hardcode chat IDs.
- Keep `MINIAPP_URL` as the prize fund miniapp link.
- Reuse `events.description` as “Условия участия” text.
- Preserve existing registration questions: bike type and gender.
- Preserve existing gift flow and confirmation-before-save behavior.
- Keep handlers thin; business decisions like withdrawal should be application-layer command logic.
- Do not add `go-telegram/ui` or any Telegram UI framework dependency.
- Do not create ad hoc Markdown reports.

## Current Findings

- `backend/internal/config/main.go`, `env.example`, and `docker-compose.yml` already include `ADMIN_CHAT_ID`, `PUBLIC_CHAT_ID`, and `MINIAPP_URL`.
- The Telegram adapter currently wires `Token`, `Debug`, `MiniappURL`, `SessionTimeout`; `AdminChat`, `PublicChat`, and bot username/deep-link support are not yet passed in.
- `backend/internal/infrastructure/telegram/bot.go` validates `MINIAPP_URL` and stores `miniappURL`, but has no admin/public chat fields and no bot username cache.
- `github.com/go-telegram/bot` v1.21.0 calls `GetMe` during `telegrambot.New` unless `WithSkipGetMe()` is used, but the returned username is not stored on the library `Bot`, so resolving username for deep links needs an explicit non-critical lookup after construction or a controlled init path.
- `keyboard.MainMenu(miniappURL)` is static (`register`, `add_gift`, optional miniapp, `submit_result`, `info`).
- `ParticipantRepository.FindByUserAndEvent(ctx, userID, event.ID)` already exists.
- Admin deletion uses `command.DeleteParticipantHandler`; user self-withdrawal command does not yet exist.
- Public-join flow is currently missing (`msg.NewChatMembers` is not handled).
- User blacklist guard exists for normal update sender, but a group join service message can involve joined users that differ from `msg.From`; public welcome must check each joined user before greeting.
- Gift persistence exists (`GiftHandler.ConfirmAddGift`) but only stores Telegram file IDs; source message references for forwarding/copy are not captured.

## Assumptions

- Public welcome is only sent in configured `PUBLIC_CHAT_ID`. If `PUBLIC_CHAT_ID=0`, no public welcome is sent and startup logs that this is disabled.
- “Призовой фонд” maps to `MINIAPP_URL`; no new miniapp feature scope.
- “Условия участия” maps to `events.description`; no schema changes.
- Public chat buttons should not start registration/result/photo flows directly in-group; they must route users to private chat.
- `/start` can include payloads (`/start conditions`) and should be parsed before generic `/start` behavior.
- Bot username can be unavailable at runtime; deep links must gracefully fall back to text instruction.
- Keep legacy callback `info` behavior as compatibility fallback, but primary event text action should be `event_conditions`.
- Telegram chat IDs are sensitive project configuration values; runtime logs should state configured/disabled or redacted IDs, not raw `ADMIN_CHAT_ID`/`PUBLIC_CHAT_ID`.

## Commit Plan

- **Commit 1** (after tasks 1-4): `feat(telegram): add participant-aware menu and public chat welcome`
- **Commit 2** (after tasks 5-7): `feat(telegram): add withdrawal and admin gift notification`
- **Commit 3** (after tasks 8-9): `test(telegram): cover menu, public chat, withdrawal, and gift notification paths`

## Tasks

### Phase 1: Adapter Configuration And Menu Contract

- [ ] 1. Pass public/admin chat config + bot username support into Telegram adapter.

  Files:
  - `backend/cmd/bot/main.go`
  - `backend/internal/infrastructure/telegram/bot.go`

  Deliverable:
  - Extend `telegram.Config` fields used by constructor with `AdminChatID` and `PublicChatID` (already declared in `config.main`, now consumed by adapter).
  - Store `adminChatID`, `publicChatID` on `Bot`.
  - Add `botUsername` cache on `Bot` and initialize it via explicit non-critical `GetMe` lookup after API construction.
  - Preserve existing startup validation behavior: do not silently weaken token validation unless implementation deliberately switches to `WithSkipGetMe()` and adds an equivalent validation path.
  - Add startup logs:
    - whether `PUBLIC_CHAT_ID`/`ADMIN_CHAT_ID` are enabled,
    - whether username was resolved (or reason unavailable),
    - no sensitive token/init data logging.

  Logging requirements:
  - INFO when public/admin notification modes are enabled/disabled.
  - INFO/WARN for bot username resolution outcome.
  - Never log bot token, raw chat IDs, init payloads, or file URL content.

  Dependencies:
  - Required before tasks 2–5, because deep links and public welcome use this state.

- [ ] 2. Replace static main menu with participant-aware menu builder and callback contract update.

  Files:
  - `backend/internal/infrastructure/telegram/keyboard/builder.go`
  - `backend/internal/infrastructure/telegram/keyboard/builder_test.go`
  - `backend/internal/infrastructure/telegram/handler/start.go`
  - `backend/internal/infrastructure/telegram/handler/start_test.go`
  - `backend/internal/infrastructure/telegram/handlers.go`

  Deliverable:
  - Replace `MainMenu(miniappURL)` with builder signature that accepts:
    - active event presence,
    - user registration status,
    - `miniappURL`,
    - optional deep-link targets.
  - Private menu for participant state:
    - not registered: `✅ Принять участие` (`register`)
    - registered: `😢 Отказаться от участия` (`withdraw_participation`)
    - registered: `🏁 Я уже проехал` (`submit_result`)
    - all users: `🎁 Добавить приз` (`add_gift`)
    - all users: `‼️ Условия участия` (`event_conditions`)
    - optional: `🏆 Призовой фонд` WebApp when `miniappURL` present.
  - Keep legacy `info` as alias/compatibility path to the same behavior as `event_conditions`.
  - Update `/start` handler to build menu based on active event and user registration status.

  Logging requirements:
  - WARN if participant status lookup fails and fallback is needed.
  - INFO for significant menu decision branches, not for every `/start`.

### Phase 2: Public Chat Entry And Conditions Delivery

- [ ] 3. Handle public-chat member joins and send onboarding message/buttons.

  Files:
  - `backend/internal/infrastructure/telegram/handlers.go`
  - `backend/internal/infrastructure/telegram/model_helpers.go`
  - `backend/internal/infrastructure/telegram/model_helpers_test.go`
  - `backend/internal/infrastructure/telegram/keyboard/builder.go`
  - `backend/internal/infrastructure/telegram/keyboard/builder_test.go`

  Deliverable:
  - Add branch in `handleMessage` for `msg.NewChatMembers`.
  - Ignore:
    - non-matching chat IDs when `PUBLIC_CHAT_ID != 0`,
    - bots and bot account itself.
  - For each joined user:
    - skip blacklisted joined users using the existing blacklist query/handler,
    - ensure user exists in users table with profile fields (reuse `/start` creation pattern),
    - load active event,
    - send welcome: `👋 Привет, {first_name}! Добро пожаловать в {event_name} 🚴`.
  - Add public-safe menu:
    - `✅ Принять участие` as deep link to bot/start (or fallback text if username unavailable),
    - `🏆 Призовой фонд` as WebApp when miniapp configured,
    - `‼️ Условия участия` as deep link to `/start conditions` (or safe fallback if username unavailable).
  - Do not run registration/result logic directly inside public chat.

  Logging requirements:
  - INFO with `telegram_user_id`, `event_id`, and redacted/configured public chat marker when welcome sent.
  - INFO/WARN for blacklisted joined user skip without message text.
  - WARN on user creation lookup/send failures.
  - No raw chat IDs, message text, or secrets in logs.

- [ ] 4. Add `event_conditions` callback and `/start` payload path.

  Files:
  - `backend/internal/infrastructure/telegram/handlers.go`
  - `backend/internal/infrastructure/telegram/handler/start.go`
  - `backend/internal/infrastructure/telegram/handler/start_test.go`

  Deliverable:
  - Handle callback `event_conditions`:
    - load active event,
    - send event `description`,
    - fallback if no event / empty description.
  - Keep `info` callback to mapped behavior for compatibility.
  - Parse `/start` payloads in `StartHandler`:
    - `/start conditions` should return same conditions response path.
    - unknown payloads should return deterministic fallback and not break normal flow.

  Logging requirements:
  - WARN on active-event lookup failures.
  - INFO when conditions is requested with `telegram_user_id` + `event_id`.
  - Never log full description text.

### Phase 3: Self-Withdrawal

- [ ] 5. Add participant self-withdrawal command and callback.

  Files:
  - `backend/internal/application/command/withdraw_participant.go`
  - `backend/internal/application/command/withdraw_participant_test.go`
  - `backend/internal/infrastructure/telegram/bot.go`
  - `backend/internal/infrastructure/telegram/handlers.go`
  - `backend/internal/infrastructure/telegram/handler/registration.go`
  - `backend/internal/infrastructure/telegram/handler/registration_test.go`

  Deliverable:
  - Add `WithdrawParticipantHandler` with `UserID`, `EventID`.
  - Handler must:
    - find participant by user+event,
    - call `DeleteWithResultCriteria`,
    - return `ErrParticipantNotFound` if absent.
  - Add callback `withdraw_participation` in bot handler:
    - load active event,
    - run command,
    - reset state/cleanup if needed,
    - re-render participant-aware menu.
  - If not registered, respond with stable message and render non-registered menu.

  Logging requirements:
  - INFO when withdrawal requested/completed with `telegram_user_id`, `event_id`, `participant_id`.
  - WARN when target participant not found.
  - ERROR/WARN on repo errors with operation stage.

### Phase 4: Gift Source Tracking And Admin Notification

- [ ] 6. Capture source message refs for gifts (separate from attachment persistence).

  Files:
  - `backend/internal/infrastructure/telegram/model_helpers.go`
  - `backend/internal/infrastructure/telegram/model_helpers_test.go`
  - `backend/internal/infrastructure/telegram/handlers.go`
  - `backend/internal/infrastructure/telegram/handler/gift.go`
  - `backend/internal/infrastructure/telegram/handler/gift_test.go`
  - `backend/internal/infrastructure/telegram/session/manager.go` (optional typed helpers)

  Deliverable:
  - Add helper to capture source `(chat_id, message_id)` for gift text/photos/documents before attachment normalization.
  - Keep existing `gift_attachments` behavior unchanged (Telegram file IDs only).
  - Store source refs separately in session and ordered by arrival.
  - Ensure media group suppression used for reply text does not suppress source ref capture.
  - Defensive session reads, no unchecked type assumptions.

  Logging requirements:
  - DEBUG on captured source ref with user ID, redacted/configured chat marker, message ID, and update kind.
  - WARN on malformed ref data in session.
  - Never log gift text/captions/file IDs.

- [ ] 7. Send admin notification after gift confirmation.

  Files:
  - `backend/internal/infrastructure/telegram/bot.go`
  - `backend/internal/infrastructure/telegram/handlers.go`
  - `backend/internal/infrastructure/telegram/handler/gift.go`
  - `backend/internal/infrastructure/telegram/handler/gift_test.go`

  Deliverable:
  - Add helper methods in bot layer to copy/forward captured source messages to admin chat.
  - Prefer library params `CopyMessageParams`/`ForwardMessageParams`; optionally use bulk `CopyMessagesParams` only when source messages share one source chat and order can be preserved.
  - Preserve ordering and copy all captured messages (including every media-group item).
  - On `confirm_gift` success:
    - if `ADMIN_CHAT_ID==0`, skip and log once,
    - else send all source messages then/and/or fallback summary.
  - Summary must include safe metadata only: `gift_id`, `event_id`, `user_id`, filters, status, photo count.
  - If forwarding/copying fails, still keep user flow successful and always send summary fallback.

  Logging requirements:
  - INFO when admin notification succeeds with `gift_id`, `event_id`, and redacted/configured admin chat marker.
  - WARN when skipped or forwarding/copying fails.
  - ERROR only if both message forwarding and summary send fail.
  - Never log raw admin/public chat IDs, gift descriptions, captions, Telegram file IDs, or token-bearing URLs.

### Phase 5: Tests And Verification

- [ ] 8. Add/extend focused tests for menu state, public joins, conditions, withdrawal, and gift notification helpers.

  Files:
  - `backend/internal/infrastructure/telegram/keyboard/builder_test.go`
  - `backend/internal/infrastructure/telegram/model_helpers_test.go`
  - `backend/internal/infrastructure/telegram/handler/start_test.go`
  - `backend/internal/infrastructure/telegram/handler/registration_test.go`
  - `backend/internal/infrastructure/telegram/handler/gift_test.go`
  - `backend/internal/application/command/withdraw_participant_test.go`

  Deliverable:
  - Participant-aware menu callback data (registered/unregistered).
  - Miniapp button conditions and `event_conditions` callback.
  - Public-join handling for allowed/unrelated chat, bot user filtering.
  - Public-join handling skips blacklisted joined users even if `msg.From` is different.
  - `/start` payload parsing tests (`conditions` + unknown payload).
  - Withdrawal command success and `ErrParticipantNotFound` path.
  - Gift source ref capture for single message and media groups.
  - Gift admin helper copies/forwards in order and summary fallback safety.
  - Ensure tests do not validate secrets and remain stable.

- [ ] 9. Verification and clean-up.

  Files:
  - All changed backend/frontend files

  Deliverable:
  - Run `gofmt` for touched Go files.
  - Run `cd backend && go test ./...`.
  - Run `cd backend && go build ./...`.
  - Run `cd backend && go vet ./...`.
  - If frontend files changed, run targeted frontend checks and `cd frontend && npm run build`.
  - Run `git diff --check`.
  - Review final log statements and confirm raw `ADMIN_CHAT_ID`/`PUBLIC_CHAT_ID` are not printed.

  Logging requirements:
  - Remove temporary diagnostics.
  - Keep final runtime logs structured and safe.

## Open Decisions For Implementation

- Public welcome `conditions` delivery path: deep link to `/start conditions` with fallback text, or callback-only fallback if deep link impossible.
- For admin gift delivery, prefer `CopyMessage` when sender attribution should stay hidden; use `ForwardMessage` only if preserving original sender attribution is explicitly desired.
- Whether to relabel admin UI “Описание” → “Условия участия” in current screens (not required for backend behavior; do only if product asks).

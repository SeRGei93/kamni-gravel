# Implementation Plan: Admin User Blacklist And Participant Deletion

Branch: feature/admin-user-blacklist
Created: 2026-05-27

## Settings

- Testing: yes
- Logging: verbose
- Docs: API docs only where contracts change; no broad README/docs update

## Assumptions

- The app is not in production yet, so no production backfill or historical-data cleanup is required.
- Current blacklisted users do not have participants, gifts, results, or prize data that must be preserved or retroactively excluded.
- Blacklist enforcement blocks future participant registration and future gift creation through protected admin/API paths.
- For Telegram bot interactions, blacklisted users must be silently ignored: no `/start` response, no callback answer, no message reply, no session mutation, and no user upsert side effect. To the user, the bot should look like it does not respond at all.
- The blacklist is not a retroactive cleanup mechanism in this iteration. If an admin needs to remove existing race data, they use explicit admin deletion controls.
- Blacklisted users may still open read-only Mini App catalog pages; the restriction is participation and gift submission.

## Commit Plan

- **Commit 1** (after tasks 1-3): `feat(admin): add user blacklist backend`
- **Commit 2** (after tasks 4-6): `feat(bot): silently ignore blacklisted users`
- **Commit 3** (after tasks 7-10): `feat(admin): add blacklist and participant controls`
- **Commit 4** (after tasks 11-12): `test(admin): cover blacklist and participant deletion`

## Tasks

### Phase 1: Backend Data Model And Application Layer

- [x] Task 1: Add blacklist persistence model.
  - Deliverable: create migration `backend/internal/infrastructure/migrations/00017_create_user_blacklist.sql` with table `user_blacklist`.
  - Schema: `telegram_user_id BIGINT PRIMARY KEY`, `reason TEXT NOT NULL DEFAULT ''`, `created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`, `updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`.
  - Do not add a FK to `users`; admins must be able to block a Telegram ID before that user has ever sent `/start`.
  - Add domain entity `backend/internal/domain/entity/user_blacklist.go` and repository interface `backend/internal/domain/repository/user_blacklist.go`.
  - Add PostgreSQL implementation `backend/internal/infrastructure/persistence/postgres/user_blacklist_repo.go`.
  - Repository operations: list with optional `LEFT JOIN users` profile data, find by Telegram ID, `IsBlacklisted`, upsert/add with reason, update reason, and delete.
  - Logging requirements: log repository write failures with `telegram_user_id` and operation; do not log auth tokens, raw request bodies, or Telegram init data.

- [x] Task 2: Add blacklist application commands, queries, and DTOs.
  - Deliverable: add command handlers under `backend/internal/application/command` for add/upsert, update reason, and remove blacklist entries.
  - Add query handlers under `backend/internal/application/query` for list and `IsBlacklisted`.
  - Add DTOs under `backend/internal/application/dto` for blacklist list/detail responses with optional existing Telegram profile fields (`username`, `first_name`, `last_name`) from the repository join.
  - Validate Telegram user ID as positive; trim reason; return typed errors for invalid ID and not found.
  - Keep domain/application free of infrastructure imports.
  - Logging requirements: log add/update/remove state changes at INFO level with `telegram_user_id`; log validation failures at WARN and repository failures at ERROR.
  - Depends on Task 1.

- [x] Task 3: Add safe participant deletion command.
  - Deliverable: create `DeleteParticipantHandler` in `backend/internal/application/command`.
  - Replace raw handler-driven deletion with a command that verifies participant existence, finds participant results, deletes `entity_criteria` rows for those results, then deletes the participant in a safe order.
  - Prefer repository methods that execute the cleanup transactionally in `backend/internal/infrastructure/persistence/postgres/participant_repo.go`; if the existing repository interface is extended, keep it domain-owned.
  - Preserve existing cascade behavior for `results`, but do not leave orphaned `entity_criteria` rows because `entity_criteria.entity_id` has no FK to `results`.
  - Logging requirements: log participant deletion start/success at INFO with `participant_id` and `event_id`; log cleanup/delete failures at ERROR with operation stage.
  - No blacklist dependency; this is placed early so the backend endpoint and admin UI can depend on the safer command.

### Phase 2: Runtime Wiring And Enforcement

- [x] Task 4: Wire blacklist and participant delete handlers into API and bot runtimes.
  - Deliverable: initialize `userBlacklistRepo` in `backend/cmd/api/main.go` and `backend/cmd/bot/main.go`.
  - Pass the repository/query handlers through `backend/internal/infrastructure/http/server.go` and `backend/internal/infrastructure/telegram/bot.go`.
  - Wire `DeleteParticipantHandler` into `ParticipantsHandler` instead of calling `participantRepo.Delete` directly from HTTP.
  - Bot wiring must make the blacklist check available at the earliest Telegram update boundary, before `/start` can create or update a `users` row.
  - Logging requirements: log wiring/startup failures only at existing startup boundaries; include operation names, not secrets.
  - Depends on Tasks 1-3.

- [x] Task 5: Enforce blacklist in participant registration and Telegram silent ignore.
  - Deliverable: update `backend/internal/application/command/register_participant.go` to reject blacklisted users before creating participants, returning typed `ErrUserBlacklisted`.
  - Update protected HTTP participant create handling in `backend/internal/infrastructure/http/handler/participants.go` to map `ErrUserBlacklisted` to `403 Forbidden`.
  - Add a central Telegram update guard in `backend/internal/infrastructure/telegram/bot.go` / `backend/internal/infrastructure/telegram/handlers.go` before command, callback, and free-text handling.
  - If `message.From.ID` or `callback.From.ID` is blacklisted, return immediately without `SendMessage`, `AnswerCallback`, `EditMessage`, `DeleteMessage`, session mutation, or user creation.
  - Ensure `/start` for a blacklisted Telegram user is silently ignored and does not call `StartHandler.Handle`.
  - Logging requirements: log silently ignored Telegram updates at INFO or WARN with `telegram_user_id`, update kind, and operation; never send any response to Telegram. Keep HTTP blocked registration attempts logged at WARN with `telegram_user_id` and `event_id` when known.
  - Depends on Task 4.

- [x] Task 6: Enforce blacklist in gift creation.
  - Deliverable: update `backend/internal/application/command/add_gift.go` to reject blacklisted users before creating a gift, returning typed blocked-user error.
  - Protected/API callers should receive a non-2xx error if this command is reused outside Telegram.
  - Do not add blacklist-specific Telegram replies in `backend/internal/infrastructure/telegram/handler/gift.go`; the central Telegram guard from Task 5 must silently drop blacklisted users before the multi-step gift flow begins.
  - No extra read-side filtering for historical gifts is required in this iteration because the app is not in production and blocked users do not have existing gifts.
  - Logging requirements: log HTTP/API blocked gift attempts at WARN with `telegram_user_id` and `event_id`; log Telegram silent drops only in the central guard.
  - Depends on Task 4.

### Phase 3: HTTP API And Admin UI

- [x] Task 7: Add protected blacklist HTTP API.
  - Deliverable: add `backend/internal/infrastructure/http/handler/user_blacklist.go`.
  - Add protected admin routes in `backend/internal/infrastructure/http/server.go`: `GET /api/user-blacklist`, `POST /api/user-blacklist`, `PUT /api/user-blacklist/{telegramUserId}`, and `DELETE /api/user-blacklist/{telegramUserId}`.
  - Responses must use `backend/internal/application/dto`, not domain entities directly.
  - Map invalid Telegram ID to `400`, missing entry on update/delete to `404`, and repository failures to `500`.
  - Update `backend/docs/swagger.yaml` for new endpoints and schemas.
  - Logging requirements: log admin add/remove/update operations at INFO with `telegram_user_id`; log invalid IDs at WARN; log repository failures at ERROR.
  - Depends on Tasks 2 and 4.

- [x] Task 8: Build admin blacklist page.
  - Deliverable: add frontend API client `frontend/src/api/userBlacklist.ts`, types in `frontend/src/types/index.ts`, page `frontend/src/app/(dashboard)/user-blacklist/page.tsx`, and sidebar navigation in `frontend/src/layout/AppSidebar.tsx`.
  - The page should allow admins to add a Telegram user ID with optional reason, list current entries with joined Telegram profile data when available, edit/update reason, and remove entries with confirmation.
  - Keep TailAdmin dashboard style and avoid unrelated redesign.
  - Use existing `get/post/put/del` client helpers from `frontend/src/api/client.ts`.
  - Logging requirements: use existing frontend error handling style; `console.error` failed API operations with safe fields (`telegram_user_id`, operation), never tokens.
  - Depends on Task 7.

- [x] Task 9: Harden backend participant deletion endpoint.
  - Deliverable: update `backend/internal/infrastructure/http/handler/participants.go` so `DELETE /api/participants/{id}` calls `DeleteParticipantHandler`.
  - Keep the route protected by existing admin auth in `backend/internal/infrastructure/http/server.go`.
  - Return `404` for missing participant, `204` for success, and `500` for unexpected cleanup/delete failures.
  - Logging requirements: log delete requests and outcomes with `participant_id`; log cleanup stage on failure.
  - Depends on Tasks 3-4.

- [x] Task 10: Add participant deletion controls in admin UI.
  - Deliverable: wire existing `participantsApi.delete` into `frontend/src/app/(dashboard)/participants/page.tsx` and `frontend/src/components/participants/ParticipantsTable.tsx`.
  - Add a delete action with confirmation, per-participant loading state, reload after success, and clear error text on retry.
  - Add delete action on `frontend/src/app/(dashboard)/participants/[id]/page.tsx`; after success redirect back to `/participants`.
  - Preserve current selected event/filter/search state on the list page as much as the existing page state allows; do not introduce a broad route-state refactor.
  - Logging requirements: log frontend failures with `participant_id`, selected `event_id`, and operation.
  - Depends on Task 9.

### Phase 4: Tests And Verification

- [x] Task 11: Add backend tests.
  - Deliverable: add focused tests for blacklist commands/queries, `RegisterParticipantHandler`, `AddGiftHandler`, `DeleteParticipantHandler`, and protected blacklist HTTP routes.
  - Add Telegram handler/bot tests proving blacklisted users receive no `/start` response, no callback answer, no message reply, and no session/user mutation.
  - Add participant deletion cleanup coverage proving result `entity_criteria` rows are removed or no longer orphaned.
  - Add HTTP handler tests where existing patterns make this cheap: `403` for blocked participant creation and CRUD status mapping for blacklist routes.
  - Logging requirements: tests should assert behavior, not log output; add production logs only in implementation paths.
  - Depends on Tasks 5-9.

- [x] Task 12: Run verification gates and smoke checks.
  - Deliverable: run `cd backend && go test ./...`.
  - Run `cd frontend && npx tsc --noEmit`.
  - Run targeted ESLint on touched frontend files, because repo-wide `npm run lint` has known unrelated baseline issues.
  - Run `cd frontend && npm run build` if frontend changes compile cleanly.
  - Smoke-check admin flows in browser: blacklist add/update/remove, participant delete from list/detail, and no broken participant/gift navigation.
  - Smoke-check Telegram behavior with a blacklisted Telegram ID at the bot boundary if a safe local/manual bot verification path is available; expected result is no visible bot response.
  - Logging requirements: record only command outcomes in implementation notes; do not create extra Markdown reports.
  - Depends on Tasks 8-11.

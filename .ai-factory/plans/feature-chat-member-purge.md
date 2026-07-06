# Implementation Plan: Chat Member Roster + Purge via Admin Panel

Branch: none (staying on `main`, matching current project workflow)
Created: 2026-07-06 · Revised: 2026-07-06 (roster-table redesign)

## Settings

- Testing: yes
- Logging: standard (project `log.Printf` INFO/WARN convention with key=value markers)
- Docs: no (warn-only)

## Goal

Maintain a persistent `chat_members` table that always reflects the **current**
members of the public chat, and let admins purge (kick) members who have not added a
gift to the active event's prize fund — all from the admin panel.

- The **bot** keeps the table current by listening to Telegram `chat_member` updates:
  upsert on join/promote, **delete the row** on leave/kick.
- The table is **seeded initially** from `chat_members.csv` (export script) via an
  upload on the admin panel.
- The **purge page** reads candidates from the table (present members minus gift
  owners, admins and bots), pre-checks everyone, admin unchecks whom to keep, confirm
  → bot-less kick from the API (`banChatMember` + `unbanChatMember`).

## Key Feasibility Facts (verified in `go-telegram/bot@v1.21.0`)

- `models.Update.ChatMember *models.ChatMemberUpdated` exists (models/update.go:27) with
  `OldChatMember`/`NewChatMember` statuses → derive joined vs left. Fires on every
  membership change because the bot is a chat admin (confirmed 2026-07-06).
- **CRITICAL**: `chat_member` is NOT delivered by default. Must pass it in
  `allowed_updates` via `telegrambot.WithAllowedUpdates([]string{...})` (options.go:101,
  constant `models.AllowedUpdateChatMember`). `allowed_updates` is a **full whitelist,
  not additive** — it must ALSO list every currently-used type (`message`,
  `edited_message`, `callback_query`, `my_chat_member`, …) or the bot stops receiving
  them and breaks. This is the main correctness risk.
- Service messages `new_chat_members`/`left_chat_member` are unreliable in large
  supergroups (esp. leaves) — `chat_member` is the source of truth. Keep the existing
  welcome handler (`handleNewChatMembers`) for greeting only.
- Kick idiom: `BanChatMember` then `UnbanChatMember{OnlyIfBanned:true}` (methods.go:239/246).

## Codebase Anchors

- **Migrations**: goose format (`-- +goose Up` / `-- +goose Down`), sequential; last is
  `00023_add_participant_status.sql` → new file `00024_create_chat_members.sql`.
  Create-table + Down example: `00017_create_user_blacklist.sql`.
- **Repo pattern with upsert**: `persistence/postgres/user_blacklist_repo.go` — `Upsert`
  uses `INSERT ... ON CONFLICT (telegram_user_id) DO UPDATE SET ...`. Domain repo iface in
  `domain/repository/user_blacklist.go`; entity in `domain/entity/user_blacklist.go`.
- **Bot opts**: `NewBot` builds `opts := []telegrambot.Option{ WithDefaultHandler(...) }`
  (bot.go ~:200) — add `WithAllowedUpdates(...)` here. Repos are injected into `Bot`
  struct + `NewBot` params + `cmd/bot/main.go` constructor call.
- **Bot update routing**: `handleUpdate` (bot.go:322) — add a `chat_member` branch;
  `isPublicChat` helper (handlers.go:1526).
- **API already has Telegram + repos**: `http.Config.BotToken` / `PublicChatID`
  (server.go:100), admin route group `r.Group` (server.go:484) guarded by
  `middleware.Auth` + `middleware.RequireRole("admin")`; per-request Telegram client
  precedent `telegrambot.New(token, WithSkipGetMe())` (handler/telegram.go).
- **Multipart upload precedent**: `EventsHandler.UploadGPXFile` (handler/events.go:276)
  — `MaxBytesReader`, `ParseMultipartForm`, `FormFile("file")`, `MultipartForm.RemoveAll()`.
- **Response pkg**: `response.Success/BadRequest/InternalServerError` (http/response/json.go).
- **Frontend**: page under `src/app/(dashboard)/chat-purge/page.tsx`, nav in
  `src/layout/AppSidebar.tsx`; api client via `postForm`/`post`/`get` (src/api/client.ts);
  `FileInput` (`components/form/input/FileInput.tsx`), `Checkbox`
  (`components/form/input/Checkbox.tsx`, `onChange:(checked:boolean)=>void`, `disabled`),
  table primitives `components/ui/table`, confirm via `window.confirm`, inline
  `isLoading`/`error` (no toast lib).

## Tasks

### Phase 1 — Data layer

#### Task 1 — Migration + domain entity + repository interface

- [x] `backend/internal/infrastructure/migrations/00024_create_chat_members.sql` (goose
      Up/Down). Table `chat_members`:
      `telegram_user_id BIGINT PRIMARY KEY, username TEXT NOT NULL DEFAULT '',
      first_name TEXT NOT NULL DEFAULT '', last_name TEXT NOT NULL DEFAULT '',
      is_bot BOOLEAN NOT NULL DEFAULT false, is_admin BOOLEAN NOT NULL DEFAULT false,
      joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`. Down: `DROP TABLE`.
      (No status/left_at — a row exists only while the member is present; leaving deletes it.)
- [x] Domain entity `backend/internal/domain/entity/chat_member.go`:
      `ChatMember{ TelegramUserID int64; Username, FirstName, LastName string; IsBot,
      IsAdmin bool; JoinedAt, UpdatedAt time.Time }` (pure, no json/db tags).
- [x] Repository interface `backend/internal/domain/repository/chat_member.go`:
      `Upsert(ctx, *entity.ChatMember) error`,
      `BulkUpsert(ctx, []*entity.ChatMember) error`,
      `Delete(ctx, telegramUserID int64) error`,
      `GetAll(ctx) ([]*entity.ChatMember, error)`,
      `Count(ctx) (int, error)`.

#### Task 2 — Postgres repository implementation

- [x] `backend/internal/infrastructure/persistence/postgres/chat_member_repo.go` following
      `user_blacklist_repo.go`. `Upsert` = `INSERT ... ON CONFLICT (telegram_user_id) DO
      UPDATE SET username=…, first_name=…, last_name=…, is_bot=…, is_admin=…,
      updated_at=CURRENT_TIMESTAMP` (do NOT overwrite `joined_at` on conflict).
      `BulkUpsert` in a single transaction (batch the seed). `Delete` = `DELETE WHERE
      telegram_user_id=$1` (no error if absent). `GetAll` ordered by first_name.
- [x] Constructor `NewChatMemberRepository(db *sql.DB) repository.ChatMemberRepository`.
- [x] Logging: `WARN` on SQL errors with `operation=` markers (project style).

### Phase 2 — Bot roster maintenance

#### Task 3 — Enable `chat_member` updates (allowed_updates)

- [x] In `NewBot` (bot.go ~:200) add `telegrambot.WithAllowedUpdates(...)` to `opts`
      listing the FULL set the bot needs: `message`, `edited_message`, `callback_query`,
      `my_chat_member`, `chat_member` (use `models.AllowedUpdate*` constants). Verify
      against every `update.*` field `handleUpdate`/`handleProxyUpdate` currently reads so
      nothing is dropped (Message, CallbackQuery). **This is the top risk — an incomplete
      list silently breaks message/callback handling.**
- [x] Logging: on start `INFO Telegram allowed updates configured: chat_member=on`.

#### Task 4 — Bot chat_member handler + roster wiring

- [x] Inject `chatMemberRepo repository.ChatMemberRepository` into the `Bot` struct,
      `NewBot` params, and the `telegram.NewBot(...)` call in `cmd/bot/main.go` (construct
      `postgres.NewChatMemberRepository(db)` there next to the other repos).
- [x] New `backend/internal/infrastructure/telegram/chat_member_tracker.go`:
      `handleChatMemberUpdate(ctx, upd *models.ChatMemberUpdated)` — ignore if not
      `isPublicChat(upd.Chat.ID)`; read `upd.NewChatMember` status:
      - present statuses (`member`, `administrator`, `creator`, `restricted` with
        `IsMember=true`) → `Upsert` (map `IsAdmin` = administrator/creator, `IsBot` from
        `upd.NewChatMember`'s user); 
      - gone statuses (`left`, `kicked`, `restricted` with `IsMember=false`) →
        `Delete(userID)`.
      Extract the affected user from the appropriate `models.ChatMember*` variant.
- [x] Wire into `handleUpdate` (bot.go): `if update.ChatMember != nil {
      b.handleChatMemberUpdate(ctx, update.ChatMember); return }` placed before the
      private-chat filter (same position rationale as the admin hook).
- [x] Logging: `INFO chat member upserted: chat=public target_user_id=%d is_admin=%t` /
      `INFO chat member removed: chat=public target_user_id=%d reason=%s`.

### Phase 3 — Seed + Purge API

#### Task 5 — CSV seed endpoint (admin panel upload)

- [x] `backend/internal/infrastructure/http/handler/chat_members.go`,
      `ChatMembersHandler.Import` — `POST /api/chat-members/import`, multipart `file`
      (MaxBytesReader ~10MB, ParseMultipartForm, RemoveAll defer). Parse CSV
      (`encoding/csv`, strip UTF-8 BOM, header
      `user_id,username,first_name,last_name,is_bot,is_deleted,role,joined_at`, skip
      malformed rows with a WARN count, skip `is_deleted=1`). Map `role in
      (admin,creator)` → `IsAdmin=true`. Call `chatMemberRepo.BulkUpsert`.
      `response.Success` with `{ imported, skipped_rows, total_in_table }`.
- [x] `GET /api/chat-members/summary` → `{ total, admins, bots }` for the page header.
- [x] Register both in the admin `r.Group` (server.go:484); wire the handler +
      `chatMemberRepo` in `NewServer` (add `postgres.NewChatMemberRepository(db)` in
      `cmd/api/main.go` and pass through `NewServer`).
- [x] Logging: `INFO chat members imported: imported=%d skipped_rows=%d`.

#### Task 6 — Purge candidate query + kick adapter + execute command

- [x] Query `GetChatPurgeCandidatesHandler` (`application/query`):
      `Handle(ctx, eventID uint) (dto.ChatPurgeCandidatesResult, error)` —
      `chatMemberRepo.GetAll` minus bots (`IsBot`), minus admins (`IsAdmin`), minus gift
      owners (`giftRepo.FindByEvent`). Optional reason via `participantRepo.FindByEvent`
      (finished / registered / not-participant). Returns only kickable candidates
      (all pre-selected) + `ProtectedGiftOwners` count. Ctor
      `NewGetChatPurgeCandidatesHandler(chatMemberRepo, giftRepo, participantRepo)`.
- [x] Kick adapter `backend/internal/infrastructure/telegram/chat_member_kicker.go`:
      `ChatMemberKicker` over a `go-telegram/bot` client, `Kick(ctx, chatID, userID) error`
      = ban then `UnbanChatMember{OnlyIfBanned:true}`; small `kickerAPI` interface for
      fakes (mirror `giftNotificationAPI`). Ctor `NewChatMemberKickerFromToken(token)`
      (WithSkipGetMe). Classify "user not found / not a member" as sentinel
      `ErrMemberNotInChat` (non-fatal → Skipped).
- [x] Command `ExecuteChatPurgeHandler` (`application/command`):
      `Handle(ctx, {EventID, UserIDs []int64}) (result, error)`. **Protected guard
      (critical)**: re-query `giftRepo.FindByEvent`; drop any submitted gift owner,
      count as `Protected`, never kick. Active-event guard (nil → typed error). Kick
      remaining ids with ~50ms context-aware inter-user delay (ban/unban are admin
      actions, not messages — no 1/sec limit); on success also
      `chatMemberRepo.Delete(userID)` directly (belt-and-suspenders; the bot's
      `chat_member` update will also delete it, but the API shouldn't rely on cross-process
      timing). Collect `{Kicked, Failed, Skipped, Protected}` + per-failure detail.
- [x] Logging: `INFO chat purge executed: event_id=%d requested=%d kicked=%d failed=%d skipped=%d protected=%d`.

#### Task 7 — Purge HTTP endpoints + wiring

- [x] In `chat_members.go` (or a `chat_purge.go`): `GET /api/chat-purge/candidates`
      (loads active event, calls Task 6 query, returns
      `{ event_name, candidates[], protected_gift_owners }`) and
      `POST /api/chat-purge/execute` (`{ user_ids: []int64 }` → Task 6 command → result).
- [x] Register in the admin `r.Group`; build `ChatMemberKicker` from `cfg.BotToken` +
      `cfg.PublicChatID` in `NewServer` (guard: missing token/chat → handler returns a
      clear "функция недоступна" error). No active event → 409/400.

### Phase 4 — Frontend + tests

#### Task 8 — Chat Purge admin page

- [x] `frontend/src/api/chatMembers.ts`: `importCsv(file)` via `postForm` →
      `/api/chat-members/import`; `getSummary()` → `/api/chat-members/summary`;
      `getCandidates()` → `/api/chat-purge/candidates`;
      `executePurge(userIds)` → `/api/chat-purge/execute`. Types in `src/types` or inline.
- [x] `frontend/src/app/(dashboard)/chat-purge/page.tsx`:
      - Top: roster summary ("В чате: N · админов: A · ботов: B") + "Обновить список из
        CSV" `FileInput` → `importCsv` → refresh summary + candidates.
      - Candidate table (ui/table): checkbox | имя (@username) | причина. All rows
        kickable, **checked by default**; header shows `event_name` + "Кандидатов: N ·
        Защищено обладателей приза: K".
      - "Кикнуть выбранных (N)" → `window.confirm` → `executePurge`. **Long request**:
        may run ~15-30s+; blocking spinner "Выполняется удаление, не закрывайте вкладку…",
        button disabled until resolved. Result summary "Кикнуто: N, ошибок: M,
        пропущено: S, защищено: K". Inline `isLoading`/`error` (no toast lib).
- [x] Sidebar nav entry "Чистка чата" in `frontend/src/layout/AppSidebar.tsx`.

#### Task 9 — Tests

- [x] Repo test (`postgres` pkg, follow `participant_repo_test.go`/`gift_repo_test.go` DB
      test style): Upsert insert+update (joined_at preserved on conflict), Delete,
      BulkUpsert, GetAll ordering, Count.
- [x] Bot tracker test (`telegram` pkg, fake repo): join/promote → Upsert with correct
      IsAdmin/IsBot; leave/kick → Delete; update outside public chat ignored;
      `my_chat_member` not misrouted.
- [x] CSV seed parser test: BOM header, valid rows, malformed skipped, `is_deleted=1`
      skipped, role→IsAdmin mapping, empty file → 0 imported (not an error).
- [x] Purge query test: bots/admins/gift-owners excluded; gift owners counted in
      `ProtectedGiftOwners`; candidates pre-selected; reasons correct.
- [x] Purge command test: protected-user guard drops a submitted gift owner (never
      kicked, counted Protected); `ErrMemberNotInChat` counted Skipped not Failed;
      successful kick triggers `chatMemberRepo.Delete`; no active event → typed error.
- [x] Kicker test (fake `kickerAPI`): ban then unban(OnlyIfBanned); ban/unban errors
      surface; not-in-chat maps to `ErrMemberNotInChat`.
- [x] `cd backend && go test ./...` green + `go vet ./...`; `cd frontend && npm run lint`.

## Dependencies

- Task 2 ← Task 1. Task 4 ← Tasks 1,2,3. Task 5 ← Tasks 1,2.
- Task 6 ← Tasks 1,2. Task 7 ← Task 6. Task 8 ← Tasks 5,7. Task 9 ← Tasks 1-7.

## Commit Plan

1. After Tasks 1-2 — `feat(chat-members): add roster table, entity and repository`
2. After Tasks 3-4 — `feat(bot): track chat membership via chat_member updates`
3. After Tasks 5-7 — `feat(api): add chat member CSV import and purge endpoints`
4. After Tasks 8-9 — `feat(dashboard): add chat purge page and tests`

## Risks / Preflight

- **allowed_updates whitelist (top risk)**: forgetting a currently-used update type in
  Task 3 silently breaks messages/callbacks. Enumerate against `handleUpdate` before
  shipping; test the bot still answers `/start` after the change.
- **Two processes write `chat_members`**: the bot (live updates) and the API (seed +
  delete-on-kick). Both use idempotent upsert/delete on the same PK — safe. The bot may
  briefly re-create a row for someone who rejoins after a kick; expected (kick, not ban).
- Bot already has admin rights with member-restrict permission (confirmed 2026-07-06);
  still handle per-user Telegram errors, `ErrMemberNotInChat` = Skipped.
- **Request duration**: kicking hundreds runs ~15-30s+ in one synchronous request —
  the Caddy read/response timeout must exceed it; frontend keeps a blocking spinner.
- Kicked users can rejoin via the public link (kick, not ban) and will be re-tracked by
  the `chat_member` handler and re-greeted by the welcome handler — expected.

## Out Of Scope

- Membership history / audit log (rows are deleted on leave).
- Automatic/scheduled purges; blacklist writes on kick.
- Backfilling the roster by scraping — initial state comes only from the CSV seed;
  ongoing accuracy comes from live `chat_member` updates.

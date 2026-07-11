# Implementation Plan: Manual Prize Distribution And “My Prizes” Miniapp

Branch: feature/manual-prize-assignment
Created: 2026-07-11
Refined: 2026-07-11

## Goal

Add an administrator-controlled “Manual distribution” mode to gifts. Manual gifts must never enter the automatic prize distribution engine. The gift owner must be able to open a new “My Prizes” Mini App screen, see gifts they added for the active event and choose who receives each manual gift. Administrators must be able to configure the same recipient from the dashboard.

## Settings

- Testing: yes; add backend and frontend regression coverage alongside each behavior change.
- Logging: verbose; use the project’s existing `log.Printf`/`console` severity-prefix style with DEBUG for safe flow details, INFO for completed state changes, WARN for rejected operations, and ERROR for persistence or integration failures. This feature does not add a new logging framework or global `LOG_LEVEL` control.
- Docs: yes; the mandatory completion checkpoint must run through `$aif-docs` and update Swagger plus the existing README Mini App section.

## Product Decisions

- Only administrators can turn “Manual distribution” on or off. The gift owner can only assign, reassign, or clear the recipient after an administrator enables the mode.
- “My Prizes” shows all gifts created by the authenticated Telegram user for the active event, including pending-review gifts.
- A manual gift may be left without a recipient. This means “awaiting assignment”.
- A recipient may be any registered participant of the same event, including the donor, a participant without a result, DNF, or disqualified. Manual assignment intentionally bypasses automatic gender, bike, criteria, place, and result-status rules.
- Assignment is allowed before gift approval. Review status still controls visibility in the public gift catalog, but does not control owner/admin recipient selection.
- One gift can have at most one manual recipient. A participant may receive multiple manual gifts.
- Deleting a recipient participant clears the assignment with `ON DELETE SET NULL`; it does not delete the gift.
- Automatic distribution and its statistics remain automatic-only. Manual assignments are shown in gift management as a separate state and are not disguised as automatic `match_reason` rows or `unassigned_slots`.
- The participants list column “Получит приз” shows the total number of automatic slots plus persisted manual gifts assigned to that participant. Pending manual gifts count because an explicit recipient is already selected; manual gifts cannot be double-counted because they are excluded from automatic distribution.
- The public gift catalog may expose `manual_distribution`, but must not expose recipient identity. Recipient data is returned only by authenticated owner/admin read models.
- Do not revive the legacy `prize_assignments` subsystem: its table was removed by migration `00014`, its routes are disabled, and it lacks the ownership/manual-mode invariants required here.
- Audit history (`assigned_by`, previous recipients, timestamps) and multi-recipient manual gifts are out of scope.

## Current Findings

- `Gift` is the current source of truth for prize metadata; automatic output is computed dynamically and is not persisted.
- The canonical automatic engine is `backend/internal/application/query/prize_distribution_engine.go`, orchestrated by `get_prize_distribution.go`.
- `GiftRepository.FindByUser` is not event-scoped, so it cannot safely power the active-event “My Prizes” screen.
- Mini App authentication already validates `X-Telegram-Init-Data` and exposes the signed Telegram user through request context. Owner identity must come from that context, never from a request body.
- `/miniapp/leaderboard` contains only participants with results and cannot be reused as the recipient picker. The public participant DTO also exposes more data than the Mini App needs.
- Mini App session currently turns a legitimate “Telegram user is not registered as a participant” result into HTTP 500. Gift donors do not have to be participants, so this must be corrected for “My Prizes”.
- Active-event lookup currently has a production/test mismatch where a missing active event can become HTTP 500 instead of 404.
- The dashboard gift list derives “distributed” only from automatic distribution, so manual gifts need explicit statuses to avoid misleading labels.
- `ParticipantsHandler.GetAll` exposes and sorts by `prizes_count`, but never populates it before filtering, sorting, and pagination; the current “Получит приз” column therefore renders empty for every participant.
- Frontend tests use Vitest but do not currently include React Testing Library/jsdom. Keep new frontend regression logic in pure utilities/API helpers unless test infrastructure is deliberately expanded.

## Data And API Direction

Persist the one-recipient manual state on `gifts`:

```text
gifts
  manual_distribution                 boolean not null default false
  manual_recipient_participant_id     integer null -> participants(id) on delete set null
```

Database constraints:

- `manual_recipient_participant_id` is null whenever `manual_distribution = false`.
- Add a partial index for non-null manual recipients.
- Enforce same-event assignment in the application layer and in the targeted SQL update where practical; PostgreSQL cannot express the cross-table event equality as a simple `CHECK`.

Protected owner API:

- `GET /api/miniapp/my-gifts` — current Telegram user’s gifts for the active event, all review statuses, with safe recipient summaries.
- `GET /api/miniapp/participants` — minimal recipient options for the active event; no Telegram user ID, notes, results, or admin-only fields.
- `PUT /api/miniapp/my-gifts/{giftId}/recipient` — replace recipient with `{ "participant_id": <id|null> }`.

Protected admin API:

- Extend `PUT /api/gifts/{id}` with presence-aware `manual_distribution` and `manual_recipient_participant_id` fields so the dashboard can save the checkbox and recipient atomically with the rest of the gift form.
- Add protected `GET /api/events/{eventId}/manual-gifts` returning recipient summaries without expanding the public `GiftDTO` recipient surface.
- Preserve the existing `PUT /api/gifts/{id}` response as `GiftDTO`; admin clients refetch the protected manual-gift view after writes instead of changing the established public response contract.

## Commit Plan

- **Commit 1** (after tasks 1-3): `feat(gifts): model manual prize assignments`
- **Commit 2** (after tasks 4-7): `feat(api): add manual prize assignment workflows`
- **Commit 3** (after tasks 8-10): `feat(ui): manage manual prize recipients`
- **Commit 4** (after tasks 11-12): `docs(gifts): document manual prize distribution`

## Tasks

### Phase 1: Domain Model And Persistence

- [x] 1. Add the manual distribution state to the gift aggregate and PostgreSQL schema.
  - Files:
    - `backend/internal/infrastructure/migrations/00028_add_gift_manual_distribution.sql`
    - `backend/internal/domain/entity/gift.go`
    - `backend/internal/domain/repository/gift.go`
    - `backend/internal/infrastructure/persistence/postgres/gift_repo.go`
    - `backend/internal/infrastructure/persistence/postgres/gift_repo_test.go`
  - Deliverable:
    - Add `ManualDistribution`, nullable `ManualRecipientParticipantID`, and an optional loaded `ManualRecipient` relation to `Gift` without persistence or JSON tags.
    - Add the boolean column, nullable participant foreign key, `ON DELETE SET NULL`, the “recipient requires manual mode” check, partial index, and reversible down migration.
    - Carry both columns through every gift INSERT, UPDATE, SELECT, scanner, nullable-field mapper, and sqlmock fixture.
    - Add an event-scoped `FindByUserAndEvent` repository operation for owner reads.
    - Add a targeted recipient update operation that can assign/reassign/clear without overwriting unrelated gift fields and protects the same-event invariant at write time.
    - Cover default automatic gifts, manual gifts without recipients, loaded recipients, assignment clearing, participant deletion behavior, cross-event rejection, and transaction rollback.
  - Logging requirements:
    - Keep the domain entity unlogged.
    - At repository boundaries log DEBUG operation stages and safe IDs, and ERROR failures with `gift_id`, `event_id`, `recipient_participant_id`, and stage.
    - Never log gift descriptions, Telegram init data, or attachment file IDs.
  - Dependencies: none.

- [x] 2. Implement shared manual-assignment invariants and actor-aware application commands.
  - Files:
    - `backend/internal/application/command/update_gift.go`
    - `backend/internal/application/command/update_gift_test.go`
    - `backend/internal/application/command/set_manual_gift_recipient.go`
    - `backend/internal/application/command/set_manual_gift_recipient_test.go`
    - optional shared policy file under `backend/internal/application/command/`
  - Deliverable:
    - Add presence-aware manual fields to the admin update command: omitted preserves, `false` disables and clears, and explicit `null` clears only the recipient.
    - Define and test the complete payload matrix: `manual_distribution=false` plus a non-null recipient is rejected as contradictory; `true` plus a valid recipient is accepted atomically; an omitted flag uses the stored mode; setting a recipient while the stored mode is automatic is rejected; explicit `null` is idempotent.
    - Keep admin gift fields, criteria, manual flag, and recipient replacement atomic in the existing update transaction path.
    - Preserve `place_rule`, gender/bike filters, and criteria while manual mode is enabled so they resume unchanged if an administrator later returns the gift to automatic distribution.
    - Add a targeted command for Mini App recipient replacement using an actor contract that receives the authenticated Telegram user ID from infrastructure.
    - Validate gift existence, owner access for Mini App calls, manual mode, recipient existence, and same-event membership.
    - Make repeating the same assignment idempotent; allow self-assignment and every participant status because automatic eligibility rules do not apply.
    - Return typed errors for not found, ownership denied, non-manual gift, missing participant, and cross-event recipient so transports can map them consistently.
  - Logging requirements:
    - DEBUG command entry and no-op/idempotent decisions.
    - INFO successful flag changes, assignment, reassignment, and clearing with actor type and safe IDs.
    - WARN rejected ownership, non-manual, missing-recipient, and cross-event attempts; ERROR persistence failures.
    - Do not log Telegram init data or personal profile fields.
  - Dependencies: Task 1.

- [x] 3. Add privacy-safe owner/admin read models and queries.
  - Files:
    - `backend/internal/application/dto/gift.go`
    - `backend/internal/application/dto/manual_gift.go`
    - `backend/internal/application/query/get_manual_gifts.go`
    - `backend/internal/application/query/get_manual_gifts_test.go`
    - `backend/internal/application/query/get_miniapp_participants.go`
    - `backend/internal/application/query/get_miniapp_participants_test.go`
  - Deliverable:
    - Add `manual_distribution` to the common gift contract, but keep recipient identity out of the public catalog DTO.
    - Create a protected manual-gift DTO with a minimal recipient summary and review status for owner/admin screens.
    - Add an owner query scoped by authenticated Telegram user plus active event and returning pending and approved gifts.
    - Add an admin event-scoped manual-gifts query for dashboard enrichment.
    - Add a minimal Mini App participant option DTO with internal participant ID, display name, optional username, and status only; do not expose Telegram user ID, notes, registration dates, or result metrics.
    - Return all same-event participants in a deterministic searchable order.
  - Logging requirements:
    - DEBUG query scope and returned counts; INFO successful protected reads only where useful.
    - WARN invalid scope and ERROR repository failures with safe IDs.
    - Do not log participant names, usernames, notes, or result payloads.
  - Dependencies: Tasks 1-2.

### Phase 2: Distribution And Backend APIs

- [x] 4. Exclude manual gifts from every automatic distribution path and lock the statistics contract with tests.
  - Files:
    - `backend/internal/application/query/get_prize_distribution.go`
    - `backend/internal/application/query/prize_distribution_engine.go`
    - `backend/internal/application/query/get_prize_distribution_test.go`
    - `backend/internal/application/query/prize_distribution_place_rule_test.go`
    - `backend/internal/application/query/prize_distribution_e2e_test.go`
    - `backend/internal/application/query/prize_distribution_status_test.go`
    - `backend/internal/application/query/get_stats.go`
    - `backend/internal/application/query/get_stats_test.go`
  - Deliverable:
    - Filter manual gifts immediately after approved gifts are loaded, before criteria loading and slot construction.
    - Add a defensive manual-mode guard in `filterApprovedPrizeGifts` and any retained compatibility matcher.
    - Prove that generic, criteria, explicit-place, and `last_n` manual gifts create no automatic assignment and no `unassigned_slots` entry.
    - Cover mixed automatic/manual events so automatic capacity, priorities, participant counts, and slots remain unchanged.
    - Keep automatic distribution stats automatic-only and make that contract explicit in tests; do not count persisted manual assignments as automatic matches.
  - Logging requirements:
    - DEBUG total approved, excluded manual, and remaining automatic gift counts per event.
    - INFO final automatic distribution summary; WARN inconsistent manual recipient state if encountered; ERROR existing query failures.
    - Do not log gift descriptions or participant profile data.
  - Dependencies: Task 1.

- [x] 5. Fix the participants-list “Получит приз” count and server-side sorting.
  - Files:
    - `backend/internal/infrastructure/http/handler/participants.go`
    - `backend/internal/infrastructure/http/handler/participants_test.go`
    - `backend/internal/infrastructure/http/handler/participants_sort_test.go`
    - `backend/internal/application/dto/participant.go`
    - `backend/internal/domain/repository/gift.go`
    - `backend/internal/infrastructure/persistence/postgres/gift_repo.go`
    - `backend/internal/infrastructure/persistence/postgres/gift_repo_test.go`
    - `frontend/src/components/participants/participantColumns.tsx`
  - Deliverable:
    - Populate `ParticipantDTO.PrizesCount` in `ParticipantsHandler.GetAll`; the field currently remains zero even though the column and sorter already exist.
    - Build one event-level map from automatic distribution, counting `matched_gift_assignments` when present and using legacy `matched_gifts` only as a fallback so the same automatic gift is not counted twice.
    - Add an event-scoped repository aggregation for persisted manual recipients and merge one count per assigned manual gift, including pending-review manual gifts.
    - Populate totals before search, `prizes_count` sorting, and pagination so ordering and values are correct across the complete result set, not only the visible page.
    - Rely on Task 4’s exclusion guarantee to prevent a manual gift from appearing in both automatic and manual counts.
    - Treat aggregation failure as an HTTP error instead of silently returning a misleading zero count.
    - Keep the column label “Получит приз”; render the total count and preserve the current empty state for zero.
  - Testing requirements:
    - Cover zero prizes, one/multiple automatic slots, one/multiple manual gifts, mixed automatic/manual totals, pending manual assignments, legacy automatic fallback, no double counting, and ascending/descending sorting before pagination.
  - Logging requirements:
    - DEBUG event-level automatic/manual count summaries and returned participant counts.
    - WARN inconsistent assignment rows and ERROR aggregation failures with event/gift/participant IDs only.
    - Do not log participant profile fields or gift descriptions.
  - Dependencies: Tasks 1, 3, and 4.

- [x] 6. Expose the administrator manual-distribution API through existing protected gift management.
  - Files:
    - `backend/internal/infrastructure/http/handler/gifts.go`
    - `backend/internal/infrastructure/http/handler/gifts_test.go`
    - `backend/internal/infrastructure/http/handler/gifts_cache_test.go`
    - `backend/internal/infrastructure/http/server.go`
    - `backend/cmd/api/main.go` if constructor wiring changes are required
  - Deliverable:
    - Decode `manual_distribution` with omitted/true/false semantics and `manual_recipient_participant_id` with omitted/value/null semantics.
    - Route both fields through the validated application update path while preserving the existing `GiftDTO` response contract.
    - Add protected `GET /api/events/{eventId}/manual-gifts` for manual gift management without adding recipient identity to public gift endpoints; the frontend refetches this view after a write.
    - Map malformed/contradictory payloads to 400, missing gifts/participants to 404, and non-manual/cross-event conflicts to 409; admin authentication remains the existing JWT middleware responsibility.
    - Invalidate the file-backed public Mini App catalog when `manual_distribution` changes on an approved gift. Recipient-only changes do not alter the public DTO and must not add a new invalidation path.
    - Test enable, disable-and-clear, assign, reassign, explicit clear, missing participant, cross-event participant, and malformed payload cases.
  - Logging requirements:
    - DEBUG decoded field-presence flags and safe IDs.
    - INFO successful admin changes with admin claim ID, gift ID, and recipient ID.
    - WARN validation/conflict responses and ERROR command failures; never log JWTs or request bodies.
  - Dependencies: Tasks 2-3.

- [x] 7. Add authenticated Mini App owner endpoints and fix session/active-event prerequisites.
  - Files:
    - `backend/internal/infrastructure/http/handler/miniapp.go`
    - `backend/internal/infrastructure/http/handler/miniapp_test.go`
    - `backend/internal/infrastructure/http/server.go`
    - `backend/internal/domain/repository/event.go`
    - `backend/internal/infrastructure/persistence/postgres/event_repo.go`
    - `backend/internal/infrastructure/persistence/postgres/event_repo_test.go`
    - `backend/internal/application/query/get_participants.go`
  - Deliverable:
    - Add `GET /api/miniapp/my-gifts`, `GET /api/miniapp/participants`, and `PUT /api/miniapp/my-gifts/{giftId}/recipient` under existing Telegram init-data middleware.
    - Derive owner ID only from verified request context and active event only from the server; ignore any client-supplied owner/event identity.
    - Return another owner’s gift as 404 to avoid confirming its existence.
    - Accept `{ "participant_id": null }` for clearing and make repeated writes idempotent.
    - Treat the existing wrapped `repository.ErrParticipantNotFound` as a valid Mini App session with no `my_result_participant_id`; do not change genuine participant-query failures into successful responses.
    - Add `repository.ErrNoActiveEvent`, return it from PostgreSQL `FindActive`, and map it through `errors.Is` to 404 for session and every new endpoint while preserving 500 for actual database failures.
    - Map malformed payloads to 400, foreign-owner/missing gift or participant to 404, and non-manual/cross-event assignment to 409.
    - Test all review statuses, empty lists, all participant/result statuses, ownership, same-event validation, malformed input, auth failures, assignment, reassignment, and clearing.
  - Logging requirements:
    - DEBUG active-event and ownership resolution using safe IDs.
    - INFO protected list counts and completed recipient changes.
    - WARN authentication/authorization/validation rejections and ERROR repository or command failures.
    - Never log raw `X-Telegram-Init-Data` or participant personal fields.
  - Dependencies: Tasks 2-3.

### Phase 3: Dashboard And Mini App UX

- [ ] 8. Add administrator checkbox, recipient picker, and clear manual statuses to gift management.
  - Files:
    - `frontend/src/types/index.ts`
    - `frontend/src/api/gifts.ts`
    - `frontend/src/components/gifts/GiftEditForm.tsx`
    - `frontend/src/components/gifts/GiftsTable.tsx`
    - `frontend/src/app/(dashboard)/gifts/[id]/page.tsx`
    - `frontend/src/app/(dashboard)/gifts/[id]/edit/page.tsx`
    - `frontend/src/app/(dashboard)/gifts/page.tsx`
    - `frontend/src/utils/manualGiftAssignment.ts`
    - `frontend/src/utils/manualGiftAssignment.test.ts`
  - Deliverable:
    - Add “Ручное распределение” with helper text “Приз не участвует в автоматическом распределении”.
    - When enabled, show a searchable same-event participant picker with an explicit “Получатель пока не выбран” option.
    - Load candidate participants once with the existing unpaginated `participantsApi.getByEvent(gift.event_id)` contract and filter the in-memory options; do not copy `GlobalSearch`’s request-on-every-search-change behavior.
    - Save checkbox, recipient value/null, and existing gift edits without losing place rules, criteria, or review status.
    - On disable, hide the picker and send an explicit recipient clear.
    - Enrich gift list/detail with protected manual read data and distinguish pending, manual-unassigned, manual-assigned, automatic-assigned, and automatic-unassigned states.
    - Keep the existing quick-approve flow from dropping the new fields.
    - Provide actionable mapped errors for stale participant, cross-event selection, validation failure, and server failure.
  - Testing requirements:
    - Cover pure picker option formatting, status derivation, payload presence/null semantics, and API error mapping with Vitest.
    - Do not add React Testing Library/jsdom unless component behavior cannot be covered safely through existing utilities and build checks.
  - Logging requirements:
    - DEBUG client load/save state with gift and participant IDs only.
    - INFO successful manual configuration save; WARN recoverable API errors; ERROR unexpected failures.
    - Do not log auth tokens, full API bodies, names, or Telegram IDs.
  - Dependencies: Tasks 3 and 6.

- [ ] 9. Build the Mini App “Мои призы” menu item and owner assignment screen.
  - Files:
    - `frontend/src/components/miniapp/MiniappTabs.tsx`
    - `frontend/src/app/(miniapp)/miniapp/my-gifts/page.tsx`
    - `frontend/src/app/(miniapp)/miniapp/my-gifts/loading.tsx`
    - `frontend/src/app/(miniapp)/miniapp/my-gifts/error.tsx`
    - `frontend/src/components/miniapp/MyGiftCard.tsx`
    - `frontend/src/components/miniapp/MyGiftRecipientSelect.tsx`
    - `frontend/src/api/miniapp.ts`
    - `frontend/src/api/miniapp.test.ts`
    - `frontend/src/types/index.ts`
    - `frontend/src/utils/miniappMyGifts.ts`
    - `frontend/src/utils/miniappMyGifts.test.ts`
  - Deliverable:
    - Add a fixed “Мои призы” navigation item with a real inline SVG icon and correct active-route behavior, independent of whether “Мой результат” exists.
    - Reuse the layout-level `MiniappSessionProvider`/`useMiniappSession`; the new screen and navigation must not issue a second `/api/miniapp/session` request.
    - Keep the fixed navigation offset and `layout.tsx` content padding based on the same `env(safe-area-inset-bottom)` formula, and verify that three/four navigation items remain usable on narrow Telegram viewports.
    - Show every gift the authenticated user added for the active event, with review status, automatic/manual mode, and current recipient.
    - For automatic gifts show a read-only explanation; for manual gifts provide recipient search/select, save, reassignment, and clear actions.
    - Include loading, empty, save-in-progress, retry, stale-recipient, and no-active-event states consistent with Telegram theme variables and current Mini App layout.
    - Keep state server-authoritative after mutations; optimistic UI is allowed only with rollback and refetch on failure.
    - Ensure the recipient picker can select participants without results and does not expose admin-only participant data.
  - Testing requirements:
    - Cover API URL/method/header/body contracts, including `participant_id: null`.
    - Cover pure presentation/status logic, recipient replacement, rollback/refetch decisions, and empty/error labels with Vitest.
  - Logging requirements:
    - DEBUG fetch/mutation lifecycle with safe IDs and counts.
    - INFO completed assignment/clear actions; WARN recoverable API failures; ERROR unexpected client failures.
    - Never log Telegram init data or participant profile details.
  - Dependencies: Tasks 3 and 7.

- [ ] 10. Make automatic-distribution and manual-assignment presentation consistent across dashboard and public Mini App surfaces.
  - Files:
    - `frontend/src/app/(dashboard)/prize-distribution/page.tsx`
    - `frontend/src/utils/prizeDistribution.ts`
    - `frontend/src/utils/prizeDistribution.test.ts`
    - `frontend/src/components/gifts/GiftsTable.tsx`
    - `frontend/src/components/miniapp/GiftCatalogTable.tsx`
    - `frontend/src/components/miniapp/GiftDetailView.tsx`
    - `frontend/src/utils/giftPlaceRule.ts`
    - `frontend/src/utils/giftPlaceRule.test.ts`
  - Deliverable:
    - Label the prize-distribution screen and its counters as automatic-only and explain that manual gifts are excluded.
    - Do not inject manual recipients into automatic `matched_gift_assignments`, `match_reason`, or `unassigned_slots`.
    - Ensure gift table badges use protected manual assignment data instead of inferring manual status from the automatic response.
    - In the public Mini App catalog and gift detail, label manual gifts as “Ручное распределение” and do not present gender, bike, criteria, or place metadata as automatic recipient-selection rules for those gifts.
    - Keep recipient identity completely absent from public catalog/detail contracts and rendering.
    - Add regression tests for automatic-only counters, manual status formatting, and public manual-gift condition formatting.
  - Logging requirements:
    - No new runtime logs in pure formatting utilities.
    - DEBUG only at data-loading boundaries with event ID and counts; WARN mismatched/incomplete protected assignment data.
  - Dependencies: Tasks 4, 5, 8, and 9.

### Phase 4: Documentation And Verification

- [ ] 11. Update API and user-facing documentation through the mandatory `$aif-docs` checkpoint.
  - Files:
    - `backend/docs/swagger.yaml`
    - `README.md`
  - Deliverable:
    - Document gift manual fields, protected admin management response, all three Mini App endpoints, nullable recipient replacement, auth requirements, typed errors, and privacy-safe participant/recipient schemas.
    - State explicitly that `/events/{eventId}/prize-distribution` and its statistics exclude manual gifts.
    - Document that participant `prizes_count` is the total automatic-slot plus persisted manual-recipient count used by the “Получит приз” column.
    - Update the existing README Mini App tabs section with “Мои призы” and its owner/manual-assignment behavior.
    - Remove or clearly deprecate the stale Swagger `/prize-assignments` endpoints that are not routed and target a table removed by migration `00014`; do not present them as the new implementation.
    - Do not create ad hoc report or instruction Markdown files.
  - Logging requirements:
    - No runtime logging changes. Documentation examples must not contain real Telegram init data, JWTs, participant identities, or other secrets.
  - Dependencies: Tasks 6-10.

- [ ] 12. Verify migrations, backend behavior, frontend contracts, and Docker runtime end to end.
  - Files:
    - only implementation/test files that require corrections discovered by verification
  - Deliverable:
    - Run focused Go tests while implementing, then `GOCACHE=/private/tmp/gravel_bot-go-build go test ./...` from `backend`.
    - Run `npm test -- --run`, `npm run build`, and `npm run lint` from `frontend`; separate known baseline lint noise from new regressions.
    - Run migration up/down/up against PostgreSQL and verify constraints, `ON DELETE SET NULL`, and existing-gift defaults.
    - Rebuild and start the relevant PostgreSQL, migrate, backend API, and frontend services with Docker Compose; use a safe local `POSTGRES_PORT` override if the default is occupied.
    - Exercise admin enable/assign/reassign/clear/disable, Mini App owner read/assign/clear, foreign-owner rejection, cross-event rejection, donor-without-participant session, no-active-event 404, automatic distribution exclusion, and “Получит приз” totals/sorting across pagination.
    - Confirm public gift/catalog responses do not expose manual recipient identity.
    - Finish with `git diff --check` and inspect the complete branch diff against `origin/main`.
  - Logging requirements:
    - Verify DEBUG/INFO/WARN/ERROR prefixes are emitted at the planned boundaries using the existing logging style; do not claim a global runtime `LOG_LEVEL` control because the project does not currently provide one.
    - Confirm logs contain no Telegram init data, JWTs, descriptions, file IDs, names, usernames, or other unnecessary personal data.
  - Dependencies: Tasks 1-11.

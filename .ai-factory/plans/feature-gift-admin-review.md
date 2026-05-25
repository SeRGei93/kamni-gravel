# Implementation Plan: Gift Admin Review And Telegram Confirmation

Branch: feature/gift-admin-review
Created: 2026-05-24

## Goal

Make Telegram the only source for gift creation, keep the current Telegram gift questions, add explicit user confirmation before persistence, make gift creation transactional, and require admin review before gifts can participate in prize distribution.

## Requirements

- Admins must not create gifts manually.
- Gifts must remain strictly tied to the Telegram user ID that created them.
- Keep the existing Telegram gift flow shape: gender filter, bike type filter, description, and photos are still collected from the user.
- Add a confirmation step before saving the gift.
- Persist gift + attachments atomically in one transaction.
- Add an admin review status so new Telegram gifts do not participate in distribution until approved.
- Add admin filtering/status UI and a visible notification when there are new unreviewed gifts.
- Prize distribution must ignore unapproved gifts.
- Criteria are more important than place matching; place is a secondary signal because final participant counts are unknown.
- Do not add `github.com/go-telegram/ui` or any UI library dependency for bot buttons.

## Settings

- Testing: yes
- Logging: standard
- Docs: yes, update Swagger/API docs only for changed public contracts

## Current Findings

- Frontend currently allows admin gift creation through `frontend/src/components/gifts/CreateGiftModal.tsx`, `frontend/src/api/gifts.ts`, and the `/gifts` page button.
- Backend exposes protected admin create through `POST /api/events/{eventId}/gifts` in `backend/internal/infrastructure/http/server.go`.
- `command.AddGiftHandler` is already shared by HTTP and Telegram; after this change it should be used by Telegram creation only.
- Gift source binding is already mostly correct: `gifts.user_id` stores the Telegram user ID and references `users(id)`.
- Current creation is not transactional: `AddGiftHandler` creates the gift and then inserts attachments one by one.
- Current Telegram flow saves on `finish_gift` / `skip_photos`; there is no review screen before persistence.
- Current distribution matches every eligible gift and does not know admin review status.
- Current gift update logic lives mostly in the HTTP handler and updates gift fields separately from criteria. This is risky once approval status is introduced, because a gift could be approved while criteria updates fail.
- Current `UpdateGiftRequest` cannot distinguish omitted `place` from explicit JSON `null`; clearing place needs custom request parsing or a field-presence helper.
- The next migration number is `00015`; migration numbering currently skips `00012`, but existing files end at `00014_drop_prize_assignments.sql`.
- Existing data should not silently disappear from distribution after migration. Existing gifts should be backfilled as approved, while new gifts default to pending review.

## Tasks

### Phase 1: Review Status Model And Persistence

- [x] 1. Add gift review status to the domain/API model and database.
  - Files:
    - `backend/internal/domain/entity/gift.go`
    - `backend/internal/application/dto/gift.go`
    - `backend/internal/infrastructure/migrations/00015_add_gift_review_status.sql`
    - `backend/internal/infrastructure/persistence/postgres/gift_repo.go`
    - `frontend/src/types/index.ts`
  - Deliverable:
    - Add a review status field, preferably `review_status`, with values `pending_review` and `approved`.
    - Define status constants and validation in the domain layer, for example `entity.GiftReviewStatusPendingReview` and `entity.GiftReviewStatusApproved`; do not scatter raw status strings through handlers.
    - New gifts must default to `pending_review`.
    - Existing gifts must be backfilled to `approved` in the migration to preserve current distribution behavior.
    - Migration should add a `CHECK` constraint and an index useful for event/status filtering, for example `(event_id, review_status)`.
    - Repository scans/selects for `FindByID`, `FindByEvent`, and `FindByUser` must include `review_status`; `FindByUser` currently returns a reduced gift shape and should be brought in line enough that DTOs do not emit misleading zero values.
    - DTO and TypeScript types must expose the status to the admin UI.
    - Do not add transport/db tags to domain entities.
  - Logging requirements:
    - No runtime logs required for pure model/migration mapping.
    - Repository errors should continue to return contextual errors to callers without logging secrets.
  - Dependencies: none.

- [x] 2. Make gift creation with attachments transactional.
  - Files:
    - `backend/internal/domain/repository/gift.go`
    - `backend/internal/infrastructure/persistence/postgres/gift_repo.go`
    - `backend/internal/application/command/add_gift.go`
  - Deliverable:
    - Add an explicit transactional repository operation, for example `CreateWithAttachments(ctx, gift, []*entity.GiftAttachment)`, or equivalent scoped API that uses domain entities rather than application DTO types.
    - `AddGiftHandler` must call the transactional operation so gift creation and attachment insertion commit or roll back together.
    - `AddGiftHandler` must set new gifts to `pending_review`.
    - Keep attachment validation at the boundary: only accepted file types from Telegram/API data should become `entity.GiftAttachment`.
    - Remove or stop using the non-transactional `Create` + `AddAttachment` path for gift creation if it is no longer needed.
    - Keep `context.Context` first and use native SQL transactions.
  - Logging requirements:
    - Log transaction begin/commit/rollback failures at the infrastructure boundary only when errors occur.
    - Include gift/event/user IDs in error context; do not log Telegram file URLs or tokens.
  - Dependencies: Task 1.

### Phase 2: Telegram Confirmation Flow

- [x] 3. Add a Telegram confirmation state before gift persistence.
  - Files:
    - `backend/internal/infrastructure/telegram/session/manager.go`
    - `backend/internal/infrastructure/telegram/handler/gift.go`
    - `backend/internal/infrastructure/telegram/keyboard/builder.go`
    - `backend/internal/infrastructure/telegram/handlers.go`
    - `backend/internal/infrastructure/telegram/keyboard/builder_test.go`
    - `backend/internal/infrastructure/telegram/handler/gift_test.go`
  - Deliverable:
    - Keep the existing user questions: gift gender, gift bike type, gift description, and photos.
    - Change `finish_gift` / `skip_photos` so they show a summary instead of immediately saving.
    - Add `StateAwaitingGiftConfirmation` to the session manager.
    - Split current `FinishAddGift` behavior into summary-building and confirmed persistence paths, for example `PreviewGift` and `ConfirmAddGift`, so the save path cannot be reached accidentally from `finish_gift`.
    - Add callback buttons such as `confirm_gift`, `restart_gift`, and `cancel`.
    - Persist only after `confirm_gift`.
    - `restart_gift` should reset the gift flow cleanly and start from the first gift question.
    - If persistence fails, keep the session data available so the user can retry confirmation or cancel instead of losing the entered gift.
    - Make session data reads type-safe enough to avoid panics from malformed `gift_attachments`, `gift_gender`, `gift_bike_type`, or `event_id` values.
    - The persisted `UserID` must always come from the Telegram update/session user ID, not user-provided text.
  - Logging requirements:
    - Log confirmation save failures with user ID and event ID.
    - Log invalid/missing session data branches with user ID and state.
    - Do not log private gift description text or Telegram token.
  - Dependencies: Task 2.

### Phase 3: Remove Admin Creation Surface

- [x] 4. Remove admin gift creation from the frontend.
  - Files:
    - `frontend/src/app/(dashboard)/gifts/page.tsx`
    - `frontend/src/components/gifts/CreateGiftModal.tsx`
    - `frontend/src/api/gifts.ts`
    - `frontend/src/types/index.ts`
  - Deliverable:
    - Remove the “Добавить подарок” button and create modal usage from `/gifts`.
    - Remove or deprecate the frontend `giftsApi.create` helper and `CreateGiftRequest` type if no longer used.
    - The gifts page must communicate by layout/status that gifts arrive from Telegram and are reviewed by admins, without adding a marketing/explanatory page.
  - Logging requirements:
    - Keep existing `console.error` for failed load/update/delete operations.
    - No new client-side noisy logs for normal UI state changes.
  - Dependencies: Task 1.

- [x] 5. Remove or block admin gift creation from the HTTP API.
  - Files:
    - `backend/internal/infrastructure/http/server.go`
    - `backend/internal/infrastructure/http/handler/gifts.go`
    - `backend/internal/infrastructure/http/handler/gifts_test.go` if handler tests are added
    - `backend/docs/swagger.yaml`
  - Deliverable:
    - Remove the protected `POST /api/events/{eventId}/gifts` route or make it return a clear non-success status if backward compatibility demands route presence.
    - Prefer removing the route entirely because the new product rule says admin creation is not allowed.
    - Remove unused HTTP create request structures and handler dependencies where practical, including the `addGiftHandler` field/constructor parameter from `GiftsHandler` and the API server wiring if it becomes unused in the API process.
    - Keep Telegram creation through `AddGiftHandler`.
    - Update Swagger so admin-create is no longer advertised.
  - Logging requirements:
    - If the route is kept as blocked, log attempted admin create at INFO/WARN with admin/request context but without request body.
    - If the route is removed, no runtime logging is required.
  - Dependencies: Task 4.

### Phase 4: Admin Review API And UI

- [x] 6. Move gift admin updates into an application command and make review updates atomic.
  - Files:
    - `backend/internal/application/command/update_gift.go`
    - `backend/internal/infrastructure/http/handler/gifts.go`
    - `backend/internal/infrastructure/http/server.go`
    - `backend/internal/application/dto/gift.go`
    - `backend/internal/domain/repository/gift.go`
    - `backend/internal/domain/repository/criteria.go`
    - `backend/internal/infrastructure/persistence/postgres/gift_repo.go`
    - `backend/internal/infrastructure/persistence/postgres/gift_criteria_repo.go`
    - `frontend/src/api/gifts.ts`
    - `frontend/src/types/index.ts`
  - Deliverable:
    - Introduce an application-layer `UpdateGiftHandler`/command for admin edits so HTTP stays thin and review rules are not embedded in transport code.
    - Extend gift update to accept `review_status`.
    - Validate allowed review status values server-side using the domain constants from Task 1.
    - Validate gender and bike filters server-side rather than trusting frontend values.
    - Keep description, gender filter, bike type filter, place, and criteria editing intact.
    - Make gift field update + criteria replacement + review status update atomic, either through one repository transaction operation or a small infrastructure transaction helper consistent with project boundaries.
    - Fix place clearing while touching update semantics: explicit JSON `place: null` from the frontend should clear the stored place, while omitted `place` should leave it unchanged. Use a field-presence approach such as a custom nullable request field or raw JSON parsing; `*int` alone is not enough.
    - Do not allow accidental approval without preserving/admin-submitting the current criteria/filter payload from the edit UI.
  - Logging requirements:
    - Log invalid review status attempts at WARN/INFO with gift ID.
    - Log update failures with gift ID and operation stage, including criteria replacement failures, without logging full descriptions.
    - Do not log full gift descriptions.
  - Dependencies: Task 1.

- [x] 7. Add review-status filtering to gift list reads.
  - Files:
    - `backend/internal/application/query/get_gifts.go`
    - `backend/internal/infrastructure/http/handler/gifts.go`
    - `backend/internal/infrastructure/persistence/postgres/gift_repo.go`
    - `backend/internal/domain/repository/gift.go`
    - `frontend/src/api/gifts.ts`
  - Deliverable:
    - Support optional `review_status` filtering on `GET /api/events/{eventId}/gifts`.
    - Keep default behavior returning all gifts for the selected event so the page can compute total and pending counts.
    - Validate invalid status filters and return a 400 rather than silently returning misleading data.
    - Preserve attachment and criteria loading behavior for filtered and unfiltered reads.
  - Logging requirements:
    - Log invalid list filter values at INFO/WARN with event ID.
    - Log repository/query failures with event ID and filter value, not gift descriptions.
  - Dependencies: Task 1.

- [x] 8. Add review status, filters, and pending notification to the gifts admin page.
  - Files:
    - `frontend/src/app/(dashboard)/gifts/page.tsx`
    - `frontend/src/components/gifts/GiftsTable.tsx`
    - `frontend/src/components/gifts/EditGiftModal.tsx`
    - `frontend/src/constants/options.ts`
    - `frontend/src/types/index.ts`
  - Deliverable:
    - Show a status badge for each gift: “Новый / на проверке” and “Проверен”.
    - Add a filter control: all, pending review, approved.
    - Show a visible notification/banner when the selected event has pending gifts.
    - Allow admins to mark a gift as approved from edit UI or a focused table action.
    - If a table quick action approves a gift, it must preserve existing description, filters, place, and criteria instead of submitting a partial payload that clears fields.
    - The “Распределен” column should not show pending gifts as ordinary failed matches; pending gifts should read as “На проверке” / “Не участвует” until approved.
    - Preserve existing TailAdmin table/modal patterns and photo preview behavior.
  - Logging requirements:
    - Keep failed load/update/delete console errors.
    - Do not log normal filter changes.
  - Dependencies: Tasks 4, 6, and 7.

### Phase 5: Prize Distribution Rules

- [x] 9. Exclude unapproved gifts from prize distribution and make criteria priority explicit.
  - Files:
    - `backend/internal/application/query/get_prize_distribution.go`
    - `backend/internal/application/dto/prize_distribution.go` if response labels need changes
    - `frontend/src/app/(dashboard)/prize-distribution/page.tsx` if labels need changes
  - Deliverable:
    - Distribution must ignore gifts whose `review_status` is not `approved`.
    - Candidate filtering order should be explicit and deterministic:
      - first apply approved status, gender filter, and bike filter;
      - then classify candidates by criteria and place;
      - use criteria-matching gifts before place-only gifts;
      - use place as an additional constraint/tie-breaker inside criteria matches, not as the dominant reason to drop a criteria-relevant gift unless the admin explicitly configured both criteria and place.
    - When a participant has criteria-matching approved gifts, do not also auto-bundle lower-priority place-only/generic gifts for the same participant unless the chosen algorithm deliberately allows all same-priority matches.
    - Document the selected priority behavior in code comments near the matching function because this is key business logic.
    - Keep gender and bike filters active.
    - Keep behavior deterministic when multiple gifts match the same result.
  - Logging requirements:
    - No noisy logs during normal distribution calculation.
    - If adding debug logs for skipped gifts, guard them behind existing debug-style control and avoid descriptions/private text.
  - Dependencies: Tasks 1 and 6.

### Phase 6: Tests And Verification

- [x] 10. Add focused backend tests for gift review, transactional creation, and matching behavior.
  - Files:
    - `backend/internal/application/command/add_gift_test.go`
    - `backend/internal/application/command/update_gift_test.go`
    - `backend/internal/application/query/get_prize_distribution_test.go`
    - `backend/internal/infrastructure/telegram/handler/gift_test.go`
    - optional `backend/internal/infrastructure/persistence/postgres/gift_repo_test.go`
  - Deliverable:
    - Test that new gifts are `pending_review`.
    - Test that gift + attachment creation is atomic, using a repository fake or SQL transaction test.
    - Test Telegram confirmation does not persist until `confirm_gift`.
    - Test `confirm_gift` persists using the Telegram user ID from session/update context and produces `pending_review`.
    - Test invalid review statuses and invalid filters are rejected.
    - Test explicit `place: null` clears place, while omitted `place` preserves it.
    - Test approving a gift and replacing criteria is atomic.
    - Test unapproved gifts are excluded from distribution.
    - Test criteria-priority matching over place-only/generic matching using the selected algorithm from Task 9.
  - Logging requirements:
    - Tests should not assert log text unless logging is the behavior under test.
    - Test fakes should capture operation order and errors without printing noisy logs.
  - Dependencies: Tasks 2, 3, 6, 7, and 9.

- [x] 11. Run full verification and cleanup.
  - Files: all changed backend/frontend/API docs files
  - Deliverable:
    - Run `gofmt` on changed Go files.
    - Run `cd backend && go test ./...`.
    - Run `cd frontend && npm run lint`.
    - Run `cd frontend && npm run build`.
    - Run `rg "go-telegram/ui|CreateGiftModal|giftsApi.create|POST /api/events/\\{eventId\\}/gifts" .` and verify removed/admin-create references are intentional only.
    - Review Swagger gift endpoint docs for consistency.
  - Logging requirements:
    - No new runtime logs in this task.
    - Verify added logs are actionable and do not expose tokens or private gift text.
  - Dependencies: Tasks 1-10.

## Commit Plan

- **Commit 1** (after tasks 1-3): `feat(backend): add gift review status and confirmation`
- **Commit 2** (after tasks 4-8): `feat(admin): require review for telegram gifts`
- **Commit 3** (after tasks 9-11): `test: verify reviewed gift distribution`

## Risks And Notes

- Existing gifts should be backfilled to `approved`; otherwise current events may suddenly lose all prize distribution.
- Removing the HTTP create route is a contract change. Swagger and frontend API wrappers must be updated in the same change.
- `place: null` currently cannot clear `gift.place`; this should be fixed while extending gift update, because place is now explicitly secondary and may be removed by admins.
- Current HTTP gift update is too fat for the architecture rules. Introducing an application command is part of this plan because review/criteria/place updates now carry business rules and atomicity requirements.
- `go-telegram/ui` is intentionally excluded. Current local keyboard builder is sufficient and avoids dependency/API instability.
- The review status should be the distribution gate. A broad `all/all` gift can still be approved deliberately by an admin, but it must never be auto-eligible just because a user submitted it.

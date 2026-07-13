# Implementation Plan: Random Assignment of Unassigned Prizes from the Admin List

Branch: main
Created: 2026-07-13

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Scope and Decision

Add a row-level action in the admin gifts list that assigns one still-unassigned, approved gift to a cryptographically random eligible participant who has no prize. This intentionally mirrors the action shown in the Mini App screenshot; it is not a bulk operation over the current page, filter, or whole event.

The action is available for both `automatic_unassigned` and `manual_unassigned` gifts. It must never be available for gifts pending review or already assigned automatically or manually.

## Tasks

### Phase 1: Application Logic

- [x] Task 1: Add a neutral application query that returns only eligible, unawarded participant IDs for an event, reusing the existing eligibility definition (finished/DNF only; exclude disqualified; exclude participants with automatic or manual prizes). Use that query and a testable `crypto/rand` selector in the new admin flow, then adapt `backend/internal/application/query/get_miniapp_participants.go` to decorate and sort those IDs for the Mini App UI. Retain Mini App owner checks in `backend/internal/application/command/assign_random_manual_gift_recipient.go`; the admin command must not depend on Mini App display DTOs or sorting. Files: new focused query and tests in `backend/internal/application/query/`, `backend/internal/application/command/assign_random_manual_gift_recipient.go`, and `backend/internal/application/query/get_miniapp_participants.go`. Logging: retain DEBUG at selection start, INFO with `gift_id`, `event_id`, candidate count and recipient ID on success, WARN for no candidates or rejected state, and ERROR with stage/context for dependency failures. Dependencies: none.

- [x] Task 2: Implement a dedicated administrator command that randomly assigns a single eligible recipient without inheriting the Telegram gift-owner constraint. The command must: load the gift; require `approved`; compute automatic assignment server-side from both `MatchedGiftAssignments` and legacy `MatchedGifts`; reject already distributed gifts; allow `manual_unassigned` gifts and convert `automatic_unassigned` gifts to manual distribution before assigning the selected recipient; preserve event and participant-eligibility validation; and return a domain-level conflict when no unawarded participant remains. Add a focused repository compare-and-set write so a second request cannot overwrite a recipient assigned after the command's initial read; map the stale state to conflict and let the UI reload. Reuse existing validation where possible, without coupling to Mini App DTOs. Files: new `backend/internal/application/command/assign_random_admin_gift_recipient.go` and tests, focused repository contract/implementation changes in `backend/internal/domain/repository/gift.go` and `backend/internal/infrastructure/persistence/postgres/gift_repo.go`, plus focused changes to `backend/internal/application/command/update_gift.go` only if shared validation is extracted. Logging: DEBUG for received state, INFO for success and mode conversion, WARN for each rejected state (including stale assignment), and ERROR for repository/distribution/random failures; never log personal data beyond participant ID. Dependencies: Task 1.

### Phase 2: Protected API and Runtime Wiring

- [x] Task 3: Expose `POST /api/gifts/{id}/random-recipient` inside the existing JWT-protected administrator route group. Add the handler method to `backend/internal/infrastructure/http/handler/gifts.go`, inject the new command through its constructor and wire it in `backend/internal/infrastructure/http/server.go`. Parse and validate the ID, return `204 No Content` on success, map not-found and business conflicts consistently, and invalidate the Mini App event gift-cache after a successful automatic-to-manual conversion. Files: `backend/internal/infrastructure/http/handler/gifts.go`, `backend/internal/infrastructure/http/server.go`, relevant cache abstraction/use site. Logging: INFO with request outcome and IDs on success, WARN for invalid ID and domain conflicts, ERROR for unexpected failures. Dependencies: Task 2.

- [x] Task 4: Add backend tests for the neutral candidate query and admin command (candidate filtering, secure-index error, no candidates, pending/unapproved gift, automatic/manual already-assigned gift, manual-unassigned gift, automatic-to-manual conversion, and compare-and-set conflict after a stale read). Add handler tests for HTTP codes, response body, unavailable dependency, and both cache paths: invalidate after automatic-to-manual conversion, but do not invalidate for assigning a recipient to an already manual gift. Preserve Mini App random-recipient tests to prove its owner-scoped behavior was not weakened. Files: `backend/internal/application/query/*_test.go`, `backend/internal/application/command/*_test.go`, `backend/internal/infrastructure/http/handler/gifts_test.go`, `backend/internal/infrastructure/http/handler/gifts_cache_test.go`, and `backend/internal/infrastructure/http/handler/miniapp_test.go` when a regression case is needed. Logging: tests must assert observable results rather than log text; production paths covered by the tests must retain the DEBUG/INFO/WARN/ERROR checkpoints defined above. Dependencies: Tasks 1-3.

### Phase 3: Admin UI, Contract, and Verification

- [x] Task 5: Add `giftsApi.assignRandomRecipient(giftId)` to `frontend/src/api/gifts.ts`, with a Vitest contract test in `frontend/src/api/gifts.test.ts`. Extract the availability predicate for the random action into `frontend/src/utils/manualGiftAssignment.ts` and cover it with its existing-style unit test: only approved `automatic_unassigned` and `manual_unassigned` gifts qualify. In `frontend/src/components/gifts/GiftsTable.tsx` use that predicate for a per-row action labelled for random distribution, keep independent pending state per gift, and pass the action to the page. In `frontend/src/app/(dashboard)/gifts/page.tsx`, call the API, reload gifts/manual assignments/distribution after success, and surface a concise inline error consistent with existing actions. Do not introduce a new React component-test stack solely for this feature. Logging: use `console.info` with operation, gift and event IDs after success, and `console.error` with the same context and the caught error on failure; do not log recipient display data. Dependencies: Task 3.

- [x] Task 6: Document the protected endpoint, its no-body request, `204` success and conflict cases in `backend/docs/swagger.yaml`; make wording clear that the recipient is randomly chosen server-side among eligible participants without prizes. Run focused and full validation: `cd backend && go test ./...`, `cd frontend && npm run test -- --run`, `cd frontend && npm run lint`, `cd frontend && npm run build`, and `git diff --check`. Logging: verify that error paths are diagnosable from the structured context described in Tasks 1-5 without exposing participant personal data. Dependencies: Tasks 4-5.

## Commit Plan

- **Commit 1** (after tasks 1-3): `feat(gifts): add admin random prize assignment endpoint`
- **Commit 2** (after tasks 4-6): `feat(admin): add random assignment action to gifts list`

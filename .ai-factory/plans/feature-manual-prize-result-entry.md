# Implementation Plan: Manual Gift And Result Entry Buttons

Branch: feature/manual-prize-result-entry
Created: 2026-06-15
Refined: 2026-06-15

## Settings
- Testing: yes
- Logging: minimal
- Docs: yes

## Scope And Product Contract

- Add admin dashboard buttons for manual entry:
  - manual gift creation from the admin gifts workflow;
  - manual result creation from the participant detail result block when the participant has no current result.
- Manual gifts and manual results must use the same persistence model as Telegram-created data:
  - manual gifts are normal `gifts` rows with `user_id` set to the entered Telegram user ID;
  - manual results are normal `results` rows attached to the participant profile currently opened in the admin dashboard.
- Do not add a migration for this feature. The existing `gifts` and `results` tables already hold the required data.
- Do not restore `prize_assignments`. That path is deprecated and is not part of manual gift/result entry.
- Manual result entry is opened from `frontend/src/app/(dashboard)/participants/[id]/page.tsx` in the existing `Result` card. The form does not ask for Telegram ID because the participant profile already identifies the Telegram user.
- Manual gift entry must require the admin to enter the Telegram user ID of the user on whose behalf the gift is created.
- Treat `required` as a hard cross-layer requirement:
  - HTML form controls use `required` where the field is mandatory.
  - TypeScript request types keep required fields non-optional.
  - Backend handlers reject missing or zero-value required fields before command execution.
  - OpenAPI request schemas declare `requestBody.required: true` and required property arrays.

## Commit Plan
- **Commit 1** (after tasks 1-3): `feat(admin): add manual gift and result api`
- **Commit 2** (after tasks 4-6): `feat(admin): add manual gift and result forms`
- **Commit 3** (after tasks 7-8): `test(admin): cover manual gift and result entry`

## Tasks

### Phase 1: Backend Contracts

- [x] Task 1: Add admin manual gift creation through the existing gift command.

  Deliverable: add a protected admin create route for gifts, preferably `POST /api/events/{eventId}/gifts`, and implement `GiftsHandler.Create` using the existing `command.AddGiftHandler`. Required backend payload: `user_id` and `description`; `eventId` remains the required path parameter. Optional payload: `gender_filter` and `bike_type_filter`, defaulting through `AddGiftHandler` the same way Telegram gift creation does. The created gift must be a normal pending-review gift with no separate admin-only table or assignment record.

  Files: `backend/internal/infrastructure/http/server.go`, `backend/internal/infrastructure/http/handler/gifts.go`, `backend/internal/application/command/add_gift.go`, `backend/docs/swagger.yaml`.

  Logging requirements: reuse the existing AddGift command logging where possible. In the HTTP handler, log `WARN` for rejected manual gift requests with `telegram_user_id`, `event_id`, and `error_class`; log `ERROR` for unexpected command or persistence failures. Do not log gift descriptions.

  Dependency notes: independent from result work. Requires wiring `AddGiftHandler` into `GiftsHandler` or another thin HTTP adapter. Do not add or edit migrations.

- [x] Task 2: Add admin manual result creation for the opened participant profile.

  Deliverable: make `POST /api/participants/{participantId}/results` support admin-created manual results without requiring a Strava link. Add a dedicated application command, for example `CreateManualResultHandler`, so Telegram submission continues to use `SubmitResultHandler` and keeps its Strava/event-start rules. Required backend payload: `elapsed_time_sec`. Optional payload: `moving_time_sec` and `result_link`; if `result_link` is present, validate it with the existing Strava result-link value object. Reject missing or non-positive `elapsed_time_sec`, negative time values, `moving_time_sec > elapsed_time_sec`, unknown participant, and creating a manual result when the participant already has a current result.

  Files: `backend/internal/application/command/create_manual_result.go`, `backend/internal/infrastructure/http/handler/results.go`, `backend/internal/infrastructure/http/server.go`, `backend/internal/domain/entity/result.go`, `backend/internal/infrastructure/persistence/postgres/result_repo.go`, `backend/docs/swagger.yaml`.

  Logging requirements: log `WARN` for rejected manual result input with `participant_id` and `error_class`; log `ERROR` for unexpected repository failures. Do not log full result URLs.

  Dependency notes: frontend result entry in task 5 depends on this backend contract. Preserve the existing Telegram/Strava submit path through `SubmitResultHandler`; do not weaken user-side result submission rules.

- [x] Task 3: Keep result creation semantics aligned with existing participant state.

  Deliverable: ensure successful manual result creation refreshes the same participant/result state that the admin page already reads: the result is current, the participant becomes finished through the existing result-derived DTO behavior, and result criteria can be added after creation. If existing repository behavior marks previous results as non-current, the manual command should still reject duplicate current-result creation before calling `ResultRepository.Create` because the UI flow is "add only when not added yet".

  Files: `backend/internal/application/dto/participant.go`, `backend/internal/application/dto/result.go`, `backend/internal/infrastructure/http/handler/participants.go`, `backend/internal/infrastructure/http/handler/results.go`, `backend/internal/infrastructure/persistence/postgres/result_repo.go`.

  Logging requirements: no extra success logs beyond the command/handler logs. Rejections should use the same `error_class` pattern as task 2.

  Dependency notes: depends on task 2. This task is a guard against silently replacing an existing current result.

### Phase 2: Frontend Forms And Required Controls

- [x] Task 4: Add reusable required support to form controls used by manual entry.

  Deliverable: extend `frontend/src/components/form/Select.tsx` with a `required?: boolean` prop and pass it to the native `<select>`. Extend `frontend/src/components/participants/TimeInput.tsx` with `required?: boolean`, controlled value handling if needed, and clear validation behavior so empty required time cannot submit.

  Files: `frontend/src/components/form/Select.tsx`, `frontend/src/components/participants/TimeInput.tsx`, `frontend/src/components/form/input/InputField.tsx` if a shared hint/required marker is needed.

  Logging requirements: no runtime logs for normal validation. If a component-level unexpected parse issue is surfaced, use `console.error` only with component name and field name, not user-entered values.

  Dependency notes: blocks tasks 5 and 6 because both manual entry forms need native required controls.

- [x] Task 5: Add manual result entry in the participant detail `Result` card.

  Deliverable: add a `Ввести результат` or `Добавить результат` button in `frontend/src/app/(dashboard)/participants/[id]/page.tsx` in the existing `Result` card when the participant has no current result. The form must require `elapsed_time_sec`, optionally accept `moving_time_sec` and optional Strava `result_link`, validate `moving_time_sec <= elapsed_time_sec`, call `resultsApi.create(participant.id, data)`, then refresh participant and result data. Keep the existing `Изменить время` path for participants that already have a current result. The UI must not silently no-op when no current result exists.

  Files: `frontend/src/app/(dashboard)/participants/[id]/page.tsx`, optional new `frontend/src/components/participants/ManualResultModal.tsx`, `frontend/src/api/results.ts`, `frontend/src/types/index.ts`.

  Logging requirements: use `console.error` for failed create/update operations with `operation`, `participant_id`, and backend `error_class` if exposed. Do not log full result URLs.

  Dependency notes: depends on tasks 2, 3, and 4.

- [x] Task 6: Add manual gift entry in the admin gifts workflow.

  Deliverable: add a visible `Добавить приз вручную` action to the gifts admin page. The form must require event selection or use the currently selected event, require `user_id` as the Telegram ID of the user on whose behalf the gift is created, and require `description`. Optional filters should match the Telegram gift filters (`gender_filter`, `bike_type_filter`) and default to `all` when not selected. Submit through `giftsApi.create(eventId, data)`, then refresh the gifts list. The created gift should appear in the normal gifts list and review flow, not in a separate manual-only view.

  Files: `frontend/src/app/(dashboard)/gifts/page.tsx`, optional new `frontend/src/components/gifts/ManualGiftModal.tsx`, `frontend/src/api/gifts.ts`, `frontend/src/types/index.ts`.

  Logging requirements: use `console.error` for failed load/create operations with `operation`, `event_id`, and `telegram_user_id` when available. Do not log gift descriptions.

  Dependency notes: depends on tasks 1 and 4.

### Phase 3: API Documentation And Required Schema Alignment

- [x] Task 7: Align OpenAPI and frontend types with required payloads.

  Deliverable: update `backend/docs/swagger.yaml` so manual gift and manual result schemas explicitly declare required properties. `POST /events/{eventId}/gifts` must have `requestBody.required: true` and required `user_id` and `description`. `POST /participants/{participantId}/results` must have `requestBody.required: true` and required `elapsed_time_sec`; `moving_time_sec` and `result_link` remain optional. Update frontend request interfaces so required fields are non-optional and optional fields are explicitly optional.

  Files: `backend/docs/swagger.yaml`, `frontend/src/types/index.ts`, `frontend/src/api/results.ts`, `frontend/src/api/gifts.ts`.

  Logging requirements: no runtime logs; this is a contract/documentation task.

  Dependency notes: depends on tasks 1 and 2 so the documented contract matches the implemented backend behavior.

### Phase 4: Tests And Verification

- [x] Task 8: Add focused tests and run verification gates.

  Deliverable: add backend tests for manual gift required fields, unknown Telegram user, blacklisted user, missing event, and successful pending-review gift creation through the HTTP handler/command path. Add backend tests for manual result required `elapsed_time_sec`, invalid time ordering, duplicate current-result rejection, optional Strava link validation, and successful creation without Strava link. Add focused frontend tests for extracted helpers if helper logic is introduced; otherwise rely on type/build validation for the changed UI surfaces.

  Files: `backend/internal/infrastructure/http/handler/gifts_test.go`, `backend/internal/application/command/add_gift_test.go` if needed, `backend/internal/application/command/create_manual_result_test.go`, `backend/internal/infrastructure/http/handler/results_test.go`, changed frontend files from tasks 4-7, and new frontend helper tests if helpers are extracted.

  Logging requirements: tests may assert rejection paths through status/error responses; no test should rely on full log text. If log capture is needed, assert only error class, not free-form values.

  Dependency notes: depends on tasks 1-7.

  Verification commands:
  - `cd backend && go test ./...`
  - `cd frontend && npm run test -- --run` if frontend tests are added/available
  - `cd frontend && npm run build`
  - `git diff --check`

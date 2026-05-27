# Implementation Plan: Strava-Only Result Submission After Event Start

Branch: main
Created: 2026-05-27

## Goal

Change result submission so new results are accepted only as Strava links and only after the participant's event start time has been reached. Event start time must be interpreted and displayed as Minsk time, fixed UTC+3.

## Settings

- Testing: yes
- Logging: standard
- Docs: yes, update Swagger/user-facing wording where result links are documented

## Requirements

- Accept new result submissions only for Strava links.
- Keep existing result persistence shape: no schema migration is needed for `results.result_link`.
- No backwards compatibility for old Komoot links is required because the project is not in production yet.
- Reject result submission before the event `start_date`.
- Use Minsk time, fixed UTC+3, for event start parsing, comparison, formatting, and user-facing messages.
- `active=true` alone is not enough to submit a result anymore.
- If an active event has no `start_date`, do not accept result submission; return a clear message that the event start time is not configured.
- Do not introduce end-date blocking unless explicitly requested; this change is only about "not before start".
- Keep handlers thin: validation belongs in domain/application helpers and commands, not duplicated in Telegram and HTTP handlers.
- Do not log raw result URLs, Telegram tokens, or private user message text.

## Current Findings

- Telegram result flow starts from callback data `submit_result` in `backend/internal/infrastructure/telegram/handlers.go`.
- `ResultHandler.StartSubmitResult` currently finds the active event, checks registration, checks `participant.IsFinished()`, stores `participant_id` in session, then asks for a "Strava или Komoot" link.
- `handleMessage` sends `msg.Text` to `ResultHandler.HandleResultLink` when the session state is `StateAwaitingResultLink`.
- `command.SubmitResultHandler` currently depends only on `ParticipantRepository` and `ResultRepository`; it does not load the event and does not check `start_date`.
- `valueobject.NewResultLink` currently accepts Strava activity links, Strava app links, and Komoot tour links.
- PostgreSQL read paths in `result_repo.go` and `participant_repo.go` parse stored result links through `valueobject.NewResultLink`, so making this value object Strava-only will affect both writes and reads. That is acceptable for this pre-production change.
- Protected HTTP `POST /api/participants/{participantId}/results` bypasses `SubmitResultHandler` through `dto.CreateResult`, so command-level rules must be wired into HTTP too.
- Events already have `start_date` and `end_date`; `EventForm` currently captures only date values, while HTTP `parseDate` can parse date/time strings.
- Frontend event form currently derives dates through `new Date(...).toISOString()`, which can shift values based on the browser timezone and must not be used for Minsk wall-time editing.
- `event_repo.FindActive` returns an error when there is no active event, so Telegram result flow must map that case to a user-facing "no active event" state instead of a generic retry error.
- `Event.IsOngoing()` exists but uses `time.Now()` directly and includes `end_date`, so it should not be reused blindly for a start-only submission gate.

## Commit Plan

- **Commit 1** (after tasks 1-3): `feat(results): enforce strava result submission rules`
- **Commit 2** (after tasks 4-6): `feat(events): use minsk start time for result gate`
- **Commit 3** (after tasks 7-8): `test(results): cover result submission gating`

## Tasks

### Phase 1: Strava-Only Result Validation

- [x] 1. Add Strava-only validation for new result submissions.
  - Files:
    - `backend/internal/domain/valueobject/result_link.go`
    - `backend/internal/domain/valueobject/result_link_test.go`
    - `backend/internal/application/command/submit_result.go`
    - `backend/internal/application/dto/result.go`, only if the DTO helper remains after HTTP rewiring
    - `backend/internal/infrastructure/persistence/postgres/result_repo.go`, only if read-side scan behavior needs an explicit update after removing Komoot support
    - `backend/internal/infrastructure/persistence/postgres/participant_repo.go`, only if participant result scans need an explicit update after removing Komoot support
  - Deliverable:
    - Make result-link validation Strava-only. Because there is no production data yet, `NewResultLink` may become Strava-only directly instead of preserving Komoot read compatibility.
    - Parse and validate links with `net/url` plus host/path checks rather than lowercasing the entire URL, so case-sensitive URL parts such as `strava.app.link` tokens are not corrupted.
    - Decide explicitly whether Strava activity links may contain query strings/fragments; if they are accepted, store the normalized URL without breaking the activity ID path.
    - Update error text to say that only Strava links are accepted.
    - Remove or deprecate `PlatformKomoot` and `IsKomoot` if they become unused.
    - Do not change the `results` table schema.
  - Logging requirements:
    - Do not log raw submitted links.
    - If invalid-link attempts are logged, log only participant/event IDs and a safe reason such as `unsupported_platform` or `invalid_strava_format`.
  - Dependencies: none.

- [x] 2. Update Telegram result prompt and result-link message handling for Strava-only submissions.
  - Files:
    - `backend/internal/infrastructure/telegram/handler/result.go`
    - `backend/internal/infrastructure/telegram/handlers.go`
    - `backend/internal/infrastructure/telegram/model_helpers.go`, only if a text/caption helper is reused for result links
  - Deliverable:
    - Replace "Strava или Komoot" wording and examples with Strava-only wording.
    - Map invalid-link errors to a concise Telegram message that tells the user to send a Strava activity link.
    - Keep session state as `StateAwaitingResultLink` after invalid input so the user can retry without starting over.
    - Use a safe text extraction path for result links; reject photos/documents/empty messages as invalid input without passing empty strings into the command.
  - Logging requirements:
    - Log invalid result submission attempts at INFO with user ID, participant ID if available, event ID if available, and safe reason.
    - Do not log raw message text or URLs.
  - Dependencies: Task 1.

- [x] 3. Route HTTP result creation through the same application command.
  - Files:
    - `backend/internal/infrastructure/http/handler/results.go`
    - `backend/internal/infrastructure/http/server.go`
    - `backend/internal/application/dto/result.go`, remove or stop using `CreateResult` if it becomes duplicate command logic
    - `backend/docs/swagger.yaml`
  - Deliverable:
    - Inject `SubmitResultHandler` into `ResultsHandler`.
    - Make protected `POST /api/participants/{participantId}/results` use `SubmitResultHandler` instead of `dto.CreateResult`.
    - Convert the returned participant into the created result response via `dto.FromResult(participant.Result)`.
    - Map command errors with `errors.Is`: invalid/non-Strava links should return 400; missing participant should return 404; submission not open, missing event start, or future start should return 409 or 400 consistently with the existing response style.
    - Update Swagger descriptions from "Strava/Komoot" to Strava-only and document that result submission is accepted only after the event start time in Minsk UTC+3.
  - Logging requirements:
    - Log command failures at WARN/ERROR with participant ID and safe error class.
    - Do not log submitted URLs.
  - Dependencies: Tasks 1 and 5.

### Phase 2: Event Start Gate With Minsk UTC+3

- [x] 4. Add a single Minsk UTC+3 time policy for event start checks.
  - Files:
    - `backend/internal/domain/valueobject/result_time.go` or another domain-level value object/helper
    - `backend/internal/domain/entity/event.go`
    - `backend/internal/domain/entity/event_test.go`
    - `backend/internal/infrastructure/http/handler/events.go`
    - `backend/internal/infrastructure/http/handler/events_test.go`
    - `frontend/src/utils/minskTime.ts`, or another frontend helper colocated with existing utilities
  - Deliverable:
    - Define a fixed Minsk location (`UTC+3`) in a domain-accessible place so application code can use it without importing infrastructure.
    - Add deterministic event helper methods such as `HasStartedAt(now time.Time)` and, if needed, `SubmissionStartTimeInMinsk()`.
    - Treat `start_date == nil` as "result submission is not open" for this result gate.
    - Update event date parsing so date-only and timezone-less datetime values are interpreted as Minsk wall time; RFC3339 inputs with an offset should be normalized consistently to Minsk.
    - Add a frontend Minsk-time helper so UI formatting and request construction do not depend on the browser timezone.
    - Add focused backend tests proving:
      - `YYYY-MM-DD` parses as `00:00:00` Minsk UTC+3.
      - `YYYY-MM-DDTHH:MM:SS` parses as Minsk UTC+3.
      - RFC3339 values with an explicit offset are preserved as the same instant and formatted/displayed consistently in Minsk time.
    - Keep `end_date` out of the result-submission gate unless the product rule changes.
  - Logging requirements:
    - No routine logging is required in pure domain helpers.
    - Parsing failures in HTTP handlers should continue to log only safe field names and not request bodies.
  - Dependencies: none.

- [x] 5. Enforce event start in `SubmitResultHandler`.
  - Files:
    - `backend/internal/application/command/submit_result.go`
    - `backend/internal/infrastructure/http/server.go`
    - `backend/internal/infrastructure/telegram/bot.go`
    - `backend/internal/infrastructure/telegram/handler/result.go`
  - Deliverable:
    - Add `EventRepository` to `SubmitResultHandler`.
    - After loading the participant, load `participant.EventID` and reject submission if the event is missing, inactive, missing `start_date`, or not started yet in Minsk UTC+3.
    - Add explicit errors such as `ErrResultSubmissionNotOpen`, `ErrEventStartNotConfigured`, and `ErrEventNotStarted`.
    - Make the clock injectable or otherwise testable so command tests can set "now" deterministically.
    - Keep existing duplicate-result behavior: if a participant already has a current result, Telegram should still report that result was already submitted before asking for a new link.
  - Logging requirements:
    - Log blocked submissions at INFO with participant ID, event ID, current Minsk time, configured start time if present, and reason.
    - Do not log raw result links.
  - Dependencies: Task 4.

- [x] 6. Block Telegram result flow before setting `StateAwaitingResultLink` when the active event has not started.
  - Files:
    - `backend/internal/infrastructure/telegram/handler/result.go`
    - `backend/internal/infrastructure/telegram/handler/result_test.go`
    - `backend/internal/infrastructure/telegram/handlers.go`, only if error mapping is centralized there
  - Deliverable:
    - In `StartSubmitResult`, after loading the active event, return a clear message if the start time is missing or still in the future.
    - Treat the repository "no active event found" error as "В данный момент нет активных событий", not as a generic technical failure.
    - Format the start time as Minsk UTC+3 in the Telegram message.
    - Do not write `participant_id` or switch the session to `StateAwaitingResultLink` until the gate passes.
    - Keep the application command gate from Task 5 as the authoritative guard for races and HTTP paths.
  - Logging requirements:
    - Log early Telegram flow blocks at INFO with user ID, event ID, and safe reason.
    - Do not log user message text.
  - Dependencies: Tasks 4-5.

### Phase 3: Admin UI And Verification

- [x] 7. Make event start time visible/editable as Minsk time in the admin dashboard.
  - Files:
    - `frontend/src/components/events/EventForm.tsx`
    - `frontend/src/components/events/EventsTable.tsx`
    - `frontend/src/types/index.ts`
    - `frontend/src/api/events.ts`, only if request formatting helpers are needed
    - `frontend/src/utils/minskTime.ts`, or the helper path chosen in Task 4
  - Deliverable:
    - Change event start/end controls from date-only behavior to date-time behavior, or add a time input next to the existing date picker.
    - Submit event start/end values with explicit `+03:00` offset or a backend-supported format documented as Minsk UTC+3.
    - Display event start/end with date and time, not only date.
    - Show start and end as separate table columns in the events table.
    - Label the timezone as Minsk UTC+3 in the admin UI where the time is edited or displayed.
    - Avoid `toISOString()` for user-entered Minsk wall time, because it converts through the browser timezone.
    - Keep all event-specific Telegram result texts editable through the existing event Telegram texts admin page.
  - Logging requirements:
    - Frontend should not add routine logs for normal form input.
    - Keep existing error handling concise and avoid logging full request payloads.
  - Dependencies: Task 4 and the final API date-time format chosen there.

- [x] 8. Add focused tests and run verification.
  - Files:
    - `backend/internal/domain/valueobject/result_link_test.go`
    - `backend/internal/domain/entity/event_test.go`
    - `backend/internal/application/command/submit_result_test.go`
    - `backend/internal/infrastructure/telegram/handler/result_test.go`
    - `backend/internal/infrastructure/http/handler/results_test.go`, if HTTP create error mapping can be tested without heavy setup
  - Deliverable:
    - Cover accepted Strava activity/app links.
    - Cover rejected Komoot and unsupported links.
    - Cover that Strava URLs with supported query strings/fragments, if allowed by Task 1, normalize without corrupting the URL.
    - Cover missing event start time, future start time, and already-started event using a deterministic Minsk UTC+3 clock.
    - Cover event date parsing for date-only, timezone-less datetime, and RFC3339 with offset.
    - Cover Telegram `StartSubmitResult` does not enter `StateAwaitingResultLink` before event start.
    - Cover Telegram maps no active event and not-yet-started event to distinct user-facing messages.
    - Cover HTTP create uses the same command-level rules as Telegram.
    - Run `cd backend && go test ./...`.
    - Run targeted frontend checks for touched files, then `cd frontend && npm run build` if frontend event form changes are included.
    - Run `git diff --check`.
  - Logging requirements:
    - Tests should assert behavior and error mapping; they should not assert raw URL logging.
    - Verification should confirm logs contain IDs/reasons only, not submitted URLs.
  - Dependencies: Tasks 1-7.

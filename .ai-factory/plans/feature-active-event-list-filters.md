# Implementation Plan: Active Event Admin List Filters

Branch: feature-active-event-list-filters
Created: 2026-06-15

## Settings
- Testing: yes
- Logging: minimal
- Docs: yes

## Commit Plan
- **Commit 1** (after tasks 1-3): "feat(admin): use active event for participants and gifts"
- **Commit 2** (after tasks 4-6): "test(admin): finish active event list cleanup"

## Tasks

### Phase 1: Active Event Source
- [x] Task 1: Centralize active event selection for dashboard list pages.
  - Deliverable: add `eventsApi.getActive()` in `frontend/src/api/events.ts` using the existing backend contract `GET /api/events?activeOnly=true`; add a small typed helper in `frontend/src/utils/events.ts` that extracts the single active event from the API response.
  - Expected behavior: participants, gifts, and prize distribution pages should call event-scoped APIs with an internal active `event_id` loaded through `eventsApi.getActive()`; users should no longer choose an event on these list pages; no page should silently fall back to the first inactive event when no active event exists.
  - Files: `frontend/src/api/events.ts`, `frontend/src/utils/events.ts`, `frontend/src/utils/events.test.ts`.
  - Logging requirements: do not log inside the pure helper or API wrapper; callers should log only load failures at `ERROR` level through existing `console.error` calls with `operation` and, when available, `event_id`.
  - Dependency notes: required before page-level rewrites so all three pages resolve the active event consistently.

### Phase 2: Page Behavior
- [x] Task 2: Update the participants page to use only the active event and add the "Добавил приз" filter.
  - Deliverable: remove the visible "Событие" select and event options from `frontend/src/app/(dashboard)/participants/page.tsx`; keep an internal active event id for `participantsApi.getByEvent`.
  - Expected behavior: the participants list always loads for the active event; existing gender, bike type, finish status, and search filters keep working; a new filter "Добавил приз" supports "Все", "Да", and "Нет" using the existing `Participant.has_gift` field; if no active event exists, show an empty state/error without offering manual event selection.
  - Files: `frontend/src/app/(dashboard)/participants/page.tsx`, `frontend/src/utils/participants.ts`, `frontend/src/utils/participants.test.ts`, `frontend/src/api/participants.ts` only if typing for the filter is intentionally shared.
  - Logging requirements: keep minimal error logs for `load_active_event`, `load_participants`, and `delete_participant`; include `operation`, `event_id` when present, and the thrown error; do not log ordinary filter changes.
  - Dependency notes: depends on Task 1; extract search plus `has_gift` filtering into a pure utility so Vitest can cover behavior without rendering the page; should not add a backend `has_gift` query parameter unless frontend-only filtering proves insufficient.

- [x] Task 3: Update the gifts page to remove event filtering while preserving review-status filtering.
  - Deliverable: remove `event_id` parsing, URL updates, event select UI, and event query preservation from `frontend/src/app/(dashboard)/gifts/page.tsx`; keep `review_status` in the URL and continue using the active event internally for list loading, manual gift creation, and assigned-gift detection.
  - Expected behavior: `/gifts` and `/gifts?review_status=...` both apply to the active event; manual gift creation no longer asks the user to choose an event; edit/detail links preserve only review status, not event id; if no active event exists, the create action stays disabled with a relevant error/empty state.
  - Files: `frontend/src/app/(dashboard)/gifts/page.tsx`, `frontend/src/components/gifts/GiftsTable.tsx` if link construction needs a narrower prop.
  - Logging requirements: keep minimal error logs for `load_active_event`, `load_gifts`, `load_prize_distribution`, `create_manual_gift`, `approve_gift`, and `delete_gift`; include `operation`, active `event_id` when present, gift/user ids where relevant, and the thrown error.
  - Dependency notes: depends on Task 1; preserve the current review-status deep-link behavior because it is independent from event filtering.

- [x] Task 4: Update the prize distribution page to remove event filtering.
  - Deliverable: remove events state used only for the visible event select and remove the "Событие" filter from `frontend/src/app/(dashboard)/prize-distribution/page.tsx`; keep internal active event loading before calling `prizeDistributionApi.getPrizeDistribution`.
  - Expected behavior: prize distribution always shows the active event; "Тип совпадения" remains the only visible filter; stats and unassigned slots clear when no active event can be resolved.
  - Files: `frontend/src/app/(dashboard)/prize-distribution/page.tsx`.
  - Logging requirements: keep minimal error logs for `load_active_event` and `load_prize_distribution`; include `operation`, active `event_id` when present, and the thrown error; do not log match-reason filter changes.
  - Dependency notes: depends on Task 1 and must preserve current count calculations from `frontend/src/utils/prizeDistribution.ts`.

- [x] Task 5: Clean up gift navigation query params and manual gift copy after removing event selection.
  - Deliverable: remove `event_id` preservation from gift detail/back navigation and redirect flows; update manual gift error copy that currently tells the user to choose an event again.
  - Expected behavior: `/gifts/[id]` back/cancel/save links preserve only supported list state such as `review_status`; `/gifts/[id]/edit` redirects without leaking stale `event_id`; manual gift errors refer to the active event state or page refresh, not to selecting an event manually.
  - Files: `frontend/src/app/(dashboard)/gifts/[id]/page.tsx`, `frontend/src/app/(dashboard)/gifts/[id]/edit/page.tsx`, `frontend/src/utils/manualGiftErrors.ts`, `frontend/src/utils/manualGiftErrors.test.ts`.
  - Logging requirements: no new runtime logs for URL cleanup or copy changes; keep existing gift edit load error logging with `operation` and `gift_id`.
  - Dependency notes: depends on Task 3 because list query shape is defined there; prevents stale URLs from reintroducing removed event filtering.

### Phase 3: Verification And Docs
- [x] Task 6: Add focused tests and run frontend validation.
  - Deliverable: cover `eventsApi.getActive()`/active-event extraction, participant search plus "Добавил приз" filtering, and updated manual gift error copy with Vitest; update existing docs only if they currently describe manual event filtering on participants, gifts, or prize distribution pages.
  - Expected behavior: tests prove active event loading does not fall back to inactive events, "Добавил приз" filtering distinguishes all/yes/no without changing backend contracts, review-status gift links do not need `event_id`, and manual gift errors match the new active-event UX.
  - Files: `frontend/src/utils/events.test.ts`, `frontend/src/utils/participants.test.ts`, `frontend/src/utils/manualGiftErrors.test.ts`, existing docs such as `README.md` only if stale wording is found.
  - Logging requirements: no new runtime logs for tests; if docs mention troubleshooting, keep it at user-visible behavior level and do not document debug logging.
  - Dependency notes: depends on Tasks 1-5; validation commands are `cd frontend && npm run test -- --run`, `cd frontend && npm run lint`, and `cd frontend && npm run build`.

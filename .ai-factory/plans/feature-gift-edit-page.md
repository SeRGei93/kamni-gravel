# Implementation Plan: Gift Edit Page And Image Preview

Branch: feature/gift-edit-page
Created: 2026-05-26

## Settings
- Testing: yes
- Logging: standard
- Docs: no

## Requirements

- Replace admin gift editing modal with a standalone dashboard page.
- Opening a gift for editing must change the browser URL so the page is linkable and refresh-safe.
- Use `/gifts/{id}` as the canonical admin edit URL, matching the existing `/events/{id}` edit-page pattern.
- Preserve the selected gifts list context when navigating into and out of edit: event filter and review-status filter must survive save/cancel/back flows.
- Preserve Telegram-only gift creation: admins can edit/review/delete existing gifts, but cannot create gifts from the admin UI.
- Show previews for all gift photo attachments in the edit screen.
- Clicking a preview must open an enlarged image view.
- Keep existing backend gift API contracts unless an implementation blocker is found; `GET /api/gifts/{id}` and `PUT /api/gifts/{id}` already cover the edit page data flow.

## Current Findings

- The gifts list page is `frontend/src/app/(dashboard)/gifts/page.tsx`.
- The table component is `frontend/src/components/gifts/GiftsTable.tsx`.
- Current editing UI is modal-based in `frontend/src/components/gifts/EditGiftModal.tsx`.
- Existing event editing uses `/events/{id}` as the canonical page, and `/events/{id}/edit` redirects to it through `frontend/src/app/(dashboard)/events/[id]/edit/page.tsx`; gifts should follow the same route style.
- The frontend already has `giftsApi.getById`, `giftsApi.update`, `criteriaApi.getAll`, and `telegramApi.getFileURL`.
- Backend `GET /api/gifts/{id}` loads attachments and criteria through `GetGiftByIDHandler`.
- `UpdateGiftHandler` rejects empty descriptions, invalid filters, non-positive `place`, invalid review status, and approval payloads without explicit `criteria_ids`; the page form should validate `place` before submit and always send `criteria_ids`.
- The table currently loads only the first photo URL, shows `+N`, and logs raw Telegram file IDs on preview failures. New admin photo loading should centralize safe logging and avoid raw Telegram file IDs/URLs in console output.
- `telegramApi.getFileURL` currently interpolates `fileId` directly into the URL path; the frontend helper should encode file IDs before calling `/api/telegram/files/{fileId}`.
- `frontend/next.config.ts` already allows `https://api.telegram.org/file/**` for `next/image`.
- Miniapp components already demonstrate a multi-photo layout, but they use a protected miniapp file proxy and should not be copied directly into admin auth flow without adapting the file loading path.
- There is no frontend unit-test runner configured; verification should use TypeScript, ESLint, build, and browser smoke checks.

## Commit Plan

- **Commit 1** (after tasks 1-3): `feat: add gift edit page routing`
- **Commit 2** (after tasks 4-5): `feat: move gift editing to page form`
- **Commit 3** (after tasks 6-8): `feat: add gift photo previews`
- **Commit 4** (after tasks 9-10): `test: verify gift edit page`

## Tasks

### Phase 1: URL State And Routing

- [x] 1. Make the gifts list filters URL-backed before adding edit navigation.
  - Deliverable: update `frontend/src/app/(dashboard)/gifts/page.tsx` so `selectedEventId` and `reviewStatusFilter` can be initialized from query params such as `event_id` and `review_status`.
  - Expected behavior: opening `/gifts?event_id=123&review_status=pending_review` selects that event and filter; changing either filter updates the browser URL without a full reload; missing params still fall back to the active event and `all` review status.
  - Files: `frontend/src/app/(dashboard)/gifts/page.tsx`.
  - Logging requirements: preserve existing list-load error logging; include event/filter context where practical, but do not log gift descriptions or Telegram file data.

- [x] 2. Add the standalone canonical dashboard route for gift editing.
  - Deliverable: create `frontend/src/app/(dashboard)/gifts/[id]/page.tsx` as a client page that reads `params.id`, loads the gift via `giftsApi.getById`, loads criteria via `criteriaApi.getAll`, and renders loading, not-found/error, and edit states.
  - Expected behavior: direct navigation to `/gifts/{id}` opens the edit screen and survives browser refresh. Invalid IDs show a local error state with a link back to `/gifts`.
  - Files: `frontend/src/app/(dashboard)/gifts/[id]/page.tsx`, optional new reusable component files under `frontend/src/components/gifts/`.
  - Logging requirements: log failed gift/criteria loading with `console.error`, including `gift_id` and operation name, but do not log full gift descriptions, raw Telegram file IDs, or Telegram file URLs.

- [x] 3. Add `/gifts/{id}/edit` compatibility redirect and replace table edit actions with links.
  - Deliverable: create `frontend/src/app/(dashboard)/gifts/[id]/edit/page.tsx` that redirects to `/gifts/{id}`, mirroring the events route; remove `editingGift` state and `EditGiftModal` usage from `frontend/src/app/(dashboard)/gifts/page.tsx`; make edit actions in `GiftsTable` use a `Link` styled like the existing events table edit link.
  - Expected behavior: clicking "Редактировать" changes the URL and opens the edit page instead of an overlay. The edit link carries the current list query params so cancel/save can return to the same event/filter context.
  - Dependency: tasks 1 and 2.
  - Files: `frontend/src/app/(dashboard)/gifts/[id]/edit/page.tsx`, `frontend/src/app/(dashboard)/gifts/page.tsx`, `frontend/src/components/gifts/GiftsTable.tsx`.
  - Logging requirements: no extra success logging; preserve existing error logging for list load/update/delete failures.

### Phase 2: Page Form And Return Flow

- [x] 4. Extract modal form logic into a page-friendly gift edit form.
  - Deliverable: move form state, validation, criteria toggles, and submit behavior out of `EditGiftModal` into `frontend/src/components/gifts/GiftEditForm.tsx`.
  - Expected behavior: the form keeps the same editable fields: description, gender filter, bike type filter, place, review status, and criteria. It trims description, rejects empty description locally, validates `place` as empty or a positive integer before submit, and always sends `criteria_ids` so approving a gift cannot hit `ErrGiftCriteriaPayloadRequired` because the field was omitted.
  - Dependency: task 2 defines the page data flow.
  - Files: `frontend/src/components/gifts/EditGiftModal.tsx`, `frontend/src/components/gifts/GiftEditForm.tsx`, `frontend/src/app/(dashboard)/gifts/[id]/page.tsx`.
  - Logging requirements: log save failures with `console.error` and `gift_id`; do not log description text, selected Telegram file IDs, or Telegram file URLs.

- [x] 5. Add the edit page header, donor summary, and deterministic return behavior.
  - Deliverable: edit page shows "Назад к подаркам", donor identity, current review status, and a stable "Сохранить" / "Отмена" action row outside any modal container.
  - Expected behavior: after successful save, return to `/gifts` with the same `event_id` and `review_status` query params that opened the edit page; cancel returns to the same list URL without saving.
  - Dependency: tasks 1-4.
  - Files: `frontend/src/app/(dashboard)/gifts/[id]/page.tsx`, `frontend/src/components/gifts/GiftEditForm.tsx`.
  - Logging requirements: log save failure and unexpected navigation-relevant errors only; no logging for normal cancel/navigation.

### Phase 3: Image Previews And Enlargement

- [x] 6. Harden admin Telegram photo URL loading and remove unsafe preview logs.
  - Deliverable: update `frontend/src/api/telegram.ts` to encode `fileId` with `encodeURIComponent` in both file URL helpers; add a small admin-side photo URL loading helper/component path so `GiftsTable` and the edit page do not duplicate unsafe logging.
  - Expected behavior: preview failures log only safe context such as `gift_id`, `attachment_id`, and operation name; raw Telegram file IDs and generated file URLs are not printed to the browser console.
  - Dependency: tasks 2 and 3 identify the table and edit-page consumers.
  - Files: `frontend/src/api/telegram.ts`, `frontend/src/components/gifts/GiftsTable.tsx`, optional `frontend/src/components/gifts/useGiftPhotoUrls.ts`.
  - Logging requirements: use `console.warn` for photo URL load failures with safe IDs only; use `console.error` only for page-level data failures.

- [x] 7. Add an admin photo preview grid for gift attachments.
  - Deliverable: create `frontend/src/components/gifts/GiftPhotoPreviewGrid.tsx` that filters `file_type === "photo"`, loads URLs through the hardened admin Telegram helper, and renders a responsive thumbnail grid in the edit page.
  - Expected behavior: all photo attachments are visible as previews. If there are no photos, show a compact empty state. If an individual photo fails to load, keep the grid stable and show a small failed-photo placeholder for that attachment.
  - Dependency: tasks 2 and 6.
  - Files: `frontend/src/components/gifts/GiftPhotoPreviewGrid.tsx`, `frontend/src/app/(dashboard)/gifts/[id]/page.tsx`.
  - Logging requirements: log failed preview URL loads with `console.warn` including `gift_id` and `attachment_id`; do not log raw Telegram file IDs or Telegram URLs.

- [x] 8. Add click-to-enlarge image viewing.
  - Deliverable: implement a lightbox-style enlarged view for previews, either inside `GiftPhotoPreviewGrid` or as `frontend/src/components/gifts/GiftPhotoLightbox.tsx`.
  - Expected behavior: clicking a thumbnail opens the image larger, supports close by button and Escape, avoids nested edit modals, and keeps keyboard focus/accessibility sane enough for admin use.
  - Dependency: task 7.
  - Files: `frontend/src/components/gifts/GiftPhotoPreviewGrid.tsx`, optional `frontend/src/components/gifts/GiftPhotoLightbox.tsx`.
  - Logging requirements: no normal click logging; log image load failures only with safe IDs.

### Phase 4: Cleanup And Verification

- [x] 9. Remove obsolete modal-only edit code and preserve list actions.
  - Deliverable: delete `frontend/src/components/gifts/EditGiftModal.tsx` if fully replaced, or leave no imports of it; keep event filter, review-status filter, approve button, delete button, and pending counters working on `/gifts`.
  - Expected behavior: gifts editing has one canonical implementation. Approving a gift from the table still calls `giftsApi.update`, refreshes the list, and does not require opening the edit page.
  - Dependency: tasks 3-8.
  - Files: `frontend/src/components/gifts/EditGiftModal.tsx`, `frontend/src/app/(dashboard)/gifts/page.tsx`, `frontend/src/components/gifts/GiftsTable.tsx`.
  - Logging requirements: keep current approve/delete/list error logging with gift/event context where available; avoid logging descriptions and Telegram file data.

- [x] 10. Verify route, build, and interaction behavior.
  - Deliverable: run focused frontend checks and a browser smoke test.
  - Expected behavior:
    - `/gifts` renders and filter changes update query params;
    - `/gifts?event_id=...&review_status=...` restores the selected list state;
    - edit actions navigate to `/gifts/{id}` and carry list query params;
    - direct `/gifts/{id}` URL loads the existing gift;
    - `/gifts/{id}/edit` redirects to `/gifts/{id}`;
    - save/cancel return to the original list URL;
    - image previews render when attachments exist;
    - clicking an image enlarges it and close restores the edit page.
  - Commands:
    - `cd frontend && npx tsc --noEmit`
    - `cd frontend && npm run lint`
    - `cd frontend && npm run build`
    - Browser smoke with local runtime if available.
  - Logging requirements: during manual smoke, check browser console for unexpected errors; expected preview failures must include only safe IDs, not raw Telegram file IDs or URLs.

## Risks And Notes

- `telegramApi.getFileURL` returns Telegram file URLs from the admin API. If those URLs expire quickly, previews should reload on page mount and not be stored long-term.
- The edit page should not introduce admin gift creation or any new backend write contract.
- The canonical route should stay `/gifts/{id}` because the existing dashboard already treats edit pages like detail/edit pages (`/events/{id}`). Keep `/gifts/{id}/edit` only as a redirect convenience.
- Query-param preservation matters because `/gifts` currently stores selected event and review status in local state; without this, save/cancel would often return admins to a different list than the one they came from.

# Implementation Plan: Participants List Sorting

Branch: feature/participants-sort
Created: 2026-07-09

## Settings
- Testing: yes (backend sort unit/handler tests)
- Logging: standard (WARN on unknown sort key; existing DEBUG list log gains sort fields)
- Docs: yes (Swagger `sort`/`order` query params)

## Scope

Add sorting to the admin **participants list** (`/participants`) by numeric and
time columns. The sort control is an icon in the table header next to the column
name; clicking it cycles the sort state.

- Sort is **server-side**: the list uses server-side pagination
  (`page`/`page_size`, page size 50–100), so client-side sorting would only order
  the visible page. The backend sorts the full filtered set **before** slicing
  the page.
- Sortable columns = all numeric and time columns; text/badge columns (username,
  name, gender, bike type, status, Strava, "added a gift") get no sort control.
- Header icon is **tri-state** per column: click cycles ascending → descending →
  off (back to the default order: ranked finishers by place, then the rest).
  Only one column is sorted at a time.
- Missing/empty values always sort last, regardless of direction. `place == 0`
  ("no place") is treated as missing.
- When no sort is active, the current default ordering is preserved.

## Key Decisions

- **Contract by column key.** The frontend sends `sort=<column key>` +
  `order=asc|desc`; the backend maps the column key to the `ParticipantDTO`
  field/comparator. One identifier across the stack (e.g. `elapsed_time` →
  `elapsed_time_sec`, `distance_km` → `distance_meters`).
- **Sort where pagination happens.** In `ParticipantsHandler.GetAll`, sort the
  `filtered []*dto.ParticipantDTO` slice (after gender/bike/finished/has_gift/
  search filtering) right before the pagination slice. `sort.SliceStable` keeps
  the existing sub-order among equal values.
- **Nulls-last comparator.** A generic comparator puts `nil`/missing last in both
  directions; only the non-nil comparison flips with `order`.
- **Unknown sort key is ignored** (default order kept), logged at WARN — never a
  400, so an old client / stray param can't break the list.
- **No URL persistence** for sort state initially — component state on the page,
  reset to page 1 on sort change (mirrors filter behavior). Can be lifted to the
  URL later if needed.

## Current State

- `frontend/src/app/(dashboard)/participants/page.tsx` loads a page via
  `participantsApi.listByEvent(eventId, { …filters, page, page_size })` and
  renders `<ParticipantsTable columns={visibleColumns} …>`. Filters reset to page
  1 via a mount-guarded effect.
- `frontend/src/api/participants.ts` `listByEvent` builds the query string
  (bike_type/gender/is_finished/has_gift/q/page/page_size).
- `frontend/src/components/participants/participantColumns.tsx` is the column
  registry (`ParticipantColumn { key, label, render, align, … }`); columns are
  user-toggleable via `useColumnPreferences`.
- `frontend/src/components/participants/ParticipantsTable.tsx` renders the header
  from `column.label` and body from `column.render`.
- Backend `ParticipantsHandler.GetAll`
  (`backend/internal/infrastructure/http/handler/participants.go`) builds
  `allDTOs`, applies has_gift + search filters into `filtered`, then paginates
  `filtered[start:end]`. `dto.ParticipantDTO` already carries every sortable
  field (`elapsed_time_sec`, `distance_meters`, `peak_speed_kmh`, `started_at`,
  `ride_date`, place fields, etc.).

## Sortable Columns → Backend Sort Key → Field

- Integers: `place`* , `place_absolute`, `place_by_gender`, `place_by_gender_bike`,
  `prizes_count`, `calories`, `avg_heart_rate`, `max_heart_rate`, `avg_cadence`,
  `user_id`; `distance_km` → `distance_meters`.
- Time (seconds): `elapsed_time` → `elapsed_time_sec`, `moving_time` →
  `moving_time_sec`, `prev_elapsed_time` → `prev_elapsed_time_sec`, `idle_time` →
  `idle_time_sec`.
- Floats: `peak_speed_kmh`, `avg_speed_kmh`, `avg_moving_speed_kmh`.
- Timestamps: `started_at`, `ride_finished_at`.
- Date string (lexicographic == chronological): `ride_date`.

\* `place == 0` is treated as missing (sorts last).

## Commit Plan

- **Commit 1** (tasks 1-2): `feat(participants): add server-side sort to participants list API`
- **Commit 2** (tasks 3-5): `feat(participants): add sortable header controls to participants list`
- **Commit 3** (tasks 6-7): `test(participants): cover sorting + document sort params`

## Tasks

### Phase 1: Backend Sorting

- [ ] 1. Add sort parsing and a nulls-last comparator to the participants list handler.

  Files:
  - `backend/internal/infrastructure/http/handler/participants.go`

  Deliverable:
  - Parse `sort` (column key) and `order` (`asc`/`desc`, default `asc`) from the
    query in `GetAll`.
  - Add a `sortParticipantDTOs(items []*dto.ParticipantDTO, sortKey, order string)`
    helper: if `sortKey` maps to a known extractor, `sort.SliceStable` the slice
    with a nulls-last comparator (missing values always last; `place == 0` treated
    as missing); otherwise return the slice unchanged.
  - Map each sortable column key to a typed extractor over `*dto.ParticipantDTO`
    (int / *int / *float64 / *time.Time / *string per the column table in this
    plan). Keep the mapping in one place (a `map[string]` of comparators or a
    switch).
  - Call the sort on `filtered` **before** computing `total` and slicing the page,
    so ordering spans all pages of the filtered set.
  - Preserve the existing default order when `sort` is empty/unknown.
  - The set of comparator keys MUST exactly match the frontend `sortable` columns
    (Task #3) — a column marked sortable on the frontend with no backend mapping
    would silently do nothing (only a WARN). Shared key list: `place`,
    `place_absolute`, `place_by_gender`, `place_by_gender_bike`, `prizes_count`,
    `distance_km`, `peak_speed_kmh`, `avg_speed_kmh`, `avg_moving_speed_kmh`,
    `calories`, `avg_heart_rate`, `max_heart_rate`, `avg_cadence`, `user_id`,
    `elapsed_time`, `moving_time`, `prev_elapsed_time`, `idle_time`, `started_at`,
    `ride_finished_at`, `ride_date`.

  Logging requirements:
  - WARN once when `sort` is non-empty but unknown (include the raw value); do not
    fail the request.
  - Extend the existing `DEBUG Participants list served` log with `sort`/`order`.
  - Never log participant PII beyond what is already logged.

- [ ] 2. Backend tests for participant sorting.

  Files:
  - `backend/internal/infrastructure/http/handler/participants_test.go`

  Deliverable:
  - Table-driven tests over `sortParticipantDTOs` (or via the handler): numeric
    asc/desc, time-seconds asc/desc, timestamp and `ride_date` ordering.
  - Assert missing/`nil` values (and `place == 0`) always land last in both
    directions.
  - Assert sorting happens before pagination (sorted order is stable across a
    page boundary) and that an unknown `sort` key leaves the default order intact.

  Logging requirements:
  - Tests assert behavior, not logs.

### Phase 2: Frontend Sort Controls

- [ ] 3. Mark sortable columns and thread sort params through the API client.

  Files:
  - `frontend/src/components/participants/participantColumns.tsx`
  - `frontend/src/api/participants.ts`

  Deliverable:
  - Add an optional `sortable?: boolean` to `ParticipantColumn`; set it on every
    numeric and time column listed in this plan (leave text/badge columns off).
  - Extend `listByEvent` params with `sort?: string` and `order?: 'asc' | 'desc'`
    and append them to the query string when present.
  - Export a small `SortOrder = 'asc' | 'desc'` type for reuse.

  Logging requirements:
  - No new runtime logs.

- [ ] 4. Add sortable header controls to the participants table.

  Files:
  - `frontend/src/components/participants/ParticipantsTable.tsx`

  Deliverable:
  - Add props `sortKey: string | null`, `sortOrder: 'asc' | 'desc'`,
    `onSortChange: (key: string) => void`.
  - For sortable columns, render the label as a `<button>` with a sort icon next
    to the name. Reuse `ArrowUpIcon`/`ArrowDownIcon` from `@/icons` (already used
    in `components/nominations/NominationsTable.tsx`): active-asc → `ArrowUpIcon`,
    active-desc → `ArrowDownIcon`, inactive sortable → a muted (low-opacity)
    `ArrowDownIcon`. Do not hand-roll an inline SVG.
    Non-sortable columns render the plain label (unchanged).
  - Clicking a sortable header calls `onSortChange(column.key)`; the page owns the
    tri-state transition. Set `aria-sort` on the header cell for accessibility.
  - Keep the existing alignment, sticky/scroll, and row styling intact; the icon
    sits inline with the label where the column name is.

  Logging requirements:
  - No runtime logs.

- [ ] 5. Wire sort state into the participants page.

  Files:
  - `frontend/src/app/(dashboard)/participants/page.tsx`

  Deliverable:
  - Add `sortKey: string | null` and `sortOrder: 'asc' | 'desc'` state.
  - `handleSortChange(key)` implements the tri-state cycle: new key → `asc`;
    same key + `asc` → `desc`; same key + `desc` → cleared (key `null`, default
    order). Only one active column.
  - Pass `sort`/`order` to `listByEvent`; add them to `loadParticipants`
    dependencies; reset to page 1 when sort changes (extend the existing
    filter-reset effect).
  - Pass `sortKey`/`sortOrder`/`onSortChange` to `ParticipantsTable`.
  - Reset the sort when its column is hidden: `visibleColumns` drops columns
    toggled off in `ColumnSettings`, so a sort on a now-hidden column would stay
    active on the backend with no header to show or clear it. Add an effect that
    clears `sortKey` (→ default order) when the active `sortKey` is no longer in
    `visibleColumns`.

  Logging requirements:
  - Keep the existing `console.debug('[participants] loaded', …)`; include
    `sort`/`order` in it.

### Phase 3: Docs

- [ ] 6. Document the sort query parameters.

  Files:
  - `backend/docs/swagger.yaml`

  Deliverable:
  - Add `sort` (enum of sortable column keys) and `order` (`asc`/`desc`) query
    parameters to `GET /events/{eventId}/participants`, noting nulls-last and
    that sorting precedes pagination.

  Logging requirements:
  - Documentation only.

- [ ] 7. Verify and fix regressions.

  Files:
  - Only files touched above, unless verification exposes a direct regression.

  Deliverable:
  - `cd backend && go test ./...`.
  - `cd frontend && npm run lint` (targeted participants files clean) and
    `npm run build`.
  - Manual/browser smoke of `/participants`: clicking a numeric/time header sorts
    the whole filtered set (verify across a page boundary), tri-state cycle works,
    missing values sort last, non-sortable headers have no control, page resets to
    1 on sort change.

  Logging requirements:
  - Remove any temporary debug logs before completion.

## Sources Checked

- `frontend/src/app/(dashboard)/participants/page.tsx`, `api/participants.ts`,
  `components/participants/{ParticipantsTable,participantColumns}.tsx`,
  `hooks/usePaginationParams.ts`
- `backend/internal/infrastructure/http/handler/participants.go`,
  `application/dto/participant.go`

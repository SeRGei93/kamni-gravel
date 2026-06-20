# Implementation Plan: Configurable Participant List Columns

Branch: feature/participants-configurable-columns
Base branch: feature/server-side-pagination (this feature depends on the
participants list/table changes that are not yet on `main`)
Created: 2026-06-21

## Settings
- Testing: yes
- Logging: verbose
- Docs: warn-only (no mandatory docs checkpoint)

## Roadmap Linkage
- Milestone: none
- Rationale: no roadmap artifact present

## Scope And Product Contract

Make the columns of the participants list (`/participants`) **user-configurable**
and surface the Strava ride metrics as additional optional columns.

- The user can choose which columns are shown via a "Columns" picker
  (checkbox dropdown). The chosen set is **remembered and survives a page
  reload** (persisted in `localStorage`, per the user's "настройки запоминаются и
  остаются после перезагрузки").
- Add the ride-result metrics from the result detail form as available columns:
  start time, finish time, distance (km), peak speed (km/h), calories, average
  heart rate, max heart rate, average cadence, and the auto-calculated values
  (idle time, average speed, average moving speed, ride date).
- **Cell values are NOT edited inline.** Editing stays in the existing result
  detail form (`/participants/[id]`). "Editable columns" here means the column
  *set* is editable/customizable.
- The metric data already exists on the `Result` entity / `ResultDTO` and the
  list query already `LEFT JOIN`s the current result. The work is to (a) carry
  those metrics through `ParticipantDTO` and the frontend `Participant` type,
  and (b) build the configurable-column UI with persistence.
- No DB migration needed — migration `00022_add_result_metrics.sql` already adds
  the metric columns.

### Available columns (registry)

Always-visible (cannot be hidden): `username` (links to detail), `name`.

Default-visible (toggleable): `place`, `gender`, `bike_type`, `elapsed_time`,
`moving_time`, `has_gift`, `prizes_count` (the current columns).

Off-by-default (toggleable, new): `place_absolute`, `place_by_gender`,
`place_by_gender_bike`, `started_at`, `finished_at`, `distance_km`,
`peak_speed_kmh`, `calories`, `avg_heart_rate`, `max_heart_rate`, `avg_cadence`,
`idle_time`, `avg_speed_kmh`, `avg_moving_speed_kmh`, `ride_date`.

Participants without a result render "-" in metric cells.

## Affected Files

Backend:
- `backend/internal/application/dto/participant.go` — add metric + computed
  fields to `ParticipantDTO`; in `FromParticipant`, reuse `dto.FromResult(p.Result)`
  to populate them (do not re-derive — keep the two DTOs in sync).
- `backend/internal/infrastructure/persistence/postgres/participant_repo.go` —
  the list loads via `participantRepo.FindByEvent` → `scanParticipantFromRows`,
  a helper **shared with `GetFinishedByEvent`** (both SELECTs are identical).
  Add the metric columns to **both** SELECTs feeding `scanParticipantFromRows`
  (`FindByEvent` ~line 92 and `GetFinishedByEvent` ~line 122) and extend
  `scanParticipantFromRows` once. (There is no paged participant repo method —
  pagination is in-handler slicing; `FindByEventWithPlaces` belongs to the
  *result* repo, not here.)
- `backend/internal/infrastructure/http/handler/participants_test.go` — assert
  the list response carries metrics.

Frontend:
- `frontend/src/types/index.ts` — add metric + computed fields to `Participant`.
- `frontend/src/components/participants/participantColumns.tsx` (NEW) — column
  registry: `ParticipantColumn { key, label, alwaysVisible?, defaultVisible,
  align?, render(participant) }` plus the ordered list and the default key set.
- `frontend/src/hooks/useColumnPreferences.ts` (NEW) — generic localStorage-backed
  visible-keys hook (SSR-safe; merges stored keys with the registry so
  added/removed columns degrade gracefully).
- `frontend/src/components/participants/ColumnSettings.tsx` (NEW) — "Columns"
  dropdown (reuse `components/ui/dropdown/Dropdown`) with a checkbox per
  toggleable column and a "Reset to defaults" action.
- `frontend/src/components/participants/ParticipantsTable.tsx` — render headers
  and cells dynamically from the visible columns; keep the finished-row
  highlight and the username detail link; replace the fixed `min-w-[1320px]`
  with natural width + `overflow-x-auto` so a variable column count scrolls.
- `frontend/src/app/(dashboard)/participants/page.tsx` — wire
  `useColumnPreferences`, render `ColumnSettings` near the filters, pass the
  visible columns to `ParticipantsTable`.

## Tasks

### Phase 1 — Backend: expose ride metrics on the list

- [x] **T1. Extend `ParticipantDTO` with ride metrics + computed fields.**
  In `dto/participant.go` add nullable fields mirroring `ResultDTO`:
  `started_at`, `finished_at`, `distance_meters`, `avg_heart_rate`,
  `max_heart_rate`, `peak_speed_kmh`, `avg_cadence`, `calories`, and computed
  `idle_time_sec`, `idle_time`, `avg_speed_kmh`, `avg_moving_speed_kmh`,
  `ride_date`. In `FromParticipant`, when `p.Result != nil`, **reuse
  `dto.FromResult(p.Result)`** and copy these metric/computed fields from it —
  do NOT re-derive (so `ParticipantDTO` and `ResultDTO` can't diverge). The
  computed values come from `Result` methods `RideDate`, `IdleTimeSec`/
  `IdleTimeFormatted`, `AvgSpeedKmh`, `AvgMovingSpeedKmh` (already invoked by
  `FromResult`). Nil-safe when `p.Result` is nil.
  Logging: `DEBUG` when a participant has a result with metrics mapped.

- [x] **T2. Load metric columns into the shared participant scan path.**
  The list loads via `participantRepo.FindByEvent` → `scanParticipantFromRows`,
  and that helper is **shared with `GetFinishedByEvent`** (both SELECTs are
  identical). Add the metric columns
  (`started_at, finished_at, distance_meters, avg_heart_rate, max_heart_rate,
  peak_speed_kmh, avg_cadence, calories`) to **both** SELECTs feeding
  `scanParticipantFromRows` (`FindByEvent` ~line 92 and `GetFinishedByEvent`
  ~line 122) and extend `scanParticipantFromRows` once to scan them into
  `p.Result` with `sql.Null*` scanners (mirror the elapsed/moving handling;
  reuse the column list from `result_repo.go`). Adding columns to only one
  SELECT would break the other query at scan time (column-count mismatch). Do
  NOT touch `scanParticipant`/`FindByID`/`FindByUserAndEvent` — the list does not
  use them. Logging: `DEBUG` row count + whether metrics were present.
  Depends on T1.

### Phase 2 — Frontend: types, registry, persistence

- [x] **T3. Extend the frontend `Participant` type.**
  In `types/index.ts` add the same optional fields returned by T1
  (`started_at?`, `finished_at?`, `distance_meters?`, `avg_heart_rate?`,
  `max_heart_rate?`, `peak_speed_kmh?`, `avg_cadence?`, `calories?`,
  `idle_time?`, `idle_time_sec?`, `avg_speed_kmh?`, `avg_moving_speed_kmh?`,
  `ride_date?`). Depends on T1.

- [x] **T4. Build the participant column registry.**
  New `participantColumns.tsx`: define `ParticipantColumn`, the ordered column
  list (existing + new metric columns per "Available columns"), `render`
  functions (reuse `formatSpeed`, `formatDistanceKm`, `metersToKm` from
  `utils/format`; show "-" when a value is missing), `alwaysVisible`/
  `defaultVisible` flags, and `DEFAULT_VISIBLE_KEYS`. Depends on T3.

- [x] **T5. Add the `useColumnPreferences` persistence hook.**
  New `hooks/useColumnPreferences.ts`: `(storageKey, allKeys, defaultKeys)` →
  `{ visibleKeys, toggle, reset, isVisible }`. **Hydration-safe:** initialize
  `visibleKeys` to `defaultKeys` (so server HTML and the first client render
  match), then read `localStorage` and apply it in a `useEffect` *after mount* —
  reading storage during the initial render would cause a hydration mismatch.
  Persist on change (guard `typeof window`), and reconcile stored keys with the
  current registry (drop unknown, keep order, apply defaults for newly added
  columns). Logging: `DEBUG` on load/persist/reset. Depends on T4.

### Phase 3 — Frontend: UI wiring

- [x] **T6. Build the `ColumnSettings` dropdown.**
  New `ColumnSettings.tsx` using `ui/dropdown/Dropdown`. The component's API:
  it takes `isOpen`/`onClose` props and only closes on outside-click when the
  trigger carries the `.dropdown-toggle` class; `DropdownItem` is for action
  links, not checkboxes. So: manage open state locally (`useState`), render a
  "Columns" trigger button whose `className` includes `dropdown-toggle`, and
  render the checkbox rows **directly inside `<Dropdown>`** (not via
  `DropdownItem`) — one per toggleable column (always-visible columns shown
  checked+disabled), plus a "Reset to defaults" action. Driven by the hook from
  T5. Depends on T5.

- [x] **T7. Render `ParticipantsTable` from visible columns.**
  Refactor `ParticipantsTable.tsx` to accept the resolved visible-column
  descriptors and render headers/cells via `column.render(participant)`. Keep
  the finished-row highlight and username link (as a column renderer). Replace
  the fixed `min-w-[1320px]` wrapper with natural width + `overflow-x-auto`.
  Depends on T4.

- [x] **T8. Wire the picker into the participants page.**
  In `participants/page.tsx`, call `useColumnPreferences` with the registry
  keys/defaults, render `ColumnSettings` next to the filters/header, and pass
  the visible columns into `ParticipantsTable`. Depends on T6, T7.

### Phase 4 — Tests

- [x] **T9. Tests.**
  Backend: extend `participants_test.go` to assert the list response includes
  the ride metrics + computed fields for a participant with a result, and "-"/
  omitted for one without. Frontend: a focused test for `useColumnPreferences`
  (default set, persist round-trip, registry reconciliation). Depends on T8.

## Commit Plan

- **C1 (after T1–T2):** `feat(participants): expose ride metrics on the list DTO`
- **C2 (after T3–T5):** `feat(participants): column registry and persisted column preferences`
- **C3 (after T6–T8):** `feat(participants): configurable columns picker in the list`
- **C4 (after T9):** `test(participants): list metrics and column preferences` (only if fixups needed)

## Notes / Risks

- **Branch base:** created from `feature/server-side-pagination`, not `main`,
  because it builds on the not-yet-merged participants list/table changes. Merge
  order must respect this (pagination branch first, or rebase onto `main` after
  it lands).
- **Persistence scope:** per-browser via `localStorage` (matches "remembered
  after reload"). Not synced per-admin across devices; revisit only if needed.
- **No migration:** metric columns already exist (migration `00022`).
- **Wide table:** with many columns enabled the table scrolls horizontally;
  keep an always-visible identity column (username) so rows stay readable.
- **Registry evolution:** the hook must tolerate stored keys that no longer
  exist and apply defaults for newly added columns, so old `localStorage` values
  never break the table.

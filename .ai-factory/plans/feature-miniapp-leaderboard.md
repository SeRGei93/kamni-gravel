# Implementation Plan: Miniapp Leaderboard

Branch: feature/miniapp-leaderboard
Created: 2026-07-08

## Settings
- Testing: yes (backend unit/handler tests)
- Logging: standard (INFO for served requests, WARN for missing active event, matches existing miniapp handler)
- Docs: yes (Swagger + README)

## Scope

Extend the existing read-only Telegram Mini App (currently a gift catalog) with a
participant **leaderboard**, and split the two experiences with a top tab bar.

- Add a persistent top tab bar with two tabs: **Лидерборд** and **Призы**.
  - "Призы" is the existing gift catalog (`/miniapp/gifts`), unchanged.
  - "Лидерборд" is the new screen (`/miniapp/leaderboard`).
- Leaderboard list shows **all participants** of the active event:
  - Finishers ranked by total (elapsed) time, then non-finishers / DNF /
    disqualified listed after without a place.
  - Each row: place, name, total time (полное), clean/moving time (чистое),
    gender and bike-type badges.
- Detail card (tap a row → `/miniapp/leaderboard/{participantId}`) shows the full
  result data mirroring the dashboard result card: status (Проехал), Strava
  result link, submitted date, and the metric grid — Общее время, Время в
  движении, Простой, Ср. скорость, Ср. скорость в движении, Дата проезда,
  Дистанция, Пиковая скорость, Средний пульс, Максимальный пульс, Средний
  каденс, Калории.
- Filters by gender and bike type, matching the gift catalog filter UX.
- Does not change result submission, admin dashboards, prize distribution, or the
  existing gift catalog behavior.

## Key Decisions

- **New endpoint** `GET /api/miniapp/leaderboard` (Telegram init-data protected,
  active event scoped). Returns all participants of the active event with a
  **public-safe DTO** (no `notes`, `has_gift`, `user_id`, prizes, registration
  date). Reuses the existing `query.GetParticipantsHandler` (already sorts
  finishers by elapsed time and assigns an absolute place) — no new ranking logic
  in Go, no duplication of `calculatePlaces`.
- **Client-side filter + rank.** The backend returns the full participant list
  once; the frontend applies gender/bike filters and computes the displayed rank
  as the position among **ranked finishers within the current filtered view**
  (sorted by `elapsed_time_sec` asc; DNF/disqualified and non-finishers excluded
  from ranking and shown after). This makes any filter combination (gender only,
  bike only, both, none) coherent without needing extra per-dimension place
  columns from the backend.
- **No dedicated detail endpoint.** The list response already carries every
  metric, so the detail page reads the participant from a shared leaderboard
  context cache (mirrors how the gift detail page reuses the catalog list). This
  keeps navigation instant and preserves filters/scroll on back.
- **Reuse existing formatting**: `formatSpeed`, `formatDistanceKm` from
  `frontend/src/utils/format.ts`; time strings (`elapsed_time`, `moving_time`,
  `idle_time`) come pre-formatted from the backend DTO (`entity.FormatSeconds`).
- **Tabs live in the miniapp layout** so they persist across both screens and do
  not remount on navigation. The existing bot WebApp button (opens `/miniapp/gifts`)
  is left unchanged; tabs let users reach the leaderboard.

## Current State

- Miniapp routes live under `frontend/src/app/(miniapp)/miniapp/gifts/*`, wrapped
  by `(miniapp)/layout.tsx` which provides `MiniappCatalogProvider` + Telegram
  theme + `telegram-web-app.js`.
- Backend miniapp routes are in `server.go` under `r.Route("/miniapp", ...)`
  behind `s.telegramWebAppAuth`: `/session`, `/gifts`, `/telegram/files/{fileId}`.
  `MiniappHandler` (`handler/miniapp.go`) resolves the active event via
  `eventRepo.FindActive`.
- `query.GetParticipantsHandler` (`get_participants.go`) returns
  `[]*ParticipantWithPlace` — filters + sorts finishers by elapsed time and
  assigns an absolute `Place`. It is constructed at `server.go:149`
  (`getParticipantsHandler`), in scope at the `NewMiniappHandler` call
  (`server.go:299`).
- `dto.FromParticipant` (`dto/participant.go`) already maps all ride metrics and
  computed fields (idle time, avg/moving speed, ride date, HR, cadence, calories)
  and formats elapsed/moving times — reuse it as the single source of truth.
- Frontend `Participant` / `Result` types (`types/index.ts`) already describe
  every metric field; `frontend/src/api/miniapp.ts` is the typed init-data client;
  `GiftFilters.tsx` / `GiftCatalogTable.tsx` are the UI patterns to mirror.

## Commit Plan

- **Commit 1** (after tasks 1-3): `feat(miniapp): add leaderboard API endpoint`
- **Commit 2** (after tasks 4-5): `feat(miniapp): add leaderboard tabs and data client`
- **Commit 3** (after tasks 6-7): `feat(miniapp): add leaderboard list and detail UI`
- **Commit 4** (after tasks 8-9): `docs(miniapp): document leaderboard endpoint + verify`

## Tasks

### Phase 1: Backend Leaderboard API

- [x] 1. Add a public-safe leaderboard DTO.

  Files:
  - `backend/internal/application/dto/miniapp_leaderboard.go` (new)

  Deliverable:
  - Add `MiniappLeaderboardEntryDTO` exposing only participant-public fields:
    `id`, `name` (display name), `gender`, `bike_type`, `status`, `is_finished`,
    `place`, `elapsed_time`/`elapsed_time_sec`, `moving_time`/`moving_time_sec`,
    `idle_time`, `result_link`, `submitted_at` (result submission date),
    `ride_date`, `distance_meters`, `avg_speed_kmh`, `avg_moving_speed_kmh`,
    `peak_speed_kmh`, `avg_heart_rate`, `max_heart_rate`, `avg_cadence`,
    `calories`. Do NOT expose `user_id`, `notes`, `has_gift`, prizes, or
    `registered_at`.
  - Add `MiniappLeaderboardResponse { Participants []*MiniappLeaderboardEntryDTO; Total int }`.
  - Add constructor `NewMiniappLeaderboardEntry(p *entity.Participant, place int)`:
    build an intermediate `FromParticipant(p)` and copy only the public fields
    (keeps metric/time formatting in one place). Display name = first+last, else
    username, else "Участник" fallback (never leak the numeric Telegram id).
  - Keep the DTO free of any dependency on the `query` package (accept
    `entity.Participant` + `int`, not `ParticipantWithPlace`) to avoid an import
    cycle.

  Logging requirements:
  - No logging in the DTO layer.

- [x] 2. Add the leaderboard handler, route, and wiring.

  Files:
  - `backend/internal/infrastructure/http/handler/miniapp.go`
  - `backend/internal/infrastructure/http/server.go`

  Deliverable:
  - Inject `*query.GetParticipantsHandler` into `MiniappHandler` (add a struct
    field + constructor param on `NewMiniappHandler`); pass the existing
    `getParticipantsHandler` (server.go:149) at the call site (server.go:299).
    Update `newMiniappHandlerWithFileFetcher` accordingly.
  - Add `func (h *MiniappHandler) Leaderboard(w, r)`:
    - Require the Telegram user from context (401 if missing), then resolve the
      active event via the existing `h.activeEvent` helper (404 when none).
    - Call `getParticipantsHandler.Handle` with only `EventID` set (no filters —
      filtering is client-side).
    - Map each `*ParticipantWithPlace` to `dto.NewMiniappLeaderboardEntry(pwp.Participant, pwp.Place)`.
    - Respond with `dto.MiniappLeaderboardResponse{Participants, Total}`.
  - Register `r.Get("/leaderboard", s.miniappHandler.Leaderboard)` inside the
    existing `r.Route("/miniapp", ...)` block in server.go.

  Logging requirements:
  - INFO on served leaderboard request: `telegram_user_id`, `event_id`,
    `participant_count` (match the phrasing/style of the existing `Gifts` handler).
  - WARN on missing Telegram user / no active event (reuse existing helpers'
    logging where possible).
  - Never log participant names, links, or init data.

- [x] 3. Add backend tests for the leaderboard handler.

  Files:
  - `backend/internal/infrastructure/http/handler/miniapp_test.go`

  Deliverable:
  - Test `GET /api/miniapp/leaderboard` happy path: returns participants with
    places for finishers and no place for non-finishers/DNF; assert list ordering
    (finishers by elapsed time first) and `total`.
  - Assert the response JSON does **not** include admin-only keys (`user_id`,
    `notes`, `has_gift`, `registered_at`).
  - Test "no active event" → 404 (reuse the existing miniapp test fakes/harness).
  - Follow the existing test style in `miniapp_test.go` (same fakes for event/
    participant repos and Telegram context).

  Logging requirements:
  - Tests assert behavior, not logs; use synthetic users/events only.

### Phase 2: Frontend Data + Navigation

- [x] 4. Add leaderboard types and API client method.

  Files:
  - `frontend/src/types/index.ts`
  - `frontend/src/api/miniapp.ts`

  Deliverable:
  - Add `MiniappLeaderboardEntry` interface mirroring the backend DTO field names
    (snake_case JSON keys) and `MiniappLeaderboardResponse { participants: MiniappLeaderboardEntry[]; total: number }`.
  - Add `miniappApi.getLeaderboard(): Promise<MiniappLeaderboardResponse>` hitting
    `GET ${MINIAPP_PREFIX}/leaderboard` via the existing `miniappRequest` helper
    (init-data header handled automatically).

  Logging requirements:
  - Reuse the existing client `console.warn` failure path; no new logging.

- [x] 5. Add the top tab bar and leaderboard context, wire into the layout.

  Files:
  - `frontend/src/components/miniapp/MiniappTabs.tsx` (new)
  - `frontend/src/components/miniapp/MiniappLeaderboardContext.tsx` (new)
  - `frontend/src/app/(miniapp)/layout.tsx`

  Deliverable:
  - `MiniappTabs`: client component using `usePathname` to render two tab links —
    "Лидерборд" (`/miniapp/leaderboard`) and "Призы" (`/miniapp/gifts`) — with the
    active tab highlighted using existing `tg-*` theme classes. Sticky/top,
    mobile-first, matches the miniapp visual language.
  - `MiniappLeaderboardProvider` (mirror `MiniappCatalogContext`): holds the
    gender filter, bike-type filter (persisted to `sessionStorage`), cached
    session, and cached participant list, so switching to a detail card and back
    keeps filters/scroll and avoids a refetch. Expose a `getParticipant(id)`
    selector over the cached list for the detail page.
  - Layout: render `<MiniappTabs />` above `{children}` and wrap children with
    `MiniappLeaderboardProvider` alongside the existing `MiniappCatalogProvider`.

  Logging requirements:
  - No runtime logs; remove any temporary diagnostics before completion.

### Phase 3: Frontend Leaderboard UI

- [x] 6. Build the leaderboard list screen (page, filters, table, ranking).

  Files:
  - `frontend/src/app/(miniapp)/miniapp/leaderboard/page.tsx` (new)
  - `frontend/src/app/(miniapp)/miniapp/leaderboard/loading.tsx` (new)
  - `frontend/src/app/(miniapp)/miniapp/leaderboard/error.tsx` (new)
  - `frontend/src/components/miniapp/LeaderboardFilters.tsx` (new)
  - `frontend/src/components/miniapp/LeaderboardTable.tsx` (new)
  - `frontend/src/components/miniapp/LeaderboardEmptyState.tsx` (new)
  - `frontend/src/utils/leaderboard.ts` (new)

  Deliverable:
  - Page loads the miniapp session then the leaderboard (via the new context;
    reuse the gifts page's Telegram `ready()`/`expand()` + `waitForTelegramInitData()`
    bootstrapping and the loading/error/empty shell states).
  - `frontend/src/utils/leaderboard.ts`: `rankAndFilter(entries, gender, bikeType)`
    → filter by gender (`all`|`male`|`female`) and bike type (`all`|specific),
    then split into ranked finishers (status active/ranked, `is_finished`,
    `elapsed_time_sec > 0`) sorted by `elapsed_time_sec` asc with sequential
    display places, followed by the rest (no place). Unit-testable pure function.
  - `LeaderboardFilters`: gender chips (Все / Мужчины / Женщины) + bike-type chips
    from `BIKE_TYPE_OPTIONS`; same chip styling as `GiftFilters`.
  - `LeaderboardTable`: dense table — columns place / name / total time / clean
    time, with small gender + bike-type badges under the name. Rows are clickable
    (`role="link"`, keyboard accessible) → `/miniapp/leaderboard/{id}`, mirroring
    `GiftCatalogTable` row behavior. "—" for participants without a place.
  - `LeaderboardEmptyState`: shown when the active event has no participants.

  Logging requirements:
  - `console.warn` on session/leaderboard load failure with a short message only
    (no names, no init data), matching the gifts page pattern.

- [x] 7. Build the leaderboard detail card.

  Files:
  - `frontend/src/app/(miniapp)/miniapp/leaderboard/[id]/page.tsx` (new)
  - `frontend/src/app/(miniapp)/miniapp/leaderboard/[id]/loading.tsx` (new)
  - `frontend/src/components/miniapp/LeaderboardDetailView.tsx` (new)

  Deliverable:
  - Detail page resolves the participant id from the route, reads it from the
    leaderboard context cache; if the cache is empty (deep link / reload), fetch
    the leaderboard once and find by id (mirror the gift detail page fallback).
    Show a compact loading state and a "не найден" state with a back link.
  - `LeaderboardDetailView`: header with place, name, gender + bike badges, and a
    status pill ("Проехал" for finished). Then the result card:
    - Strava result link (when present) + submitted date.
    - Metric grid (2–3 cols, mobile-first) reusing dashboard labels/values:
      Общее время (`elapsed_time`), Время в движении (`moving_time`), Простой
      (`idle_time`), Ср. скорость (`formatSpeed(avg_speed_kmh)`), Ср. скорость в
      движении (`formatSpeed(avg_moving_speed_kmh)`), Дата проезда (`ride_date`),
      Дистанция (`formatDistanceKm(distance_meters)`), Пиковая скорость
      (`formatSpeed(peak_speed_kmh)`), Средний пульс / Максимальный пульс
      (`… уд/мин`), Средний каденс (`… об/мин`), Калории (`… ккал`).
    - Gracefully hide/placeholder metrics that are absent for a participant.
  - Style with `tg-*` theme variables; keep readable on ~390px width.

  Logging requirements:
  - `console.warn` on fallback fetch failure only; no names/links/init data.

### Phase 4: Docs + Verification

- [x] 8. Document the leaderboard endpoint.

  Files:
  - `backend/docs/swagger.yaml` (or the project's OpenAPI source, if different)
  - `README.md`

  Deliverable:
  - Add an OpenAPI entry for `GET /api/miniapp/leaderboard` (Telegram init-data
    protected, returns `MiniappLeaderboardResponse`), consistent with the existing
    `/api/miniapp/gifts` documentation.
  - Add a short README note that the miniapp now exposes a leaderboard tab in
    addition to the gift catalog.

  Logging requirements:
  - Documentation only; no runtime logs.

- [x] 9. Verify and fix regressions.

  Files:
  - Only files touched above, unless verification surfaces a direct regression.

  Deliverable:
  - `cd backend && go test ./...`.
  - `cd frontend && npm run lint` (targeted miniapp files must be clean; note any
    pre-existing dashboard lint failures unrelated to this change).
  - `cd frontend && npm run build`.
  - Mobile smoke (~390px) via browser/Playwright:
    - tabs switch between Лидерборд and Призы without full reload;
    - leaderboard list ranks finishers, lists non-finishers after without a place;
    - gender/bike filters re-rank the visible view;
    - a row opens the detail card with all present metrics;
    - no dashboard sidebar/header; rows do not overflow the viewport.

  Logging requirements:
  - Keep standard runtime logs; remove any temporary debug logs before completion.

## Sources Checked

- Existing gift miniapp plan: `.ai-factory/plans/feature-gift-miniapp.md`
- Backend: `handler/miniapp.go`, `server.go`, `query/get_participants.go`,
  `dto/participant.go`, `domain/repository/result.go`
- Frontend: `app/(miniapp)/**`, `components/miniapp/**`, `api/miniapp.ts`,
  `types/index.ts`, `utils/format.ts`, `constants/options.ts`,
  `app/(dashboard)/participants/[id]/page.tsx` (result card reference)

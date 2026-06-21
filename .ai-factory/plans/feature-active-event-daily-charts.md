# Active-event daily charts (finishers per day + new participants per day)

- **Branch:** `feature/active-event-daily-charts`
- **Created:** 2026-06-21
- **Type:** feature (backend + frontend)

## Settings

- **Testing:** yes (Go unit tests for the new daily-stats query handler: bucketing, gap-filling, start-date anchoring)
- **Logging:** verbose (DEBUG logs of resolved event/start date, bucket counts, date ranges)
- **Docs:** warn-only (no mandatory docs checkpoint)

## Roadmap Linkage

- Milestone: "none"
- Rationale: Skipped by user (no `.ai-factory/ROADMAP.md` present).

## Goal

Add two per-day charts for the **active event** to the main admin dashboard (`/`):

1. **Finishers per day** — number of participants who submitted a ride result on each day. Counted by `DATE(results.submitted_at)`. The day axis is **anchored at the event start** (`events.start_date`) and runs through today, gap-filled with zeros. Daily increments (not cumulative).
2. **New participants per day** — number of registrations per day, counted by `DATE(participants.registered_at)`. The day axis runs from the first registration through today, gap-filled with zeros.

## Approach & Key Decisions

- **In-memory aggregation**, mirroring the existing `query.GetStatsHandler` (`backend/internal/application/query/get_stats.go`). A single `participantRepo.FindByEvent(ctx, eventID)` call returns participants **with their current result preloaded** (`p.Result`, populated via `LEFT JOIN results ... is_current = true`), so both series come from one query — no new repository methods, no raw SQL.
  - Registrations bucket: `participant.RegisteredAt`.
  - Finishes bucket: `participant.Result.SubmittedAt` for participants where `p.Result != nil`.
- **One endpoint, both series, keyed by explicit `eventId`.** `GET /api/events/{eventId}/stats/daily` returns `{ event_id, event_name, start_date, registrations: [{date,count}], finishes: [{date,count}] }`. Public read (no auth), consistent with the existing `/api/stats` and `/api/events/{eventId}/stats` routes. The query resolves the event via `eventRepo.FindByID` only — **no `FindActive` fallback** (the route always carries an id, so a `*uint`/active-event branch would be dead code).
- **Active event resolution on the frontend**, reusing the established pattern: `extractActiveEvent(await eventsApi.getActive())` (`frontend/src/utils/events.ts`, returns `Event | null`) → `activeEventId`, then `statsApi.getDailyByEvent(activeEventId)`.
- **Date bucketing timezone:** stored timestamps are `TIMESTAMP` (no tz). Bucket by calendar day via a `dayKey(t)` helper using `t.UTC()` (**never `.Local()`** — that would shift buckets by the container TZ) and format dates as `YYYY-MM-DD`. Per-event timezone is a known, out-of-scope simplification.
- **Gap-filling + range guards:** emit one point per calendar day across the series range with `count: 0` for empty days, so the chart x-axis is continuous. Clamp each range so `start ≤ end`.
  - Finishes range: `[ DATE(start_date) .. DATE(today) ]`. If `start_date` is in the future (event not started) → empty finishes. If `start_date` is nil → fall back to the first finish date (or empty series if none).
  - Registrations range: `[ DATE(first registration) .. DATE(today) ]`. Empty series if there are no participants.
- **Logging convention:** the project has no `slog` and no DEBUG level — it uses `log.Printf("INFO|WARN|ERROR ... key=value", ...)` (see `command/register_participant.go`). "Verbose" here means one `INFO` summary line per request plus `WARN`/`ERROR` on failure paths.
- **Charts:** reusable prop-driven **column/bar** ApexChart component (both charts are per-day counts), following the existing `BarChartOne`/`LineChartOne` conventions (dynamic import, `ssr: false`, `ApexOptions`). The existing template chart components are left untouched; a new prop-driven component is added.

## Tasks

### Phase 1 — Backend: daily-stats query, DTO, endpoint, tests

- [x] **Task 1 — Daily-stats query handler.** New file `backend/internal/application/query/get_daily_stats.go`.
  - `GetDailyStatsQuery{ EventID uint }` — resolve via `eventRepo.FindByID` **only** (error `"event not found"` when nil, mirroring `get_stats.go`). No `FindActive` fallback: the route always carries an id, so an active-event branch would be dead code.
  - Result types: `DailyCount{ Date string; Count int }` and `EventDailyStats{ EventID uint; EventName string; StartDate *time.Time; Registrations []DailyCount; Finishes []DailyCount }`.
  - `NewGetDailyStatsHandler(eventRepo repository.EventRepository, participantRepo repository.ParticipantRepository)`.
  - `Handle(ctx, GetDailyStatsQuery) (*EventDailyStats, error)`: `FindByID` → `participantRepo.FindByEvent` → bucket registrations by `RegisteredAt` and finishes by `Result.SubmittedAt` (only `p.Result != nil`) → gap-fill each series over its range (finishes anchored at `DATE(start_date)`; registrations from first registration) → `YYYY-MM-DD` keys, sorted ascending.
  - `dayKey(t time.Time) string` buckets with `t.UTC()` (never `.Local()`).
  - **Edge guards:** clamp each range so `start ≤ end`; future `start_date` → empty finishes; nil `start_date` → first-finish fallback (or empty); no participants → empty registrations.
  - **Logging (project convention — `log.Printf` with `INFO`/`WARN`/`ERROR` prefix + `key=value`, no slog/DEBUG):** one `INFO` summary line (event_id, event_name, start_date, participant count, registrations range+total, finishes range+total); `WARN`/`ERROR` on not-found / repo-error paths.
  - Files: `backend/internal/application/query/get_daily_stats.go`.

- [x] **Task 2 — DTO.** New file `backend/internal/application/dto/daily_stats.go`, mirroring `dto/stats.go`.
  - `DailyCountDTO{ Date string `json:"date"`; Count int `json:"count"` }`.
  - `DailyStatsDTO{ EventID uint `json:"event_id"`; EventName string `json:"event_name"`; StartDate *string `json:"start_date"`; Registrations []DailyCountDTO `json:"registrations"`; Finishes []DailyCountDTO `json:"finishes"` }`.
  - `FromEventDailyStats(s *query.EventDailyStats) *DailyStatsDTO` (format `StartDate` as RFC3339 or nil).
  - Files: `backend/internal/application/dto/daily_stats.go`.
  - **Depends on Task 1** (maps its result type).

- [x] **Task 3 — HTTP handler, route, wiring.**
  - Extend `StatsHandler` (`backend/internal/infrastructure/http/handler/stats.go`): add `getDailyStatsHandler *query.GetDailyStatsHandler` field, update `NewStatsHandler` signature to accept it, and add `GetDailyByEventID(w, r)` — parse `chi.URLParam(r, "eventId")` via `strconv.ParseUint` (BadRequest on parse error, NotFound on "event not found"), call the query handler, map with `dto.FromEventDailyStats`, return via `response.Success`. Mirror `GetByEventID`.
  - Register route in `setupRouter` (`backend/internal/infrastructure/http/server.go`, right after line ~453 `r.Get("/events/{eventId}/stats", ...)`): `r.Get("/events/{eventId}/stats/daily", s.statsHandler.GetDailyByEventID)` in the public `/api` group.
  - Wire in `NewServer`: construct `getDailyStatsHandler := query.NewGetDailyStatsHandler(eventRepo, participantRepo)` near the existing `getStatsHandler` construction (~line 164) and pass it into `handler.NewStatsHandler(...)` (~line 288).
  - **Logging:** handler logs request event id and error paths (mirror existing `log.Printf` usage in `stats.go`).
  - Files: `backend/internal/infrastructure/http/handler/stats.go`, `backend/internal/infrastructure/http/server.go`.
  - **Depends on Task 1, Task 2.**

- [x] **Task 4 — Backend unit tests.** New file `backend/internal/application/query/get_daily_stats_test.go`, following the hand-written repo-fake pattern in `get_gifts_test.go` / `command/submit_result_test.go`.
  - Fakes must implement the **full** `EventRepository` and `ParticipantRepository` interfaces (Go requires it) — stub unused methods returning `nil`; only `FindByID` (event with known `start_date`) and `FindByEvent` (participants with `RegisteredAt` across several days, some with `Result.SubmittedAt`) carry data. Mirror `submitEventRepoFake`.
  - Assert: registrations per-day counts and range; finishes per-day counts; finishes axis starts at `DATE(start_date)`; gap days are filled with `0`; participants without a result are excluded from finishes; date keys sorted ascending.
  - Edge cases: nil `start_date` (registrations-only / finishes fallback); future `start_date` (empty finishes); no participants (empty registrations).
  - Run: `cd backend && go test ./internal/application/query/...`.
  - Files: `backend/internal/application/query/get_daily_stats_test.go`.
  - **Depends on Task 1.**

### Phase 2 — Frontend: types, API, chart component, dashboard integration

- [x] **Task 5 — Types + API client.**
  - `frontend/src/types/index.ts`: add `export interface DailyCount { date: string; count: number; }` and `export interface EventDailyStats { event_id: number; event_name: string; start_date: string | null; registrations: DailyCount[]; finishes: DailyCount[]; }`.
  - `frontend/src/api/stats.ts`: add `async getDailyByEvent(eventId: number): Promise<EventDailyStats> { return get<EventDailyStats>(`${EVENTS_PREFIX}/${eventId}/stats/daily`); }`.
  - Files: `frontend/src/types/index.ts`, `frontend/src/api/stats.ts`.
  - **Depends on Task 3** (response contract).

- [x] **Task 6 — Reusable daily bar-chart component.** New file `frontend/src/components/charts/EventDailyChart.tsx`.
  - Props: `{ title: string; categories: string[]; data: number[]; color?: string }`.
  - Dynamic `react-apexcharts` import with `ssr: false`, `ApexOptions`, `type: "bar"` column chart following `BarChartOne` styling (borderRadius, columnWidth, hidden toolbar, Outfit font).
  - Wrap the chart in a horizontal-scroll container like `BarChartOne` (`<div className="max-w-full overflow-x-auto custom-scrollbar"><div className="min-w-[...]">`) so multi-week events don't crush x-axis labels. Wrap that in a titled card consistent with the dashboard's card styling (`rounded-lg border ... dark:...`). Show an empty-state message when `data` is empty.
  - Files: `frontend/src/components/charts/EventDailyChart.tsx`.
  - **Independent** (no backend dependency).

- [x] **Task 7 — Dashboard integration.** Edit `frontend/src/app/(dashboard)/page.tsx`.
  - Resolve the active event: `extractActiveEvent(await eventsApi.getActive())` (`@/utils/events`, returns `Event | null`) → `activeEventId`; then `statsApi.getDailyByEvent(activeEventId)` → `EventDailyStats`. Use its own state/effect, independent of the existing `loadStats`, so one failure doesn't blank the page.
  - Add a new **"Активное событие"** section (above or below the existing per-event stats) rendering two `EventDailyChart`s: "Проехавшие по дням" (from `finishes`) and "Новые участники по дням" (from `registrations`).
  - Map each series: `data` = counts; `categories` = short date labels formatted with **plain JS** (no date lib exists — e.g. `new Date(p.date).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' })`, or slice the `YYYY-MM-DD`). Do not add a dependency.
  - Gracefully handle: no active event (hide the section / show a note), loading, and fetch error (`console.error`, consistent with the existing `loadStats` catch).
  - Files: `frontend/src/app/(dashboard)/page.tsx`.
  - **Depends on Task 5, Task 6.**

## Commit Plan

- **Commit 1 — after Task 4** (backend complete): `feat(stats): add per-day daily stats endpoint for active event`
  - Tasks 1–4: query handler, DTO, endpoint + wiring, unit tests.
  - Checkpoint: `cd backend && go build ./... && go test ./internal/application/query/...`.
- **Commit 2 — after Task 7** (frontend complete): `feat(dashboard): active-event daily charts (finishers + new participants per day)`
  - Tasks 5–7: types/API, chart component, dashboard integration.
  - Checkpoint: `cd frontend && npm run lint && npm run build`.

## Verification

- Backend: `cd backend && go build ./... && go test ./internal/application/query/...`.
- Frontend: `cd frontend && npm run lint && npm run build`.
- End-to-end (Docker): `make docker-up` + `docker-compose run --rm migrate up`, log in to the dashboard, confirm both charts render for the active event with sane day buckets (finishers axis starts at the event start date; registrations span from the first registration).

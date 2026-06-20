# Server-side pagination for gifts / participants / prize-distribution / criteria

- **Branch:** `feature/server-side-pagination`
- **Created:** 2026-06-20
- **Type:** enhancement (backend + frontend)

## Settings

- **Testing:** yes (backend unit tests for helper, repos, queries, handlers; frontend query-string/build tests where a test setup exists)
- **Logging:** verbose (DEBUG logs of resolved page params, totals, filters)
- **Docs:** warn-only (no mandatory docs checkpoint)
- **API style:** `page` + `page_size` query params → internal `LIMIT/OFFSET`; response keeps its named array key and adds `total` (full filtered count), `page`, `page_size`.

## Roadmap Linkage

- Milestone: "none"
- Rationale: Skipped by user (no `.ai-factory/ROADMAP.md` present).

## Goal

Add **server-side** pagination to 4 admin pages that currently load the entire result set:
`gifts`, `participants`, `prize-distribution`, `criteria`.

## Requirements / Decisions

1. **Page size is configurable, constrained to 50–100** (user requirement). Default **50**; UI selector offers **50 / 100**; server clamps any value into `[50, 100]`. `page` is 1-based (default 1).
2. **Response envelope:** keep the existing entity key (`gifts` / `participants` / `distribution` / `criteria`) and add `total`, `page`, `page_size`. `total` becomes the **full filtered count** (via `COUNT(*)`), not `len(page)`.
3. **Client-side filters/search must move server-side.** With server pagination, any filter applied only to the loaded page would be wrong. Affected: participants (bike_type, gender, is_finished, has_gift, search) and prize-distribution (match_reason). Gifts `review_status` and criteria `type` are already server-side.
4. **Aggregates that span the whole set must come from the server**, since the client no longer holds all rows: gifts status-tab badges (`status_counts`) and prize-distribution statistics cards (`stats`).
5. **Two non-trivial cases:**
   - **Participants:** `place` fields are *global ranks*. Compute the places map from the full result set (`FindByEventWithPlaces`, lightweight id→place), then attach to the paginated rows. Filters/search pushed into SQL `WHERE` so `LIMIT/OFFSET` + `COUNT` are consistent.
   - **Prize-distribution:** computed in-memory by the matching engine (needs the full dataset). Pagination = compute full distribution → apply `match_reason` filter → slice the page. Reduces client payload only (DB load is inherent to the engine).

## Current state (from exploration)

- No pagination anywhere; all list SQL is plain `SELECT ... ORDER BY ...` with no `LIMIT/OFFSET`.
- Handlers parse query params via `r.URL.Query().Get(...)`; no shared param/response helper.
- Routes in `backend/internal/infrastructure/http/server.go` (`/api/...`, public read group + protected write group).
- Frontend already has a reusable `frontend/src/components/tables/Pagination.tsx` and a generic API client (`frontend/src/api/client.ts`).
- Response types already include `total` (currently `len(items)`).

## Tasks

Tracked in the task list (`/tasks`). Dependency order:

### Phase 1 — Backend foundation
- **T1** Shared pagination helper `ParsePageParams` + meta (page/page_size 50–100, limit/offset) — `infrastructure/http/handler/pagination.go` (+ tests).

### Phase 2 — Criteria (template, simplest)
- **T2** Backend: paginate criteria (repo LIMIT/OFFSET + COUNT, query, DTO, handler). *blocked by T1*
- **T3** Frontend: criteria page pagination + page-size selector; establish the reusable FE pattern (usePagination / PageSizeSelect). *blocked by T2*

### Phase 3 — Gifts
- **T4** Backend: paginate gifts + `status_counts` aggregate. *blocked by T1*
- **T5** Frontend: gifts page pagination; use server `status_counts` (drop the all-gifts fetch). *blocked by T4*

### Phase 4 — Participants (most complex)
- **T6** Backend: push filters+search to SQL, LIMIT/OFFSET + COUNT, attach global places to the page. *blocked by T1*
- **T7** Frontend: participants page pagination + server-side filters/search (debounced), reset to page 1 on change. *blocked by T6*

### Phase 5 — Prize-distribution
- **T8** Backend: compute-then-slice pagination + server-side `match_reason` filter + `stats` aggregate. *blocked by T1*
- **T9** Frontend: prize-distribution page pagination + server-side filter; stat cards from server `stats`. *blocked by T8*

### Phase 6 — Verify
- **T10** Build + vet + tests (backend), lint + type-check (frontend), manual check of all 4 pages against the restored local DB. *blocked by T3, T5, T7, T9*

## Commit Plan

- **C1 (after T1):** `feat(api): add shared server-side pagination helper (page/page_size 50–100)`
- **C2 (after T2–T3):** `feat(criteria): server-side pagination`
- **C3 (after T4–T5):** `feat(gifts): server-side pagination with status counts`
- **C4 (after T6–T7):** `feat(participants): server-side pagination, filters and search`
- **C5 (after T8–T9):** `feat(prize-distribution): server-side pagination`
- **C6 (after T10):** `test/chore: verify pagination across pages` (only if fixups needed)

## Notes / Risks

- **Working tree carries the earlier Strava-result feature** (telegram-bot files) — a disjoint file set from this work. Recommend committing/PR-ing that separately before implementing so the two features don't intermix.
- Keep `ORDER BY` deterministic (already present) so `LIMIT/OFFSET` paging is stable.
- `page_size` selector is intentionally limited to 50/100 per the user ("и все") — do not add more options.

# Implementation Plan: Participant Edit Lock (pessimistic locking)

Branch: main (no feature branch — work directly on main, per user choice)
Created: 2026-06-22

## Settings
- Testing: yes
- Logging: verbose (DEBUG during development)
- Docs: warn-only (no mandatory docs checkpoint)

## Roadmap Linkage
- Milestone: none
- Rationale: no roadmap artifact present

## Scope And Product Contract

When an admin edits a **participant** in the admin panel, the record is locked
so a second admin **cannot make changes** and **sees that the record is already
being edited** (and by whom). This applies **only to the participant entity**
(the `/participants/[id]` detail page); events, gifts, criteria-as-entities,
admin-users, etc. are out of scope.

Behaviour contract:

- **Acquire on edit, not on view.** Merely opening `/participants/[id]` does NOT
  lock it. The lock is acquired when the admin enters an edit action on the page.
- **One lock per participant, shared by all edit sections.** The detail page has
  three independent edit areas (Notes, Result/timing, Delete) plus criteria edits
  on the result. A single participant-scoped lock covers all of them.
- **Server-side enforcement (the real guarantee).** Every write reachable from the
  participant detail page is rejected with **HTTP 409 Conflict** when the
  participant is locked by *another* admin. The UI indicator is advisory; the
  middleware is authoritative. Guarded endpoints:
  - `PUT /api/participants/{id}` (notes/bike/gender)
  - `DELETE /api/participants/{id}` (delete participant)
  - `POST /api/participants/{participantId}/results` (create result)
  - `PUT /api/results/{id}` (update result/timing) — participant resolved via result
  - `DELETE /api/results/{id}` (delete result) — participant resolved via result
  - `POST /api/results/{id}/criteria` (add criterion) — participant resolved via result
  - `DELETE /api/results/{id}/criteria/{criteriaId}` (remove criterion) — resolved via result
- **Owner-friendly.** The lock owner can always re-acquire/refresh and write
  freely. Acquire by the current owner is idempotent (refresh).
- **Self-healing TTL.** Lock storage is **in-memory** with a TTL. A heartbeat from
  the open editor keeps it alive; if the editor closes the tab / crashes / loses
  network, the lock **expires** after the TTL and the next admin can take over.
  No DB migration, no new table.
- **Visible owner.** A second admin sees a banner: «Запись редактирует
  <username>» and all Edit buttons are disabled until the lock frees (auto-clears
  via a light poll / on next acquire attempt).

Timing constants (single source of truth in the lock package; tune later):
- Lock TTL: **90s** (server-side expiry)
- Frontend heartbeat: **every 30s** while at least one section is in edit mode
- Cleanup goroutine: **every 60s** (memory hygiene; expiry is also checked lazily)
- Foreign-lock UI poll: **every ~15s** while a foreign lock is shown and the user
  is not editing (to auto-re-enable the buttons once it frees)

## Architecture Notes / Decisions

- **In-memory, concrete type — no domain interface.** Mirrors the existing
  precedent `infrastructure/telegram/session.Manager` (map + `sync.RWMutex` +
  cleanup goroutine, used directly, no domain abstraction). The lock is an
  infrastructure concern (process-local runtime state), so it lives in
  infrastructure and is used directly by the HTTP layer. Tradeoff: lost on
  backend restart and single-instance only — acceptable for this admin panel
  (documented in Notes/Risks).
- **Layering stays clean.** The lock type is `infrastructure/lock`. The response
  DTO and mapping live in the HTTP handler package (also infrastructure), so
  nothing in `application/dto` or `domain` imports infrastructure (no dependency
  inversion violation).
- **Enforcement via per-route middleware, handlers untouched.** A middleware
  factory `RequireParticipantUnlocked(mgr, resolveParticipantID)` runs inside the
  existing protected group (after `Auth` + `RequireRole`, so JWT claims are in
  context). Participant identity for guarded routes comes from one of three
  resolvers (route `{id}`, route `{participantId}`, or `{id}` → result →
  `ParticipantID` via `s.resultRepo.FindByID`). Existing participant/result/
  criteria handlers need **no** changes — keeps handlers thin per architecture
  rules.
- **Admin identity** comes from `middleware.GetUserFromContext(ctx)` →
  `claims.UserID` + `claims.Username` (already available; just never used on these
  routes before).
- **Conflict body.** `response.Conflict(w, msg)` carries only a message, so the
  structured conflict payload (with owner info) is written via
  `response.JSON(w, http.StatusConflict, <lockStatusResponse>)`.

## Affected Files

Backend (new):
- `backend/internal/infrastructure/lock/manager.go` — `Manager` + `Lock`,
  in-memory map keyed by participant id, TTL, cleanup goroutine, methods
  `Acquire`, `Refresh`, `Release`, `Get`, `LockedByOther`.
- `backend/internal/infrastructure/lock/manager_test.go` — unit tests.
- `backend/internal/infrastructure/http/handler/participant_lock.go` —
  `ParticipantLockHandler` (`Acquire` POST, `Release` DELETE, `Status` GET) +
  `lockStatusResponse` struct and mapping from `*lock.Lock`.
- `backend/internal/infrastructure/http/handler/participant_lock_test.go` —
  handler + enforcement tests.
- `backend/internal/infrastructure/http/middleware/participant_lock.go` —
  `RequireParticipantUnlocked` factory + the three participant-id resolvers.

Backend (modify):
- `backend/internal/infrastructure/http/server.go` — construct `lock.Manager`
  (no external deps; cleanup goroutine starts in its constructor), store on
  `Server`, build `ParticipantLockHandler`, register the lock routes and wrap the
  seven guarded write routes with the appropriate guard inside the protected
  group.

Frontend (new):
- `frontend/src/hooks/useParticipantLock.ts` — lock lifecycle hook
  (acquire-on-edit, heartbeat, release on stop/unmount/visibility/`beforeunload`,
  foreign-lock state + poll).
- `frontend/src/components/participants/ParticipantLockBanner.tsx` — «Запись
  редактирует <username>» banner.
- `frontend/src/hooks/useParticipantLock.test.ts` — hook lifecycle test.

Frontend (modify):
- `frontend/src/types/index.ts` — add `LockStatus` type.
- `frontend/src/api/participants.ts` — `acquireLock`, `releaseLock`, `getLock`
  (refresh = re-acquire).
- `frontend/src/app/(dashboard)/participants/[id]/page.tsx` — integrate the hook:
  check lock on mount, gate the three Edit sections behind `beginEdit()`, show the
  banner + disable Edit buttons on foreign lock, release on save/cancel/all-closed.

## Tasks

### Phase 1 — Backend: in-memory lock manager

- [x] **T1. Build the in-memory lock manager + unit tests.**
  New package `backend/internal/infrastructure/lock`.
  `Lock` struct: `ParticipantID uint`, `OwnerUserID uint`, `OwnerUsername string`,
  `AcquiredAt time.Time`, `ExpiresAt time.Time`.
  `Manager`: `map[uint]*Lock`, `sync.RWMutex`, `ttl time.Duration`; constructor
  `NewManager(ttl)` starts `go cleanupLoop(ctx)` (mirror `session.Manager`).
  Methods:
  - `Acquire(participantID, ownerUserID uint, ownerUsername string) (*Lock, bool)` —
    grant if free OR expired OR already owned by this user (refresh `ExpiresAt =
    now+ttl`); return `(existingLock, false)` if held by another and not expired.
    "Steal expired" path covered.
  - `Refresh(participantID, ownerUserID uint) (*Lock, bool)` — owner-only heartbeat
    (extends `ExpiresAt`); `false` if not owner / not held.
  - `Release(participantID, ownerUserID uint) bool` — owner-only delete.
  - `Get(participantID uint) (*Lock, bool)` — active (non-expired) lock or none.
  - `LockedByOther(participantID, userID uint) (*Lock, bool)` — used by enforcement.
  Define exported defaults (`DefaultTTL = 90 * time.Second`, cleanup tick 60s).
  Logging: `DEBUG` on acquire/refresh/release and on steal-expired and
  blocked-by-other (include participant id + owner username); `cleanup` logs
  `DEBUG` count of expired locks removed.
  Tests (`manager_test.go`): acquire-when-free; idempotent re-acquire by owner;
  blocked-by-other returns existing owner + false; expired lock is stealable;
  refresh by non-owner fails; release by non-owner fails; `Get` returns none for
  expired. Use an injected/short TTL so expiry is testable without real waits
  (e.g. construct with a tiny ttl, or expose a clock seam — prefer a tiny ttl +
  `time.Sleep` only if unavoidable; better: set `ExpiresAt` directly in the test
  via a helper).

### Phase 2 — Backend: lock HTTP endpoints

- [x] **T2. Add the participant-lock HTTP handler.**
  New `handler/participant_lock.go` with `ParticipantLockHandler{ mgr *lock.Manager }`
  and `NewParticipantLockHandler(mgr)`.
  - `lockStatusResponse{ participant_id, locked bool, locked_by_user_id,
    locked_by_username, acquired_at, expires_at, is_mine bool }` + a mapper from
    `*lock.Lock` and the current `claims.UserID`.
  - `Acquire(w, r)` — `POST /api/participants/{id}/lock`: parse id; read claims via
    `middleware.GetUserFromContext`; `mgr.Acquire(...)`. On success →
    `response.Success(w, status)` (200). On blocked-by-other →
    `response.JSON(w, http.StatusConflict, status)` (409 with owner info). Doubles
    as the heartbeat (owner re-acquire = refresh).
  - `Release(w, r)` — `DELETE /api/participants/{id}/lock`: owner-only release;
    `response.NoContent(w)` (idempotent — releasing a non-owned/absent lock still
    returns 204, to keep `beforeunload`/cancel cleanup simple).
  - `Status(w, r)` — `GET /api/participants/{id}/lock`: return current
    `lockStatusResponse` (locked=false when free). Used by the page on mount and by
    the foreign-lock poll.
  Keep the handler thin (parse → claims → manager call → map → respond).
  Logging: `INFO` on acquire/release with admin username + participant id;
  `DEBUG` on status checks.
  Depends on T1.

### Phase 3 — Backend: enforcement middleware + wiring + routes

- [x] **T3. Add the lock-guard middleware.**
  New `middleware/participant_lock.go`:
  `RequireParticipantUnlocked(mgr *lock.Manager, resolve func(*http.Request)
  (uint, error)) func(http.Handler) http.Handler`. Inside: read claims
  (`GetUserFromContext`); `pid, err := resolve(r)` (on resolver error →
  `response.BadRequest`/`NotFound` as appropriate); if `lock, ok :=
  mgr.LockedByOther(pid, claims.UserID); ok` → `response.JSON(w, 409,
  lockStatusResponse(lock))` and stop; else `next`. Provide resolver helpers:
  - `participantIDFromParam(name string)` — `chi.URLParam(r, name)` → uint
    (used with `"id"` and `"participantId"`).
  - `participantIDFromResult(resultRepo)` — `chi.URLParam(r, "id")` →
    `resultRepo.FindByID(ctx, id)` → `result.ParticipantID`.
  Note: the 409 body shape must match T2's `lockStatusResponse`; factor the mapper
  so both the handler and the middleware reuse it (export it from the handler
  package or a small shared helper — avoid duplicating the JSON shape).
  Logging: `DEBUG` on every guard decision (allow/deny + participant id + actor);
  `WARN` when a resolver fails to map a request to a participant.
  Depends on T1, T2.

- [x] **T4. Wire the manager + handler + routes into the server.**
  In `server.go`:
  - Add `lockManager *lock.Manager` and `participantLockHandler
    *handler.ParticipantLockHandler` fields on `Server`.
  - In `NewServer`, construct `lock.NewManager(lock.DefaultTTL)` and the handler
    (the cleanup goroutine starts inside the constructor — no extra start call).
  - In `setupRouter`, inside the existing protected group
    (`Auth` + `RequireRole("admin")`):
    - Register `POST /participants/{id}/lock`, `DELETE /participants/{id}/lock`,
      `GET /participants/{id}/lock`.
    - Wrap the existing guarded writes with `r.With(guard)`:
      - `PUT /participants/{id}` and `DELETE /participants/{id}` →
        guard with `participantIDFromParam("id")`.
      - `POST /participants/{participantId}/results` →
        guard with `participantIDFromParam("participantId")`.
      - `PUT /results/{id}`, `DELETE /results/{id}`, `POST /results/{id}/criteria`,
        `DELETE /results/{id}/criteria/{criteriaId}` →
        guard with `participantIDFromResult(s.resultRepo)`.
    Keep the route ordering/structure otherwise unchanged.
  Logging: `INFO` one-line startup log confirming the lock manager is enabled with
  its TTL.
  Depends on T2, T3.

### Phase 4 — Frontend: API + lock hook

- [x] **T5. Add lock API wrappers + `LockStatus` type.**
  `types/index.ts`: `LockStatus { participant_id: number; locked: boolean;
  locked_by_user_id?: number; locked_by_username?: string; acquired_at?: string;
  expires_at?: string; is_mine: boolean }`.
  `api/participants.ts`: `acquireLock(id) → POST /api/participants/{id}/lock`
  (returns `LockStatus`; throws `ApiError` with `status===409` and the
  `LockStatus` body when held by another), `releaseLock(id) → DELETE` (ignore
  errors — best-effort), `getLock(id) → GET`. `refreshLock = acquireLock` (the
  POST is idempotent for the owner).
  Logging: `console.debug` on each call (guard with the project's existing debug
  convention if any).
  Depends on T4.

- [x] **T6. Build the `useParticipantLock` hook.**
  New `hooks/useParticipantLock.ts`. Responsibilities:
  - State: `lockStatus`, `isLockedByOther` (derived), `ownerUsername`.
  - `beginEdit(): Promise<boolean>` — calls `acquireLock`; on success start the
    heartbeat interval (30s) and increment an internal active-edit ref count;
    return `true`. On `ApiError` 409 → set foreign-lock state from the body and
    return `false`.
  - `endEdit()` — decrement the ref count; when it reaches 0, stop the heartbeat
    and `releaseLock` (best-effort).
  - On mount → `getLock` to seed `lockStatus`; while a foreign lock is shown and
    the user is not editing, poll `getLock` every ~15s and clear it when freed.
  - Release on unmount, on `visibilitychange→hidden`, and on `beforeunload` (use
    `navigator.sendBeacon` for the unload path so the lock is released even as the
    tab closes; fall back to `releaseLock`).
  - SSR/hydration-safe: no `window`/`localStorage` access during render; do all
    side effects in `useEffect` after mount.
  Logging: `console.debug` on acquire/heartbeat/release/foreign-detected
  transitions.
  Depends on T5.

### Phase 5 — Frontend: page integration

- [x] **T7. Build the lock banner component.**
  New `components/participants/ParticipantLockBanner.tsx`: given `ownerUsername`,
  render a prominent warning banner «Запись редактирует <username>. Изменения
  недоступны.» using the existing alert/badge styling of the dashboard. Renders
  nothing when not locked-by-other.
  Depends on T6.

- [x] **T8. Integrate locking into the participant detail page.**
  In `participants/[id]/page.tsx`:
  - Call `useParticipantLock(participantId)`.
  - On mount the hook seeds lock status; when `isLockedByOther`, render
    `ParticipantLockBanner` and **disable all three Edit triggers** (Notes edit,
    Result edit, Delete) — buttons disabled + tooltip.
  - When the admin clicks Edit on any section, `await beginEdit()` first; only flip
    that section's `isEditing` to `true` if it returns `true`. If `false`, show the
    banner (do not enter edit).
  - Call `endEdit()` when a section's edit finishes: on successful save, on cancel,
    and when leaving the page (the hook also handles unmount/visibility/unload).
  - One shared lock backs all three sections (the hook's ref count handles
    multiple simultaneously-open sections — release only when the last closes).
  - Keep the existing `loadParticipant` / `refreshMatchedGifts` flows intact.
  Logging: `console.debug` on each Edit-gate decision.
  Depends on T6, T7.

### Phase 6 — Tests

- [x] **T9. Backend enforcement + handler tests.**
  `handler/participant_lock_test.go`: Acquire returns 200 for a free participant;
  a second admin's Acquire returns 409 with the first owner's username; owner
  re-acquire (heartbeat) stays 200; Release frees it; Status reflects each state.
  Enforcement: with a lock held by admin A, a `PUT /api/participants/{id}` and a
  `PUT /api/results/{id}` issued as admin B return **409**; the same as admin A
  succeed; after Release, admin B succeeds. (Drive through the wired router /
  guard with a stub `resultRepo` for the result→participant resolver.)
  Depends on T4 (and reuses T1's manager).

- [x] **T10. Frontend hook test.**
  `hooks/useParticipantLock.test.ts`: `beginEdit` acquires and starts heartbeat;
  a mocked 409 sets `isLockedByOther` + `ownerUsername` and returns `false`;
  `endEdit` releases only after the last open section closes (ref count);
  unmount/visibility release path calls `releaseLock`/`sendBeacon`.
  Depends on T6.

## Commit Plan

- **C1 (after T1):** `feat(participant-lock): in-memory edit-lock manager`
- **C2 (after T2–T4):** `feat(participant-lock): lock API, enforcement guard, and routes`
- **C3 (after T5–T6):** `feat(participant-lock): frontend lock API and lifecycle hook`
- **C4 (after T7–T8):** `feat(participant-lock): edit-lock banner and detail-page integration`
- **C5 (after T9–T10):** `test(participant-lock): backend enforcement and frontend hook`

## Notes / Risks

- **In-memory tradeoff (accepted by user):** locks are lost on backend restart and
  are not shared across multiple backend instances. For this single-instance admin
  panel that is acceptable; the short TTL means a "stuck" lock self-heals quickly.
  If the deployment ever scales out or needs restart-durability, migrate to a
  DB-backed `participant_edit_locks` table (domain interface + postgres repo) —
  the HTTP/middleware contract above stays the same.
- **Result→participant resolution cost:** the result-keyed guards do one extra
  `resultRepo.FindByID` per guarded write. These are low-frequency admin writes, so
  the cost is negligible; no caching needed.
- **Unload release is best-effort:** `beforeunload` + `sendBeacon` covers most tab
  closes, but a hard crash relies on the TTL. That is by design.
- **Acquire-on-edit, not on-view:** chosen so read-only viewing never blocks other
  admins. The server enforces on write regardless, so even if the UI gate is
  bypassed the data is protected.
- **409 vs 423:** 409 Conflict is used (a `response.Conflict` helper already
  exists and 409 is the conventional choice here); 423 Locked is the stricter
  WebDAV semantic alternative if ever desired.
- **Scope is participant-only** per the request; the result/criteria writes are
  guarded *because they are edited from the participant page*, keyed on the owning
  participant — not because results/criteria get their own independent locks.

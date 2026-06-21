# Implementation Plan: Miniapp First Screen Caching

Branch: feature/miniapp-first-screen-cache
Created: 2026-06-21

## Settings
- Testing: yes
- Logging: minimal (WARN/ERROR only — no cache hit/miss logging)
- Docs: yes (mandatory docs checkpoint at completion via /aif-docs)
- Cache layer: backend, **file-backed** (persisted to disk under a configurable cache dir)

## Scope

Speed up the Telegram Mini App first screen (`/miniapp/gifts`) by caching the gift catalog
on the backend, and invalidate that cache whenever a gift is approved.

The first screen loads two things on open:
1. `GET /api/miniapp/session` — user + active event (cheap, left uncached).
2. `GET /api/miniapp/gifts?gender=&bike_type=` — the approved-gift catalog. In the default
   `all_genders` state the frontend fires **3 parallel requests** (gender=all/male/female),
   each hitting the DB with criteria/attachment joins. This is the expensive part and the
   target of the cache.

In scope:
- A thread-safe **file-backed** cache of the assembled gift-catalog DTOs, keyed by
  `(eventID, gender, bikeType)`. Entries are persisted as JSON files on disk so the cache
  stays warm across backend restarts/deploys. Atomic writes (temp file + rename); TTL stored
  inside each entry. An optional in-memory mirror may front the files for latency.
- Read-through integration in `MiniappHandler.Gifts` (the gifts query is served from cache;
  participant count stays fresh).
- Cache invalidation on gift approval (explicit user requirement), plus the correctness
  companions: edits to an already-approved gift and deletion of an approved gift.
- Wiring of a single shared cache instance in `server.go`.
- Unit tests for the cache and a handler test for approval-time invalidation.

Out of scope:
- Frontend caching (localStorage/SWR/react-query) — user chose the backend layer. The backend
  cache already makes the 3 parallel default-state calls cheap. Frontend dedup is noted as a
  future optimization only.
- Caching `/api/miniapp/session` and the participant count (they change independently of gift
  approval; kept fresh).
- Caching the Telegram file proxy (`/telegram/files/{fileId}`) — separate concern.

## Current State

- The miniapp is a route group inside the Next.js frontend (`frontend/src/app/(miniapp)/`),
  not a separate app. First screen: `frontend/src/app/(miniapp)/miniapp/gifts/page.tsx`.
- No caching exists anywhere: no SWR/react-query, no localStorage data cache, no backend
  `Cache-Control`/`ETag`. Confirmed across frontend and Go code.
- Backend catalog path: `MiniappHandler.Gifts`
  (`backend/internal/infrastructure/http/handler/miniapp.go`) calls
  `GetMiniappParticipantCountHandler.Handle` (fresh) then `GetMiniappGiftsHandler.Handle`
  (approved gifts, filter-aware), assembles `dto.GiftListResponse`.
- Approval happens at a single point: `command.UpdateGiftHandler.Handle`
  (`backend/internal/application/command/update_gift.go`), reached only via the admin HTTP
  `GiftsHandler.Update` (`backend/internal/infrastructure/http/handler/gifts.go`, ~line 273).
  On transition it returns `UpdateGiftResult{BecameApproved: true}`; the handler already fires
  `notifyPublicGiftApproved` at lines 334-336. The Telegram `ConfirmAddGift` path only creates
  pending gifts — there is **no bot approval path**. So invalidation has exactly one hook.
- DI wiring lives in `backend/internal/infrastructure/http/server.go`:
  miniapp query handlers built ~lines 142-143, `MiniappHandler` ~line 275, `updateGiftHandler`
  ~line 177, `GiftsHandler` ~lines 251-258.

## Assumptions

- "Кеширование первого экрана" = caching the gift catalog response (the heavy query). Session
  and participant count are cheap and stay fresh.
- "Кеш сбрасываем каждый раз когда будет одобрен новый подарок" = invalidate on the
  `BecameApproved` transition in `GiftsHandler.Update`. We also invalidate on edits/deletes of
  already-approved gifts, because otherwise the cache would serve a stale approved catalog —
  a correctness bug, not a new feature.
- File-backed cache: entries persist on disk, so the catalog stays warm across backend
  restarts/deploys. To survive container **recreation** (not just process restart), the cache
  dir must be a mounted Docker volume — otherwise it lives only inside the container FS.
- Single backend instance (Docker Compose). The cache dir is per-process unless a shared volume
  is mounted; event-based invalidation is exact for the instance that handles approval. A
  backstop TTL (default 1 hour, configurable; `<=0` disables expiry) bounds staleness if the
  deployment ever scales to multiple instances or an invalidation path is missed. Documented as
  a known limitation.
- The frontend sends stable filter values (all/male/female × bike types), so
  `(eventID, gender, bikeType)` keys are stable; empty string == "all".

## Tasks

### Phase 1: Cache component

- [x] 1. Create file-backed miniapp gifts cache package + unit tests.

  Files:
  - `backend/internal/infrastructure/cache/miniapp_gifts_cache.go` (new)
  - `backend/internal/infrastructure/cache/miniapp_gifts_cache_test.go` (new)

  Deliverable:
  - `MiniappGiftsCache` struct backed by a directory on disk, thread-safe (`sync.RWMutex`,
    or per-key locking). Keyed by `{eventID uint, gender, bikeType string}`; cached value is
    `[]*dto.GiftDTO`.
  - On-disk layout: one JSON file per key, e.g.
    `<dir>/gift_<eventID>__<gender>__<bikeType>.json`. Empty filter == `all`. Sanitize
    gender/bikeType to a safe charset (they are already validated upstream, but guard anyway).
  - File payload: `{ "cached_at": <unix nanos>, "gifts": [...] }` so TTL is independent of
    filesystem mtime.
  - `NewMiniappGiftsCache(dir string, ttl time.Duration) (*MiniappGiftsCache, error)` —
    `os.MkdirAll(dir)` on construct. Package const `DefaultMiniappGiftsCacheTTL = 1 * time.Hour`;
    `ttl <= 0` => no expiry.
  - `Get(eventID, gender, bikeType) ([]*dto.GiftDTO, bool)` — read+unmarshal the file; miss on
    not-exist, unmarshal error (treat corrupt file as miss, best-effort remove), or expired TTL.
  - `Set(...)` — marshal and write **atomically**: write to `<file>.tmp` then `os.Rename` to the
    final path (atomic on the same filesystem). Stamp `cached_at` with `time.Now()`.
  - `InvalidateEvent(eventID)` — remove all files for that event (glob `gift_<eventID>__*.json`);
    ignore not-exist errors. Optional `InvalidateAll()`.
  - Optional in-memory mirror for latency — only if kept in strict lockstep with the files
    (Set writes both; InvalidateEvent clears both). Not required for correctness.
  - Tests (use `t.TempDir()` for the cache dir): hit/miss, key isolation, `InvalidateEvent`
    scoping, TTL expiry, **persistence** (a second cache instance over the same dir reads prior
    `Set` values), corrupt-file => miss, atomic write (no partial reads), `-race` safety.

  Logging: minimal — WARN only on unexpected file I/O errors during Set/Invalidate (write/rename
  failure); no hit/miss logs. A failed write must degrade to "no cache", never break the request.

### Phase 2: Serve first screen from cache

- [x] 2. Read-through cache in `MiniappHandler.Gifts`. (blocked by 1)

  Files:
  - `backend/internal/infrastructure/http/handler/miniapp.go`

  Deliverable:
  - Narrow interface `miniappGiftsCache { Get(...); Set(...) }` in the handler package;
    `*cache.MiniappGiftsCache` satisfies it. `MiniappHandler` tolerates a nil cache.
  - Add `giftsCache` field + constructor param to `NewMiniappHandler` /
    `newMiniappHandlerWithFileFetcher`; update in-package tests to compile.
  - In `Gifts()`: keep the participant-count call **first** (it validates filters and returns
    400 on invalid gender/bike_type — do not reorder). On cache hit use the cached
    `[]*dto.GiftDTO`; on miss call `getMiniappGiftsHandler.Handle`, build DTOs, then `Set`.
    Preserve existing error handling and the `dto.GiftListResponse` shape.

  Logging: minimal — keep existing lines, add none.

### Phase 3: Invalidate on approval

- [x] 3. Invalidate cache on gift approval (and edits/deletes of approved gifts). (blocked by 1)

  Files:
  - `backend/internal/infrastructure/http/handler/gifts.go`

  Deliverable:
  - Narrow interface `miniappGiftsCacheInvalidator { InvalidateEvent(eventID uint) }`;
    `*cache.MiniappGiftsCache` satisfies it. `GiftsHandler` tolerates a nil invalidator.
  - Add `giftsCache` field + param to `NewGiftsHandler` (keep the variadic `publicGiftNotifier`
    signature valid).
  - In `Update()` (~lines 334-336): invalidate when the resulting gift is approved —
    `if updatedGift != nil && updatedGift.ReviewStatus == approved { InvalidateEvent(updatedGift.EventID) }`.
    This covers both the `BecameApproved` transition (the explicit requirement) and edits to an
    already-approved gift. Keep `notifyPublicGiftApproved` gated on `BecameApproved` only.
  - Delete path: invalidate the event's cache when an **approved** gift is deleted (load its
    `event_id`/`review_status` first if needed).

  Logging: minimal — no new logs; invalidation never blocks/fails the response.

### Phase 4: Wiring

- [x] 4. Wire a single shared cache instance in `server.go` (+ optional TTL config). (blocked by 2, 3)

  Files:
  - `backend/internal/infrastructure/http/server.go`
  - `backend/internal/config/main.go`, `env.example`, `docker-compose.yml`

  Deliverable:
  - Add config for the cache directory (required for file-backed storage): `MINIAPP_CACHE_DIR`
    env → `Config` field, default e.g. `./data/miniapp-cache` (or `/data/miniapp-cache` in the
    container). Optionally also `MINIAPP_GIFTS_CACHE_TTL` env → `time.Duration`, defaulting to
    `cache.DefaultMiniappGiftsCacheTTL`.
  - Build one `miniappGiftsCache, err := cache.NewMiniappGiftsCache(cfg.MiniappCacheDir, ttl)`
    (~lines 142-143); handle the construction error (log WARN and continue with a nil cache =>
    caching disabled, rather than failing startup). Inject the **same** instance into
    `NewMiniappHandler` (~line 275) and `NewGiftsHandler` (~lines 251-258). Do not create two
    caches.
  - docker-compose.yml: mount a named volume at the cache dir so the cache survives container
    recreation; document that without the volume the cache is ephemeral per container.
  - Verify: `cd backend && go build ./... && go test ./...` pass.

  Logging: minimal — single startup INFO/WARN for cache dir init result is acceptable.

### Phase 5: Tests

- [x] 5. Test cache invalidation on approval in `GiftsHandler`. (blocked by 3, 4)

  Files:
  - `backend/internal/infrastructure/http/handler/gifts_test.go`

  Deliverable:
  - Fake invalidator recording `InvalidateEvent` calls.
  - Approval transition (`BecameApproved=true`) invalidates the gift's `EventID` once.
  - Edit of an already-approved gift still invalidates.
  - (If delete invalidation implemented) deleting an approved gift invalidates; non-approved
    does not.
  - Reuse existing `gifts_test.go` scaffolding for repos/handlers.

  Logging: n/a (test code).

## Commit Plan

- **Commit 1** (after tasks 1-2): `feat(miniapp): cache first-screen gift catalog (backend)`
- **Commit 2** (after tasks 3-4): `feat(miniapp): invalidate gift cache on approval`
- **Commit 3** (after task 5): `test(miniapp): cover approval cache invalidation`

(Single squashed commit is also fine given the small surface; checkpoints listed for clarity.)

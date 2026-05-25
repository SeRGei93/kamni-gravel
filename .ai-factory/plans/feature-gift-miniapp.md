# Implementation Plan: Gift Miniapp

Branch: feature/gift-miniapp
Created: 2026-05-25
Refined: 2026-05-25

## Settings
- Testing: yes
- Logging: standard
- Docs: yes

## Scope

Build a Telegram Mini App for viewing the gift catalog. The first version is a read-focused participant experience:

- Opened from the existing Telegram bot.
- Shows a catalog of all participant-visible gifts for the active event. To preserve the admin review gate, participant-visible means `approved` gifts only.
- Shows a polished gift card with image preview when a gift has a photo.
- Shows distribution conditions for every gift: gender filter, bike type filter, place condition, and criteria.
- Provides filters by gender and bike type so a participant can quickly see gifts relevant to their category.
- Does not create gifts, approve gifts, edit gifts, distribute gifts, or reintroduce admin gift creation.
- Preserves the current Telegram gift creation flow and the admin review gate.

## Current State

- Current branch was created from local `main`; `origin` is not configured, so `git pull origin main` could not run.
- Existing gift admin review work is present in code: `review_status`, pending/approved UI, Telegram confirmation flow, public read gift endpoints, and admin-only update/delete routes.
- Existing public `GET /api/events/{eventId}/gifts` can return all gifts unless `review_status` is provided. The miniapp must not rely on that default because it would expose unapproved gifts in the public catalog.
- Current frontend has no test runner beyond `npm run lint` and `npm run build`.
- `github.com/go-telegram/bot v1.21.0` already provides `ValidateWebappRequest(values, token)` and `models.WebAppInfo`, so backend code can reuse the Telegram SDK for init data validation and web app buttons.
- Telegram Mini Apps documentation says the frontend should send `Telegram.WebApp.initData` to the backend, the backend must validate it before trusting user data, and `auth_date` should be checked to avoid stale data.
- Current CORS middleware allows only `Content-Type, Authorization`; it must allow `X-Telegram-Init-Data` for miniapp API calls.
- The root Next.js layout currently wraps all routes in admin-oriented providers. The miniapp route needs to avoid admin auth/sidebar bootstrap.

## Assumptions

- "Miniapp для просмотра подарков" means a Telegram user-facing Mini App, not another admin dashboard page.
- "Все подарки" means all gifts that participants are allowed to see in the miniapp: approved gifts for the active event.
- Gender and bike filters should use matching semantics, not exact raw field semantics: selecting `male` should show gifts with `gender_filter=all` or `gender_filter=male`; selecting `gravel` should show gifts with `bike_type_filter=all` or `bike_type_filter=gravel`.
- Distribution conditions shown in the miniapp should mirror the existing prize distribution logic: filters first, criteria before place-only gifts, and place interpreted by existing backend rules.
- The active event remains the miniapp event source. A later version can add explicit event selection if needed.

## Commit Plan

- **Commit 1** (after tasks 1-4): `feat: add miniapp gift catalog API`
- **Commit 2** (after tasks 5-8): `feat: add Telegram gift miniapp UI`
- **Commit 3** (after tasks 9-11): `test: cover gift miniapp and docs`

## Tasks

### Phase 1: Backend Miniapp Auth And Catalog API

- [x] 1. Add miniapp configuration and Telegram WebApp launch support.

  Files:
  - `backend/internal/config/main.go`
  - `backend/cmd/bot/main.go`
  - `backend/internal/infrastructure/telegram/bot.go`
  - `backend/internal/infrastructure/telegram/handler/start.go`
  - `backend/internal/infrastructure/telegram/keyboard/builder.go`
  - `backend/internal/infrastructure/telegram/keyboard/builder_test.go`
  - `env.example`
  - `docker-compose.yml`

  Deliverable:
  - Add `MINIAPP_URL` to bot/runtime configuration and pass it through `cmd/bot/main.go` into `telegram.Config`.
  - Add a `web_app` inline keyboard button such as "🎁 Смотреть подарки" in the main Telegram menu when `MINIAPP_URL` is set.
  - Preserve existing callback buttons for registration, gift creation, result submission, and info.
  - Do not block bot startup if `MINIAPP_URL` is empty in local development; omit the miniapp button and log the omission at INFO.
  - Update keyboard tests so they verify both callback buttons and the optional WebApp button without treating WebApp buttons as callback data.

  Logging requirements:
  - INFO log once during bot setup when miniapp URL is configured or omitted.
  - WARN log invalid/malformed miniapp URL configuration.
  - Do not log `BOT_TOKEN`, Telegram init data, or full miniapp URLs with secrets/query params.

- [x] 2. Add Telegram Mini App init data validation and CORS support.

  Files:
  - `backend/internal/infrastructure/http/middleware/telegram_webapp.go`
  - `backend/internal/infrastructure/http/middleware/telegram_webapp_test.go`
  - `backend/internal/infrastructure/http/middleware/cors.go`
  - `backend/internal/infrastructure/http/server.go`
  - `backend/cmd/api/main.go`

  Deliverable:
  - Validate `Telegram.WebApp.initData` from an `X-Telegram-Init-Data` header.
  - Reuse `telegrambot.ValidateWebappRequest` from `github.com/go-telegram/bot`.
  - Copy `url.Values` before validation because the SDK helper removes `hash`.
  - Parse and validate `auth_date` freshness with a conservative default, for example 24 hours.
  - Reject init data from the future beyond a small clock-skew allowance.
  - Reject missing user payloads, bot users, and Telegram user IDs less than or equal to zero.
  - Put the validated Telegram user ID and public profile fields into request context for miniapp handlers.
  - Pass `BOT_TOKEN` into the middleware from `cmd/api/main.go` through HTTP server config.
  - Allow `X-Telegram-Init-Data` in both CORS middleware variants.
  - Return `401` for missing, invalid, expired, or not-yet-valid init data.

  Logging requirements:
  - INFO log successful miniapp auth with `telegram_user_id` and request path.
  - WARN log missing/invalid/expired/future init data with reason and path.
  - Never log raw init data, hash, signature, or bot token.

- [x] 3. Add a miniapp gift catalog query with gender and bike filters.

  Files:
  - `backend/internal/application/query/get_miniapp_gifts.go`
  - `backend/internal/application/query/get_miniapp_gifts_test.go`
  - `backend/internal/domain/repository/gift.go`
  - `backend/internal/infrastructure/persistence/postgres/gift_repo.go`
  - `backend/internal/application/dto/gift.go`

  Deliverable:
  - Add an application query for the active event gift catalog.
  - Always restrict catalog results to `review_status=approved`.
  - Support optional `gender` and `bike_type` filters.
  - Apply participant-friendly matching semantics:
    - `gender=all` returns every approved gift regardless of `gender_filter`;
    - `gender=male` returns gifts where `gender_filter` is `all`, empty, or `male`;
    - `gender=female` returns gifts where `gender_filter` is `all`, empty, or `female`;
    - bike type filter follows the same `all` or exact match rule.
  - Validate filter values using existing domain values/constants where available; return a typed query error for invalid filters.
  - Load criteria and attachments consistently with `GetGiftsHandler`.
  - Return enough DTO data for condition display: gender filter, bike type filter, place, criteria names/types, attachments, donor display fields, and created date.
  - Keep domain entities free of transport tags and keep SQL in the Postgres adapter.

  Logging requirements:
  - Query handler should wrap errors with event ID and normalized filter values.
  - Repository logs only unexpected DB failures at ERROR/WARN equivalent using existing `log.Printf` style.
  - Do not log full gift descriptions or Telegram file IDs unless needed to diagnose a single failed attachment lookup.

- [x] 4. Add miniapp HTTP handlers and routes.

  Files:
  - `backend/internal/infrastructure/http/handler/miniapp.go`
  - `backend/internal/infrastructure/http/handler/miniapp_test.go`
  - `backend/internal/infrastructure/http/server.go`
  - `backend/internal/application/dto/gift.go`

  Deliverable:
  - Add routes under `/api/miniapp`, protected by Telegram init data validation:
    - `GET /api/miniapp/session` returns validated Telegram user info and active event summary.
    - `GET /api/miniapp/gifts?gender=all|male|female&bike_type=all|gravel|mtb|road|single_speed|tandem` returns approved catalog gifts.
    - `GET /api/miniapp/telegram/files/{fileId}` streams Telegram file content or returns a short-lived blob response for attachments.
  - Use the active event from `EventRepository.FindActive`.
  - Treat "no active event" as a stable miniapp state: return `404` with a clear response or a typed empty-state payload, not an unclassified `500`.
  - Return an empty gift list, not an error, when no gifts match the filters.
  - Keep admin gift endpoints unchanged.
  - Avoid returning direct Telegram file download URLs that expose the bot token; miniapp images should be fetched through backend with validated init data.

  Logging requirements:
  - INFO log miniapp session and gift-list requests with Telegram user ID, event ID, filter values, and result count.
  - WARN log invalid filters and missing active event.
  - ERROR/WARN log Telegram file proxy failures with file ID only if necessary; never log generated Telegram download URLs or token-bearing values.

### Phase 2: Frontend Miniapp Experience

- [ ] 5. Add a typed miniapp API client and Telegram WebApp runtime helpers.

  Files:
  - `frontend/src/api/miniapp.ts`
  - `frontend/src/types/telegram-webapp.d.ts`
  - `frontend/src/utils/telegramWebApp.ts`
  - `frontend/src/types/index.ts`

  Deliverable:
  - Read `window.Telegram.WebApp.initData` only on the client.
  - Send init data in `X-Telegram-Init-Data` for all miniapp API calls.
  - Do not reuse the shared `frontend/src/api/client.ts` for miniapp calls, because it injects admin JWT/localStorage and is not designed for protected blob/file responses.
  - Add typed responses for miniapp session, gift catalog, filter params, and attachment blob loading.
  - Expose safe helper methods for `ready()`, `expand()`, theme params, color scheme, and fallback browser mode.
  - Do not store init data in localStorage/sessionStorage.

  Logging requirements:
  - Use `console.warn` for missing Telegram runtime in local browser mode.
  - Use `console.error` for API failures without dumping raw init data.
  - Do not log blob URLs, raw init data, or full gift descriptions.

- [ ] 6. Isolate miniapp routes from admin dashboard providers.

  Files:
  - `frontend/src/app/layout.tsx`
  - `frontend/src/app/(dashboard)/layout.tsx`
  - `frontend/src/app/(auth)/layout.tsx`
  - `frontend/src/app/(miniapp)/layout.tsx`

  Deliverable:
  - Prevent miniapp routes from bootstrapping admin auth/sidebar behavior.
  - Move `AuthProvider` and `SidebarProvider` out of root layout if needed and into dashboard/auth layouts where they are actually required.
  - Keep global font, global CSS, and theme behavior available where needed without coupling miniapp to protected dashboard routing.
  - Verify `/miniapp/gifts` does not render dashboard header/sidebar or trigger `/api/auth/me`.

  Logging requirements:
  - This task should not add runtime logs.
  - If temporary diagnostics are used while verifying provider behavior, remove them before completion.

- [ ] 7. Build the miniapp route and layout.

  Files:
  - `frontend/src/app/(miniapp)/layout.tsx`
  - `frontend/src/app/(miniapp)/miniapp/gifts/page.tsx`
  - `frontend/src/app/(miniapp)/miniapp/gifts/loading.tsx`
  - `frontend/src/app/(miniapp)/miniapp/gifts/error.tsx`

  Deliverable:
  - Add a standalone, mobile-first miniapp page outside the protected dashboard layout.
  - Include Telegram's `telegram-web-app.js` script in the miniapp layout.
  - On load, call `ready()` and `expand()` when Telegram runtime exists.
  - Load miniapp session and show active event name as compact context.
  - Show compact loading, empty, and error states suitable for Telegram's viewport.
  - Do not show admin navigation/sidebar/auth UI.

  Logging requirements:
  - Client logs only lifecycle failures and failed API calls.
  - No raw init data, no file URLs, no full gift descriptions in logs.

- [ ] 8. Build the gift catalog UI with images, conditions, and filters.

  Files:
  - `frontend/src/components/miniapp/GiftCard.tsx`
  - `frontend/src/components/miniapp/GiftImage.tsx`
  - `frontend/src/components/miniapp/GiftFilters.tsx`
  - `frontend/src/components/miniapp/GiftConditionList.tsx`
  - `frontend/src/components/miniapp/GiftEmptyState.tsx`
  - `frontend/src/app/(miniapp)/miniapp/gifts/page.tsx`
  - `frontend/src/constants/options.ts`
  - `frontend/src/utils/criteria.ts`

  Deliverable:
  - Main miniapp screen is a gift catalog, not a "my gifts" dashboard.
  - Show all approved gifts returned by the backend for the active event and selected filters.
  - Add filter controls for gender and bike type using existing option values and labels where possible.
  - Filter changes should update the catalog without full page reload and should keep a stable mobile layout.
  - Render each gift as a polished compact card:
    - image preview first when a photo exists;
    - clear fallback visual when no image exists;
    - description;
    - donor display name when available;
    - distribution conditions section.
  - Distribution conditions section must show:
    - gender: "для всех", "мужчины", or "женщины";
    - bike type: "любой", "Gravel", "MTB", etc.;
    - place condition when set, with wording aligned to current distribution semantics;
    - criteria badges/names when present;
    - fallback text for generic gifts without place/criteria restrictions.
  - Load attachment previews through the miniapp file endpoint using `fetch` and object URLs so no bot-token file URL appears in markup.
  - Use a regular `<img>` for blob object URLs, not `next/image`, and revoke object URLs on unmount or image change.
  - Use Telegram theme variables where available and keep layout readable on narrow mobile screens.

  Logging requirements:
  - Log image fetch failures at `console.warn` with gift ID and attachment ID only.
  - Log filter/catalog data load failures at `console.error` with normalized filter values only.
  - Do not log gift descriptions, raw Telegram init data, blob URLs, or token-bearing URLs.

### Phase 3: Tests, Docs, Verification

- [ ] 9. Add backend tests for miniapp auth, catalog visibility, filters, and handlers.

  Files:
  - `backend/internal/infrastructure/http/middleware/telegram_webapp_test.go`
  - `backend/internal/application/query/get_miniapp_gifts_test.go`
  - `backend/internal/infrastructure/http/handler/miniapp_test.go`
  - `backend/internal/infrastructure/telegram/keyboard/builder_test.go`
  - Existing fakes in related test files may be reused or split into focused helpers.

  Deliverable:
  - Unit-test valid init data, bad hash, missing hash, expired `auth_date`, future `auth_date`, malformed user payload, bot users, and invalid user IDs.
  - Test that catalog results always exclude `pending_review` gifts.
  - Test gender filter semantics: selected gender includes matching and `all` gifts, rejects opposite-gender gifts.
  - Test bike type filter semantics: selected bike type includes matching and `all` gifts, rejects other bike-specific gifts.
  - Test invalid filters and missing active event responses.
  - Test the main menu preserves existing callback buttons and adds the WebApp button only when configured.

  Logging requirements:
  - Tests should assert behavior, not log output.
  - Any test helper that creates init data must use synthetic bot tokens and users only.

- [ ] 10. Update API docs and project docs.

  Files:
  - `backend/docs/swagger.yaml`
  - `README.md`
  - `env.example`

  Deliverable:
  - Document `MINIAPP_URL` and the Docker Compose/runtime expectation for exposing the frontend over HTTPS in Telegram.
  - Document that `NEXT_PUBLIC_API_URL` must also be a public HTTPS URL reachable from the user's Telegram client in non-local environments.
  - Document `ALLOWED_ORIGINS` and the `X-Telegram-Init-Data` CORS/header requirement.
  - Add OpenAPI entries for `/api/miniapp/session`, `/api/miniapp/gifts` with `gender` and `bike_type` query parameters, and the miniapp file endpoint.
  - Document that miniapp calls require `X-Telegram-Init-Data` and that this header must come from `Telegram.WebApp.initData`.
  - Keep docs concise; do not add separate ad hoc Markdown reports.

  Logging requirements:
  - Documentation task has no runtime logs.
  - If examples include headers, use placeholders and never real init data.

- [ ] 11. Run verification and fix regressions.

  Files:
  - No planned source file ownership; fix only files touched by previous tasks unless verification exposes a direct regression.

  Deliverable:
  - Run `cd backend && go test ./...`.
  - Run `cd frontend && npm run lint`.
  - Run `cd frontend && npm run build`.
  - Use Browser/Playwright for a mobile smoke check of `/miniapp/gifts` at about 390px width:
    - route renders without dashboard sidebar/header;
    - loading/error/empty states fit the viewport;
    - gift cards do not overflow;
    - filter controls are usable;
    - image fallback and image slots keep stable dimensions.
  - If Docker/runtime wiring changed substantially, run `make docker-up` and smoke-check:
    - bot starts with `MINIAPP_URL` set;
    - miniapp route renders;
    - backend rejects invalid init data;
    - backend accepts synthetic valid init data in tests.

  Logging requirements:
  - Preserve standard runtime logs.
  - If adding temporary debug logs during implementation, remove them before completion or lower them behind existing debug controls.

## Sources Checked

- Telegram Mini Apps docs: https://core.telegram.org/bots/webapps
- Local Telegram SDK helper: `github.com/go-telegram/bot v1.21.0`, `ValidateWebappRequest`, `models.WebAppInfo`

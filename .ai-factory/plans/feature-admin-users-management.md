# Implementation Plan: Admin Users Management

Branch: feature/admin-users-management
Created: 2026-05-31

## Settings
- Testing: yes
- Logging: minimal
- Docs: yes

## Requirements
- Add an authenticated admin-users list in the dashboard.
- Add a UI button/form for creating a new admin user.
- Allow the current admin to change their own password.
- Keep backend Clean Architecture boundaries: domain repository contracts, application commands/queries, infrastructure HTTP/Postgres adapters.
- Use the existing `admin_users` table; no migration is expected unless implementation discovers a schema gap.
- Keep logs minimal: only validation/security warnings and operation failures, no plaintext passwords, password hashes, JWTs, or sensitive request bodies.
- Do not add admin deletion, role-management UI, or refresh-token revocation in this feature unless implementation discovers a hard blocker.

## Commit Plan
- **Commit 1** (after tasks 1-5): `feat: add admin user management api`
- **Commit 2** (after tasks 6-8): `feat: add admin user dashboard screens`
- **Commit 3** (after tasks 9-10): `docs: document admin user management`

## Tasks

### Phase 1: Backend Domain And Application
- [x] Task 1: Extend the admin repository contract and PostgreSQL adapter.
  - Deliverable: `repository.AdminRepository` supports listing admins and updating only password hashes without changing username/role.
  - Expected behavior:
    - Add sentinel errors to `backend/internal/domain/repository/admin.go`, at minimum `ErrAdminNotFound` and `ErrAdminUsernameTaken`.
    - Add `List(ctx context.Context) ([]*entity.Admin, error)` to `backend/internal/domain/repository/admin.go`.
    - Add `UpdatePassword(ctx context.Context, id uint, passwordHash string) error` to the same interface.
    - Implement both methods in `backend/internal/infrastructure/persistence/postgres/admin_repo.go`.
    - Map `sql.ErrNoRows` from `FindByID`/`FindByUsername` to `repository.ErrAdminNotFound`.
    - Map PostgreSQL unique violation `23505` on username insert/update to `repository.ErrAdminUsernameTaken`; do not parse error strings.
    - Keep `FindByID` and `UpdatePassword` distinguishable enough for application code to map missing current admins to not-found behavior.
    - Return admins ordered predictably, preferably by `created_at DESC, id DESC`.
    - Keep `context.Context` first and keep native SQL.
  - Logging requirements:
    - Do not log passwords or password hashes.
    - Repository methods should not add noisy success logs; return wrapped errors to callers where needed.
    - Log only at higher layers for failed list/update operations.
  - Files:
    - `backend/internal/domain/repository/admin.go`
    - `backend/internal/infrastructure/persistence/postgres/admin_repo.go`
    - affected test fakes in `backend/internal/infrastructure/http/handler/*_test.go`

- [x] Task 2: Add admin DTOs and application query/commands.
  - Deliverable: application layer exposes list, create, and change-own-password operations for admin users.
  - Depends on: Task 1.
  - Expected behavior:
    - Add `backend/internal/application/dto/admin.go` with `AdminDTO` and `AdminListResponse` containing `id`, `username`, `role`, `created_at`, and nullable `last_login`.
    - Add `backend/internal/application/query/admin_users.go` with `ListAdminUsersHandler`.
    - Add `backend/internal/application/command/admin_users.go` with:
      - `CreateAdminHandler` for creating username/password admin records.
      - `ChangeAdminPasswordHandler` for current-password verification and new-password update.
    - Define a small application-owned password service interface in the command package, for example `Hash(password string) (string, error)` and `Compare(hash, password string) error`, so commands can hash/compare passwords without importing infrastructure packages.
    - Validate usernames and passwords centrally: trimmed username required, password required, sensible minimum length, duplicate username mapped from `repository.ErrAdminUsernameTaken` to a command-level user-facing error.
    - Map `repository.ErrAdminNotFound` to command/query-level errors where needed instead of leaking infrastructure messages.
    - Preserve current role model but create new dashboard admins as `entity.AdminRoleAdmin` unless implementation intentionally exposes role selection.
  - Logging requirements:
    - Log `WARN` for validation failures without password values.
    - Log `WARN` for invalid current password attempts with `admin_id` only.
    - Log `ERROR` for repository/hash failures with operation name and `admin_id` or username only.
  - Files:
    - `backend/internal/application/dto/admin.go`
    - `backend/internal/application/query/admin_users.go`
    - `backend/internal/application/command/admin_users.go`

### Phase 2: Backend HTTP Integration
- [x] Task 3: Add bcrypt password service adapter and wire it into the server.
  - Deliverable: infrastructure provides the concrete implementation of the application password service.
  - Depends on: Task 2.
  - Expected behavior:
    - Add a small infrastructure adapter, for example `backend/internal/infrastructure/security/password_hasher.go`, that wraps `golang.org/x/crypto/bcrypt`.
    - Adapter methods must satisfy the application-owned password service interface from Task 2.
    - Wire the adapter in `backend/internal/infrastructure/http/server.go` when constructing `CreateAdminHandler` and `ChangeAdminPasswordHandler`.
    - Remove or replace duplicate helper behavior from `backend/internal/infrastructure/http/handler/auth.go`, especially `HashPassword` and `CreateAdmin`, if those helpers become unused after command-based admin creation.
    - Keep bcrypt implementation in infrastructure; application/domain must not import `golang.org/x/crypto/bcrypt`.
  - Logging requirements:
    - Do not log plaintext passwords, hashes, or bcrypt compare inputs.
    - Adapter should return errors to command handlers; do not add success logs.
  - Files:
    - `backend/internal/infrastructure/security/password_hasher.go`
    - `backend/internal/infrastructure/http/server.go`
    - `backend/internal/infrastructure/http/handler/auth.go`

- [x] Task 4: Add HTTP handlers and routes for admin management.
  - Deliverable: protected REST endpoints for listing admins, creating an admin, and changing the current admin password.
  - Depends on: Tasks 1, 2, and 3.
  - Expected behavior:
    - Add an admin-users HTTP handler, or extend auth handling cleanly if that better matches the implementation.
    - Wire `ListAdminUsersHandler`, `CreateAdminHandler`, `ChangeAdminPasswordHandler`, and any new `AdminUsersHandler` fields/constructors in `backend/internal/infrastructure/http/server.go`.
    - Add routes under the existing authenticated admin group in `backend/internal/infrastructure/http/server.go`:
      - `GET /api/admin-users`
      - `POST /api/admin-users`
      - `PUT /api/auth/me/password`
    - `POST /api/admin-users` accepts `username` and `password`, returns created admin DTO, and never returns password/hash.
    - `PUT /api/auth/me/password` reads the authenticated admin only via `middleware.GetUserFromContext(r.Context())`, requires `current_password` and `new_password`, and returns no password/hash.
    - Map validation errors to `400`, current-password mismatch to `401`, duplicate usernames to `409` using existing `response.Conflict`, not-found to `404`, and unexpected errors to `500`.
    - Keep `AuthHandler.Me` using `middleware.GetUserFromContext`; do not reintroduce raw context-key access.
  - Logging requirements:
    - Log decode/validation failures at `WARN` with operation name only.
    - Log auth-context missing and current-password mismatch at `WARN`, without secrets.
    - Log unexpected command/query failures at `ERROR`.
  - Files:
    - `backend/internal/infrastructure/http/handler/admin_users.go`
    - `backend/internal/infrastructure/http/handler/auth.go`
    - `backend/internal/infrastructure/http/server.go`

- [x] Task 5: Add focused backend tests.
  - Deliverable: Go tests cover the new application behavior and HTTP routing behavior.
  - Depends on: Tasks 1, 2, 3, and 4.
  - Expected behavior:
    - Add application tests for list, create validation, duplicate username handling, password hashing dependency use, current-password mismatch, and successful own-password change.
    - Add HTTP handler tests for authenticated list, create, and password change responses.
    - Add a middleware-chain regression test for `PUT /api/auth/me/password` that proves JWT claims flow through `middleware.Auth` and `middleware.RequireRole` into the handler.
    - Update existing fakes after `AdminRepository` interface changes.
    - Keep tests deterministic; use fakes for application/handler tests instead of requiring PostgreSQL.
    - Test status mappings for duplicate username (`409`), current password mismatch (`401`), missing auth context (`401`), and successful password change.
  - Logging requirements:
    - Tests should assert behavior, not exact log text.
    - Do not print password values in test failure messages.
  - Files:
    - `backend/internal/application/query/admin_users_test.go`
    - `backend/internal/application/command/admin_users_test.go`
    - `backend/internal/infrastructure/http/handler/admin_users_test.go`
    - `backend/internal/infrastructure/http/handler/auth_test.go`

### Phase 3: Frontend API And Dashboard UI
- [x] Task 6: Add typed frontend API contracts.
  - Deliverable: frontend has typed clients for admin list/create and password change.
  - Depends on: Task 4 final API request/response contracts.
  - Expected behavior:
    - Add `AdminUser`, `AdminUserListResponse`, `CreateAdminRequest`, and `ChangePasswordRequest` types in `frontend/src/types/index.ts`.
    - Add `frontend/src/api/adminUsers.ts` with `getAll()` and `create()`.
    - Extend `frontend/src/api/auth.ts` with `changePassword()`.
    - Keep all API calls through `frontend/src/api/client.ts`.
  - Logging requirements:
    - No frontend logs for successful password changes.
    - Log failed API operations with operation names only, never password fields.
  - Files:
    - `frontend/src/types/index.ts`
    - `frontend/src/api/adminUsers.ts`
    - `frontend/src/api/auth.ts`

- [x] Task 7: Add admin-users dashboard page and sidebar entry.
  - Deliverable: dashboard page lists admin users and provides an add-admin button/form.
  - Depends on: Task 6.
  - Expected behavior:
    - Create `frontend/src/app/(dashboard)/admin-users/page.tsx`.
    - Mark the page with `'use client'`, matching existing interactive dashboard pages.
    - Show table columns: username, role, created date, last login.
    - Add a visible `Добавить админа` button that opens an inline form or modal using existing TailAdmin components.
    - Reuse existing UI pieces such as `Button`, `Input`, `Label`, and `Table`; avoid introducing a new component library.
    - Form fields: username, password, confirm password; validate required fields and matching passwords client-side.
    - On successful create, clear form and reload/append the list.
    - Add sidebar navigation item in `frontend/src/layout/AppSidebar.tsx`, preferably named `Администраторы`, using an existing icon such as `UserIcon` or `UserCircleIcon`.
  - Logging requirements:
    - Log load/create failures in `console.error` with operation and username only.
    - Never log entered passwords.
  - Files:
    - `frontend/src/app/(dashboard)/admin-users/page.tsx`
    - `frontend/src/layout/AppSidebar.tsx`

- [x] Task 8: Add current-admin password change UI.
  - Deliverable: current admin can open a password-change screen from the user dropdown and submit current/new password.
  - Depends on: Task 6.
  - Expected behavior:
    - Create `frontend/src/app/(dashboard)/account/password/page.tsx`.
    - Mark the page with `'use client'`.
    - Add a `Сменить пароль` item to `frontend/src/components/header/UserDropdown.tsx`.
    - Use existing dropdown/menu components where possible; if using a `Link`, close the dropdown before navigation.
    - Form fields: current password, new password, confirm new password.
    - Validate required fields, minimum password length, and confirmation match before calling the API.
    - After successful password change, clear local auth tokens and redirect to `/login`, or show a clear success state before requiring re-login.
  - Logging requirements:
    - Log failed password-change requests with operation name only.
    - Never log current or new passwords.
  - Files:
    - `frontend/src/app/(dashboard)/account/password/page.tsx`
    - `frontend/src/components/header/UserDropdown.tsx`
    - `frontend/src/context/AuthContext.tsx` if a shared logout-after-password-change helper is needed.

### Phase 4: Documentation And Verification
- [x] Task 9: Update API and user-facing documentation.
  - Deliverable: docs mention the new admin-user endpoints and password-change flow.
  - Depends on: Task 4 final API request/response contracts and Task 8 final UI behavior.
  - Expected behavior:
    - Update `backend/docs/swagger.yaml` with:
      - `PUT /auth/me/password`; do not add `GET /auth/me/password`.
      - `GET /admin-users`
      - `POST /admin-users`
      - request/response schemas for admin DTOs and password-change/create-admin requests.
      - `409 Conflict` response for duplicate admin username.
    - Update `README.md` only if it currently documents admin login/admin dashboard workflows; keep the change concise.
  - Logging requirements:
    - No runtime logging changes in docs task.
  - Files:
    - `backend/docs/swagger.yaml`
    - `README.md`

- [x] Task 10: Run verification and fix regressions.
  - Deliverable: backend and frontend checks pass, or failures are documented with concrete blockers.
  - Depends on: Tasks 1 through 9.
  - Expected behavior:
    - Run `cd backend && go test ./...`.
    - Run `cd frontend && npm run lint`.
    - Run `cd frontend && npm run build`.
    - Run `git diff --check`.
    - Run a targeted grep over changed files for accidental password/JWT/hash logging before finishing.
    - If frontend build needs environment variables, use current project conventions and note any missing env values.
  - Logging requirements:
    - During verification, grep changed files for accidental password logging before finishing.
    - Do not add new logs that include plaintext passwords, password hashes, JWTs, or full auth request bodies.
  - Files:
    - verification only; no planned source file unless a check reveals a defect.

# Implementation Plan: Create Criterion From Gift Edit Page

Branch: feature/gift-edit-create-criterion
Base: main
Created: 2026-06-16

## Settings
- Testing: yes
- Logging: minimal
- Docs: no

## Roadmap Linkage
Milestone: "none"
Rationale: No .ai-factory/ROADMAP.md exists in the project.

## Context
The gift edit page (`frontend/src/app/(dashboard)/gifts/[id]/page.tsx`) loads all criteria via
`criteriaApi.getAll()` and passes them to `GiftEditForm`, which renders them as selectable badges.
Today an admin can only attach **existing** criteria; to add a new one they must leave for the
`/criteria` page. Both the `POST /api/criteria` endpoint (`criteriaApi.create`) and a reusable
`CriteriaForm` component (already used in a `Modal` on the `/criteria` page) exist, so this feature
is **frontend-only**: surface the same create form in a modal on the gift edit page and auto-select
the newly created criterion.

Design decisions (reasonable defaults, no backend changes):
- Reuse the existing `CriteriaForm` + `Modal` + `useModal` rather than building a new form.
- After creation, auto-select the new criterion for the gift (admin creates it in order to attach it).
- The page owns the `criteria` list; `GiftEditForm` owns `selectedCriteriaIds`. The page exposes an
  `onCreateCriteria` callback that creates + merges into its list and returns the new criterion;
  `GiftEditForm` auto-selects the returned id.

## Tasks

### Phase 1: Data + Helpers
- [x] Task 7: Add the criteria-create handler and pure selection helpers.
  - Deliverable: in `frontend/src/app/(dashboard)/gifts/[id]/page.tsx` add
    `handleCreateCriteria(data: CreateCriteriaRequest): Promise<Criteria>` that calls
    `criteriaApi.create(data)`, merges the created criterion into the page `criteria` state
    (dedupe by id), and returns it; pass it to `GiftEditForm` as `onCreateCriteria`.
    Add pure helpers to `frontend/src/utils/criteria.ts`:
    `mergeCriterion(list: Criteria[], criterion: Criteria): Criteria[]` (append, or replace by id)
    and `addSelectedCriterionId(ids: number[], id: number): number[]` (add only if absent).
  - Expected behavior: creating a criterion makes it appear in the gift edit criteria list without a
    full reload; calling create twice for the same id never duplicates it.
  - Files: `frontend/src/app/(dashboard)/gifts/[id]/page.tsx`, `frontend/src/utils/criteria.ts`.
  - Logging requirements: minimal — on failure
    `console.error('Failed to create criteria:', { operation: 'create_criteria', error: err })`
    and re-throw so the modal can show the error. No logs inside the pure helpers.
  - Dependency notes: foundation for Task 8; keep helpers pure so Vitest can cover them.

### Phase 2: UI
- [x] Task 8: Add an inline "Создать критерий" modal to `GiftEditForm`.
  - Deliverable: add optional prop `onCreateCriteria?: (data: CreateCriteriaRequest) => Promise<Criteria>`
    to `frontend/src/components/gifts/GiftEditForm.tsx`. Near the "Критерии" section add a
    "Создать критерий" button that opens a `Modal` (via `useModal`) containing the existing
    `CriteriaForm`. On submit: call `onCreateCriteria(data)`, auto-select the returned criterion with
    `addSelectedCriterionId`, then close the modal. Track a create loading/error state and reset the
    `CriteriaForm` by unmounting it (conditional render / `key`) when the modal closes.
  - Expected behavior: the admin creates a criterion without leaving the gift edit page; the new
    criterion appears as a selected badge immediately; the button is hidden/disabled when
    `onCreateCriteria` is not provided; existing criteria selection keeps working.
  - Files: `frontend/src/components/gifts/GiftEditForm.tsx`.
  - Logging requirements: minimal — surface create errors inside the modal; the page handler from
    Task 7 already logs the failure. No new runtime logs for opening/closing the modal.
  - Dependency notes: depends on Task 7 (uses the `onCreateCriteria` contract and
    `addSelectedCriterionId`). Reuse `Modal`, `useModal`, `CriteriaForm`, `CRITERIA_TYPE_OPTIONS` —
    do not duplicate the criteria form fields.

### Phase 3: Tests + Validation
- [x] Task 9: Cover the helpers with Vitest and run frontend validation.
  - Deliverable: in `frontend/src/utils/criteria.test.ts` test `mergeCriterion` (appends a new
    criterion, no duplicate when the id already exists, replaces updated fields by id) and
    `addSelectedCriterionId` (adds an id when absent, no duplicate when present).
  - Expected behavior: helpers prove dedupe-by-id and idempotent selection without rendering React.
  - Files: `frontend/src/utils/criteria.test.ts`.
  - Logging requirements: none.
  - Dependency notes: depends on Tasks 7-8; validation commands are
    `cd frontend && npm run test -- --run`, `cd frontend && npm run lint`,
    and `cd frontend && npm run build`.

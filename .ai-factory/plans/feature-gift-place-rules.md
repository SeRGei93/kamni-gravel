# Implementation Plan: Gift Place Rules And Slot-Based Prize Distribution

Branch: feature/gift-place-rules
Created: 2026-05-28

## Goal

Replace the current "one gift - one place" behavior with a rule-based prize placement model. A single approved gift must be able to target multiple places, ranges, and "last N" positions inside any selected eligible group, while the admin distribution table still shows the result as separate prize assignments for concrete participants.

## Settings

- Testing: yes, TDD is mandatory. Write failing tests before implementation in each phase.
- Logging: verbose during implementation, safe DEBUG/INFO around new assignment decisions, no private descriptions or Telegram file IDs in logs.
- Docs: yes, update Swagger/API docs and user-facing admin labels for changed gift place contracts.

## Requirements

- Keep Telegram gift creation as-is: users submit gifts through Telegram, moderators configure review status, criteria, filters, and place rules in the admin dashboard.
- Do not duplicate full `gifts` rows for multi-place prizes. One gift keeps one description, owner, photos, criteria set, review status, and filters.
- Add place rules for gifts:
  - no place rule;
  - explicit places and ranges, for example `1`, `1, 3, 10-15`;
  - `last_n`, for example "5 last places".
- Do not build a visible 1-200 button grid as the primary UI. Use compact inputs and previews because around 200 participants are expected.
- Eligible group is computed before place matching:
  - `review_status = approved`;
  - gift `gender_filter`;
  - gift `bike_type_filter`;
  - all criteria attached to the gift.
- Criteria are more important than place. Therefore place/range/last rules apply inside the already-filtered eligible group. Example: "5 last female gravel with criterion X" means last five among female gravel finishers who have criterion X.
- Support all group scopes through existing filters:
  - `all + all`: event absolute;
  - `male/female + all`: gender group;
  - `all + gravel/mtb/...`: bike-type absolute;
  - `male/female + gravel/mtb/...`: gender plus bike group.
- For explicit places/ranges, try exact rank first. If exact rank has no eligible/free result, assign the nearest free eligible result inside the same group.
- Fallback must never leave the eligible group and must never assign the same gift twice to the same participant.
- Fallback tie rule: if two free ranks are equally close, prefer the worse place, meaning the larger rank, then the better place. This is a product assumption to confirm before implementation if needed.
- `last_n` uses the actual eligible group size after results are known. If fewer eligible results exist than `N`, assign all eligible results.
- A gift without a place rule keeps the current capacity behavior: it can be assigned once, to the first eligible participant who is not blocked by a higher-priority assignment.
- Preserve current priority behavior unless tests show an intentional product change:
  - criteria plus place rule;
  - criteria only;
  - place rule only;
  - generic gift.
- A participant should not receive lower-priority gifts when higher-priority gifts match, but may still receive multiple same-priority gifts, matching the current behavior.
- Distribution output must allow the same `gift.id` to appear for several participants when the gift has multiple assigned slots.
- The gifts list and miniapp catalog must still show the source gift once, with a readable rule summary.
- Existing `gifts.place` values must be migrated/backfilled into the new rule model as a single explicit place.
- Keep existing Clean Architecture boundaries: domain/value objects do validation, application query owns distribution logic, infrastructure owns SQL and HTTP/Telegram adapters.

## Current Findings

- `Gift` currently stores a single `Place *int` in `backend/internal/domain/entity/gift.go`.
- The database currently stores a single nullable `gifts.place INTEGER` in `backend/internal/infrastructure/migrations/00005_create_gifts.sql`.
- Admin updates currently decode `place` with explicit null handling in `backend/internal/infrastructure/http/handler/gifts.go`.
- Distribution is implemented in `backend/internal/application/query/get_prize_distribution.go`.
- Distribution currently iterates result rows, finds matching gifts, and marks `gift.ID` as used. This blocks one source gift from being assigned to multiple places.
- Current priority is already documented in code: criteria are above place-only and generic gifts.
- `FindByEventWithPlaces` currently returns absolute, gender, and gender plus bike ranks from SQL, but the new rule must rank dynamically inside the gift's eligible group after criteria filtering.
- `PrizeDistributionDTO` currently exposes only participant-level `matched_gifts` and one row-level `match_reason`; slot-based output needs per-gift assignment metadata while preserving `matched_gifts` compatibility during transition.
- Participant detail and gift list assigned-state code read `matched_gifts`; stats currently count participants with prizes, not total prize assignments.
- Frontend has no unit test runner today. Backend has Go tests and `go-sqlmock` available.

## Data Model Direction

Use a normalized place-rule model instead of duplicating gifts:

```text
gifts
  id
  ...
  place        -- legacy/backfill source, keep temporarily for compatibility

gift_place_rules
  gift_id      -- primary key, references gifts(id)
  rule_type    -- places | last_n
  last_count   -- only for last_n

gift_place_rule_places
  gift_id
  place
  primary key (gift_id, place)
```

Domain model:

```text
GiftPlaceRule
  Type: none | places | last_n
  Places: []int       -- normalized sorted unique places for exact/range input
  LastCount: *int     -- positive integer for last_n
```

DTO/API model:

```json
{
  "place_rule": {
    "type": "places",
    "places": [10, 11, 12, 13, 14, 15]
  }
}
```

Legacy transition:

- Existing `place` in API may be accepted as a single-place rule during the transition.
- `place_rule: null` clears the place rule.
- Omitted `place_rule` preserves the existing rule.
- No row in `gift_place_rules` means "no place rule"; do not persist a `none` row.
- `GiftDTO.place` may remain temporarily as the first explicit place for older UI consumers, but new UI must use `place_rule`.
- Prize distribution responses should expose new `matched_gift_assignments` with assignment metadata and keep legacy `matched_gifts` populated from those assignments until all frontend consumers are migrated.

## Commit Plan

- TDD red tests from Task 2 are temporary working-tree state. Commit only green checkpoints unless the user explicitly asks to preserve a red test commit.
- **Commit 1** (after task 1): `test(gifts): characterize current prize distribution rules`
- **Commit 2** (after tasks 4-6): `feat(gifts): add gift place rule model`
- **Commit 3** (after tasks 2 and 7-11): `feat(distribution): assign prize slots by place rules`
- **Commit 4** (after tasks 12-14): `feat(gifts): add admin place rule editing`
- **Commit 5** (after tasks 15-16): `test(distribution): verify slot assignment edge cases`

## Tasks

### Phase 1: TDD Baseline And Characterization

- [x] 1. Add characterization tests for the current distribution behavior before changing implementation.
  - Files:
    - `backend/internal/application/query/get_prize_distribution_test.go`
  - Deliverable:
    - Cover approved-only distribution.
    - Cover criteria before place-only and generic gifts.
    - Cover criteria plus place as a tie-breaker above criteria-only.
    - Cover same-priority behavior: participant can receive multiple same-priority gifts.
    - Cover lower-priority behavior: participant with higher-priority gifts does not receive lower-priority gifts in the same pass.
    - Cover no-place gift capacity: a criteria-only or generic gift is assigned once and can cascade to the next eligible participant if a faster participant has a higher-priority match.
    - Cover current `usedGifts` limitation as a characterization test, then replace that assertion in later tasks with slot-based behavior.
  - TDD requirement:
    - These tests should pass before feature changes, except tests explicitly marked as new desired behavior in later tasks.
  - Logging requirements:
    - No new runtime logs. Test failures must include gift IDs, participant IDs, priority, and expected assignment shape.
  - Dependencies: none.

- [x] 2. Add failing tests for the new product rules without implementing them yet.
  - Files:
    - `backend/internal/application/query/get_prize_distribution_test.go`
    - optional new file `backend/internal/application/query/prize_distribution_engine_test.go`
  - Deliverable:
    - Tests must describe the desired behavior for:
      - one gift assigned to explicit places `10-15` as multiple slots;
      - `last_n` inside female gravel group;
      - criteria applied before rank calculation;
      - bike-only absolute group such as `all + gravel`;
      - nearest fallback when requested place is outside group size;
      - nearest fallback when exact participant is already used by the same gift;
      - fallback staying inside criteria/gender/bike group;
      - no duplicate assignment of the same gift to the same participant;
      - same source gift appearing in several participant rows;
      - assignment metadata exposing target rank, assigned rank, and fallback flag.
  - TDD requirement:
    - Commit the failing tests only after confirming they fail for the expected reason.
  - Logging requirements:
    - No runtime logs. Add test helper output with compact assignment tables when assertions fail.
  - Dependencies: Task 1.

- [x] 3. Add a test matrix comment/table near the distribution tests so future changes know the covered cases.
  - Files:
    - `backend/internal/application/query/get_prize_distribution_test.go`
  - Deliverable:
    - Include the matrix for rule type, group scope, criteria, fallback, priority, and capacity.
    - Do not create ad hoc Markdown reports.
  - Logging requirements:
    - None.
  - Dependencies: Task 2.

### Phase 2: Gift Place Rule Domain Model And Persistence

- [x] 4. Add a `GiftPlaceRule` value object with exhaustive unit tests.
  - Files:
    - `backend/internal/domain/valueobject/gift_place_rule.go`
    - `backend/internal/domain/valueobject/gift_place_rule_test.go`
    - `backend/internal/domain/entity/gift.go`
  - Deliverable:
    - Add constructors/validators for `none`, `places`, and `last_n`.
    - Add parser/normalizer for admin input `1, 3, 10-15` if parsing is kept backend-side.
    - Normalize explicit places to sorted unique positive integers.
    - Reject zero, negative values, non-numeric tokens, reversed ranges, empty ranges, and `last_n <= 0`.
    - Allow large positive place numbers; do not hardcode 200 in the backend.
    - Add `Gift.PlaceRule` while keeping legacy `Gift.Place` until migration compatibility is complete.
    - Add helpers such as `HasPlaceRule`, `HasPlaceConstraint`, and `FirstLegacyPlace` only if they simplify DTO compatibility without leaking persistence concerns.
  - TDD requirement:
    - Write value object tests first, including table-driven invalid cases.
  - Logging requirements:
    - No logging in pure domain/valueobject code.
  - Dependencies: Tasks 1-3.

- [x] 5. Add migrations and repository support for gift place rules.
  - Files:
    - `backend/internal/infrastructure/migrations/00018_create_gift_place_rules.sql`
    - `backend/internal/domain/repository/gift.go`
    - `backend/internal/infrastructure/persistence/postgres/gift_repo.go`
    - `backend/internal/infrastructure/persistence/postgres/gift_repo_test.go`
  - Deliverable:
    - Create `gift_place_rules` and `gift_place_rule_places`.
    - Add `CHECK` constraints for allowed `rule_type`, positive `last_count`, and positive explicit places.
    - Add `ON DELETE CASCADE` foreign keys and useful indexes for `gift_id`.
    - Include a down migration that drops the new tables without touching legacy `gifts.place`.
    - Backfill existing `gifts.place` rows as `places` rules with one place.
    - Treat absence of a rule row as "none"; clearing a rule deletes both the parent row and explicit place rows.
    - Load place rules in `FindByID`, `FindByEvent`, `FindByEventAndReviewStatus`, and `FindByUser`.
    - Prefer a small repository helper that batch-loads place rules for gift slices, so event gift lists do not introduce one extra query per gift.
    - Update `UpdateWithCriteria` or add an equivalent transaction method so gift fields, criteria, and place rule are replaced atomically.
    - New Telegram-created gifts must default to no place rule.
    - Preserve old `gifts.place` enough for transition, but source new behavior from `Gift.PlaceRule`.
  - TDD requirement:
    - Add `go-sqlmock` repository tests for loading no rule, loading places, loading `last_n`, clearing rules, replacing places, replacing criteria plus rule in one transaction, and rollback on rule insert failure.
  - Logging requirements:
    - Log repository transaction failures at ERROR with `gift_id`, operation stage, and safe rule metadata only.
    - Do not log full gift descriptions or Telegram file IDs.
  - Dependencies: Task 4.

- [x] 6. Extend application update command and HTTP decoding for `place_rule`.
  - Files:
    - `backend/internal/application/command/update_gift.go`
    - `backend/internal/application/command/update_gift_test.go`
    - `backend/internal/infrastructure/http/handler/gifts.go`
    - `backend/internal/infrastructure/http/handler/gifts_test.go`
    - `backend/internal/application/dto/gift.go`
    - `frontend/src/types/index.ts`
  - Deliverable:
    - Add `PlaceRule` and `PlaceRuleSet` semantics similar to current `Place` and `PlaceSet`.
    - `place_rule` omitted preserves current rule.
    - `place_rule: null` clears current rule.
    - Structured `places` rule accepts normalized explicit places.
    - Structured `last_n` rule accepts positive count.
    - Legacy `place` is mapped to a single explicit place when `place_rule` is not present.
    - If both `place_rule` and legacy `place` are present, `place_rule` wins and legacy `place` is ignored.
    - DTO exposes `place_rule` and keeps legacy `place` temporarily for compatibility.
  - TDD requirement:
    - Tests first for omitted/null/places/last_n/invalid/legacy place behavior and `place_rule` winning over legacy `place`.
  - Logging requirements:
    - Log invalid rule payloads at WARN with `gift_id`, `rule_type`, and safe reason.
    - Log update failures with `gift_id` and stage, no descriptions.
  - Dependencies: Task 5.

### Phase 3: Slot-Based Distribution Engine

- [x] 7. Extract pure participant context and eligible-group helpers.
  - Files:
    - `backend/internal/application/query/prize_distribution_engine.go`
    - `backend/internal/application/query/prize_distribution_engine_test.go`
    - `backend/internal/application/query/get_prize_distribution.go`
  - Deliverable:
    - Build participant/result contexts from `ResultWithPlace` plus participant map.
    - Sort contexts deterministically by elapsed time, then result ID, then participant ID.
    - Handle missing participant user data safely when building display names; do not panic if `participant.User` is nil in tests or partial repository results.
    - Filter eligible contexts by gift gender filter, bike type filter, and all gift criteria.
    - Compute rank inside the eligible group after criteria filtering.
    - Support scopes `all/all`, `gender/all`, `all/bike`, and `gender/bike` through the same filter path.
    - Keep existing SQL-calculated ranks only as display fields; dynamic group rank must be computed in application code from eligible contexts.
  - TDD requirement:
    - Tests first for every group scope and criteria-before-rank behavior.
  - Logging requirements:
    - Keep helper pure and unlogged. If debugging is needed, add optional DEBUG logs only in the caller around group sizes.
  - Dependencies: Task 6.

- [x] 8. Implement place slot expansion with exact, range/list, and `last_n` rules.
  - Files:
    - `backend/internal/application/query/prize_distribution_engine.go`
    - `backend/internal/application/query/prize_distribution_engine_test.go`
  - Deliverable:
    - Convert explicit places into deterministic gift slots.
    - Convert ranges into explicit place slots through the normalized value object.
    - Convert `last_n` into target ranks based on actual eligible group size.
    - For `last_n` with fewer eligible participants than requested, assign all eligible participants.
    - `last_n` slots should map directly to concrete eligible participants from the end of the group and should not run the nearest fallback algorithm.
    - Preserve stable slot order for deterministic output.
  - TDD requirement:
    - Tests first for single place, list, range, duplicate range input, `last_n`, `last_n` larger than group, and empty eligible group.
  - Logging requirements:
    - Optional DEBUG logs in the public query handler with `gift_id`, rule type, eligible count, requested slots, assigned slots.
    - Do not log gift descriptions.
  - Dependencies: Task 7.

- [x] 9. Add nearest-free fallback and slot capacity rules.
  - Files:
    - `backend/internal/application/query/prize_distribution_engine.go`
    - `backend/internal/application/query/prize_distribution_engine_test.go`
  - Deliverable:
    - Exact rank wins when available.
    - If target rank is missing or already unavailable for this gift, search outward by distance inside the same eligible group.
    - Tie rule: prefer larger rank first, then smaller rank.
    - Do not assign the same gift to the same participant twice.
    - If no eligible free result exists, leave the slot unassigned and expose enough debug information for troubleshooting.
  - TDD requirement:
    - Tests first for target beyond group end, occupied target, two-sided tie, no free candidate, and criteria/filter isolation.
  - Logging requirements:
    - DEBUG log fallback decisions with `gift_id`, target rank, selected rank, selected participant ID, and reason.
    - INFO log only aggregate unassigned slot counts per event to avoid noisy production logs.
  - Dependencies: Task 8.

- [x] 10. Integrate slot assignment into `GetPrizeDistributionHandler`.
  - Files:
    - `backend/internal/application/query/get_prize_distribution.go`
    - `backend/internal/application/query/get_prize_distribution_test.go`
    - `backend/internal/application/dto/prize_distribution.go`
    - `backend/internal/infrastructure/http/handler/prize_distribution.go`
  - Deliverable:
    - Replace `usedGifts map[uint]bool` with capacity-aware assignment:
      - source gifts without a place rule have capacity one;
      - source gifts with explicit slots have capacity one per slot;
      - used key for slot rules is the slot, not only `gift.ID`.
    - Add a per-assignment result type, for example `PrizeGiftAssignment`, containing gift, match reason, rule type, target rank, assigned rank, fallback flag, and fallback reason.
    - Add `PlaceByGenderBike` to `PrizeDistributionResult` and `PrizeDistributionDTO` so existing calculated rank is not dropped in the distribution API.
    - Preserve approved-only gating.
    - Preserve priority behavior across `criteria+place`, `criteria`, `place`, and generic gifts.
    - Lower-priority gifts must not be bundled when a participant has higher-priority assignments.
    - Same-priority multiple gifts remain allowed.
    - Distribution can return the same source gift ID for multiple participants when different slots are assigned.
    - Keep legacy `MatchedGifts` populated from assignment gifts until participant detail, gifts page, and frontend tables consume assignment metadata.
    - Track unassigned slots in the application result and expose them through the HTTP DTO as `unassigned_slots` so admins can see rule slots that could not be assigned even after fallback.
  - TDD requirement:
    - Existing characterization tests must be updated only where product behavior intentionally changes.
    - New tests must cover mixed exact slots, `last_n`, fallback, lower-priority blocking, same-priority multiple gifts, and deterministic ordering.
  - Logging requirements:
    - INFO log final distribution summary per event: participants, approved gifts, assigned slots, unassigned slots.
    - DEBUG log per-gift assignment details behind normal log-level control.
    - No descriptions, Telegram file IDs, raw private text, or result URLs.
  - Dependencies: Task 9.

- [x] 11. Update dependent distribution consumers for slot assignment counts and metadata.
  - Files:
    - `backend/internal/application/query/get_stats.go`
    - `backend/internal/application/dto/stats.go`
    - `backend/internal/application/dto/participant.go`
    - `backend/internal/infrastructure/http/handler/participants.go`
    - `frontend/src/types/index.ts`
    - `frontend/src/app/(dashboard)/gifts/page.tsx`
    - `frontend/src/app/(dashboard)/participants/[id]/page.tsx`
    - `frontend/src/app/(dashboard)/page.tsx`
  - Deliverable:
    - Count total prize assignments separately from participants-with-prizes where the dashboard/stat DTO needs that distinction.
    - Keep `prizes_assigned_count` behavior explicit: either migrate it to assignment count with updated labels, or add a separate `participants_with_prizes_count`.
    - Participant detail should expose and render assignment metadata while keeping `matched_gifts` compatibility if needed.
    - The gifts admin page's `assignedGiftIds` should still mark a source gift as distributed if any assignment uses it, even if it appears in several slots.
    - Frontend types must model both legacy `matched_gifts` and new assignment metadata during transition.
  - TDD requirement:
    - Add backend tests for stats assignment count vs participant count and participant detail DTO mapping.
    - Add frontend helper/type-level coverage where practical through Task 12 helper tests; otherwise include these surfaces in browser smoke.
  - Logging requirements:
    - Keep existing concise error logs with event/participant/gift IDs and operation names.
    - Do not log descriptions, Telegram file IDs, tokens, or private text.
  - Dependencies: Task 10.

### Phase 4: Admin And Miniapp Surfaces

- [x] 12. Add frontend place-rule and assignment types, formatting helpers, and helper tests.
  - Files:
    - `frontend/src/types/index.ts`
    - `frontend/src/utils/giftPlaceRule.ts`
    - `frontend/src/utils/giftPlaceRule.test.ts`
    - `frontend/package.json`
    - `frontend/package-lock.json`
  - Deliverable:
    - Add `GiftPlaceRule` TypeScript types.
    - Add `PrizeGiftAssignment` and `UnassignedPrizeSlot` TypeScript types if the backend exposes them.
    - Add helper to parse admin input `1, 3, 10-15` into sorted unique places.
    - Add helper to format rule summaries: `Места 10-15`, `Места 1, 3, 5`, `5 последних`, `Без привязки`.
    - Add helper to format assignment summaries, including fallback text such as `место 15 -> выдано месту 14`.
    - Add a minimal frontend test runner, preferably Vitest, because the project currently has no frontend unit tests.
  - TDD requirement:
    - Tests first for parsing, invalid input, duplicate normalization, range formatting, `last_n` formatting, empty rule formatting, and assignment/fallback formatting.
  - Logging requirements:
    - No runtime logging in pure frontend helpers.
  - Dependencies: Tasks 6 and 11.

- [x] 13. Update gift edit/review UI for place rules.
  - Files:
    - `frontend/src/components/gifts/GiftEditForm.tsx`
    - `frontend/src/app/(dashboard)/gifts/page.tsx`
    - `frontend/src/components/gifts/GiftsTable.tsx`
    - `frontend/src/api/gifts.ts`
    - `frontend/src/constants/options.ts`
  - Deliverable:
    - Replace the single place input with a compact place-rule editor:
      - no place rule;
      - exact places/ranges text input;
      - last N numeric input.
    - Add optional quick range controls `from` and `to` if it improves moderation speed.
    - Show a preview of normalized places or last-N rule before save.
    - Hydrate the form from existing `gift.place_rule`; fall back to legacy `gift.place` only if `place_rule` is absent.
    - Keep approval flow safe: approving a gift still submits description, filters, criteria IDs, review status, and place rule together.
    - Do not add a 1-200 visible button grid.
  - TDD requirement:
    - Cover parser/formatter through helper tests from Task 12.
    - Add manual/browser smoke checklist to the plan notes for the form, because no React test stack currently exists beyond the new helper tests.
  - Logging requirements:
    - Keep existing `console.error` for failed load/update/delete.
    - Do not log normal form edits or full gift descriptions.
  - Dependencies: Task 12.

- [x] 14. Update miniapp and admin display of place rules.
  - Files:
    - `frontend/src/components/miniapp/GiftCatalogTable.tsx`
    - `frontend/src/components/miniapp/GiftDetailView.tsx`
    - `frontend/src/app/(dashboard)/prize-distribution/page.tsx`
    - `frontend/src/types/index.ts`
  - Deliverable:
    - Miniapp catalog/detail shows readable place-rule summary instead of a single `place`.
    - Distribution table shows assignment metadata, including target place, assigned rank, `last_n`, and fallback status where available.
    - Distribution table shows the same source gift on multiple participant rows when slots assign it multiple times.
    - Add distribution stats for both participants with prizes and total prize assignments, because one participant row can contain multiple gifts and one source gift can create multiple assignments.
    - Match reason labels continue to make sense for place and last-N rules.
  - TDD requirement:
    - Helper tests from Task 12 must cover all display summaries.
    - Add Playwright/browser smoke during verification for gift edit, miniapp gift detail, and prize distribution table after implementation.
  - Logging requirements:
    - Keep concise `console.error` with event/gift IDs and operation names.
    - Do not log descriptions or Telegram file IDs.
  - Dependencies: Tasks 10-13.

### Phase 5: API Docs And Verification

- [x] 15. Update Swagger and API documentation for the new gift place contract.
  - Files:
    - `backend/docs/swagger.yaml`
  - Deliverable:
    - Document `Gift.place_rule`.
    - Document `UpdateGiftRequest.place_rule` omitted/null/places/last_n semantics.
    - Document `PrizeDistribution.matched_gift_assignments`, assignment metadata fields, and any `unassigned_slots` response field.
    - Document `place_by_gender_bike` in prize distribution if it is added to the DTO.
    - Mark `place` as legacy or compatibility-only if it remains in response/request schemas.
    - Add examples:
      - explicit range `10-15`;
      - `last_n = 5`;
      - no place rule.
  - TDD requirement:
    - No unit test needed for Swagger, but build/lint verification must catch YAML syntax issues where possible.
  - Logging requirements:
    - None.
  - Dependencies: Tasks 6, 13, 14.

- [x] 16. Run full verification and fix regressions.
  - Files:
    - all changed backend, frontend, migration, and docs files
  - Deliverable:
    - Run `cd backend && go test ./...`.
    - Run frontend helper tests, for example `cd frontend && npm run test -- --run` if Vitest is added.
    - Run targeted frontend lint for touched files or `cd frontend && npm run lint` if repo baseline allows it.
    - Run `cd frontend && npm run build`.
    - Run `git diff --check`.
    - If Docker Compose is needed to validate migrations, run `make docker-up`, `make migrate-up`, and a focused smoke against the admin API.
    - Browser-smoke the dashboard gift edit page and `/prize-distribution` for a representative event.
  - TDD requirement:
    - Do not mark implementation complete while any new tests are skipped or failing.
    - If "all possible" cases cannot be literally exhaustive, ensure every rule branch and conflict branch from the test matrix is covered.
  - Logging requirements:
    - Verify logs contain IDs, rule types, counts, and safe reasons only.
    - Verify logs do not contain gift descriptions, Telegram file IDs, tokens, raw private messages, or result URLs.
  - Dependencies: Tasks 1-15.

## Required Test Matrix

Backend distribution cases:

- Status:
  - approved gift participates;
  - pending gift is ignored.
- Group scopes:
  - all gender plus all bike;
  - male/female plus all bike;
  - all gender plus gravel/mtb/road/etc.;
  - male/female plus gravel/mtb/road/etc.
- Criteria:
  - no criteria;
  - one criterion;
  - multiple criteria all required;
  - participant missing one criterion excluded;
  - criteria applied before rank and before `last_n`.
- Rule types:
  - no place rule;
  - one explicit place;
  - multiple explicit places;
  - range;
  - range plus list;
  - duplicates normalized;
  - last N;
  - clear rule.
- Capacity:
  - no-place gift assigned once;
  - place-rule gift assigned once per slot;
  - same source gift assigned to multiple participants;
  - same source gift not assigned twice to the same participant.
- Fallback:
  - exact rank available;
  - target rank beyond group size;
  - target rank occupied by same gift slot selection;
  - nearest lower and higher candidates, tie chooses larger rank;
  - no free eligible candidate;
  - fallback cannot escape criteria group;
  - fallback cannot escape gender/bike group.
- Priority:
  - criteria plus place above criteria-only;
  - criteria-only above place-only;
  - place-only above generic;
  - lower-priority gifts are not bundled with higher-priority gifts;
  - multiple same-priority gifts are allowed.
- Ordering:
  - deterministic participant order by elapsed time, result ID, participant ID;
  - deterministic gift order by gift ID;
  - deterministic slot order by normalized rule order.
- API/update:
  - `place_rule` omitted preserves;
  - `place_rule: null` clears;
  - invalid rule returns 400;
  - legacy `place` maps to one explicit place;
  - `place_rule` wins when both `place_rule` and legacy `place` are present;
  - approving gift preserves criteria/filter/rule payload.
- Distribution response:
  - `matched_gift_assignments` includes gift, rule type, target rank, assigned rank, fallback flag, and match reason;
  - legacy `matched_gifts` remains populated during transition;
  - `unassigned_slots` exposes gift/rule/target/reason for slots that could not be assigned;
  - `place_by_gender_bike` is preserved in the API response.
- Dependent consumers:
  - participant detail shows assignment metadata and keeps backward-compatible gifts;
  - gifts page marks a source gift as distributed if any assignment uses it;
  - stats distinguish total prize assignments from participants with prizes.
- UI/helpers:
  - parse `1,3,10-15`;
  - reject malformed input;
  - format exact/range/last/none summaries;
  - format fallback assignment summaries;
  - preserve query filters when navigating from gifts list to edit page and back.

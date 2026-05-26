# Implementation Plan: Telegram Gift Photo Albums And Captions

Branch: main
Created: 2026-05-26

## Goal

Fix Telegram gift creation so photo albums and photos with captions are handled as normal gift input instead of falling through to the generic `/start` prompt.

## Requirements

- Keep the existing Telegram gift flow: gender -> bike type -> description -> photos -> confirmation.
- Do not change gift creation contracts, admin review rules, distribution logic, or persistence schema.
- Do not add `go-telegram/ui` or another bot UI dependency.
- When a user is inside an active gift flow, unsupported text/photo input must receive a contextual gift-flow prompt, not `Используйте /start для начала работы с ботом.`
- Telegram media groups are delivered as several `Message` updates with the same `MediaGroupID`; one user album action must not produce a repeated fallback response per photo.
- A photo sent with text/caption must be accepted:
  - in `StateAwaitingGiftDesc`, use the text/caption as the gift description and attach the photo from the same message;
  - in `StateAwaitingGiftPhoto`, attach the photo and do not overwrite the already collected description.
- Do not log gift descriptions, Telegram tokens, or raw file IDs.

## Settings

- Testing: yes
- Logging: standard
- Docs: no public API/docs changes expected

## Current Findings

- `backend/internal/infrastructure/telegram/handlers.go` handles normal messages by session state.
- `StateAwaitingGiftDesc` currently passes only `msg.Text` into `GiftHandler.HandleGiftDescription`, so a photo with caption can lose the caption and the photo is not attached in the same update.
- `StateAwaitingGiftPhoto` accepts a single `msg.Photo` update and stores the largest `PhotoSize` file ID through `GiftHandler.HandleGiftPhoto`.
- Telegram albums are not one backend message: each photo arrives as its own update, usually with the same `Message.MediaGroupID`.
- The generic fallback in `handleMessage` sends `Используйте /start для начала работы с ботом.` for any message state not explicitly handled. This is why photo album updates in states such as `StateAwaitingGiftGender` can create multiple fallback messages.
- Attachment persistence in `command.AddGiftHandler` already supports multiple `GiftAttachmentData` items and saves through `CreateWithAttachments`, so the fix should stay in Telegram infrastructure/session handling.

## Tasks

### Phase 1: Routing Model And Telegram Message Helpers

- [x] 1. Extract testable gift-flow message routing and Telegram message helpers.
  - Files:
    - `backend/internal/infrastructure/telegram/model_helpers.go`
    - `backend/internal/infrastructure/telegram/model_helpers_test.go`
    - `backend/internal/infrastructure/telegram/handlers.go` only for calling the extracted helpers/action model
    - `backend/internal/infrastructure/telegram/session/manager.go` only if a small session data helper is needed
  - Deliverable:
    - Add a helper that returns trimmed user text from `msg.Text`, falling back to trimmed `msg.Caption`.
    - Add a helper that returns the largest photo file ID from `msg.Photo`, with no panic on nil/empty messages.
    - Add a small infrastructure-local action/routing helper for gift messages, for example `giftMessageAction(state, msg, mediaGroupAlreadyReplied)`, so tests can verify behavior without a real Telegram API client.
    - The action model must distinguish processing from replying: every photo in a media group can still be attached, while repeated bot replies for the same `MediaGroupID` can be suppressed.
    - Store same-`MediaGroupID` reply suppression in session data with enough context to avoid suppressing unrelated later albums. Track only response suppression, not attachment processing.
    - Keep Telegram `models.Message` usage inside infrastructure; do not move Telegram types into application/domain packages.
    - Reuse existing event Telegram texts and step prompts for contextual responses; do not add domain text fields, migrations, or public API/docs changes.
  - Logging requirements:
    - Log duplicate media group response suppression at debug level only, with user ID, state, and media group ID.
    - Log malformed/nil message branches at debug or info level without text/caption/file IDs.
    - Do not log gift descriptions, Telegram tokens, or raw file IDs.
  - Dependencies: none.

### Phase 2: Focused Tests

- [x] 2. Add focused tests for gift-flow message inputs, captions, and album response suppression.
  - Files:
    - `backend/internal/infrastructure/telegram/model_helpers_test.go`
    - `backend/internal/infrastructure/telegram/handler/gift_test.go`
    - `backend/internal/infrastructure/telegram/handlers.go` extracted helpers/action model from Task 1
  - Deliverable:
    - Cover text extraction from `Message.Text` and `Message.Caption`, including whitespace-only values.
    - Cover largest-photo file ID extraction from `Message.Photo` using the current `github.com/go-telegram/bot v1.21.0` `models.PhotoSize` shape.
    - Cover media group duplicate-response suppression by `MediaGroupID`: repeated updates may suppress replies, but must not suppress attachment processing.
    - Cover `StateAwaitingGiftDesc` with `Caption + Photo`: description is stored from caption and the photo is added to `gift_attachments`.
    - Cover `StateAwaitingGiftDesc` with photo but no text/caption: state remains `StateAwaitingGiftDesc`, no empty `gift_description` is stored, and the response asks for description instead of moving to photos.
    - Cover `StateAwaitingGiftPhoto` with `Caption + Photo`: photo is added and existing `gift_description` is preserved.
    - Cover active gift states that receive out-of-order photo/text input so they do not fall through to the idle `/start` fallback.
  - Logging requirements:
    - Tests should assert state/data/action outcomes; no new runtime logging is required for test-only code.
    - If log capture is added, assert only metadata such as user ID/state/media group ID, not descriptions or file IDs.
  - Dependencies: Task 1.

### Phase 3: Update Gift Flow Message Handling

- [x] 3. Update `handleMessage` and `GiftHandler` wiring so captions and albums are accepted in the right states.
  - Files:
    - `backend/internal/infrastructure/telegram/handlers.go`
    - `backend/internal/infrastructure/telegram/handler/gift.go`
    - `backend/internal/infrastructure/telegram/handler/gift_test.go`
  - Deliverable:
    - In `StateAwaitingGiftDesc`, use the new text-or-caption helper. If the same message has a photo, save the description and immediately append that photo to `gift_attachments`.
    - In `StateAwaitingGiftDesc`, if the message has a photo but no non-empty text/caption, keep the user in `StateAwaitingGiftDesc`, do not store an empty description, and respond with the existing description step guidance.
    - In `StateAwaitingGiftPhoto`, append photo updates even when `msg.Caption` is present; keep the existing description unchanged.
    - For media groups, process every photo attachment, but suppress repeated contextual/fallback bot responses for the same `MediaGroupID` where possible. Do not suppress the second photo attachment just because the first update already produced a response.
    - Avoid sending a misleading final photo count for the first update in an album if more album photos can still arrive; prefer either a generic contextual response or one response after the flow state is stable.
    - In active gift states that expect buttons (`StateAwaitingGiftGender`, `StateAwaitingGiftBikeType`, `StateAwaitingGiftConfirmation`), respond with the current step guidance or a short contextual message instead of the global `/start` fallback; apply media-group response suppression so one album does not trigger repeated replies.
    - Leave the idle/default `/start` fallback for users with no active session.
    - Keep application command contracts, persistence, schema, admin review, and prize distribution untouched.
  - Logging requirements:
    - Log out-of-order gift-flow input at info/debug with user ID, state, update kind, and media group ID.
    - Log added photo counts only as counts; do not log Telegram file IDs.
    - Log missing caption/text when a description is required, but do not log the raw message content.
  - Dependencies: Task 2.

### Phase 4: Verify Backend Behavior

- [x] 4. Run focused and full backend checks.
  - Files:
    - No production files beyond Tasks 2-3.
  - Deliverable:
    - Run `cd backend && go test ./internal/infrastructure/telegram/...`.
    - Run `cd backend && go test ./...`.
    - Manually inspect the final diff for unchanged architecture boundaries: Telegram models stay in infrastructure, application command contracts stay unchanged, no ORM or schema changes.
    - If a live Telegram smoke test is available, verify:
      - two photos sent as one album do not produce two `/start` messages;
      - a photo with caption can create/continue gift input;
      - the confirmation preview shows the expected description and photo count.
  - Logging requirements:
    - Verification should confirm normal gift photo handling does not produce noisy logs.
    - Any failing path should include enough metadata to diagnose user ID/state/update kind without private text or file IDs.
  - Dependencies: Task 3.

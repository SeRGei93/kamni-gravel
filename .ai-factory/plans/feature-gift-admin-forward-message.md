# Implementation Plan: Gift Admin Forward Message

Branch: feature/gift-admin-forward-message
Created: 2026-05-28

## Settings
- Testing: yes
- Logging: standard
- Docs: no

## Goal

Replace the current admin-chat gift review notification, which copies the user's text/photo messages and then sends a separate technical summary, with one public-ready Telegram notification.

The resulting admin-chat notification must be directly forwardable to the public chat and must show:

- who submitted the gift;
- gift description;
- gender filter;
- bike type filter;
- all gift photos when available.

It must not show internal technical data such as gift ID, event ID, review status, raw Telegram file IDs, or source message copy/forward artifacts.

## Current Findings

- The current split notification is produced in `backend/internal/infrastructure/telegram/bot.go` by `notifyAdminAboutGift`, `copyOrForwardGiftSources`, `sendAdminGiftSummary`, and `adminGiftSummaryText`.
- The confirmation callback in `backend/internal/infrastructure/telegram/handlers.go` passes captured source message refs into `notifyAdminAboutGift`.
- Source refs are captured in `backend/internal/infrastructure/telegram/model_helpers.go` only to copy/forward original user messages to the admin chat.
- The saved `entity.Gift` already has enough data for a clean notification: `User`, `Description`, `GenderFilter`, `BikeTypeFilter`, and `Attachments`.
- Telegram can send one photo as `SendPhoto` with a caption. Multiple photos should be sent as a media group/album via `SendMediaGroup`, with the public-ready text as the caption on the first photo. Do not send a separate text summary after the album. This preserves all photos without copying the user's original source messages. A strict single Telegram message with multiple photos would require generated collage/image processing and is outside this plan unless the user explicitly asks for it.

## Scope Boundaries

- Do not change gift creation questions or confirmation flow.
- Do not change persistence schema, domain contracts, HTTP API contracts, admin dashboard, miniapp, review statuses, or prize distribution logic.
- Do not add `go-telegram/ui` or new Telegram UI dependencies.
- Keep the change inside `backend/internal/infrastructure/telegram` unless a compile-time interface adjustment requires a nearby test fake update.
- Keep admin diagnostics in logs, not in the public-ready Telegram text.
- Do not drop extra gift photos. Use all usable `photo` attachments with non-empty Telegram file IDs.

## Tasks

### Phase 1: Message Composition

- [x] 1. Add a public-ready admin gift notification text composer.

  Deliverable:
  - Replace the technical summary wording with a human-readable message in `backend/internal/infrastructure/telegram/bot.go`.
  - Include donor display name from `gift.User` when available. Prefer full name plus `@username`; fall back to `@username`; fall back to `user_id` only if no profile fields exist.
  - Include `gift.Description`, `giftGenderLabel(gift.GenderFilter)`, and `giftBikeTypeLabel(gift.BikeTypeFilter)`.
  - Exclude gift ID, event ID, raw status, internal attachment count, and raw Telegram file IDs from the Telegram message body.
  - Handle missing/nil gift data without panic and still return a non-empty fallback text.
  - Enforce Telegram message limits:
    - caption text for `SendPhoto` and first `SendMediaGroup` item must fit Telegram caption limits;
    - text-only fallback must fit Telegram text message limits;
    - truncate only the description when needed, keeping donor, gender, and bike type visible.

  Files:
  - `backend/internal/infrastructure/telegram/bot.go`
  - `backend/internal/infrastructure/telegram/bot_test.go`

  Logging requirements:
  - No logs for normal pure text composition.
  - Do not log full gift descriptions or Telegram file IDs.
  - Keep malformed/nil gift diagnostics at WARN only if the sending path needs to fall back.

### Phase 2: Telegram Send Path With All Photos

- [x] 2. Replace source copy/forward plus summary with a public-ready admin notification that includes all photos.

  Deliverable:
  - Extend the local `telegramAPI` interface with `SendPhoto` and `SendMediaGroup`; update test fakes.
  - Add a helper such as `giftPhotoFileIDs(gift)` that returns all usable photo attachment IDs in saved order:
    - include only `FileType == "photo"`;
    - trim and skip empty `TelegramFileID`;
    - ignore documents and unsupported attachment types.
  - Change `notifyAdminAboutGift` so it sends a clean admin-chat notification:
    - if the gift has exactly one usable photo, call `SendPhoto` with `models.InputFileString{Data: <telegram_file_id>}` and the public-ready caption;
    - if the gift has multiple usable photos, call `SendMediaGroup` with one `models.InputMediaPhoto` per photo and the public-ready caption on the first item;
    - do not send an additional text message after a successful `SendMediaGroup`; the album caption is the user-visible summary;
    - if the photo count exceeds the Telegram media-group limit, send all photos in valid media-group chunks, with the caption only on the first media item of the first chunk and structured logs for the chunk count;
    - if the gift has no usable photos, call `SendMessage` with the public-ready text;
    - if `SendPhoto` or `SendMediaGroup` fails, log a warning and send the public-ready text fallback so admins still see the gift data.
  - Stop copying or forwarding the user's original source messages into the admin chat.
  - Remove or bypass `copyOrForwardGiftSources`, `sendAdminGiftSummary`, and technical summary text behavior.
  - Preserve existing behavior where disabled `ADMIN_CHAT_ID` skips notification without failing the user confirmation flow.

  Files:
  - `backend/internal/infrastructure/telegram/bot.go`
  - `backend/internal/infrastructure/telegram/handlers_test.go`

  Logging requirements:
  - INFO when an admin notification is sent, with `gift_id`, `event_id`, `user_id`, `chat=admin`, `photo_count`, and `media_group_count`.
  - WARN when admin chat is disabled, attachments exist but no usable photos are found, or photo/media-group sending falls back to text.
  - ERROR only if the final one-message notification cannot be sent.
  - Never log raw Telegram file IDs or full gift descriptions.

### Phase 3: Remove Obsolete Source-Ref Plumbing

- [x] 3. Remove admin-notification source-ref capture that is no longer needed.

  Deliverable:
  - Update the gift confirmation callback in `backend/internal/infrastructure/telegram/handlers.go` to call `notifyAdminAboutGift(ctx, gift)` without source refs.
  - Remove session source-ref cleanup from the confirmation callback if it becomes unused.
  - Remove `giftSourceRef`, `giftSourceRefsKey`, `captureGiftMessageSourceRef`, `giftSourceRefs`, and `setGiftSourceRefs` if they are no longer referenced.
  - Remove CopyMessage/ForwardMessage from `telegramAPI` and from `telegramAPIFake` only if no other Telegram code still uses them.
  - Delete or replace tests that only validate old source-ref collection order, because the new notification should use persisted gift attachments instead of copied source messages.
  - Keep message ID cleanup for the user's private gift flow unchanged.

  Files:
  - `backend/internal/infrastructure/telegram/handlers.go`
  - `backend/internal/infrastructure/telegram/model_helpers.go`
  - `backend/internal/infrastructure/telegram/model_helpers_test.go`
  - `backend/internal/infrastructure/telegram/handlers_test.go`
  - `backend/internal/infrastructure/telegram/bot.go`

  Logging requirements:
  - No new logs are needed for removing source-ref storage.
  - Preserve existing logs for user-facing gift flow state handling.
  - If any source-ref removal exposes malformed session data, log only user ID, state, and key names, not message text or file IDs.

### Phase 4: Focused Tests And Verification

- [x] 4. Update and run focused tests for the new notification behavior.

  Deliverable:
  - Replace tests that expect copied/forwarded source messages with tests that expect:
    - one `SendPhoto` call when exactly one usable photo exists;
    - one `SendMediaGroup` call when multiple usable photos exist within Telegram's media-group limit;
    - all usable photos included in order and no copied/forwarded messages.
  - Add/adjust tests for text-only gift notification.
  - Add/adjust tests for mixed attachments: documents and empty file IDs are ignored, but every usable photo is sent.
  - Add/adjust tests proving a successful media group does not produce a separate text summary.
  - Add/adjust tests for photo/media-group send failure falling back to text.
  - Add/adjust tests for photo counts above Telegram's media-group limit so all photos are still attempted in chunks.
  - Add/adjust integration-style callback test proving `confirm_gift` sends the admin notification from persisted gift attachments while the private user success message remains unchanged.
  - Add/adjust tests proving the notification text contains donor, description, gender, and bike type, and does not contain internal gift ID/event ID/review status labels.
  - Run focused Telegram tests and then backend tests.

  Files:
  - `backend/internal/infrastructure/telegram/bot_test.go`
  - `backend/internal/infrastructure/telegram/handlers_test.go`
  - `backend/internal/infrastructure/telegram/model_helpers_test.go`

  Verification commands:
  - `cd backend && go test ./internal/infrastructure/telegram/...`
  - `cd backend && go test ./...`
  - `cd backend && go vet ./...`

  Logging requirements:
  - Tests should assert user-visible message content, not logs.
  - Keep production logs structured enough to diagnose send failures without exposing message contents.

## Expected Final Behavior

For a gift with one photo, admins receive one photo message with a caption similar to:

```text
Новый подарок на проверку

От: Alex Rider (@alex)
Описание: Лабуба за 1 и 10 место
Гендер: 👨 Мужской
Велосипед: 🚴 Шоссе
```

For a gift with multiple photos, admins receive a media group containing all usable photos, with the same caption on the first photo and no separate text summary after the album.

For a gift without usable photos, admins receive the same content as one text message.

All variants should be forwardable to the public chat without exposing internal IDs, review statuses, raw file IDs, or copied source-message fragments.

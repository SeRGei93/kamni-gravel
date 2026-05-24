package handler

import (
	"context"
	"testing"
	"time"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/infrastructure/telegram/session"
)

func TestGiftHandlerHandleGiftDescriptionSetsPhotoState(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewGiftHandler(manager, nil, nil)

	text, markup := h.HandleGiftDescription(context.Background(), 123, "Bottle cage")

	if text == "" {
		t.Fatal("text mismatch: got empty text")
	}
	if markup == nil {
		t.Fatal("markup mismatch: got nil")
	}
	if got := manager.GetState(123); got != session.StateAwaitingGiftPhoto {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingGiftPhoto)
	}

	descriptionRaw, ok := manager.GetData(123, "gift_description")
	if !ok {
		t.Fatal("gift_description missing from session")
	}
	if descriptionRaw != "Bottle cage" {
		t.Fatalf("gift_description mismatch: got %v, want Bottle cage", descriptionRaw)
	}
}

func TestGiftHandlerHandleGiftPhotoTracksAttachment(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewGiftHandler(manager, nil, nil)

	text := h.HandleGiftPhoto(123, "telegram-file-id")

	if text == "" {
		t.Fatal("text mismatch: got empty text")
	}

	attachmentsRaw, ok := manager.GetData(123, "gift_attachments")
	if !ok {
		t.Fatal("gift_attachments missing from session")
	}

	attachments, ok := attachmentsRaw.([]command.GiftAttachmentData)
	if !ok {
		t.Fatalf("gift_attachments type mismatch: got %T", attachmentsRaw)
	}
	if len(attachments) != 1 {
		t.Fatalf("attachment count mismatch: got %d, want 1", len(attachments))
	}
	if attachments[0].TelegramFileID != "telegram-file-id" {
		t.Fatalf("telegram file id mismatch: got %q", attachments[0].TelegramFileID)
	}
	if attachments[0].FileType != "photo" {
		t.Fatalf("file type mismatch: got %q, want photo", attachments[0].FileType)
	}
}

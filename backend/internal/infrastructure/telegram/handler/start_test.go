package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestStartHandlerHandleMissingSender(t *testing.T) {
	h := NewStartHandler(nil, nil, "")

	text, markup := h.Handle(context.Background(), &models.Message{
		ID:   10,
		Chat: models.Chat{ID: 20},
	})

	if markup != nil {
		t.Fatalf("markup mismatch: got %#v, want nil", markup)
	}
	if !strings.Contains(text, "Не удалось определить пользователя") {
		t.Fatalf("text mismatch: got %q", text)
	}
}

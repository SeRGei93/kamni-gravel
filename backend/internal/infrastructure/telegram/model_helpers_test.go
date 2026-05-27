package telegram

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/infrastructure/telegram/session"
)

func TestMessageCommand(t *testing.T) {
	tests := []struct {
		name string
		msg  *models.Message
		want string
	}{
		{
			name: "start command",
			msg: &models.Message{
				Text:     "/start",
				Entities: []models.MessageEntity{{Type: models.MessageEntityTypeBotCommand, Offset: 0, Length: 6}},
			},
			want: "start",
		},
		{
			name: "start command with bot username",
			msg: &models.Message{
				Text:     "/start@GravelBot",
				Entities: []models.MessageEntity{{Type: models.MessageEntityTypeBotCommand, Offset: 0, Length: 16}},
			},
			want: "start",
		},
		{
			name: "menu command",
			msg: &models.Message{
				Text:     "/menu",
				Entities: []models.MessageEntity{{Type: models.MessageEntityTypeBotCommand, Offset: 0, Length: 5}},
			},
			want: "menu",
		},
		{
			name: "not a command entity",
			msg: &models.Message{
				Text:     "/start",
				Entities: []models.MessageEntity{{Type: models.MessageEntityTypeMention, Offset: 0, Length: 6}},
			},
			want: "",
		},
		{
			name: "nil message",
			msg:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := messageCommand(tt.msg); got != tt.want {
				t.Fatalf("command mismatch: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCallbackMessage(t *testing.T) {
	t.Run("accessible message", func(t *testing.T) {
		ref, ok := callbackMessage(&models.CallbackQuery{
			ID: "callback-id",
			Message: models.MaybeInaccessibleMessage{
				Message: &models.Message{
					ID:   77,
					Chat: models.Chat{ID: 99},
				},
			},
		})
		if !ok {
			t.Fatal("callback message mismatch: got inaccessible")
		}
		if ref.ChatID != 99 || ref.MessageID != 77 {
			t.Fatalf("callback message ref mismatch: got %#v", ref)
		}
	})

	t.Run("inaccessible message", func(t *testing.T) {
		_, ok := callbackMessage(&models.CallbackQuery{
			ID: "callback-id",
			Message: models.MaybeInaccessibleMessage{
				InaccessibleMessage: &models.InaccessibleMessage{
					Chat:      models.Chat{ID: 99},
					MessageID: 77,
				},
			},
		})
		if ok {
			t.Fatal("callback message mismatch: got accessible, want inaccessible")
		}
	})
}

func TestMessageTextOrCaption(t *testing.T) {
	tests := []struct {
		name string
		msg  *models.Message
		want string
	}{
		{
			name: "text wins over caption",
			msg:  &models.Message{Text: "  text value  ", Caption: "caption value"},
			want: "text value",
		},
		{
			name: "caption fallback",
			msg:  &models.Message{Caption: "  caption value  "},
			want: "caption value",
		},
		{
			name: "whitespace only",
			msg:  &models.Message{Text: " \n\t ", Caption: "  "},
			want: "",
		},
		{
			name: "nil message",
			msg:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := messageTextOrCaption(tt.msg); got != tt.want {
				t.Fatalf("text mismatch: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLargestPhotoFileID(t *testing.T) {
	tests := []struct {
		name string
		msg  *models.Message
		want string
		ok   bool
	}{
		{
			name: "largest by file size",
			msg: &models.Message{Photo: []models.PhotoSize{
				{FileID: "small", FileSize: 10, Width: 10, Height: 10},
				{FileID: "large", FileSize: 30, Width: 20, Height: 20},
				{FileID: "medium", FileSize: 20, Width: 30, Height: 30},
			}},
			want: "large",
			ok:   true,
		},
		{
			name: "largest by dimensions when file size is missing",
			msg: &models.Message{Photo: []models.PhotoSize{
				{FileID: "small", Width: 10, Height: 10},
				{FileID: "large", Width: 40, Height: 30},
			}},
			want: "large",
			ok:   true,
		},
		{
			name: "empty photo file id",
			msg:  &models.Message{Photo: []models.PhotoSize{{FileID: "  ", Width: 40, Height: 30}}},
			want: "",
			ok:   false,
		},
		{
			name: "nil message",
			msg:  nil,
			want: "",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := largestPhotoFileID(tt.msg)
			if ok != tt.ok {
				t.Fatalf("ok mismatch: got %t, want %t", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("file id mismatch: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGiftMessageAction(t *testing.T) {
	t.Run("description caption with photo processes both and asks for photo step", func(t *testing.T) {
		action := giftMessageAction(session.StateAwaitingGiftDesc, &models.Message{
			Caption: "  Bottle cage  ",
			Photo: []models.PhotoSize{
				{FileID: "photo-small", Width: 10, Height: 10},
				{FileID: "photo-large", Width: 20, Height: 20},
			},
		}, false)

		if !action.ProcessDescription || !action.ProcessPhoto {
			t.Fatalf("action should process description and photo: %#v", action)
		}
		if action.Description != "Bottle cage" {
			t.Fatalf("description mismatch: got %q", action.Description)
		}
		if action.PhotoFileID != "photo-large" {
			t.Fatalf("photo file id mismatch: got %q", action.PhotoFileID)
		}
		if action.Reply != giftMessageReplyGiftPhotoStep {
			t.Fatalf("reply mismatch: got %v, want gift photo step", action.Reply)
		}
	})

	t.Run("description photo without caption keeps description state", func(t *testing.T) {
		action := giftMessageAction(session.StateAwaitingGiftDesc, &models.Message{
			Photo: []models.PhotoSize{{FileID: "photo", Width: 20, Height: 20}},
		}, false)

		if action.ProcessDescription || action.ProcessPhoto {
			t.Fatalf("action should not process incomplete description input: %#v", action)
		}
		if action.Reply != giftMessageReplyGiftDescriptionStep {
			t.Fatalf("reply mismatch: got %v, want gift description step", action.Reply)
		}
		if !action.MissingInput || !action.OutOfOrder {
			t.Fatalf("action should flag missing/out-of-order input: %#v", action)
		}
	})

	t.Run("media group duplicate suppresses reply but keeps photo processing", func(t *testing.T) {
		action := giftMessageAction(session.StateAwaitingGiftPhoto, &models.Message{
			MediaGroupID: "album-1",
			Photo:        []models.PhotoSize{{FileID: "photo", Width: 20, Height: 20}},
		}, true)

		if !action.ProcessPhoto {
			t.Fatalf("action should keep processing photo: %#v", action)
		}
		if !action.SuppressReply {
			t.Fatalf("action should suppress duplicate reply: %#v", action)
		}
		if action.Reply != giftMessageReplyNone {
			t.Fatalf("reply mismatch: got %v, want none", action.Reply)
		}
	})

	t.Run("photo state with caption processes only photo", func(t *testing.T) {
		action := giftMessageAction(session.StateAwaitingGiftPhoto, &models.Message{
			Caption: "should not replace description",
			Photo:   []models.PhotoSize{{FileID: "photo", Width: 20, Height: 20}},
		}, false)

		if action.ProcessDescription {
			t.Fatalf("action should not process caption as description in photo state: %#v", action)
		}
		if !action.ProcessPhoto {
			t.Fatalf("action should process photo: %#v", action)
		}
		if action.Reply != giftMessageReplyGiftPhotoAdded {
			t.Fatalf("reply mismatch: got %v, want photo added", action.Reply)
		}
	})

	t.Run("active button states get contextual replies", func(t *testing.T) {
		tests := []struct {
			name  string
			state session.SessionState
			reply giftMessageReply
		}{
			{name: "gender", state: session.StateAwaitingGiftGender, reply: giftMessageReplyGiftGenderStep},
			{name: "bike", state: session.StateAwaitingGiftBikeType, reply: giftMessageReplyGiftBikeStep},
			{name: "confirmation", state: session.StateAwaitingGiftConfirmation, reply: giftMessageReplyGiftConfirmationStep},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				action := giftMessageAction(tt.state, &models.Message{
					Text: "unexpected text",
				}, false)

				if action.Reply != tt.reply {
					t.Fatalf("reply mismatch: got %v, want %v", action.Reply, tt.reply)
				}
				if !action.OutOfOrder {
					t.Fatalf("action should flag out-of-order input: %#v", action)
				}
				if action.ProcessDescription || action.ProcessPhoto {
					t.Fatalf("action should not process button-state message: %#v", action)
				}
			})
		}
	})
}

func TestGiftMediaGroupAlreadyReplied(t *testing.T) {
	userID := int64(123)
	b := &Bot{
		debug:          true,
		sessionManager: session.NewManager(time.Minute),
	}
	msg := &models.Message{
		ID:           10,
		Chat:         models.Chat{ID: 99},
		MediaGroupID: "album-1",
	}

	if b.giftMediaGroupAlreadyReplied(userID, session.StateAwaitingGiftDesc, msg) {
		t.Fatal("first media group message should not be treated as duplicate")
	}
	if !b.giftMediaGroupAlreadyReplied(userID, session.StateAwaitingGiftPhoto, msg) {
		t.Fatal("same media group should suppress repeated reply across gift states")
	}
	if b.giftMediaGroupAlreadyReplied(userID, session.StateAwaitingGiftPhoto, &models.Message{
		ID:           12,
		Chat:         models.Chat{ID: 99},
		MediaGroupID: "album-2",
	}) {
		t.Fatal("new media group should not be suppressed")
	}
}

func TestGiftMessageIDTracking(t *testing.T) {
	b := &Bot{sessionManager: session.NewManager(time.Minute)}

	b.appendGiftMessageID(123, 10)
	b.appendGiftMessageID(123, 11)

	want := []int{10, 11}
	if got := b.giftMessageIDs(123); !reflect.DeepEqual(got, want) {
		t.Fatalf("gift message IDs mismatch: got %v, want %v", got, want)
	}
}

func TestCaptureGiftMessageSourceRefAddsMessagesInArrivalOrder(t *testing.T) {
	b := &Bot{
		sessionManager: session.NewManager(time.Minute),
	}

	first := &models.Message{
		ID:   10,
		Chat: models.Chat{ID: 55},
		Text: "first",
	}
	second := &models.Message{
		ID:    11,
		Chat:  models.Chat{ID: 55},
		Photo: []models.PhotoSize{{FileID: "photo-1", Width: 10, Height: 10}},
	}
	third := &models.Message{
		ID:      12,
		Chat:    models.Chat{ID: 55},
		Caption: "caption",
		Photo:   []models.PhotoSize{{FileID: "photo-2", Width: 12, Height: 12}},
	}

	b.captureGiftMessageSourceRef(123, first)
	b.captureGiftMessageSourceRef(123, second)
	b.captureGiftMessageSourceRef(123, third)

	refs := b.giftSourceRefs(123)
	if len(refs) != 3 {
		t.Fatalf("source refs mismatch: got %d, want %d", len(refs), 3)
	}
	if refs[0].MessageID != 10 || refs[1].MessageID != 11 || refs[2].MessageID != 12 {
		t.Fatalf("source refs order mismatch: got %#v", refs)
	}
	if refs[0].UpdateKind != "text" || refs[1].UpdateKind != "photo" || refs[2].UpdateKind != "photo" {
		t.Fatalf("source ref kinds mismatch: got %#v", refs)
	}
}

func TestCaptureGiftMessageSourceRefSkipsUnsupportedKinds(t *testing.T) {
	b := &Bot{
		sessionManager: session.NewManager(time.Minute),
	}

	b.captureGiftMessageSourceRef(123, &models.Message{
		ID:   10,
		Chat: models.Chat{ID: 55},
	})

	refs := b.giftSourceRefs(123)
	if len(refs) != 0 {
		t.Fatalf("unsupported update kind should be skipped, got %#v", refs)
	}
}

func TestHandleGiftMessageDoesNotCaptureOutOfOrderSourceRef(t *testing.T) {
	b := &Bot{
		api:            &telegramAPIFake{},
		sessionManager: session.NewManager(time.Minute),
	}

	b.handleGiftMessage(context.Background(), &models.Message{
		ID:   10,
		Chat: models.Chat{ID: 55},
		Text: "unexpected text",
	}, 123, session.StateAwaitingGiftGender)

	refs := b.giftSourceRefs(123)
	if len(refs) != 0 {
		t.Fatalf("out-of-order gift input should not be captured as source ref, got %#v", refs)
	}
}

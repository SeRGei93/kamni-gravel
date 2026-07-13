package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
	"gravel_bot/internal/infrastructure/http/middleware"
)

func TestMiniappSessionReturnsTelegramUserAndActiveEvent(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	h := newMiniappTestHandler(&miniappEventRepoFake{
		activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Description: "Race gifts", Active: true},
	}, nil, nil, nil)

	rr := miniappRequest(t, token, now, h.Session, "/api/miniapp/session")

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var got MiniappSessionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.User.ID != 42 || got.User.FirstName != "Alex" {
		t.Fatalf("user mismatch: %#v", got.User)
	}
	if got.Event.ID != 77 || got.Event.Name != "Gravel Race" {
		t.Fatalf("event mismatch: %#v", got.Event)
	}
	if got.HasMyGifts {
		t.Fatal("has my gifts = true, want false")
	}
}

func TestMiniappSessionIncludesOwnerGiftPresence(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	h := newMiniappTestHandler(
		&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Active: true}},
		&miniappHandlerGiftRepoFake{hasOwnerGifts: true},
		nil,
		nil,
	)

	rr := miniappRequest(t, token, now, h.Session, "/api/miniapp/session")
	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var got MiniappSessionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !got.HasMyGifts {
		t.Fatal("has my gifts = false, want true")
	}
}

func TestMiniappSessionIncludesCurrentUsersFinishedResult(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	h := newMiniappTestHandler(
		&miniappEventRepoFake{
			activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Active: true},
		},
		nil,
		nil,
		&miniappHandlerParticipantRepoFake{
			participants: []*entity.Participant{
				{ID: 11, UserID: 42, EventID: 77, Result: &entity.Result{}},
				{ID: 12, UserID: 84, EventID: 77, Result: &entity.Result{}},
			},
		},
	)

	rr := miniappRequest(t, token, now, h.Session, "/api/miniapp/session")

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var got MiniappSessionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.MyResultParticipantID == nil || *got.MyResultParticipantID != 11 {
		t.Fatalf("my result participant mismatch: %#v", got.MyResultParticipantID)
	}
}

func TestMiniappSessionReturnsNotFoundWhenNoActiveEvent(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	h := newMiniappTestHandler(&miniappEventRepoFake{}, nil, nil, nil)

	rr := miniappRequest(t, token, now, h.Session, "/api/miniapp/session")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
	}
}

func TestMiniappSessionReturnsNotFoundForTypedNoActiveEvent(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	h := newMiniappTestHandler(&miniappEventRepoFake{
		activeErr: fmt.Errorf("find active event: %w", repository.ErrNoActiveEvent),
	}, nil, nil, nil)

	rr := miniappRequest(t, token, now, h.Session, "/api/miniapp/session")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
	}
}

func TestMiniappSessionTreatsMissingParticipantAsValid(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	h := newMiniappTestHandler(
		&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Active: true}},
		nil,
		nil,
		&miniappHandlerParticipantRepoFake{findByUserAndEventErr: fmt.Errorf("query participant: %w", repository.ErrParticipantNotFound)},
	)

	rr := miniappRequest(t, token, now, h.Session, "/api/miniapp/session")
	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var got MiniappSessionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.MyResultParticipantID != nil {
		t.Fatalf("my result participant id = %v, want nil", *got.MyResultParticipantID)
	}
}

func TestMiniappMyGiftsReturnsOnlyVerifiedUsersActiveEventGifts(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	recipientID := uint(11)
	place := 2
	giftRepo := &miniappHandlerGiftRepoFake{
		ownerGifts: []*entity.Gift{
			{ID: 1, UserID: 42, EventID: 77, Description: "Manual bottle", ManualDistribution: true, ManualRecipientParticipantID: &recipientID, ReviewStatus: entity.GiftReviewStatusPendingReview, ManualRecipient: &entity.Participant{ID: recipientID, User: &entity.User{FirstName: "Ivan", Username: "ivan"}}},
			{ID: 2, UserID: 42, EventID: 77, Description: "Automatic cap", GenderFilter: "female", BikeTypeFilter: "gravel", Place: &place, ReviewStatus: entity.GiftReviewStatusApproved},
		},
		attachments: map[uint][]*entity.GiftAttachment{
			1: {{ID: 13, GiftID: 1, TelegramFileID: "gift-photo", FileType: "photo"}},
		},
	}
	criteriaRepo := &miniappHandlerCriteriaRepoFake{criteriaByGift: map[uint][]*entity.Criteria{
		2: {{ID: 5, Name: "Самая быстрая", CriteriaType: "speed"}},
	}}
	h := newMiniappTestHandler(
		&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Active: true}}, giftRepo, criteriaRepo, nil,
	)

	rr := miniappRequest(t, token, now, h.MyGifts, "/api/miniapp/my-gifts")
	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var got dto.ManualGiftListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Gifts) != 2 || got.Gifts[0].ID != 1 || got.Gifts[1].ID != 2 || got.Gifts[0].Recipient == nil || got.Gifts[0].Recipient.ID != recipientID {
		t.Fatalf("my gifts mismatch: %#v", got.Gifts)
	}
	if len(got.Gifts[0].Attachments) != 1 || got.Gifts[0].Attachments[0].TelegramFileID != "gift-photo" {
		t.Fatalf("my gift attachments mismatch: %#v", got.Gifts[0].Attachments)
	}
	if got.Gifts[1].GenderFilter != "female" || got.Gifts[1].BikeTypeFilter != "gravel" || got.Gifts[1].Place == nil || *got.Gifts[1].Place != place || len(got.Gifts[1].Criteria) != 1 || got.Gifts[1].Criteria[0].ID != 5 {
		t.Fatalf("automatic gift conditions mismatch: %#v", got.Gifts[1])
	}
	if strings.Contains(rr.Body.String(), "user_id") {
		t.Fatalf("response leaks recipient telegram user id: %s", rr.Body.String())
	}
}

func TestMiniappParticipantsReturnsMinimalActiveEventOptions(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	giftRepo := &miniappHandlerGiftRepoFake{manualRecipientCounts: map[uint]int{2: 1}}
	h := newMiniappTestHandler(
		&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Active: true}},
		giftRepo,
		nil,
		&miniappHandlerParticipantRepoFake{participants: []*entity.Participant{
			{ID: 2, EventID: 77, UserID: 202, Status: valueobject.ParticipantStatusDNF, Notes: "private", User: &entity.User{FirstName: "Zoe", Username: "zoe"}},
			{ID: 1, EventID: 77, UserID: 101, Status: valueobject.ParticipantStatusActive, User: &entity.User{FirstName: "Alex", Username: "alex"}},
		}},
	)

	rr := miniappRequest(t, token, now, h.Participants, "/api/miniapp/participants")
	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var got struct {
		Participants []dto.MiniappParticipantOptionDTO `json:"participants"`
		Total        int                               `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 2 || len(got.Participants) != 2 || got.Participants[0].ID != 1 || got.Participants[1].Status != string(valueobject.ParticipantStatusDNF) || got.Participants[0].HasPrize || !got.Participants[1].HasPrize {
		t.Fatalf("participant options mismatch: %#v", got)
	}
	if strings.Contains(rr.Body.String(), "user_id") || strings.Contains(rr.Body.String(), "private") {
		t.Fatalf("response leaks private participant data: %s", rr.Body.String())
	}
}

func TestMiniappUpdateMyGiftRecipient(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	manualGift := &entity.Gift{ID: 9, UserID: 42, EventID: 77, ManualDistribution: true}
	giftRepo := &miniappHandlerGiftRepoFake{giftByID: manualGift}
	participantRepo := &miniappHandlerParticipantRepoFake{participants: []*entity.Participant{{ID: 11, EventID: 77}}}
	h := newMiniappTestHandler(&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Active: true}}, giftRepo, nil, participantRepo)

	rr := miniappUpdateRecipientRequest(t, token, now, h, `{"participant_id":11}`)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("assign status mismatch: got %d, want %d body=%s", rr.Code, http.StatusNoContent, rr.Body.String())
	}
	if giftRepo.setCalls != 1 || giftRepo.setRecipientID == nil || *giftRepo.setRecipientID != 11 {
		t.Fatalf("recipient write mismatch: calls=%d recipient=%v", giftRepo.setCalls, giftRepo.setRecipientID)
	}

	rr = miniappUpdateRecipientRequest(t, token, now, h, `{"participant_id":null}`)
	if rr.Code != http.StatusNoContent || giftRepo.setCalls != 2 || giftRepo.setRecipientID != nil {
		t.Fatalf("clear mismatch: status=%d calls=%d recipient=%v", rr.Code, giftRepo.setCalls, giftRepo.setRecipientID)
	}
}

func TestMiniappUpdateMyGiftRecipientIsOwnerScopedAndValidatesEvent(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	tests := []struct {
		name         string
		gift         *entity.Gift
		participants []*entity.Participant
		wantStatus   int
	}{
		{
			name:       "foreign gift is not found",
			gift:       &entity.Gift{ID: 9, UserID: 99, EventID: 77, ManualDistribution: true},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "gift from another event is not found",
			gift:       &entity.Gift{ID: 9, UserID: 42, EventID: 88, ManualDistribution: true},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "automatic gift conflicts",
			gift:       &entity.Gift{ID: 9, UserID: 42, EventID: 77},
			wantStatus: http.StatusConflict,
		},
		{
			name:         "cross event recipient conflicts",
			gift:         &entity.Gift{ID: 9, UserID: 42, EventID: 77, ManualDistribution: true},
			participants: []*entity.Participant{{ID: 11, EventID: 88}},
			wantStatus:   http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			giftRepo := &miniappHandlerGiftRepoFake{giftByID: tt.gift}
			h := newMiniappTestHandler(
				&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Active: true}},
				giftRepo,
				nil,
				&miniappHandlerParticipantRepoFake{participants: tt.participants},
			)
			rr := miniappUpdateRecipientRequest(t, token, now, h, `{"participant_id":11}`)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if giftRepo.setCalls != 0 {
				t.Fatalf("unexpected recipient write: %d", giftRepo.setCalls)
			}
		})
	}
}

func TestMiniappAssignRandomMyGiftRecipientUsesOnlyUnawardedParticipants(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	manualGift := &entity.Gift{ID: 9, UserID: 42, EventID: 77, ManualDistribution: true}
	giftRepo := &miniappHandlerGiftRepoFake{
		giftByID:              manualGift,
		manualRecipientCounts: map[uint]int{11: 1},
	}
	h := newMiniappTestHandler(
		&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Active: true}},
		giftRepo,
		nil,
		&miniappHandlerParticipantRepoFake{participants: []*entity.Participant{
			{ID: 11, EventID: 77},
			{ID: 12, EventID: 77},
		}},
	)

	rr := miniappAssignRandomRecipientRequest(t, token, now, h)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusNoContent, rr.Body.String())
	}
	if giftRepo.setCalls != 1 || giftRepo.setRecipientID == nil || *giftRepo.setRecipientID != 12 {
		t.Fatalf("random recipient write mismatch: calls=%d recipient=%v, want 12", giftRepo.setCalls, giftRepo.setRecipientID)
	}
}

func TestMiniappGiftsUsesActiveEventAndApprovedCatalog(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	giftRepo := &miniappHandlerGiftRepoFake{
		gifts: []*entity.Gift{
			{
				ID:             1,
				UserID:         123,
				EventID:        77,
				Description:    "Bottle cage",
				GenderFilter:   "male",
				BikeTypeFilter: "gravel",
				ReviewStatus:   entity.GiftReviewStatusApproved,
				User:           &entity.User{ID: 123, FirstName: "Alex"},
			},
			{
				ID:             2,
				UserID:         124,
				EventID:        77,
				Description:    "Women prize",
				GenderFilter:   "female",
				BikeTypeFilter: "gravel",
				ReviewStatus:   entity.GiftReviewStatusApproved,
				User:           &entity.User{ID: 124, FirstName: "Kate"},
			},
			{
				ID:             3,
				UserID:         125,
				EventID:        77,
				Description:    "Absolute prize",
				GenderFilter:   "all",
				BikeTypeFilter: "gravel",
				ReviewStatus:   entity.GiftReviewStatusApproved,
				User:           &entity.User{ID: 125, FirstName: "Sam"},
			},
		},
		attachments: map[uint][]*entity.GiftAttachment{
			1: {{ID: 10, GiftID: 1, TelegramFileID: "file-1", FileType: "photo"}},
		},
	}
	criteriaRepo := &miniappHandlerCriteriaRepoFake{
		criteriaByGift: map[uint][]*entity.Criteria{
			1: {{ID: 5, Name: "Speed"}},
		},
	}
	h := newMiniappTestHandler(
		&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Active: true}},
		giftRepo,
		criteriaRepo,
		&miniappHandlerParticipantRepoFake{
			participants: []*entity.Participant{
				{ID: 1, EventID: 77, Gender: valueobject.GenderMale, BikeType: valueobject.BikeTypeGravel},
				{ID: 2, EventID: 77, Gender: valueobject.GenderMale, BikeType: valueobject.BikeTypeMTB},
				{ID: 3, EventID: 77, Gender: valueobject.GenderFemale, BikeType: valueobject.BikeTypeGravel},
			},
		},
	)

	rr := miniappRequest(t, token, now, h.Gifts, "/api/miniapp/gifts?gender=male&bike_type=gravel")

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if !giftRepo.findByStatusCalled {
		t.Fatal("FindByEventAndReviewStatus was not called")
	}
	if giftRepo.eventID != 77 || giftRepo.reviewStatus != entity.GiftReviewStatusApproved {
		t.Fatalf("approved catalog query mismatch: event_id=%d review_status=%s", giftRepo.eventID, giftRepo.reviewStatus)
	}

	var got struct {
		Gifts []struct {
			ID          uint   `json:"id"`
			Description string `json:"description"`
			Attachments []struct {
				TelegramFileID string `json:"telegram_file_id"`
			} `json:"attachments"`
			Criteria []struct {
				ID uint `json:"id"`
			} `json:"criteria"`
		} `json:"gifts"`
		Total            int `json:"total"`
		ParticipantCount int `json:"participant_count"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 1 || len(got.Gifts) != 1 || got.Gifts[0].ID != 1 {
		t.Fatalf("gift list mismatch: %#v", got)
	}
	if got.ParticipantCount != 1 {
		t.Fatalf("participant count mismatch: got %d, want 1", got.ParticipantCount)
	}
	if len(got.Gifts[0].Attachments) != 1 || got.Gifts[0].Attachments[0].TelegramFileID != "file-1" {
		t.Fatalf("attachments mismatch: %#v", got.Gifts[0].Attachments)
	}
	if len(got.Gifts[0].Criteria) != 1 || got.Gifts[0].Criteria[0].ID != 5 {
		t.Fatalf("criteria mismatch: %#v", got.Gifts[0].Criteria)
	}
}

func TestMiniappGiftsRejectsInvalidFilters(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	giftRepo := &miniappHandlerGiftRepoFake{}
	h := newMiniappTestHandler(
		&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Active: true}},
		giftRepo,
		&miniappHandlerCriteriaRepoFake{},
		nil,
	)

	rr := miniappRequest(t, token, now, h.Gifts, "/api/miniapp/gifts?gender=everyone&bike_type=gravel")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if giftRepo.findByStatusCalled {
		t.Fatal("repository should not be called for invalid filters")
	}
}

func TestMiniappLeaderboardRanksFinishersAndListsOthers(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()

	link, err := valueobject.NewResultLink("https://www.strava.com/activities/123")
	if err != nil {
		t.Fatalf("build result link: %v", err)
	}

	participants := []*entity.Participant{
		{
			ID: 1, UserID: 111, EventID: 77,
			Gender: valueobject.GenderMale, BikeType: valueobject.BikeTypeGravel,
			Status: valueobject.ParticipantStatusActive,
			User:   &entity.User{ID: 111, FirstName: "Ivan", LastName: "Petrov"},
			Result: &entity.Result{
				ID: 1, ParticipantID: 1, IsCurrent: true, SubmittedAt: now,
				ResultLink:     link,
				ElapsedTimeSec: miniappIntPtr(25500), // 07:05:00
				MovingTimeSec:  miniappIntPtr(25138),
			},
		},
		{
			ID: 2, UserID: 222, EventID: 77,
			Gender: valueobject.GenderFemale, BikeType: valueobject.BikeTypeMTB,
			Status:             valueobject.ParticipantStatusActive,
			PrevElapsedTimeSec: miniappIntPtr(30000),
			User:               &entity.User{ID: 222, FirstName: "Anna", LastName: "K"},
			Result: &entity.Result{
				ID: 2, ParticipantID: 2, IsCurrent: true, SubmittedAt: now,
				ElapsedTimeSec: miniappIntPtr(28800), // 08:00:00
				MovingTimeSec:  miniappIntPtr(28000),
			},
		},
		{
			// Сошёл с дистанции: есть результат, но в зачёт не идёт — без места.
			ID: 3, UserID: 333, EventID: 77,
			Gender: valueobject.GenderMale, BikeType: valueobject.BikeTypeGravel,
			Status: valueobject.ParticipantStatusDNF,
			User:   &entity.User{ID: 333, FirstName: "Max", LastName: "D"},
			Result: &entity.Result{
				ID: 3, ParticipantID: 3, IsCurrent: true, SubmittedAt: now,
				ElapsedTimeSec: miniappIntPtr(20000),
			},
		},
		{
			// Не финишировал: результата нет — без места.
			ID: 4, UserID: 444, EventID: 77,
			Gender: valueobject.GenderMale, BikeType: valueobject.BikeTypeRoad,
			Status: valueobject.ParticipantStatusActive,
			User:   &entity.User{ID: 444, FirstName: "Oleg", LastName: "R"},
		},
	}

	h := newMiniappTestHandler(
		&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Active: true}},
		nil, nil,
		&miniappHandlerParticipantRepoFake{participants: participants},
	)
	h.resultRepo = &miniappResultRepoFake{prevElapsedByUser: map[int64]int{111: 26400, 222: 28000}}

	rr := miniappRequest(t, token, now, h.Leaderboard, "/api/miniapp/leaderboard")

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var got struct {
		Participants []struct {
			ID                  uint    `json:"id"`
			Name                string  `json:"name"`
			Gender              string  `json:"gender"`
			BikeType            string  `json:"bike_type"`
			Status              string  `json:"status"`
			IsFinished          bool    `json:"is_finished"`
			Place               int     `json:"place"`
			ElapsedTime         *string `json:"elapsed_time"`
			MovingTime          *string `json:"moving_time"`
			ResultLink          *string `json:"result_link"`
			PrevElapsedDelta    *string `json:"prev_elapsed_delta"`
			PrevElapsedDeltaSec *int    `json:"prev_elapsed_delta_sec"`
		} `json:"participants"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Участник без результата (#4, Oleg) исключён; остаются два финишировавших
	// и DNF с результатом (#3).
	if got.Total != 3 || len(got.Participants) != 3 {
		t.Fatalf("expected 3 participants, got total=%d len=%d", got.Total, len(got.Participants))
	}
	for _, p := range got.Participants {
		if p.Name == "Oleg R" {
			t.Fatalf("participant without result must be excluded from leaderboard: %#v", p)
		}
	}

	// Финишировавшие идут первыми, отсортированы по общему времени.
	if got.Participants[0].Place != 1 || got.Participants[0].Name != "Ivan Petrov" {
		t.Fatalf("first place mismatch: %#v", got.Participants[0])
	}
	if got.Participants[0].ElapsedTime == nil || *got.Participants[0].ElapsedTime != "07:05:00" {
		t.Fatalf("first place elapsed time mismatch: %#v", got.Participants[0].ElapsedTime)
	}
	if got.Participants[0].ResultLink == nil || *got.Participants[0].ResultLink != "https://www.strava.com/activities/123" {
		t.Fatalf("first place result link mismatch: %#v", got.Participants[0].ResultLink)
	}
	if got.Participants[1].Place != 2 || got.Participants[1].Name != "Anna K" {
		t.Fatalf("second place mismatch: %#v", got.Participants[1])
	}
	if got.Participants[0].PrevElapsedDelta == nil || *got.Participants[0].PrevElapsedDelta != "+00:15:00" || got.Participants[0].PrevElapsedDeltaSec == nil || *got.Participants[0].PrevElapsedDeltaSec != 900 {
		t.Fatalf("automatic previous event delta mismatch: %#v", got.Participants[0])
	}
	if got.Participants[1].PrevElapsedDelta == nil || *got.Participants[1].PrevElapsedDelta != "+00:20:00" || got.Participants[1].PrevElapsedDeltaSec == nil || *got.Participants[1].PrevElapsedDeltaSec != 1200 {
		t.Fatalf("manual previous event delta must take priority: %#v", got.Participants[1])
	}

	// DNF с результатом остаётся, но без места (0).
	if got.Participants[2].Place != 0 || got.Participants[2].Name != "Max D" {
		t.Fatalf("DNF-with-result participant mismatch: %#v", got.Participants[2])
	}

	// Публичный DTO не должен раскрывать административные/приватные поля.
	body := rr.Body.String()
	for _, forbidden := range []string{`"user_id"`, `"notes"`, `"has_gift"`, `"registered_at"`, `"prizes"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("leaderboard response leaked field %s: %s", forbidden, body)
		}
	}
}

func TestMiniappLeaderboardReturnsNotFoundWhenNoActiveEvent(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	h := newMiniappTestHandler(&miniappEventRepoFake{}, nil, nil, nil)

	rr := miniappRequest(t, token, now, h.Leaderboard, "/api/miniapp/leaderboard")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
	}
}

func TestMiniappTelegramFileStreamsContent(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	h := newMiniappTestHandler(
		&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Active: true}},
		nil,
		nil,
		nil,
	)
	h.fileFetcher = miniappFileFetcherFunc(func(ctx context.Context, fileID string) (*http.Response, error) {
		if fileID != "photo-1" {
			t.Fatalf("file ID mismatch: got %q", fileID)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type":   {"image/jpeg"},
				"Content-Length": {"5"},
			},
			Body: io.NopCloser(strings.NewReader("image")),
		}, nil
	})

	router := chi.NewRouter()
	router.Use(middleware.TelegramWebAppAuthWithConfig(middleware.TelegramWebAppAuthConfig{
		BotToken: token,
		Now:      func() time.Time { return now },
	}))
	router.Get("/api/miniapp/telegram/files/{fileId}", h.TelegramFile)

	req := httptest.NewRequest(http.MethodGet, "/api/miniapp/telegram/files/photo-1", nil)
	req.Header.Set(middleware.TelegramInitDataHeader, signedMiniappInitData(t, token, now))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("content type mismatch: got %q", got)
	}
	if got := rr.Body.String(); got != "image" {
		t.Fatalf("body mismatch: got %q", got)
	}
}

func newMiniappTestHandler(
	eventRepo *miniappEventRepoFake,
	giftRepo *miniappHandlerGiftRepoFake,
	criteriaRepo *miniappHandlerCriteriaRepoFake,
	participantRepo *miniappHandlerParticipantRepoFake,
) *MiniappHandler {
	if giftRepo == nil {
		giftRepo = &miniappHandlerGiftRepoFake{}
	}
	if criteriaRepo == nil {
		criteriaRepo = &miniappHandlerCriteriaRepoFake{}
	}
	if participantRepo == nil {
		participantRepo = &miniappHandlerParticipantRepoFake{}
	}

	handler := newMiniappHandlerWithFileFetcher(
		eventRepo,
		query.NewGetMiniappGiftsHandler(giftRepo, criteriaRepo),
		query.NewGetMiniappParticipantCountHandler(participantRepo),
		query.NewGetParticipantsHandler(participantRepo),
		query.NewGetParticipantByUserAndEventHandler(participantRepo),
		&miniappResultRepoFake{},
		miniappFileFetcherFunc(func(ctx context.Context, fileID string) (*http.Response, error) {
			return nil, fmt.Errorf("unexpected file fetch: %s", fileID)
		}),
		nil,
	)
	participantOptionsHandler := query.NewGetMiniappParticipantsHandler(participantRepo, giftRepo, &miniappPrizeDistributionReaderFake{})
	setRecipientHandler := command.NewSetManualGiftRecipientHandler(giftRepo, participantRepo)
	handler.ConfigureManualGiftManagement(
		query.NewGetOwnerManualGiftsHandler(giftRepo, criteriaRepo),
		query.NewHasOwnerGiftsHandler(giftRepo),
		participantOptionsHandler,
		setRecipientHandler,
		command.NewAssignRandomManualGiftRecipientHandler(participantOptionsHandler, setRecipientHandler),
	)
	return handler
}

func miniappRequest(t *testing.T, token string, now time.Time, handlerFunc http.HandlerFunc, target string) *httptest.ResponseRecorder {
	return miniappRequestWithMethod(t, token, now, http.MethodGet, handlerFunc, target, nil)
}

func miniappRequestWithMethod(t *testing.T, token string, now time.Time, method string, handlerFunc http.HandlerFunc, target string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()

	handler := middleware.TelegramWebAppAuthWithConfig(middleware.TelegramWebAppAuthConfig{
		BotToken: token,
		Now:      func() time.Time { return now },
	})(handlerFunc)

	req := httptest.NewRequest(method, target, body)
	req.Header.Set(middleware.TelegramInitDataHeader, signedMiniappInitData(t, token, now))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func miniappUpdateRecipientRequest(t *testing.T, token string, now time.Time, h *MiniappHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	return miniappRequestWithMethod(t, token, now, http.MethodPut, func(w http.ResponseWriter, r *http.Request) {
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("giftId", "9")
		h.UpdateMyGiftRecipient(w, r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeContext)))
	}, "/api/miniapp/my-gifts/9/recipient", strings.NewReader(body))
}

func miniappAssignRandomRecipientRequest(t *testing.T, token string, now time.Time, h *MiniappHandler) *httptest.ResponseRecorder {
	t.Helper()
	return miniappRequestWithMethod(t, token, now, http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("giftId", "9")
		h.AssignRandomMyGiftRecipient(w, r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeContext)))
	}, "/api/miniapp/my-gifts/9/random-recipient", nil)
}

func miniappIntPtr(v int) *int { return &v }

type miniappFileFetcherFunc func(ctx context.Context, fileID string) (*http.Response, error)

func (f miniappFileFetcherFunc) Fetch(ctx context.Context, fileID string) (*http.Response, error) {
	return f(ctx, fileID)
}

type miniappEventRepoFake struct {
	activeEvent *entity.Event
	activeErr   error
}

func (r *miniappEventRepoFake) Create(ctx context.Context, event *entity.Event) error { return nil }
func (r *miniappEventRepoFake) Update(ctx context.Context, event *entity.Event) error { return nil }
func (r *miniappEventRepoFake) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	return nil, nil
}
func (r *miniappEventRepoFake) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	return nil, nil
}
func (r *miniappEventRepoFake) FindActive(ctx context.Context) (*entity.Event, error) {
	return r.activeEvent, r.activeErr
}
func (r *miniappEventRepoFake) GetAll(ctx context.Context) ([]*entity.Event, error) {
	return nil, nil
}
func (r *miniappEventRepoFake) Delete(ctx context.Context, id uint) error { return nil }

type miniappHandlerGiftRepoFake struct {
	findByStatusCalled       bool
	eventID                  uint
	reviewStatus             entity.GiftReviewStatus
	gifts                    []*entity.Gift
	attachments              map[uint][]*entity.GiftAttachment
	ownerGifts               []*entity.Gift
	giftByID                 *entity.Gift
	findByIDErr              error
	setRecipientID           *uint
	setRecipientErr          error
	setCalls                 int
	hasOwnerGifts            bool
	hasOwnerGiftsErr         error
	manualRecipientCounts    map[uint]int
	manualRecipientCountsErr error
}

func (r *miniappHandlerGiftRepoFake) Create(ctx context.Context, gift *entity.Gift) error {
	return nil
}
func (r *miniappHandlerGiftRepoFake) CreateWithAttachments(ctx context.Context, gift *entity.Gift, attachments []*entity.GiftAttachment) error {
	return nil
}
func (r *miniappHandlerGiftRepoFake) Update(ctx context.Context, gift *entity.Gift) error {
	return nil
}
func (r *miniappHandlerGiftRepoFake) UpdateWithCriteria(ctx context.Context, gift *entity.Gift, criteriaIDs []uint) error {
	return nil
}
func (r *miniappHandlerGiftRepoFake) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	if r.findByIDErr != nil {
		return nil, r.findByIDErr
	}
	if r.giftByID != nil && r.giftByID.ID == id {
		return r.giftByID, nil
	}
	for _, gift := range r.gifts {
		if gift.ID == id {
			return gift, nil
		}
	}
	return nil, repository.ErrGiftNotFound
}
func (r *miniappHandlerGiftRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *miniappHandlerGiftRepoFake) FindByEventAndReviewStatus(ctx context.Context, eventID uint, reviewStatus entity.GiftReviewStatus) ([]*entity.Gift, error) {
	r.findByStatusCalled = true
	r.eventID = eventID
	r.reviewStatus = reviewStatus
	return r.gifts, nil
}
func (r *miniappHandlerGiftRepoFake) ListByEventPaged(ctx context.Context, eventID uint, reviewStatus *entity.GiftReviewStatus, limit, offset int) ([]*entity.Gift, int, error) {
	r.findByStatusCalled = true
	r.eventID = eventID
	if reviewStatus != nil {
		r.reviewStatus = *reviewStatus
	}
	return r.gifts, len(r.gifts), nil
}
func (r *miniappHandlerGiftRepoFake) CountsByReviewStatus(ctx context.Context, eventID uint) (map[string]int, error) {
	return nil, nil
}
func (r *miniappHandlerGiftRepoFake) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *miniappHandlerGiftRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) ([]*entity.Gift, error) {
	if r.ownerGifts != nil {
		return r.ownerGifts, nil
	}
	matched := make([]*entity.Gift, 0)
	for _, gift := range r.gifts {
		if gift.UserID == userID && gift.EventID == eventID {
			matched = append(matched, gift)
		}
	}
	return matched, nil
}
func (r *miniappHandlerGiftRepoFake) HasByUserAndEvent(ctx context.Context, userID int64, eventID uint) (bool, error) {
	return r.hasOwnerGifts, r.hasOwnerGiftsErr
}
func (r *miniappHandlerGiftRepoFake) SetManualRecipient(ctx context.Context, giftID uint, recipientParticipantID *uint) error {
	r.setCalls++
	if r.setRecipientErr != nil {
		return r.setRecipientErr
	}
	r.setRecipientID = recipientParticipantID
	if r.giftByID != nil && r.giftByID.ID == giftID {
		r.giftByID.ManualRecipientParticipantID = recipientParticipantID
	}
	return nil
}
func (r *miniappHandlerGiftRepoFake) ManualRecipientCountsByEvent(ctx context.Context, eventID uint) (map[uint]int, error) {
	return r.manualRecipientCounts, r.manualRecipientCountsErr
}

type miniappPrizeDistributionReaderFake struct {
	results []*query.PrizeDistributionResult
	err     error
}

func (r *miniappPrizeDistributionReaderFake) Handle(context.Context, query.GetPrizeDistributionQuery) ([]*query.PrizeDistributionResult, error) {
	return r.results, r.err
}
func (r *miniappHandlerGiftRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *miniappHandlerGiftRepoFake) AddAttachment(ctx context.Context, attachment *entity.GiftAttachment) error {
	return nil
}
func (r *miniappHandlerGiftRepoFake) GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error) {
	if r.attachments == nil {
		return nil, nil
	}
	return r.attachments[giftID], nil
}

type miniappHandlerCriteriaRepoFake struct {
	criteriaByGift map[uint][]*entity.Criteria
}

func (r *miniappHandlerCriteriaRepoFake) Create(ctx context.Context, criteria *entity.Criteria) error {
	return nil
}
func (r *miniappHandlerCriteriaRepoFake) Update(ctx context.Context, criteria *entity.Criteria) error {
	return nil
}
func (r *miniappHandlerCriteriaRepoFake) Delete(ctx context.Context, id uint) error {
	return nil
}
func (r *miniappHandlerCriteriaRepoFake) FindByID(ctx context.Context, id uint) (*entity.Criteria, error) {
	return nil, nil
}
func (r *miniappHandlerCriteriaRepoFake) FindAll(ctx context.Context) ([]*entity.Criteria, error) {
	return nil, nil
}
func (r *miniappHandlerCriteriaRepoFake) FindByType(ctx context.Context, criteriaType valueobject.CriteriaType) ([]*entity.Criteria, error) {
	return nil, nil
}
func (r *miniappHandlerCriteriaRepoFake) ListPaged(ctx context.Context, criteriaType *valueobject.CriteriaType, limit, offset int) ([]*entity.Criteria, int, error) {
	return nil, 0, nil
}
func (r *miniappHandlerCriteriaRepoFake) FindByGift(ctx context.Context, giftID uint) ([]*entity.Criteria, error) {
	if r.criteriaByGift == nil {
		return nil, nil
	}
	return r.criteriaByGift[giftID], nil
}
func (r *miniappHandlerCriteriaRepoFake) FindByResult(ctx context.Context, resultID uint) ([]*entity.Criteria, error) {
	return nil, nil
}

type miniappHandlerParticipantRepoFake struct {
	participants          []*entity.Participant
	eventID               uint
	findByIDErr           error
	findByUserAndEventErr error
}

type miniappResultRepoFake struct {
	repository.ResultRepository
	prevElapsedByUser map[int64]int
}

func (r *miniappResultRepoFake) FindPrevEventElapsedByUser(ctx context.Context, eventID uint) (map[int64]int, error) {
	return r.prevElapsedByUser, nil
}

func (r *miniappHandlerParticipantRepoFake) Create(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *miniappHandlerParticipantRepoFake) Update(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *miniappHandlerParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	if r.findByIDErr != nil {
		return nil, r.findByIDErr
	}
	for _, participant := range r.participants {
		if participant.ID == id {
			return participant, nil
		}
	}
	return nil, repository.ErrParticipantNotFound
}
func (r *miniappHandlerParticipantRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error) {
	if r.findByUserAndEventErr != nil {
		return nil, r.findByUserAndEventErr
	}
	for _, participant := range r.participants {
		if participant.UserID == userID && participant.EventID == eventID {
			return participant, nil
		}
	}
	return nil, repository.ErrParticipantNotFound
}
func (r *miniappHandlerParticipantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	r.eventID = eventID
	return r.participants, nil
}
func (r *miniappHandlerParticipantRepoFake) UpdateNotes(ctx context.Context, id uint, notes string) error {
	return nil
}
func (r *miniappHandlerParticipantRepoFake) Delete(ctx context.Context, id uint) error {
	return nil
}
func (r *miniappHandlerParticipantRepoFake) DeleteWithResultCriteria(ctx context.Context, id uint) error {
	return nil
}
func (r *miniappHandlerParticipantRepoFake) GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}

func signedMiniappInitData(t *testing.T, token string, now time.Time) string {
	t.Helper()

	values := url.Values{
		"auth_date": {strconv.FormatInt(now.Unix(), 10)},
		"query_id":  {"query-1"},
		"user":      {`{"id":42,"first_name":"Alex","username":"alex"}`},
	}
	values.Set("hash", miniappHash(token, values))
	return values.Encode()
}

func miniappHash(token string, values url.Values) string {
	pairs := make([]string, 0, len(values))
	for key, value := range values {
		if key == "hash" {
			continue
		}
		pairs = append(pairs, key+"="+value[0])
	}
	sort.Strings(pairs)

	secretHMAC := hmac.New(sha256.New, []byte("WebAppData"))
	secretHMAC.Write([]byte(token))
	secret := secretHMAC.Sum(nil)

	dataHMAC := hmac.New(sha256.New, secret)
	dataHMAC.Write([]byte(strings.Join(pairs, "\n")))
	return fmt.Sprintf("%x", dataHMAC.Sum(nil))
}

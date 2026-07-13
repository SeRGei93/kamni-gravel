package query

import (
	"context"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"
)

func TestGetNotificationRecipientsFilters(t *testing.T) {
	participants := []*entity.Participant{
		{UserID: 1, User: &entity.User{FirstName: "Anna", Username: "anna"}, Result: &entity.Result{ID: 1}},
		{UserID: 2, User: &entity.User{FirstName: "Boris"}, Result: &entity.Result{ID: 2}},
		{UserID: 3, User: &entity.User{FirstName: "Cesar"}},
		{UserID: 4, User: &entity.User{FirstName: "Dana"}, Result: &entity.Result{ID: 4}},
		{UserID: 5, User: &entity.User{FirstName: "Eva"}, Status: valueobject.ParticipantStatusDNF, Result: &entity.Result{ID: 5}},
		{UserID: 6, User: &entity.User{FirstName: "Fedor"}},
		{UserID: 7, User: &entity.User{FirstName: "Gleb"}, Status: valueobject.ParticipantStatusDisqualified},
	}
	gifts := []*entity.Gift{
		{ID: 1, UserID: 2},
		{ID: 2, UserID: 3},
		{ID: 3, UserID: 4},
	}
	distribution := &notificationDistributionFake{
		output: &PrizeDistributionOutput{
			UnassignedSlots: []*UnassignedPrizeSlot{{GiftID: 3}},
		},
	}
	handler := NewGetNotificationRecipientsHandler(
		&purgeParticipantRepoFake{participants: participants},
		&purgeGiftRepoFake{gifts: gifts},
		distribution,
	)

	tests := []struct {
		name     string
		filter   NotificationRecipientFilter
		wantIDs  []int64
		wantCall int
	}{
		{name: "all", filter: NotificationRecipientFilterAll, wantIDs: []int64{1, 2, 3, 4, 5, 6, 7}},
		{name: "finished without gift", filter: NotificationRecipientFilterFinishedWithoutGift, wantIDs: []int64{1}},
		{name: "gift without finish", filter: NotificationRecipientFilterGiftWithoutFinish, wantIDs: []int64{3}},
		{name: "unassigned gifts", filter: NotificationRecipientFilterUnassignedGifts, wantIDs: []int64{4}, wantCall: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			distribution.calls = 0
			recipients, err := handler.Handle(context.Background(), 77, test.filter)
			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			gotIDs := make([]int64, 0, len(recipients))
			for _, recipient := range recipients {
				gotIDs = append(gotIDs, recipient.UserID)
			}
			assertNotificationRecipientIDs(t, gotIDs, test.wantIDs)
			if distribution.calls != test.wantCall {
				t.Fatalf("distribution calls = %d, want %d", distribution.calls, test.wantCall)
			}
		})
	}
}

func TestNotificationRecipientFilterValidation(t *testing.T) {
	filter, err := NewNotificationRecipientFilter("")
	if err != nil {
		t.Fatalf("NewNotificationRecipientFilter() error = %v", err)
	}
	if filter != NotificationRecipientFilterAll {
		t.Fatalf("default filter = %q, want %q", filter, NotificationRecipientFilterAll)
	}

	if _, err := NewNotificationRecipientFilter("unknown"); err == nil {
		t.Fatal("NewNotificationRecipientFilter() must reject an unknown filter")
	}
}

func assertNotificationRecipientIDs(t *testing.T, got, want []int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("recipient count = %d, want %d; got=%v", len(got), len(want), got)
	}
	for index, expected := range want {
		if got[index] != expected {
			t.Fatalf("recipient at index %d = %d, want %d; got=%v", index, got[index], expected, got)
		}
	}
}

type notificationDistributionFake struct {
	output *PrizeDistributionOutput
	err    error
	calls  int
}

func (f *notificationDistributionFake) HandleDetailed(ctx context.Context, query GetPrizeDistributionQuery) (*PrizeDistributionOutput, error) {
	f.calls++
	return f.output, f.err
}

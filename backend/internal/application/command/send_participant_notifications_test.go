package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
)

func TestSendParticipantNotificationsOnlyDeliversToEventParticipants(t *testing.T) {
	notifier := &participantNotifierFake{failing: map[int64]bool{2: true}}
	handler := NewSendParticipantNotificationsHandler(
		&notificationParticipantRepoFake{participants: []*entity.Participant{{UserID: 1}, {UserID: 2}}},
		notifier,
	)
	handler.sleep = noSleep

	result, err := handler.Handle(context.Background(), SendParticipantNotificationsCommand{
		EventID: 77,
		UserIDs: []int64{1, 99, 1, 2},
		Text:    "Сообщение",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if result.Sent != 1 || result.Failed != 1 || result.Skipped != 2 {
		t.Fatalf("result = %+v, want sent=1 failed=1 skipped=2", result)
	}
	if len(notifier.sent) != 1 || notifier.sent[0] != 1 {
		t.Fatalf("notifier sent = %v, want only [1]", notifier.sent)
	}
}

func TestSendParticipantNotificationsValidatesText(t *testing.T) {
	handler := NewSendParticipantNotificationsHandler(
		&notificationParticipantRepoFake{},
		&participantNotifierFake{},
	)

	if _, err := handler.Handle(context.Background(), SendParticipantNotificationsCommand{Text: "   "}); !errors.Is(err, ErrParticipantNotificationTextEmpty) {
		t.Fatalf("empty text error = %v, want %v", err, ErrParticipantNotificationTextEmpty)
	}

	tooLong := make([]rune, maxParticipantNotificationTextLength+1)
	for index := range tooLong {
		tooLong[index] = 'я'
	}
	if _, err := handler.Handle(context.Background(), SendParticipantNotificationsCommand{Text: string(tooLong)}); !errors.Is(err, ErrParticipantNotificationTextTooLong) {
		t.Fatalf("too long text error = %v, want %v", err, ErrParticipantNotificationTextTooLong)
	}
}

func TestParticipantNotificationJobManagerDeliversQueuedJobInBackground(t *testing.T) {
	notifier := &participantNotifierFake{}
	sendHandler := NewSendParticipantNotificationsHandler(
		&notificationParticipantRepoFake{participants: []*entity.Participant{{UserID: 1}, {UserID: 2}}},
		notifier,
	)
	sendHandler.sleep = noSleep
	manager := NewParticipantNotificationJobManager(sendHandler)
	manager.newID = func() (string, error) { return "job-1", nil }

	queued, err := manager.Submit(SendParticipantNotificationsCommand{
		EventID: 77,
		UserIDs: []int64{1, 2},
		Text:    "Сообщение",
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if queued.Status != ParticipantNotificationJobQueued {
		t.Fatalf("queued status = %q, want %q", queued.Status, ParticipantNotificationJobQueued)
	}
	if len(notifier.sent) != 0 {
		t.Fatalf("notification must not be delivered before worker starts, sent=%v", notifier.sent)
	}

	ctx, cancel := context.WithCancel(context.Background())
	workerDone := make(chan struct{})
	go func() {
		manager.Run(ctx)
		close(workerDone)
	}()
	defer func() {
		cancel()
		<-workerDone
	}()

	completed := waitForParticipantNotificationJob(t, manager, queued.ID, ParticipantNotificationJobCompleted)
	if completed.Result.Sent != 2 || completed.Result.Failed != 0 || completed.Result.Skipped != 0 {
		t.Fatalf("job result = %+v, want sent=2 failed=0 skipped=0", completed.Result)
	}
	if len(notifier.sent) != 2 {
		t.Fatalf("notifier sent = %v, want two recipients", notifier.sent)
	}
}

func waitForParticipantNotificationJob(
	t *testing.T,
	manager *ParticipantNotificationJobManager,
	jobID string,
	wantStatus ParticipantNotificationJobStatus,
) ParticipantNotificationJob {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		job, ok := manager.Get(jobID)
		if ok && job.Status == wantStatus {
			return job
		}
		time.Sleep(time.Millisecond)
	}
	job, _ := manager.Get(jobID)
	t.Fatalf("job status = %q, want %q", job.Status, wantStatus)
	return ParticipantNotificationJob{}
}

type participantNotifierFake struct {
	sent    []int64
	failing map[int64]bool
}

func (f *participantNotifierFake) Send(ctx context.Context, userID int64, text string) error {
	if f.failing[userID] {
		return errors.New("telegram unavailable")
	}
	f.sent = append(f.sent, userID)
	return nil
}

type notificationParticipantRepoFake struct {
	participants []*entity.Participant
}

func (r *notificationParticipantRepoFake) Create(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *notificationParticipantRepoFake) Update(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *notificationParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	return nil, nil
}
func (r *notificationParticipantRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error) {
	return nil, nil
}
func (r *notificationParticipantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return r.participants, nil
}
func (r *notificationParticipantRepoFake) UpdateNotes(ctx context.Context, id uint, notes string) error {
	return nil
}
func (r *notificationParticipantRepoFake) Delete(ctx context.Context, id uint) error {
	return nil
}
func (r *notificationParticipantRepoFake) DeleteWithResultCriteria(ctx context.Context, id uint) error {
	return nil
}
func (r *notificationParticipantRepoFake) GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}

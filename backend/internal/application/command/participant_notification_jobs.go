package command

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const participantNotificationQueueCapacity = 32

var (
	// ErrParticipantNotificationQueueFull означает, что worker ещё не обработал
	// предыдущие рассылки и не может принять новую без риска перегрузки Telegram.
	ErrParticipantNotificationQueueFull = errors.New("participant notification queue is full")
)

// ParticipantNotificationJobStatus описывает состояние фоновой рассылки.
type ParticipantNotificationJobStatus string

const (
	ParticipantNotificationJobQueued    ParticipantNotificationJobStatus = "queued"
	ParticipantNotificationJobRunning   ParticipantNotificationJobStatus = "running"
	ParticipantNotificationJobCompleted ParticipantNotificationJobStatus = "completed"
	ParticipantNotificationJobFailed    ParticipantNotificationJobStatus = "failed"
	ParticipantNotificationJobCancelled ParticipantNotificationJobStatus = "cancelled"
)

// ParticipantNotificationJob — безопасный для API снимок состояния рассылки.
type ParticipantNotificationJob struct {
	ID         string
	Status     ParticipantNotificationJobStatus
	Requested  int
	Result     SendParticipantNotificationsResult
	Error      string
	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
}

type participantNotificationJobState struct {
	job     ParticipantNotificationJob
	command SendParticipantNotificationsCommand
}

// ParticipantNotificationJobManager последовательно отправляет рассылки в
// фоне. Один worker сохраняет лимит Telegram для всех админов и не держит HTTP
// запрос открытым на время доставки.
type ParticipantNotificationJobManager struct {
	sendHandler *SendParticipantNotificationsHandler

	mu    sync.RWMutex
	jobs  map[string]*participantNotificationJobState
	queue chan string
	now   func() time.Time
	newID func() (string, error)
}

// NewParticipantNotificationJobManager создаёт очередь фоновой рассылки.
func NewParticipantNotificationJobManager(sendHandler *SendParticipantNotificationsHandler) *ParticipantNotificationJobManager {
	return &ParticipantNotificationJobManager{
		sendHandler: sendHandler,
		jobs:        make(map[string]*participantNotificationJobState),
		queue:       make(chan string, participantNotificationQueueCapacity),
		now:         time.Now,
		newID:       newParticipantNotificationJobID,
	}
}

// Submit проверяет запрос и ставит рассылку в очередь. Доставка выполняется
// асинхронно после возврата HTTP-ответа.
func (m *ParticipantNotificationJobManager) Submit(cmd SendParticipantNotificationsCommand) (ParticipantNotificationJob, error) {
	var empty ParticipantNotificationJob
	if m == nil || m.sendHandler == nil {
		return empty, ErrParticipantNotificationsNotConfigured
	}

	text, err := m.sendHandler.validate(cmd.Text)
	if err != nil {
		return empty, err
	}
	cmd.Text = text

	jobID, err := m.newID()
	if err != nil {
		return empty, fmt.Errorf("create participant notification job id: %w", err)
	}
	now := m.now().UTC()
	state := &participantNotificationJobState{
		job: ParticipantNotificationJob{
			ID:        jobID,
			Status:    ParticipantNotificationJobQueued,
			Requested: len(cmd.UserIDs),
			CreatedAt: now,
		},
		command: cmd,
	}

	m.mu.Lock()
	m.jobs[jobID] = state
	m.mu.Unlock()

	select {
	case m.queue <- jobID:
		log.Printf("INFO [FIX:participant-notification-queue] job queued: job_id=%s event_id=%d requested=%d text_length=%d", jobID, cmd.EventID, len(cmd.UserIDs), utf8.RuneCountInString(cmd.Text))
		job, ok := m.Get(jobID)
		if !ok {
			return empty, fmt.Errorf("queued participant notification job %s is unavailable", jobID)
		}
		return job, nil
	default:
		m.mu.Lock()
		delete(m.jobs, jobID)
		m.mu.Unlock()
		log.Printf("WARN [FIX:participant-notification-queue] job rejected: event_id=%d requested=%d reason=queue_full", cmd.EventID, len(cmd.UserIDs))
		return empty, ErrParticipantNotificationQueueFull
	}
}

// Get возвращает текущий снимок задачи по её ID.
func (m *ParticipantNotificationJobManager) Get(jobID string) (ParticipantNotificationJob, bool) {
	if m == nil {
		return ParticipantNotificationJob{}, false
	}
	m.mu.RLock()
	state, ok := m.jobs[jobID]
	if !ok {
		m.mu.RUnlock()
		return ParticipantNotificationJob{}, false
	}
	job := cloneParticipantNotificationJob(state.job)
	m.mu.RUnlock()
	return job, true
}

// Run запускает единственный worker очереди. Метод завершается при отмене ctx.
func (m *ParticipantNotificationJobManager) Run(ctx context.Context) {
	if m == nil {
		return
	}
	log.Printf("INFO [FIX:participant-notification-queue] worker started")
	defer log.Printf("INFO [FIX:participant-notification-queue] worker stopped: error=%v", ctx.Err())

	for {
		select {
		case <-ctx.Done():
			return
		case jobID := <-m.queue:
			m.deliver(ctx, jobID)
		}
	}
}

func (m *ParticipantNotificationJobManager) deliver(ctx context.Context, jobID string) {
	m.mu.Lock()
	state, ok := m.jobs[jobID]
	if !ok {
		m.mu.Unlock()
		return
	}
	now := m.now().UTC()
	state.job.Status = ParticipantNotificationJobRunning
	state.job.StartedAt = &now
	cmd := state.command
	m.mu.Unlock()

	log.Printf("INFO [FIX:participant-notification-queue] job started: job_id=%s event_id=%d requested=%d", jobID, cmd.EventID, len(cmd.UserIDs))
	result, err := m.sendHandler.Handle(ctx, cmd)

	m.mu.Lock()
	state, ok = m.jobs[jobID]
	if !ok {
		m.mu.Unlock()
		return
	}
	finishedAt := m.now().UTC()
	state.job.Result = result
	state.job.FinishedAt = &finishedAt
	switch {
	case errors.Is(err, context.Canceled):
		state.job.Status = ParticipantNotificationJobCancelled
		state.job.Error = "Рассылка остановлена при завершении сервера"
	case err != nil:
		state.job.Status = ParticipantNotificationJobFailed
		state.job.Error = "Не удалось выполнить рассылку"
	default:
		state.job.Status = ParticipantNotificationJobCompleted
	}
	job := cloneParticipantNotificationJob(state.job)
	m.mu.Unlock()

	if err != nil {
		log.Printf("WARN [FIX:participant-notification-queue] job finished with error: job_id=%s event_id=%d status=%s sent=%d failed=%d skipped=%d error=%v", job.ID, cmd.EventID, job.Status, job.Result.Sent, job.Result.Failed, job.Result.Skipped, err)
		return
	}
	log.Printf("INFO [FIX:participant-notification-queue] job completed: job_id=%s event_id=%d sent=%d failed=%d skipped=%d", job.ID, cmd.EventID, job.Result.Sent, job.Result.Failed, job.Result.Skipped)
}

func normalizeParticipantNotificationText(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ErrParticipantNotificationTextEmpty
	}
	if utf8.RuneCountInString(text) > maxParticipantNotificationTextLength {
		return "", ErrParticipantNotificationTextTooLong
	}
	return text, nil
}

func cloneParticipantNotificationJob(job ParticipantNotificationJob) ParticipantNotificationJob {
	if job.StartedAt != nil {
		startedAt := *job.StartedAt
		job.StartedAt = &startedAt
	}
	if job.FinishedAt != nil {
		finishedAt := *job.FinishedAt
		job.FinishedAt = &finishedAt
	}
	return job
}

func newParticipantNotificationJobID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

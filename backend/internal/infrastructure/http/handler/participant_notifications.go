package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/http/response"
)

const maxParticipantNotificationRequestSize = 1 << 20

// ParticipantNotificationsHandler обслуживает выбор и рассылку уведомлений
// участникам активного события из админ-панели.
type ParticipantNotificationsHandler struct {
	eventRepo         repository.EventRepository
	recipientsHandler *query.GetNotificationRecipientsHandler
	jobManager        *command.ParticipantNotificationJobManager
}

// NewParticipantNotificationsHandler создаёт HTTP handler админской рассылки.
func NewParticipantNotificationsHandler(
	eventRepo repository.EventRepository,
	recipientsHandler *query.GetNotificationRecipientsHandler,
	jobManager *command.ParticipantNotificationJobManager,
) *ParticipantNotificationsHandler {
	return &ParticipantNotificationsHandler{
		eventRepo:         eventRepo,
		recipientsHandler: recipientsHandler,
		jobManager:        jobManager,
	}
}

type participantNotificationRecipientResponse struct {
	UserID             int64  `json:"user_id"`
	Label              string `json:"label"`
	Username           string `json:"username,omitempty"`
	Status             string `json:"status"`
	HasGift            bool   `json:"has_gift"`
	HasUnassignedGifts bool   `json:"has_unassigned_gifts"`
}

type participantNotificationRecipientsResponse struct {
	EventName  string                                     `json:"event_name"`
	Filter     string                                     `json:"filter"`
	Recipients []participantNotificationRecipientResponse `json:"recipients"`
}

type sendParticipantNotificationsRequest struct {
	UserIDs []int64 `json:"user_ids"`
	Text    string  `json:"text"`
}

type participantNotificationJobResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Requested int    `json:"requested"`
	Sent      int    `json:"sent"`
	Failed    int    `json:"failed"`
	Skipped   int    `json:"skipped"`
	Error     string `json:"error,omitempty"`
}

// Recipients обрабатывает GET /api/participant-notifications/recipients.
func (h *ParticipantNotificationsHandler) Recipients(w http.ResponseWriter, r *http.Request) {
	filter, err := query.NewNotificationRecipientFilter(r.URL.Query().Get("filter"))
	if err != nil {
		response.BadRequest(w, "Некорректный фильтр получателей")
		return
	}

	event, err := h.eventRepo.FindActive(r.Context())
	if err != nil {
		log.Printf("ERROR participant notification recipients active event lookup failed: error=%v", err)
		response.InternalServerError(w, "Не удалось получить активное событие")
		return
	}
	if event == nil {
		response.Conflict(w, "Нет активного события")
		return
	}

	recipients, err := h.recipientsHandler.Handle(r.Context(), event.ID, filter)
	if err != nil {
		log.Printf("ERROR participant notification recipients lookup failed: event_id=%d filter=%s error=%v", event.ID, filter, err)
		response.InternalServerError(w, "Не удалось подготовить список участников")
		return
	}

	items := make([]participantNotificationRecipientResponse, 0, len(recipients))
	for _, recipient := range recipients {
		items = append(items, participantNotificationRecipientResponse{
			UserID:             recipient.UserID,
			Label:              recipient.Label,
			Username:           recipient.Username,
			Status:             recipient.Status,
			HasGift:            recipient.HasGift,
			HasUnassignedGifts: recipient.HasUnassignedGifts,
		})
	}

	log.Printf("INFO participant notification recipients served: event_id=%d filter=%s recipient_count=%d", event.ID, filter, len(items))
	response.Success(w, participantNotificationRecipientsResponse{
		EventName:  event.Name,
		Filter:     string(filter),
		Recipients: items,
	})
}

// Send обрабатывает POST /api/participant-notifications/send и немедленно
// возвращает задачу; доставку выполняет отдельный worker.
func (h *ParticipantNotificationsHandler) Send(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxParticipantNotificationRequestSize)
	var req sendParticipantNotificationsRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		response.BadRequest(w, "Некорректное тело запроса")
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if len(req.UserIDs) == 0 {
		response.BadRequest(w, "Не выбрано ни одного участника")
		return
	}
	if req.Text == "" {
		response.BadRequest(w, "Введите текст уведомления")
		return
	}

	event, err := h.eventRepo.FindActive(r.Context())
	if err != nil {
		log.Printf("ERROR participant notification send active event lookup failed: error=%v", err)
		response.InternalServerError(w, "Не удалось получить активное событие")
		return
	}
	if event == nil {
		response.Conflict(w, "Нет активного события")
		return
	}

	job, err := h.jobManager.Submit(command.SendParticipantNotificationsCommand{
		EventID: event.ID,
		UserIDs: req.UserIDs,
		Text:    req.Text,
	})
	if err != nil {
		switch {
		case errors.Is(err, command.ErrParticipantNotificationsNotConfigured):
			response.Conflict(w, "Рассылка недоступна: не настроен токен Telegram-бота")
		case errors.Is(err, command.ErrParticipantNotificationTextEmpty):
			response.BadRequest(w, "Введите текст уведомления")
		case errors.Is(err, command.ErrParticipantNotificationTextTooLong):
			response.BadRequest(w, "Текст уведомления длиннее 4096 символов")
		case errors.Is(err, command.ErrParticipantNotificationQueueFull):
			response.Error(w, http.StatusTooManyRequests, "Очередь рассылок занята, попробуйте позже")
		default:
			log.Printf("ERROR participant notification send failed: event_id=%d selected_count=%d text_length=%d error=%v", event.ID, len(req.UserIDs), len([]rune(req.Text)), err)
			response.InternalServerError(w, "Не удалось отправить уведомления")
		}
		return
	}

	log.Printf("INFO [FIX:participant-notification-queue] job accepted: job_id=%s event_id=%d selected_count=%d", job.ID, event.ID, len(req.UserIDs))
	response.JSON(w, http.StatusAccepted, participantNotificationJobResponseFrom(job))
}

// Status обрабатывает GET /api/participant-notifications/jobs/:id.
func (h *ParticipantNotificationsHandler) Status(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(chi.URLParam(r, "id"))
	if jobID == "" {
		response.BadRequest(w, "Некорректный ID рассылки")
		return
	}
	job, ok := h.jobManager.Get(jobID)
	if !ok {
		response.NotFound(w, "Рассылка не найдена")
		return
	}
	response.Success(w, participantNotificationJobResponseFrom(job))
}

func participantNotificationJobResponseFrom(job command.ParticipantNotificationJob) participantNotificationJobResponse {
	return participantNotificationJobResponse{
		ID:        job.ID,
		Status:    string(job.Status),
		Requested: job.Requested,
		Sent:      job.Result.Sent,
		Failed:    job.Result.Failed,
		Skipped:   job.Result.Skipped,
		Error:     job.Error,
	}
}

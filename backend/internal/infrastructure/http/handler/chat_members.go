package handler

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/http/response"
)

const maxChatMembersCSVSize = 10 << 20 // 10 MiB

// ChatMembersHandler обслуживает импорт ростера чата и чистку по подаркам.
type ChatMembersHandler struct {
	chatMemberRepo         repository.ChatMemberRepository
	eventRepo              repository.EventRepository
	purgeCandidatesHandler *query.GetChatPurgeCandidatesHandler
	executePurgeHandler    *command.ExecuteChatPurgeHandler
}

// NewChatMembersHandler создаёт новый handler.
func NewChatMembersHandler(
	chatMemberRepo repository.ChatMemberRepository,
	eventRepo repository.EventRepository,
	purgeCandidatesHandler *query.GetChatPurgeCandidatesHandler,
	executePurgeHandler *command.ExecuteChatPurgeHandler,
) *ChatMembersHandler {
	return &ChatMembersHandler{
		chatMemberRepo:         chatMemberRepo,
		eventRepo:              eventRepo,
		purgeCandidatesHandler: purgeCandidatesHandler,
		executePurgeHandler:    executePurgeHandler,
	}
}

// --- Response payloads ---

type chatMembersImportResponse struct {
	Imported     int `json:"imported"`
	SkippedRows  int `json:"skipped_rows"`
	TotalInTable int `json:"total_in_table"`
}

type chatPurgeCandidateResponse struct {
	UserID   int64  `json:"user_id"`
	Label    string `json:"label"`
	Username string `json:"username,omitempty"`
	Reason   string `json:"reason"`
}

type chatPurgeCandidatesResponse struct {
	EventName           string                       `json:"event_name"`
	Candidates          []chatPurgeCandidateResponse `json:"candidates"`
	ProtectedGiftOwners int                          `json:"protected_gift_owners"`
}

type chatPurgeExecuteRequest struct {
	UserIDs []int64 `json:"user_ids"`
}

type chatPurgeExecuteResponse struct {
	Kicked    int `json:"kicked"`
	Failed    int `json:"failed"`
	Skipped   int `json:"skipped"`
	Protected int `json:"protected"`
}

// Import заливает первичный ростер из CSV (export скрипта) в таблицу.
func (h *ChatMembersHandler) Import(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxChatMembersCSVSize)
	if err := r.ParseMultipartForm(maxChatMembersCSVSize); err != nil {
		response.BadRequest(w, "Некорректная форма или файл слишком большой")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "CSV-файл обязателен")
		return
	}
	defer file.Close()

	members, skipped, err := parseChatMembersCSV(file)
	if err != nil {
		log.Printf("Chat members import parse failed: error=%v", err)
		response.BadRequest(w, "Не удалось разобрать CSV")
		return
	}

	if err := h.chatMemberRepo.BulkUpsert(r.Context(), members); err != nil {
		log.Printf("Chat members import failed: imported=%d error=%v", len(members), err)
		response.InternalServerError(w, "Не удалось сохранить участников чата")
		return
	}

	total, err := h.chatMemberRepo.Count(r.Context())
	if err != nil {
		log.Printf("Chat members count after import failed: error=%v", err)
		total = len(members)
	}

	log.Printf("INFO chat members imported: imported=%d skipped_rows=%d", len(members), skipped)
	response.Success(w, chatMembersImportResponse{
		Imported:     len(members),
		SkippedRows:  skipped,
		TotalInTable: total,
	})
}

// Candidates возвращает кандидатов на чистку по активному событию.
func (h *ChatMembersHandler) Candidates(w http.ResponseWriter, r *http.Request) {
	event, err := h.eventRepo.FindActive(r.Context())
	if err != nil {
		log.Printf("Chat purge candidates active event lookup failed: error=%v", err)
		response.InternalServerError(w, "Не удалось получить активное событие")
		return
	}
	if event == nil {
		response.Conflict(w, "Нет активного события")
		return
	}

	result, err := h.purgeCandidatesHandler.Handle(r.Context(), event.ID)
	if err != nil {
		log.Printf("Chat purge candidates failed: event_id=%d error=%v", event.ID, err)
		response.InternalServerError(w, "Не удалось подготовить список кандидатов")
		return
	}

	candidates := make([]chatPurgeCandidateResponse, 0, len(result.Candidates))
	for _, candidate := range result.Candidates {
		candidates = append(candidates, chatPurgeCandidateResponse{
			UserID:   candidate.UserID,
			Label:    candidate.Label,
			Username: candidate.Username,
			Reason:   candidate.Reason,
		})
	}

	response.Success(w, chatPurgeCandidatesResponse{
		EventName:           event.Name,
		Candidates:          candidates,
		ProtectedGiftOwners: result.ProtectedGiftOwners,
	})
}

// Execute кикает выбранных пользователей.
func (h *ChatMembersHandler) Execute(w http.ResponseWriter, r *http.Request) {
	var req chatPurgeExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Некорректное тело запроса")
		return
	}
	if len(req.UserIDs) == 0 {
		response.BadRequest(w, "Не выбрано ни одного пользователя")
		return
	}

	event, err := h.eventRepo.FindActive(r.Context())
	if err != nil {
		log.Printf("Chat purge execute active event lookup failed: error=%v", err)
		response.InternalServerError(w, "Не удалось получить активное событие")
		return
	}
	if event == nil {
		response.Conflict(w, "Нет активного события")
		return
	}

	result, err := h.executePurgeHandler.Handle(r.Context(), command.ExecuteChatPurgeCommand{
		EventID: event.ID,
		UserIDs: req.UserIDs,
	})
	if err != nil {
		if errors.Is(err, command.ErrChatPurgeNotConfigured) {
			response.Conflict(w, "Функция недоступна: не настроены токен бота или публичный чат")
			return
		}
		log.Printf("Chat purge execute failed: event_id=%d error=%v", event.ID, err)
		response.InternalServerError(w, "Не удалось выполнить чистку")
		return
	}

	response.Success(w, chatPurgeExecuteResponse{
		Kicked:    result.Kicked,
		Failed:    result.Failed,
		Skipped:   result.Skipped,
		Protected: result.Protected,
	})
}

// parseChatMembersCSV разбирает CSV экспорта участников чата. Возвращает
// участников (людей, не удалённые аккаунты) и количество пропущенных строк.
// Формат: user_id,username,first_name,last_name,is_bot,is_deleted,role,joined_at
func parseChatMembersCSV(r io.Reader) ([]*entity.ChatMember, int, error) {
	reader := csv.NewReader(stripBOM(r))
	reader.FieldsPerRecord = -1

	members := make([]*entity.ChatMember, 0)
	skipped := 0
	rowIndex := -1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, skipped, err
		}
		rowIndex++

		// Заголовок.
		if rowIndex == 0 && len(record) > 0 && strings.EqualFold(strings.TrimSpace(record[0]), "user_id") {
			continue
		}
		if len(record) < 7 {
			skipped++
			continue
		}

		userID, err := strconv.ParseInt(strings.TrimSpace(record[0]), 10, 64)
		if err != nil || userID <= 0 {
			skipped++
			continue
		}
		if csvBool(record[5]) { // is_deleted
			skipped++
			continue
		}

		role := strings.ToLower(strings.TrimSpace(record[6]))
		members = append(members, &entity.ChatMember{
			TelegramUserID: userID,
			Username:       strings.TrimSpace(record[1]),
			FirstName:      strings.TrimSpace(record[2]),
			LastName:       strings.TrimSpace(record[3]),
			IsBot:          csvBool(record[4]),
			IsAdmin:        role == "admin" || role == "creator",
		})
	}

	if skipped > 0 {
		log.Printf("WARN chat members csv parse: skipped_rows=%d", skipped)
	}
	return members, skipped, nil
}

func csvBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "t", "yes", "y":
		return true
	default:
		return false
	}
}

// stripBOM снимает UTF-8 BOM (export пишет utf-8-sig), не читая весь поток в память.
func stripBOM(r io.Reader) io.Reader {
	return &bomReader{r: r}
}

type bomReader struct {
	r       io.Reader
	checked bool
	buf     []byte
}

func (b *bomReader) Read(p []byte) (int, error) {
	if !b.checked {
		b.checked = true
		var head [3]byte
		n, err := io.ReadFull(b.r, head[:])
		if n == 3 && head[0] == 0xEF && head[1] == 0xBB && head[2] == 0xBF {
			// BOM снят.
		} else {
			b.buf = append(b.buf, head[:n]...)
		}
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			return 0, err
		}
	}
	if len(b.buf) > 0 {
		n := copy(p, b.buf)
		b.buf = b.buf[n:]
		return n, nil
	}
	return b.r.Read(p)
}

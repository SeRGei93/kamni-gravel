package dto

import (
	"context"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

// ResultDTO представляет DTO результата для API
type ResultDTO struct {
	ID             uint          `json:"id"`
	ParticipantID  uint          `json:"participant_id"`
	ResultLink     *string       `json:"result_link,omitempty"`
	ElapsedTimeSec *int          `json:"elapsed_time_sec,omitempty"`
	MovingTimeSec  *int          `json:"moving_time_sec,omitempty"`
	ElapsedTime    *string       `json:"elapsed_time,omitempty"` // формат ЧЧ:ММ:СС
	MovingTime     *string       `json:"moving_time,omitempty"`  // формат ЧЧ:ММ:СС
	IsCurrent      bool          `json:"is_current"`
	SubmittedAt    time.Time     `json:"submitted_at"`
	Criteria       []*CriteriaDTO `json:"criteria,omitempty"` // критерии результата
}

// FromResult создаёт DTO из entity.Result
func FromResult(r *entity.Result) *ResultDTO {
	if r == nil {
		return nil
	}

	dto := &ResultDTO{
		ID:             r.ID,
		ParticipantID:  r.ParticipantID,
		ElapsedTimeSec: r.ElapsedTimeSec,
		MovingTimeSec:  r.MovingTimeSec,
		IsCurrent:      r.IsCurrent,
		SubmittedAt:    r.SubmittedAt,
	}

	if r.ResultLink != nil {
		link := r.ResultLink.URL
		dto.ResultLink = &link
	}

	if r.ElapsedTimeSec != nil {
		formatted := r.ElapsedTimeFormatted()
		dto.ElapsedTime = &formatted
	}

	if r.MovingTimeSec != nil {
		formatted := r.MovingTimeFormatted()
		dto.MovingTime = &formatted
	}

	// Добавляем критерии
	if len(r.Criteria) > 0 {
		dto.Criteria = make([]*CriteriaDTO, len(r.Criteria))
		for i, c := range r.Criteria {
			dto.Criteria[i] = FromCriteria(c)
		}
	}

	return dto
}

// ResultListResponse представляет ответ со списком результатов
type ResultListResponse struct {
	Results []*ResultDTO `json:"results"`
	Total   int          `json:"total"`
}

// CreateResult создаёт новый результат
func CreateResult(ctx context.Context, resultRepo repository.ResultRepository, participantID uint, resultLink string) (*entity.Result, error) {
	// Валидируем ссылку на результат
	link, err := valueobject.NewResultLink(resultLink)
	if err != nil {
		return nil, err
	}

	result := &entity.Result{
		ParticipantID: participantID,
		ResultLink:    link,
		IsCurrent:     true,
		SubmittedAt:   time.Now(),
	}

	if err := resultRepo.Create(ctx, result); err != nil {
		return nil, err
	}

	return result, nil
}

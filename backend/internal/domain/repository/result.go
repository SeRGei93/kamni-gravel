package repository

import (
	"context"
	"gravel_bot/internal/domain/entity"
)

// ResultRepository определяет интерфейс для работы с результатами
type ResultRepository interface {
	// Create создаёт новый результат
	Create(ctx context.Context, result *entity.Result) error

	// FindByID находит результат по ID
	FindByID(ctx context.Context, id uint) (*entity.Result, error)

	// FindCurrentByParticipant находит актуальный результат участника
	FindCurrentByParticipant(ctx context.Context, participantID uint) (*entity.Result, error)

	// FindByParticipant находит все результаты участника
	FindByParticipant(ctx context.Context, participantID uint) ([]*entity.Result, error)

	// UpdateTime обновляет время результата
	UpdateTime(ctx context.Context, id uint, elapsedSec, movingSec *int) error

	// UpdateMetrics обновляет время и метрики заезда результата по его ID
	// (старт, финиш, дистанция, пульс, пиковая скорость, каденс, калории).
	UpdateMetrics(ctx context.Context, result *entity.Result) error

	// MarkAsNotCurrent помечает результат как неактуальный
	MarkAsNotCurrent(ctx context.Context, id uint) error

	// Delete удаляет результат
	Delete(ctx context.Context, id uint) error

	// AddCriteria добавляет критерий к результату
	AddCriteria(ctx context.Context, resultID, criteriaID uint) error

	// RemoveCriteria удаляет критерий из результата
	RemoveCriteria(ctx context.Context, resultID, criteriaID uint) error

	// FindWithCriteria находит результат с критериями
	FindWithCriteria(ctx context.Context, resultID uint) (*entity.Result, error)

	// FindByEventWithPlaces находит результаты события с рассчитанными местами
	FindByEventWithPlaces(ctx context.Context, eventID uint) ([]*ResultWithPlace, error)

	// FindPrevEventElapsedByUser возвращает общее время (секунды) актуальных
	// результатов предыдущего события в разрезе user_id. Предыдущее событие —
	// ближайшее по дате старта (или дате создания, если старт не задан) до
	// события eventID. Пустая map, если предыдущего события нет.
	FindPrevEventElapsedByUser(ctx context.Context, eventID uint) (map[int64]int, error)
}

// ResultWithPlace представляет результат с рассчитанными местами
type ResultWithPlace struct {
	*entity.Result
	ParticipantGender   string
	ParticipantBikeType string
	PlaceAbsolute       int
	PlaceByGender       int
	PlaceByGenderBike   int
}

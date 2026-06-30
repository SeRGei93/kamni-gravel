package valueobject

import "fmt"

// ParticipantStatus представляет статус участия в зачёте.
//
// Семантика для распределения призов:
//   - active        — обычный участник: места в зачёте и все призы.
//   - dnf           — «Сошёл с дистанции»: исключён из зачёта и призов по
//     местам, но остаётся кандидатом на призы по критериям (gift с criteria
//     без привязки к месту).
//   - disqualified  — «Дисквалификация»: полностью исключён из зачёта и любых
//     призов (и по местам, и по критериям).
type ParticipantStatus string

const (
	// ParticipantStatusActive — участник в зачёте (значение по умолчанию).
	ParticipantStatusActive ParticipantStatus = "active"
	// ParticipantStatusDNF — сошёл с дистанции.
	ParticipantStatusDNF ParticipantStatus = "dnf"
	// ParticipantStatusDisqualified — дисквалифицирован.
	ParticipantStatusDisqualified ParticipantStatus = "disqualified"
)

// NewParticipantStatus создаёт и валидирует статус. Пустая строка трактуется
// как active — это позволяет старым данным и записям без явного статуса
// оставаться полноценными участниками зачёта.
func NewParticipantStatus(value string) (ParticipantStatus, error) {
	if value == "" {
		return ParticipantStatusActive, nil
	}
	s := ParticipantStatus(value)
	if !s.IsValid() {
		return "", fmt.Errorf("invalid participant status: %s. Must be one of: active, dnf, disqualified", value)
	}
	return s, nil
}

// IsValid проверяет, валиден ли статус.
func (s ParticipantStatus) IsValid() bool {
	switch s {
	case ParticipantStatusActive, ParticipantStatusDNF, ParticipantStatusDisqualified:
		return true
	}
	return false
}

// String возвращает строковое представление.
func (s ParticipantStatus) String() string {
	return string(s)
}

// DisplayName возвращает читаемое название.
func (s ParticipantStatus) DisplayName() string {
	switch s {
	case ParticipantStatusDNF:
		return "Сошёл с дистанции"
	case ParticipantStatusDisqualified:
		return "Дисквалификация"
	default:
		return "Участвует"
	}
}

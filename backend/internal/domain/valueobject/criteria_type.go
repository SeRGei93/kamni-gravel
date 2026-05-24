package valueobject

import "errors"

// CriteriaType представляет тип критерия
type CriteriaType string

const (
	CriteriaTypeSpeed  CriteriaType = "speed"
	CriteriaTypePhoto  CriteriaType = "photo"
	CriteriaTypeBeer   CriteriaType = "beer"
	CriteriaTypeCustom CriteriaType = "custom"
)

var (
	ErrInvalidCriteriaType = errors.New("invalid criteria type")
)

// NewCriteriaType создаёт и валидирует тип критерия
func NewCriteriaType(value string) (CriteriaType, error) {
	ct := CriteriaType(value)
	
	switch ct {
	case CriteriaTypeSpeed, CriteriaTypePhoto, CriteriaTypeBeer, CriteriaTypeCustom:
		return ct, nil
	default:
		return "", ErrInvalidCriteriaType
	}
}

// String возвращает строковое представление
func (ct CriteriaType) String() string {
	return string(ct)
}

// IsValid проверяет валидность типа критерия
func (ct CriteriaType) IsValid() bool {
	switch ct {
	case CriteriaTypeSpeed, CriteriaTypePhoto, CriteriaTypeBeer, CriteriaTypeCustom:
		return true
	default:
		return false
	}
}

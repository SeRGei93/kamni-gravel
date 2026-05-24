package valueobject

import "fmt"

// Gender представляет пол участника
type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

// NewGender создаёт и валидирует пол
func NewGender(value string) (Gender, error) {
	g := Gender(value)
	if !g.IsValid() {
		return "", fmt.Errorf("invalid gender: %s. Must be one of: male, female", value)
	}
	return g, nil
}

// IsValid проверяет, валиден ли пол
func (g Gender) IsValid() bool {
	switch g {
	case GenderMale, GenderFemale:
		return true
	}
	return false
}

// String возвращает строковое представление
func (g Gender) String() string {
	return string(g)
}

// DisplayName возвращает читаемое название
func (g Gender) DisplayName() string {
	switch g {
	case GenderMale:
		return "Мужской"
	case GenderFemale:
		return "Женский"
	default:
		return string(g)
	}
}

// GenderFilter представляет фильтр по полу для номинаций
type GenderFilter string

const (
	GenderFilterAll    GenderFilter = "all"
	GenderFilterMale   GenderFilter = "male"
	GenderFilterFemale GenderFilter = "female"
)

// NewGenderFilter создаёт и валидирует фильтр
func NewGenderFilter(value string) (GenderFilter, error) {
	gf := GenderFilter(value)
	if !gf.IsValid() {
		return "", fmt.Errorf("invalid gender filter: %s", value)
	}
	return gf, nil
}

// IsValid проверяет валидность фильтра
func (gf GenderFilter) IsValid() bool {
	switch gf {
	case GenderFilterAll, GenderFilterMale, GenderFilterFemale:
		return true
	}
	return false
}

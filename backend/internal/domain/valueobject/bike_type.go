package valueobject

import "fmt"

// BikeType представляет тип велосипеда
type BikeType string

const (
	BikeTypeGravel    BikeType = "gravel"
	BikeTypeMTB       BikeType = "mtb"
	BikeTypeRoad      BikeType = "road"
	BikeTypeFixedGear BikeType = "single_speed"
	BikeTypeTandem    BikeType = "tandem"
)

// NewBikeType создаёт и валидирует тип велосипеда
func NewBikeType(value string) (BikeType, error) {
	bt := BikeType(value)
	if !bt.IsValid() {
		return "", fmt.Errorf("invalid bike type: %s. Must be one of: gravel, mtb, road, single_speed, tandem", value)
	}
	return bt, nil
}

// IsValid проверяет, валиден ли тип велосипеда
func (bt BikeType) IsValid() bool {
	switch bt {
	case BikeTypeGravel, BikeTypeMTB, BikeTypeRoad, BikeTypeFixedGear, BikeTypeTandem:
		return true
	}
	return false
}

// String возвращает строковое представление
func (bt BikeType) String() string {
	return string(bt)
}

// DisplayName возвращает читаемое название
func (bt BikeType) DisplayName() string {
	switch bt {
	case BikeTypeGravel:
		return "Гравийник"
	case BikeTypeMTB:
		return "МТБ"
	case BikeTypeRoad:
		return "Шоссейник"
	case BikeTypeFixedGear:
		return "Фикс"
	case BikeTypeTandem:
		return "Тандем"
	default:
		return string(bt)
	}
}

// BikeTypeFilter представляет фильтр по типу велосипеда для номинаций
type BikeTypeFilter string

const (
	BikeTypeFilterAll       BikeTypeFilter = "all"
	BikeTypeFilterGravel    BikeTypeFilter = "gravel"
	BikeTypeFilterMTB       BikeTypeFilter = "mtb"
	BikeTypeFilterRoad      BikeTypeFilter = "road"
	BikeTypeFilterFixedGear BikeTypeFilter = "single_speed"
	BikeTypeFilterTandem    BikeTypeFilter = "tandem"
)

// NewBikeTypeFilter создаёт и валидирует фильтр
func NewBikeTypeFilter(value string) (BikeTypeFilter, error) {
	btf := BikeTypeFilter(value)
	if !btf.IsValid() {
		return "", fmt.Errorf("invalid bike type filter: %s", value)
	}
	return btf, nil
}

// IsValid проверяет валидность фильтра
func (btf BikeTypeFilter) IsValid() bool {
	switch btf {
	case BikeTypeFilterAll, BikeTypeFilterGravel, BikeTypeFilterMTB, BikeTypeFilterRoad, BikeTypeFilterFixedGear, BikeTypeFilterTandem:
		return true
	}
	return false
}

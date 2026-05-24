package dto

// PrizeDistributionDTO представляет результат автоматического распределения призов
type PrizeDistributionDTO struct {
	ParticipantID   uint          `json:"participant_id"`
	ParticipantName string        `json:"participant_name"`
	Gender          string        `json:"gender"`
	BikeType        string        `json:"bike_type"`
	PlaceAbsolute   int           `json:"place_absolute"`
	PlaceByGender   int           `json:"place_by_gender"`
	ResultCriteria  []*CriteriaDTO `json:"result_criteria"`
	MatchedGifts    []*GiftDTO    `json:"matched_gifts,omitempty"`
	MatchReason     string         `json:"match_reason"` // "criteria", "place", "no_match"
}

// PrizeDistributionListResponse представляет ответ со списком распределения призов
type PrizeDistributionListResponse struct {
	Distribution []*PrizeDistributionDTO `json:"distribution"`
	Total        int                     `json:"total"`
}

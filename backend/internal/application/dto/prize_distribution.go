package dto

import "gravel_bot/internal/application/query"

// PrizeDistributionDTO представляет результат автоматического распределения призов
type PrizeDistributionDTO struct {
	ParticipantID          uint                      `json:"participant_id"`
	ParticipantName        string                    `json:"participant_name"`
	Gender                 string                    `json:"gender"`
	BikeType               string                    `json:"bike_type"`
	PlaceAbsolute          int                       `json:"place_absolute"`
	PlaceByGender          int                       `json:"place_by_gender"`
	PlaceByGenderBike      int                       `json:"place_by_gender_bike"`
	ResultCriteria         []*CriteriaDTO            `json:"result_criteria"`
	MatchedGifts           []*GiftDTO                `json:"matched_gifts,omitempty"`
	MatchedGiftAssignments []*PrizeGiftAssignmentDTO `json:"matched_gift_assignments,omitempty"`
	MatchReason            string                    `json:"match_reason"` // "criteria", "place", "match", "no_match"
}

// PrizeGiftAssignmentDTO представляет одно конкретное назначение слота подарка.
type PrizeGiftAssignmentDTO struct {
	Gift           *GiftDTO `json:"gift"`
	GiftID         uint     `json:"gift_id"`
	RuleType       string   `json:"rule_type"`
	TargetRank     int      `json:"target_rank,omitempty"`
	AssignedRank   int      `json:"assigned_rank"`
	IsFallback     bool     `json:"is_fallback"`
	FallbackReason string   `json:"fallback_reason,omitempty"`
	MatchReason    string   `json:"match_reason"`
}

// UnassignedPrizeSlotDTO представляет невыданный слот правила подарка.
type UnassignedPrizeSlotDTO struct {
	GiftID         uint     `json:"gift_id"`
	Gift           *GiftDTO `json:"gift,omitempty"`
	RuleType       string   `json:"rule_type"`
	TargetRank     int      `json:"target_rank,omitempty"`
	Reason         string   `json:"reason"`
	FallbackReason string   `json:"fallback_reason,omitempty"`
}

// PrizeDistributionListResponse представляет ответ со списком распределения призов
type PrizeDistributionListResponse struct {
	Distribution    []*PrizeDistributionDTO   `json:"distribution"`
	UnassignedSlots []*UnassignedPrizeSlotDTO `json:"unassigned_slots,omitempty"`
	Total           int                       `json:"total"`
}

// FromPrizeGiftAssignment создаёт DTO назначения слота подарка.
func FromPrizeGiftAssignment(assignment *query.PrizeGiftAssignment) *PrizeGiftAssignmentDTO {
	if assignment == nil {
		return nil
	}
	dto := &PrizeGiftAssignmentDTO{
		GiftID:         giftIDFromAssignment(assignment),
		RuleType:       assignment.RuleType,
		TargetRank:     assignment.TargetRank,
		AssignedRank:   assignment.AssignedRank,
		IsFallback:     assignment.IsFallback,
		FallbackReason: assignment.FallbackReason,
		MatchReason:    assignment.MatchReason,
	}
	if assignment.Gift != nil {
		dto.Gift = FromGift(assignment.Gift)
	}
	return dto
}

// FromUnassignedPrizeSlot создаёт DTO невыданного слота.
func FromUnassignedPrizeSlot(slot *query.UnassignedPrizeSlot) *UnassignedPrizeSlotDTO {
	if slot == nil {
		return nil
	}
	dto := &UnassignedPrizeSlotDTO{
		GiftID:         slot.GiftID,
		RuleType:       slot.RuleType,
		TargetRank:     slot.TargetRank,
		Reason:         slot.Reason,
		FallbackReason: slot.FallbackReason,
	}
	if slot.Gift != nil {
		dto.Gift = FromGift(slot.Gift)
	}
	return dto
}

func giftIDFromAssignment(assignment *query.PrizeGiftAssignment) uint {
	if assignment.Gift != nil {
		return assignment.Gift.ID
	}
	return 0
}

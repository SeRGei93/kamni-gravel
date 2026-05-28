package query

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

// PrizeDistributionOutput contains participant rows and event-level slot diagnostics.
type PrizeDistributionOutput struct {
	Results         []*PrizeDistributionResult
	UnassignedSlots []*UnassignedPrizeSlot
}

// PrizeGiftAssignment contains metadata for one concrete prize slot assignment.
type PrizeGiftAssignment struct {
	ParticipantID  uint
	Gift           *entity.Gift
	MatchReason    string
	RuleType       string
	TargetRank     int
	AssignedRank   int
	IsFallback     bool
	FallbackReason string
}

// UnassignedPrizeSlot describes a requested gift slot that could not be assigned.
type UnassignedPrizeSlot struct {
	Gift           *entity.Gift
	GiftID         uint
	RuleType       string
	TargetRank     int
	Reason         string
	FallbackReason string
}

type prizeParticipantContext struct {
	rwp         *repository.ResultWithPlace
	participant *entity.Participant
}

type prizeEligibleContext struct {
	*prizeParticipantContext
	rank int
}

type prizeRuleSlot struct {
	ruleType   string
	targetRank int
	direct     bool
}

func (h *GetPrizeDistributionHandler) distributePrizeSlots(
	resultsWithPlaces []*repository.ResultWithPlace,
	gifts []*entity.Gift,
	participantMap map[uint]*entity.Participant,
) *PrizeDistributionOutput {
	contexts := buildPrizeParticipantContexts(resultsWithPlaces, participantMap)
	rowsByParticipant := make(map[uint]*PrizeDistributionResult, len(contexts))
	results := make([]*PrizeDistributionResult, 0, len(contexts))

	for _, ctx := range contexts {
		row := prizeDistributionResultFromContext(ctx)
		rowsByParticipant[ctx.participant.ID] = row
		results = append(results, row)
	}

	approvedGifts := filterApprovedPrizeGifts(gifts)
	sort.SliceStable(approvedGifts, func(i, j int) bool {
		return approvedGifts[i].ID < approvedGifts[j].ID
	})

	blockedByHigherPriority := make(map[uint]bool)
	var unassignedSlots []*UnassignedPrizeSlot
	priorities := []giftMatchPriority{
		giftMatchPriorityCriteriaPlace,
		giftMatchPriorityCriteria,
		giftMatchPriorityPlace,
		giftMatchPriorityGeneric,
	}

	for _, priority := range priorities {
		assignedThisPriority := make(map[uint]bool)

		for _, gift := range approvedGifts {
			giftPriority, ok := classifyGiftForSlotEngine(gift)
			if !ok || giftPriority != priority {
				continue
			}

			eligible := eligiblePrizeContexts(contexts, gift)
			if len(eligible) == 0 {
				unassignedSlots = append(unassignedSlots, unassignedSlotsForEmptyGroup(gift)...)
				continue
			}

			rule := giftPlaceRuleForDistribution(gift)
			if rule.IsNone() {
				if assignment := assignNoPlaceGift(gift, eligible, blockedByHigherPriority); assignment != nil {
					rowsByParticipant[assignment.ParticipantID].appendGiftAssignment(assignment)
					assignedThisPriority[assignment.ParticipantID] = true
				}
				continue
			}

			giftParticipantUsed := make(map[uint]bool)
			for _, slot := range prizeRuleSlots(rule, len(eligible)) {
				assignment, unassigned := assignPrizeRuleSlot(gift, slot, eligible, blockedByHigherPriority, giftParticipantUsed)
				if unassigned != nil {
					unassignedSlots = append(unassignedSlots, unassigned)
					continue
				}
				rowsByParticipant[assignment.ParticipantID].appendGiftAssignment(assignment)
				assignedThisPriority[assignment.ParticipantID] = true
				giftParticipantUsed[assignment.ParticipantID] = true
			}
		}

		for participantID := range assignedThisPriority {
			blockedByHigherPriority[participantID] = true
		}
	}

	for _, row := range results {
		if len(row.MatchedGiftAssignments) == 0 {
			row.MatchReason = "no_match"
			continue
		}
		row.MatchReason = row.MatchedGiftAssignments[0].MatchReason
		row.MatchedGifts = make([]*entity.Gift, 0, len(row.MatchedGiftAssignments))
		for _, assignment := range row.MatchedGiftAssignments {
			row.MatchedGifts = append(row.MatchedGifts, assignment.Gift)
		}
	}
	if len(results) > 0 && len(unassignedSlots) > 0 {
		results[0].UnassignedSlots = unassignedSlots
	}

	return &PrizeDistributionOutput{
		Results:         results,
		UnassignedSlots: unassignedSlots,
	}
}

func buildPrizeParticipantContexts(
	resultsWithPlaces []*repository.ResultWithPlace,
	participantMap map[uint]*entity.Participant,
) []*prizeParticipantContext {
	contexts := make([]*prizeParticipantContext, 0, len(resultsWithPlaces))
	for _, rwp := range resultsWithPlaces {
		if rwp == nil || rwp.Result == nil {
			continue
		}
		participant := participantMap[rwp.Result.ParticipantID]
		if participant == nil {
			continue
		}
		contexts = append(contexts, &prizeParticipantContext{
			rwp:         rwp,
			participant: participant,
		})
	}

	sort.SliceStable(contexts, func(i, j int) bool {
		left := contexts[i].rwp.Result
		right := contexts[j].rwp.Result
		leftElapsed := resultElapsedForSort(left)
		rightElapsed := resultElapsedForSort(right)
		if leftElapsed != rightElapsed {
			return leftElapsed < rightElapsed
		}
		if left.ID != right.ID {
			return left.ID < right.ID
		}
		return left.ParticipantID < right.ParticipantID
	})

	return contexts
}

func prizeDistributionResultFromContext(ctx *prizeParticipantContext) *PrizeDistributionResult {
	return &PrizeDistributionResult{
		ParticipantID:          ctx.participant.ID,
		ParticipantName:        prizeParticipantDisplayName(ctx.participant),
		Gender:                 string(ctx.participant.Gender),
		BikeType:               string(ctx.participant.BikeType),
		PlaceAbsolute:          ctx.rwp.PlaceAbsolute,
		PlaceByGender:          ctx.rwp.PlaceByGender,
		PlaceByGenderBike:      ctx.rwp.PlaceByGenderBike,
		ResultCriteria:         ctx.rwp.Result.Criteria,
		MatchedGifts:           []*entity.Gift{},
		MatchedGiftAssignments: []*PrizeGiftAssignment{},
		MatchReason:            "no_match",
	}
}

func resultElapsedForSort(result *entity.Result) int {
	if result.ElapsedTimeSec == nil {
		return math.MaxInt
	}
	return *result.ElapsedTimeSec
}

func prizeParticipantDisplayName(participant *entity.Participant) string {
	if participant.User == nil {
		return fmt.Sprintf("%d", participant.ID)
	}
	if strings.TrimSpace(participant.User.Username) != "" {
		return participant.User.Username
	}
	name := strings.TrimSpace(strings.TrimSpace(participant.User.FirstName) + " " + strings.TrimSpace(participant.User.LastName))
	if name != "" {
		return name
	}
	return fmt.Sprintf("%d", participant.ID)
}

func filterApprovedPrizeGifts(gifts []*entity.Gift) []*entity.Gift {
	approved := make([]*entity.Gift, 0, len(gifts))
	for _, gift := range gifts {
		if gift == nil || gift.ReviewStatus != entity.GiftReviewStatusApproved {
			continue
		}
		approved = append(approved, gift)
	}
	return approved
}

func classifyGiftForSlotEngine(gift *entity.Gift) (giftMatchPriority, bool) {
	hasCriteria := len(gift.Criteria) > 0
	hasPlace := giftPlaceRuleForDistribution(gift).HasPlaceConstraint()

	switch {
	case hasCriteria && hasPlace:
		return giftMatchPriorityCriteriaPlace, true
	case hasCriteria:
		return giftMatchPriorityCriteria, true
	case hasPlace:
		return giftMatchPriorityPlace, true
	default:
		return giftMatchPriorityGeneric, true
	}
}

func eligiblePrizeContexts(contexts []*prizeParticipantContext, gift *entity.Gift) []*prizeEligibleContext {
	eligible := make([]*prizeEligibleContext, 0, len(contexts))
	for _, ctx := range contexts {
		if !giftMatchesParticipantFilters(gift, ctx.participant) {
			continue
		}
		if !allPrizeCriteriaMatch(gift.Criteria, ctx.rwp.Result.Criteria) {
			continue
		}
		eligible = append(eligible, &prizeEligibleContext{
			prizeParticipantContext: ctx,
			rank:                    len(eligible) + 1,
		})
	}
	return eligible
}

func giftMatchesParticipantFilters(gift *entity.Gift, participant *entity.Participant) bool {
	if gift.BikeTypeFilter != "" && gift.BikeTypeFilter != "all" && gift.BikeTypeFilter != string(participant.BikeType) {
		return false
	}
	if gift.GenderFilter != "" && gift.GenderFilter != "all" && gift.GenderFilter != string(participant.Gender) {
		return false
	}
	return true
}

func allPrizeCriteriaMatch(giftCriteria, resultCriteria []*entity.Criteria) bool {
	if len(giftCriteria) == 0 {
		return true
	}
	if len(resultCriteria) == 0 {
		return false
	}

	resultCriteriaMap := make(map[uint]bool, len(resultCriteria))
	for _, criteria := range resultCriteria {
		resultCriteriaMap[criteria.ID] = true
	}
	for _, criteria := range giftCriteria {
		if !resultCriteriaMap[criteria.ID] {
			return false
		}
	}
	return true
}

func giftPlaceRuleForDistribution(gift *entity.Gift) valueobject.GiftPlaceRule {
	if !gift.PlaceRule.IsNone() {
		return gift.PlaceRule
	}
	if gift.Place == nil {
		return valueobject.NewGiftPlaceRuleNone()
	}
	rule, err := valueobject.NewGiftPlaceRulePlaces([]int{*gift.Place})
	if err != nil {
		return valueobject.NewGiftPlaceRuleNone()
	}
	return rule
}

func prizeRuleSlots(rule valueobject.GiftPlaceRule, eligibleCount int) []prizeRuleSlot {
	switch rule.Type() {
	case valueobject.GiftPlaceRuleTypePlaces:
		places := rule.Places()
		slots := make([]prizeRuleSlot, 0, len(places))
		for _, place := range places {
			slots = append(slots, prizeRuleSlot{
				ruleType:   string(valueobject.GiftPlaceRuleTypePlaces),
				targetRank: place,
			})
		}
		return slots
	case valueobject.GiftPlaceRuleTypeLastN:
		if eligibleCount == 0 {
			return nil
		}
		count := rule.LastCount()
		if count > eligibleCount {
			count = eligibleCount
		}
		startRank := eligibleCount - count + 1
		slots := make([]prizeRuleSlot, 0, count)
		for rank := startRank; rank <= eligibleCount; rank++ {
			slots = append(slots, prizeRuleSlot{
				ruleType:   string(valueobject.GiftPlaceRuleTypeLastN),
				targetRank: rank,
				direct:     true,
			})
		}
		return slots
	default:
		return nil
	}
}

func assignNoPlaceGift(
	gift *entity.Gift,
	eligible []*prizeEligibleContext,
	blockedByHigherPriority map[uint]bool,
) *PrizeGiftAssignment {
	for _, candidate := range eligible {
		if blockedByHigherPriority[candidate.participant.ID] {
			continue
		}
		return &PrizeGiftAssignment{
			ParticipantID: candidate.participant.ID,
			Gift:          gift,
			MatchReason:   giftMatchReasonForAssignment(gift),
			RuleType:      string(valueobject.GiftPlaceRuleTypeNone),
			AssignedRank:  candidate.rank,
		}
	}
	return nil
}

func assignPrizeRuleSlot(
	gift *entity.Gift,
	slot prizeRuleSlot,
	eligible []*prizeEligibleContext,
	blockedByHigherPriority map[uint]bool,
	giftParticipantUsed map[uint]bool,
) (*PrizeGiftAssignment, *UnassignedPrizeSlot) {
	if slot.direct {
		candidate := eligibleByRank(eligible, slot.targetRank)
		if candidate != nil && prizeCandidateAvailable(candidate, blockedByHigherPriority, giftParticipantUsed) {
			return newPrizeGiftAssignment(gift, slot, candidate, false, ""), nil
		}
		return nil, newUnassignedPrizeSlot(gift, slot, "target_unavailable")
	}

	candidate := eligibleByRank(eligible, slot.targetRank)
	if candidate != nil && prizeCandidateAvailable(candidate, blockedByHigherPriority, giftParticipantUsed) {
		return newPrizeGiftAssignment(gift, slot, candidate, false, ""), nil
	}

	fallbackReason := "target_unavailable"
	if candidate == nil {
		fallbackReason = "target_out_of_range"
	}
	fallbackCandidate := nearestFallbackPrizeCandidate(eligible, slot.targetRank, blockedByHigherPriority, giftParticipantUsed)
	if fallbackCandidate == nil {
		return nil, newUnassignedPrizeSlot(gift, slot, fallbackReason)
	}

	return newPrizeGiftAssignment(gift, slot, fallbackCandidate, true, fallbackReason), nil
}

func eligibleByRank(eligible []*prizeEligibleContext, rank int) *prizeEligibleContext {
	if rank <= 0 || rank > len(eligible) {
		return nil
	}
	return eligible[rank-1]
}

func prizeCandidateAvailable(
	candidate *prizeEligibleContext,
	blockedByHigherPriority map[uint]bool,
	giftParticipantUsed map[uint]bool,
) bool {
	participantID := candidate.participant.ID
	return !blockedByHigherPriority[participantID] && !giftParticipantUsed[participantID]
}

func nearestFallbackPrizeCandidate(
	eligible []*prizeEligibleContext,
	targetRank int,
	blockedByHigherPriority map[uint]bool,
	giftParticipantUsed map[uint]bool,
) *prizeEligibleContext {
	for distance := 1; distance <= len(eligible)+targetRank; distance++ {
		higherRank := targetRank + distance
		if candidate := eligibleByRank(eligible, higherRank); candidate != nil && prizeCandidateAvailable(candidate, blockedByHigherPriority, giftParticipantUsed) {
			return candidate
		}

		lowerRank := targetRank - distance
		if candidate := eligibleByRank(eligible, lowerRank); candidate != nil && prizeCandidateAvailable(candidate, blockedByHigherPriority, giftParticipantUsed) {
			return candidate
		}
	}
	return nil
}

func newPrizeGiftAssignment(
	gift *entity.Gift,
	slot prizeRuleSlot,
	candidate *prizeEligibleContext,
	isFallback bool,
	fallbackReason string,
) *PrizeGiftAssignment {
	return &PrizeGiftAssignment{
		ParticipantID:  candidate.participant.ID,
		Gift:           gift,
		MatchReason:    giftMatchReasonForAssignment(gift),
		RuleType:       slot.ruleType,
		TargetRank:     slot.targetRank,
		AssignedRank:   candidate.rank,
		IsFallback:     isFallback,
		FallbackReason: fallbackReason,
	}
}

func newUnassignedPrizeSlot(gift *entity.Gift, slot prizeRuleSlot, reason string) *UnassignedPrizeSlot {
	return &UnassignedPrizeSlot{
		Gift:           gift,
		GiftID:         gift.ID,
		RuleType:       slot.ruleType,
		TargetRank:     slot.targetRank,
		Reason:         reason,
		FallbackReason: reason,
	}
}

func unassignedSlotsForEmptyGroup(gift *entity.Gift) []*UnassignedPrizeSlot {
	rule := giftPlaceRuleForDistribution(gift)
	if rule.Type() == valueobject.GiftPlaceRuleTypeLastN {
		return nil
	}
	slots := prizeRuleSlots(rule, 0)
	unassigned := make([]*UnassignedPrizeSlot, 0, len(slots))
	for _, slot := range slots {
		unassigned = append(unassigned, newUnassignedPrizeSlot(gift, slot, "empty_eligible_group"))
	}
	return unassigned
}

func giftMatchReasonForAssignment(gift *entity.Gift) string {
	if len(gift.Criteria) > 0 {
		return "criteria"
	}
	if giftPlaceRuleForDistribution(gift).HasPlaceConstraint() {
		return "place"
	}
	return "match"
}

func (r *PrizeDistributionResult) appendGiftAssignment(assignment *PrizeGiftAssignment) {
	if assignment == nil {
		return
	}
	r.MatchedGiftAssignments = append(r.MatchedGiftAssignments, assignment)
}

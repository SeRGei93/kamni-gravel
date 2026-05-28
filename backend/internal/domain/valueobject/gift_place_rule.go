package valueobject

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// GiftPlaceRuleType identifies how a gift is bound to finishing ranks.
type GiftPlaceRuleType string

const (
	GiftPlaceRuleTypeNone   GiftPlaceRuleType = "none"
	GiftPlaceRuleTypePlaces GiftPlaceRuleType = "places"
	GiftPlaceRuleTypeLastN  GiftPlaceRuleType = "last_n"
)

// GiftPlaceRule describes optional rank constraints for prize distribution.
type GiftPlaceRule struct {
	ruleType  GiftPlaceRuleType
	places    []int
	lastCount int
}

// NewGiftPlaceRuleNone creates an unconstrained place rule.
func NewGiftPlaceRuleNone() GiftPlaceRule {
	return GiftPlaceRule{ruleType: GiftPlaceRuleTypeNone}
}

// NewGiftPlaceRulePlaces creates a normalized explicit-place rule.
func NewGiftPlaceRulePlaces(places []int) (GiftPlaceRule, error) {
	normalized, err := normalizeGiftPlaceRulePlaces(places)
	if err != nil {
		return GiftPlaceRule{}, err
	}

	return GiftPlaceRule{
		ruleType: GiftPlaceRuleTypePlaces,
		places:   normalized,
	}, nil
}

// ParseGiftPlaceRulePlaces parses admin input such as "1, 3, 10-15".
func ParseGiftPlaceRulePlaces(input string) (GiftPlaceRule, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return GiftPlaceRule{}, fmt.Errorf("gift place rule places cannot be empty")
	}

	var places []int
	tokens := strings.Split(input, ",")
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			return GiftPlaceRule{}, fmt.Errorf("gift place rule contains empty place token")
		}

		if strings.Contains(token, "-") {
			rangePlaces, err := parseGiftPlaceRuleRange(token)
			if err != nil {
				return GiftPlaceRule{}, err
			}
			places = append(places, rangePlaces...)
			continue
		}

		place, err := parseGiftPlaceRulePositiveInt(token)
		if err != nil {
			return GiftPlaceRule{}, err
		}
		places = append(places, place)
	}

	return NewGiftPlaceRulePlaces(places)
}

// NewGiftPlaceRuleLastN creates a rule for the last N ranks of the eligible group.
func NewGiftPlaceRuleLastN(count int) (GiftPlaceRule, error) {
	if count <= 0 {
		return GiftPlaceRule{}, fmt.Errorf("gift place rule last_n count must be greater than zero")
	}

	return GiftPlaceRule{
		ruleType:  GiftPlaceRuleTypeLastN,
		lastCount: count,
	}, nil
}

// Type returns the rule type. A zero-value rule is treated as none.
func (r GiftPlaceRule) Type() GiftPlaceRuleType {
	if r.ruleType == "" {
		return GiftPlaceRuleTypeNone
	}
	return r.ruleType
}

// Places returns a copy of normalized explicit places.
func (r GiftPlaceRule) Places() []int {
	places := make([]int, len(r.places))
	copy(places, r.places)
	return places
}

// LastCount returns the N value for last_n rules.
func (r GiftPlaceRule) LastCount() int {
	return r.lastCount
}

// IsNone reports whether the rule has no place constraint.
func (r GiftPlaceRule) IsNone() bool {
	return r.Type() == GiftPlaceRuleTypeNone
}

// HasPlaceConstraint reports whether the rule constrains assignment by rank.
func (r GiftPlaceRule) HasPlaceConstraint() bool {
	return r.Type() == GiftPlaceRuleTypePlaces || r.Type() == GiftPlaceRuleTypeLastN
}

// FirstLegacyPlace returns the first explicit place for legacy API compatibility.
func (r GiftPlaceRule) FirstLegacyPlace() *int {
	if r.Type() != GiftPlaceRuleTypePlaces || len(r.places) == 0 {
		return nil
	}
	place := r.places[0]
	return &place
}

func normalizeGiftPlaceRulePlaces(places []int) ([]int, error) {
	if len(places) == 0 {
		return nil, fmt.Errorf("gift place rule places cannot be empty")
	}

	seen := make(map[int]bool, len(places))
	normalized := make([]int, 0, len(places))
	for _, place := range places {
		if place <= 0 {
			return nil, fmt.Errorf("gift place rule place must be greater than zero")
		}
		if seen[place] {
			continue
		}
		seen[place] = true
		normalized = append(normalized, place)
	}

	sort.Ints(normalized)
	return normalized, nil
}

func parseGiftPlaceRuleRange(token string) ([]int, error) {
	parts := strings.Split(token, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("gift place rule range %q is invalid", token)
	}

	start, err := parseGiftPlaceRulePositiveInt(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, err
	}
	end, err := parseGiftPlaceRulePositiveInt(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, err
	}
	if start > end {
		return nil, fmt.Errorf("gift place rule range %q is reversed", token)
	}

	places := make([]int, 0, end-start+1)
	for place := start; place <= end; place++ {
		places = append(places, place)
	}
	return places, nil
}

func parseGiftPlaceRulePositiveInt(token string) (int, error) {
	if token == "" {
		return 0, fmt.Errorf("gift place rule place cannot be empty")
	}

	value, err := strconv.Atoi(token)
	if err != nil {
		return 0, fmt.Errorf("gift place rule place %q must be numeric", token)
	}
	if value <= 0 {
		return 0, fmt.Errorf("gift place rule place must be greater than zero")
	}

	return value, nil
}

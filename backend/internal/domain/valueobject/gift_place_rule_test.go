package valueobject

import (
	"fmt"
	"testing"
)

func TestNewGiftPlaceRuleNone(t *testing.T) {
	rule := NewGiftPlaceRuleNone()

	if !rule.IsNone() {
		t.Fatalf("none rule should be none, got %s", rule.Type())
	}
	if rule.HasPlaceConstraint() {
		t.Fatal("none rule should not have place constraint")
	}
	if legacyPlace := rule.FirstLegacyPlace(); legacyPlace != nil {
		t.Fatalf("none rule legacy place = %d, want nil", *legacyPlace)
	}
}

func TestNewGiftPlaceRulePlacesNormalizesSortedUniquePositivePlaces(t *testing.T) {
	rule, err := NewGiftPlaceRulePlaces([]int{10, 3, 1, 3, 10, 2})
	if err != nil {
		t.Fatalf("NewGiftPlaceRulePlaces() error = %v", err)
	}

	assertGiftPlaceRulePlaces(t, rule.Places(), []int{1, 2, 3, 10})
	if rule.Type() != GiftPlaceRuleTypePlaces {
		t.Fatalf("rule type = %s, want %s", rule.Type(), GiftPlaceRuleTypePlaces)
	}
	if !rule.HasPlaceConstraint() {
		t.Fatal("places rule should have place constraint")
	}
	if legacyPlace := rule.FirstLegacyPlace(); legacyPlace == nil || *legacyPlace != 1 {
		t.Fatalf("legacy place = %v, want 1", legacyPlace)
	}
}

func TestParseGiftPlaceRulePlaces(t *testing.T) {
	rule, err := ParseGiftPlaceRulePlaces("1, 3, 10-15")
	if err != nil {
		t.Fatalf("ParseGiftPlaceRulePlaces() error = %v", err)
	}

	assertGiftPlaceRulePlaces(t, rule.Places(), []int{1, 3, 10, 11, 12, 13, 14, 15})
}

func TestGiftPlaceRulePlacesReturnsCopy(t *testing.T) {
	rule, err := NewGiftPlaceRulePlaces([]int{1, 2})
	if err != nil {
		t.Fatalf("NewGiftPlaceRulePlaces() error = %v", err)
	}

	places := rule.Places()
	places[0] = 99

	assertGiftPlaceRulePlaces(t, rule.Places(), []int{1, 2})
}

func TestNewGiftPlaceRulePlacesRejectsInvalidPlaces(t *testing.T) {
	tests := []struct {
		name   string
		places []int
	}{
		{name: "empty", places: nil},
		{name: "zero", places: []int{1, 0}},
		{name: "negative", places: []int{-1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewGiftPlaceRulePlaces(tt.places); err == nil {
				t.Fatal("NewGiftPlaceRulePlaces() error = nil, want error")
			}
		})
	}
}

func TestParseGiftPlaceRulePlacesRejectsInvalidInput(t *testing.T) {
	tests := []string{
		"",
		" ",
		"0",
		"-1",
		"1, 0",
		"abc",
		"1, abc",
		"15-10",
		"1-",
		"-3",
		"1,,3",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseGiftPlaceRulePlaces(input); err == nil {
				t.Fatal("ParseGiftPlaceRulePlaces() error = nil, want error")
			}
		})
	}
}

func TestNewGiftPlaceRuleLastN(t *testing.T) {
	rule, err := NewGiftPlaceRuleLastN(5)
	if err != nil {
		t.Fatalf("NewGiftPlaceRuleLastN() error = %v", err)
	}

	if rule.Type() != GiftPlaceRuleTypeLastN {
		t.Fatalf("rule type = %s, want %s", rule.Type(), GiftPlaceRuleTypeLastN)
	}
	if rule.LastCount() != 5 {
		t.Fatalf("last count = %d, want 5", rule.LastCount())
	}
	if !rule.HasPlaceConstraint() {
		t.Fatal("last_n rule should have place constraint")
	}
	if legacyPlace := rule.FirstLegacyPlace(); legacyPlace != nil {
		t.Fatalf("last_n legacy place = %d, want nil", *legacyPlace)
	}
}

func TestNewGiftPlaceRuleLastNRejectsInvalidCounts(t *testing.T) {
	for _, count := range []int{0, -1} {
		t.Run(fmt.Sprintf("%d", count), func(t *testing.T) {
			if _, err := NewGiftPlaceRuleLastN(count); err == nil {
				t.Fatal("NewGiftPlaceRuleLastN() error = nil, want error")
			}
		})
	}
}

func assertGiftPlaceRulePlaces(t *testing.T, got []int, want []int) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("places = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("places = %v, want %v", got, want)
		}
	}
}

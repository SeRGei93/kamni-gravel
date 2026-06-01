package valueobject

import "testing"

func TestNewCriteriaTypeAcceptsRandom(t *testing.T) {
	criteriaType, err := NewCriteriaType("random")
	if err != nil {
		t.Fatalf("NewCriteriaType(random) returned error: %v", err)
	}

	if criteriaType != CriteriaTypeRandom {
		t.Fatalf("criteria type mismatch: got %q, want %q", criteriaType, CriteriaTypeRandom)
	}

	if !criteriaType.IsValid() {
		t.Fatal("random criteria type should be valid")
	}
}

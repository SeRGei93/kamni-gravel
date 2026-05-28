package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"gravel_bot/internal/domain/valueobject"
)

func TestDecodeUpdateGiftRequestPlacePresence(t *testing.T) {
	withNull := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"place":null}`))
	req, err := decodeUpdateGiftRequest(withNull)
	if err != nil {
		t.Fatalf("decode null place error: %v", err)
	}
	if !req.PlaceSet {
		t.Fatal("place field should be marked as present")
	}
	if req.Place != nil {
		t.Fatalf("null place should decode to nil, got %v", *req.Place)
	}

	omitted := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"description":"Gift"}`))
	req, err = decodeUpdateGiftRequest(omitted)
	if err != nil {
		t.Fatalf("decode omitted place error: %v", err)
	}
	if req.PlaceSet {
		t.Fatal("omitted place should not be marked as present")
	}
}

func TestDecodeUpdateGiftRequestPlaceRulePresence(t *testing.T) {
	withNull := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"place_rule":null}`))
	req, err := decodeUpdateGiftRequest(withNull)
	if err != nil {
		t.Fatalf("decode null place_rule error: %v", err)
	}
	if !req.PlaceRuleSet {
		t.Fatal("place_rule field should be marked as present")
	}
	if !req.PlaceRule.IsNone() {
		t.Fatalf("null place_rule should decode to none, got %s", req.PlaceRule.Type())
	}

	omitted := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"description":"Gift"}`))
	req, err = decodeUpdateGiftRequest(omitted)
	if err != nil {
		t.Fatalf("decode omitted place_rule error: %v", err)
	}
	if req.PlaceRuleSet {
		t.Fatal("omitted place_rule should not be marked as present")
	}
}

func TestDecodeUpdateGiftRequestStructuredPlaceRules(t *testing.T) {
	places := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"place_rule":{"type":"places","places":[3,1,3]}}`))
	req, err := decodeUpdateGiftRequest(places)
	if err != nil {
		t.Fatalf("decode places rule error: %v", err)
	}
	assertDecodedGiftRulePlaces(t, req.PlaceRule, []int{1, 3})

	lastN := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"place_rule":{"type":"last_n","last_count":5}}`))
	req, err = decodeUpdateGiftRequest(lastN)
	if err != nil {
		t.Fatalf("decode last_n rule error: %v", err)
	}
	if req.PlaceRule.Type() != valueobject.GiftPlaceRuleTypeLastN || req.PlaceRule.LastCount() != 5 {
		t.Fatalf("place_rule = %s/%d, want last_n/5", req.PlaceRule.Type(), req.PlaceRule.LastCount())
	}
}

func TestDecodeUpdateGiftRequestRejectsInvalidPlaceRule(t *testing.T) {
	tests := []string{
		`{"place_rule":{"type":"places","places":[]}}`,
		`{"place_rule":{"type":"places","places":[0]}}`,
		`{"place_rule":{"type":"last_n","last_count":0}}`,
		`{"place_rule":{"type":"last_n"}}`,
		`{"place_rule":{"type":"unknown"}}`,
	}

	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(body))
			if _, err := decodeUpdateGiftRequest(req); err == nil {
				t.Fatal("decodeUpdateGiftRequest() error = nil, want error")
			}
		})
	}
}

func TestDecodeUpdateGiftRequestPlaceRuleWinsOverLegacyPlacePayload(t *testing.T) {
	request := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"place":2,"place_rule":{"type":"places","places":[10]}}`))
	req, err := decodeUpdateGiftRequest(request)
	if err != nil {
		t.Fatalf("decode request error: %v", err)
	}

	if !req.PlaceSet || req.Place == nil || *req.Place != 2 {
		t.Fatalf("legacy place decode mismatch: set=%t place=%v", req.PlaceSet, req.Place)
	}
	assertDecodedGiftRulePlaces(t, req.PlaceRule, []int{10})
}

func assertDecodedGiftRulePlaces(t *testing.T, rule valueobject.GiftPlaceRule, want []int) {
	t.Helper()

	got := rule.Places()
	if len(got) != len(want) {
		t.Fatalf("place_rule places = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("place_rule places = %v, want %v", got, want)
		}
	}
}

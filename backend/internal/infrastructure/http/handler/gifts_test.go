package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
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

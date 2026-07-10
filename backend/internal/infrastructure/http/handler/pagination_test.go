package handler

import (
	"net/http/httptest"
	"testing"
)

func TestParsePageParams(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantPage   int
		wantSize   int
		wantOffset int
		wantAll    bool
	}{
		{name: "defaults when empty", query: "", wantPage: 1, wantSize: 50, wantOffset: 0},
		{name: "explicit valid", query: "?page=3&page_size=75", wantPage: 3, wantSize: 75, wantOffset: 150},
		{name: "page_size below min clamped up", query: "?page_size=10", wantPage: 1, wantSize: 50, wantOffset: 0},
		{name: "page_size above max clamped down", query: "?page_size=200", wantPage: 1, wantSize: 100, wantOffset: 0},
		{name: "page_size at bounds 100", query: "?page=2&page_size=100", wantPage: 2, wantSize: 100, wantOffset: 100},
		{name: "non-numeric page_size falls back", query: "?page_size=abc", wantPage: 1, wantSize: 50, wantOffset: 0},
		{name: "non-numeric page falls back", query: "?page=abc&page_size=50", wantPage: 1, wantSize: 50, wantOffset: 0},
		{name: "zero page falls back to 1", query: "?page=0", wantPage: 1, wantSize: 50, wantOffset: 0},
		{name: "negative page falls back to 1", query: "?page=-5", wantPage: 1, wantSize: 50, wantOffset: 0},
		{name: "all disables pagination", query: "?page=3&page_size=all", wantPage: 1, wantSize: 0, wantOffset: 0, wantAll: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/criteria"+tt.query, nil)
			pp := ParsePageParams(r)

			if pp.Page != tt.wantPage {
				t.Errorf("page mismatch: got %d, want %d", pp.Page, tt.wantPage)
			}
			if pp.PageSize != tt.wantSize {
				t.Errorf("page_size mismatch: got %d, want %d", pp.PageSize, tt.wantSize)
			}
			if pp.Limit != tt.wantSize {
				t.Errorf("limit mismatch: got %d, want %d", pp.Limit, tt.wantSize)
			}
			if pp.Offset != tt.wantOffset {
				t.Errorf("offset mismatch: got %d, want %d", pp.Offset, tt.wantOffset)
			}
			if pp.All != tt.wantAll {
				t.Errorf("all mismatch: got %t, want %t", pp.All, tt.wantAll)
			}
		})
	}
}

func TestPageParamsTotalPages(t *testing.T) {
	tests := []struct {
		pageSize int
		total    int
		want     int
	}{
		{50, 0, 0},
		{50, 1, 1},
		{50, 50, 1},
		{50, 51, 2},
		{100, 196, 2},
		{50, 196, 4},
	}
	for _, tt := range tests {
		pp := PageParams{PageSize: tt.pageSize}
		if got := pp.TotalPages(tt.total); got != tt.want {
			t.Errorf("TotalPages(pageSize=%d,total=%d) = %d, want %d", tt.pageSize, tt.total, got, tt.want)
		}
	}
}

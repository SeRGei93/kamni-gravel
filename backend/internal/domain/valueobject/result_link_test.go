package valueobject

import (
	"strings"
	"testing"
)

func TestNewResultLinkAcceptsStravaLinks(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantURL string
	}{
		{
			name:    "activity link",
			rawURL:  "https://www.strava.com/activities/14758223172",
			wantURL: "https://www.strava.com/activities/14758223172",
		},
		{
			name:    "activity link with query and fragment",
			rawURL:  "HTTPS://WWW.STRAVA.COM/activities/14758223172?utm_source=telegram#comments",
			wantURL: "https://www.strava.com/activities/14758223172?utm_source=telegram#comments",
		},
		{
			name:    "app link preserves case-sensitive token",
			rawURL:  "https://strava.app.link/AbC123_Token-9",
			wantURL: "https://strava.app.link/AbC123_Token-9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link, err := NewResultLink(tt.rawURL)
			if err != nil {
				t.Fatalf("NewResultLink() error = %v", err)
			}
			if link.URL != tt.wantURL {
				t.Fatalf("NewResultLink().URL = %q, want %q", link.URL, tt.wantURL)
			}
			if !link.IsStrava() {
				t.Fatal("NewResultLink().IsStrava() = false, want true")
			}
		})
	}
}

func TestNewResultLinkRejectsNonStravaLinks(t *testing.T) {
	tests := []struct {
		name       string
		rawURL     string
		wantErrSub string
	}{
		{
			name:       "komoot tour",
			rawURL:     "https://www.komoot.com/tour/2308024419",
			wantErrSub: "Only Strava links are accepted",
		},
		{
			name:       "unsupported host",
			rawURL:     "https://example.com/activities/14758223172",
			wantErrSub: "Only Strava links are accepted",
		},
		{
			name:       "strava activity without numeric id",
			rawURL:     "https://www.strava.com/activities/not-a-number",
			wantErrSub: "invalid Strava URL format",
		},
		{
			name:       "missing scheme",
			rawURL:     "www.strava.com/activities/14758223172",
			wantErrSub: "invalid Strava URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewResultLink(tt.rawURL)
			if err == nil {
				t.Fatal("NewResultLink() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Fatalf("NewResultLink() error = %q, want substring %q", err.Error(), tt.wantErrSub)
			}
		})
	}
}

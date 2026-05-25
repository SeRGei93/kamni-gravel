package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTelegramWebAppAuthAcceptsValidInitData(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()

	initData := signedInitData(t, token, url.Values{
		"auth_date": {strconv.FormatInt(now.Unix(), 10)},
		"query_id":  {"query-1"},
		"user":      {telegramWebAppUserJSON(t, telegramWebAppTestUser{ID: 42, FirstName: "Alex", Username: "alex"})},
	})

	var gotUser *TelegramWebAppUser
	handler := TelegramWebAppAuthWithConfig(TelegramWebAppAuthConfig{
		BotToken: token,
		Now:      func() time.Time { return now },
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetTelegramWebAppUserFromContext(r.Context())
		if !ok {
			t.Fatal("telegram webapp user missing from context")
		}
		gotUser = user
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/miniapp/session", nil)
	req.Header.Set(TelegramInitDataHeader, initData)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusNoContent, rr.Body.String())
	}
	if gotUser == nil || gotUser.ID != 42 || gotUser.Username != "alex" || gotUser.FirstName != "Alex" {
		t.Fatalf("context user mismatch: %#v", gotUser)
	}
}

func TestTelegramWebAppAuthRejectsInvalidInitData(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()

	validValues := url.Values{
		"auth_date": {strconv.FormatInt(now.Unix(), 10)},
		"user":      {telegramWebAppUserJSON(t, telegramWebAppTestUser{ID: 42, FirstName: "Alex"})},
	}

	tests := []struct {
		name     string
		initData string
	}{
		{
			name:     "missing header",
			initData: "",
		},
		{
			name:     "invalid signature",
			initData: initDataWithHash(t, signedInitData(t, token, validValues), "bad"),
		},
		{
			name: "expired auth date",
			initData: signedInitData(t, token, url.Values{
				"auth_date": {strconv.FormatInt(now.Add(-25*time.Hour).Unix(), 10)},
				"user":      {telegramWebAppUserJSON(t, telegramWebAppTestUser{ID: 42})},
			}),
		},
		{
			name: "future auth date",
			initData: signedInitData(t, token, url.Values{
				"auth_date": {strconv.FormatInt(now.Add(10*time.Minute).Unix(), 10)},
				"user":      {telegramWebAppUserJSON(t, telegramWebAppTestUser{ID: 42})},
			}),
		},
		{
			name: "missing user payload",
			initData: signedInitData(t, token, url.Values{
				"auth_date": {strconv.FormatInt(now.Unix(), 10)},
			}),
		},
		{
			name: "bot user",
			initData: signedInitData(t, token, url.Values{
				"auth_date": {strconv.FormatInt(now.Unix(), 10)},
				"user":      {telegramWebAppUserJSON(t, telegramWebAppTestUser{ID: 42, IsBot: true})},
			}),
		},
		{
			name: "invalid user id",
			initData: signedInitData(t, token, url.Values{
				"auth_date": {strconv.FormatInt(now.Unix(), 10)},
				"user":      {telegramWebAppUserJSON(t, telegramWebAppTestUser{ID: 0})},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := TelegramWebAppAuthWithConfig(TelegramWebAppAuthConfig{
				BotToken: token,
				Now:      func() time.Time { return now },
			})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("next handler should not be called")
			}))

			req := httptest.NewRequest(http.MethodGet, "/api/miniapp/session", nil)
			if tt.initData != "" {
				req.Header.Set(TelegramInitDataHeader, tt.initData)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusUnauthorized, rr.Body.String())
			}
		})
	}
}

func TestCORSAllowsTelegramInitDataHeader(t *testing.T) {
	tests := []struct {
		name    string
		handler http.Handler
	}{
		{
			name:    "default CORS",
			handler: CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})),
		},
		{
			name:    "origin CORS",
			handler: CORSWithOrigins([]string{"https://example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, "/api/miniapp/session", nil)
			req.Header.Set("Origin", "https://example.com")
			rr := httptest.NewRecorder()

			tt.handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusNoContent {
				t.Fatalf("status mismatch: got %d, want %d", rr.Code, http.StatusNoContent)
			}
			if got := rr.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, TelegramInitDataHeader) {
				t.Fatalf("allow headers mismatch: got %q, want it to contain %q", got, TelegramInitDataHeader)
			}
		})
	}
}

type telegramWebAppTestUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

func telegramWebAppUserJSON(t *testing.T, user telegramWebAppTestUser) string {
	t.Helper()

	payload, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("marshal user: %v", err)
	}

	return string(payload)
}

func signedInitData(t *testing.T, token string, values url.Values) string {
	t.Helper()

	copied := cloneValues(values)
	copied.Set("hash", telegramWebAppHash(token, values))
	return copied.Encode()
}

func initDataWithHash(t *testing.T, initData string, hash string) string {
	t.Helper()

	values, err := url.ParseQuery(initData)
	if err != nil {
		t.Fatalf("parse init data: %v", err)
	}
	values.Set("hash", hash)
	return values.Encode()
}

func telegramWebAppHash(token string, values url.Values) string {
	pairs := make([]string, 0, len(values))
	for key, value := range values {
		pairs = append(pairs, key+"="+value[0])
	}
	sort.Strings(pairs)

	secretHMAC := hmac.New(sha256.New, []byte("WebAppData"))
	secretHMAC.Write([]byte(token))
	secret := secretHMAC.Sum(nil)

	dataHMAC := hmac.New(sha256.New, secret)
	dataHMAC.Write([]byte(strings.Join(pairs, "\n")))
	return fmt.Sprintf("%x", dataHMAC.Sum(nil))
}

package middleware

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	telegrambot "github.com/go-telegram/bot"

	"gravel_bot/internal/infrastructure/http/response"
)

const (
	TelegramInitDataHeader = "X-Telegram-Init-Data"

	defaultTelegramWebAppMaxAge    = 24 * time.Hour
	defaultTelegramWebAppClockSkew = 5 * time.Minute
)

const telegramWebAppUserContextKey contextKey = "telegram_webapp_user"

// TelegramWebAppUser содержит безопасные данные пользователя из Telegram Mini App.
type TelegramWebAppUser struct {
	ID           int64
	Username     string
	FirstName    string
	LastName     string
	LanguageCode string
	PhotoURL     string
	IsPremium    bool
}

type TelegramWebAppAuthConfig struct {
	BotToken  string
	MaxAge    time.Duration
	ClockSkew time.Duration
	Now       func() time.Time
}

// TelegramWebAppAuth проверяет Telegram.WebApp.initData из HTTP-заголовка.
func TelegramWebAppAuth(botToken string) func(http.Handler) http.Handler {
	return TelegramWebAppAuthWithConfig(TelegramWebAppAuthConfig{BotToken: botToken})
}

func TelegramWebAppAuthWithConfig(cfg TelegramWebAppAuthConfig) func(http.Handler) http.Handler {
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = defaultTelegramWebAppMaxAge
	}
	if cfg.ClockSkew <= 0 {
		cfg.ClockSkew = defaultTelegramWebAppClockSkew
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			rawInitData := r.Header.Get(TelegramInitDataHeader)
			if rawInitData == "" {
				log.Printf("WARN Telegram miniapp auth failed: reason=missing_init_data path=%s", path)
				response.Unauthorized(w, "Missing Telegram init data")
				return
			}
			if cfg.BotToken == "" {
				log.Printf("ERROR Telegram miniapp auth failed: reason=missing_bot_token path=%s", path)
				response.Unauthorized(w, "Invalid Telegram init data")
				return
			}

			values, err := url.ParseQuery(rawInitData)
			if err != nil {
				log.Printf("WARN Telegram miniapp auth failed: reason=malformed_init_data path=%s error=%v", path, err)
				response.Unauthorized(w, "Invalid Telegram init data")
				return
			}
			if values.Get("user") == "" {
				log.Printf("WARN Telegram miniapp auth failed: reason=missing_user_payload path=%s", path)
				response.Unauthorized(w, "Invalid Telegram init data")
				return
			}

			user, ok := telegrambot.ValidateWebappRequest(cloneValues(values), cfg.BotToken)
			if !ok || user == nil {
				log.Printf("WARN Telegram miniapp auth failed: reason=invalid_signature path=%s", path)
				response.Unauthorized(w, "Invalid Telegram init data")
				return
			}

			authTime, err := parseTelegramAuthDate(values.Get("auth_date"))
			if err != nil {
				log.Printf("WARN Telegram miniapp auth failed: reason=invalid_auth_date path=%s error=%v", path, err)
				response.Unauthorized(w, "Invalid Telegram init data")
				return
			}

			now := cfg.Now().UTC()
			if authTime.After(now.Add(cfg.ClockSkew)) {
				log.Printf("WARN Telegram miniapp auth failed: reason=future_auth_date path=%s auth_date=%d now=%d", path, authTime.Unix(), now.Unix())
				response.Unauthorized(w, "Invalid Telegram init data")
				return
			}
			if now.Sub(authTime) > cfg.MaxAge {
				log.Printf("WARN Telegram miniapp auth failed: reason=expired_init_data path=%s auth_date=%d now=%d max_age=%s", path, authTime.Unix(), now.Unix(), cfg.MaxAge)
				response.Unauthorized(w, "Expired Telegram init data")
				return
			}
			if user.ID <= 0 {
				log.Printf("WARN Telegram miniapp auth failed: reason=invalid_user_id path=%s telegram_user_id=%d", path, user.ID)
				response.Unauthorized(w, "Invalid Telegram init data")
				return
			}
			if user.IsBot {
				log.Printf("WARN Telegram miniapp auth failed: reason=bot_user path=%s telegram_user_id=%d", path, user.ID)
				response.Unauthorized(w, "Invalid Telegram init data")
				return
			}

			webAppUser := &TelegramWebAppUser{
				ID:           user.ID,
				Username:     user.Username,
				FirstName:    user.FirstName,
				LastName:     user.LastName,
				LanguageCode: user.LanguageCode,
				PhotoURL:     user.PhotoURL,
				IsPremium:    user.IsPremium,
			}
			log.Printf("INFO Telegram miniapp auth succeeded: telegram_user_id=%d path=%s", webAppUser.ID, path)

			ctx := context.WithValue(r.Context(), telegramWebAppUserContextKey, webAppUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetTelegramWebAppUserFromContext(ctx context.Context) (*TelegramWebAppUser, bool) {
	user, ok := ctx.Value(telegramWebAppUserContextKey).(*TelegramWebAppUser)
	return user, ok
}

func cloneValues(values url.Values) url.Values {
	cloned := make(url.Values, len(values))
	for key, value := range values {
		cloned[key] = append([]string(nil), value...)
	}
	return cloned
}

func parseTelegramAuthDate(raw string) (time.Time, error) {
	unixTime, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	if unixTime <= 0 {
		return time.Time{}, strconv.ErrSyntax
	}

	return time.Unix(unixTime, 0).UTC(), nil
}

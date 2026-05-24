package middleware

import (
	"context"
	"net/http"
	"strings"

	"gravel_bot/internal/infrastructure/http/response"
	"gravel_bot/internal/pkg/jwt"
)

// contextKey тип для ключей контекста
type contextKey string

const (
	// UserContextKey ключ для хранения данных пользователя в контексте
	UserContextKey contextKey = "user"
)

// Auth middleware для проверки JWT токена
func Auth(jwtManager *jwt.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем токен из заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "Missing authorization header")
				return
			}

			// Проверяем формат: "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.Unauthorized(w, "Invalid authorization header format")
				return
			}

			tokenString := parts[1]

			// Валидируем токен
			claims, err := jwtManager.ValidateToken(tokenString)
			if err != nil {
				if err == jwt.ErrExpiredToken {
					response.Unauthorized(w, "Token expired")
				} else {
					response.Unauthorized(w, "Invalid token")
				}
				return
			}

			// Сохраняем claims в контексте
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext извлекает данные пользователя из контекста
func GetUserFromContext(ctx context.Context) (*jwt.Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*jwt.Claims)
	return claims, ok
}

// RequireRole middleware для проверки роли пользователя
func RequireRole(jwtManager *jwt.Manager, allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserFromContext(r.Context())
			if !ok {
				response.Unauthorized(w, "User not found in context")
				return
			}

			// Проверяем роль
			roleAllowed := false
			for _, role := range allowedRoles {
				if claims.Role == role {
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				response.Forbidden(w, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

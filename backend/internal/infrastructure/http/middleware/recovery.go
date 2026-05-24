package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"gravel_bot/internal/infrastructure/http/response"
)

// Recovery восстанавливается после паники
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v\n%s", err, debug.Stack())
				response.InternalServerError(w, "Internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

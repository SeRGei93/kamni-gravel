package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/http/response"
	"gravel_bot/internal/pkg/jwt"
)

// AuthHandler обрабатывает запросы авторизации
type AuthHandler struct {
	adminRepo  repository.AdminRepository
	jwtManager *jwt.Manager
}

// NewAuthHandler создаёт новый handler
func NewAuthHandler(
	adminRepo repository.AdminRepository,
	jwtManager *jwt.Manager,
) *AuthHandler {
	return &AuthHandler{
		adminRepo:  adminRepo,
		jwtManager: jwtManager,
	}
}

// LoginRequest представляет запрос на вход
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse представляет ответ на вход
type LoginResponse struct {
	*jwt.TokenPair
	User UserInfo `json:"user"`
}

// UserInfo представляет информацию о пользователе
type UserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Login обрабатывает вход в систему
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Валидация
	if req.Username == "" || req.Password == "" {
		response.BadRequest(w, "Username and password are required")
		return
	}

	// Находим админа по username
	admin, err := h.adminRepo.FindByUsername(r.Context(), req.Username)
	if err != nil {
		// Если админ не найден - возвращаем Unauthorized (не раскрываем, что пользователя нет)
		log.Printf("Admin not found or error: %v", err)
		response.Unauthorized(w, "Invalid credentials")
		return
	}

	// Проверяем пароль
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		response.Unauthorized(w, "Invalid credentials")
		return
	}

	// Генерируем токены
	tokenPair, err := h.jwtManager.GenerateTokenPair(admin.ID, admin.Username, string(admin.Role))
	if err != nil {
		log.Printf("Error generating tokens: %v", err)
		response.InternalServerError(w, "Failed to generate tokens")
		return
	}

	// Обновляем время последнего входа
	if err := h.adminRepo.UpdateLastLogin(r.Context(), admin.ID); err != nil {
		log.Printf("Error updating last login: %v", err)
		// Не возвращаем ошибку, это не критично
	}

	// Возвращаем токены и информацию о пользователе
	resp := LoginResponse{
		TokenPair: tokenPair,
		User: UserInfo{
			ID:       admin.ID,
			Username: admin.Username,
			Role:     string(admin.Role),
		},
	}

	response.Success(w, resp)
}

// RefreshRequest представляет запрос на обновление токена
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh обрабатывает обновление токенов
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if req.RefreshToken == "" {
		response.BadRequest(w, "Refresh token is required")
		return
	}

	// Обновляем токены
	tokenPair, err := h.jwtManager.RefreshTokens(req.RefreshToken)
	if err != nil {
		if err == jwt.ErrExpiredToken {
			response.Unauthorized(w, "Refresh token expired")
		} else {
			response.Unauthorized(w, "Invalid refresh token")
		}
		return
	}

	response.Success(w, tokenPair)
}

// Me возвращает информацию о текущем пользователе
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	// Получаем claims из контекста (установлены Auth middleware)
	claims, ok := r.Context().Value("user").(*jwt.Claims)
	if !ok {
		response.Unauthorized(w, "User not found in context")
		return
	}

	// Получаем полную информацию об админе
	admin, err := h.adminRepo.FindByID(r.Context(), claims.UserID)
	if err != nil {
		log.Printf("Admin not found or error: %v", err)
		response.NotFound(w, "User not found")
		return
	}

	userInfo := UserInfo{
		ID:       admin.ID,
		Username: admin.Username,
		Role:     string(admin.Role),
	}

	response.Success(w, userInfo)
}

// HashPassword хэширует пароль (вспомогательная функция)
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CreateAdmin создаёт нового админа (для внутреннего использования)
func (h *AuthHandler) CreateAdmin(ctx context.Context, username, password, role string) error {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return err
	}

	admin := &entity.Admin{
		Username:     username,
		PasswordHash: hashedPassword,
		Role:         entity.AdminRole(role),
	}

	return h.adminRepo.Create(ctx, admin)
}

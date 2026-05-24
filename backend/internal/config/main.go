package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config представляет полную конфигурацию приложения
type Config struct {
	Env string
	DB  DBConfig

	Bot BotConfig
	API APIConfig
}

// DBConfig представляет конфигурацию подключения к базе данных
type DBConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

// BotConfig представляет конфигурацию Telegram бота
type BotConfig struct {
	Token         string
	AdminChat     int64
	PublicChat    int64
	Debug         bool
	SessionTimeout time.Duration
}

// APIConfig представляет конфигурацию HTTP API сервера
type APIConfig struct {
	Host          string
	Port          int
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL  time.Duration
	AllowedOrigins []string
}

// MustLoad загружает конфигурацию из переменных окружения
// Если конфигурация не может быть загружена, программа завершается с ошибкой
func MustLoad(_ string) *Config {
	cfg := &Config{
		Env: getEnv("ENV", "local"),

		DB: DBConfig{
			Host:     getEnvRequired("DB_HOST"),
			Port:     getEnvInt("DB_PORT", 5432),
			Name:     getEnvRequired("DB_NAME"),
			User:     getEnvRequired("DB_USER"),
			Password: getEnvRequired("DB_PASSWORD"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},

		Bot: BotConfig{
			Token:          getEnvRequired("BOT_TOKEN"),
			AdminChat:      getEnvInt64("ADMIN_CHAT_ID", 0),
			PublicChat:     getEnvInt64("PUBLIC_CHAT_ID", 0),
			Debug:          getEnvBool("BOT_DEBUG", false),
			SessionTimeout: getEnvDuration("BOT_SESSION_TIMEOUT", 30*time.Minute),
		},

		API: APIConfig{
			Host:           getEnv("API_HOST", "0.0.0.0"),
			Port:           getEnvInt("API_PORT", 8080),
			JWTSecret:      getEnvRequired("JWT_SECRET"),
			JWTAccessTTL:   getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
			JWTRefreshTTL:  getEnvDuration("JWT_REFRESH_TTL", 168*time.Hour),
			AllowedOrigins: parseAllowedOrigins(getEnv("ALLOWED_ORIGINS", "*")),
		},
	}

	// Валидация обязательных полей
	if err := validate(cfg); err != nil {
		log.Fatalf("config validation failed: %v", err)
	}

	return cfg
}

// getEnv получает переменную окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvRequired получает обязательную переменную окружения
func getEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return value
}

// getEnvInt получает переменную окружения как int или возвращает значение по умолчанию
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("invalid value for %s: %s, using default: %d", key, value, defaultValue)
		return defaultValue
	}
	return intValue
}

// getEnvInt64 получает переменную окружения как int64 или возвращает значение по умолчанию
func getEnvInt64(key string, defaultValue int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Printf("invalid value for %s: %s, using default: %d", key, value, defaultValue)
		return defaultValue
	}
	return intValue
}

// getEnvBool получает переменную окружения как bool или возвращает значение по умолчанию
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("invalid value for %s: %s, using default: %v", key, value, defaultValue)
		return defaultValue
	}
	return boolValue
}

// getEnvDuration получает переменную окружения как time.Duration или возвращает значение по умолчанию
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("invalid duration for %s: %s, using default: %v", key, value, defaultValue)
		return defaultValue
	}
	return duration
}

// parseAllowedOrigins парсит строку с origins в массив строк
func parseAllowedOrigins(value string) []string {
	if value == "*" {
		return []string{"*"}
	}
	
	origins := strings.Split(value, ",")
	result := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			result = append(result, origin)
		}
	}
	
	if len(result) == 0 {
		return []string{"*"}
	}
	
	return result
}

// validate проверяет корректность конфигурации
func validate(cfg *Config) error {
	if cfg.Bot.Token == "" {
		return fmt.Errorf("BOT_TOKEN is required")
	}

	if cfg.API.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if cfg.DB.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}

	if cfg.DB.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}

	if cfg.DB.User == "" {
		return fmt.Errorf("DB_USER is required")
	}

	if cfg.DB.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}

	return nil
}

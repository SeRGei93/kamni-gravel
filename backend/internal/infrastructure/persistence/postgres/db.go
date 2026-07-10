package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// Config содержит параметры подключения к PostgreSQL
type Config struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

// InitDB инициализирует подключение к PostgreSQL базе данных
func InitDB(cfg Config) (*sql.DB, error) {
	// Формируем строку подключения. timezone задаёт зону сессии: TIMESTAMPTZ
	// значения возвращаются с офсетом +03, и вычисления вида RideDate()
	// (дата проезда = день старта) идут по минскому календарному дню.
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s timezone=Europe/Minsk",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)

	// Открываем подключение
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Настройки пула подключений для PostgreSQL
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}

// Close закрывает подключение к БД
func Close(db *sql.DB) error {
	if db != nil {
		return db.Close()
	}
	return nil
}

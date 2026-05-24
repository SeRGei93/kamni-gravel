package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"gravel_bot/internal/infrastructure/persistence/postgres"

	"github.com/pressly/goose/v3"
)

const (
	migrationsDir = "internal/infrastructure/migrations"
)

func main() {
	var command string
	var version int64
	flag.StringVar(&command, "command", "up", "Migration command: up, down, down-to, status, version")
	flag.Int64Var(&version, "version", 0, "Target version for down-to command")
	flag.Parse()

	// Если команда передана как аргумент (не флаг)
	if flag.NArg() > 0 {
		command = flag.Arg(0)
		if flag.NArg() > 1 {
			// Пытаемся распарсить версию из аргумента
			fmt.Sscanf(flag.Arg(1), "%d", &version)
		}
	}

	// Получаем параметры подключения из переменных окружения
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := 5432
	if portStr := os.Getenv("DB_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			dbPort = p
		}
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "gravel_bot"
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "gravel"
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "gravel_password"
	}

	dbSSLMode := os.Getenv("DB_SSLMODE")
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	// Инициализируем БД
	db, err := postgres.InitDB(postgres.Config{
		Host:     dbHost,
		Port:     dbPort,
		Database: dbName,
		User:     dbUser,
		Password: dbPassword,
		SSLMode:  dbSSLMode,
	})
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer db.Close()

	// Устанавливаем диалект для goose
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Failed to set dialect: %v", err)
	}

	// Выполняем команду миграции
	switch command {
	case "up":
		if err := goose.Up(db, migrationsDir); err != nil {
			log.Fatalf("Failed to run migrations up: %v", err)
		}
		fmt.Println("✅ Migrations applied successfully")
	case "down":
		if err := goose.Down(db, migrationsDir); err != nil {
			log.Fatalf("Failed to run migrations down: %v", err)
		}
		fmt.Println("✅ Migrations rolled back successfully")
	case "down-to":
		if version < 0 {
			log.Fatalf("Version must be >= 0")
		}
		if err := goose.DownTo(db, migrationsDir, version); err != nil {
			log.Fatalf("Failed to run migrations down-to: %v", err)
		}
		fmt.Printf("✅ Migrations rolled back to version %d successfully\n", version)
	case "status":
		if err := goose.Status(db, migrationsDir); err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}
	case "version":
		version, err := goose.GetDBVersion(db)
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Printf("Current version: %d\n", version)
	default:
		log.Fatalf("Unknown command: %s. Use: up, down, status, version", command)
	}
}

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gravel_bot/internal/config"
	"gravel_bot/internal/infrastructure/http"
	"gravel_bot/internal/infrastructure/persistence/postgres"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.MustLoad("")

	// Инициализируем БД
	db, err := postgres.InitDB(postgres.Config{
		Host:     cfg.DB.Host,
		Port:     cfg.DB.Port,
		Database: cfg.DB.Name,
		User:     cfg.DB.User,
		Password: cfg.DB.Password,
		SSLMode:  cfg.DB.SSLMode,
	})
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := postgres.Close(db); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Создаём репозитории
	userRepo := postgres.NewUserRepository(db)
	eventRepo := postgres.NewEventRepository(db)
	participantRepo := postgres.NewParticipantRepository(db)
	resultRepo := postgres.NewResultRepository(db)
	giftRepo := postgres.NewGiftRepository(db)
	criteriaRepo := postgres.NewCriteriaRepository(db)
	prizeAssignmentRepo := postgres.NewPrizeAssignmentRepository(db)
	userBlacklistRepo := postgres.NewUserBlacklistRepository(db)
	adminRepo := postgres.NewAdminRepository(db)

	// Создаём HTTP сервер
	server := http.NewServer(
		http.Config{
			Host:            cfg.API.Host,
			Port:            cfg.API.Port,
			JWTSecret:       cfg.API.JWTSecret,
			JWTAccessTTL:    cfg.API.JWTAccessTTL,
			JWTRefreshTTL:   cfg.API.JWTRefreshTTL,
			AllowedOrigins:  cfg.API.AllowedOrigins,
			BotToken:        cfg.Bot.Token,
			FileStoragePath: cfg.Files.Path,
		},
		userRepo,
		eventRepo,
		participantRepo,
		resultRepo,
		giftRepo,
		criteriaRepo,
		prizeAssignmentRepo,
		userBlacklistRepo,
		adminRepo,
	)

	// Канал для сигналов ОС
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Запускаем сервер в горутине
	serverErrChan := make(chan error, 1)
	go func() {
		log.Printf("Starting HTTP server on %s:%d...", cfg.API.Host, cfg.API.Port)
		if err := server.Start(); err != nil {
			serverErrChan <- err
		}
	}()

	// Ждём сигнала завершения или ошибки
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v. Shutting down gracefully...", sig)

		// Даём время на завершение работы
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer shutdownCancel()

		// Останавливаем сервер
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		} else {
			log.Println("Server stopped gracefully")
		}

		// Проверяем, не было ли ошибки запуска
		select {
		case err := <-serverErrChan:
			if err != nil {
				log.Printf("Server error: %v", err)
			}
		default:
		}
	case err := <-serverErrChan:
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}

	log.Println("API server shutdown complete")
}

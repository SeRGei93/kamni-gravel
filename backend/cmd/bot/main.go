package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gravel_bot/internal/config"
	"gravel_bot/internal/infrastructure/persistence/postgres"
	"gravel_bot/internal/infrastructure/telegram"
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
	giftCriteriaRepo := postgres.NewGiftCriteriaRepository(db)
	prizeAssignmentRepo := postgres.NewPrizeAssignmentRepository(db)

	// Создаём бота
	bot, err := telegram.NewBot(
		telegram.Config{
			Token:          cfg.Bot.Token,
			Debug:          cfg.Bot.Debug,
			SessionTimeout: cfg.Bot.SessionTimeout,
		},
		userRepo,
		eventRepo,
		participantRepo,
		resultRepo,
		giftRepo,
		criteriaRepo,
		giftCriteriaRepo,
		prizeAssignmentRepo,
	)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Настраиваем graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Канал для сигналов ОС
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Запускаем бота в горутине
	botErrChan := make(chan error, 1)
	go func() {
		log.Println("Starting Telegram bot...")
		if err := bot.Start(ctx); err != nil {
			botErrChan <- err
		}
	}()

	// Ждём сигнала завершения или ошибки
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v. Shutting down gracefully...", sig)
		cancel()
		
		// Даём время на завершение работы
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		// Ждём завершения бота
		select {
		case <-shutdownCtx.Done():
			log.Println("Shutdown timeout exceeded, forcing exit")
		case err := <-botErrChan:
			if err != nil && err != context.Canceled {
				log.Printf("Bot stopped with error: %v", err)
			} else {
				log.Println("Bot stopped gracefully")
			}
		}
	case err := <-botErrChan:
		if err != nil {
			log.Fatalf("Bot error: %v", err)
		}
	}

	log.Println("Bot shutdown complete")
}

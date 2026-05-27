package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/http/handler"
	"gravel_bot/internal/infrastructure/http/middleware"
	"gravel_bot/internal/infrastructure/storage"
	"gravel_bot/internal/pkg/jwt"
)

// Server представляет HTTP сервер
type Server struct {
	router     *chi.Mux
	httpServer *http.Server

	// Repositories
	userRepo            repository.UserRepository
	eventRepo           repository.EventRepository
	participantRepo     repository.ParticipantRepository
	resultRepo          repository.ResultRepository
	giftRepo            repository.GiftRepository
	criteriaRepo        repository.CriteriaRepository
	prizeAssignmentRepo repository.PrizeAssignmentRepository
	userBlacklistRepo   repository.UserBlacklistRepository
	adminRepo           repository.AdminRepository

	// Command handlers
	registerParticipantHandler       *command.RegisterParticipantHandler
	updateGiftHandler                *command.UpdateGiftHandler
	submitResultHandler              *command.SubmitResultHandler
	assignPrizeHandler               *command.AssignPrizeHandler
	addUserBlacklistHandler          *command.AddUserBlacklistHandler
	updateUserBlacklistReasonHandler *command.UpdateUserBlacklistReasonHandler
	removeUserBlacklistHandler       *command.RemoveUserBlacklistHandler
	deleteParticipantHandler         *command.DeleteParticipantHandler

	// Query handlers
	getParticipantsHandler        *query.GetParticipantsHandler
	getParticipantByIDHandler     *query.GetParticipantByIDHandler
	getGiftsHandler               *query.GetGiftsHandler
	getGiftByIDHandler            *query.GetGiftByIDHandler
	getEventsHandler              *query.GetEventsHandler
	getEventByIDHandler           *query.GetEventByIDHandler
	getPrizeAssignmentsHandler    *query.GetPrizeAssignmentsHandler
	getPrizeAssignmentByIDHandler *query.GetPrizeAssignmentByIDHandler
	getStatsHandler               *query.GetStatsHandler
	listUserBlacklistHandler      *query.ListUserBlacklistHandler
	isUserBlacklistedHandler      *query.IsUserBlacklistedHandler

	// HTTP handlers
	authHandler              *handler.AuthHandler
	eventsHandler            *handler.EventsHandler
	participantsHandler      *handler.ParticipantsHandler
	resultsHandler           *handler.ResultsHandler
	giftsHandler             *handler.GiftsHandler
	criteriaHandler          *handler.CriteriaHandler
	prizeAssignmentsHandler  *handler.PrizeAssignmentsHandler
	prizeDistributionHandler *handler.PrizeDistributionHandler
	statsHandler             *handler.StatsHandler
	telegramHandler          *handler.TelegramHandler
	miniappHandler           *handler.MiniappHandler
	userBlacklistHandler     *handler.UserBlacklistHandler

	// JWT Manager
	jwtManager         *jwt.Manager
	telegramWebAppAuth func(http.Handler) http.Handler
}

// Config представляет конфигурацию сервера
type Config struct {
	Host            string
	Port            int
	AllowedOrigins  []string
	JWTSecret       string
	JWTAccessTTL    time.Duration
	JWTRefreshTTL   time.Duration
	BotToken        string // Токен Telegram бота для получения файлов
	FileStoragePath string
}

// NewServer создаёт новый HTTP сервер
func NewServer(
	cfg Config,
	userRepo repository.UserRepository,
	eventRepo repository.EventRepository,
	participantRepo repository.ParticipantRepository,
	resultRepo repository.ResultRepository,
	giftRepo repository.GiftRepository,
	criteriaRepo repository.CriteriaRepository,
	prizeAssignmentRepo repository.PrizeAssignmentRepository,
	userBlacklistRepo repository.UserBlacklistRepository,
	adminRepo repository.AdminRepository,
) *Server {
	// Создаём command handlers
	registerParticipantHandler := command.NewRegisterParticipantHandler(
		userRepo,
		eventRepo,
		participantRepo,
		userBlacklistRepo,
	)

	submitResultHandler := command.NewSubmitResultHandler(
		participantRepo,
		eventRepo,
		resultRepo,
	)

	assignPrizeHandler := command.NewAssignPrizeHandler(
		participantRepo,
		giftRepo,
		prizeAssignmentRepo,
	)

	// Создаём query handlers
	getParticipantsHandler := query.NewGetParticipantsHandler(participantRepo)
	getParticipantByIDHandler := query.NewGetParticipantByIDHandler(participantRepo)
	getGiftsHandler := query.NewGetGiftsHandler(giftRepo, criteriaRepo)
	getGiftByIDHandler := query.NewGetGiftByIDHandler(giftRepo, criteriaRepo)
	getMiniappGiftsHandler := query.NewGetMiniappGiftsHandler(giftRepo, criteriaRepo)
	getEventsHandler := query.NewGetEventsHandler(eventRepo)
	getEventByIDHandler := query.NewGetEventByIDHandler(eventRepo)
	getPrizeAssignmentsHandler := query.NewGetPrizeAssignmentsHandler(prizeAssignmentRepo)
	getPrizeAssignmentByIDHandler := query.NewGetPrizeAssignmentByIDHandler(prizeAssignmentRepo)
	listUserBlacklistHandler := query.NewListUserBlacklistHandler(userBlacklistRepo)
	isUserBlacklistedHandler := query.NewIsUserBlacklistedHandler(userBlacklistRepo)
	getStatsHandler := query.NewGetStatsHandler(
		eventRepo,
		participantRepo,
		giftRepo,
		resultRepo,
		criteriaRepo,
	)

	// Создаём command handlers для events
	createEventHandler := command.NewCreateEventHandler(eventRepo)
	updateEventHandler := command.NewUpdateEventHandler(eventRepo)

	// Создаём command handlers для participants
	updateParticipantHandler := command.NewUpdateParticipantHandler(participantRepo)
	deleteParticipantHandler := command.NewDeleteParticipantHandler(participantRepo)

	// Создаём command handlers для criteria
	createCriteriaHandler := command.NewCreateCriteriaHandler(criteriaRepo)
	updateCriteriaHandler := command.NewUpdateCriteriaHandler(criteriaRepo)

	// Создаём command handlers для gifts
	updateGiftHandler := command.NewUpdateGiftHandler(giftRepo)

	// Создаём command handlers для blacklist пользователей
	addUserBlacklistHandler := command.NewAddUserBlacklistHandler(userBlacklistRepo)
	updateUserBlacklistReasonHandler := command.NewUpdateUserBlacklistReasonHandler(userBlacklistRepo)
	removeUserBlacklistHandler := command.NewRemoveUserBlacklistHandler(userBlacklistRepo)

	// Создаём query handlers для criteria
	getCriteriaHandler := query.NewGetCriteriaHandler(criteriaRepo)
	getCriteriaByIDHandler := query.NewGetCriteriaByIDHandler(criteriaRepo)

	// Создаём JWT Manager
	jwtManager := jwt.NewManager(
		cfg.JWTSecret,
		cfg.JWTAccessTTL,
		cfg.JWTRefreshTTL,
	)

	// Создаём HTTP handlers
	authHandler := handler.NewAuthHandler(adminRepo, jwtManager)
	eventFileStorage := storage.NewLocalFileStorage(cfg.FileStoragePath)
	eventsHandler := handler.NewEventsHandler(
		eventRepo,
		getEventsHandler,
		getEventByIDHandler,
		createEventHandler,
		updateEventHandler,
		eventFileStorage,
	)
	// Создаём query handlers для распределения призов (нужны для participantsHandler)
	getPrizeDistributionHandlerTemp := query.NewGetPrizeDistributionHandler(
		resultRepo,
		giftRepo,
		participantRepo,
		criteriaRepo,
	)

	participantsHandler := handler.NewParticipantsHandler(
		participantRepo,
		resultRepo,
		giftRepo,
		criteriaRepo,
		prizeAssignmentRepo,
		getParticipantsHandler,
		getParticipantByIDHandler,
		getPrizeDistributionHandlerTemp,
		registerParticipantHandler,
		updateParticipantHandler,
		deleteParticipantHandler,
	)
	giftsHandler := handler.NewGiftsHandler(
		giftRepo,
		getGiftsHandler,
		getGiftByIDHandler,
		updateGiftHandler,
	)
	criteriaHandler := handler.NewCriteriaHandler(
		criteriaRepo,
		getCriteriaHandler,
		getCriteriaByIDHandler,
		createCriteriaHandler,
		updateCriteriaHandler,
	)
	prizeAssignmentsHandler := handler.NewPrizeAssignmentsHandler(
		prizeAssignmentRepo,
		getPrizeAssignmentsHandler,
		getPrizeAssignmentByIDHandler,
		assignPrizeHandler,
	)
	resultsHandler := handler.NewResultsHandler(resultRepo, participantRepo, criteriaRepo, submitResultHandler)
	statsHandler := handler.NewStatsHandler(getStatsHandler)
	telegramHandler := handler.NewTelegramHandler(cfg.BotToken)
	miniappHandler := handler.NewMiniappHandler(
		eventRepo,
		getMiniappGiftsHandler,
		cfg.BotToken,
	)
	userBlacklistHandler := handler.NewUserBlacklistHandler(
		listUserBlacklistHandler,
		addUserBlacklistHandler,
		updateUserBlacklistReasonHandler,
		removeUserBlacklistHandler,
	)

	// Создаём query handlers для распределения призов
	getPrizeDistributionHandler := query.NewGetPrizeDistributionHandler(
		resultRepo,
		giftRepo,
		participantRepo,
		criteriaRepo,
	)
	getResultsWithPlacesHandler := query.NewGetResultsWithPlacesHandler(resultRepo)

	prizeDistributionHandler := handler.NewPrizeDistributionHandler(
		getPrizeDistributionHandler,
		getResultsWithPlacesHandler,
	)

	s := &Server{
		userRepo:                         userRepo,
		eventRepo:                        eventRepo,
		participantRepo:                  participantRepo,
		resultRepo:                       resultRepo,
		giftRepo:                         giftRepo,
		criteriaRepo:                     criteriaRepo,
		prizeAssignmentRepo:              prizeAssignmentRepo,
		userBlacklistRepo:                userBlacklistRepo,
		adminRepo:                        adminRepo,
		registerParticipantHandler:       registerParticipantHandler,
		updateGiftHandler:                updateGiftHandler,
		submitResultHandler:              submitResultHandler,
		assignPrizeHandler:               assignPrizeHandler,
		addUserBlacklistHandler:          addUserBlacklistHandler,
		updateUserBlacklistReasonHandler: updateUserBlacklistReasonHandler,
		removeUserBlacklistHandler:       removeUserBlacklistHandler,
		deleteParticipantHandler:         deleteParticipantHandler,
		getParticipantsHandler:           getParticipantsHandler,
		getParticipantByIDHandler:        getParticipantByIDHandler,
		getGiftsHandler:                  getGiftsHandler,
		getGiftByIDHandler:               getGiftByIDHandler,
		getEventsHandler:                 getEventsHandler,
		getEventByIDHandler:              getEventByIDHandler,
		getPrizeAssignmentsHandler:       getPrizeAssignmentsHandler,
		getPrizeAssignmentByIDHandler:    getPrizeAssignmentByIDHandler,
		getStatsHandler:                  getStatsHandler,
		listUserBlacklistHandler:         listUserBlacklistHandler,
		isUserBlacklistedHandler:         isUserBlacklistedHandler,
		authHandler:                      authHandler,
		eventsHandler:                    eventsHandler,
		participantsHandler:              participantsHandler,
		resultsHandler:                   resultsHandler,
		giftsHandler:                     giftsHandler,
		criteriaHandler:                  criteriaHandler,
		prizeAssignmentsHandler:          prizeAssignmentsHandler,
		prizeDistributionHandler:         prizeDistributionHandler,
		statsHandler:                     statsHandler,
		telegramHandler:                  telegramHandler,
		miniappHandler:                   miniappHandler,
		userBlacklistHandler:             userBlacklistHandler,
		jwtManager:                       jwtManager,
		telegramWebAppAuth:               middleware.TelegramWebAppAuth(cfg.BotToken),
	}

	// Создаём router
	s.router = s.setupRouter(cfg)

	// Создаём HTTP сервер
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// setupRouter настраивает маршруты
func (s *Server) setupRouter(cfg Config) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)

	if len(cfg.AllowedOrigins) > 0 {
		r.Use(middleware.CORSWithOrigins(cfg.AllowedOrigins))
	} else {
		r.Use(middleware.CORS)
	}

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Swagger documentation
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusMovedPermanently)
	})
	r.Handle("/docs/*", http.StripPrefix("/docs", http.FileServer(http.Dir("./docs"))))

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Auth routes (public)
		r.Post("/auth/login", s.authHandler.Login)
		r.Post("/auth/refresh", s.authHandler.Refresh)

		// Events routes (public read, protected write)
		r.Get("/events", s.eventsHandler.GetAll)
		r.Get("/events/{id}", s.eventsHandler.GetByID)

		// Participants routes (public read, protected write)
		r.Get("/events/{eventId}/participants", s.participantsHandler.GetAll)
		r.Get("/participants/{id}", s.participantsHandler.GetByID)
		r.Get("/participants/{id}/gifts", s.participantsHandler.GetGifts)
		// r.Get("/participants/{id}/prizes", s.participantsHandler.GetPrizes) // deprecated - use prize-distribution

		// Results routes (public read, protected write)
		r.Get("/participants/{participantId}/results", s.resultsHandler.GetByParticipant)
		r.Get("/results/{id}", s.resultsHandler.GetByID)

		// Prize Distribution routes (public read)
		r.Get("/events/{id}/prize-distribution", s.prizeDistributionHandler.GetPrizeDistribution)
		r.Get("/events/{id}/results", s.prizeDistributionHandler.GetResultsWithPlaces)

		// Gifts routes (public read, protected write)
		r.Get("/events/{eventId}/gifts", s.giftsHandler.GetAll)
		r.Get("/gifts/{id}", s.giftsHandler.GetByID)

		// Criteria routes (public read, protected write)
		r.Get("/criteria", s.criteriaHandler.GetAll)
		r.Get("/criteria/{id}", s.criteriaHandler.GetByID)

		// Prize Assignments routes (deprecated - use prize-distribution instead)
		// r.Get("/events/{eventId}/prize-assignments", s.prizeAssignmentsHandler.GetAll)
		// r.Get("/participants/{participantId}/prize-assignments", s.prizeAssignmentsHandler.GetAll)
		// r.Get("/prize-assignments/{id}", s.prizeAssignmentsHandler.GetByID)

		// Stats routes (public read)
		r.Get("/stats", s.statsHandler.GetAll)
		r.Get("/events/{eventId}/stats", s.statsHandler.GetByEventID)

		// Telegram file routes (public read)
		r.Get("/telegram/files/{fileId}", s.telegramHandler.GetFileURL)
		r.Get("/telegram/files/{fileId}/info", s.telegramHandler.GetFileInfo)

		// Telegram Mini App routes (protected by Telegram init data)
		r.Route("/miniapp", func(r chi.Router) {
			r.Use(s.telegramWebAppAuth)
			r.Get("/session", s.miniappHandler.Session)
			r.Get("/gifts", s.miniappHandler.Gifts)
			r.Get("/telegram/files/{fileId}", s.miniappHandler.TelegramFile)
		})

		// Protected routes (require authentication and admin role)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(s.jwtManager))
			r.Use(middleware.RequireRole(s.jwtManager, "admin"))

			// Events admin routes
			r.Post("/events", s.eventsHandler.Create)
			r.Put("/events/{id}", s.eventsHandler.Update)
			r.Post("/events/{id}/gpx-file", s.eventsHandler.UploadGPXFile)
			r.Delete("/events/{id}", s.eventsHandler.Delete)

			// Participants admin routes
			r.Post("/events/{eventId}/participants", s.participantsHandler.Create)
			r.Put("/participants/{id}", s.participantsHandler.Update)
			r.Delete("/participants/{id}", s.participantsHandler.Delete)

			// Results admin routes
			r.Post("/participants/{participantId}/results", s.resultsHandler.Create)
			r.Put("/results/{id}", s.resultsHandler.Update)
			r.Delete("/results/{id}", s.resultsHandler.Delete)
			r.Post("/results/{id}/criteria", s.resultsHandler.AddCriteria)
			r.Delete("/results/{id}/criteria/{criteriaId}", s.resultsHandler.RemoveCriteria)

			// Gifts admin routes
			r.Put("/gifts/{id}", s.giftsHandler.Update)
			r.Delete("/gifts/{id}", s.giftsHandler.Delete)

			// User blacklist admin routes
			r.Get("/user-blacklist", s.userBlacklistHandler.GetAll)
			r.Post("/user-blacklist", s.userBlacklistHandler.Create)
			r.Put("/user-blacklist/{telegramUserId}", s.userBlacklistHandler.Update)
			r.Delete("/user-blacklist/{telegramUserId}", s.userBlacklistHandler.Delete)

			// Criteria admin routes
			r.Post("/criteria", s.criteriaHandler.Create)
			r.Put("/criteria/{id}", s.criteriaHandler.Update)
			r.Delete("/criteria/{id}", s.criteriaHandler.Delete)

			// Prize Assignments admin routes (deprecated - use prize-distribution instead)
			// r.Post("/prize-assignments", s.prizeAssignmentsHandler.Create)
			// r.Delete("/prize-assignments/{id}", s.prizeAssignmentsHandler.Delete)

			// User info
			r.Get("/auth/me", s.authHandler.Me)
		})
	})

	return r
}

// Start запускает сервер
func (s *Server) Start() error {
	log.Printf("Starting HTTP server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully останавливает сервер
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down HTTP server...")
	return s.httpServer.Shutdown(ctx)
}

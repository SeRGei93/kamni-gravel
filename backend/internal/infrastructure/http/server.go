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
	"gravel_bot/internal/infrastructure/cache"
	"gravel_bot/internal/infrastructure/http/handler"
	"gravel_bot/internal/infrastructure/http/middleware"
	"gravel_bot/internal/infrastructure/lock"
	"gravel_bot/internal/infrastructure/security"
	"gravel_bot/internal/infrastructure/storage"
	telegraminfra "gravel_bot/internal/infrastructure/telegram"
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
	createAdminHandler               *command.CreateAdminHandler
	changeAdminPasswordHandler       *command.ChangeAdminPasswordHandler

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
	getDailyStatsHandler          *query.GetDailyStatsHandler
	listUserBlacklistHandler      *query.ListUserBlacklistHandler
	isUserBlacklistedHandler      *query.IsUserBlacklistedHandler
	listAdminUsersHandler         *query.ListAdminUsersHandler

	// HTTP handlers
	authHandler                     *handler.AuthHandler
	eventsHandler                   *handler.EventsHandler
	participantsHandler             *handler.ParticipantsHandler
	resultsHandler                  *handler.ResultsHandler
	giftsHandler                    *handler.GiftsHandler
	criteriaHandler                 *handler.CriteriaHandler
	prizeAssignmentsHandler         *handler.PrizeAssignmentsHandler
	prizeDistributionHandler        *handler.PrizeDistributionHandler
	statsHandler                    *handler.StatsHandler
	telegramHandler                 *handler.TelegramHandler
	miniappHandler                  *handler.MiniappHandler
	userBlacklistHandler            *handler.UserBlacklistHandler
	adminUsersHandler               *handler.AdminUsersHandler
	participantLockHandler          *handler.ParticipantLockHandler
	chatMembersHandler              *handler.ChatMembersHandler
	participantNotificationsHandler *handler.ParticipantNotificationsHandler
	participantNotificationJobs     *command.ParticipantNotificationJobManager
	participantNotificationJobsCtx  context.Context
	participantNotificationJobsStop context.CancelFunc

	// Lock manager (in-memory participant edit locks)
	lockManager *lock.Manager

	// JWT Manager
	jwtManager         *jwt.Manager
	telegramWebAppAuth func(http.Handler) http.Handler
}

// Config представляет конфигурацию сервера
type Config struct {
	Host                       string
	Port                       int
	AllowedOrigins             []string
	JWTSecret                  string
	JWTAccessTTL               time.Duration
	JWTRefreshTTL              time.Duration
	BotToken                   string // Токен Telegram бота для получения файлов
	PublicChatID               int64
	MiniappURL                 string
	FileStoragePath            string
	MiniappCacheDir            string        // Каталог файлового кеша каталога подарков мини-приложения
	MiniappGiftsCacheTTL       time.Duration // Страховочный TTL записей кеша подарков мини-приложения
	MiniappLocalTelegramUserID int64         // Локальный Telegram-пользователь для браузерной Mini App проверки
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
	chatMemberRepo repository.ChatMemberRepository,
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
	createManualResultHandler := command.NewCreateManualResultHandler(
		participantRepo,
		resultRepo,
	)
	updateResultHandler := command.NewUpdateResultHandler(resultRepo)

	assignPrizeHandler := command.NewAssignPrizeHandler(
		participantRepo,
		giftRepo,
		prizeAssignmentRepo,
	)

	// Создаём query handlers
	getParticipantsHandler := query.NewGetParticipantsHandler(participantRepo)
	getParticipantByIDHandler := query.NewGetParticipantByIDHandler(participantRepo)
	getParticipantByUserHandler := query.NewGetParticipantByUserAndEventHandler(participantRepo)
	getGiftsHandler := query.NewGetGiftsHandler(giftRepo, criteriaRepo)
	getGiftByIDHandler := query.NewGetGiftByIDHandler(giftRepo, criteriaRepo)
	getManualGiftsHandler := query.NewGetManualGiftsHandler(giftRepo)
	getMiniappGiftsHandler := query.NewGetMiniappGiftsHandler(giftRepo, criteriaRepo)
	getMiniappParticipantCountHandler := query.NewGetMiniappParticipantCountHandler(participantRepo)

	// Файловый кеш каталога подарков мини-приложения (первый экран). Один общий инстанс
	// используют и чтение (MiniappHandler), и инвалидация при одобрении (GiftsHandler).
	// При ошибке создания каталога кеш остаётся nil — кеширование просто отключается.
	var miniappGiftsCache *cache.MiniappGiftsCache
	if c, err := cache.NewMiniappGiftsCache(cfg.MiniappCacheDir, cfg.MiniappGiftsCacheTTL); err != nil {
		log.Printf("WARN Miniapp gifts cache disabled: dir=%q error=%v", cfg.MiniappCacheDir, err)
	} else {
		miniappGiftsCache = c
		log.Printf("INFO Miniapp gifts cache enabled: dir=%q ttl=%s", cfg.MiniappCacheDir, cfg.MiniappGiftsCacheTTL)
	}
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
	getDailyStatsHandler := query.NewGetDailyStatsHandler(eventRepo, participantRepo)

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
	addGiftHandler := command.NewAddGiftHandler(
		userRepo,
		eventRepo,
		giftRepo,
		userBlacklistRepo,
	)
	updateGiftHandler := command.NewUpdateGiftHandler(giftRepo, participantRepo)
	var copyGiftHandler *command.CopyGiftHandler
	if copyRepo, ok := giftRepo.(repository.GiftCopyRepository); ok {
		copyGiftHandler = command.NewCopyGiftHandler(copyRepo)
	} else {
		log.Printf("WARN Gift copy handler disabled: repository does not support copying")
	}

	// Создаём command handlers для blacklist пользователей
	addUserBlacklistHandler := command.NewAddUserBlacklistHandler(userBlacklistRepo)
	updateUserBlacklistReasonHandler := command.NewUpdateUserBlacklistReasonHandler(userBlacklistRepo)
	removeUserBlacklistHandler := command.NewRemoveUserBlacklistHandler(userBlacklistRepo)

	// Создаём command/query handlers для администраторов
	passwordHasher := security.NewBcryptPasswordHasher()
	listAdminUsersHandler := query.NewListAdminUsersHandler(adminRepo)
	createAdminHandler := command.NewCreateAdminHandler(adminRepo, passwordHasher)
	changeAdminPasswordHandler := command.NewChangeAdminPasswordHandler(adminRepo, passwordHasher)

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
	manualGiftRepo, supportsManualGifts := giftRepo.(repository.ManualGiftRepository)
	var eligibleUnawardedParticipantIDsHandler *query.GetEligibleUnawardedParticipantIDsHandler
	var assignRandomAdminGiftRecipientHandler *command.AssignRandomAdminGiftRecipientHandler
	if supportsManualGifts {
		eligibleUnawardedParticipantIDsHandler = query.NewGetEligibleUnawardedParticipantIDsHandler(
			participantRepo,
			manualGiftRepo,
			getPrizeDistributionHandlerTemp,
		)
		if randomRecipientWriter, ok := giftRepo.(repository.RandomManualGiftRecipientRepository); ok {
			assignRandomAdminGiftRecipientHandler = command.NewAssignRandomAdminGiftRecipientHandler(
				giftRepo,
				randomRecipientWriter,
				participantRepo,
				eligibleUnawardedParticipantIDsHandler,
				getPrizeDistributionHandlerTemp,
			)
		} else {
			log.Printf("ERROR Admin random gift assignment unavailable: gift repository does not implement RandomManualGiftRecipientRepository")
		}
	} else {
		log.Printf("ERROR Gift recipient management unavailable: gift repository does not implement ManualGiftRepository")
	}

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
	var publicGiftNotifier *telegraminfra.GiftNotifier
	if cfg.PublicChatID != 0 {
		notifier, err := telegraminfra.NewGiftNotifierFromToken(cfg.BotToken, telegraminfra.GiftNotifierConfig{
			ChatID:     cfg.PublicChatID,
			ChatName:   "public",
			MiniappURL: cfg.MiniappURL,
		})
		if err != nil {
			log.Printf("WARN Public gift notifier disabled: chat=public error=%v", err)
		} else {
			publicGiftNotifier = notifier
		}
	}

	var giftPublicationNotifiers []handler.GiftPublicationNotifier
	if publicGiftNotifier != nil {
		giftPublicationNotifiers = append(giftPublicationNotifiers, publicGiftNotifier)
	}
	giftsHandler := handler.NewGiftsHandler(
		giftRepo,
		getGiftsHandler,
		getGiftByIDHandler,
		getManualGiftsHandler,
		addGiftHandler,
		updateGiftHandler,
		copyGiftHandler,
		assignRandomAdminGiftRecipientHandler,
		miniappGiftsCache,
		giftPublicationNotifiers...,
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
	resultsHandler := handler.NewResultsHandler(resultRepo, participantRepo, criteriaRepo, submitResultHandler, createManualResultHandler, updateResultHandler)
	statsHandler := handler.NewStatsHandler(getStatsHandler, getDailyStatsHandler)
	telegramHandler := handler.NewTelegramHandler(cfg.BotToken)
	miniappHandler := handler.NewMiniappHandler(
		eventRepo,
		getMiniappGiftsHandler,
		getMiniappParticipantCountHandler,
		getParticipantsHandler,
		getParticipantByUserHandler,
		resultRepo,
		cfg.BotToken,
		miniappGiftsCache,
	)
	if supportsManualGifts {
		participantOptionsHandler := query.NewGetMiniappParticipantsHandler(participantRepo, manualGiftRepo, getPrizeDistributionHandlerTemp)
		setManualGiftRecipientHandler := command.NewSetManualGiftRecipientHandler(manualGiftRepo, participantRepo)
		miniappHandler.ConfigureManualGiftManagement(
			query.NewGetOwnerManualGiftsHandler(manualGiftRepo, criteriaRepo, participantRepo, getPrizeDistributionHandlerTemp),
			query.NewHasOwnerGiftsHandler(manualGiftRepo),
			participantOptionsHandler,
			setManualGiftRecipientHandler,
			command.NewAssignRandomManualGiftRecipientHandler(eligibleUnawardedParticipantIDsHandler, setManualGiftRecipientHandler),
		)
	} else {
		log.Printf("ERROR Miniapp manual gift management unavailable: gift repository does not implement ManualGiftRepository")
	}
	userBlacklistHandler := handler.NewUserBlacklistHandler(
		listUserBlacklistHandler,
		addUserBlacklistHandler,
		updateUserBlacklistReasonHandler,
		removeUserBlacklistHandler,
	)
	adminUsersHandler := handler.NewAdminUsersHandler(
		listAdminUsersHandler,
		createAdminHandler,
		changeAdminPasswordHandler,
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

	// In-memory pessimistic edit lock for participant records (admin panel).
	// The cleanup goroutine starts inside NewManager.
	lockManager := lock.NewManager(lock.DefaultTTL)
	participantLockHandler := handler.NewParticipantLockHandler(lockManager)

	// Чистка публичного чата: kicker включается только при заданных токене и
	// публичном чате, иначе execute вернёт ErrChatPurgeNotConfigured.
	var chatPurgeKicker command.ChatMemberKicker
	kicker, err := telegraminfra.NewChatMemberKickerFromToken(cfg.BotToken, cfg.PublicChatID)
	if err != nil {
		log.Printf("WARN Chat member kicker disabled: error=%v", err)
	} else if kicker != nil {
		chatPurgeKicker = kicker
	}
	chatMembersHandler := handler.NewChatMembersHandler(
		chatMemberRepo,
		eventRepo,
		query.NewGetChatPurgeCandidatesHandler(chatMemberRepo, giftRepo, participantRepo),
		command.NewExecuteChatPurgeHandler(chatMemberRepo, giftRepo, chatPurgeKicker),
	)

	var participantNotifier command.ParticipantNotifier
	notifier, err := telegraminfra.NewParticipantNotifierFromToken(cfg.BotToken)
	if err != nil {
		log.Printf("WARN Participant notifications disabled: error=%v", err)
	} else if notifier != nil {
		participantNotifier = notifier
	}
	participantNotificationJobsCtx, participantNotificationJobsStop := context.WithCancel(context.Background())
	participantNotificationJobs := command.NewParticipantNotificationJobManager(
		command.NewSendParticipantNotificationsHandler(participantRepo, participantNotifier),
	)
	participantNotificationsHandler := handler.NewParticipantNotificationsHandler(
		eventRepo,
		query.NewGetNotificationRecipientsHandler(participantRepo, giftRepo),
		participantNotificationJobs,
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
		createAdminHandler:               createAdminHandler,
		changeAdminPasswordHandler:       changeAdminPasswordHandler,
		getParticipantsHandler:           getParticipantsHandler,
		getParticipantByIDHandler:        getParticipantByIDHandler,
		getGiftsHandler:                  getGiftsHandler,
		getGiftByIDHandler:               getGiftByIDHandler,
		getEventsHandler:                 getEventsHandler,
		getEventByIDHandler:              getEventByIDHandler,
		getPrizeAssignmentsHandler:       getPrizeAssignmentsHandler,
		getPrizeAssignmentByIDHandler:    getPrizeAssignmentByIDHandler,
		getStatsHandler:                  getStatsHandler,
		getDailyStatsHandler:             getDailyStatsHandler,
		listUserBlacklistHandler:         listUserBlacklistHandler,
		isUserBlacklistedHandler:         isUserBlacklistedHandler,
		listAdminUsersHandler:            listAdminUsersHandler,
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
		adminUsersHandler:                adminUsersHandler,
		participantLockHandler:           participantLockHandler,
		chatMembersHandler:               chatMembersHandler,
		participantNotificationsHandler:  participantNotificationsHandler,
		participantNotificationJobs:      participantNotificationJobs,
		participantNotificationJobsCtx:   participantNotificationJobsCtx,
		participantNotificationJobsStop:  participantNotificationJobsStop,
		lockManager:                      lockManager,
		jwtManager:                       jwtManager,
		telegramWebAppAuth: middleware.TelegramWebAppAuthWithConfig(middleware.TelegramWebAppAuthConfig{
			BotToken:               cfg.BotToken,
			LocalDevTelegramUserID: cfg.MiniappLocalTelegramUserID,
		}),
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
		r.Get("/events/{eventId}/stats/daily", s.statsHandler.GetDailyByEventID)

		// Telegram file routes (public read)
		r.Get("/telegram/files/{fileId}", s.telegramHandler.GetFileURL)
		r.Get("/telegram/files/{fileId}/info", s.telegramHandler.GetFileInfo)

		// Telegram Mini App routes (protected by Telegram init data)
		r.Route("/miniapp", func(r chi.Router) {
			r.Use(s.telegramWebAppAuth)
			r.Get("/session", s.miniappHandler.Session)
			r.Get("/gifts", s.miniappHandler.Gifts)
			r.Get("/my-gifts", s.miniappHandler.MyGifts)
			r.Get("/participants", s.miniappHandler.Participants)
			r.Put("/my-gifts/{giftId}/recipient", s.miniappHandler.UpdateMyGiftRecipient)
			r.Post("/my-gifts/{giftId}/random-recipient", s.miniappHandler.AssignRandomMyGiftRecipient)
			r.Get("/leaderboard", s.miniappHandler.Leaderboard)
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

			// Participant edit-lock guards (in-memory pessimistic lock). Writes
			// reachable from the participant detail page are rejected with 409 when
			// the participant is locked by another admin.
			participantLockGuard := middleware.RequireParticipantUnlocked(s.lockManager, middleware.ParticipantIDFromParam("id"))
			newResultLockGuard := middleware.RequireParticipantUnlocked(s.lockManager, middleware.ParticipantIDFromParam("participantId"))
			resultLockGuard := middleware.RequireParticipantUnlocked(s.lockManager, middleware.ParticipantIDFromResult(s.resultRepo))

			// Participants admin routes
			r.Post("/events/{eventId}/participants", s.participantsHandler.Create)
			r.With(participantLockGuard).Put("/participants/{id}", s.participantsHandler.Update)
			r.With(participantLockGuard).Delete("/participants/{id}", s.participantsHandler.Delete)

			// Participant edit-lock routes (acquire / release / status)
			r.Post("/participants/{id}/lock", s.participantLockHandler.Acquire)
			r.Delete("/participants/{id}/lock", s.participantLockHandler.Release)
			r.Get("/participants/{id}/lock", s.participantLockHandler.Status)

			// Results admin routes (guarded by the owning participant's edit-lock)
			r.With(newResultLockGuard).Post("/participants/{participantId}/results", s.resultsHandler.Create)
			r.With(resultLockGuard).Put("/results/{id}", s.resultsHandler.Update)
			r.With(resultLockGuard).Delete("/results/{id}", s.resultsHandler.Delete)
			r.With(resultLockGuard).Post("/results/{id}/criteria", s.resultsHandler.AddCriteria)
			r.With(resultLockGuard).Delete("/results/{id}/criteria/{criteriaId}", s.resultsHandler.RemoveCriteria)

			// Gifts admin routes
			r.Post("/events/{eventId}/gifts", s.giftsHandler.Create)
			r.Get("/events/{eventId}/manual-gifts", s.giftsHandler.GetManualByEvent)
			r.Post("/gifts/{id}/random-recipient", s.giftsHandler.AssignRandomRecipient)
			r.Post("/gifts/{id}/copies", s.giftsHandler.Copy)
			r.Put("/gifts/{id}", s.giftsHandler.Update)
			r.Delete("/gifts/{id}", s.giftsHandler.Delete)

			// Chat members / purge admin routes
			r.Post("/chat-members/import", s.chatMembersHandler.Import)
			r.Get("/chat-purge/candidates", s.chatMembersHandler.Candidates)
			r.Post("/chat-purge/execute", s.chatMembersHandler.Execute)

			// Participant notification admin routes
			r.Get("/participant-notifications/recipients", s.participantNotificationsHandler.Recipients)
			r.Post("/participant-notifications/send", s.participantNotificationsHandler.Send)
			r.Get("/participant-notifications/jobs/{id}", s.participantNotificationsHandler.Status)

			// User blacklist admin routes
			r.Get("/user-blacklist", s.userBlacklistHandler.GetAll)
			r.Post("/user-blacklist", s.userBlacklistHandler.Create)
			r.Put("/user-blacklist/{telegramUserId}", s.userBlacklistHandler.Update)
			r.Delete("/user-blacklist/{telegramUserId}", s.userBlacklistHandler.Delete)

			// Admin users routes
			r.Get("/admin-users", s.adminUsersHandler.GetAll)
			r.Post("/admin-users", s.adminUsersHandler.Create)
			r.Put("/auth/me/password", s.adminUsersHandler.ChangeOwnPassword)

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
	if s.participantNotificationJobs != nil && s.participantNotificationJobsCtx != nil {
		go s.participantNotificationJobs.Run(s.participantNotificationJobsCtx)
	}
	log.Printf("Starting HTTP server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully останавливает сервер
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down HTTP server...")
	if s.participantNotificationJobsStop != nil {
		s.participantNotificationJobsStop()
	}
	return s.httpServer.Shutdown(ctx)
}

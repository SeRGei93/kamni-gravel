# Gravel Bot

Telegram бот для организации велогонок с DDD архитектурой и админ-панелью.

## Структура проекта

```
gravel_bot/
├── backend/          # Go backend (DDD)
├── frontend/         # Next.js админ-панель
├── docker-compose.yml
├── .env.template
└── Makefile
```

## Быстрый старт

### Запуск с Docker (рекомендуется)

1. Скопируйте `env.example` в `.env` и заполните переменные:
```bash
cp env.example .env
# Отредактируйте .env файл, установите BOT_TOKEN и JWT_SECRET
```

2. Запустите все сервисы:
```bash
docker-compose up -d
```

Это запустит:
- PostgreSQL базу данных
- Миграции (автоматически)
- Telegram бота
- REST API сервер
- Frontend (Next.js)

### Локальный запуск

См. [README_LOCAL.md](README_LOCAL.md) для подробных инструкций по локальному запуску.

## Переменные окружения

Создайте файл `.env` на основе `env.example`:

```bash
# Telegram Bot
BOT_TOKEN=your_telegram_bot_token_here
MINIAPP_URL=https://example.com/miniapp/gifts
JWT_SECRET=your_jwt_secret_key_here

# PostgreSQL
POSTGRES_DB=gravel_bot
POSTGRES_USER=gravel
POSTGRES_PASSWORD=gravel_password
POSTGRES_PORT=5432

# Database Connection
DB_HOST=postgres
DB_PORT=5432
DB_NAME=gravel_bot
DB_USER=gravel
DB_PASSWORD=gravel_password
DB_SSLMODE=disable

# API
API_HOST=0.0.0.0
API_PORT=8080
NEXT_PUBLIC_API_URL=https://api.example.com
ALLOWED_ORIGINS=https://example.com
```

### Telegram Mini App

`MINIAPP_URL` включает кнопку "Смотреть подарки" в Telegram-боте. Для реального Telegram Mini App URL должен быть публичным HTTPS-адресом frontend-маршрута `/miniapp/gifts`.

В non-local окружениях `NEXT_PUBLIC_API_URL` тоже должен быть публичным HTTPS API URL, доступным из Telegram-клиента пользователя. Если `ALLOWED_ORIGINS` не равен `*`, добавьте origin miniapp frontend. Miniapp-запросы отправляют заголовок `X-Telegram-Init-Data` со значением из `Telegram.WebApp.initData`; backend валидирует этот заголовок перед доступом к `/api/miniapp/*`.

## Доступные команды

```bash
make help           # Показать все команды
make build          # Собрать бинарники
make run-bot        # Запустить бота (локально)
make run-api        # Запустить API (локально)
make migrate-up     # Применить миграции (локально)
make migrate-down   # Откатить миграции (локально)
make test           # Запустить тесты
make docker-up      # Запустить в Docker
make docker-down    # Остановить Docker контейнеры
make docker-logs    # Показать логи Docker
```

## 📚 API Документация

После запуска API сервера, Swagger документация доступна по адресу:

**http://localhost:8080/docs/**

Или локально:
```bash
cd backend/docs
python3 -m http.server 8000
# Откройте http://localhost:8000
```

## Технологии

### Backend
- Go 1.23+
- PostgreSQL + goose миграции
- Chi router
- JWT авторизация
- Telegram Bot API

### Frontend
- Next.js 16
- TailwindCSS
- TypeScript

### Infrastructure
- Docker & Docker Compose
- PostgreSQL 16

## Архитектура

Проект следует принципам Domain-Driven Design (DDD):

- **Domain Layer**: Entities, Value Objects, Repository interfaces
- **Application Layer**: Use Cases (Commands, Queries)
- **Infrastructure Layer**: Repositories, HTTP, Telegram adapters

### Слои приложения

```
backend/
├── cmd/
│   ├── api/         # HTTP API entry point
│   ├── bot/         # Telegram bot entry point
│   └── migrate/     # Migration tool
├── internal/
│   ├── application/      # Application layer (CQRS)
│   │   ├── command/     # Commands (write operations)
│   │   ├── query/       # Queries (read operations)
│   │   └── dto/         # Data Transfer Objects
│   ├── domain/          # Domain layer
│   │   ├── entity/      # Domain entities
│   │   ├── valueobject/ # Value objects
│   │   └── repository/  # Repository interfaces
│   ├── infrastructure/  # Infrastructure layer
│   │   ├── http/        # HTTP handlers
│   │   ├── persistence/ # Database implementations
│   │   │   └── postgres/  # PostgreSQL repositories
│   │   ├── telegram/    # Telegram bot handlers
│   │   └── migrations/  # SQL migrations
│   └── config/          # Configuration
```

## Миграции

Миграции автоматически применяются при запуске через Docker Compose.

Для ручного управления миграциями:

```bash
# Применить все миграции
docker-compose run --rm migrate up

# Откатить последнюю миграцию
docker-compose run --rm migrate down

# Показать статус миграций
docker-compose run --rm migrate status
```

## Разработка

### Требования

- Go 1.23+
- PostgreSQL 16+
- Node.js 18+ (для frontend)
- Docker & Docker Compose (опционально)

### Установка зависимостей

```bash
# Backend
cd backend
go mod download

# Frontend
cd frontend
npm install
```

### Запуск тестов

```bash
cd backend
go test ./...
```

## Дефолтные учетные данные

После применения миграций создается дефолтный админ:
- **Username**: admin
- **Password**: admin123

⚠️ **Важно**: Смените пароль после первого входа!

## Порты

- **3000**: Frontend (Next.js)
- **8080**: Backend API
- **5432**: PostgreSQL

## Лицензия

MIT

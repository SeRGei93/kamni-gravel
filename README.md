# Gravel Bot

Telegram бот для организации велогонок с DDD архитектурой и админ-панелью.

## Структура проекта

```
gravel_bot/
├── backend/          # Go backend (DDD)
├── frontend/         # Next.js админ-панель
├── docker-compose.yml
├── docker-compose.prod.yml
├── nginx/            # production reverse proxy config
├── scripts/          # operational scripts
├── env.example
└── Makefile
```

## Быстрый старт

### Запуск с Docker (рекомендуется)

1. Скопируйте `env.example` в `.env` и заполните переменные:
```bash
cp env.example .env
# Отредактируйте .env файл, установите BOT_TOKEN и JWT_SECRET
```

2. Запустите все сервисы для локальной разработки:
```bash
docker-compose up -d
```

Это запустит:
- PostgreSQL базу данных
- Миграции (автоматически)
- Telegram бота
- REST API сервер
- Frontend (Next.js)

Локальный compose публикует сервисы на host-порты для удобства разработки. Production nginx подключается только через `docker-compose.prod.yml`.

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
NEXT_PUBLIC_API_URL=https://gravel.example.com
ALLOWED_ORIGINS=https://gravel.example.com

# Production nginx and SSL
PUBLIC_DOMAIN=gravel.example.com
CERTBOT_EMAIL=admin@example.com
```

### Telegram Mini App

`MINIAPP_URL` включает кнопку "Смотреть подарки" в Telegram-боте. Для реального Telegram Mini App URL должен быть публичным HTTPS-адресом frontend-маршрута `/miniapp/gifts`.

В non-local окружениях `NEXT_PUBLIC_API_URL` тоже должен быть публичным HTTPS API URL, доступным из Telegram-клиента пользователя. В production рекомендуется один публичный origin:

```env
PUBLIC_DOMAIN=gravel.example.com
MINIAPP_URL=https://gravel.example.com/miniapp/gifts
NEXT_PUBLIC_API_URL=https://gravel.example.com
ALLOWED_ORIGINS=https://gravel.example.com
```

Miniapp-запросы отправляют заголовок `X-Telegram-Init-Data` со значением из `Telegram.WebApp.initData`; backend валидирует этот заголовок перед доступом к `/api/miniapp/*`.

### Production nginx и SSL

Для production нужен один публичный домен `PUBLIC_DOMAIN`. На нем nginx маршрутизирует:

- `/` и `/miniapp/*` во frontend;
- `/api/*`, `/health`, `/docs/*` в backend API.

Отдельный `api.` поддомен сейчас не нужен: API доступен по path routing на том же домене, поэтому frontend и miniapp работают с тем же origin. Отдельный Telegram webhook домен тоже не нужен, потому что бот работает через polling. PostgreSQL, API, frontend, bot и migrate в production остаются во внутренней Docker-сети; наружу публикуются только `80` и `443` у nginx.

Перед запуском production укажите DNS `A/AAAA` записи `PUBLIC_DOMAIN` на сервер и откройте входящие `80/tcp` и `443/tcp`.

Первичная выдача Let's Encrypt сертификата:

```bash
make ssl-cert
```

Скрипт `scripts/generate-ssl-cert.sh` создает временный self-signed сертификат, поднимает nginx для HTTP-01 challenge, запрашивает реальный сертификат через certbot и перезагружает nginx. Для проверки без боевого лимита Let's Encrypt можно поставить `SSL_STAGING=true`.

Сертификаты сохраняются локально на сервере в `nginx/certbot/conf` и монтируются в nginx/certbot как `/etc/letsencrypt`. Эта директория добавлена в `.gitignore`. Повторный `make ssl-cert` не перевыпускает сертификат, если существующий сертификат еще валиден дольше `SSL_RENEW_BEFORE_SECONDS` секунд, по умолчанию 30 дней.

Запуск production:

```bash
make docker-prod-up
```

Продление сертификата:

```bash
make ssl-renew
```

Для cron достаточно запускать `make ssl-renew` периодически, например раз в день.

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
make docker-prod-up # Запустить production compose с nginx
make ssl-cert       # Выпустить initial SSL сертификат
make ssl-renew      # Продлить SSL сертификат
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

Локальный compose:

- **3000**: Frontend (Next.js)
- **8080/18080**: Backend API, в зависимости от `API_PUBLIC_PORT`
- **5432**: PostgreSQL

Production compose:

- **80**: nginx HTTP и Let's Encrypt HTTP-01 challenge
- **443**: nginx HTTPS
- Backend API, frontend и PostgreSQL доступны только внутри Docker-сети.

## Лицензия

MIT

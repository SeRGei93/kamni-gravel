# Gravel Bot

Telegram бот для организации велогонок с DDD архитектурой и админ-панелью.

## Структура проекта

```
gravel_bot/
├── backend/          # Go backend (DDD)
├── frontend/         # Next.js админ-панель
├── docker-compose.yml      # локальная разработка (свой postgres, порты наружу)
├── docker-compose.prod.yml # production под docker-server (Caddy + общий postgres)
├── scripts/          # operational scripts
├── env.example       # шаблон .env для локалки
├── .env.prod.example # шаблон .env для production (см. DEPLOY.md)
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

Локальный compose публикует сервисы на host-порты для удобства разработки. Production-деплой (Caddy + общий postgres сервера `docker-server`) — отдельный самодостаточный `docker-compose.prod.yml`, см. [DEPLOY.md](DEPLOY.md).

### Локальный запуск

См. [README_LOCAL.md](README_LOCAL.md) для подробных инструкций по локальному запуску.

## Переменные окружения

Создайте файл `.env` на основе `env.example`:

```bash
# Telegram Bot
BOT_TOKEN=your_telegram_bot_token_here
ADMIN_CHAT_ID=0
PUBLIC_CHAT_ID=0
BOT_MESSAGES_CHAT_ID=0
MINIAPP_URL=https://example.com/miniapp/gifts
JWT_SECRET=your_jwt_secret_key_here

# PostgreSQL
POSTGRES_DB=gravel_bot
POSTGRES_USER=gravel
POSTGRES_PASSWORD=gravel_password
POSTGRES_PORT=5432

# Database Connection (name/user/password/port берутся из POSTGRES_* выше)
DB_HOST=postgres
DB_SSLMODE=disable

# API
API_HOST=0.0.0.0
API_PORT=8080
NEXT_PUBLIC_API_URL=https://gravel.example.com
ALLOWED_ORIGINS=https://gravel.example.com

# Miniapp first-screen cache (backend, file-backed)
MINIAPP_CACHE_DIR=/app/cache/miniapp
MINIAPP_GIFTS_CACHE_TTL=1h
```

> Для production используйте отдельный шаблон `.env.prod.example` (общий postgres,
> без своих портов/SSL) — см. [DEPLOY.md](DEPLOY.md).

### Telegram Mini App

`MINIAPP_URL` включает кнопку запуска Mini App в Telegram-боте. Для реального Telegram Mini App URL должен быть публичным HTTPS-адресом frontend-маршрута `/miniapp/leaderboard` (экран по умолчанию). Маршрут `/miniapp` также редиректит на лидерборд.

Mini App разделён табами на три экрана: **Лидерборд** (`/miniapp/leaderboard`, открывается по умолчанию — список участников активного события, отправивших результат, с местами, полным и чистым временем; тап по строке открывает карточку со всеми метриками результата), **Призы** (`/miniapp/gifts`, каталог одобренных подарков) и **Мои призы** (`/miniapp/my-gifts`). В «Моих призах» даритель видит все добавленные им призы активного события, включая ожидающие проверки; для призов, которым администратор включил ручное распределение, он выбирает, меняет или очищает получателя. Получатель может быть любым участником того же события, в том числе без результата, DNF или дисквалифицированным.

Ручной приз не участвует в автоматическом распределении по критериям и местам. Его получатель доступен только в защищённых owner/admin API, а публичный каталог содержит лишь флаг `manual_distribution`. Экран «Автоматическое распределение призов» и его статистика исключают ручные призы. Колонка админ-списка участников «Получит приз» складывает автоматические слоты и сохранённые ручные назначения, включая ручные подарки на проверке.

`BOT_MESSAGES_CHAT_ID` включает служебный Telegram-чат для прокси-переписки с пользователями. Значение `0` отключает этот режим.

В non-local окружениях `NEXT_PUBLIC_API_URL` тоже должен быть публичным HTTPS API URL, доступным из Telegram-клиента пользователя. В production рекомендуется один публичный origin:

```env
PUBLIC_DOMAIN=gravel.example.com
MINIAPP_URL=https://gravel.example.com/miniapp/leaderboard
NEXT_PUBLIC_API_URL=https://gravel.example.com
ALLOWED_ORIGINS=https://gravel.example.com
```

Miniapp-запросы отправляют заголовок `X-Telegram-Init-Data` со значением из `Telegram.WebApp.initData`; backend валидирует этот заголовок перед доступом к `/api/miniapp/*`.

Для локальной проверки Mini App в обычном браузере можно включить имитацию участника: задайте `MINIAPP_LOCAL_MODE=true` и `MINIAPP_LOCAL_TELEGRAM_USER_ID=<telegram_user_id участника>` в локальном `.env`. В этом режиме frontend не ждёт Telegram WebApp, а backend подставляет указанного пользователя **только при `ENV=local` и только если заголовок init data отсутствует**. Невалидный Telegram init data всё равно отклоняется; вне `ENV=local` такая конфигурация не запускается.

#### Кеш первого экрана Mini App

Каталог одобренных подарков (`GET /api/miniapp/gifts`) кешируется на стороне backend в файлах на диске, чтобы ускорить первый экран Mini App (в дефолтном состоянии фронтенд шлёт три параллельных запроса). Параметры:

- `MINIAPP_CACHE_DIR` — каталог файлового кеша. По умолчанию `./data/miniapp-cache`; в контейнере `/app/cache/miniapp`, смонтирован как Docker volume `miniapp_cache`, чтобы кеш переживал пересоздание контейнера.
- `MINIAPP_GIFTS_CACHE_TTL` — страховочный TTL записей (по умолчанию `1h`; `0` отключает истечение по времени).

Кеш события сбрасывается сразу при одобрении, правке или удалении одобренного подарка через админ-эндпоинты `PUT`/`DELETE /api/gifts/{id}`; изменение `manual_distribution` у одобренного подарка также сбрасывает кеш. Замена только ручного получателя не меняет публичный каталог и кеш не инвалидирует. TTL служит лишь подстраховкой. Сессия (`GET /api/miniapp/session`) и счётчик участников не кешируются и остаются актуальными.

### Production (Caddy + docker-server)

Production разворачивается как проект общего edge-сервера **docker-server**: TLS и
маршрутизацию по одному публичному домену (`PUBLIC_DOMAIN`) держит **Caddy**
(авто Let's Encrypt), БД — **общий postgres** сервера. Свой nginx/certbot/postgres
не поднимаются. Caddy маршрутизирует:

- `/api/*`, `/health`, `/docs/*` → backend API;
- всё остальное (`/`, `/miniapp/*`, websocket) → frontend.

Бот работает через polling, поэтому отдельный webhook-домен не нужен. Сервисы
`docker-compose.prod.yml` наружу портов не публикуют — Caddy видит их через
сеть `edge`, БД доступна через `shared-db`.

Полная инструкция (add-db, `.env.prod.example`, блок в Caddyfile, запуск) —
**[DEPLOY.md](DEPLOY.md)**. Кратко:

```bash
make docker-prod-up   # docker compose -f docker-compose.prod.yml up -d --build
```

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
make docker-prod-up # Запустить production compose (Caddy/общий postgres)
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
│   │   ├── cache/       # File-backed caches (miniapp gift catalog)
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

### Выгрузка участников

На административной странице **Участники** кнопка «Экспорт в CSV» скачивает
весь набор, соответствующий текущим фильтрам, поиску и серверной сортировке,
а не только открытую страницу. В файл попадают только выбранные в настройках
таблицы колонки и в том же порядке. CSV создаётся в UTF-8 с BOM и разделителем
`;`, поэтому его можно открыть в Excel без дополнительной библиотеки.

### Выгрузка призов

На административной странице **Призы** кнопка «Экспорт в CSV» скачивает весь
текущий отфильтрованный список, а не только открытую страницу. Экспорт учитывает
фильтры статуса проверки, автора, поиска и распределения, включая варианты
«Ручное распределение» и «Ручное распределение — получатель не назначен».
Получатель включается в файл только для ручного назначения и доступен только в
админском контексте; при отсутствии назначения соответствующая ячейка остаётся
пустой.

## Дефолтные учетные данные

После применения миграций создается дефолтный админ:
- **Username**: admin
- **Password**: admin123

⚠️ **Важно**: Смените пароль после первого входа!

В админ-панели новые администраторы создаются на странице **Администраторы**.
Текущий администратор может сменить свой пароль через меню пользователя → **Сменить пароль**; после смены пароля потребуется войти заново.

## Порты

Локальный compose:

- **3000**: Frontend (Next.js)
- **8080/18080**: Backend API, в зависимости от `API_PUBLIC_PORT`
- **5432**: PostgreSQL

Production compose (docker-server/Caddy):

- Сервисы наружу портов **не публикуют**. `80/443` держит Caddy сервера.
- `gravel-bot-api` и `gravel-bot-frontend` доступны Caddy через сеть `edge`;
  БД — через `shared-db`. Подробнее — [DEPLOY.md](DEPLOY.md).

## Лицензия

MIT

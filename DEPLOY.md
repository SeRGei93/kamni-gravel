# Деплой

Production gravel_bot разворачивается как проект общего edge-сервера
`docker-server`: TLS и маршрутизацию делает **Caddy**, БД — **общий postgres**
сервера. Свой postgres/nginx/certbot не поднимаются. Полный пример Caddyfile —
в разделе [«Полный Caddyfile»](#полный-caddyfile) ниже.

Файлы для этого режима:
- `docker-compose.prod.yml` — самодостаточный compose: сервисы без своего
  postgres/nginx, на сетях `edge` (Caddy) и `shared-db` (общий postgres), без
  публикации портов;
- `.env.prod.example` — шаблон переменных.

Контейнеры-цели для Caddy: `gravel-bot-api:8080` и `gravel-bot-frontend:3000`.

## Предусловия

Сервер `docker-server` поднят: `./up.sh` создал сети `edge` и `shared-db` и
запустил Caddy + общий `postgres`.

## Шаги

1. **Положить проект** в `docker-server/projects/gravel-bot/` (клонировать репозиторий сюда).

2. **Завести базу** в общем postgres (из каталога `docker-server`):

   ```bash
   ./add-db.sh gravel_bot gravel_bot      # выведет сгенерированный пароль — сохраните
   ```

3. **Заполнить `.env`** проекта:

   ```bash
   cd projects/gravel-bot
   cp .env.prod.example .env
   ```

   Обязательно: `PUBLIC_DOMAIN`, `DB_PASSWORD` (из шага 2), `BOT_TOKEN`,
   `JWT_SECRET`, `*_CHAT_ID`. Для бэкапа: `BACKUP_TELEGRAM_ID` и
   `POSTGRES_PASSWORD` (= `DB_PASSWORD`).

4. **Собрать и поднять** (миграции прогонит одноразовый `gravel-bot-migrate`, затем стартуют api/bot/frontend):

   ```bash
   docker compose -f docker-compose.prod.yml up -d --build
   ```

   > `NEXT_PUBLIC_API_URL` вшивается в фронтенд на этапе сборки из
   > `PUBLIC_DOMAIN`. После смены домена пересоберите фронтенд (`--build`).

5. **Добавить домен в Caddy.** `add-site.sh` умеет только один бэкенд на домен,
   а здесь нужен path-routing (`/api` → api, остальное → frontend). Допишите
   site-блок в `docker-server/Caddyfile` вручную (глобальный блок `{ email }`
   там уже есть). Полный блок — в разделе [«Полный Caddyfile»](#полный-caddyfile);
   минимальный вариант:

   ```caddy
   example.com {
       @api path /api/* /health /docs /docs/*
       handle @api {
           reverse_proxy gravel-bot-api:8080
       }
       handle {
           reverse_proxy gravel-bot-frontend:3000
       }
   }
   ```

   Проверить и перечитать (из каталога `docker-server`):

   ```bash
   docker run --rm -v "$PWD/Caddyfile:/etc/caddy/Caddyfile:ro" \
     caddy:2-alpine validate --adapter caddyfile --config /etc/caddy/Caddyfile
   docker compose exec caddy caddy reload --config /etc/caddy/Caddyfile
   ```

   Маршруты совпадают с прежним nginx: `/api/*`, `/health`, `/docs`, `/docs/*`
   → API; всё остальное (дашборд, `/miniapp/*`, websocket) → фронтенд.

## Полный Caddyfile

Полный рабочий конфиг: глобальный блок + site с path-routing, лимитом загрузки и
security-заголовками (паритет с прежним nginx). На общем `docker-server`
глобальный `{ email }` уже есть — добавляйте только site-блок (шаг 5); файл
целиком нужен для standalone-Caddy или как референс.

```caddy
{
	# Email для уведомлений Let's Encrypt (ACME).
	email admin@example.com
}

# gravel_bot — один домен, path-routing API/фронтенд.
example.com {
	# Лимит размера запроса (фото подарков загружаются через /api) — как в nginx.
	request_body {
		max_size 20MB
	}

	# Заголовки безопасности (паритет с прежним nginx).
	header {
		Strict-Transport-Security "max-age=31536000"
		X-Content-Type-Options "nosniff"
		X-Frame-Options "SAMEORIGIN"
		Referrer-Policy "strict-origin-when-cross-origin"
		-Server
	}

	# API, health-check и Swagger → backend.
	@api path /api/* /health /docs /docs/*
	handle @api {
		reverse_proxy gravel-bot-api:8080
	}

	# Дашборд, /miniapp/*, websocket → frontend.
	handle {
		reverse_proxy gravel-bot-frontend:3000
	}
}
```

Локальная разработка без ACME — тот же site-блок с `tls internal` и
`*.localhost`-доменом (Caddy выпустит сертификат внутренним CA):

```caddy
app.localhost {
	tls internal
	@api path /api/* /health /docs /docs/*
	handle @api {
		reverse_proxy gravel-bot-api:8080
	}
	handle {
		reverse_proxy gravel-bot-frontend:3000
	}
}
```

> `X-Frame-Options: SAMEORIGIN` повторяет прежний nginx и работает для Telegram
> Mini App в мобильных клиентах (webview). Если нужен запуск в Telegram Web
> (web.telegram.org грузит Mini App в iframe со своего origin) — уберите
> `X-Frame-Options` или замените на CSP `frame-ancestors`.

## Обновление

```bash
cd docker-server/projects/gravel-bot
git pull
docker compose -f docker-compose.prod.yml up -d --build
```

## Бэкап БД в Telegram (опционально)

Скрипт `scripts/backup-telegram.sh` уже параметризован — на общий postgres его
наводят переменные `.env` (`POSTGRES_CONTAINER=postgres`, `POSTGRES_DB/USER/PASSWORD`).
Cron на хосте, ежечасно:

```cron
0 * * * * cd /path/to/docker-server/projects/gravel-bot && ./scripts/backup-telegram.sh >> backup/backup.log 2>&1
```

## Заметки

- Общий postgres — **PG18** (локальный dev — PG16); миграции на чистом SQL
  (goose), совместимы.
- Загруженные файлы и кэш miniapp лежат в именованных томах `event_files` и
  `miniapp_cache`. Дамп БД их не включает — бэкапьте `event_files` отдельно,
  если храните там фото подарков.
- Если порт api/frontend не отвечает при `add-site.sh`/reload — проверьте, что
  контейнер в сети `edge`: `docker inspect -f '{{json .NetworkSettings.Networks}}' gravel-bot-api`.

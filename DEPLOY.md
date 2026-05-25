# Deploy

Краткая инструкция для production-развертывания с nginx и Let's Encrypt.

## 1. Подготовить сервер

На сервере должны быть установлены Docker и Docker Compose. В DNS направьте домен на сервер:

```text
gravel.example.com A <server-ip>
```

Откройте входящие порты. `80/tcp` нужен для первичного выпуска и продления Let's Encrypt сертификата:

```text
80/tcp
443/tcp
```

В production наружу публикуется только nginx. `postgres`, `backend-api`, `frontend` и `bot` остаются во внутренней Docker-сети.

## 2. Залить код

Вариант через git:

```bash
ssh user@server
mkdir -p /opt
cd /opt
git clone <repo-url> gravel_bot
cd gravel_bot
```

Для обновления уже развернутого проекта:

```bash
cd /opt/gravel_bot
git pull
```

## 3. Настроить .env

```bash
cp env.example .env
nano .env
```

Минимально проверьте и замените:

```env
ENV=production
PUBLIC_DOMAIN=gravel.example.com
CERTBOT_EMAIL=admin@example.com

BOT_TOKEN=...
JWT_SECRET=...
POSTGRES_PASSWORD=...

MINIAPP_URL=https://gravel.example.com/miniapp/gifts
NEXT_PUBLIC_API_URL=https://gravel.example.com
ALLOWED_ORIGINS=https://gravel.example.com
```

`MINIAPP_URL`, `NEXT_PUBLIC_API_URL` и `ALLOWED_ORIGINS` используют один и тот же публичный домен. Отдельный поддомен для miniapp не нужен.

## 4. Выпустить SSL-сертификат

Первый запуск:

```bash
make ssl-cert
```

На время выпуска скрипт останавливает nginx и запускает certbot в standalone-режиме на `80/tcp`. Сертификат сохраняется в:

```text
nginx/certbot/conf
```

Эта директория не коммитится. Повторный `make ssl-cert` не перевыпускает сертификат, если текущий сертификат еще валиден дольше `SSL_RENEW_BEFORE_SECONDS`, по умолчанию 30 дней.

Для тестовой проверки без боевых лимитов Let's Encrypt:

```env
SSL_STAGING=true
```

После успешной проверки верните:

```env
SSL_STAGING=false
```

И выпустите реальный сертификат.

## 5. Запустить production

```bash
make docker-prod-up
```

Проверить логи:

```bash
make docker-prod-logs
```

Проверить снаружи:

```bash
curl -I https://gravel.example.com/health
curl -I https://gravel.example.com/
```

## 6. Продление сертификата

Ручное продление:

```bash
make ssl-renew
```

Пример cron-задачи:

```cron
0 3 * * * cd /opt/gravel_bot && make ssl-renew >> /var/log/gravel-ssl-renew.log 2>&1
```

## 7. Обновление приложения

```bash
cd /opt/gravel_bot
git pull
make docker-prod-up
```

Если менялись только env-переменные:

```bash
make docker-prod-up
```

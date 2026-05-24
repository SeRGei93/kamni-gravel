# Multi-stage build для backend

# Базовый образ для сборки
FROM golang:1.25-alpine AS builder

# Устанавливаем необходимые инструменты для CGO (SQLite требует C компилятор)
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Копируем go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем все бинарники
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/bot ./cmd/bot
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/api ./cmd/api
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/migrate ./cmd/migrate

# Финальный образ для bot
FROM alpine:latest AS bot

RUN apk --no-cache add ca-certificates sqlite

WORKDIR /app

COPY --from=builder /app/bin/bot /app/bot

CMD ["/app/bot"]

# Финальный образ для api
FROM alpine:latest AS api

RUN apk --no-cache add ca-certificates sqlite

WORKDIR /app

COPY --from=builder /app/bin/api /app/api

EXPOSE 8080

CMD ["/app/api"]

# Финальный образ для migrate
FROM alpine:latest AS migrate

RUN apk --no-cache add ca-certificates sqlite

WORKDIR /app

COPY --from=builder /app/bin/migrate /app/migrate
COPY --from=builder /app/internal/infrastructure/migrations ./internal/infrastructure/migrations

# Устанавливаем переменную окружения для пути к миграциям
ENV MIGRATIONS_DIR=internal/infrastructure/migrations

ENTRYPOINT ["/app/migrate"]
CMD ["up"]

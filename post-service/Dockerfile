# Этап сборки
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Устанавливаем зависимости для сборки
RUN apk add --no-cache git

# Копируем go.mod и go.sum отдельно для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Собираем бинарник
RUN go build -o post-service ./cmd/app
# Собираем бинарник для миграций
RUN go build -o migrator ./cmd/migrator

# Финальный минимальный образ
FROM alpine:3.20.3

WORKDIR /root/

# Устанавливаем сертификаты (нужны для работы HTTP-клиентов)
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/post-service .
# Бинарник который умеет выполнять миграции
COPY --from=builder /app/migrator .
# Копируем файлы миграции
COPY --from=builder /app/migrations ./migrations
# Прокидываем config
COPY --from=builder /app/config ./config

EXPOSE 8000

CMD ["sh", "-c", "./migrator --migrations-path ./migrations && ./post-service"]


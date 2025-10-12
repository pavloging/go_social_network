# Этап сборки
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o notification-service ./cmd/app

# Финальный образ
FROM alpine:3.20.3

WORKDIR /root/

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/notification-service .
# Прокидываем config
COPY --from=builder /app/config ./config

EXPOSE 8001

CMD ["./notification-service"]

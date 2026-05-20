# Проект: Post & Notification System

## ⚙️ Как запустить (Docker, Version 2)

```bash
docker-compose up -d --build
```

Сервисы поднимаются с dev-конфигурацией из `docker-compose.yml` — отдельный `.env` не обязателен.

Отправьте POST-запрос:
```json
	{
		"title": "Hi",
		"author": "Jorsh",
		"content": "Hi, my name is Jorsh, i am is student and i created this service. Are you stay?",
		"tags": ["test"]
	}
```
6. Проверьте логи notification-service и Redis (см. раздел Version 2 ниже).

Для локального запуска без Docker скопируйте `.env.example` в `.env` в каждом сервисе.

## 🎯 Идея
- **post-service**: создает посты вручную или автоматически (например, генерация текста, случайные пользователи, теги) и публикует события в Kafka.  
- **notification-service**: подписан на Kafka, принимает события, фильтрует их, сохраняет посты в Redis и генерирует уведомления по ключевым событиям.  

Цель — полностью event-driven процесс: нет множества REST-запросов, все происходит через Kafka.

---

## ⚙️ Архитектура (микросервисы)

```less
    [post-service]
        │
        ▼
    Kafka (topic: posts)
        │
        ▼
[notification-service] → Redis (кэш постов / уведомлений)
│
├── Prometheus (метрики)
│
└── Promtail → Loki (логи)
```

---

### Post Service
- REST API (POST /posts)
- сохраняет пост в базу (может быть PostgreSQL)
- публикует событие в Kafka (topic=posts)

Тестирование:
```bash
cd post-service
golangci-lint run ./...
```

### Kafka
- наш Event Bus (message broker)
- хранит и доставляет сообщения (посты)
- гарантирует, что Notification Service получит событие

### Notification Service
- подписан на topic=posts
- получает сообщение → парсит его → делает "уведомление" (в минимальной версии просто пишет в логи)
- в будущем: может слать email, пуши, сохранять в БД

### 📌 Зачем Kafka
- 🔹 Развязывает сервисы (Post не зависит от Notification)
- 🔹 Гарантированная доставка сообщений
- 🔹 Легко добавлять новые подписчики (например, Analytics Service)
- 🔹 Масштабируемость

## 📦 Подробнее про реализацию сервисов

### 1️⃣ Post-Service
- Генерация постов:
  - автор, текст, теги, timestamp
  - можно использовать `faker` или random генерацию  
- Публикация в Kafka (`posts`)  

**Пример события:**
```json
{
  "id": "post_12345",
  "author": "user_42",
  "content": "Kafka makes event-driven fun!",
  "tags": ["kafka", "golang"],
  "created_at": "2025-08-17T12:00:00Z"
}
```

### 2️⃣ Notification-Service
- Подписан на топик posts
- Фильтрует посты по ключевым словам (error, alert)
- Сохраняет последние посты в Redis (например, последние 100)
- Генерирует уведомления (stdout, Redis queue или топик posts.alerts)
- Экспонирует метрики для Prometheus:
    - posts_total
    - alerts_total
    - posts_per_second

**Логи для Loki через Promtail:**

- "Generated post id=post_12345 by user_42"
- "ALERT: Post post_12345 contains keyword 'error'"

---

### 📊 Kafka топики
posts — все новые посты

posts.alerts — уведомления/алерты

---

### 🚀 Возможные расширения
Analytics-Service: подсчёт топ-авторов, топ-ключевых слов, частоты тегов

Отложенные уведомления: TTL в Redis

Kafka Streams: агрегирование событий и расчёт метрик в реальном времени

Auto-scaling consumers: демонстрация управления потребителями Kafka

---

## Version 2 — Reliable Kafka Event Bus

### Что добавлено

- **EventEnvelope** вместо голого `Post` в Kafka
- **Transactional Outbox** в post-service (таблица `outbox_events`)
- **Outbox worker** — асинхронная публикация в Kafka
- Kafka topics: `posts`, `posts.dlq`
- **Manual commit** offsets в notification-service
- **Retry** обработки при временных ошибках Redis
- **DLQ** для невалидных/необрабатываемых сообщений
- **Идемпотентность** через Redis `processed_events:<event_id>`
- Список уведомлений в Redis `notifications` (последние 100)

### Поток данных

```
POST /posts
  → post + outbox_event (одна транзакция Postgres)
  → outbox worker → Kafka topic posts (EventEnvelope)
  → notification-service
      → validate envelope
      → idempotency check (processed_events)
      → save notification in Redis
      → manual commit offset
```

### Запуск

```bash
docker-compose up -d --build
```

Kafka UI: http://localhost:8080

### Создать пост

```bash
curl -X POST http://localhost:8000/posts \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Kafka reliability",
    "author": "student",
    "content": "Testing reliable delivery",
    "tags": ["go", "kafka"]
  }'
```

### Логи

```bash
docker-compose logs -f post-service
docker-compose logs -f notification-service
```

### Redis: notifications

```bash
docker-compose exec redis redis-cli LRANGE notifications 0 10
```

### Redis: processed events

```bash
docker-compose exec redis redis-cli KEYS "processed_events:*"
```

### Outbox в Postgres

```bash
docker-compose exec postgres psql -U postgres -d postgres -c \
  "SELECT id, event_type, status, attempts, next_attempt_at, last_error FROM outbox_events ORDER BY created_at DESC LIMIT 10;"
```

### Проверка retry и DLQ

1. Остановить Redis: `docker-compose stop redis`
2. Создать пост через `curl` (см. выше)
3. Смотреть retry в логах notification-service
4. После max attempts сообщение попадёт в topic `posts.dlq` (Kafka UI)

### Проверка DLQ для невалидного JSON

Отправьте невалидное сообщение в topic `posts` через Kafka UI и проверьте topic `posts.dlq`.

### Тесты

```bash
cd post-service && go test ./...
cd notification-service && go test ./...
```

### Нагрузочный тест (опционально)

```bash
docker-compose --profile loadtest up bombardier
```

<!--
TODO Добавить описание для версии 3
TODO Сделать самому версию 3
TODO Сделать новый репо и поудалять тут все лишнее
TODO Написать всем что сделал практику и сейчас делаю видео для неё
TODO Сделать видео обзор Dockerfile, Docker-Compose, Kafka, Redis
-->
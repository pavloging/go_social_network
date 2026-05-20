# Проект: Post & Notification System

Event-driven система из двух микросервисов: **post-service** (REST + outbox) и **notification-service** (Kafka consumer + Redis).

---

## Version 2 — Reliable Kafka Event Bus

### Что добавлено

- **EventEnvelope** — единый формат события в Kafka
- **Transactional Outbox** в post-service (таблица `outbox_events`)
- **Outbox Worker** — асинхронная публикация в Kafka
- Kafka topic **posts**
- Kafka topic **posts.dlq** (dead letter queue)
- **Manual offset commit** в notification-service
- **Retry** обработки при временных ошибках Redis
- **DLQ** для невалидных и необрабатываемых сообщений
- **Idempotency** через Redis `processed_events:<event_id>`
- Список уведомлений в Redis **`notifications`** (последние 100)

### Поток данных

```
POST /posts
  → post-service создаёт Post
  → post-service создаёт EventEnvelope
  → post-service сохраняет post + outbox_event в одной транзакции (Postgres)
  → outbox worker забирает pending event
  → outbox worker публикует EventEnvelope в Kafka topic posts
  → notification-service читает topic posts
  → notification-service валидирует EventEnvelope
  → notification-service делает retry при временных ошибках
  → notification-service сохраняет notification в Redis list notifications
  → notification-service сохраняет processed_events:<event_id>
  → notification-service вручную commit offset
  → плохие сообщения уходят в posts.dlq (commit только после успешной публикации в DLQ)
```

**Важно:** `CreatePost` **не** публикует в Kafka напрямую. Публикация только через outbox worker.

### Архитектура

```less
[post-service]
  │ REST POST /posts
  ▼
Postgres (posts + outbox_events)
  │
  │ outbox worker
  ▼
Kafka (topic: posts)
  │
  ▼
[notification-service]
  │ validate → retry → idempotency
  ▼
Redis (notifications, processed_events:*)
  │
  │ invalid / max retry
  ▼
Kafka (topic: posts.dlq)
```

---

## Как запустить

```bash
docker-compose up -d --build
```

Сервисы поднимаются с dev-конфигурацией из `docker-compose.yml` — отдельный `.env` не обязателен.

Для локального запуска без Docker скопируйте `.env.example` в `.env` в каждом сервисе.

Kafka UI: http://localhost:8080

---

## Как создать пост

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

---

## Как проверить outbox

```bash
docker-compose exec postgres psql -U postgres -d postgres -c \
  "SELECT id, event_type, status, attempts, next_attempt_at, last_error FROM outbox_events ORDER BY created_at DESC LIMIT 10;"
```

Ожидаемо: после публикации worker'ом статус `published`.

### Outbox retry при недоступной Kafka

При временной недоступности Kafka post-service всё равно сохраняет `post` и `outbox_event`. Ошибка публикации в Kafka **не** переводит событие в terminal `failed`:

- `attempts` увеличивается (может расти выше `OUTBOX_MAX_ATTEMPTS` — параметр только для диагностики в логах);
- `last_error` хранит последнюю ошибку Kafka;
- `status` возвращается в `pending`;
- `next_attempt_at` задаёт паузу перед следующей попыткой;
- outbox worker повторяет публикацию до успеха;
- после восстановления Kafka событие становится `published`.

`status='failed'` зарезервирован для будущих невосстановимых ошибок outbox (не используется для временных ошибок Kafka в Version 2).

### Проверка Kafka down / outbox retry

1. Остановить Kafka:

```bash
docker-compose stop kafka
```

2. Создать пост:

```bash
curl -X POST http://localhost:8000/posts \
  -H "Content-Type: application/json" \
  -d '{"title":"Kafka down test","author":"student","content":"Outbox should keep event","tags":["outbox"]}'
```

3. Проверить outbox:

```bash
docker-compose exec postgres psql -U postgres -d postgres -c \
  "SELECT id, event_type, status, attempts, last_error FROM outbox_events ORDER BY created_at DESC LIMIT 5;"
```

Ожидаемо:

- event есть;
- `status` = `pending` или `processing` (не terminal `failed`);
- `attempts` растёт при повторных попытках worker;
- `last_error` содержит ошибку Kafka.

4. Запустить Kafka:

```bash
docker-compose start kafka
docker-compose start kafka-init
```

5. Подождать 10–30 секунд и проверить снова:

```bash
docker-compose exec postgres psql -U postgres -d postgres -c \
  "SELECT id, event_type, status, attempts, last_error FROM outbox_events ORDER BY created_at DESC LIMIT 5;"
```

Ожидаемо: `status = published`.

---

## Как проверить Redis notifications

```bash
docker-compose exec redis redis-cli LRANGE notifications 0 10
```

---

## Как проверить processed events

```bash
docker-compose exec redis redis-cli KEYS "processed_events:*"
```

---

## Как смотреть логи

```bash
docker-compose logs -f post-service
docker-compose logs -f notification-service
```

---

## Как проверить DLQ

1. Откройте Kafka UI: http://localhost:8080
2. Отправьте **невалидное** сообщение в topic `posts` (например, plain text или битый JSON)
3. Убедитесь, что notification-service **не падает** (при ошибке DLQ consumer останавливает claim и перечитывает сообщение)
4. После успешной отправки в DLQ проверьте topic **`posts.dlq`**

---

## Как проверить retry

1. Остановите Redis: `docker-compose stop redis`
2. Создайте пост через `curl` (см. выше)
3. Смотрите логи notification-service — должны быть retry
4. После max attempts сообщение должно попасть в **`posts.dlq`**
5. Запустите Redis обратно: `docker-compose start redis`

---

## Тесты

```bash
cd post-service && go test ./...
cd notification-service && go test ./...
```

### post-service

```bash
cd post-service
golangci-lint run ./...
```

### Нагрузочный тест (опционально)

```bash
docker-compose --profile loadtest up bombardier
```

---

## Мониторинг

В `docker-compose` также подняты **Prometheus** и **Grafana** (порты 9090 и 3000). Конфигурация Prometheus не менялась под Version 2.

---

## Пример EventEnvelope в Kafka

```json
{
  "event_id": "uuid",
  "event_type": "post.created",
  "version": 1,
  "occurred_at": "2025-08-17T12:00:00Z",
  "payload": {
    "id": "post-uuid",
    "title": "Kafka reliability",
    "author": "student",
    "content": "Testing reliable delivery",
    "tags": ["go", "kafka"],
    "created_at": "2025-08-17T12:00:00Z"
  }
}
```

---

### 🚀 Возможные расширения

- Analytics-Service: подсчёт топ-авторов, топ-ключевых слов, частоты тегов
- Отложенные уведомления: TTL в Redis
- Kafka Streams: агрегирование событий и расчёт метрик в реальном времени
- Auto-scaling consumers

<!--
TODO Добавить описание для версии 3
TODO Сделать самому версию 3
TODO Сделать новый репо и поудалять тут все лишнее
TODO Написать всем что сделал практику и сейчас делаю видео для неё
TODO Сделать видео обзор Dockerfile, Docker-Compose, Kafka, Redis
-->

# Проект: Post & Notification System

## ⚙️ Как запустить?
1. Склонируйте репозиторий и перейдите в него.
2. В post-service добавьте .env по примеру .example.env.
3. В корне запустите docker-compose `docker-compose up -d`.
4. Запустите post-service и notification-service через `go run путь`.
5. Отправьте POST-запрос по адресу http://localhost:8000/posts, пример запроса.
```json
	{
		"title": "Hi",
		"author": "Jorsh",
		"content": "Hi, my name is Jorsh, i am is student and i created this service. Are you stay?",
		"tags": ["test"]
	}
```
6. Все готово, перейдите в запущенный notification-service и смотрите полученные сообщения. Также сообщения сохраняются в базу данных PostgreSQL.

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

# TODO: Перенести context с r.Context http запроса
# TODO: Добавить описание для версии 2 и 3
# TODO: Сделать самому версии 2 и 3
# TODO: Сделать новый репо и поудалять тут все лишнее
# TODO: Написать всем что сделал практику и сейчас делаю видео для неё
# TODO: Сделать видео обзор Dockerfile, Docker-Compose, Kafka, Redis
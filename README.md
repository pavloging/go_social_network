# Проект: Post & Notification System

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

## 📦 Сервисы

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

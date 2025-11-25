# Kwaaka Menu Parser

Микросервис на Go для парсинга меню ресторанов из Google Sheets с сохранением в MongoDB и обработкой событий через RabbitMQ.

## Архитектура

Проект построен на **слоистой архитектуре**:

- **Repository Pattern** для доступа к данным
- **Dependency Injection** для слабой связанности
- **Очереди сообщений** для асинхронной обработки
- **RESTful API** с роутером Chi

### Project Structure

```
kwaaka-tz/
├── cmd/api/                    # Точка входа приложения
│   ├── main.go                 # Dependency injection и инициализация
│   ├── api.go                  # Роутер и жизненный цикл сервера
│   ├── parse.go                # Эндпоинты парсинга
│   ├── menu.go                 # Эндпоинты меню
│   ├── products.go             # Эндпоинты продуктов
│   ├── health.go               # Health check
│   ├── errors.go               # Обработчики ошибок
│   └── json.go                 # JSON утилиты
├── internal/
│   ├── domain/                 # Доменные модели
│   │   ├── menu.go
│   │   ├── parsing_task.go
│   │   ├── product_status_audit.go
│   │   └── events.go
│   ├── repo/                   # Интерфейсы репозиториев
│   │   ├── menu.go
│   │   ├── parsing_task.go
│   │   └── product_status_audit.go
│   ├── store/mongo/            # MongoDB реализации
│   │   ├── storage.go
│   │   ├── menu.go
│   │   ├── parsing_task.go
│   │   └── product_status_audit.go
│   ├── service/                # Бизнес-логика
│   │   ├── parsing.go
│   │   └── product.go
│   ├── parser/                 # Google Sheets парсер
│   │   └── google_sheets.go
│   ├── queue/                  # Message broker
│   │   ├── broker.go
│   │   └── rabbitmq.go
│   ├── worker/                 # Воркеры очередей
│   │   ├── menu_parsing.go
│   │   └── product_status.go
│   ├── env/                    # Environment утилиты
│   └── ratelimiter/            # Rate limiting
├── .env                        # Переменные окружения для локальной разработки
├── .env.docker                 # Переменные окружения для Docker
└── docker-compose.yml          # Настройка инфраструктуры
```

## Запуск

### Требования

- Docker и Docker Compose
- Google Cloud Service Account (для Sheets API)

### 1. Настройка Google Sheets API

1. Создайте проект в Google Cloud Console
2. Включите Google Sheets API
3. Создайте Service Account
4. Скачайте JSON с credentials
5. Сохраните файл как `credentials.json` в корне проекта
6. Расшарьте ваш Google Sheet с email сервисного аккаунта

### 2. Запуск приложения

Убедитесь, что файл `credentials.json` находится в корне проекта, затем:

```bash
docker-compose up -d --build
```

Эта команда запустит:

- **MongoDB** на порту `27017` (логин: `admin`, пароль: `password`)
- **RabbitMQ** на порту `5672` (логин: `admin`, пароль: `password`)
- **RabbitMQ Management UI** на порту `15672` (http://localhost:15672)
- **API сервер** на порту `8080` (http://localhost:8080)
- **Worker** для обработки очередей

### Swagger документация

Документация доступна по адресу (http://localhost:8080/api/v1/swagger/index.html#/)

### 3. Файлы конфигурации

Проект использует два файла конфигурации:

#### `.env.docker` - для Docker Compose

Используется когда всё приложение запускается в Docker (включая API и Worker).

Файл `.env.docker` уже создан в проекте.

#### `.env` - для локальной разработки

Используется когда вы запускаете приложение локально (через `air` или `go run`), а MongoDB и RabbitMQ работают в Docker. Это опционально.

Создайте файл `.env` как `.env.docker`.

**Когда нужен `.env`?**

- Только для локальной разработки с hot-reload (Air)
- Можно не создавать, если используете только Docker

**Когда нужен `.env.docker`?**

- Для запуска в Docker Compose (уже создан в проекте)
- Используется автоматически контейнерами `api` и `worker`

## Формат Google Sheets

Таблица должна соответствовать структуре из примера.

**Заголовки столбцов:**

```
ProductID | ProductName | IsCombo | Price | Description | AttributeGroupID | AttributeGroupName | Min | Max | AttributeID | AttributeName | AttributeMin | AttributeMax | AttributePrice
```

## Обработка очередей

### Очередь парсинга меню (`menu-parsing`)

- Получает задачи парсинга из API
- Парсит Google Sheets
- Сохраняет меню в MongoDB
- Обновляет статус задачи

### Очередь статусов продуктов (`product-status`)

- Получает события изменения статуса продуктов
- Создает записи аудита
- Поддерживает события:
  - `product.created`
  - `product.updated`
  - `product.status_changed`
  - `product.deleted`

### Механизм повторных попыток

- **Максимум попыток:** 3
- **Задержка:** Экспоненциальная (2^n секунд: 2s, 4s, 8s)
- **Dead Letter Queue (DLQ):** Неудачные сообщения отправляются в `{queue-name}-dlq`

## Коллекции MongoDB

### `menus`

```json
{
  "_id": "ObjectId",
  "name": "Название ресторана",
  "restaurant_id": "restaurant-slug",
  "products": [],
  "attributes_groups": [],
  "attributes": [],
  "created_at": "2025-11-24T10:00:00Z",
  "updated_at": "2025-11-24T10:00:00Z"
}
```

### `parsing_tasks`

```json
{
  "_id": "ObjectId",
  "status": "completed",
  "spreadsheet_id": "1ABC...",
  "restaurant_name": "Название ресторана",
  "menu_id": "ObjectId",
  "error_message": "",
  "retry_count": 0,
  "created_at": "2025-11-24T10:00:00Z",
  "updated_at": "2025-11-24T10:01:00Z"
}
```

### `product_status_audit`

```json
{
  "_id": "ObjectId",
  "product_id": "1001",
  "event_type": "product.status_changed",
  "old_status": "available",
  "new_status": "not_available",
  "reason": "out_of_stock",
  "user_id": "admin_123",
  "timestamp": "2025-11-24T10:00:00Z"
}
```

## Разработка и отладка

### Доступ к MongoDB

```bash
docker exec -it kwaaka-mongo mongosh -u admin -p password
use kwaaka
db.menus.find()
db.parsing_tasks.find()
db.product_status_audit.find()
```

## Graceful Shutdown

Приложение корректно обрабатывает сигналы `SIGINT` и `SIGTERM`:

1. Прекращает принимать новые HTTP запросы
2. Останавливает воркеры очередей
3. Закрывает соединения с MongoDB
4. Закрывает соединения с RabbitMQ
5. Ждёт до 30 секунд для завершения операций

## Технические особенности

- **Rate limiting:** 20 запросов в 5 секунд (по умолчанию)
- **Context timeouts:** Все операции с БД имеют таймауты 5-30 секунд
- **Connection pooling:** MongoDB использует пул соединений (MaxPoolSize=100, MinPoolSize=10)
- **Транзакции:** Критичные операции (создание меню + обновление задачи, обновление статуса + создание аудита) выполняются атомарно
- **Индексы:** Автоматическое создание индексов при старте приложения
- **Retry mechanism:** Экспоненциальная задержка между попытками (2^n секунд)
- **Health checks:** Все сервисы имеют health checks для мониторинга

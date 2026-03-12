# telegram-mock-ai

Mock-сервер Telegram Bot API с генерацией ответов через LLM. Полностью эмулирует `api.telegram.org` для тестирования ботов без подключения к реальному Telegram.

## Возможности

- **Полная эмуляция Bot API** — 27 методов: `sendMessage`, `getUpdates`, `setWebhook`, `getMe`, `getChat`, медиа-методы и другие
- **Long polling** — `getUpdates` с поддержкой `timeout`, `offset`, `limit`
- **Webhook** — доставка обновлений на указанный URL с ретраями и `X-Telegram-Bot-Api-Secret-Token`
- **LLM-генерация ответов** — любой OpenAI-совместимый endpoint (OpenAI, Ollama, LM Studio, vLLM и др.)
- **Проактивный режим** — сервер самостоятельно генерирует события: сообщения от пользователей, вход/выход из чата, фото, стикеры
- **Admin API** — управление состоянием: создание пользователей/чатов, инъекция сообщений и обновлений
- **Авто-регистрация ботов** — любой токен автоматически создаёт бота при первом обращении
- **Seed-данные** — предустановленные пользователи, чаты и боты через конфигурацию

## Быстрый старт

### Из исходников

```bash
# Клонировать
git clone https://github.com/skrashevich/telegram-mock-ai.git
cd telegram-mock-ai

# Скопировать конфиг
cp config.example.yaml config.yaml

# Собрать и запустить
make run
```

### Docker

```bash
# Скопировать конфиг
cp config.example.yaml config.yaml

# Запустить с Ollama в качестве LLM-провайдера
docker compose up -d

# Загрузить модель в Ollama (при первом запуске)
docker exec ollama ollama pull llama3
```

### Docker (без Ollama — только mock API)

```bash
docker build -t telegram-mock-ai .
docker run -p 8081:8081 -p 8082:8082 \
  -e TELEGRAM_MOCK_LLM_ENABLED=false \
  telegram-mock-ai
```

Сервер запустится на `http://localhost:8081` (Bot API) и `http://localhost:8082` (Admin API).

## Подключение бота

Замените базовый URL Telegram API в вашем боте на адрес mock-сервера:

### Python (python-telegram-bot)

```python
from telegram.ext import ApplicationBuilder

app = (
    ApplicationBuilder()
    .token("123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11")
    .base_url("http://localhost:8081/bot")
    .build()
)
```

### Python (aiogram)

```python
from aiogram import Bot
from aiogram.client.session.aiohttp import AiohttpSession

session = AiohttpSession()
session.api = "http://localhost:8081"
bot = Bot(token="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", session=session)
```

### Go (telebot)

```go
import tele "gopkg.in/telebot.v4"

bot, _ := tele.NewBot(tele.Settings{
    Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
    URL:   "http://localhost:8081",
})
```

### Node.js (telegraf)

```javascript
const { Telegraf } = require('telegraf');

const bot = new Telegraf('123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11', {
  telegram: { apiRoot: 'http://localhost:8081' }
});
```

### curl

```bash
# getMe
curl http://localhost:8081/bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11/getMe

# sendMessage
curl -X POST http://localhost:8081/bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11/sendMessage \
  -H 'Content-Type: application/json' \
  -d '{"chat_id": -1001, "text": "Hello from bot!"}'

# getUpdates (long polling, 10 секунд)
curl -X POST http://localhost:8081/bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11/getUpdates \
  -H 'Content-Type: application/json' \
  -d '{"timeout": 10, "offset": 0}'
```

## Конфигурация

### Файл config.yaml

```yaml
server:
  host: "0.0.0.0"         # Адрес для Bot API
  port: 8081               # Порт Bot API
  read_timeout: 60s
  write_timeout: 60s

llm:
  enabled: true
  base_url: "http://localhost:11434/v1"  # OpenAI-совместимый endpoint
  api_key: ""                             # Для OpenAI: sk-...
  model: "gpt-4o-mini"                   # Имя модели
  temperature: 0.8
  max_tokens: 512
  timeout: 30s
  response_delay_min: 500ms              # Минимальная задержка перед ответом
  response_delay_max: 3s                 # Максимальная задержка

proactive:
  enabled: false                          # Включить проактивный режим
  interval_min: 10s                       # Минимальный интервал между событиями
  interval_max: 60s                       # Максимальный интервал
  scenarios:
    - type: user_message                  # Текстовое сообщение
      weight: 0.6
    - type: new_member                    # Новый участник чата
      weight: 0.1
    - type: member_left                   # Участник покинул чат
      weight: 0.05
    - type: photo_message                 # Фото с подписью
      weight: 0.15
    - type: sticker_message              # Стикер
      weight: 0.1

webhook:
  max_retries: 3                          # Число повторных попыток доставки
  retry_delay: 1s                         # Задержка между попытками
  timeout: 10s

seed:
  users:                                  # Предустановленные пользователи
    - id: 1001
      first_name: "Alice"
      username: "alice"
    - id: 1002
      first_name: "Bob"
      last_name: "Smith"
      username: "bob_smith"
  chats:                                  # Предустановленные чаты
    - id: -1001
      type: "group"
      title: "Test Group"
      members: [1001, 1002]
  bots:                                   # Предустановленные боты
    - token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
      username: "test_bot"
      first_name: "Test Bot"

log:
  level: "info"                           # debug, info, warn, error
  format: "text"                          # text или json

admin:
  enabled: true
  host: "127.0.0.1"                       # Admin API только на localhost
  port: 8082
```

### Переменные окружения

Переменные окружения имеют приоритет над значениями из `config.yaml`:

| Переменная | Описание | Пример |
|---|---|---|
| `TELEGRAM_MOCK_SERVER_HOST` | Адрес Bot API | `0.0.0.0` |
| `TELEGRAM_MOCK_SERVER_PORT` | Порт Bot API | `8081` |
| `TELEGRAM_MOCK_LLM_ENABLED` | Включить LLM | `true` |
| `TELEGRAM_MOCK_LLM_BASE_URL` | URL LLM endpoint | `http://localhost:11434/v1` |
| `TELEGRAM_MOCK_LLM_API_KEY` | API-ключ LLM | `sk-...` |
| `TELEGRAM_MOCK_LLM_MODEL` | Модель LLM | `gpt-4o-mini` |
| `TELEGRAM_MOCK_PROACTIVE_ENABLED` | Проактивный режим | `true` |
| `TELEGRAM_MOCK_LOG_LEVEL` | Уровень логирования | `debug` |
| `TELEGRAM_MOCK_ADMIN_PORT` | Порт Admin API | `8082` |

## LLM-провайдеры

### Ollama (локально)

```yaml
llm:
  base_url: "http://localhost:11434/v1"
  model: "llama3"
```

```bash
# Установить и запустить Ollama
ollama pull llama3
ollama serve
```

### OpenAI

```yaml
llm:
  base_url: "https://api.openai.com/v1"
  api_key: "sk-..."
  model: "gpt-4o-mini"
```

### LM Studio

```yaml
llm:
  base_url: "http://localhost:1234/v1"
  model: "local-model"
```

### OpenRouter

```yaml
llm:
  base_url: "https://openrouter.ai/api/v1"
  api_key: "sk-or-..."
  model: "meta-llama/llama-3-8b-instruct"
```

### Без LLM

```yaml
llm:
  enabled: false
```

При отключённом LLM бот будет получать только те обновления, которые вы создаёте через Admin API.

## Проактивный режим

Когда включён проактивный режим, сервер самостоятельно генерирует события — как в реальном Telegram, когда пользователи пишут в чат без запроса от бота.

### Включение

```yaml
proactive:
  enabled: true
  interval_min: 10s    # Минимум 10 секунд между событиями
  interval_max: 60s    # Максимум 60 секунд
```

Или через переменную окружения:

```bash
TELEGRAM_MOCK_PROACTIVE_ENABLED=true
```

### Типы событий

| Тип | Описание | Вес по умолчанию |
|---|---|---|
| `user_message` | Текстовое сообщение от пользователя | 0.6 |
| `photo_message` | Фото с подписью, сгенерированной LLM | 0.15 |
| `sticker_message` | Случайный стикер (эмодзи) | 0.1 |
| `new_member` | Новый участник вступает в чат | 0.1 |
| `member_left` | Участник покидает чат | 0.05 |

Веса определяют вероятность каждого типа события. Чем выше вес — тем чаще событие.

### Как работает

1. Таймер срабатывает каждые `interval_min`...`interval_max` секунд (случайный интервал)
2. Выбирается случайный бот и случайный чат, в котором он состоит
3. Выбирается тип события по весам
4. LLM генерирует контент (текст сообщения, подпись к фото и т.д.)
5. Обновление доставляется боту через его очередь или webhook

## Admin API

Admin API позволяет управлять состоянием mock-сервера: создавать пользователей и чаты, инъецировать сообщения, просматривать состояние.

По умолчанию доступен только на `127.0.0.1:8082`.

### Endpoints

#### Health Check

```bash
GET /api/health
# {"status": "ok"}
```

#### Пользователи

```bash
# Список всех пользователей
GET /api/users

# Создать пользователя
POST /api/users
{"first_name": "Diana", "username": "diana", "id": 2001}
```

#### Чаты

```bash
# Список всех чатов
GET /api/chats

# Создать чат с участниками
POST /api/chats
{"type": "group", "title": "New Group", "id": -2001, "members": [1001, 1002]}

# Список участников чата
GET /api/chats/-1001/members

# Добавить участника
POST /api/chats/-1001/members
{"user_id": 2001, "status": "member"}

# История сообщений (последние 50)
GET /api/chats/-1001/messages?limit=50
```

#### Инъекция сообщений

Имитация отправки сообщения от пользователя — все подключённые боты получат обновление:

```bash
POST /api/chats/-1001/messages
{"user_id": 1001, "text": "Привет, бот!"}
```

#### Инъекция обновлений

Отправка произвольного Update конкретному боту:

```bash
POST /api/bots/123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11/updates
{
  "message": {
    "message_id": 999,
    "from": {"id": 1001, "first_name": "Alice", "is_bot": false},
    "chat": {"id": -1001, "type": "group", "title": "Test Group"},
    "date": 1709000000,
    "text": "Custom update"
  }
}
```

#### Боты

```bash
# Список зарегистрированных ботов
GET /api/bots
```

#### Полный дамп состояния

```bash
GET /api/state
```

## Реализованные методы Bot API

### Информация

| Метод | Описание |
|---|---|
| `getMe` | Информация о боте |
| `getChat` | Информация о чате |
| `getChatMember` | Информация об участнике чата |
| `getChatMemberCount` | Количество участников |
| `getChatAdministrators` | Список администраторов |

### Обновления

| Метод | Описание |
|---|---|
| `getUpdates` | Long polling с timeout |
| `setWebhook` | Установить webhook URL |
| `deleteWebhook` | Удалить webhook |
| `getWebhookInfo` | Информация о webhook |

### Сообщения

| Метод | Описание |
|---|---|
| `sendMessage` | Отправить текст |
| `editMessageText` | Редактировать текст |
| `editMessageReplyMarkup` | Редактировать inline-клавиатуру |
| `deleteMessage` | Удалить сообщение |
| `forwardMessage` | Переслать сообщение |
| `copyMessage` | Копировать сообщение |
| `answerCallbackQuery` | Ответить на callback |

### Медиа

| Метод | Описание |
|---|---|
| `sendPhoto` | Отправить фото (stub file_id) |
| `sendDocument` | Отправить документ |
| `sendVideo` | Отправить видео |
| `sendAudio` | Отправить аудио |
| `sendVoice` | Отправить голосовое |
| `sendSticker` | Отправить стикер |
| `sendAnimation` | Отправить анимацию |
| `sendLocation` | Отправить локацию |

### Управление чатом

| Метод | Описание |
|---|---|
| `banChatMember` | Забанить участника |
| `unbanChatMember` | Разбанить участника |
| `restrictChatMember` | Ограничить участника |
| `promoteChatMember` | Повысить до администратора |
| `leaveChat` | Покинуть чат |

## Архитектура

```
cmd/telegram-mock-ai/main.go     — точка входа, wiring, graceful shutdown
internal/
├── api/                          — HTTP-обработчики Bot API и Admin API
├── bot/                          — Реестр ботов (авто-регистрация по токену)
├── state/                        — In-memory хранилище (users, chats, messages, members)
├── updates/                      — Очередь обновлений + диспетчер (queue/webhook)
├── llm/                          — Клиент OpenAI-совместимого API
├── webhook/                      — Доставка обновлений по webhook с ретраями
├── proactive/                    — Движок проактивной генерации событий
├── config/                       — YAML + env конфигурация
└── models/                       — Структуры данных Telegram API
```

### Поток данных

```
Бот → POST /bot{token}/sendMessage → Сохранение в state → Ответ боту
                                          ↓
                                    [async] LLM генерирует ответ
                                          ↓
                                    Обновление в очередь/webhook → Бот
```

```
Проактивный движок → Таймер → Выбор сценария → LLM → Обновление → Бот
```

## Сборка

```bash
# Собрать бинарник
make build

# Собрать и запустить
make run

# Тесты
make test

# Docker
docker build -t telegram-mock-ai .
```

## Требования

- Go 1.22+ (для сборки из исходников)
- Docker и Docker Compose (для Docker-сборки)
- OpenAI-совместимый LLM endpoint (опционально)

## Лицензия

MIT

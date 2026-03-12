# telegram-mock-ai

Mock-сервер Telegram Bot API с генерацией ответов через LLM. Эмулирует `api.telegram.org` — бот подключается к нему вместо реального Telegram и работает с виртуальными пользователями, чатами и сообщениями.

Зачем это нужно:

- **Разработка без Telegram** — не нужен интернет, токен от BotFather и реальные пользователи
- **Автоматическое тестирование** — воспроизводимые сценарии: входящие сообщения, callback-запросы, медиа, вход/выход участников
- **Нагрузочное тестирование** — проактивный режим генерирует поток событий с настраиваемой частотой
- **Интеграционные тесты в CI** — сервер поднимается в Docker, бот работает как в продакшене

LLM (Ollama, OpenAI, Claude, LM Studio, любой OpenAI/Anthropic-совместимый endpoint) генерирует реалистичные ответы «от пользователей» и может создать стартовый набор чатов/юзеров, чтобы не прописывать их вручную.

---

## Установка и запуск

### Вариант 1: Docker + Ollama (рекомендуется)

```bash
git clone https://github.com/skrashevich/telegram-mock-ai.git
cd telegram-mock-ai
cp config.example.yaml config.yaml
docker compose up -d
```

При первом запуске загрузите модель в Ollama:

```bash
docker exec ollama ollama pull llama3
```

Готово. Bot API доступен на `http://localhost:8081`, Admin API — на `http://localhost:8082`.

### Вариант 2: Docker без LLM

Если LLM не нужен — только mock API с ручным управлением через Admin API:

```bash
docker build -t telegram-mock-ai .
docker run -p 8081:8081 -p 8082:8082 \
  -e TELEGRAM_MOCK_LLM_ENABLED=false \
  telegram-mock-ai
```

### Вариант 3: из исходников

Требуется Go 1.22+.

```bash
git clone https://github.com/skrashevich/telegram-mock-ai.git
cd telegram-mock-ai
cp config.example.yaml config.yaml
make run
```

---

## Подключение бота

Замените базовый URL Telegram API на адрес mock-сервера. Токен может быть любым — сервер автоматически зарегистрирует бота при первом обращении.

### Python (python-telegram-bot)

```python
from telegram.ext import ApplicationBuilder

app = (
    ApplicationBuilder()
    .token("YOUR_TOKEN")
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
bot = Bot(token="YOUR_TOKEN", session=session)
```

### Go (telebot)

```go
bot, _ := tele.NewBot(tele.Settings{
    Token: "YOUR_TOKEN",
    URL:   "http://localhost:8081",
})
```

### Node.js (telegraf)

```javascript
const bot = new Telegraf('YOUR_TOKEN', {
  telegram: { apiRoot: 'http://localhost:8081' }
});
```

### curl

```bash
# Проверить подключение
curl http://localhost:8081/botYOUR_TOKEN/getMe

# Отправить сообщение в чат
curl -X POST http://localhost:8081/botYOUR_TOKEN/sendMessage \
  -H 'Content-Type: application/json' \
  -d '{"chat_id": -1001, "text": "Hello!"}'

# Long polling (ожидание 10 сек)
curl -X POST http://localhost:8081/botYOUR_TOKEN/getUpdates \
  -d '{"timeout": 10}'
```

---

## Типичный сценарий использования

После запуска сервер уже содержит тестовые данные из конфига: три пользователя (Alice, Bob, Charlie), два чата и одного бота. Бот автоматически добавлен во все чаты, причём **в первом чате — с правами администратора**.

При первом подключении бота (первый вызов любого метода API) сервер **сразу отправляет сообщение** от случайного пользователя в один из чатов. В течение **30 секунд** бот получит сообщения и во всех остальных чатах — это имитирует реальную активность сразу после запуска. Если включён LLM, сообщения генерируются нейросетью; без LLM используются шаблонные приветствия.

Чтобы не описывать пользователей вручную, можно включить **автогенерацию через LLM** — сервер сам создаст реалистичных пользователей, группы и каналы с именами на нужном языке.

### Автогенерация seed-данных

**Через конфиг** (при запуске сервера):

```yaml
seed:
  generate:
    enabled: true
    users_count: 10
    groups_count: 3
    channels_count: 1
    locale: "ru"
```

**Через Admin API** (в любой момент):

```bash
curl -X POST http://localhost:8082/api/seed/generate \
  -H 'Content-Type: application/json' \
  -d '{"users_count": 10, "groups_count": 3, "channels_count": 1, "locale": "ru"}'
```

LLM сгенерирует пользователей с реалистичными именами и username, создаст группы и каналы с осмысленными названиями, распределит участников по чатам. Все зарегистрированные боты автоматически добавляются в каждый сгенерированный чат, причём **в первом чате — с правами администратора**. Ответ содержит созданные сущности с присвоенными ID.

### Инъекция сообщений

Имитация отправки сообщения «от пользователя» — все подключённые боты получат обновление:

```bash
curl -X POST http://localhost:8082/api/chats/-1001/messages \
  -H 'Content-Type: application/json' \
  -d '{"user_id": 1001, "text": "Привет, бот!"}'
```

### Проактивный режим

Сервер может сам генерировать поток событий — сообщения, вход/выход участников, фото, стикеры — с настраиваемой частотой:

```yaml
proactive:
  enabled: true
  interval_min: 10s
  interval_max: 60s
```

---

## Конфигурация

### LLM-провайдеры

Сервер поддерживает два протокола API: OpenAI-совместимый (по умолчанию) и Anthropic.

**Ollama (локально):**

```yaml
llm:
  base_url: "http://localhost:11434/v1"
  model: "llama3"
```

**OpenAI:**

```yaml
llm:
  base_url: "https://api.openai.com/v1"
  api_key: "sk-..."
  model: "gpt-4o-mini"
```

**Anthropic (Claude):**

```yaml
llm:
  api_type: "anthropic"
  base_url: "https://api.anthropic.com/v1"
  api_key: "sk-ant-..."
  model: "claude-sonnet-4-5-20250929"
  max_tokens: 1024
```

**LM Studio / vLLM / любой OpenAI-совместимый сервер:**

```yaml
llm:
  base_url: "http://localhost:1234/v1"
  model: "local-model"
```

**Без LLM:**

```yaml
llm:
  enabled: false
```

При отключённом LLM бот получает только обновления, созданные вручную через Admin API или прописанные в seed-данных.

### Полный конфиг (config.yaml)

```yaml
server:
  host: "0.0.0.0"
  port: 8081
  read_timeout: 60s
  write_timeout: 60s

llm:
  enabled: true
  api_type: "openai"              # "openai" или "anthropic"
  base_url: "http://localhost:11434/v1"
  api_key: ""
  model: "gpt-4o-mini"
  temperature: 0.8
  max_tokens: 512
  timeout: 30s
  response_delay_min: 500ms       # Имитация «печатает...»
  response_delay_max: 3s

proactive:
  enabled: false
  interval_min: 10s
  interval_max: 60s
  scenarios:
    - type: user_message
      weight: 0.6
    - type: new_member
      weight: 0.1
    - type: member_left
      weight: 0.05
    - type: photo_message
      weight: 0.15
    - type: sticker_message
      weight: 0.1

webhook:
  max_retries: 3
  retry_delay: 1s
  timeout: 10s

seed:
  generate:
    enabled: false
    users_count: 10
    groups_count: 3
    channels_count: 1
    locale: "ru"
    max_retries: 2
  users:
    - id: 1001
      first_name: "Alice"
      username: "alice"
    - id: 1002
      first_name: "Bob"
      last_name: "Smith"
      username: "bob_smith"
    - id: 1003
      first_name: "Charlie"
      username: "charlie"
  chats:
    - id: -1001
      type: "group"
      title: "Test Group"
      members: [1001, 1002, 1003]
    - id: -1002
      type: "supergroup"
      title: "Development Chat"
      members: [1001, 1002]
  bots:
    - token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
      username: "test_bot"
      first_name: "Test Bot"

log:
  level: "info"                   # debug, info, warn, error
  format: "text"                  # text, json

admin:
  enabled: true
  host: "127.0.0.1"              # Только localhost
  port: 8082
```

### Переменные окружения

Имеют приоритет над `config.yaml`:

| Переменная | Описание |
|---|---|
| `TELEGRAM_MOCK_SERVER_HOST` | Адрес Bot API (по умолчанию `0.0.0.0`) |
| `TELEGRAM_MOCK_SERVER_PORT` | Порт Bot API (по умолчанию `8081`) |
| `TELEGRAM_MOCK_LLM_ENABLED` | Включить LLM (`true`/`false`) |
| `TELEGRAM_MOCK_LLM_API_TYPE` | Протокол API: `openai`, `anthropic` |
| `TELEGRAM_MOCK_LLM_BASE_URL` | URL LLM endpoint |
| `TELEGRAM_MOCK_LLM_API_KEY` | API-ключ |
| `TELEGRAM_MOCK_LLM_MODEL` | Имя модели |
| `TELEGRAM_MOCK_PROACTIVE_ENABLED` | Включить проактивный режим |
| `TELEGRAM_MOCK_SEED_GENERATE_ENABLED` | Включить автогенерацию seed-данных |
| `TELEGRAM_MOCK_LOG_LEVEL` | Уровень логов: `debug`, `info`, `warn`, `error` |
| `TELEGRAM_MOCK_ADMIN_PORT` | Порт Admin API (по умолчанию `8082`) |

---

## Admin API

Управление состоянием mock-сервера. По умолчанию доступен на `127.0.0.1:8082`.

| Метод | Endpoint | Описание |
|---|---|---|
| `GET` | `/api/health` | Health check |
| `GET` | `/api/state` | Полный дамп состояния (пользователи, чаты, боты) |
| `GET` | `/api/users` | Список пользователей |
| `POST` | `/api/users` | Создать пользователя |
| `GET` | `/api/chats` | Список чатов |
| `POST` | `/api/chats` | Создать чат |
| `GET` | `/api/chats/{id}/members` | Список участников чата |
| `POST` | `/api/chats/{id}/members` | Добавить участника |
| `GET` | `/api/chats/{id}/messages` | История сообщений |
| `POST` | `/api/chats/{id}/messages` | Инъекция сообщения от пользователя |
| `GET` | `/api/bots` | Список ботов |
| `POST` | `/api/bots/{token}/updates` | Инъекция произвольного Update |
| `POST` | `/api/seed/generate` | Генерация seed-данных через LLM |

### Примеры

```bash
# Создать пользователя
curl -X POST http://localhost:8082/api/users \
  -d '{"first_name": "Diana", "username": "diana"}'

# Создать группу
curl -X POST http://localhost:8082/api/chats \
  -d '{"type": "group", "title": "New Group", "members": [1001, 1002]}'

# Отправить сообщение от пользователя (боты получат обновление)
curl -X POST http://localhost:8082/api/chats/-1001/messages \
  -d '{"user_id": 1001, "text": "Привет!"}'

# Сгенерировать пользователей и чаты через LLM
curl -X POST http://localhost:8082/api/seed/generate \
  -d '{"users_count": 5, "groups_count": 2, "locale": "ru"}'

# Инъекция произвольного Update конкретному боту
curl -X POST http://localhost:8082/api/bots/YOUR_TOKEN/updates \
  -d '{"message":{"message_id":1,"from":{"id":1001,"first_name":"Alice"},"chat":{"id":-1001,"type":"group"},"text":"test"}}'
```

---

## Реализованные методы Bot API

27 методов, покрывающих основные сценарии работы ботов:

| Категория | Методы |
|---|---|
| Информация | `getMe`, `getChat`, `getChatMember`, `getChatMemberCount`, `getChatAdministrators` |
| Обновления | `getUpdates`, `setWebhook`, `deleteWebhook`, `getWebhookInfo` |
| Сообщения | `sendMessage`, `editMessageText`, `editMessageReplyMarkup`, `deleteMessage`, `forwardMessage`, `copyMessage`, `answerCallbackQuery` |
| Медиа | `sendPhoto`, `sendDocument`, `sendVideo`, `sendAudio`, `sendVoice`, `sendSticker`, `sendAnimation`, `sendLocation` |
| Управление чатом | `banChatMember`, `unbanChatMember`, `restrictChatMember`, `promoteChatMember`, `leaveChat` |

---

## Архитектура

```
cmd/telegram-mock-ai/main.go     — точка входа, wiring, graceful shutdown
internal/
├── api/          — HTTP-обработчики Bot API и Admin API
├── bot/          — Реестр ботов (авто-регистрация по токену)
├── state/        — In-memory хранилище (users, chats, messages, members)
├── updates/      — Очередь обновлений + диспетчер (queue/webhook)
├── seed/         — LLM-генерация seed-данных
├── llm/          — Клиент OpenAI/Anthropic API + промпты
├── webhook/      — Доставка обновлений по webhook с ретраями
├── proactive/    — Движок проактивной генерации событий
├── config/       — YAML + env конфигурация
└── models/       — Структуры данных Telegram API
```

```
Бот → POST /bot{token}/sendMessage → state → [async LLM ответ] → очередь/webhook → Бот
Проактивный движок → таймер → сценарий → LLM → обновление → Бот
Admin API → POST /api/seed/generate → LLM → users + chats в state
```

## Сборка

```bash
make build    # бинарник
make run      # собрать и запустить
make test     # тесты
```

## Лицензия

Apache 2.0

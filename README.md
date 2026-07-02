# telegram-mock-ai
[![Go Report Card](https://img.shields.io/badge/go%20report-A%2B-brightgreen?style=flat&logo=go)](https://goreportcard.com/report/github.com/skrashevich/telegram-mock-ai)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/skrashevich/telegram-mock-ai.svg)](https://pkg.go.dev/github.com/skrashevich/telegram-mock-ai)
[![GitHub release](https://img.shields.io/github/v/release/skrashevich/telegram-mock-ai?include_prereleases)](https://github.com/skrashevich/telegram-mock-ai/releases)
[![Download nightly](https://img.shields.io/badge/dawnl.ink-nightly%20builds-blue)](https://dawnl.ink/skrashevich/telegram-mock-ai/workflows/nightly/main)

<!-- badges:start -->
[![GitHub stars](https://img.shields.io/github/stars/skrashevich/telegram-mock-ai?style=flat-square)](https://github.com/skrashevich/telegram-mock-ai/stargazers)
[![Last commit](https://img.shields.io/github/last-commit/skrashevich/telegram-mock-ai?style=flat-square)](https://github.com/skrashevich/telegram-mock-ai/commits/main)
[![License](https://img.shields.io/github/license/skrashevich/telegram-mock-ai?style=flat-square)](https://github.com/skrashevich/telegram-mock-ai/blob/main/LICENSE)
<!-- badges:end -->

Mock Telegram Bot API server with LLM-generated replies. It emulates `api.telegram.org`, so a bot connects to it instead of the real Telegram API and works with virtual users, chats, and messages.

Why use it:

- **Develop without Telegram**: no internet access, no BotFather token, no real users required
- **Automated testing**: reproducible scenarios for incoming messages, callback queries, media, joins, and leaves
- **Load testing**: proactive mode generates a configurable event stream
- **CI integration tests**: run the server in Docker and test the bot as if it were in production

An LLM (Ollama, OpenAI, Claude, LM Studio, or any OpenAI/Anthropic-compatible endpoint) can generate realistic user replies and create an initial set of chats and users so you do not have to define everything by hand.

---

## Installation and Startup

### Option 1: Docker + Ollama (recommended)

```bash
git clone https://github.com/skrashevich/telegram-mock-ai.git
cd telegram-mock-ai
cp config.example.yaml config.yaml
docker compose up -d
```

On first run, pull a model into Ollama:

```bash
docker exec ollama ollama pull llama3
```

Done. The Bot API will be available at `http://localhost:8081`, and the Admin API at `http://localhost:8082`.

### Option 2: Docker without an LLM

If you only need the mock API with manual control via the Admin API:

```bash
docker build -t telegram-mock-ai .
docker run -p 8081:8081 -p 8082:8082 \
  -e TELEGRAM_MOCK_LLM_ENABLED=false \
  telegram-mock-ai
```

### Option 3: From source

Requires Go 1.22+.

```bash
git clone https://github.com/skrashevich/telegram-mock-ai.git
cd telegram-mock-ai
cp config.example.yaml config.yaml
make run
```

---

## Connecting a Bot

Replace the Telegram API base URL with the mock server address. The token can be anything: the server automatically registers the bot on first use and **adds it to all existing chats**.

### Python (`python-telegram-bot`)

```python
from telegram.ext import ApplicationBuilder

app = (
    ApplicationBuilder()
    .token("YOUR_TOKEN")
    .base_url("http://localhost:8081/bot")
    .build()
)
```

### Python (`aiogram`)

```python
from aiogram import Bot
from aiogram.client.session.aiohttp import AiohttpSession

session = AiohttpSession()
session.api = "http://localhost:8081"
bot = Bot(token="YOUR_TOKEN", session=session)
```

### Go (`telebot`)

```go
bot, _ := tele.NewBot(tele.Settings{
    Token: "YOUR_TOKEN",
    URL:   "http://localhost:8081",
})
```

### Node.js (`telegraf`)

```javascript
const bot = new Telegraf('YOUR_TOKEN', {
  telegram: { apiRoot: 'http://localhost:8081' }
});
```

### `curl`

```bash
# Check connectivity
curl http://localhost:8081/botYOUR_TOKEN/getMe

# Send a message to a chat
curl -X POST http://localhost:8081/botYOUR_TOKEN/sendMessage \
  -H 'Content-Type: application/json' \
  -d '{"chat_id": -1001, "text": "Hello!"}'

# Long polling (wait for 10 seconds)
curl -X POST http://localhost:8081/botYOUR_TOKEN/getUpdates \
  -d '{"timeout": 10}'
```

---

## Typical Workflow

After startup, the server already contains test data from the config: three users (Alice, Bob, Charlie), two chats, and one bot. The bot is automatically added to every chat, and **in the first chat it gets admin rights**.

When a bot connects for the first time (the first call to any Bot API method), the server **immediately sends a message** from a random user in one of the chats. During the next **30 seconds**, the bot will also receive messages in all other chats, simulating real activity right after launch. If an LLM is enabled, those messages are AI-generated; otherwise, the server falls back to template-based greetings.

To avoid defining users manually, you can enable **LLM-powered seed generation** so the server creates realistic users, groups, and channels in the language you need.

### Seed Data Generation

**Via config** (on server startup):

```yaml
seed:
  generate:
    enabled: true
    users_count: 10
    groups_count: 3
    channels_count: 1
    locale: "ru"
```

**Via the Admin API** (at any time):

```bash
curl -X POST http://localhost:8082/api/seed/generate \
  -H 'Content-Type: application/json' \
  -d '{"users_count": 10, "groups_count": 3, "channels_count": 1, "locale": "ru"}'
```

The LLM generates users with realistic names and usernames, creates groups and channels with meaningful titles, and distributes members across chats. All registered bots are automatically added to every generated chat, and **the first chat grants them admin rights**. The response includes all created entities with their assigned IDs.

### Message Injection

Simulate a message sent "by a user" so all connected bots receive an update:

```bash
curl -X POST http://localhost:8082/api/chats/-1001/messages \
  -H 'Content-Type: application/json' \
  -d '{"user_id": 1001, "text": "Hi, bot!"}'
```

### Proactive Mode

The server can generate its own event stream, including messages, joins, leaves, photos, and stickers, with configurable frequency and **content style**:

```yaml
proactive:
  enabled: true
  interval_min: 10s
  interval_max: 60s
  style: "normal"        # style of generated messages
```

#### Message Styles

The `style` parameter defines the tone of generated content, which is handy when testing moderation bots:

| Style | Description | Example use case |
|---|---|---|
| `normal` | Regular conversational messages | General testing |
| `spam` | Crypto scams, fake giveaways, suspicious links | Anti-spam bot testing |
| `toxic` | Profanity, insults, hate speech | Anti-toxicity bot testing |
| `flood` | Repeated characters, emoji spam, ALL CAPS, nonsense sequences | Anti-flood bot testing |
| `mixed` | Random mix: 40% normal, 20% spam, 20% toxic, 20% flood (default) | End-to-end moderation testing |

#### Custom Prompt

If the built-in presets are not enough, `custom_prompt` lets you supply any instruction for the LLM. It takes priority over `style`:

```yaml
proactive:
  enabled: true
  interval_min: 5s
  interval_max: 30s
  custom_prompt: "Generate messages advertising online casinos and sports betting. Use typical tricks: easy-money promises, fake testimonials, and links like casino-xyz.com"
```

More `custom_prompt` examples:

- `"Generate Ukrainian-language messages discussing current news"` for multilingual testing
- `"Write very long messages, 500+ characters, with quotes and links"` for limit testing
- `"Alternate normal messages with phishing attempts: ask users to open a link or enter a password"` for anti-phishing bot testing

### File Downloads

The mock server fully emulates file handling through `getFile` plus the file download endpoint, generating placeholder content on the fly:

```bash
# 1. Get file_path from file_id
curl http://localhost:8081/botYOUR_TOKEN/getFile?file_id=AgACAgIAAxkBAAI...

# Response: {"ok":true,"result":{"file_id":"...","file_path":"photos/file_abc123.jpg"}}

# 2. Download the file by file_path
curl http://localhost:8081/file/botYOUR_TOKEN/photos/file_abc123.jpg -o photo.jpg
```

The placeholder type is chosen automatically from the `file_id` prefix:

| `file_id` prefix | Type | Format | Size |
|---|---|---|---|
| `AgAC...` | Photo | JPEG (gradient + shape) | 800x600 |
| `CAAC...` | Sticker | WebP (transparent background, emoji-like) | 512x512 |
| `BAADAgAD...` | Video | JPEG (preview frame) | 640x480 |
| `BQAC...` | Document | Stub PDF | - |
| `CQACAgIAAxkBAAI...` | Audio | Stub MP3 | - |
| `DQAC...` | Voice | Stub OGG | - |

Photos and stickers are real generated images with a unique pattern derived from the `file_path` hash, so each `file_id` produces a visually distinct result.

---

## Configuration

### LLM Providers

The server supports two API protocols: OpenAI-compatible (default) and Anthropic.

**Ollama (local):**

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

**LM Studio / vLLM / any OpenAI-compatible server:**

```yaml
llm:
  base_url: "http://localhost:1234/v1"
  model: "local-model"
```

**Without an LLM:**

```yaml
llm:
  enabled: false
```

When the LLM is disabled, the bot only receives updates created manually via the Admin API or pre-defined in the seed data.

### Full config (`config.yaml`)

```yaml
server:
  host: "0.0.0.0"
  port: 8081
  read_timeout: 60s
  write_timeout: 60s

llm:
  enabled: true
  api_type: "openai"              # "openai" or "anthropic"
  base_url: "http://localhost:11434/v1"
  api_key: ""
  model: "gpt-4o-mini"
  temperature: 0.8
  max_tokens: 512
  timeout: 30s
  response_delay_min: 500ms       # Simulates "typing..."
  response_delay_max: 3s

proactive:
  enabled: false
  interval_min: 10s
  interval_max: 60s
  style: "normal"                 # "normal", "spam", "toxic", "flood", "mixed"
  # custom_prompt: "..."          # Free-form instruction (takes priority over style)
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
  host: "127.0.0.1"               # localhost only
  port: 8082
```

### Environment Variables

These override values from `config.yaml`:

| Variable | Description |
|---|---|
| `TELEGRAM_MOCK_SERVER_HOST` | Bot API host (default: `0.0.0.0`) |
| `TELEGRAM_MOCK_SERVER_PORT` | Bot API port (default: `8081`) |
| `TELEGRAM_MOCK_LLM_ENABLED` | Enable LLM (`true`/`false`) |
| `TELEGRAM_MOCK_LLM_API_TYPE` | API protocol: `openai`, `anthropic` |
| `TELEGRAM_MOCK_LLM_BASE_URL` | LLM endpoint URL |
| `TELEGRAM_MOCK_LLM_API_KEY` | API key |
| `TELEGRAM_MOCK_LLM_MODEL` | Model name |
| `TELEGRAM_MOCK_PROACTIVE_ENABLED` | Enable proactive mode |
| `TELEGRAM_MOCK_SEED_GENERATE_ENABLED` | Enable automatic seed generation |
| `TELEGRAM_MOCK_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` |
| `TELEGRAM_MOCK_ADMIN_PORT` | Admin API port (default: `8082`) |

---

## Admin API

Manage the mock server state. By default, it is available at `127.0.0.1:8082`.

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/health` | Health check |
| `GET` | `/api/state` | Full state dump (users, chats, bots) |
| `GET` | `/api/users` | List users |
| `POST` | `/api/users` | Create a user |
| `GET` | `/api/chats` | List chats |
| `POST` | `/api/chats` | Create a chat |
| `GET` | `/api/chats/{id}/members` | List chat members |
| `POST` | `/api/chats/{id}/members` | Add a member |
| `GET` | `/api/chats/{id}/messages` | Message history |
| `POST` | `/api/chats/{id}/messages` | Inject a user message |
| `GET` | `/api/bots` | List bots |
| `POST` | `/api/bots/{token}/updates` | Inject an arbitrary Update |
| `POST` | `/api/seed/generate` | Generate seed data via LLM |

### Examples

```bash
# Create a user
curl -X POST http://localhost:8082/api/users \
  -d '{"first_name": "Diana", "username": "diana"}'

# Create a group
curl -X POST http://localhost:8082/api/chats \
  -d '{"type": "group", "title": "New Group", "members": [1001, 1002]}'

# Send a user message (bots will receive an update)
curl -X POST http://localhost:8082/api/chats/-1001/messages \
  -d '{"user_id": 1001, "text": "Hi!"}'

# Generate users and chats via LLM
curl -X POST http://localhost:8082/api/seed/generate \
  -d '{"users_count": 5, "groups_count": 2, "locale": "ru"}'

# Inject an arbitrary Update into a specific bot
curl -X POST http://localhost:8082/api/bots/YOUR_TOKEN/updates \
  -d '{"message":{"message_id":1,"from":{"id":1001,"first_name":"Alice"},"chat":{"id":-1001,"type":"group"},"text":"test"}}'
```

---

## Implemented Bot API Methods

28 methods covering the most common bot workflows:

| Category | Methods |
|---|---|
| Information | `getMe`, `getChat`, `getChatMember`, `getChatMemberCount`, `getChatAdministrators` |
| Updates | `getUpdates`, `setWebhook`, `deleteWebhook`, `getWebhookInfo` |
| Messages | `sendMessage`, `editMessageText`, `editMessageReplyMarkup`, `deleteMessage`, `forwardMessage`, `copyMessage`, `answerCallbackQuery` |
| Media | `sendPhoto`, `sendDocument`, `sendVideo`, `sendAudio`, `sendVoice`, `sendSticker`, `sendAnimation`, `sendLocation` |
| Files | `getFile` + download endpoint `/file/bot{token}/{path}` |
| Chat management | `banChatMember`, `unbanChatMember`, `restrictChatMember`, `promoteChatMember`, `leaveChat` |

---

## Architecture

```text
cmd/telegram-mock-ai/main.go     - entry point, wiring, graceful shutdown
internal/
|- api/          - Bot API and Admin API HTTP handlers
|- bot/          - Bot registry (auto-register by token)
|- state/        - In-memory store (users, chats, messages, members)
|- updates/      - Update queue and dispatcher (queue/webhook)
|- seed/         - LLM seed generation
|- llm/          - OpenAI/Anthropic client and prompts
|- webhook/      - Webhook delivery with retries
|- proactive/    - Proactive event generation engine
|- config/       - YAML and env configuration
`- models/       - Telegram API data structures
```

```text
Bot -> POST /bot{token}/sendMessage -> state -> [async LLM reply] -> queue/webhook -> Bot
Bot -> getFile(file_id) -> file_path -> GET /file/bot{token}/{path} -> placeholder JPEG/WebP/stub
Proactive engine -> timer -> scenario -> LLM (style/custom_prompt) -> update -> Bot
Admin API -> POST /api/seed/generate -> LLM -> users + chats in state
```

## Build

```bash
make build    # binary
make run      # build and run
make test     # tests
```

## License

Apache 2.0

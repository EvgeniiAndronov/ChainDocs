# 🖥 Сервер ChainDocs

**Версия:** 2.0.0  
**Статус:** Production Ready

---

## 📋 Содержание

1. [Роль сервера](#роль-сервера)
2. [Архитектура](#архитектура)
3. [Компоненты](#компоненты)
4. [API Reference](#api-reference)
5. [WebSocket Protocol](#websocket-protocol)
6. [Конфигурация](#конфигурация)
7. [Развёртывание](#развёртывание)

---

## 🎯 Роль сервера

Сервер в ChainDocs выполняет роль **центрального координатора** и **хранителя блокчейна**.

### Основные функции

1. **Хранение блокчейна**
   - BBolt key-value БД
   - Атомарные транзакции
   - Индексы по высоте и хэшу

2. **Координация консенсуса**
   - Подсчёт подписей
   - Динамический расчёт (51% от активных)
   - Уведомление о достижении консенсуса

3. **WebSocket Hub**
   - Real-time уведомления
   - Рассылка событий клиентам
   - Управление подключениями

4. **HTTP API**
   - RESTful endpoints
   - Аутентификация для Web UI
   - Rate limiting (опционально)

5. **Discovery**
   - Список зарегистрированных ключей
   - Активность клиентов (24h окно)
   - P2P адреса для клиентов

---

## 🏗 Архитектура

```
┌─────────────────────────────────────────────┐
│                 ChainDocs Server            │
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │           HTTP Router               │   │
│  │  (Chi middleware: CORS, Logger)     │   │
│  └──────────────┬──────────────────────┘   │
│                 │                           │
│  ┌──────────────┼──────────────────────┐   │
│  │              │                      │   │
│  ▼              ▼                      ▼   │
│ ┌────────┐ ┌────────┐           ┌────────┐│
│ │  REST  │ │  WS    │           │ Static ││
│ │  API   │ │  Hub   │           │  Files ││
│ └───┬────┘ └───┬────┘           └───┬────┘│
│     │          │                    │     │
│     │    ┌─────┴─────┐              │     │
│     │    │  Broadcast│              │     │
│     │    │   Queue   │              │     │
│     │    └─────┬─────┘              │     │
│     │          │                    │     │
│     ▼          ▼                    ▼     │
│ ┌──────────────────────────────────────┐  │
│ │         Business Logic Layer         │  │
│ │  • handleUpload                      │  │
│ │  • handleSignature                   │  │
│ │  • handleGetBlocks                   │  │
│ │  • handleGetPendingBlocks            │  │
│ └──────────────────────────────────────┘  │
│                 │                          │
│                 ▼                          │
│ ┌──────────────────────────────────────┐  │
│ │         Storage Layer (BBolt)        │  │
│ │  • blocks bucket                     │  │
│ │  • height bucket                     │  │
│ │  • pubkeys bucket                    │  │
│ │  • activity bucket                   │  │
│ │  • revoked bucket                    │  │
│ │  • categories bucket                 │  │
│ └──────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

---

## 🧩 Компоненты

### 1. HTTP Router

**Библиотека:** `go-chi/chi/v5`

**Middleware:**
```go
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(middleware.RealIP)
r.Use(middleware.RequestID)
r.Use(cors.Handler(cors.Options{...}))
r.Use(srv.authMiddleware) // для /web/*
```

**Routes:**
```go
// API
r.Get("/api/blocks", handleGetBlocks)
r.Get("/api/blocks/last", handleGetLastBlock)
r.Get("/api/blocks/{hash}", handleGetBlock)
r.Get("/api/blocks/pending", handleGetPendingBlocks)
r.Post("/api/upload", handleUpload)
r.Post("/api/upload/bulk", handleBulkUpload)
r.Post("/api/sign", handleSignature)
r.Post("/api/register", handleRegisterKey)
r.Get("/api/keys", handleGetKeys)
r.Get("/api/keys/active", handleGetActiveKeys)
r.Get("/api/keys/revoked", handleGetRevokedKeys)
r.Get("/api/categories", handleGetCategories)
r.Post("/api/categories", handleCreateCategory)

// WebSocket
r.Get("/ws/notifications", wsHub.Handler())

// Web UI
r.Get("/web/", handleWebDashboard)
r.Get("/web/blocks", handleWebBlocks)
r.Get("/web/upload", handleWebUpload)
r.Get("/web/categories", handleWebCategories)
r.Get("/web/keys", handleWebKeys)
```

---

### 2. WebSocket Hub

**Файл:** `internal/websocket/hub.go`

**Структура:**
```go
type Hub struct {
    mu        sync.RWMutex
    clients   map[string]*ClientInfo
    broadcast chan Message
    serverID  string
}

type ClientInfo struct {
    ID        string
    Addr      string
    PublicKey string
    Conn      *websocket.Conn
}
```

**Функции:**

1. **AddClient** — регистрация нового подключения
```go
func (h *Hub) AddClient(id string, conn *websocket.Conn, addr string, publicKey string)
```

2. **BroadcastBlock** — рассылка уведомления о новом блоке
```go
func (h *Hub) BroadcastBlock(b *block.Block)
```

3. **BroadcastConsensus** — рассылка обновления консенсуса
```go
func (h *Hub) BroadcastConsensus(blockHash string, signatures, required int, percent float64, reached bool)
```

4. **periodicPendingCheck** — проверка неподписанных блоков (каждые 2 мин)
```go
func (h *Hub) periodicPendingCheck()
```

**Типы сообщений:**
```go
type MessageType string

const (
    MsgBlockAnnounce   MessageType = "block_announce"
    MsgConsensusUpdate MessageType = "consensus_update"
    MsgPeerUpdate      MessageType = "peer_update"
)

type Message struct {
    Type      MessageType      `json:"type"`
    Block     *block.Block     `json:"block,omitempty"`
    Consensus *ConsensusStatus `json:"consensus,omitempty"`
    Peers     []PeerInfo       `json:"peers,omitempty"`
    Timestamp string           `json:"timestamp"`
}
```

---

### 3. Storage Layer

**Библиотека:** `go.etcd.io/bbolt`

**Buckets:**

| Bucket | Key | Value |
|--------|-----|-------|
| `blocks` | `block_hash[:32]` | `Block` (JSON) |
| `height` | `height (uint64)` | `block_hash[:32]` |
| `pubkeys` | `public_key (hex)` | `""` |
| `activity` | `public_key (hex)` | `KeyActivity` (JSON) |
| `revoked` | `public_key (hex)` | `RevocationInfo` (JSON) |
| `categories` | `category_id` | `Category` (JSON) |
| `documents` | `doc_hash[:32]` | `DocumentMetadata` (JSON) |

**Методы:**

```go
// Блоки
func (s *Storage) SaveBlock(b *block.Block) error
func (s *Storage) GetBlock(hash [32]byte) (*block.Block, error)
func (s *Storage) GetLastBlock() (*block.Block, error)
func (s *Storage) GetAllBlocks() ([]*block.Block, error)

// Ключи
func (s *Storage) SavePublicKey(publicKey string) error
func (s *Storage) GetAllPublicKeys() ([]string, error)
func (s *Storage) GetActiveKeys(window time.Duration) ([]KeyActivity, error)
func (s *Storage) UpdateKeyActivity(publicKey string) error

// Отозванные ключи
func (s *Storage) RevokePublicKey(publicKey, reason string, revokedAt time.Time) error
func (s *Storage) IsKeyRevoked(publicKey string) (bool, *RevocationInfo, error)

// Категории
func (s *Storage) CreateCategory(id, name, description string) error
func (s *Storage) GetAllCategories() ([]Category, error)
func (s *Storage) IncrementCategoryDocCount(categoryID string) error
```

---

## 📡 API Reference

### Blocks

#### GET /api/blocks

**Ответ:**
```json
[
  {
    "height": 0,
    "hash": "abc123...",
    "prev_hash": "def456...",
    "document_hash": "789ghi...",
    "signatures": [...],
    "timestamp": "2026-02-24T12:00:00Z"
  }
]
```

#### GET /api/blocks/last

**Ответ:** Последний блок

#### GET /api/blocks/{hash}

**Ответ:** Блок по хэшу

#### GET /api/blocks/pending

**Описание:** Блоки, ожидающие подписи

**Ответ:**
```json
{
  "count": 2,
  "pending_blocks": [
    {
      "hash": "abc123...",
      "height": 5,
      "signatures": 0,
      "required": 2,
      "percent": 0,
      "timestamp": "2026-02-24T12:00:00Z"
    }
  ]
}
```

#### GET /api/blocks/{hash}/consensus

**Ответ:**
```json
{
  "block_hash": "abc123...",
  "height": 5,
  "total_keys": 3,
  "active_keys": 3,
  "signatures": 2,
  "required": 2,
  "percent": 66.67,
  "consensus_reached": true,
  "signatures_list": [...]
}
```

---

### Upload

#### POST /api/upload

**Request:** `multipart/form-data`
- `file` — PDF файл
- `category` — опционально

**Ответ:**
```json
{
  "hash": "abc123...",
  "filename": "document.pdf",
  "size": 1024,
  "uploaded": "2026-02-24T12:00:00Z",
  "block_hash": "def456...",
  "category": "diplomas"
}
```

#### POST /api/upload/bulk

**Request:** `multipart/form-data`
- `files[]` — массив PDF файлов (до 50)
- `category` — опционально

**Ответ:**
```json
{
  "total": 3,
  "success": 3,
  "failed": 0,
  "category": "diplomas",
  "results": [
    {
      "filename": "doc1.pdf",
      "hash": "abc...",
      "block_hash": "def...",
      "size": 1024,
      "success": true
    }
  ]
}
```

---

### Signatures

#### POST /api/sign

**Request:**
```json
{
  "block_hash": "abc123...",
  "signature": "def456...",
  "public_key": "789ghi..."
}
```

**Ответ:**
```json
{
  "status": "signature saved",
  "signatures": 2,
  "required": 2,
  "percent": 66.67,
  "consensus": true,
  "active_keys": 3
}
```

---

### Keys

#### GET /api/keys

**Ответ:**
```json
{
  "count": 3,
  "keys": [
    "abc123...",
    "def456...",
    "789ghi..."
  ]
}
```

#### GET /api/keys/active

**Ответ:**
```json
{
  "count": 3,
  "window": "24h0m0s",
  "activities": [
    {
      "public_key": "abc123...",
      "last_seen": "2026-02-24T12:00:00Z",
      "block_count": 10
    }
  ]
}
```

#### GET /api/keys/revoked

**Ответ:**
```json
{
  "count": 1,
  "keys": [
    {
      "public_key": "xyz789...",
      "reason": "compromised",
      "revoked_at": "2026-02-24T12:00:00Z"
    }
  ]
}
```

---

### Categories

#### GET /api/categories

**Ответ:**
```json
{
  "count": 2,
  "categories": [
    {
      "id": "diplomas",
      "name": "Дипломы студентов",
      "description": "Дипломы выпускников",
      "created": "2026-02-24T12:00:00Z",
      "doc_count": 50
    }
  ]
}
```

#### POST /api/categories

**Request:**
```json
{
  "id": "contracts",
  "name": "Договоры",
  "description": "Учебные договоры"
}
```

**Ответ:**
```json
{
  "id": "contracts",
  "status": "created"
}
```

---

## 🔌 WebSocket Protocol

### Подключение

```
ws://localhost:8080/ws/notifications?public_key=abc123...
```

### Сообщения сервера

#### Block Announce
```json
{
  "type": "block_announce",
  "block": {
    "height": 5,
    "hash": "abc123...",
    "document_hash": "def456...",
    "signatures": [],
    "timestamp": "2026-02-24T12:00:00Z"
  },
  "block_hash": "abc123...",
  "timestamp": "2026-02-24T12:00:00Z"
}
```

#### Consensus Update
```json
{
  "type": "consensus_update",
  "block_hash": "abc123...",
  "consensus": {
    "signatures": 2,
    "required": 2,
    "percent": 66.67,
    "consensus_reached": false
  },
  "timestamp": "2026-02-24T12:00:00Z"
}
```

#### Peer Update
```json
{
  "type": "peer_update",
  "peers": [
    {
      "id": "abc123...",
      "address": "[::1]:9001",
      "connected": true
    }
  ],
  "timestamp": "2026-02-24T12:00:00Z"
}
```

---

## ⚙ Конфигурация

### config.json

```json
{
  "port": 8080,
  "db_path": "blockchain.db",
  "upload_dir": "./uploads",
  "log_file": "./logs/server.log",
  "log_level": "info",
  "consensus": {
    "type": "percentage",
    "percentage": 51,
    "min_signatures": 2,
    "max_signatures": 0,
    "use_active_keys": true
  },
  "activity": {
    "window": "24h",
    "auto_cleanup": true
  },
  "tls": {
    "enabled": false,
    "cert_file": "",
    "key_file": ""
  },
  "rate_limit": {
    "enabled": false,
    "requests_per_second": 10,
    "burst": 20
  }
}
```

### Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `CHAINDOCS_CONFIG` | Путь к config.json | `config.json` |
| `CHAINDOCS_AUTH_TOKEN` | Токен для Web UI | Случайный |
| `CHAINDOCS_DB` | Путь к БД | Из config.json |
| `CHAINDOCS_BACKUP_DIR` | Директория backup | `./backups` |

---

## 🚀 Развёртывание

### Локально

```bash
# Сборка
go build -o bin/server ./cmd/server/main.go ./cmd/server/config.go

# Запуск
./bin/server

# Или с конфигом
export CHAINDOCS_CONFIG=config.json
./bin/server
```

### Docker

```bash
# Сборка образа
docker build -t chaindocs-server:latest .

# Запуск
docker run -d \
  -p 8080:8080 \
  -v chaindocs-data:/app/data \
  -v chaindocs-uploads:/app/uploads \
  -v $(pwd)/config.json:/app/config.json \
  chaindocs-server:latest
```

### Docker Compose

```yaml
version: '3'
services:
  chaindocs-server:
    image: chaindocs-server:latest
    ports:
      - "8080:8080"
    volumes:
      - chaindocs-data:/app/data
      - chaindocs-uploads:/app/uploads
      - ./config.json:/app/config.json
    environment:
      - CHAINDOCS_AUTH_TOKEN=demo_token

volumes:
  chaindocs-data:
  chaindocs-uploads:
```

---

## 📊 Мониторинг

### Метрики Prometheus

**Endpoint:** `/metrics`

**Метрики:**
```prometheus
# Блоки
chaindocs_blocks_total

# Ключи
chaindocs_active_keys
chaindocs_registered_keys_total

# Консенсус
chaindocs_consensus_percent

# Производительность
chaindocs_requests_total
chaindocs_request_duration_seconds
```

### Логи

**Формат:** Text или JSON

**Уровни:**
- `DEBUG` — отладочная информация
- `INFO` — обычная работа
- `WARN` — предупреждения
- `ERROR` — ошибки

**Пример:**
```
2026/02/24 12:00:00 [INFO] 🚀 ChainDocs Server starting...
2026/02/24 12:00:00 [INFO] 📄 Config loaded: port=8080, db=blockchain.db
2026/02/24 12:00:00 [INFO] ✅ Genesis block created: 78cf5864
2026/02/24 12:00:01 [INFO] 🚀 Server starting on :8080
2026/02/24 12:00:05 [INFO] 📄 File uploaded: document.pdf (1024 bytes), hash: abc123...
2026/02/24 12:00:05 [INFO] 🔗 Block created: height=1, hash=def456...
2026/02/24 12:00:06 [INFO] 📢 Broadcasted block: height=1, hash=def456...
2026/02/24 12:00:06 [INFO] ✅ Signature saved for block 1 from abc123... [1/2 = 50.0%]
2026/02/24 12:00:06 [INFO] 🎉 CONSENSUS REACHED for block 1! (2/2 signatures)
```

---

## 🔧 Troubleshooting

### Сервер не запускается

**Проблема:** Порт 8080 занят

**Решение:**
```bash
lsof -i :8080
kill <PID>
```

### Ошибка БД

**Проблема:** Файл БД заблокирован

**Решение:**
```bash
# Остановить сервер
pkill -f chaindocs-server

# Проверить блокировки
lsof blockchain.db

# Удалить lock файл
rm blockchain.db-shm blockchain.db-wal
```

### WebSocket не подключается

**Проблема:** Firewall блокирует

**Решение:**
```bash
# Проверить firewall
sudo ufw status

# Разрешить порт 8080
sudo ufw allow 8080/tcp
```

---

**См. также:**
- [ARCHITECTURE.md](ARCHITECTURE.md) — общая архитектура
- [CLIENT.md](CLIENT.md) — клиент-подписант
- [P2P_PROTOCOL.md](P2P_PROTOCOL.md) — P2P протокол

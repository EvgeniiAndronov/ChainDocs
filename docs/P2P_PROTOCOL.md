# 🔀 P2P Протокол ChainDocs

**Версия:** 2.0.0

---

## Обзор

P2P (peer-to-peer) протокол используется для **прямой коммуникации между клиентами** без участия сервера.

### Назначение

1. **Gossip-протокол** — распространение подписей между клиентами
2. **Service Discovery** — обнаружение других клиентов в сети
3. **Health Check** — проверка доступности пиров (ping/pong)

### Архитектура

```
Клиент 1 (9001) ◄────► Клиент 2 (9002)
       ▲                   ▲
       │                   │
       └─────────┬─────────┘
                 │
                 ▼
         Клиент 3 (9003)
```

---

## Транспорт

### WebSocket

**Порт:** 9001-9003 (настраивается)

**URL:** `ws://{host}:{port}/p2p?public_key={public_key}`

**Пример:**
```
ws://localhost:9001/p2p?public_key=abc123...
```

### Формат сообщений

```json
{
  "type": "signature",
  "peer_id": "abc123...",
  "block_hash": "def456...",
  "signature": "789ghi...",
  "public_key": "xyz789...",
  "from_client": true,
  "timestamp": "2026-02-24T12:00:00Z"
}
```

---

## Типы сообщений

### 1. Ping/Pong

**Назначение:** Проверка доступности пира

**Ping:**
```json
{
  "type": "ping",
  "peer_id": "abc123...",
  "timestamp": "2026-02-24T12:00:00Z"
}
```

**Pong:**
```json
{
  "type": "pong",
  "peer_id": "abc123...",
  "timestamp": "2026-02-24T12:00:00Z"
}
```

**Интервал:** 30 секунд

---

### 2. Signature

**Назначение:** Рассылка подписей

**Сообщение:**
```json
{
  "type": "signature",
  "peer_id": "abc123...",
  "block_hash": "def456...",
  "signature": "789ghi...",
  "public_key": "xyz789...",
  "from_client": true,
  "timestamp": "2026-02-24T12:00:00Z"
}
```

**Обработка:**
1. Проверка валидности подписи
2. Сохранение в локальное хранилище
3. Пересылка другим пирам (gossip)

---

### 3. Block Announce

**Назначение:** Уведомление о новом блоке

**Сообщение:**
```json
{
  "type": "block_announce",
  "peer_id": "abc123...",
  "block": {
    "height": 5,
    "hash": "def456...",
    "document_hash": "789ghi...",
    "signatures": [],
    "timestamp": "2026-02-24T12:00:00Z"
  },
  "timestamp": "2026-02-24T12:00:00Z"
}
```

---

### 4. Peer List

**Назначение:** Обмен списками пиров

**Сообщение:**
```json
{
  "type": "peer_list",
  "peer_id": "abc123...",
  "peers": [
    {
      "id": "def456...",
      "address": "localhost:9002",
      "connected": true
    }
  ],
  "timestamp": "2026-02-24T12:00:00Z"
}
```

---

## Обнаружение пиров

### 1. Через сервер

```go
// GET /api/peers
// Ответ:
{
  "peers": [
    {"id": "abc123...", "address": "localhost:9001", "connected": true}
  ],
  "count": 1
}
```

### 2. Через P2P

При подключении пиры обмениваются списками известных им пиров.

---

## Поддержание соединений

### Check Connections

Каждые 30 секунд:
```go
for addr, peer := range peers {
    if time.Since(peer.LastSeen) > 2*time.Minute {
        // Peer offline
        peer.Connected = false
    }
}
```

### Reconnect

При обрыве соединения:
```go
go n.connectToPeer(addr)
```

---

## Безопасность

### Аутентификация

1. Public key передаётся в URL при подключении
2. Проверка подписи в сообщениях

### Rate Limiting

Опционально может быть включён для защиты от flood.

---

**См. также:**
- [CLIENT.md](CLIENT.md) — клиент
- [SERVER.md](SERVER.md) — сервер

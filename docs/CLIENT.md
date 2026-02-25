# 🔑 Клиент ChainDocs

**Версия:** 2.0.0  
**Статус:** Production Ready

---

## 📋 Содержание

1. [Роль клиента](#роль-клиента)
2. [Архитектура](#архитектура)
3. [Режимы работы](#режимы-работы)
4. [P2P коммуникация](#p2p-коммуникация)
5. [WebSocket интеграция](#websocket-интеграция)
6. [Конфигурация](#конфигурация)
7. [Развёртывание](#развёртывание)

---

## 🎯 Роль клиента

Клиент в ChainDocs выполняет роль **независимого подписанта** блоков.

### Основные функции

1. **Подписание блоков**
   - Загрузка приватного ключа (AES-256-GCM)
   - Подписание хэша блока (Ed25519)
   - Отправка подписи на сервер

2. **Мониторинг блоков**
   - WebSocket real-time уведомления
   - HTTP polling (резервный режим)
   - Проверка pending блоков при подключении

3. **P2P коммуникация**
   - Подключение к другим клиентам
   - Gossip-протокол для подписей
   - Ping/Pong для поддержания соединений

4. **Self-healing**
   - Детекция чужих подписей
   - Webhook уведомления
   - Auto-revoke (опционально)

---

## 🏗 Архитектура

```
┌─────────────────────────────────────────────┐
│              ChainDocs Client               │
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │          Main Loop (daemon)         │   │
│  │  • Check interval: 5s               │   │
│  │  • Graceful shutdown                │   │
│  └──────────────┬──────────────────────┘   │
│                 │                           │
│  ┌──────────────┼──────────────────────┐   │
│  │              │                      │   │
│  ▼              ▼                      ▼   │
│ ┌────────┐ ┌────────┐           ┌────────┐│
│ │  WS    │ │  P2P   │           │ Config ││
│ │Client  │ │ Node   │           │ Loader ││
│ └───┬────┘ └───┬────┘           └───┬────┘│
│     │          │                    │     │
│     │    ┌─────┴─────┐              │     │
│     │    │  Message  │              │     │
│     │    │ Handlers  │              │     │
│     │    └─────┬─────┘              │     │
│     │          │                    │     │
│     ▼          ▼                    ▼     │
│ ┌──────────────────────────────────────┐  │
│ │         Business Logic Layer         │  │
│ │  • processBlock                      │  │
│ │  • checkPendingBlocks                │  │
│ │  • sendSignature                     │  │
│ │  • broadcastSignature                │  │
│ └──────────────────────────────────────┘  │
│                 │                          │
│                 ▼                          │
│ ┌──────────────────────────────────────┐  │
│ │         Crypto Layer                 │  │
│ │  • LoadPrivateKey (AES-256-GCM)      │  │
│ │  • Sign (Ed25519)                    │  │
│ │  • Verify                            │  │
│ └──────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

---

## 🔀 Режимы работы

### 1. Daemon (постоянный)

**Описание:** Клиент работает в фоновом режиме, периодически проверяя новые блоки.

**Конфигурация:**
```json
{
  "mode": "daemon",
  "daemon": {
    "interval": "5s",
    "sign_unsigned_only": false,
    "stop_on_consensus": false
  }
}
```

**Поток:**
```
1. Подключение к WebSocket
         │
         ▼
2. Проверка pending блоков
         │
         ▼
3. Цикл каждые 5 сек:
   ├─ Если WebSocket активен → пропуск polling
   ├─ Если WebSocket отключён → HTTP polling
   └─ Проверка последнего блока
         │
         ▼
4. Обработка сигналов (SIGINT/SIGTERM)
         │
         ▼
5. Graceful shutdown
```

**Использование:**
```bash
export CHAINDOCS_CLIENT1_PASSWORD="mypassword"
./bin/client -config client1-config.json
```

---

### 2. Oneshot (однократный)

**Описание:** Клиент проверяет и подписывает один блок, затем завершает работу.

**Конфигурация:**
```json
{
  "mode": "oneshot"
}
```

**Поток:**
```
1. Загрузка ключа
         │
         ▼
2. Получение последнего блока
         │
         ▼
3. Проверка подписи
         │
         ▼
4. Подписание (если нужно)
         │
         ▼
5. Отправка подписи
         │
         ▼
6. Выход
```

**Использование:**
```bash
export CHAINDOCS_KEY_PASSWORD="mypassword"
./bin/client -config client1-config.json -mode oneshot
```

---

## 🔌 P2P Коммуникация

### Архитектура P2P

**Файл:** `internal/p2p/node.go`

**Структура:**
```go
type P2PNode struct {
    mu            sync.RWMutex
    peerID        string
    publicKey     string
    peers         map[string]*PeerInfo
    inboundConns  map[string]*websocket.Conn
    outboundConns map[string]*websocket.Conn
    serverURL     string
    listenAddr    string
    listener      net.Listener
    onBlock       func(*block.Block)
    onSignature   func(string, []byte, string)
}
```

### Подключение к пирам

**1. Получение списка пиров от сервера:**
```go
func (n *P2PNode) connectToServer() error {
    peers, err := n.fetchPeersFromServer()
    // GET /api/peers
    // Returns: [{id, address, connected}, ...]
    
    for _, peer := range peers {
        go n.connectToPeer(peer.Address)
    }
}
```

**2. Подключение к пиру:**
```go
func (n *P2PNode) connectToPeer(addr string) error {
    // WebSocket подключение
    wsURL := fmt.Sprintf("ws://%s/p2p?public_key=%s", addr, n.publicKey)
    conn, _, err := websocket.Dial(ctx, wsURL, nil)
    
    // Сохранение подключения
    n.outboundConns[addr] = conn
    
    // Запуск обработчика сообщений
    go n.handleP2PMessages(addr, conn)
    
    // Отправка приветствия
    n.sendHello(conn)
}
```

### Типы P2P сообщений

```go
type MessageType string

const (
    MsgBlockAnnounce  MessageType = "block_announce"
    MsgBlockRequest   MessageType = "block_request"
    MsgBlockResponse  MessageType = "block_response"
    MsgPeerList       MessageType = "peer_list"
    MsgSignature      MessageType = "signature"
    MsgConsensusState MessageType = "consensus_state"
    MsgPing           MessageType = "ping"
    MsgPong           MessageType = "pong"
)

type Message struct {
    Type       MessageType      `json:"type"`
    PeerID     string           `json:"peer_id"`
    Block      *block.Block     `json:"block,omitempty"`
    BlockHash  string           `json:"block_hash,omitempty"`
    Signature  []byte           `json:"signature,omitempty"`
    PublicKey  string           `json:"public_key,omitempty"`
    Peers      []PeerInfo       `json:"peers,omitempty"`
    Consensus  *ConsensusState  `json:"consensus,omitempty"`
    Timestamp  string           `json:"timestamp"`
    FromClient bool             `json:"from_client,omitempty"`
}
```

### Обработка сообщений

#### Block Announce
```go
func (n *P2PNode) handleBlockAnnounce(msg *Message) {
    log.Printf("📦 Block announced: %s", msg.BlockHash)
    
    if msg.Block != nil && n.onBlock != nil {
        go n.onBlock(msg.Block)
    }
}
```

#### Signature
```go
func (n *P2PNode) handleSignature(msg *Message) {
    log.Printf("✍️  Received signature from %s for block %s", 
        msg.PublicKey[:16], msg.BlockHash)
    
    if n.onSignature != nil && msg.FromClient {
        go n.onSignature(msg.BlockHash, msg.Signature, msg.PublicKey)
    }
}
```

#### Ping/Pong
```go
func (n *P2PNode) handlePing(fromPeer string) {
    // Отправка Pong
    msg := Message{
        Type:      MsgPong,
        PeerID:    n.peerID,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
    n.sendToPeer(fromPeer, msg)
}
```

### Maintenance

**Периодические задачи:**
```go
func (n *P2PNode) maintenanceLoop() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-n.ctx.Done():
            return
        case <-ticker.C:
            n.checkConnections()
            n.sendPing()
        }
    }
}
```

**Проверка соединений:**
```go
func (n *P2PNode) checkConnections() {
    n.mu.RLock()
    defer n.mu.RUnlock()
    
    for addr, peer := range n.peers {
        if time.Since(peer.LastSeen) > 2*time.Minute {
            log.Printf("⚠️  Peer %s seems offline", addr[:16])
            peer.Connected = false
        }
    }
}
```

---

## 🌐 WebSocket интеграция

### Подключение

```go
func (c *Client) connectWebSocket() {
    wsURL := strings.Replace(c.config.Server, "http://", "ws://", 1)
    wsURL = fmt.Sprintf("%s/ws/notifications?public_key=%s", wsURL, c.publicKeyHex)
    
    conn, _, err := websocket.Dial(ctx, wsURL, nil)
    
    c.wsConn = conn
    c.useWebSocket = true
    
    // Запуск обработчика
    go c.handleWebSocketMessages()
    
    // Проверка pending блоков
    go c.checkPendingBlocks()
}
```

### Обработка сообщений

```go
func (c *Client) handleWebSocketMessages() {
    for {
        _, msg, err := c.wsConn.Read(context.Background())
        if err != nil {
            c.logger.Printf("⚠️  WebSocket read error: %v", err)
            c.useWebSocket = false
            return
        }
        
        var wsMsg struct {
            Type      string          `json:"type"`
            Block     *block.Block    `json:"block,omitempty"`
            BlockHash string          `json:"block_hash,omitempty"`
            Consensus *struct {
                Signatures       int     `json:"signatures"`
                Required         int     `json:"required"`
                Percent          float64 `json:"percent"`
                ConsensusReached bool    `json:"consensus_reached"`
            } `json:"consensus,omitempty"`
        }
        
        json.Unmarshal(msg, &wsMsg)
        
        switch wsMsg.Type {
        case "block_announce":
            c.processBlock(wsMsg.Block)
        case "consensus_update":
            c.logger.Printf("📊 Consensus: %d/%d (%.1f%%)",
                wsMsg.Consensus.Signatures,
                wsMsg.Consensus.Required,
                wsMsg.Consensus.Percent)
        }
    }
}
```

### Проверка pending блоков

```go
func (c *Client) checkPendingBlocks() {
    time.Sleep(2 * time.Second) // Дать WebSocket подключиться
    
    resp, err := c.httpClient.Get(c.config.Server + "/api/blocks/pending")
    
    var result struct {
        Count         int `json:"count"`
        PendingBlocks []struct {
            Hash       string  `json:"hash"`
            Height     int64   `json:"height"`
            Signatures int     `json:"signatures"`
            Required   int     `json:"required"`
        } `json:"pending_blocks"`
    }
    
    if result.Count > 0 {
        c.logger.Printf("📋 Found %d pending blocks", result.Count)
        
        // Проверка последнего блока
        lastBlock, _ := c.getLastBlock()
        if !lastBlock.IsSignedBy(c.keyPair.PublicKey) {
            c.processBlock(lastBlock)
        }
    }
}
```

---

## ⚙ Конфигурация

### client-config.json

```json
{
  "server": "http://localhost:8080",
  "key_file": "client1.enc",
  "password_env": "CHAINDOCS_CLIENT1_PASSWORD",
  "mode": "daemon",
  "daemon": {
    "interval": "5s",
    "max_blocks_per_cycle": 0,
    "sign_unsigned_only": false,
    "stop_on_consensus": false
  },
  "p2p": {
    "enabled": true,
    "listen_port": 9001
  },
  "logging": {
    "level": "info",
    "file": "./logs/client1.log",
    "format": "text"
  },
  "self_healing": {
    "enabled": true,
    "alert_on_foreign_signature": true,
    "alert_webhook": "",
    "auto_revoke": false
  }
}
```

### Поля конфигурации

| Поле | Тип | Описание | По умолчанию |
|------|-----|----------|--------------|
| `server` | string | URL сервера | `http://localhost:8080` |
| `key_file` | string | Путь к ключу | `client.enc` |
| `password_env` | string | Переменная с паролем | `CHAINDOCS_KEY_PASSWORD` |
| `mode` | string | Режим (daemon/oneshot) | `daemon` |
| `daemon.interval` | duration | Интервал проверки | `5s` |
| `p2p.enabled` | bool | Включить P2P | `true` |
| `p2p.listen_port` | int | P2P порт | `9001` |
| `self_healing.enabled` | bool | Self-healing | `true` |

---

## 🚀 Развёртывание

### Локально

```bash
# Генерация ключа
./bin/keygen -password "mypassword" -out client1.enc

# Регистрация ключа на сервере
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"public_key":"abc123..."}'

# Запуск клиента
export CHAINDOCS_CLIENT1_PASSWORD="mypassword"
./bin/client -config client1-config.json
```

### Как демон (Linux systemd)

**/etc/systemd/system/chaindocs-client.service:**
```ini
[Unit]
Description=ChainDocs Client
After=network.target

[Service]
Type=simple
User=chaindocs
WorkingDirectory=/opt/chaindocs
Environment="CHAINDOCS_CLIENT1_PASSWORD=mypassword"
ExecStart=/opt/chaindocs/client -config client1-config.json
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

**Установка:**
```bash
sudo systemctl daemon-reload
sudo systemctl enable chaindocs-client
sudo systemctl start chaindocs-client
sudo systemctl status chaindocs-client
```

### Как демон (macOS launchd)

**/Library/LaunchDaemons/com.chaindocs.client.plist:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.chaindocs.client</string>
    <key>ProgramArguments</key>
    <array>
        <string>/opt/chaindocs/client</string>
        <string>-config</string>
        <string>/opt/chaindocs/client1-config.json</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>CHAINDOCS_CLIENT1_PASSWORD</key>
        <string>mypassword</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

**Установка:**
```bash
sudo launchctl load /Library/LaunchDaemons/com.chaindocs.client.plist
sudo launchctl start com.chaindocs.client
```

---

## 📊 Мониторинг

### Логи

**Формат:** Text

**Уровни:**
- `INFO` — обычная работа
- `WARN` — предупреждения
- `ERROR` — ошибки

**Пример:**
```
2026/02/24 12:00:00 📄 Config loaded from client1-config.json
2026/02/24 12:00:00 🔑 Loading private key...
2026/02/24 12:00:00 ✅ Key loaded. Public key: abc123...
2026/02/24 12:00:00 ✅ WebSocket connected (real-time mode)
2026/02/24 12:00:00 🔄 Running in daemon mode (hybrid: WebSocket + P2P)
2026/02/24 12:00:00 🌐 P2P Node starting on localhost:9001
2026/02/24 12:00:01 ✅ P2P node started on localhost:9001
2026/02/24 12:00:02 📋 Checking for pending blocks...
2026/02/24 12:00:02 🔍 Last block: height=5, hash=abc123..., signatures=0
2026/02/24 12:00:02 ✍️  Block 5 not signed by us, signing...
2026/02/24 12:00:02 📦 Processing block: height=5, hash=abc123..., signatures=0
2026/02/24 12:00:02 ✍️  Signing block...
2026/02/24 12:00:02 ✅ Signature created: def456...
2026/02/24 12:00:02 📢 Signature broadcasted via P2P
2026/02/24 12:00:03 📩 [WS] Raw message received: {"type":"consensus_update",...}
2026/02/24 12:00:03 📊 [WS] Consensus update: 2/2 (66.7%)
```

### Метрики

**P2P подключения:**
```bash
grep "Connected to peer" client1.log | wc -l
```

**Подписанные блоки:**
```bash
grep "Signature created" client1.log | wc -l
```

**Ошибки:**
```bash
grep "ERROR\|panic" client1.log
```

---

## 🔧 Troubleshooting

### Клиент не подключается к WebSocket

**Проблема:** Сервер недоступен

**Решение:**
```bash
# Проверить доступность сервера
curl http://localhost:8080/api/blocks/last

# Проверить логи клиента
tail -f logs/client1.log | grep WebSocket
```

### P2P подключения не работают

**Проблема:** Порты заблокированы firewall

**Решение:**
```bash
# Проверить firewall
sudo ufw status

# Разрешить порты 9001-9003
sudo ufw allow 9001-9003/tcp
```

### Клиент падает с паникой

**Проблема:** Slice bounds out of range

**Решение:**
```bash
# Проверить версию клиента
./bin/client --version

# Обновить до последней версии
go build -o bin/client ./cmd/client/main.go ./cmd/client/config.go
```

---

**См. также:**
- [ARCHITECTURE.md](ARCHITECTURE.md) — общая архитектура
- [SERVER.md](SERVER.md) — сервер
- [P2P_PROTOCOL.md](P2P_PROTOCOL.md) — P2P протокол

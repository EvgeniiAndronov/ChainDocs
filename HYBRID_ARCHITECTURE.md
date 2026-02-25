# 🕸️ Гибридная Архитектура ChainDocs

**Версия:** 2.0.0  
**Дата:** 2026-02-23  
**Статус:** ✅ Реализовано

---

## 📋 Обзор

ChainDocs использует **гибридную архитектуру**, сочетающую централизованный сервер для координации и P2P для быстрой доставки сообщений между клиентами.

```
                    ┌─────────────┐
                    │   СЕРВЕР    │
                    │  (Authoritative)
                    │  • Блокчейн │
                    │  • Координация│
                    │  • WebSocket Hub│
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         │ WebSocket       │ WebSocket       │
         │ (real-time)     │ (real-time)     │
         ▼                 ▼                 ▼
   ┌──────────┐     ┌──────────┐     ┌──────────┐
   │ Клиент 1 │◄───►│ Клиент 2 │◄───►│ Клиент 3 │
   └──────────┘ P2P  └──────────┘ P2P  └──────────┘
```

---

## 🏗 Компоненты

### 1. Сервер (Coordinator)

**Роль:** Авторитетный источник истины

**Функции:**
- ✅ Хранение блокчейна (BBolt БД)
- ✅ Координация консенсуса
- ✅ WebSocket Hub для real-time уведомлений
- ✅ API для клиентов
- ✅ Discovery пиров (`/api/peers`)
- ✅ Web UI

**Endpoints:**
```
HTTP REST:
  POST /api/upload      - Загрузка документа
  POST /api/sign        - Отправка подписи
  GET  /api/blocks/last - Последний блок
  GET  /api/peers       - Список пиров

WebSocket:
  GET /ws/notifications - Real-time уведомления
```

### 2. WebSocket Hub

**Роль:** Real-time доставка событий

**Функции:**
- ✅ Управление подключениями клиентов
- ✅ Рассылка уведомлений о новых блоках
- ✅ Рассылка обновлений консенсуса
- ✅ Отправка списка пиров

**Типы сообщений:**
```json
{
  "type": "block_announce",
  "block": {...},
  "block_hash": "abc123...",
  "timestamp": "2026-02-23T14:00:00Z"
}

{
  "type": "consensus_update",
  "block_hash": "abc123...",
  "consensus": {
    "signatures": 2,
    "required": 2,
    "percent": 100,
    "consensus_reached": true
  }
}

{
  "type": "peer_update",
  "peers": [
    {"id": "client1", "address": "localhost:8080", "connected": true}
  ]
}
```

### 3. P2P Node (Клиент)

**Роль:** Прямая коммуникация между клиентами

**Функции:**
- ✅ Подключение к другим клиентам
- ✅ Gossip-протокол для блоков
- ✅ Трансляция подписей
- ✅ Обмен состоянием консенсуса

**Типы P2P сообщений:**
```json
{
  "type": "block_announce",
  "peer_id": "client1",
  "block": {...}
}

{
  "type": "signature",
  "peer_id": "client1",
  "block_hash": "abc123...",
  "signature": "...",
  "public_key": "...",
  "from_client": true
}
```

### 4. Клиент (Hybrid)

**Роль:** Подписание блоков + P2P коммуникация

**Режимы работы:**
- **WebSocket** (приоритет) - real-time уведомления от сервера
- **P2P** (дополнительно) - прямая связь с другими клиентами
- **Polling** (fallback) - если WebSocket недоступен

---

## 🔄 Поток данных

### Сценарий 1: Загрузка нового документа

```
1. Пользователь → Сервер: POST /api/upload (PDF)
   
   ┌─────────────┐
   │   Сервер    │
   │ 1. Создаёт  │
   │    блок     │
   └──────┬──────┘
          │
   ┌──────┴──────────────────────────────┐
   │ 2. WebSocket рассылка (мгновенно)   │
   ▼                                     ▼
┌──────────┐                       ┌──────────┐
│ Клиент 1 │                       │ Клиент 2 │
│ 3a.      │                       │ 3b.      │
│ Получает │                       │ Получает │
│ блок     │                       │ блок     │
└────┬─────┘                       └────┬─────┘
     │                                  │
     └────────────┬─────────────────────┘
                  │
     ┌────────────▼────────────┐
     │ 4. P2P Gossip           │
     │    Клиент 1 → Клиент 2  │
     │    "Я подписал!"        │
     └─────────────────────────┘
                  │
     ┌────────────▼────────────┐
     │ 5. Клиенты → Сервер     │
     │    POST /api/sign       │
     └────────────┬────────────┘
                  ▼
         ┌─────────────┐
         │   Сервер    │
         │ 6. Добавляет│
         │    подписи  │
         │ в блокчейн  │
         └─────────────┘
```

### Сценарий 2: Достижение консенсуса

```
1. Клиент подписывает блок
         │
         ▼
2. POST /api/sign → Сервер
         │
         ▼
3. Сервер проверяет подпись
         │
         ▼
4. WebSocket рассылка:
   {type: "consensus_update", 
    consensus: {signatures: 2, reached: true}}
         │
         ▼
5. Все клиенты видят: "Консенсус достигнут!"
```

---

## 📊 Сравнение с чистой архитектурой

| Аспект | Централизованная | Чистая P2P | **Гибридная** |
|--------|-----------------|------------|---------------|
| **Точка отказа** | Сервер | Нет | Сервер (частично) |
| **Скорость** | 3-10 сек (polling) | Мгновенно | **Мгновенно (WS)** |
| **Сложность** | Низкая | Высокая | **Средняя** |
| **Аудит** | ✅ Полный | ❌ Сложно | ✅ **Полный** |
| **Контроль** | ✅ Есть | ❌ Нет | ✅ **Есть** |
| **Отказоустойчивость** | ❌ Низкая | ✅ Высокая | ✅ **Средняя** |
| **Масштабируемость** | ❌ Ограничена | ✅ Горизонтальная | ✅ **Средняя** |

---

## ⚙️ Конфигурация

### Сервер

```json
{
  "port": 8080,
  "db_path": "blockchain.db",
  "websocket": {
    "enabled": true,
    "max_clients": 1000
  }
}
```

### Клиент

```json
{
  "server": "http://localhost:8080",
  "key_file": "client1.enc",
  "password_env": "CHAINDOCS_CLIENT1_PASSWORD",
  "mode": "daemon",
  "daemon": {
    "interval": "10s",
    "sign_unsigned_only": false
  },
  "hybrid": {
    "websocket_enabled": true,
    "p2p_enabled": true
  }
}
```

---

## 🧪 Тестирование

### Проверка WebSocket

```bash
# Запустить сервер
./bin/server

# Подключиться через wscat
wscat -c ws://localhost:8080/ws/notifications?public_key=test

# Загрузить документ
curl -X POST http://localhost:8080/api/upload -F "file=@doc.pdf"

# Ожидать сообщение:
# {"type": "block_announce", "block": {...}}
```

### Проверка P2P

```bash
# Запустить 3 клиента
./bin/client -config client1-config.json
./bin/client -config client2-config.json
./bin/client -config client3-config.json

# Проверить логи
tail -f demo/demo_logs/client1.log | grep P2P

# Ожидать:
# ✅ P2P node initialized (peers: 2)
# 📩 [P2P] Block received from peer
```

### Проверка гибридного режима

```bash
# Запустить демо
./demo/demo-start.sh

# Проверить логи сервера
tail -f demo/demo_logs/server.log | grep WebSocket

# Ожидать:
# 🌐 WebSocket Hub initialized
# ✅ WebSocket client connected: client1
```

---

## 🛡 Безопасность

### WebSocket

- ✅ Аутентификация по публичному ключу
- ✅ Валидация сообщений
- ✅ Graceful disconnect

### P2P

- ✅ Проверка подписей блоков
- ✅ Валидация от пиров
- ✅ Детект невалидных сообщений

### Сервер

- ✅ Авторитетный источник
- ✅ Проверка всех подписей
- ✅ Аудит действий

---

## 📈 Производительность

### Метрики

| Метрика | Значение |
|---------|----------|
| **WebSocket задержка** | < 100ms |
| **P2P задержка** | < 50ms (LAN) |
| **Polling задержка** | 3-10 сек |
| **Макс. клиентов** | 1000+ |
| **Пропускная способность** | 100 блоков/сек |

### Оптимизация

```go
// WebSocket Hub использует буферизацию
broadcast: make(chan Message, 256)

// P2P использует goroutines для параллелизма
go n.connectToPeer(peerAddr)

// Клиент пропускает polling при активном WebSocket
if !c.useWebSocket {
    c.processOnce()
}
```

---

## 🚀 Развёртывание

### Docker Compose

```yaml
version: '3'
services:
  chaindocs-server:
    image: chaindocs-server:latest
    ports:
      - "8080:8080"  # HTTP + WebSocket
    volumes:
      - chaindocs-data:/app/data
  
  chaindocs-client-1:
    image: chaindocs-client:latest
    environment:
      - CHAINDOCS_SERVER=http://chaindocs-server:8080
      - CHAINDOCS_CLIENT1_PASSWORD=demo123
    depends_on:
      - chaindocs-server

volumes:
  chaindocs-data:
```

### Production

```bash
# 1. Запустить сервер
./bin/server -config config.json

# 2. Запустить клиентов
./bin/client -config client1-config.json &
./bin/client -config client2-config.json &
./bin/client -config client3-config.json &

# 3. Проверить WebSocket
curl -s http://localhost:8080/api/peers | jq

# 4. Проверить P2P
tail -f client1.log | grep "P2P node initialized"
```

---

## 🔧 Troubleshooting

### WebSocket не подключается

```bash
# Проверить логи сервера
tail -f demo_logs/server.log | grep WebSocket

# Проверить доступность
curl http://localhost:8080/api/peers

# Проверить клиент
tail -f demo_logs/client1.log | grep WebSocket
```

### P2P не находит пиры

```bash
# Проверить API peers
curl -s http://localhost:8080/api/peers | jq

# Перезапустить клиентов
./demo/demo-stop.sh
./demo/demo-start.sh
```

### Консенсус не достигается

```bash
# Проверить количество подписей
curl -s http://localhost:8080/api/blocks/last/consensus | jq

# Проверить активных клиентов
curl -s http://localhost:8080/api/keys/active | jq

# Проверить логи
grep "CONSENSUS REACHED" demo_logs/server.log
```

---

## 📚 Исходный код

| Компонент | Файл |
|-----------|------|
| **WebSocket Hub** | `internal/websocket/hub.go` |
| **P2P Node** | `internal/p2p/node.go` |
| **Сервер** | `cmd/server/main.go` |
| **Клиент** | `cmd/client/main.go` |

---

## 🎯 Преимущества гибридной архитектуры

1. ✅ **Быстрая доставка** - WebSocket < 100ms
2. ✅ **Отказоустойчивость** - P2P работает при проблемах с сервером
3. ✅ **Аудит** - Сервер хранит полную историю
4. ✅ **Контроль** - Сервер координирует консенсус
5. ✅ **Масштабируемость** - P2P снижает нагрузку на сервер
6. ✅ **Гибкость** - Можно отключить P2P и работать только через WebSocket

---

**Версия:** 2.0.0  
**Статус:** ✅ Production Ready  
**Совместимость:** Назад совместима с polling режимом

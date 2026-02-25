# 🔄 Изменения в ветке `feature/consensus-daemon-healing`

## 📋 Обзор изменений

Эта ветка добавляет три основные функции:

1. **Консенсус (51%)** — блоки подписываются несколькими клиентами
2. **Клиент-демон** — легко развёртываемый сервис с конфигурацией
3. **Self-Healing** — система обнаружения и отзыва скомпрометированных ключей

---

## 1️⃣ КОНСЕНСУС (51%+)

### Что изменилось

**Структура блока:**
```go
// Было
Signature []byte `json:"signature"`

// Стало
type Signature struct {
    PublicKey string `json:"public_key"`
    Signature string `json:"signature"`
    Timestamp string `json:"timestamp"`
}

Signatures []Signature `json:"signatures"`
```

### Консенсус 51%

Блок считается подтверждённым, если подписан **51%+ зарегистрированных ключей**:

```
Зарегистрировано ключей: 5
Нужно подписей для консенсуса: 3 (60%)

Зарегистрировано ключей: 10
Нужно подписей для консенсуса: 6 (60%)
```

### Новые API эндпоинты

#### `GET /api/blocks/{hash}/consensus`

Статус консенсуса для блока:

```bash
curl http://localhost:8080/api/blocks/<hash>/consensus
```

**Ответ:**
```json
{
  "block_hash": "abc123...",
  "height": 5,
  "total_keys": 5,
  "signatures": 3,
  "required": 3,
  "percent": 60.0,
  "consensus_reached": true,
  "signatures_list": [
    {
      "public_key": "key1...",
      "signature": "sig1...",
      "timestamp": "2026-02-22T12:00:00Z"
    }
  ]
}
```

#### `GET /api/keys`

Список зарегистрированных ключей:

```bash
curl http://localhost:8080/api/keys
```

### Обновлённые эндпоинты

#### `POST /api/sign`

Теперь возвращает информацию о консенсусе:

```json
{
  "status": "signature saved",
  "signatures": 3,
  "required": 3,
  "percent": 60.0,
  "consensus": true
}
```

---

## 2️⃣ КЛИЕНТ-ДЕМОН

### Конфигурация

Клиент теперь использует JSON-конфиг:

```bash
# Сгенерировать пример конфига
./bin/client -gen-config
```

**Пример `config.json`:**
```json
{
  "server": "http://localhost:8080",
  "key_file": "key.enc",
  "password_env": "CHAINDOCS_KEY_PASSWORD",
  "mode": "daemon",
  "daemon": {
    "interval": "10s",
    "max_blocks_per_cycle": 0,
    "sign_unsigned_only": true,
    "stop_on_consensus": true
  },
  "logging": {
    "level": "info",
    "file": "",
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

### Запуск клиента

```bash
# oneshot режим (один блок и выход)
./bin/client -config config.json -mode oneshot

# daemon режим (постоянная работа)
./bin/client -config config.json

# С флагами (без конфига)
./bin/client -server http://localhost:8080 \
             -key key.enc \
             -password mypassword \
             -mode daemon \
             -interval 30s
```

### Установка как демон

#### Linux (systemd)

```bash
sudo ./scripts/install/install-client.sh -b ./bin/client -d
```

Проверка:
```bash
systemctl status chaindocs-client
journalctl -u chaindocs-client -f
```

#### macOS (launchd)

```bash
sudo ./scripts/install/install-client.sh -b ./bin/client -d
```

Проверка:
```bash
launchctl list | grep chaindocs
tail -f /opt/chaindocs/logs/chaindocs-client.log
```

### Удаление

```bash
sudo ./scripts/install/install-client.sh --uninstall
```

---

## 3️⃣ SELF-HEALING (Отзыв ключей)

### Детектор компрометации

Клиент автоматически обнаруживает:
- **Чужие подписи** на блоках
- **Подозрительную активность**

При обнаружении отправляется уведомление на вебхук (если настроен).

### API отзыва ключей

#### `POST /api/revoke`

Отозвать ключ (требуется подтверждение новым ключом):

```bash
# 1. Генерируем новый ключ
./bin/keygen -password newpass -out newkey.enc

# 2. Регистрируем новый ключ на сервере
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"public_key":"<NEW_PUB_KEY>"}'

# 3. Подписываем сообщение для отзыва
./bin/signer -key newkey.enc -password newpass \
  -message "revoke:<OLD_PUB_KEY>"

# 4. Отправляем запрос на отзыв
curl -X POST http://localhost:8080/api/revoke \
  -H "Content-Type: application/json" \
  -d '{
    "public_key": "<OLD_PUB_KEY>",
    "new_public_key": "<NEW_PUB_KEY>",
    "new_signature": "<SIGNATURE>",
    "reason": "compromised"
  }'
```

**Ответ:**
```json
{
  "status": "revoked",
  "key": "abc123...",
  "reason": "compromised"
}
```

#### `GET /api/keys/revoked`

Список отозванных ключей:

```bash
curl http://localhost:8080/api/keys/revoked
```

**Ответ:**
```json
{
  "count": 1,
  "keys": [
    {
      "public_key": "abc123...",
      "reason": "compromised",
      "revoked_at": "2026-02-22T12:00:00Z"
    }
  ]
}
```

### Защита от использования отозванного ключа

Попытка подписать блок отозванным ключом вернёт ошибку:

```json
{
  "error": "Public key revoked at 2026-02-22T12:00:00Z: compromised"
}
```

---

## 4️⃣ DOCKER

### Сборка образа

```bash
# Сервер
docker build -t chaindocs-server:latest .

# Клиент
docker build -f Dockerfile.client -t chaindocs-client:latest .
```

### Запуск через docker-compose

```bash
# Только сервер
docker-compose up -d chaindocs-server

# С клиентом (раскомментировать в docker-compose.yml)
docker-compose up -d
```

**Данные сохраняются в volumes:**
- `chaindocs-data` — блокчейн БД
- `chaindocs-uploads` — загруженные файлы

---

## 5️⃣ ТЕСТЫ

### Запуск всех тестов

```bash
go test -v ./...
```

### Интеграционные тесты

```bash
go test -v ./test/integration/...
```

**Тесты проверяют:**
- ✅ Консенсус 51%
- ✅ Мульти-подписи
- ✅ Отзыв ключей
- ✅ Детектор чужих подписей

---

## 📊 СРАВНЕНИЕ: БЫЛО / СТАЛО

| Функция | Было | Стало |
|---------|------|-------|
| Подписей на блок | 1 | ∞ (все зарегистрированные) |
| Консенсус | Нет | 51%+ ключей |
| Режим клиента | oneshot | oneshot + daemon |
| Конфигурация | Флаги | JSON файл + флаги |
| Отзыв ключей | Нет | API + подписи новым ключом |
| Детектор компрометации | Нет | Вебхук + логирование |
| Docker | Нет | Dockerfile + docker-compose |
| Установка демона | Нет | systemd + launchd |

---

## 🚀 БЫСТРЫЙ СТАРТ

### 1. Запуск сервера

```bash
make run
```

### 2. Генерация ключей для 3 клиентов

```bash
# Клиент 1
make keygen PASSWORD=pass1 OUT=client1.enc

# Клиент 2
make keygen PASSWORD=pass2 OUT=client2.enc

# Клиент 3
make keygen PASSWORD=pass3 OUT=client3.enc
```

### 3. Регистрация ключей

```bash
# Для каждого клиента (заменить PUBLIC_KEY)
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"public_key":"<PUBLIC_KEY>"}'
```

### 4. Загрузка документа

```bash
curl -X POST http://localhost:8080/api/upload \
  -F "file=@document.pdf"
```

### 5. Запуск клиентов (подпишут блок)

```bash
# В разных терминалах или фоне
./bin/client -password pass1 -mode oneshot
./bin/client -password pass2 -mode oneshot
./bin/client -password pass3 -mode oneshot
```

### 6. Проверка консенсуса

```bash
# Последний блок должен иметь 3 подписи
curl http://localhost:8080/api/blocks/last | jq

# Статус консенсуса
curl http://localhost:8080/api/blocks/<hash>/consensus | jq
```

---

## ⚠️ BREAKING CHANGES

### Изменения в API

1. **`GET /api/blocks/last`** — поле `signature` заменено на `signatures` (массив)
2. **`POST /api/sign`** — теперь добавляет подпись, а не заменяет

### Изменения в структуре БД

Добавлены бакеты:
- `pubkeys` — зарегистрированные ключи
- `revoked` — отозванные ключи

**Миграция:** Старые БД совместимы, новые бакеты создаются автоматически.

---

## 📝 TODO (Планы)

- [ ] Веб-интерфейс для просмотра блоков
- [ ] P2P коммуникация между клиентами
- [ ] Автоматический отзыв при детекции компрометации
- [ ] Поддержка пороговых подписей (threshold signatures)
- [ ] Аудит логов подписей

---

**Версия:** 0.2.0  
**Дата:** 2026-02-22

# 🛠️ ПОЛНАЯ ИНСТРУКЦИЯ ПО УСТАНОВКЕ И НАСТРОЙКЕ CHAINDOCS

## Содержание

1. [Установка сервера](#1-установка-сервера)
2. [Установка клиента-демона](#2-установка-клиента-демона)
3. [Боевое тестирование](#3-боевое-тестирование)
4. [Мониторинг и управление](#4-мониторинг-и-управление)

---

## 1. Установка сервера

### Вариант A: Docker (рекомендуется)

```bash
# 1. Клонируем репозиторий
git clone https://github.com/EvgeniiAndronov/ChainDocs.git
cd ChainDocs

# 2. Запускаем через docker-compose
docker-compose up -d

# 3. Проверяем статус
docker-compose ps
docker-compose logs -f chaindocs-server
```

**Сервер доступен:** `http://localhost:8080`

### Вариант B: Локальный запуск

```bash
# 1. Сборка
cd /path/to/ChainDocs
go build -o bin/server ./cmd/server/main.go

# 2. Запуск
./bin/server

# 3. Или через make
make run
```

---

## 2. Установка клиента-демона

### Шаг 2.1: Подготовка

```bash
# 1. Сборка клиента
go build -o bin/client ./cmd/client/main.go ./cmd/client/config.go

# 2. Генерация ключа клиента
./bin/keygen -password "SecurePassword123!" -out client1.enc

# Сохраните публичный ключ из вывода!
# Пример: 5d1cb17def07135434c50169817062037fcfbf420ca9c9bf5eade6c8f39cf155
```

### Шаг 2.2: Регистрация ключа на сервере

```bash
# Зарегистрируйте публичный ключ (замените на ваш)
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"public_key":"5d1cb17def07135434c50169817062037fcfbf420ca9c9bf5eade6c8f39cf155"}'
```

### Шаг 2.3: Создание конфигурации

```bash
# 1. Генерируем пример конфига
./bin/client -gen-config > config.json

# 2. Редактируем конфиг
nano config.json
```

**Пример config.json:**
```json
{
  "server": "http://localhost:8080",
  "key_file": "client1.enc",
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

### Шаг 2.4: Установка как демон

#### Linux (systemd)

```bash
# 1. Создаём директорию
sudo mkdir -p /opt/chaindocs
sudo mkdir -p /opt/chaindocs/logs
sudo mkdir -p /opt/chaindocs/data

# 2. Копируем файлы
sudo cp bin/client /opt/chaindocs/chaindocs-client
sudo cp config.json /opt/chaindocs/config.json
sudo cp client1.enc /opt/chaindocs/client1.enc

# 3. Устанавливаем права
sudo chmod +x /opt/chaindocs/chaindocs-client
sudo chmod 600 /opt/chaindocs/config.json /opt/chaindocs/client1.enc
sudo chown -R root:root /opt/chaindocs

# 4. Устанавливаем systemd сервис
sudo cp scripts/install/chaindocs-client.service /etc/systemd/system/
sudo systemctl daemon-reload

# 5. Включаем и запускаем
sudo systemctl enable chaindocs-client
sudo systemctl start chaindocs-client

# 6. Проверяем статус
sudo systemctl status chaindocs-client
```

**Полезные команды:**
```bash
# Просмотр логов
sudo journalctl -u chaindocs-client -f

# Перезапуск
sudo systemctl restart chaindocs-client

# Остановка
sudo systemctl stop chaindocs-client

# Удаление
sudo systemctl disable chaindocs-client
sudo rm /etc/systemd/system/chaindocs-client.service
sudo systemctl daemon-reload
```

#### macOS (launchd)

```bash
# 1. Создаём директорию
sudo mkdir -p /opt/chaindocs
sudo mkdir -p /opt/chaindocs/logs

# 2. Копируем файлы
sudo cp bin/client /opt/chaindocs/chaindocs-client
sudo cp config.json /opt/chaindocs/config.json
sudo cp client1.enc /opt/chaindocs/client1.enc

# 3. Устанавливаем права
sudo chmod +x /opt/chaindocs/chaindocs-client
sudo chmod 600 /opt/chaindocs/config.json /opt/chaindocs/client1.enc

# 4. Устанавливаем launchd plist
sudo cp scripts/install/com.chaindocs.client.plist /Library/LaunchDaemons/

# 5. Загружаем и запускаем
sudo launchctl load /Library/LaunchDaemons/com.chaindocs.client.plist
sudo launchctl start com.chaindocs.client

# 6. Проверяем
sudo launchctl list | grep chaindocs
```

**Полезные команды:**
```bash
# Просмотр логов
tail -f /opt/chaindocs/logs/chaindocs-client.log

# Перезапуск
sudo launchctl unload /Library/LaunchDaemons/com.chaindocs.client.plist
sudo launchctl load /Library/LaunchDaemons/com.chaindocs.client.plist

# Удаление
sudo launchctl unload /Library/LaunchDaemons/com.chaindocs.client.plist
sudo rm /Library/LaunchDaemons/com.chaindocs.client.plist
```

### Альтернатива: Автоматическая установка

```bash
# Простая установка (Linux/macOS)
sudo ./scripts/install/install-client.sh \
  -b ./bin/client \
  -c ./config.json \
  -d

# Удаление
sudo ./scripts/install/install-client.sh --uninstall
```

---

## 3. Боевое тестирование

### Сценарий: 3 клиента подписывают документ

```bash
#!/bin/bash
# test-live.sh - Скрипт для тестирования

set -e

echo "🧪 Начинаем боевое тестирование ChainDocs"
echo "=========================================="

# Очистка (опционально)
# rm -f blockchain.db *.enc

# Шаг 1: Запуск сервера (в фоне)
echo "📀 Запуск сервера..."
./bin/server &
SERVER_PID=$!
sleep 2

# Проверка сервера
curl -s http://localhost:8080/api/blocks/last | jq '.height' || {
    echo "❌ Сервер не запустился"
    kill $SERVER_PID
    exit 1
}
echo "✅ Сервер запущен"

# Шаг 2: Генерация 3 ключей
echo "🔑 Генерация ключей..."
./bin/keygen -password pass1 -out client1.enc
./bin/keygen -password pass2 -out client2.enc
./bin/keygen -password pass3 -out client3.enc

# Извлекаем публичные ключи
PUB1=$(./bin/keygen -password pass1 -out client1.enc 2>&1 | grep "Public key:" | awk '{print $NF}')
PUB2=$(./bin/keygen -password pass2 -out client2.enc 2>&1 | grep "Public key:" | awk '{print $NF}')
PUB3=$(./bin/keygen -password pass3 -out client3.enc 2>&1 | grep "Public key:" | awk '{print $NF}')

echo "Клиент 1: $PUB1"
echo "Клиент 2: $PUB2"
echo "Клиент 3: $PUB3"

# Шаг 3: Регистрация ключей
echo "📝 Регистрация ключей..."
curl -s -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"$PUB1\"}" | jq -r '.status'
curl -s -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"$PUB2\"}" | jq -r '.status'
curl -s -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"$PUB3\"}" | jq -r '.status'

# Проверка количества ключей
KEYS_COUNT=$(curl -s http://localhost:8080/api/keys | jq '.count')
echo "✅ Зарегистрировано ключей: $KEYS_COUNT"

# Шаг 4: Загрузка тестового документа
echo "📄 Загрузка документа..."
echo "Test Document" > test.pdf
UPLOAD_RESULT=$(curl -s -X POST http://localhost:8080/api/upload \
  -F "file=@test.pdf" | jq -r '.block_hash')
echo "✅ Блок создан: $UPLOAD_RESULT"

# Шаг 5: Запуск клиентов
echo "✍️  Подписание блока клиентами..."

CHAINDOCS_KEY_PASSWORD=pass1 ./bin/client -mode oneshot
CHAINDOCS_KEY_PASSWORD=pass2 ./bin/client -mode oneshot
CHAINDOCS_KEY_PASSWORD=pass3 ./bin/client -mode oneshot

# Шаг 6: Проверка консенсуса
echo "📊 Проверка консенсуса..."
sleep 1

BLOCK_HASH=$(curl -s http://localhost:8080/api/blocks/last | jq -r '.hash')
CONSENSUS=$(curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus" | jq)

echo "Статус консенсуса:"
echo "$CONSENSUS" | jq -r '"\(.signatures)/\(.required) подписей (\(.percent)%)"'

if [ "$(echo "$CONSENSUS" | jq -r '.consensus_reached')" = "true" ]; then
    echo "✅ КОНСЕНСУС ДОСТИГНУТ!"
else
    echo "❌ Консенсус не достигнут"
fi

# Шаг 7: Проверка блока
echo "🔍 Проверка блока..."
curl -s http://localhost:8080/api/blocks/last | jq '.signatures | length'

# Завершение
echo ""
echo "=========================================="
echo "✅ Боевое тестирование завершено!"
echo ""

kill $SERVER_PID
```

### Запуск теста

```bash
chmod +x test-live.sh
./test-live.sh
```

---

## 4. Мониторинг и управление

### API для мониторинга

```bash
# Последний блок
curl http://localhost:8080/api/blocks/last | jq

# Все блоки
curl http://localhost:8080/api/blocks | jq 'length'

# Статус консенсуса для блока
curl http://localhost:8080/api/blocks/<HASH>/consensus | jq

# Зарегистрированные ключи
curl http://localhost:8080/api/keys | jq

# Отозванные ключи
curl http://localhost:8080/api/keys/revoked | jq

# Загруженные документы
ls -la uploads/
```

### Логи

```bash
# Сервер (journalctl)
sudo journalctl -u chaindocs-server -f

# Клиент (journalctl)
sudo journalctl -u chaindocs-client -f

# Клиент (файл)
tail -f /opt/chaindocs/logs/chaindocs-client.log
```

---

## 📊 Чек-лист проверки

- [ ] Сервер запущен и отвечает на `/api/blocks/last`
- [ ] Ключи сгенерированы и зарегистрированы
- [ ] Клиент установлен как демон
- [ ] Демон запускается автоматически после перезагрузки
- [ ] Блоки подписываются клиентами
- [ ] Консенсус достигается при 51%+ подписей
- [ ] Логи пишутся без ошибок

---

## 🆘 Troubleshooting

### Клиент не подключается к серверу

```bash
# Проверьте доступность сервера
curl http://localhost:8080/api/blocks/last

# Проверьте конфиг
cat /opt/chaindocs/config.json | jq '.server'

# Проверьте права на ключ
ls -la /opt/chaindocs/client1.enc
```

### Демон не запускается

```bash
# Linux: проверка статуса
sudo systemctl status chaindocs-client

# Linux: подробные логи
sudo journalctl -u chaindocs-client -n 50

# macOS: проверка
sudo launchctl list | grep chaindocs

# macOS: логи
tail -f /opt/chaindocs/logs/chaindocs-client.log
```

### Ошибка "Password required"

```bash
# Установите переменную окружения в systemd сервисе
sudo systemctl edit chaindocs-client

# Добавьте:
[Service]
Environment="CHAINDOCS_KEY_PASSWORD=your_password"

# Перезапустите
sudo systemctl daemon-reload
sudo systemctl restart chaindocs-client
```

---

**Версия:** 0.2.0  
**Дата:** 2026-02-22

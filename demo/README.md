# ChainDocs Demonстрационная среда

## Быстрый старт

### 1. Запуск сервера

```bash
# В одном терминале
export CHAINDOCS_AUTH_TOKEN="demo_token"
./demo/bin/server -config demo/server-config.json
```

### 2. Регистрация ключей

```bash
# В другом терминале
source demo/demo-keys/public_keys.txt

curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\\"public_key\\":\\"$CLIENT1_PUBLIC_KEY\\"}"

curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\\"public_key\\":\\"$CLIENT2_PUBLIC_KEY\\"}"

curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\\"public_key\\":\\"$CLIENT3_PUBLIC_KEY\\"}"
```

### 3. Запуск клиентов

```bash
# Клиент 1
export CHAINDOCS_CLIENT1_PASSWORD="demo123"
./demo/bin/client -config demo/client1-config.json &

# Клиент 2
export CHAINDOCS_CLIENT2_PASSWORD="demo123"
./demo/bin/client -config demo/client2-config.json &

# Клиент 3
export CHAINDOCS_CLIENT3_PASSWORD="demo123"
./demo/bin/client -config demo/client3-config.json &
```

### 4. Загрузка документа

```bash
# Создайте тестовый PDF
echo "Test Document" > test.txt

# Загрузите
curl -X POST http://localhost:8080/api/upload \
  -F "file=@test.txt"
```

### 5. Проверка консенсуса

```bash
# Получите хэш последнего блока
BLOCK_HASH=8c1b0478dfb795b14efc1cc026ca6496a591d3e65b85e8bbdf8cadfbd54fcc07

# Проверьте статус консенсуса
curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus" | jq
```

### 6. Веб-интерфейс

Откройте в браузере:
```
http://localhost:8080/web/login?token=demo_token
```

## Остановка

```bash
# Остановить все процессы
pkill -f "demo/bin/server"
pkill -f "demo/bin/client"
```

## Очистка

```bash
./demo/demo-cleanup.sh
```

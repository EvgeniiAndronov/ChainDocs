#!/bin/bash
# demo-run.sh - Запуск демонстрационной среды

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "=================================="
echo "ChainDocs Demo - Запуск"
echo "=================================="
echo ""

# Запуск сервера
echo "🚀 Запуск сервера..."
export CHAINDOCS_AUTH_TOKEN="demo_token"
./demo/bin/server -config demo/server-config.json > demo/demo_logs/server.log 2>&1 &
SERVER_PID=$!
echo "✅ Сервер запущен (PID: $SERVER_PID)"

sleep 3

# Проверка что сервер работает
if ! curl -s http://localhost:8080/api/blocks/last > /dev/null 2>&1; then
    echo "❌ Сервер не запустился!"
    exit 1
fi

# Загрузка публичных ключей
echo ""
echo "📋 Загрузка публичных ключей..."
if [ -f "demo/demo-keys/public_keys.txt" ]; then
    # Используем set -a для экспорта переменных из файла
    set -a
    . demo/demo-keys/public_keys.txt
    set +a
    echo "  ✅ Ключи загружены"
    echo "  CLIENT1: ${CLIENT1_PUBLIC_KEY:0:16}..."
    echo "  CLIENT2: ${CLIENT2_PUBLIC_KEY:0:16}..."
    echo "  CLIENT3: ${CLIENT3_PUBLIC_KEY:0:16}..."
else
    echo "❌ Файл public_keys.txt не найден!"
    echo "Запустите: ./demo/demo-setup.sh"
    exit 1
fi

# Регистрация ключей
echo ""
echo "🔑 Регистрация ключей на сервере..."

register_key() {
    local key=$1
    local name=$2
    result=$(curl -s -X POST http://localhost:8080/api/register \
      -H "Content-Type: application/json" \
      -d "{\"public_key\":\"$key\"}" | jq -r '.status')
    if [ "$result" = "registered" ]; then
        echo "  ✅ $name зарегистрирован"
    else
        echo "  ❌ $name не зарегистрирован: $result"
    fi
}

register_key "$CLIENT1_PUBLIC_KEY" "Client 1"
register_key "$CLIENT2_PUBLIC_KEY" "Client 2"
register_key "$CLIENT3_PUBLIC_KEY" "Client 3"

echo "✅ Ключи зарегистрированы"

# Запуск клиентов
echo ""
echo "👥 Запуск клиентов..."

export CHAINDOCS_CLIENT1_PASSWORD="demo123"
./demo/bin/client -config demo/client1-config.json > demo/demo_logs/client1.log 2>&1 &
echo "✅ Клиент 1 запущен"

export CHAINDOCS_CLIENT2_PASSWORD="demo123"
./demo/bin/client -config demo/client2-config.json > demo/demo_logs/client2.log 2>&1 &
echo "✅ Клиент 2 запущен"

export CHAINDOCS_CLIENT3_PASSWORD="demo123"
./demo/bin/client -config demo/client3-config.json > demo/demo_logs/client3.log 2>&1 &
echo "✅ Клиент 3 запущен"

echo ""
echo "=================================="
echo "✅ Все сервисы запущены!"
echo "=================================="
echo ""
echo "Веб-интерфейс: http://localhost:8080/web/login?token=demo_token"
echo "API: http://localhost:8080/api/blocks/last"
echo ""
echo "Для остановки: ./demo/demo-stop.sh"
echo ""

# Сохранение PID
echo $SERVER_PID > demo/demo_logs/server.pid

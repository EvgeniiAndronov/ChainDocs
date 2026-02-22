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

sleep 2

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

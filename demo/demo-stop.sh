#!/bin/bash
# demo-stop.sh - Остановка демонстрационной среды ChainDocs

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Цвета
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok() { echo -e "${GREEN}✅${NC} $1"; }

echo ""
log_info "🛑 Остановка демонстрационной среды..."
echo ""

# Остановка клиентов
if [ -f "demo/demo_logs/client1.pid" ]; then
    CLIENT1_PID=$(cat demo/demo_logs/client1.pid)
    if kill -0 $CLIENT1_PID 2>/dev/null; then
        kill $CLIENT1_PID
        log_ok "Клиент 1 остановлен (PID: $CLIENT1_PID)"
    fi
    rm -f demo/demo_logs/client1.pid
fi

if [ -f "demo/demo_logs/client2.pid" ]; then
    CLIENT2_PID=$(cat demo/demo_logs/client2.pid)
    if kill -0 $CLIENT2_PID 2>/dev/null; then
        kill $CLIENT2_PID
        log_ok "Клиент 2 остановлен (PID: $CLIENT2_PID)"
    fi
    rm -f demo/demo_logs/client2.pid
fi

if [ -f "demo/demo_logs/client3.pid" ]; then
    CLIENT3_PID=$(cat demo/demo_logs/client3.pid)
    if kill -0 $CLIENT3_PID 2>/dev/null; then
        kill $CLIENT3_PID
        log_ok "Клиент 3 остановлен (PID: $CLIENT3_PID)"
    fi
    rm -f demo/demo_logs/client3.pid
fi

# Остановка сервера
if [ -f "demo/demo_logs/server.pid" ]; then
    SERVER_PID=$(cat demo/demo_logs/server.pid)
    if kill -0 $SERVER_PID 2>/dev/null; then
        kill $SERVER_PID
        log_ok "Сервер остановлен (PID: $SERVER_PID)"
    fi
    rm -f demo/demo_logs/server.pid
fi

# Дополнительная очистка процессов
pkill -f "demo/bin/server" 2>/dev/null || true
pkill -f "demo/bin/client" 2>/dev/null || true

echo ""
log_ok "✅ Все сервисы остановлены"
echo ""
log_info "Для повторного запуска: ./demo/demo-start.sh"
echo ""

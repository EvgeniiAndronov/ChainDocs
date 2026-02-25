#!/bin/bash
# demo-cleanup.sh - Полная очистка демонстрационной среды ChainDocs

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok() { echo -e "${GREEN}✅${NC} $1"; }
log_warn() { echo -e "${YELLOW}⚠️${NC} $1"; }

echo ""
log_info "🧹 Очистка демонстрационной среды..."
echo ""

# Остановка сервисов
log_info "Остановка сервисов..."
pkill -f "demo/bin/server" 2>/dev/null || true
pkill -f "demo/bin/client" 2>/dev/null || true
sleep 1
log_ok "Сервисы остановлены"

# Удаление данных
log_info "Удаление данных..."
rm -rf demo/demo_blockchain.db
rm -rf blockchain.db
rm -rf demo/demo_uploads/*
rm -rf demo/demo_logs/*
rm -rf demo/demo-keys/*.enc
rm -f demo/demo-keys/public_keys.txt
rm -f demo/demo-test-document.pdf
rm -f /tmp/key*.log
rm -f /tmp/demo-*.txt
log_ok "Данные удалены"

# Удаление PID файлов
rm -f demo/demo_logs/*.pid
log_ok "PID файлы удалены"

echo ""
log_ok "✅ Очистка завершена"
echo ""
log_info "Для нового запуска выполните: ./demo/demo-start.sh"
echo ""

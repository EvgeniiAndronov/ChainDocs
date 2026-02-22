#!/bin/bash
# demo-cleanup.sh - Полная очистка демонстрационной среды

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "🧹 Полная очистка демонстрационной среды..."

# Остановка сервисов
echo "⏹️  Остановка сервисов..."
pkill -f "demo/bin/server" 2>/dev/null || true
pkill -f "demo/bin/client" 2>/dev/null || true
sleep 1

# Удаление данных
echo "🗑️  Удаление данных..."
rm -f demo/demo_blockchain.db
rm -f blockchain.db
rm -rf demo/demo-keys
rm -rf demo/demo_uploads/*
rm -rf demo/demo_logs/*
rm -f demo/demo-test-document.txt
rm -f /tmp/demo-*.txt
rm -f /tmp/key*.log

# Удаление скомпилированных бинарников
echo "🗑️  Удаление бинарников..."
rm -rf demo/bin

echo "✅ Очистка завершена"
echo ""
echo "Для повторной настройки выполните:"
echo "  ./demo/demo-setup.sh"

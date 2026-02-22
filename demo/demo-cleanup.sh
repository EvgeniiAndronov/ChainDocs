#!/bin/bash
# demo-cleanup.sh - Очистка демонстрационной среды

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "🧹 Очистка демонстрационной среды..."

# Остановка сервисов
pkill -f "demo/bin/server" 2>/dev/null || true
pkill -f "demo/bin/client" 2>/dev/null || true

# Удаление данных
rm -rf demo/demo_blockchain.db
rm -rf demo/demo_uploads/*
rm -rf demo/demo_logs/*
rm -rf demo/demo-keys/*.enc

echo "✅ Очистка завершена"

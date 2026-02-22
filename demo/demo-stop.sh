#!/bin/bash
# demo-stop.sh - Остановка демонстрационной среды

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "🛑 Остановка сервисов..."

pkill -f "demo/bin/server"
pkill -f "demo/bin/client"

echo "✅ Все сервисы остановлены"

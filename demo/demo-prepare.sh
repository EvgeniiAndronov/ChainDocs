#!/bin/bash
# demo-prepare.sh - Подготовка демонстрационной среды ChainDocs
# Очищает старое окружение, собирает проект, генерирует ключи

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok() { echo -e "${GREEN}✅${NC} $1"; }
log_warn() { echo -e "${YELLOW}⚠️${NC} $1"; }
log_error() { echo -e "${RED}❌${NC} $1"; }
log_step() { echo -e "\n${CYAN}══════════════════════════════════════${NC}"; echo -e "${CYAN}║${NC} $1"; echo -e "${CYAN}══════════════════════════════════════${NC}\n"; }

echo ""
log_step "🧹 ChainDocs Demo - Подготовка окружения"
echo ""

# Шаг 1: Очистка предыдущей демонстрации
log_info "Очистка предыдущей демонстрации..."
pkill -f "demo/bin/server" 2>/dev/null || true
pkill -f "demo/bin/client" 2>/dev/null || true
sleep 1

rm -f demo/demo_blockchain.db
rm -f blockchain.db
rm -rf demo/demo_uploads/*
rm -rf demo/demo_logs/*
rm -f demo/demo-keys/*.enc
rm -f demo/demo-keys/public_keys.txt
rm -f demo/demo-test-document.pdf
rm -f /tmp/key*.log
log_ok "Предыдущая демонстрация очищена"

# Создание директорий
mkdir -p demo/demo-keys
mkdir -p demo/demo_uploads
mkdir -p demo/demo_logs
log_ok "Директории созданы"

# Шаг 2: Сборка проекта
log_step "📦 Сборка проекта"
log_info "Компиляция бинарных файлов..."

go build -o demo/bin/server ./cmd/server/main.go ./cmd/server/config.go || { log_error "Ошибка сборки сервера"; exit 1; }
go build -o demo/bin/client ./cmd/client/main.go ./cmd/client/config.go || { log_error "Ошибка сборки клиента"; exit 1; }
go build -o demo/bin/keygen ./cmd/keygen/main.go || { log_error "Ошибка сборки keygen"; exit 1; }
log_ok "Проект собран"

# Шаг 3: Генерация ключей для клиентов
log_step "🔑 Генерация ключей для клиентов"

log_info "Генерация ключа для клиента 1..."
./demo/bin/keygen -password "demo123" -out demo/demo-keys/client1.enc > /tmp/key1.log 2>&1
CLIENT1_PUB=$(grep "Public key" /tmp/key1.log | sed 's/.*: //')
log_ok "Клиент 1: ${CLIENT1_PUB:0:32}..."

log_info "Генерация ключа для клиента 2..."
./demo/bin/keygen -password "demo123" -out demo/demo-keys/client2.enc > /tmp/key2.log 2>&1
CLIENT2_PUB=$(grep "Public key" /tmp/key2.log | sed 's/.*: //')
log_ok "Клиент 2: ${CLIENT2_PUB:0:32}..."

log_info "Генерация ключа для клиента 3..."
./demo/bin/keygen -password "demo123" -out demo/demo-keys/client3.enc > /tmp/key3.log 2>&1
CLIENT3_PUB=$(grep "Public key" /tmp/key3.log | sed 's/.*: //')
log_ok "Клиент 3: ${CLIENT3_PUB:0:32}..."

# Сохранение публичных ключей
cat > demo/demo-keys/public_keys.txt << EOF
# Публичные ключи для демонстрации
# Сгенерированы: $(date)

CLIENT1_PUBLIC_KEY="$CLIENT1_PUB"
CLIENT2_PUBLIC_KEY="$CLIENT2_PUB"
CLIENT3_PUBLIC_KEY="$CLIENT3_PUB"
EOF
log_ok "Публичные ключи сохранены в demo/demo-keys/public_keys.txt"

# Копируем конфиг сервера в корень
cp demo/server-config.json config.json
log_ok "Конфиг сервера скопирован в config.json"

# Создание тестового PDF документа
echo "%PDF-1.4" > demo/demo-test-document.pdf
echo "Test Document for ChainDocs Demo" >> demo/demo-test-document.pdf
echo "Created: $(date)" >> demo/demo-test-document.pdf
echo "Document ID: $(openssl rand -hex 8)" >> demo/demo-test-document.pdf
echo "%%EOF" >> demo/demo-test-document.pdf
log_ok "Тестовый документ создан"

echo ""
log_ok "✅ Окружение подготовлено!"
echo ""
log_info "Следующий шаг: ./demo/demo-start.sh"
echo ""

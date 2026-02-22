#!/bin/bash
# demo-setup.sh - Настройка демонстрационной среды ChainDocs

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok() { echo -e "${GREEN}✅${NC} $1"; }
log_warn() { echo -e "${YELLOW}⚠️${NC} $1"; }
log_error() { echo -e "${RED}❌${NC} $1"; }

echo "=================================="
echo "ChainDocs Demo Setup"
echo "=================================="
echo ""

# Очистка предыдущей демонстрации
log_info "Очистка предыдущей демонстрации..."
pkill -f "demo/bin/server" 2>/dev/null || true
pkill -f "demo/bin/client" 2>/dev/null || true
sleep 1

rm -f demo/demo_blockchain.db
rm -f blockchain.db
rm -rf demo/demo-keys
rm -rf demo/demo_uploads/*
rm -rf demo/demo_logs/*
rm -f demo/demo-test-document.txt
rm -f /tmp/demo-*.txt
rm -f /tmp/key*.log
log_ok "Предыдущая демонстрация очищена"
mkdir -p demo/demo-keys
mkdir -p demo/demo_uploads
mkdir -p demo/demo_logs
log_ok "Директории созданы"

# Сборка проекта
log_info "Сборка проекта..."
go build -o demo/bin/server ./cmd/server/main.go ./cmd/server/config.go
go build -o demo/bin/client ./cmd/client/main.go ./cmd/client/config.go
go build -o demo/bin/keygen ./cmd/keygen/main.go
go build -o demo/bin/signer ./cmd/signer/main.go
log_ok "Проект собран"

# Генерация ключей для клиентов
log_info "Генерация ключей для клиентов..."

./demo/bin/keygen -password "demo123" -out demo/demo-keys/client1.enc > /tmp/key1.log 2>&1
CLIENT1_PUB=$(grep "Public key" /tmp/key1.log | sed 's/.*: //')

./demo/bin/keygen -password "demo123" -out demo/demo-keys/client2.enc > /tmp/key2.log 2>&1
CLIENT2_PUB=$(grep "Public key" /tmp/key2.log | sed 's/.*: //')

./demo/bin/keygen -password "demo123" -out demo/demo-keys/client3.enc > /tmp/key3.log 2>&1
CLIENT3_PUB=$(grep "Public key" /tmp/key3.log | sed 's/.*: //')

log_ok "Ключи сгенерированы"

# Сохранение публичных ключей
cat > demo/demo-keys/public_keys.txt << EOF
# Публичные ключи для демонстрации
# Сохраните их для регистрации на сервере

CLIENT1_PUBLIC_KEY="$CLIENT1_PUB"
CLIENT2_PUBLIC_KEY="$CLIENT2_PUB"
CLIENT3_PUBLIC_KEY="$CLIENT3_PUB"
EOF

log_info "Публичные ключи сохранены в demo/demo-keys/public_keys.txt"

# Создание README для демо
cat > demo/README.md << EOF
# ChainDocs Demonстрационная среда

## Быстрый старт

### 1. Запуск сервера

\`\`\`bash
# В одном терминале
export CHAINDOCS_AUTH_TOKEN="demo_token"
./demo/bin/server -config demo/server-config.json
\`\`\`

### 2. Регистрация ключей

\`\`\`bash
# В другом терминале
source demo/demo-keys/public_keys.txt

curl -X POST http://localhost:8080/api/register \\
  -H "Content-Type: application/json" \\
  -d "{\\\"public_key\\\":\\\"\$CLIENT1_PUBLIC_KEY\\\"}"

curl -X POST http://localhost:8080/api/register \\
  -H "Content-Type: application/json" \\
  -d "{\\\"public_key\\\":\\\"\$CLIENT2_PUBLIC_KEY\\\"}"

curl -X POST http://localhost:8080/api/register \\
  -H "Content-Type: application/json" \\
  -d "{\\\"public_key\\\":\\\"\$CLIENT3_PUBLIC_KEY\\\"}"
\`\`\`

### 3. Запуск клиентов

\`\`\`bash
# Клиент 1
export CHAINDOCS_CLIENT1_PASSWORD="demo123"
./demo/bin/client -config demo/client1-config.json &

# Клиент 2
export CHAINDOCS_CLIENT2_PASSWORD="demo123"
./demo/bin/client -config demo/client2-config.json &

# Клиент 3
export CHAINDOCS_CLIENT3_PASSWORD="demo123"
./demo/bin/client -config demo/client3-config.json &
\`\`\`

### 4. Загрузка документа

\`\`\`bash
# Создайте тестовый PDF
echo "Test Document" > test.txt

# Загрузите
curl -X POST http://localhost:8080/api/upload \\
  -F "file=@test.txt"
\`\`\`

### 5. Проверка консенсуса

\`\`\`bash
# Получите хэш последнего блока
BLOCK_HASH=$(curl -s http://localhost:8080/api/blocks/last | jq -r '.hash')

# Проверьте статус консенсуса
curl -s "http://localhost:8080/api/blocks/\$BLOCK_HASH/consensus" | jq
\`\`\`

### 6. Веб-интерфейс

Откройте в браузере:
\`\`\`
http://localhost:8080/web/login?token=demo_token
\`\`\`

## Остановка

\`\`\`bash
# Остановить все процессы
pkill -f "demo/bin/server"
pkill -f "demo/bin/client"
\`\`\`

## Очистка

\`\`\`bash
./demo/demo-cleanup.sh
\`\`\`
EOF

log_ok "README создан"

# Скрипт запуска
cat > demo/demo-run.sh << 'RUNEOF'
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
RUNEOF

chmod +x demo/demo-run.sh

# Скрипт остановки
cat > demo/demo-stop.sh << 'STOPEOF'
#!/bin/bash
# demo-stop.sh - Остановка демонстрационной среды

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "🛑 Остановка сервисов..."

pkill -f "demo/bin/server"
pkill -f "demo/bin/client"

echo "✅ Все сервисы остановлены"
STOPEOF

chmod +x demo/demo-stop.sh

# Скрипт очистки
cat > demo/demo-cleanup.sh << 'CLEANEOF'
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
CLEANEOF

chmod +x demo/demo-cleanup.sh

log_ok "Скрипты созданы"

# Создание тестового документа
echo "Test Document for ChainDocs Demo" > demo/demo-test-document.txt
echo "Created: $(date)" >> demo/demo-test-document.txt
log_ok "Тестовый документ создан"

# Финальные инструкции
echo ""
echo "=================================="
echo "✅ Demo setup completed!"
echo "=================================="
echo ""
echo "Следующие шаги:"
echo "  1. ./demo/demo-run.sh - запуск сервера и клиентов"
echo "  2. Зарегистрировать ключи (см. demo/README.md)"
echo "  3. Загрузить тестовый документ"
echo "  4. Проверить консенсус"
echo ""
echo "Веб-интерфейс:"
echo "  http://localhost:8080/web/login?token=demo_token"
echo ""
echo "Для остановки:"
echo "  ./demo/demo-stop.sh"
echo ""
echo "Для очистки:"
echo "  ./demo/demo-cleanup.sh"
echo ""

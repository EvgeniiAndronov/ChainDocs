#!/bin/bash
# demo-start.sh - Запуск демонстрационной среды ChainDocs
# Запускает сервер, регистрирует ключи, запускает клиентов

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Цвета
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok() { echo -e "${GREEN}✅${NC} $1"; }
log_step() { echo -e "\n${CYAN}══════════════════════════════════════${NC}"; echo -e "${CYAN}║${NC} $1"; echo -e "${CYAN}══════════════════════════════════════${NC}\n"; }

echo ""
log_step "🚀 ChainDocs Demo - Запуск"
echo ""

# Проверка что окружение подготовлено
if [ ! -f "demo/demo-keys/public_keys.txt" ]; then
    echo "❌ Окружение не подготовлено!"
    echo "Выполните: ./demo/demo-prepare.sh"
    exit 1
fi

# Загрузка публичных ключей
source demo/demo-keys/public_keys.txt

# Шаг 1: Запуск сервера
log_step "🖥️  Запуск сервера"

export CHAINDOCS_AUTH_TOKEN="demo_token"
log_info "Запуск сервера на порту 8080..."
./demo/bin/server > demo/demo_logs/server.log 2>&1 &
SERVER_PID=$!
echo $SERVER_PID > demo/demo_logs/server.pid
log_ok "Сервер запущен (PID: $SERVER_PID)"

# Ожидание запуска сервера
log_info "Ожидание запуска сервера..."
for i in {1..10}; do
    if curl -s http://localhost:8080/api/blocks/last > /dev/null 2>&1; then
        log_ok "Сервер готов"
        break
    fi
    sleep 1
done

if ! curl -s http://localhost:8080/api/blocks/last > /dev/null 2>&1; then
    echo "❌ Сервер не запустился. Проверьте demo/demo_logs/server.log"
    exit 1
fi

# Шаг 2: Регистрация ключей на сервере
log_step "📝 Регистрация ключей на сервере"

log_info "Регистрация ключа клиента 1..."
curl -s -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"$CLIENT1_PUBLIC_KEY\"}" > /dev/null
log_ok "Клиент 1 зарегистрирован"

log_info "Регистрация ключа клиента 2..."
curl -s -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"$CLIENT2_PUBLIC_KEY\"}" > /dev/null
log_ok "Клиент 2 зарегистрирован"

log_info "Регистрация ключа клиента 3..."
curl -s -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"$CLIENT3_PUBLIC_KEY\"}" > /dev/null
log_ok "Клиент 3 зарегистрирован"

log_ok "Все 3 клиента зарегистрированы на сервере"

# Шаг 3: Запуск клиентов-демонов
log_step "👥 Запуск клиентов-демонов (P2P + WebSocket)"

log_info "Запуск клиента 1..."
CHAINDOCS_CLIENT1_PASSWORD="demo123" ./demo/bin/client -config demo/client1-config.json > demo/demo_logs/client1.log 2>&1 &
CLIENT1_PID=$!
echo $CLIENT1_PID > demo/demo_logs/client1.pid
log_ok "Клиент 1 запущен (PID: $CLIENT1_PID, P2P порт: 9001)"

log_info "Запуск клиента 2..."
CHAINDOCS_CLIENT2_PASSWORD="demo123" ./demo/bin/client -config demo/client2-config.json > demo/demo_logs/client2.log 2>&1 &
CLIENT2_PID=$!
echo $CLIENT2_PID > demo/demo_logs/client2.pid
log_ok "Клиент 2 запущен (PID: $CLIENT2_PID, P2P порт: 9002)"

log_info "Запуск клиента 3..."
CHAINDOCS_CLIENT3_PASSWORD="demo123" ./demo/bin/client -config demo/client3-config.json > demo/demo_logs/client3.log 2>&1 &
CLIENT3_PID=$!
echo $CLIENT3_PID > demo/demo_logs/client3.pid
log_ok "Клиент 3 запущен (PID: $CLIENT3_PID, P2P порт: 9003)"

# Ожидание подключения
sleep 10

# Шаг 4: Проверка состояния
log_step "📊 Проверка состояния системы"

# Простая проверка без subshell
if curl -s http://localhost:8080/api/blocks/last > /dev/null 2>&1; then
    log_ok "Сервер работает"
else
    log_error "Сервер не отвечает"
fi

# Шаг 5: Тестовая загрузка документа
log_step "📄 Загрузка тестового документа"

log_info "Загрузка документа..."
curl -s -X POST http://localhost:8080/api/upload \
  -F "file=@demo/demo-test-document.pdf" > /tmp/upload_result.json

DOC_HASH=$(jq -r '.hash' /tmp/upload_result.json 2>/dev/null)
BLOCK_HASH=$(jq -r '.block_hash' /tmp/upload_result.json 2>/dev/null)

if [ -n "$DOC_HASH" ] && [ "$DOC_HASH" != "null" ]; then
    log_ok "Документ загружен!"
    log_info "Хэш документа: ${DOC_HASH:0:32}..."
    log_info "Хэш блока: ${BLOCK_HASH:0:32}..."
    
    # Ожидание подписей клиентов через P2P
    log_step "✍️  Ожидание подписей клиентов (P2P gossip)"
    
    for i in {1..10}; do
        sleep 2
        curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus" > /tmp/consensus.json
        SIGS=$(jq -r '.signatures' /tmp/consensus.json 2>/dev/null)
        REQUIRED=$(jq -r '.required' /tmp/consensus.json 2>/dev/null)
        REACHED=$(jq -r '.consensus_reached' /tmp/consensus.json 2>/dev/null)
        
        log_info "Прогресс: $SIGS/$REQUIRED"
        
        if [ "$REACHED" = "true" ]; then
            log_ok "🎉 Консенсус достигнут!"
            break
        fi
    done
    
    # Финальная проверка
    curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH" > /tmp/block.json
    SIGNATURES=$(jq -r '.signatures | length' /tmp/block.json 2>/dev/null)
    log_ok "Блок подписан: $SIGNATURES подписей(и)"
else
    log_warn "Не удалось загрузить документ"
fi

# Финальный вывод
echo ""
log_step "✅ Демонстрационная среда запущена!"
echo ""
echo "┌─────────────────────────────────────────────────────────────┐"
echo "│  📊 Статус системы                                          │"
echo "├─────────────────────────────────────────────────────────────┤"
echo "│  Сервер:       http://localhost:8080                        │"
echo "│  Веб-интерфейс: http://localhost:8080/web/login?token=demo_token"
echo "│  API:          http://localhost:8080/api/blocks/last        │"
echo "├─────────────────────────────────────────────────────────────┤"
echo "│  P2P Архитектура:                                           │"
echo "│  - Клиент 1:   PID $CLIENT1_PID (P2P порт: 9001)                     │"
echo "│  - Клиент 2:   PID $CLIENT2_PID (P2P порт: 9002)                     │"
echo "│  - Клиент 3:   PID $CLIENT3_PID (P2P порт: 9003)                     │"
echo "├─────────────────────────────────────────────────────────────┤"
echo "│  Зарегистрировано ключей: $KEYS                              │"
echo "│  Блоков в цепочке: $BLOCKS                                   │"
echo "└─────────────────────────────────────────────────────────────┘"
echo ""
echo "📝 Команды для управления:"
echo ""
echo "  • Загрузить новый документ:"
echo "    curl -X POST http://localhost:8080/api/upload -F \"file=@ваш_файл.pdf\""
echo ""
echo "  • Проверить последний блок:"
echo "    curl -s http://localhost:8080/api/blocks/last | jq"
echo ""
echo "  • Проверить P2P подключения:"
echo "    grep \"P2P\" demo/demo_logs/client1.log"
echo ""
echo "  • Остановить демонстрацию:"
echo "    ./demo/demo-stop.sh"
echo ""
echo "  • Полная очистка:"
echo "    ./demo/demo-prepare.sh (заново подготовить)"
echo ""
log_ok "Готово к работе!"
echo ""

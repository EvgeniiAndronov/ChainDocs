#!/bin/bash
# demo-scenario.sh - Сценарий демонстрации ChainDocs

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

echo_step() { echo -e "\n${CYAN}══════════════════════════════════════${NC}"; echo -e "${CYAN}║${NC} $1"; echo -e "${CYAN}══════════════════════════════════════${NC}\n"; }
echo_info() { echo -e "${BLUE}ℹ️${NC} $1"; }
echo_ok() { echo -e "${GREEN}✅${NC} $1"; }
echo_warn() { echo -e "${YELLOW}⚠️${NC} $1"; }

# Проверка что сервер запущен
check_server() {
    if ! curl -s http://localhost:8080/api/blocks/last > /dev/null; then
        echo_warn "Сервер не отвечает. Запустите: ./demo/demo-run.sh"
        exit 1
    fi
}

# Начало
echo_step "1. Проверка системы"

check_server
echo_ok "Сервер работает"

# Получение информации о системе
BLOCKS=$(curl -s http://localhost:8080/api/blocks | jq 'length')
KEYS=$(curl -s http://localhost:8080/api/keys | jq '.count')
echo_info "Блоков в цепочке: $BLOCKS"
echo_info "Зарегистрировано ключей: $KEYS"

# Шаг 2
echo_step "2. Просмотр последнего блока"

LAST_BLOCK=$(curl -s http://localhost:8080/api/blocks/last)
echo_info "Последний блок:"
echo "$LAST_BLOCK" | jq '{
  height: .height,
  hash: (.hash[0:16] + "..."),
  signatures: (.signatures | length),
  timestamp: .timestamp
}'

# Шаг 3
echo_step "3. Загрузка нового документа"

echo_info "Создание тестового документа..."
cat > /tmp/demo-contract.txt << EOF
DEMO CONTRACT
=============
Date: $(date)
This is a demonstration document for ChainDocs blockchain system.
Document ID: $(openssl rand -hex 8)
EOF

echo_ok "Документ создан"

echo_info "Загрузка документа в блокчейн..."
UPLOAD_RESULT=$(curl -s -X POST http://localhost:8080/api/upload -F "file=@/tmp/demo-contract.txt")

DOC_HASH=$(echo "$UPLOAD_RESULT" | jq -r '.hash')
BLOCK_HASH=$(echo "$UPLOAD_RESULT" | jq -r '.block_hash')

echo_ok "Документ загружен!"
echo_info "Хэш документа: ${DOC_HASH:0:32}..."
echo_info "Хэш блока: ${BLOCK_HASH:0:32}..."

# Шаг 4
echo_step "4. Ожидание подписей клиентов"

echo_info "Клиенты подписывают блок..."
for i in {1..5}; do
    sleep 1
    SIGS=$(curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus" | jq -r '.signatures')
    REQUIRED=$(curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus" | jq -r '.required')
    PERCENT=$(curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus" | jq -r '.percent')
    echo_info "Прогресс: $SIGS/$REQUIRED ($PERCENT%)"
    
    if [ "$SIGS" -ge "$REQUIRED" ]; then
        echo_ok "Консенсус достигнут!"
        break
    fi
done

# Шаг 5
echo_step "5. Проверка статуса консенсуса"

CONSENSUS=$(curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus")
echo_info "Статус консенсуса:"
echo "$CONSENSUS" | jq '{
  signatures: .signatures,
  required: .required,
  percent: .percent,
  consensus_reached: .consensus_reached,
  active_keys: .active_keys
}'

# Шаг 6
echo_step "6. Просмотр подписей"

BLOCK=$(curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH")
echo_info "Подписи в блоке:"
echo "$BLOCK" | jq -r '.signatures[] | "  • \(.public_key[0:16])... at \(.timestamp)"'

# Шаг 7
echo_step "7. Проверка категорий"

echo_info "Создание категории для дипломов..."
curl -s -X POST http://localhost:8080/api/categories \
  -H "Content-Type: application/json" \
  -d '{"id":"diplomas","name":"Дипломы","description":"Дипломы студентов"}' > /dev/null

echo_ok "Категория создана"

CATEGORIES=$(curl -s http://localhost:8080/api/categories | jq '.count')
echo_info "Всего категорий: $CATEGORIES"

# Шаг 8
echo_step "8. Массовая загрузка (Bulk Upload)"

echo_info "Создание нескольких документов..."
for i in {1..3}; do
    echo "Document $i - $(date)" > /tmp/demo-doc-$i.txt
done

echo_info "Загрузка 3 документов..."
BULK_RESULT=$(curl -s -X POST http://localhost:8080/api/upload/bulk \
  -F "files=@/tmp/demo-doc-1.txt" \
  -F "files=@/tmp/demo-doc-2.txt" \
  -F "files=@/tmp/demo-doc-3.txt" \
  -F "category=diplomas")

TOTAL=$(echo "$BULK_RESULT" | jq '.total')
SUCCESS=$(echo "$BULK_RESULT" | jq '.success')
echo_ok "Загружено: $SUCCESS/$TOTAL документов"

# Шаг 9
echo_step "9. Проверка метрик"

echo_info "Метрики Prometheus:"
curl -s http://localhost:8080/metrics | grep "^chaindocs_" | head -5

# Шаг 10
echo_step "10. Финальная статистика"

FINAL_BLOCKS=$(curl -s http://localhost:8080/api/blocks | jq 'length')
FINAL_KEYS=$(curl -s http://localhost:8080/api/keys | jq '.count')
ACTIVE_KEYS=$(curl -s http://localhost:8080/api/keys/active | jq '.count')

echo_info "📊 Итоговая статистика:"
echo "  Blocks: $FINAL_BLOCKS"
echo "  Registered Keys: $FINAL_KEYS"
echo "  Active Keys (24h): $ACTIVE_KEYS"
echo ""

echo_ok "✅ Демонстрация завершена!"
echo ""
echo "Следующие шаги:"
echo "  • Посетите веб-интерфейс: http://localhost:8080/web/login?token=demo_token"
echo "  • Проверьте API документацию: http://localhost:8080/"
echo "  • Для остановки: ./demo/demo-stop.sh"
echo ""

# Очистка временных файлов
rm -f /tmp/demo-*.txt

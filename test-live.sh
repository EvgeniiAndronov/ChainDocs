#!/bin/bash
# test-live.sh - Полное боевое тестирование ChainDocs
# Тестирует: сервер, генерацию ключей, регистрацию, загрузку документов, консенсус

set -e
cd "$(dirname "$0")"

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok() { echo -e "${GREEN}✅${NC} $1"; }
log_error() { echo -e "${RED}❌${NC} $1"; }
log_warn() { echo -e "${YELLOW}⚠️${NC} $1"; }

echo ""
echo "════════════════════════════════════════"
echo "🧪 ChainDocs Live Tests"
echo "════════════════════════════════════════"
echo ""

# Счётчик тестов
TESTS_PASSED=0
TESTS_FAILED=0

# Очистка
cleanup() {
    log_info "Очистка..."
    kill $SERVER_PID 2>/dev/null || true
    kill $CLIENT1_PID 2>/dev/null || true
    kill $CLIENT2_PID 2>/dev/null || true
    kill $CLIENT3_PID 2>/dev/null || true
    rm -f test_blockchain.db client*.enc test.pdf /tmp/key*.log 2>/dev/null || true
}

trap cleanup EXIT

# ==================== ТЕСТ 1: Запуск сервера ====================
echo "════════════════════════════════════════"
echo "Тест 1: Запуск сервера"
echo "════════════════════════════════════════"

log_info "Запуск сервера..."
export CHAINDOCS_DB="test_blockchain.db"
./bin/server > /tmp/server.log 2>&1 &
SERVER_PID=$!

sleep 3

if curl -s http://localhost:8080/api/blocks/last > /dev/null; then
    log_ok "Сервер запущен и отвечает"
    ((TESTS_PASSED++))
else
    log_error "Сервер не запустился"
    cat /tmp/server.log
    ((TESTS_FAILED++))
    exit 1
fi

# ==================== ТЕСТ 2: Генерация ключей ====================
echo ""
echo "════════════════════════════════════════"
echo "Тест 2: Генерация ключей"
echo "════════════════════════════════════════"

./bin/keygen -password pass1 -out client1.enc > /tmp/key1.log 2>&1
./bin/keygen -password pass2 -out client2.enc > /tmp/key2.log 2>&1
./bin/keygen -password pass3 -out client3.enc > /tmp/key3.log 2>&1

PUB1=$(grep "Public key" /tmp/key1.log | sed 's/.*: //')
PUB2=$(grep "Public key" /tmp/key2.log | sed 's/.*: //')
PUB3=$(grep "Public key" /tmp/key3.log | sed 's/.*: //')

if [ -n "$PUB1" ] && [ -n "$PUB2" ] && [ -n "$PUB3" ]; then
    log_ok "Ключи сгенерированы"
    log_info "Клиент 1: ${PUB1:0:32}..."
    log_info "Клиент 2: ${PUB2:0:32}..."
    log_info "Клиент 3: ${PUB3:0:32}..."
    ((TESTS_PASSED++))
else
    log_error "Не удалось сгенерировать ключи"
    ((TESTS_FAILED++))
fi

# ==================== ТЕСТ 3: Регистрация ключей ====================
echo ""
echo "════════════════════════════════════════"
echo "Тест 3: Регистрация ключей"
echo "════════════════════════════════════════"

curl -s -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"$PUB1\"}" > /dev/null

curl -s -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"$PUB2\"}" > /dev/null

curl -s -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"$PUB3\"}" > /dev/null

KEYS_COUNT=$(curl -s http://localhost:8080/api/keys | jq -r '.count')

if [ "$KEYS_COUNT" -ge 3 ]; then
    log_ok "Зарегистрировано ключей: $KEYS_COUNT"
    ((TESTS_PASSED++))
else
    log_error "Ожидалось >= 3 ключа, получено: $KEYS_COUNT"
    ((TESTS_FAILED++))
fi

# ==================== ТЕСТ 4: Загрузка документа ====================
echo ""
echo "════════════════════════════════════════"
echo "Тест 4: Загрузка документа"
echo "════════════════════════════════════════"

echo "%PDF-1.4" > test.pdf
echo "Test Document for ChainDocs" >> test.pdf
echo "%%EOF" >> test.pdf

UPLOAD_RESPONSE=$(curl -s -X POST http://localhost:8080/api/upload \
  -F "file=@test.pdf")

BLOCK_HASH=$(echo "$UPLOAD_RESPONSE" | jq -r '.block_hash')
DOC_HASH=$(echo "$UPLOAD_RESPONSE" | jq -r '.hash')

if [ -n "$BLOCK_HASH" ] && [ "$BLOCK_HASH" != "null" ]; then
    log_ok "Документ загружен"
    log_info "Хэш документа: ${DOC_HASH:0:32}..."
    log_info "Хэш блока: ${BLOCK_HASH:0:32}..."
    ((TESTS_PASSED++))
else
    log_error "Не удалось загрузить документ"
    ((TESTS_FAILED++))
fi

# ==================== ТЕСТ 5: Запуск клиентов ====================
echo ""
echo "════════════════════════════════════════"
echo "Тест 5: Запуск клиентов-демонов"
echo "════════════════════════════════════════"

# Создаём конфиги для клиентов с правильным интервалом
cat > client1-config.json << EOF
{
  "server": "http://localhost:8080",
  "key_file": "client1.enc",
  "password_env": "CHAINDOCS_CLIENT1_PASSWORD",
  "mode": "daemon",
  "daemon": {
    "interval": "3s",
    "sign_unsigned_only": false,
    "stop_on_consensus": false
  }
}
EOF

cat > client2-config.json << EOF
{
  "server": "http://localhost:8080",
  "key_file": "client2.enc",
  "password_env": "CHAINDOCS_CLIENT2_PASSWORD",
  "mode": "daemon",
  "daemon": {
    "interval": "3s",
    "sign_unsigned_only": false,
    "stop_on_consensus": false
  }
}
EOF

cat > client3-config.json << EOF
{
  "server": "http://localhost:8080",
  "key_file": "client3.enc",
  "password_env": "CHAINDOCS_CLIENT3_PASSWORD",
  "mode": "daemon",
  "daemon": {
    "interval": "3s",
    "sign_unsigned_only": false,
    "stop_on_consensus": false
  }
}
EOF

export CHAINDOCS_CLIENT1_PASSWORD="pass1"
export CHAINDOCS_CLIENT2_PASSWORD="pass2"
export CHAINDOCS_CLIENT3_PASSWORD="pass3"

./bin/client -config client1-config.json > /tmp/client1.log 2>&1 &
CLIENT1_PID=$!

./bin/client -config client2-config.json > /tmp/client2.log 2>&1 &
CLIENT2_PID=$!

./bin/client -config client3-config.json > /tmp/client3.log 2>&1 &
CLIENT3_PID=$!

log_ok "Клиенты запущены (PID: $CLIENT1_PID, $CLIENT2_PID, $CLIENT3_PID)"

# Даём клиентам время подключиться и начать работу
log_info "Ожидание подключения клиентов..."
sleep 10

# ==================== ТЕСТ 6: Проверка консенсуса ====================
echo ""
echo "════════════════════════════════════════"
echo "Тест 6: Проверка консенсуса"
echo "════════════════════════════════════════"

CONSENSUS=$(curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus" | jq)
SIGNATURES=$(echo "$CONSENSUS" | jq -r '.signatures')
REQUIRED=$(echo "$CONSENSUS" | jq -r '.required')
PERCENT=$(echo "$CONSENSUS" | jq -r '.percent')
REACHED=$(echo "$CONSENSUS" | jq -r '.consensus_reached')

log_info "Прогресс: $SIGNATURES/$REQUIRED ($PERCENT%)"
log_info "Консенсус достигнут: $REACHED"

if [ "$REACHED" = "true" ] && [ "$SIGNATURES" -ge 2 ]; then
    log_ok "Консенсус достигнут ($SIGNATURES из $REQUIRED)"
    ((TESTS_PASSED++))
else
    log_error "Консенсус не достигнут ($SIGNATURES из $REQUIRED)"
    ((TESTS_FAILED++))
fi

# ==================== ТЕСТ 7: Проверка блока ====================
echo ""
echo "════════════════════════════════════════"
echo "Тест 7: Проверка блока"
echo "════════════════════════════════════════"

BLOCK=$(curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH" | jq)
BLOCK_SIGS=$(echo "$BLOCK" | jq -r '.signatures | length')

log_info "Подписей в блоке: $BLOCK_SIGS"

if [ "$BLOCK_SIGS" -ge 2 ]; then
    log_ok "Блок подписан ($BLOCK_SIGS подписей)"
    ((TESTS_PASSED++))
else
    log_error "Недостаточно подписей в блоке"
    ((TESTS_FAILED++))
fi

# ==================== ТЕСТ 8: Загрузка второго документа ====================
echo ""
echo "════════════════════════════════════════"
echo "Тест 8: Второй документ (автоматическая подпись)"
echo "════════════════════════════════════════"

echo "%PDF-1.4" > test2.pdf
echo "Second Test Document" >> test2.pdf
echo "%%EOF" >> test2.pdf

UPLOAD_RESPONSE2=$(curl -s -X POST http://localhost:8080/api/upload \
  -F "file=@test2.pdf")

BLOCK_HASH2=$(echo "$UPLOAD_RESPONSE2" | jq -r '.block_hash')
log_info "Второй блок создан: ${BLOCK_HASH2:0:32}..."

# Даём клиентам время подписать второй блок
sleep 10

CONSENSUS2=$(curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH2/consensus" | jq)
SIGNATURES2=$(echo "$CONSENSUS2" | jq -r '.signatures')
REACHED2=$(echo "$CONSENSUS2" | jq -r '.consensus_reached')

if [ "$REACHED2" = "true" ] && [ "$SIGNATURES2" -ge 2 ]; then
    log_ok "Второй блок автоматически подписан"
    ((TESTS_PASSED++))
else
    log_error "Второй блок не подписан"
    ((TESTS_FAILED++))
fi

# ==================== ИТОГИ ====================
echo ""
echo "════════════════════════════════════════"
echo "📊 Итоги тестирования"
echo "════════════════════════════════════════"
echo ""
echo "Пройдено тестов: $TESTS_PASSED"
echo "Провалено тестов: $TESTS_FAILED"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo "${GREEN}✅ Все тесты пройдены!${NC}"
    exit 0
else
    echo "${RED}❌ Часть тестов провалена${NC}"
    exit 1
fi

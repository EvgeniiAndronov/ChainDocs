#!/bin/bash
# test-live.sh - Боевое тестирование ChainDocs
# Проверка работы сервера с несколькими клиентами

set -e

cd "$(dirname "$0")"

echo "🧪 Боевое тестирование ChainDocs"
echo "=========================================="
echo ""

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_ok() { echo -e "${GREEN}✅ $1${NC}"; }
log_err() { echo -e "${RED}❌ $1${NC}"; }
log_info() { echo -e "${YELLOW}ℹ️  $1${NC}"; }

# Очистка старых тестовых файлов
rm -f test_blockchain.db test_*.enc test.pdf
rm -f client1.enc client2.enc client3.enc

# Шаг 1: Запуск сервера
log_info "Запуск сервера..."
export CHAINDOCS_DB="test_blockchain.db"
./bin/server > /tmp/server.log 2>&1 &
SERVER_PID=$!
sleep 2

# Проверка сервера
if curl -s http://localhost:8080/api/blocks/last > /dev/null 2>&1; then
    log_ok "Сервер запущен (PID: $SERVER_PID)"
else
    log_err "Сервер не запустился!"
    cat /tmp/server.log
    exit 1
fi

# Шаг 2: Генерация 3 ключей
log_info "Генерация ключей для 3 клиентов..."

./bin/keygen -password pass1 -out client1.enc > /tmp/key1.log 2>&1
./bin/keygen -password pass2 -out client2.enc > /tmp/key2.log 2>&1
./bin/keygen -password pass3 -out client3.enc > /tmp/key3.log 2>&1

# Извлекаем публичные ключи (последнее слово в строке с "Public key")
PUB1=$(grep "Public key" /tmp/key1.log | sed 's/.*: //')
PUB2=$(grep "Public key" /tmp/key2.log | sed 's/.*: //')
PUB3=$(grep "Public key" /tmp/key3.log | sed 's/.*: //')

log_ok "Клиент 1: ${PUB1:0:16}..."
log_ok "Клиент 2: ${PUB2:0:16}..."
log_ok "Клиент 3: ${PUB3:0:16}..."

# Шаг 3: Регистрация ключей
log_info "Регистрация ключей на сервере..."

curl -s -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"${PUB1}\"}" > /tmp/reg1.log 2>&1

curl -s -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"${PUB2}\"}" > /tmp/reg2.log 2>&1

curl -s -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"${PUB3}\"}" > /tmp/reg3.log 2>&1

# Проверка регистрации
cat /tmp/reg1.log
cat /tmp/reg2.log
cat /tmp/reg3.log

KEYS_COUNT=$(curl -s http://localhost:8080/api/keys | jq -r '.count')
log_ok "Зарегистрировано ключей: $KEYS_COUNT"

# Шаг 4: Загрузка тестового документа
log_info "Загрузка тестового документа..."

echo "Test Document - ChainDocs Live Test $(date)" > test.pdf

UPLOAD_RESPONSE=$(curl -s -X POST http://localhost:8080/api/upload \
  -F "file=@test.pdf")

BLOCK_HASH=$(echo "$UPLOAD_RESPONSE" | jq -r '.block_hash')
DOC_HASH=$(echo "$UPLOAD_RESPONSE" | jq -r '.hash')

log_ok "Документ загружен"
log_info "Хэш документа: ${DOC_HASH:0:16}..."
log_info "Хэш блока: ${BLOCK_HASH:0:16}..."

# Шаг 5: Проверка блока до подписи
log_info "Проверка блока до подписи..."

BLOCK_BEFORE=$(curl -s http://localhost:8080/api/blocks/last)
SIGS_BEFORE=$(echo "$BLOCK_BEFORE" | jq -r '.signatures | length')
log_info "Подписей до: $SIGS_BEFORE"

# Шаг 6: Запуск клиентов (подписание)
log_info "Подписание блока клиентами..."

log_info "Клиент 1 подписывает..."
CHAINDOCS_KEY_PASSWORD=pass1 ./bin/client -key client1.enc -mode oneshot 2>&1 | tee /tmp/client1.log

sleep 1

log_info "Клиент 2 подписывает..."
CHAINDOCS_KEY_PASSWORD=pass2 ./bin/client -key client2.enc -mode oneshot 2>&1 | tee /tmp/client2.log

sleep 1

log_info "Клиент 3 подписывает..."
CHAINDOCS_KEY_PASSWORD=pass3 ./bin/client -key client3.enc -mode oneshot 2>&1 | tee /tmp/client3.log

# Шаг 7: Проверка консенсуса
echo ""
log_info "Проверка консенсуса..."
sleep 1

CONSENSUS_RESPONSE=$(curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus")

SIGNATURES=$(echo "$CONSENSUS_RESPONSE" | jq -r '.signatures')
REQUIRED=$(echo "$CONSENSUS_RESPONSE" | jq -r '.required')
PERCENT=$(echo "$CONSENSUS_RESPONSE" | jq -r '.percent')
CONSENSUS_REACHED=$(echo "$CONSENSUS_RESPONSE" | jq -r '.consensus_reached')

echo ""
echo "📊 Статус консенсуса:"
echo "   Подписей: $SIGNATURES"
echo "   Требуется: $REQUIRED"
echo "   Процент: $PERCENT%"
echo "   Консенсус: $CONSENSUS_REACHED"
echo ""

if [ "$CONSENSUS_REACHED" = "true" ]; then
    log_ok "КОНСЕНСУС ДОСТИГНУТ! ($SIGNATURES/$REQUIRED)"
else
    log_err "Консенсус не достигнут ($SIGNATURES/$REQUIRED)"
fi

# Шаг 8: Проверка блока после подписи
log_info "Проверка финального состояния блока..."

BLOCK_AFTER=$(curl -s http://localhost:8080/api/blocks/last)
SIGS_AFTER=$(echo "$BLOCK_AFTER" | jq -r '.signatures | length')

log_ok "Подписей после: $SIGS_AFTER"

# Проверка всех подписей
echo ""
log_info "Список подписей:"
echo "$BLOCK_AFTER" | jq -r '.signatures[] | "  - \(.public_key[0:16])... at \(.timestamp)"'

# Шаг 9: Проверка всех блоков
echo ""
log_info "Состояние блокчейна:"
TOTAL_BLOCKS=$(curl -s http://localhost:8080/api/blocks | jq 'length')
log_ok "Всего блоков: $TOTAL_BLOCKS"

# Финальный отчёт
echo ""
echo "=========================================="
echo "📋 ИТОГОВЫЙ ОТЧЁТ"
echo "=========================================="
echo ""

TESTS_PASSED=0
TESTS_TOTAL=6

# Тест 1: Сервер запущен
if curl -s http://localhost:8080/api/blocks/last > /dev/null; then
    log_ok "Сервер работает"
    ((TESTS_PASSED++))
else
    log_err "Сервер недоступен"
fi

# Тест 2: Ключи зарегистрированы
if [ "$KEYS_COUNT" -eq 3 ]; then
    log_ok "3 ключа зарегистрировано"
    ((TESTS_PASSED++))
else
    log_err "Ожидалось 3 ключа, получилось: $KEYS_COUNT"
fi

# Тест 3: Блок создан
if [ -n "$BLOCK_HASH" ] && [ "$BLOCK_HASH" != "null" ]; then
    log_ok "Блок создан"
    ((TESTS_PASSED++))
else
    log_err "Блок не создан"
fi

# Тест 4: Подписи добавлены
if [ "$SIGS_AFTER" -ge 2 ]; then
    log_ok "2+ подписей добавлено ($SIGS_AFTER)"
    ((TESTS_PASSED++))
else
    log_err "Ожидалось 2+ подписи, получилось: $SIGS_AFTER"
fi

# Тест 5: Консенсус достигнут
if [ "$CONSENSUS_REACHED" = "true" ]; then
    log_ok "Консенсус 51%+ достигнут"
    ((TESTS_PASSED++))
else
    log_err "Консенсус не достигнут"
fi

# Тест 6: Блок валиден
BLOCK_HEIGHT=$(echo "$BLOCK_AFTER" | jq -r '.height')
if [ "$BLOCK_HEIGHT" -gt 0 ]; then
    log_ok "Блок в цепочке (height=$BLOCK_HEIGHT)"
    ((TESTS_PASSED++))
else
    log_err "Блок не в цепочке"
fi

echo ""
echo "=========================================="
echo "Результат: $TESTS_PASSED/$TESTS_TOTAL тестов пройдено"
echo "=========================================="

# Очистка
echo ""
log_info "Очистка..."
kill $SERVER_PID 2>/dev/null || true
rm -f /tmp/key1.log /tmp/key2.log /tmp/key3.txt test.pdf client1.enc client2.enc client3.enc
rm -f /tmp/reg1.log /tmp/reg2.log /tmp/reg3.log /tmp/client1.log /tmp/client2.log /tmp/client3.log
# blockchain.db не удаляем - можно посмотреть

if [ $TESTS_PASSED -eq $TESTS_TOTAL ]; then
    log_ok "Все тесты пройдены! 🎉"
    exit 0
else
    log_err "Некоторые тесты не пройдены"
    exit 1
fi

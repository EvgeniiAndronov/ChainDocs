#!/bin/bash
# test-live.sh - Боевое тестирование ChainDocs

set -e
cd "$(dirname "$0")"

echo "🧪 Боевое тестирование ChainDocs"
echo "=========================================="

# Очистка
rm -f test_blockchain.db client1.enc client2.enc client3.enc

# Запуск сервера
export CHAINDOCS_DB="test_blockchain.db"
./bin/server > /tmp/server.log 2>&1 &
SERVER_PID=$!
sleep 2

if ! curl -s http://localhost:8080/api/blocks/last > /dev/null; then
    echo "❌ Сервер не запустился"
    exit 1
fi
echo "✅ Сервер запущен"

# Генерация 3 ключей
./bin/keygen -password pass1 -out client1.enc > /tmp/key1.log 2>&1
./bin/keygen -password pass2 -out client2.enc > /tmp/key2.log 2>&1
./bin/keygen -password pass3 -out client3.enc > /tmp/key3.log 2>&1

PUB1=$(grep "Public key" /tmp/key1.log | sed 's/.*: //')
PUB2=$(grep "Public key" /tmp/key2.log | sed 's/.*: //')
PUB3=$(grep "Public key" /tmp/key3.log | sed 's/.*: //')

echo "✅ Клиент 1: ${PUB1:0:16}..."
echo "✅ Клиент 2: ${PUB2:0:16}..."
echo "✅ Клиент 3: ${PUB3:0:16}..."

# Регистрация ключей
curl -s -X POST http://localhost:8080/api/register -H "Content-Type: application/json" -d "{\"public_key\":\"$PUB1\"}" > /dev/null
curl -s -X POST http://localhost:8080/api/register -H "Content-Type: application/json" -d "{\"public_key\":\"$PUB2\"}" > /dev/null
curl -s -X POST http://localhost:8080/api/register -H "Content-Type: application/json" -d "{\"public_key\":\"$PUB3\"}" > /dev/null

KEYS_COUNT=$(curl -s http://localhost:8080/api/keys | jq -r '.count')
echo "✅ Зарегистрировано ключей: $KEYS_COUNT"

# Загрузка документа
echo "Test Document" > test.pdf
UPLOAD_RESPONSE=$(curl -s -X POST http://localhost:8080/api/upload -F "file=@test.pdf")
BLOCK_HASH=$(echo "$UPLOAD_RESPONSE" | jq -r '.block_hash')
echo "✅ Блок создан: ${BLOCK_HASH:0:16}..."

# Подписание клиентами
CHAINDOCS_KEY_PASSWORD=pass1 ./bin/client -key client1.enc -mode oneshot 2>&1 | grep -E "(✅|🎉)" || true
sleep 1
CHAINDOCS_KEY_PASSWORD=pass2 ./bin/client -key client2.enc -mode oneshot 2>&1 | grep -E "(✅|🎉)" || true
sleep 1
CHAINDOCS_KEY_PASSWORD=pass3 ./bin/client -key client3.enc -mode oneshot 2>&1 | grep -E "(✅|🎉)" || true

# Проверка консенсуса
sleep 1
CONSENSUS=$(curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus" | jq)
SIGNATURES=$(echo "$CONSENSUS" | jq -r '.signatures')
REQUIRED=$(echo "$CONSENSUS" | jq -r '.required')
REACHED=$(echo "$CONSENSUS" | jq -r '.consensus_reached')

echo ""
echo "📊 Консенсус: $SIGNATURES/$REQUIRED (достигнут: $REACHED)"

# Итог
TESTS=0
[ "$KEYS_COUNT" -eq 3 ] && ((TESTS++)) || echo "❌ Ключи"
[ -n "$BLOCK_HASH" ] && ((TESTS++)) || echo "❌ Блок"
[ "$SIGNATURES" -ge 1 ] && ((TESTS++)) || echo "❌ Подписи"
[ "$REACHED" = "true" ] && ((TESTS++)) || echo "❌ Консенсус"

echo ""
echo "=========================================="
echo "Результат: $TESTS/4 тестов пройдено"
echo "=========================================="

kill $SERVER_PID 2>/dev/null || true
rm -f /tmp/key*.log test.pdf client*.enc

[ $TESTS -eq 4 ] && echo "✅ Все тесты пройдены!" || echo "⚠️  Часть тестов не пройдена"
exit 0

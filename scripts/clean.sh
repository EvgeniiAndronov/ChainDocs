#!/bin/bash
# clean.sh - Очистка тестовых данных ChainDocs

set -e

echo "🧹 Очистка тестовых данных ChainDocs"
echo "===================================="
echo ""

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }
log_ok() { echo -e "${GREEN}✅ $1${NC}"; }
log_err() { echo -e "${RED}❌ $1${NC}"; }

# Проверка
if [ ! -f "go.mod" ]; then
    log_err "Запустите из корня проекта ChainDocs"
    exit 1
fi

echo "Будут удалены:"
echo "  - blockchain.db (база данных)"
echo "  - uploads/* (загруженные файлы)"
echo "  - *.enc (ключи)"
echo "  - test_*.db (тестовые БД)"
echo ""

read -p "Продолжить? (y/N): " confirm
if [[ ! $confirm =~ ^[Yy]$ ]]; then
    log_warn "Отменено"
    exit 0
fi

# Очистка
echo ""
log_warn "Начинаю очистку..."

# БД
if [ -f "blockchain.db" ]; then
    rm -f blockchain.db
    log_ok "Удалён blockchain.db"
fi

# Uploads
if [ -d "uploads" ]; then
    count=$(ls -1 uploads/*.pdf 2>/dev/null | wc -l)
    rm -rf uploads/*
    log_ok "Удалено файлов из uploads: $count"
fi

# Ключи
enc_count=$(ls -1 *.enc 2>/dev/null | wc -l)
rm -f *.enc
log_ok "Удалено ключей: $enc_count"

# Тестовые БД
test_count=$(ls -1 test_*.db 2>/dev/null | wc -l)
rm -f test_*.db
log_ok "Удалено тестовых БД: $test_count"

# Временные файлы
rm -f pub*.txt
rm -f /tmp/key*.log /tmp/reg*.log /tmp/client*.log /tmp/server*.log
log_ok "Удалены временные файлы"

# Bin (опционально)
if [ -d "bin" ]; then
    log_warn "bin/ директория сохранена (перекомпилируйте при необходимости)"
fi

echo ""
echo "===================================="
log_ok "Очистка завершена!"
echo ""
echo "Следующие шаги:"
echo "  1. Запустить сервер: make run"
echo "  2. Сгенерировать ключи: make keygen PASSWORD=xxx OUT=key.enc"
echo "  3. Зарегистрировать ключи на сервере"
echo "  4. Загрузить тестовый документ"
echo ""

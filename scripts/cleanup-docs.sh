#!/bin/bash
# cleanup-docs.sh - Очистка устаревших файлов документации

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

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

echo ""
log_info "🧹 Очистка устаревших файлов документации..."
echo ""

# Создаём директорию для архива
mkdir -p docs/archive

# Файлы для перемещения в архив
ARCHIVE_FILES=(
    "AUDIT_SUMMARY.md"
    "CHANGES.md"
    "CLEANUP_SUMMARY.md"
    "DOCUMENT_SIGNATURE.md"
    "DYNAMIC_CONSENSUS.md"
    "FINAL_RELEASE.md"
    "FULL_DOCUMENTATION.md"
    "GODOC.md"
    "HYBRID_ARCHITECTURE.md"
    "PRESENTATION.md"
    "README_FINAL.md"
    "TODO_ANALYSIS.md"
)

# Перемещаем в архив
log_info "Перемещение устаревших файлов в docs/archive/..."
for file in "${ARCHIVE_FILES[@]}"; do
    if [ -f "$file" ]; then
        mv "$file" docs/archive/
        log_ok "  $file → docs/archive/"
    fi
done

# Файлы для удаления
DELETE_FILES=(
    "client1-config.json"
    "client2-config.json"
    "client3-config.json"
    "config.json"
    "document.pdf"
    "test2.pdf"
)

# Удаляем
log_info "Удаление временных файлов..."
for file in "${DELETE_FILES[@]}"; do
    if [ -f "$file" ]; then
        rm -f "$file"
        log_ok "  Удалён: $file"
    fi
done

# Директории для очистки
log_info "Очистка директорий..."
rm -rf uploads/*
log_ok "  uploads/ очищен"

rm -rf bin/*
log_ok "  bin/ очищен"

# Typst файлы (оставляем только актуальные)
log_info "Проверка Typst файлов..."
if [ -f "docs/chaindocs.typ — копия" ]; then
    rm -f "docs/chaindocs.typ — копия"
    log_ok "  Удалена копия chaindocs.typ"
fi

# Итог
echo ""
log_ok "✅ Очистка завершена!"
echo ""
log_info "📁 Новая структура документации:"
echo ""
echo "docs/"
echo "├── ARCHITECTURE.md      # Основная архитектура"
echo "├── SERVER.md            # Сервер"
echo "├── CLIENT.md            # Клиент"
echo "├── P2P_PROTOCOL.md      # P2P протокол"
echo "├── CATEGORIES_AND_BULK_UPLOAD.md"
echo "├── WEB_UI_CATEGORIES_BULK.md"
echo "├── archive/             # Устаревшая документация"
echo "│   ├── AUDIT_SUMMARY.md"
echo "│   ├── CHANGES.md"
echo "│   └── ..."
echo "└── *.typ, *.pdf         # Typst документы"
echo ""
echo "demo/"
echo "├── README.md            # Демо документация"
echo "├── QUICKSTART.md        # Быстрый старт"
echo "└── *.sh                 # Скрипты"
echo ""
echo "Корень:"
echo "├── README.md            # Главная документация"
echo "├── INSTALL.md           # Установка"
echo "├── PRODUCTION.md        # Production"
echo "└── Makefile             # Команды"
echo ""

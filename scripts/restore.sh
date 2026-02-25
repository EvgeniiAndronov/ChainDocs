#!/bin/bash
# restore.sh - Восстановление ChainDocs из backup

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Конфигурация
DB_PATH="${CHAINDOCS_DB:-$PROJECT_DIR/blockchain.db}"
BACKUP_DIR="${CHAINDOCS_BACKUP_DIR:-$PROJECT_DIR/backups}"

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -f, --file FILE    Backup file to restore from"
    echo "  -l, --list         List available backups"
    echo "  -h, --help         Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 --list"
    echo "  $0 --file blockchain_backup_20260222_120000.db.gz"
    exit 1
}

# Парсинг аргументов
BACKUP_FILE=""
LIST_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--file)
            BACKUP_FILE="$2"
            shift 2
            ;;
        -l|--list)
            LIST_ONLY=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            ;;
    esac
done

# Список backup
if [ "$LIST_ONLY" = true ]; then
    log_info "Available backups in $BACKUP_DIR:"
    echo ""
    ls -lh "$BACKUP_DIR"/blockchain_backup_*.db* 2>/dev/null || echo "No backups found"
    exit 0
fi

# Проверка файла
if [ -z "$BACKUP_FILE" ]; then
    log_error "Backup file not specified. Use --file or --list"
    usage
fi

BACKUP_PATH="$BACKUP_DIR/$BACKUP_FILE"
if [ ! -f "$BACKUP_PATH" ]; then
    log_error "Backup file not found: $BACKUP_PATH"
    exit 1
fi

# Предупреждение
log_warn "This will overwrite the current database!"
log_warn "Current database: $DB_PATH"
log_warn "Backup file: $BACKUP_PATH"
echo ""
read -p "Continue? (y/N): " confirm
if [[ ! $confirm =~ ^[Yy]$ ]]; then
    log_warn "Cancelled"
    exit 0
fi

# Остановка сервера (если работает)
log_info "Stopping server (if running)..."
pkill -f "chaindocs.*server" 2>/dev/null || true
sleep 2

# Восстановление
log_info "Restoring backup..."

# Распаковка если нужно
if [[ "$BACKUP_FILE" == *.gz ]]; then
    log_info "Decompressing backup..."
    gunzip -c "$BACKUP_PATH" > "$DB_PATH"
else
    cp "$BACKUP_PATH" "$DB_PATH"
fi

# Проверка
if [ -f "$DB_PATH" ]; then
    log_info "Restore completed successfully!"
    log_info "Database: $DB_PATH"
    log_info "Size: $(du -h "$DB_PATH" | cut -f1)"
    echo ""
    log_info "You can now start the server"
else
    log_error "Restore failed!"
    exit 1
fi

#!/bin/bash
# backup.sh - Резервное копирование ChainDocs

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Конфигурация
DB_PATH="${CHAINDOCS_DB:-$PROJECT_DIR/blockchain.db}"
BACKUP_DIR="${CHAINDOCS_BACKUP_DIR:-$PROJECT_DIR/backups}"
RETENTION_DAYS="${CHAINDOCS_RETENTION_DAYS:-30}"

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Проверка БД
if [ ! -f "$DB_PATH" ]; then
    log_error "Database not found: $DB_PATH"
    exit 1
fi

# Создание директории backup
mkdir -p "$BACKUP_DIR"

# Имя файла backup
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/blockchain_backup_$TIMESTAMP.db"

log_info "Starting backup..."
log_info "Source: $DB_PATH"
log_info "Destination: $BACKUP_FILE"

# Копирование БД
cp "$DB_PATH" "$BACKUP_FILE"

# Сжатие
if command -v gzip &> /dev/null; then
    log_info "Compressing backup..."
    gzip "$BACKUP_FILE"
    BACKUP_FILE="$BACKUP_FILE.gz"
fi

# Информация о файле
BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
log_info "Backup completed: $BACKUP_FILE ($BACKUP_SIZE)"

# Очистка старых backup
log_info "Cleaning up backups older than $RETENTION_DAYS days..."
find "$BACKUP_DIR" -name "blockchain_backup_*.db*" -mtime +$RETENTION_DAYS -delete
DELETED_COUNT=$(find "$BACKUP_DIR" -name "blockchain_backup_*.db*" -mtime +$RETENTION_DAYS 2>/dev/null | wc -l)
log_info "Deleted $DELETED_COUNT old backups"

# Список текущих backup
echo ""
log_info "Current backups:"
ls -lh "$BACKUP_DIR"/blockchain_backup_*.db* 2>/dev/null | tail -5 || echo "No backups found"

echo ""
log_info "Backup successful!"

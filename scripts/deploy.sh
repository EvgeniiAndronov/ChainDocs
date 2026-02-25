#!/bin/bash
# deploy.sh - Развёртывание ChainDocs в Docker

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

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
    echo "  -p, --production   Deploy production stack (server + client + monitoring)"
    echo "  -s, --server-only  Deploy server only"
    echo "  -d, --down         Stop and remove all containers"
    echo "  -c, --clean        Clean all volumes and data"
    echo "  -h, --help         Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 --server-only"
    echo "  $0 --production"
    echo "  $0 --down"
    exit 1
}

# Проверка Docker
if ! command -v docker &> /dev/null; then
    log_error "Docker not found. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    log_error "docker-compose not found. Please install docker-compose first."
    exit 1
fi

# Парсинг аргументов
MODE="server"
ACTION="up"

while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--production)
            MODE="production"
            shift
            ;;
        -s|--server-only)
            MODE="server"
            shift
            ;;
        -d|--down)
            ACTION="down"
            shift
            ;;
        -c|--clean)
            ACTION="clean"
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

# Действия
if [ "$ACTION" = "down" ]; then
    log_warn "Stopping all containers..."
    docker-compose -f docker-compose.prod.yml down
    log_info "Containers stopped"
    exit 0
fi

if [ "$ACTION" = "clean" ]; then
    log_warn "Stopping containers and removing volumes..."
    docker-compose -f docker-compose.prod.yml down -v
    log_info "Containers and volumes removed"
    exit 0
fi

# Создание конфигов если нет
if [ ! -f "config.json" ]; then
    log_info "Creating default config.json..."
    cat > config.json << 'EOF'
{
  "port": 8080,
  "db_path": "/app/data/blockchain.db",
  "upload_dir": "/app/uploads",
  "log_file": "/app/logs/server.log",
  "log_level": "info",
  "consensus": {
    "type": "percentage",
    "percentage": 51,
    "min_signatures": 2,
    "use_active_keys": true
  },
  "activity": {
    "window": "24h",
    "auto_cleanup": true
  },
  "tls": {
    "enabled": false,
    "cert_file": "",
    "key_file": ""
  },
  "rate_limit": {
    "enabled": false,
    "requests_per_second": 10,
    "burst": 20
  }
}
EOF
    log_info "config.json created"
fi

# Создание директорий
mkdir -p backups logs

# Сборка образов
log_info "Building Docker images..."
if [ "$MODE" = "production" ]; then
    docker-compose -f docker-compose.prod.yml build
else
    docker-compose -f docker-compose.prod.yml build chaindocs-server
fi

# Запуск
log_info "Starting containers..."
if [ "$MODE" = "production" ]; then
    docker-compose -f docker-compose.prod.yml up -d
    log_info ""
    log_info "✅ Production stack deployed!"
    log_info ""
    log_info "Services:"
    log_info "  - ChainDocs Server: http://localhost:8080"
    log_info "  - ChainDocs Web UI: http://localhost:8080/web/"
    log_info "  - Prometheus:       http://localhost:9090"
    log_info "  - Grafana:          http://localhost:3000 (admin/admin)"
    log_info ""
    log_info "Logs:"
    log_info "  docker-compose -f docker-compose.prod.yml logs -f"
    log_info ""
    log_info "Stop:"
    log_info "  docker-compose -f docker-compose.prod.yml down"
else
    docker-compose -f docker-compose.prod.yml up -d chaindocs-server
    log_info ""
    log_info "✅ Server deployed!"
    log_info ""
    log_info "Services:"
    log_info "  - ChainDocs Server: http://localhost:8080"
    log_info "  - ChainDocs Web UI: http://localhost:8080/web/"
    log_info ""
    log_info "Logs:"
    log_info "  docker-compose logs -f chaindocs-server"
    log_info ""
    log_info "Stop:"
    log_info "  docker-compose -f docker-compose.prod.yml down"
fi

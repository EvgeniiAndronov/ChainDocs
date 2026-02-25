#!/bin/bash
# ChainDocs Client Installer
# Usage: sudo ./install-client.sh [options]

set -e

# Configuration
INSTALL_DIR="/opt/chaindocs"
BIN_NAME="chaindocs-client"
CONFIG_FILE="config.json"
SERVICE_NAME="chaindocs-client"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -b, --binary PATH    Path to client binary (default: ./bin/client)"
    echo "  -c, --config PATH    Path to config file (default: ./cmd/client/config.example.json)"
    echo "  -d, --daemon         Install as daemon (systemd on Linux, launchd on macOS)"
    echo "  -u, --uninstall      Uninstall instead of install"
    echo "  -h, --help           Show this help"
    echo ""
    echo "Examples:"
    echo "  sudo $0 -b ./bin/client -d"
    echo "  sudo $0 --uninstall"
}

# Parse arguments
BINARY_PATH="./bin/client"
CONFIG_PATH="./cmd/client/config.example.json"
INSTALL_DAEMON=false
UNINSTALL=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -b|--binary)
            BINARY_PATH="$2"
            shift 2
            ;;
        -c|--config)
            CONFIG_PATH="$2"
            shift 2
            ;;
        -d|--daemon)
            INSTALL_DAEMON=true
            shift
            ;;
        -u|--uninstall)
            UNINSTALL=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root (sudo)"
    exit 1
fi

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux*)
        INIT_SYSTEM="systemd"
        if ! command -v systemctl &> /dev/null; then
            log_warn "systemd not found, daemon installation may fail"
        fi
        ;;
    Darwin*)
        INIT_SYSTEM="launchd"
        ;;
    *)
        log_error "Unsupported OS: $OS"
        exit 1
        ;;
esac

log_info "Detected OS: $OS ($INIT_SYSTEM)"

# Uninstall function
do_uninstall() {
    log_info "Uninstalling ChainDocs Client..."
    
    # Stop service
    if [[ "$INIT_SYSTEM" == "systemd" ]]; then
        systemctl stop $SERVICE_NAME 2>/dev/null || true
        systemctl disable $SERVICE_NAME 2>/dev/null || true
        rm -f /etc/systemd/system/${SERVICE_NAME}.service
        systemctl daemon-reload
    elif [[ "$INIT_SYSTEM" == "launchd" ]]; then
        launchctl bootout gui/$(id -u)/com.chaindocs.client 2>/dev/null || true
        rm -f /Library/LaunchDaemons/com.chaindocs.client.plist
    fi
    
    # Remove installation directory
    rm -rf "$INSTALL_DIR"
    
    log_info "Uninstallation complete"
    exit 0
}

if [[ "$UNINSTALL" == true ]]; then
    do_uninstall
fi

# Check binary exists
if [[ ! -f "$BINARY_PATH" ]]; then
    log_error "Binary not found: $BINARY_PATH"
    log_info "Build it first with: go build -o bin/client cmd/client/main.go cmd/client/config.go"
    exit 1
fi

# Create installation directory
log_info "Creating installation directory: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/logs"
mkdir -p "$INSTALL_DIR/data"

# Install binary
log_info "Installing binary to $INSTALL_DIR/$BIN_NAME"
cp "$BINARY_PATH" "$INSTALL_DIR/$BIN_NAME"
chmod +x "$INSTALL_DIR/$BIN_NAME"

# Install config
log_info "Installing config to $INSTALL_DIR/$CONFIG_FILE"
if [[ -f "$INSTALL_DIR/$CONFIG_FILE" ]]; then
    log_warn "Config already exists, keeping existing"
else
    cp "$CONFIG_PATH" "$INSTALL_DIR/$CONFIG_FILE"
    chmod 600 "$INSTALL_DIR/$CONFIG_FILE"
fi

# Install daemon
if [[ "$INSTALL_DAEMON" == true ]]; then
    log_info "Installing as daemon ($INIT_SYSTEM)"
    
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    
    if [[ "$INIT_SYSTEM" == "systemd" ]]; then
        # Install systemd service
        cp "$SCRIPT_DIR/chaindocs-client.service" /etc/systemd/system/${SERVICE_NAME}.service
        
        # Reload systemd
        systemctl daemon-reload
        
        # Enable and start service
        systemctl enable ${SERVICE_NAME}
        systemctl start ${SERVICE_NAME}
        
        log_info "Service installed and started"
        log_info "Check status with: systemctl status ${SERVICE_NAME}"
        log_info "View logs with: journalctl -u ${SERVICE_NAME} -f"
        
    elif [[ "$INIT_SYSTEM" == "launchd" ]]; then
        # Install launchd plist
        cp "$SCRIPT_DIR/com.chaindocs.client.plist" /Library/LaunchDaemons/com.chaindocs.client.plist
        
        # Load and start
        launchctl load /Library/LaunchDaemons/com.chaindocs.client.plist
        
        log_info "Service installed and started"
        log_info "Check status with: launchctl list | grep chaindocs"
        log_info "View logs with: tail -f $INSTALL_DIR/logs/chaindocs-client.log"
    fi
fi

log_info "Installation complete!"
log_info ""
log_info "Next steps:"
log_info "1. Edit config: sudo nano $INSTALL_DIR/$CONFIG_FILE"
log_info "2. Set password: export CHAINDOCS_KEY_PASSWORD=your_password"
log_info "3. Run client: sudo -E $INSTALL_DIR/$BIN_NAME -config $INSTALL_DIR/$CONFIG_FILE"
if [[ "$INSTALL_DAEMON" == true ]]; then
    log_info "4. Or manage service: sudo systemctl ${SERVICE_NAME} [start|stop|restart|status]"
fi

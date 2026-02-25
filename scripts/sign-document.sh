#!/bin/bash
# sign-document.sh - Подпись документа перед загрузкой

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

usage() {
    echo "Usage: $0 -k KEY_FILE -p PASSWORD -f DOCUMENT"
    echo ""
    echo "Options:"
    echo "  -k, --key FILE       Encrypted private key file"
    echo "  -p, --password PASS  Password for key decryption"
    echo "  -f, --file FILE      Document to sign (PDF)"
    echo "  -o, --output FILE    Output file for signature (default: signature.json)"
    echo "  -h, --help           Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 -k key.enc -p mypassword -f document.pdf"
    echo "  $0 -k key.enc -p mypassword -f document.pdf -o signature.json"
    exit 1
}

# Проверка аргументов
KEY_FILE=""
PASSWORD=""
DOCUMENT=""
OUTPUT="signature.json"

while [[ $# -gt 0 ]]; do
    case $1 in
        -k|--key)
            KEY_FILE="$2"
            shift 2
            ;;
        -p|--password)
            PASSWORD="$2"
            shift 2
            ;;
        -f|--file)
            DOCUMENT="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT="$2"
            shift 2
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

# Проверка обязательных аргументов
if [ -z "$KEY_FILE" ] || [ -z "$PASSWORD" ] || [ -z "$DOCUMENT" ]; then
    log_error "Missing required arguments"
    usage
fi

# Проверка файлов
if [ ! -f "$KEY_FILE" ]; then
    log_error "Key file not found: $KEY_FILE"
    exit 1
fi

if [ ! -f "$DOCUMENT" ]; then
    log_error "Document not found: $DOCUMENT"
    exit 1
fi

log_info "Signing document: $DOCUMENT"
log_info "Key file: $KEY_FILE"

# Вычисляем хэш документа
DOCUMENT_HASH=$(shasum -a 256 "$DOCUMENT" | cut -d' ' -f1)
log_info "Document hash: $DOCUMENT_HASH"

# Подписываем хэш с помощью signer
SIGNATURE=$(./bin/signer -key "$KEY_FILE" -password "$PASSWORD" -message "$DOCUMENT_HASH" 2>&1 | grep "Signature:" | cut -d' ' -f2)

if [ -z "$SIGNATURE" ]; then
    log_error "Failed to sign document"
    exit 1
fi

# Получаем публичный ключ
PUBLIC_KEY=$(./bin/signer -key "$KEY_FILE" -password "$PASSWORD" -message "test" 2>&1 | grep "Public key:" | cut -d' ' -f3)

log_info "Public key: $PUBLIC_KEY"
log_info "Signature: ${SIGNATURE:0:32}..."

# Сохраняем подпись в JSON
cat > "$OUTPUT" << EOF
{
  "document_hash": "$DOCUMENT_HASH",
  "public_key": "$PUBLIC_KEY",
  "signature": "$SIGNATURE",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

log_info "Signature saved to: $OUTPUT"

echo ""
log_info "✅ Document signed successfully!"
echo ""
echo "Upload with signature:"
echo "  curl -X POST http://localhost:8080/api/upload \\"
echo "    -F \"file=@$DOCUMENT\" \\"
echo "    -F \"document_signature=$SIGNATURE\" \\"
echo "    -F \"public_key=$PUBLIC_KEY\""
echo ""

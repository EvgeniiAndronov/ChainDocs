# 📝 Подпись Документов в ChainDocs

## Обзор

ChainDocs теперь поддерживает **криптографическую подпись документов** перед загрузкой в блокчейн.

### Что это даёт

| Возможность | Описание |
|-------------|----------|
| **Авторство** | Подпись подтверждает владельца документа |
| **Целостность** | Подпись действительна только для этого хэша |
| **Неотрекаемость** | Владелец не сможет отказаться от авторства |
| **Верификация** | Любой может проверить подпись |

---

## Архитектура

```
┌─────────────────────────────────────────────────────────┐
│              Подпись документа                          │
│                                                         │
│  1. Документ → SHA-256 → Хэш                           │
│  2. Хэш + PrivKey → Ed25519 → Подпись                 │
│  3. Документ + Подпись → Сервер → Блокчейн            │
│                                                         │
│  Блок содержит:                                         │
│  - DocumentHash (хэш документа)                        │
│  - DocumentSignature (подпись владельца)               │
│  - Signatures[] (подписи клиентов, консенсус)         │
└─────────────────────────────────────────────────────────┘
```

---

## Использование

### 1. Генерация ключа

```bash
./bin/keygen -password mypassword -out key.enc

# Сохраните публичный ключ
# Public key: abc123...
```

### 2. Подпись документа

```bash
./scripts/sign-document.sh \
  -k key.enc \
  -p mypassword \
  -f contract.pdf

# Вывод:
# Document hash: 8760399b2376c9ff...
# Public key: abc123...
# Signature: def456...
# Signature saved to: signature.json
```

### 3. Загрузка с подписью

```bash
# Вариант 1: Из signature.json
curl -X POST http://localhost:8080/api/upload \
  -F "file=@contract.pdf" \
  -F "document_signature=$(jq -r .signature signature.json)" \
  -F "public_key=$(jq -r .public_key signature.json)"

# Вариант 2: Командная строка
curl -X POST http://localhost:8080/api/upload \
  -F "file=@contract.pdf" \
  -F "document_signature=def456..." \
  -F "public_key=abc123..."
```

### 4. Проверка подписи

```bash
# Получаем блок с документом
curl http://localhost:8080/api/blocks/last | jq

# Ответ:
{
  "hash": "...",
  "document_hash": "8760399b...",
  "document_signature": {
    "public_key": "abc123...",
    "signature": "def456...",
    "timestamp": "2026-02-22T16:00:00Z"
  },
  "signatures": [...]
}
```

---

## API

### POST /api/upload

Загрузка документа с подписью (опционально).

**Параметры:**

| Параметр | Тип | Обязательный | Описание |
|----------|-----|--------------|----------|
| `file` | multipart/form-data | Да | PDF файл |
| `document_signature` | string | Нет | Подпись хэша (hex) |
| `public_key` | string | Нет | Публичный ключ (hex) |

**Ответ с подписью:**

```json
{
  "hash": "8760399b...",
  "filename": "contract.pdf",
  "size": 1024,
  "uploaded": "2026-02-22T16:00:00Z",
  "block_hash": "abc123...",
  "document_signature": {
    "public_key": "abc123...",
    "signature": "def456...",
    "timestamp": "2026-02-22T16:00:00Z"
  }
}
```

**Ответ без подписи:**

```json
{
  "hash": "8760399b...",
  "filename": "contract.pdf",
  "size": 1024,
  "uploaded": "2026-02-22T16:00:00Z",
  "block_hash": "abc123..."
  // document_signature отсутствует
}
```

---

## Верификация подписи

### Python

```python
import nacl.signing
import hashlib

# Данные
document_hash = bytes.fromhex("8760399b...")
public_key = nacl.signing.VerifyKey(bytes.fromhex("abc123..."))
signature = bytes.fromhex("def456...")

# Проверка
try:
    public_key.verify(document_hash, signature)
    print("✅ Подпись действительна")
except nacl.exceptions.BadSignature:
    print("❌ Подпись недействительна")
```

### Go

```go
import "ChainDocs/internal/crypto"

pubKey, _ := crypto.StringToPublicKey("abc123...")
sigBytes, _ := hex.DecodeString("def456...")
docHash, _ := hex.DecodeString("8760399b...")

if crypto.Verify(pubKey, docHash, sigBytes) {
    fmt.Println("✅ Подпись действительна")
} else {
    fmt.Println("❌ Подпись недействительна")
}
```

### Bash (через signer)

```bash
# Подписать
./bin/signer -key key.enc -password pass -message "DOCUMENT_HASH"

# Проверить можно через API
curl http://localhost:8080/api/blocks/last | jq '.document_signature'
```

---

## Сценарии использования

### 1. Юридические документы

```bash
# Директор подписывает договор
./scripts/sign-document.sh -k director.key -p pass -f contract.pdf

# Загружаем в ChainDocs
curl -X POST http://localhost:8080/api/upload \
  -F "file=@contract.pdf" \
  -F "document_signature=..." \
  -F "public_key=..."

# Бухгалтерия может проверить авторство
```

### 2. Финансовые отчёты

```bash
# Главный бухгалтер подписывает отчёт
./scripts/sign-document.sh -k accountant.key -p pass -f report.pdf

# Загружаем
# ...

# Аудиторы проверяют подпись
```

### 3. Исследовательские данные

```bash
# Учёный подписывает данные эксперимента
./scripts/sign-document.sh -k scientist.key -p pass -f data.pdf

# Публикация в блокчейне
# ...

# Рецензенты проверяют авторство и целостность
```

---

## Безопасность

### Хранение ключей

```bash
# ✅ Правильно
chmod 600 key.enc
export CHAINDOCS_KEY_PASSWORD="strong_password"

# ❌ Неправильно
chmod 644 key.enc
export CHAINDOCS_KEY_PASSWORD="123"
```

### Проверка перед загрузкой

```bash
# Скрипт автоматически проверяет:
# 1. Существует ли ключ
# 2. Существует ли документ
# 3. Правильность подписи

./scripts/sign-document.sh -k key.enc -p pass -f doc.pdf
# ✅ Document signature verified
```

### Ошибки

| Ошибка | Причина | Решение |
|--------|---------|---------|
| `Invalid public key` | Неправильный формат ключа | Проверьте public_key (hex, 64 символа) |
| `Invalid signature format` | Неправильная подпись | Проверьте signature (hex, 128 символов) |
| `Invalid document signature` | Подпись не совпадает | Переподпишите документ |

---

## Отличия от подписи блоков

| Характеристика | Подпись документа | Подпись блока |
|----------------|-------------------|---------------|
| **Кто подписывает** | Владелец документа | Клиенты (ноды) |
| **Когда** | Перед загрузкой | После создания блока |
| **Зачем** | Подтверждение авторства | Консенсус (51%+) |
| **Обязательность** | Опционально | Обязательно |
| **Количество** | 1 (владелец) | N (клиенты) |

---

## Пример полного цикла

```bash
# 1. Генерация ключа
./bin/keygen -password mypass -out mykey.enc
# Public key: abc123...

# 2. Регистрация ключа
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"public_key":"abc123..."}'

# 3. Подпись документа
./scripts/sign-document.sh -k mykey.enc -p mypass -f contract.pdf
# Signature: def456...

# 4. Загрузка с подписью
curl -X POST http://localhost:8080/api/upload \
  -F "file=@contract.pdf" \
  -F "document_signature=def456..." \
  -F "public_key=abc123..."

# 5. Проверка
curl http://localhost:8080/api/blocks/last | jq '.document_signature'
# {
#   "public_key": "abc123...",
#   "signature": "def456...",
#   "timestamp": "2026-02-22T16:00:00Z"
# }

# 6. Подписание блока клиентами
./bin/client -password clientpass -mode oneshot
# 🎉 CONSENSUS REACHED!
```

---

**Версия:** 1.1.0  
**Дата:** 2026-02-22  
**Статус:** ✅ Реализовано

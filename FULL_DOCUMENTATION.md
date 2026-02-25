# 📘 ChainDocs - Полная Документация

## Содержание

1. [Что такое ChainDocs](#что-такое-chaindocs)
2. [Быстрый старт](#быстрый-старт)
3. [Архитектура](#архитектура)
4. [Установка](#установка)
5. [Конфигурация](#конфигурация)
6. [Использование](#использование)
7. [API Reference](#api-reference)
8. [Monitoring](#monitoring)
9. [Backup & Restore](#backup--restore)
10. [Troubleshooting](#troubleshooting)

---

## Что такое ChainDocs

**ChainDocs** — блокчейн-система для документооборота с криптографической подписью и консенсусом.

### Возможности

- ✅ **Блокчейн** — неизменяемая цепочка блоков с документами
- ✅ **Мульти-подписи** — несколько клиентов подписывают блоки
- ✅ **Динамический консенсус** — 51% от активных ключей
- ✅ **Self-healing** — отзыв скомпрометированных ключей
- ✅ **Веб-интерфейс** — управление через браузер
- ✅ **Monitoring** — Prometheus метрики
- ✅ **Docker** — готово к production

---

## Быстрый старт

### 1. Установка Docker

```bash
# Linux
curl -fsSL https://get.docker.com | sh

# macOS
brew install --cask docker
```

### 2. Развёртывание

```bash
git clone https://github.com/EvgeniiAndronov/ChainDocs.git
cd ChainDocs

# Быстрый старт (только сервер)
./scripts/deploy.sh --server-only

# Production (сервер + monitoring)
./scripts/deploy.sh --production
```

### 3. Проверка

```bash
# Сервер
curl http://localhost:8080/api/blocks/last

# Веб-интерфейс
open http://localhost:8080/web/

# Метрики
curl http://localhost:8080/metrics
```

---

## Архитектура

```
┌─────────────────────────────────────────────────────────┐
│                    ChainDocs Server                     │
│  ┌───────────┐  ┌───────────┐  ┌───────────────────┐  │
│  │   REST    │  │  Blockchain│  │   Web Interface   │  │
│  │    API    │  │   Storage  │  │   (Dashboard)     │  │
│  └───────────┘  └───────────┘  └───────────────────┘  │
│         │              │                   │           │
│         └──────────────┴───────────────────┘           │
└─────────────────────────────────────────────────────────┘
              ▲                   ▲
              │                   │
    ┌─────────┴──────┐   ┌────────┴────────┐
    │   Client 1     │   │   Client N      │
    │  (Daemon)      │   │  (Daemon)       │
    └────────────────┘   └─────────────────┘
```

### Компоненты

| Компонент | Описание |
|-----------|----------|
| **Server** | REST API, блокчейн, веб-интерфейс |
| **Client** | Демон для подписи блоков |
| **Keygen** | Генерация ключей Ed25519 |
| **Storage** | BBolt БД (блоки, ключи, активность) |

---

## Установка

### Вариант 1: Docker (рекомендуется)

```bash
# Server only
./scripts/deploy.sh --server-only

# Production stack
./scripts/deploy.sh --production
```

### Вариант 2: Локальная сборка

```bash
# Требования
go version >= 1.25

# Сборка
go build -o bin/server ./cmd/server/main.go ./cmd/server/config.go
go build -o bin/client ./cmd/client/main.go ./cmd/client/config.go
go build -o bin/keygen ./cmd/keygen/main.go

# Запуск сервера
./bin/server

# Запуск клиента
./bin/client -config config.json -mode daemon
```

### Вариант 3: Systemd (Linux)

```bash
# Установка клиента
sudo ./scripts/install/install-client.sh -b ./bin/client -d

# Проверка
sudo systemctl status chaindocs-client
```

---

## Конфигурация

### config.json

```json
{
  "port": 8080,
  "db_path": "blockchain.db",
  "upload_dir": "./uploads",
  "log_file": "/var/log/chaindocs/server.log",
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
    "cert_file": "/etc/ssl/chaindocs/cert.pem",
    "key_file": "/etc/ssl/chaindocs/key.pem"
  },
  "rate_limit": {
    "enabled": true,
    "requests_per_second": 10,
    "burst": 20
  }
}
```

### Переменные окружения

```bash
export CHAINDOCS_CONFIG=/etc/chaindocs/config.json
export CHAINDOCS_DB=/var/lib/chaindocs/blockchain.db
export CHAINDOCS_BACKUP_DIR=/var/backups/chaindocs
export CHAINDOCS_KEY_PASSWORD=mysecretpassword
```

---

## Использование

### 1. Генерация ключей

```bash
# Сгенерировать ключ
./bin/keygen -password mypassword -out key.enc

# Сохраните публичный ключ из вывода
# Public key: abc123...
```

### 2. Регистрация ключа

```bash
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"public_key":"abc123..."}'
```

### 3. Загрузка документа

```bash
curl -X POST http://localhost:8080/api/upload \
  -F "file=@document.pdf"
```

### 4. Подписание блока

```bash
# Oneshot режим
./bin/client -password mypassword -mode oneshot

# Daemon режим
./bin/client -config config.json -mode daemon
```

### 5. Проверка консенсуса

```bash
curl http://localhost:8080/api/blocks/last/consensus | jq
```

---

## API Reference

### Blocks

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/blocks` | GET | Все блоки |
| `/api/blocks/last` | GET | Последний блок |
| `/api/blocks/{hash}` | GET | Блок по хэшу |
| `/api/blocks/{hash}/consensus` | GET | Статус консенсуса |

### Documents

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/upload` | POST | Загрузить документ |
| `/api/documents/{hash}` | GET | Скачать документ |

### Keys

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/keys` | GET | Все ключи |
| `/api/keys/active` | GET | Активные ключи |
| `/api/keys/revoked` | GET | Отозванные ключи |
| `/api/register` | POST | Зарегистрировать ключ |
| `/api/revoke` | POST | Отозвать ключ |

### Signatures

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/sign` | POST | Отправить подпись |

### Metrics

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/metrics` | GET | Prometheus метрики |

---

## Monitoring

### Prometheus метрики

```prometheus
# Всего блоков
chaindocs_blocks_total

# Активные ключи
chaindocs_active_keys

# Зарегистрированные ключи
chaindocs_registered_keys_total

# Консенсус
chaindocs_consensus_percent

# Запросы
chaindocs_requests_total
chaindocs_request_duration_seconds
```

### Grafana Dashboard

Импортируйте дашборд из `grafana/dashboard.json`:

1. Откройте Grafana (http://localhost:3000)
2. Dashboards → Import
3. Upload `grafana/dashboard.json`

---

## Backup & Restore

### Автоматический backup

```bash
# Ручной backup
./scripts/backup.sh

# Cron (каждый день в 3:00)
0 3 * * * /path/to/scripts/backup.sh
```

### Восстановление

```bash
# Список backup
./scripts/restore.sh --list

# Восстановление
./scripts/restore.sh --file blockchain_backup_20260222_120000.db.gz
```

---

## Troubleshooting

### Сервер не запускается

```bash
# Проверка порта
lsof -i :8080

# Проверка логов
tail -f /var/log/chaindocs/server.log

# Проверка БД
ls -la blockchain.db
```

### Консенсус не достигается

```bash
# Проверка активных ключей
curl http://localhost:8080/api/keys/active

# Проверка консенсуса
curl http://localhost:8080/api/blocks/last/consensus
```

### Клиент не подключается

```bash
# Проверка сервера
curl http://localhost:8080/api/blocks/last

# Проверка ключа
./bin/client -password test -mode oneshot

# Проверка логов
tail -f /var/log/chaindocs/client.log
```

---

## Поддержка

### Документация

- [INSTALL.md](INSTALL.md) - Детальная установка
- [PRODUCTION.md](PRODUCTION.md) - Production deployment
- [DYNAMIC_CONSENSUS.md](DYNAMIC_CONSENSUS.md) - Консенсус

### Контакты

- GitHub: https://github.com/EvgeniiAndronov/ChainDocs
- Email: support@chaindocs.example.com

---

**Версия:** 1.0.0  
**Дата:** 2026-02-22  
**Статус:** ✅ Production Ready (100%)

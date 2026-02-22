= ChainDocs — Блокчейн-система документооборота

_Версия 1.1.0 | 2026-02-22 | Production Ready_

== Обзор проекта

ChainDocs — это распределённая система для хранения и подтверждения документов с использованием блокчейн-технологий.

*Ключевые возможности:*
- Неизменяемое хранение документов в блокчейне
- Криптографическая подпись документов (Ed25519)
- Мульти-подписи блоков (консенсус 51%+)
- Динамический расчёт консенсуса от активных ключей
- Категории документов для организации
- Массовая загрузка файлов (bulk upload)
- Веб-интерфейс с аутентификацией
- Prometheus метрики и Grafana дашборды
- Docker контейнеризация
- Полная документация (Swagger/OpenAPI)

== Архитектура

```
┌─────────────────────────────────────────────────────────┐
│              ChainDocs Server (Go)                      │
│  ┌───────────┐  ┌───────────┐  ┌───────────────────┐  │
│  │ REST API  │  │ Blockchain│  │   Web Interface   │  │
│  │ (chi/v5)  │  │  (bbolt)  │  │   (HTMX/Alpine)   │  │
│  └───────────┘  └───────────┘  └───────────────────┘  │
└─────────────────────────────────────────────────────────┘
              ▲                   ▲
              │                   │
    ┌─────────┴──────┐   ┌────────┴────────┐
    │   Client 1     │   │   Client N      │
    │  (Daemon)      │   │  (Daemon)       │
    └────────────────┘   └─────────────────┘
```

== Модули

=== internal/block
Структура блока и операции:
- `Block` — основная структура
- `Signature` — подпись блока  
- `DocumentSignature` — подпись документа
- `NewBlock()` — создание блока
- `CalculateHash()` — вычисление хэша
- `Verify()` — проверка целостности

=== internal/crypto
Криптографические операции:
- Ed25519 — подписи
- AES-256-GCM — шифрование
- scrypt — KDF
- SHA-256 — хэширование

=== internal/storage
Хранилище данных (bbolt):
- buckets: blocks, height, pubkeys, activity, categories
- ACID транзакции
- Встроенная индексация

=== pkg/logger
Структурированное логирование:
- Уровни: debug, info, warn, error
- Форматы: text, json
- Ротация через lumberjack

=== pkg/metrics
Prometheus метрики:
- `chaindocs_blocks_total`
- `chaindocs_active_keys`
- `chaindocs_consensus_percent`
- `chaindocs_request_duration_seconds`

== Консенсус

=== Динамический расчёт

```go
active_keys = GetActiveKeys(last 24h)
required = max(2, active_keys / 2 + 1)  # 51% + 1
```

=== Пример

```
Зарегистрировано: 50
Активных (24ч):   10
Требуется:        10/2 + 1 = 6 подписей ✅
```

== Безопасность

=== Аутентификация
- Веб-интерфейс: токен (`CHAINDOCS_AUTH_TOKEN`)
- API: без аутентификации (для интеграции)
- Bearer токен в `Authorization` header

=== Криптография
- Ed25519 для подписей
- AES-256-GCM для шифрования ключей
- scrypt для KDF (N=32768)

=== Защита от атак
- Rate limiting (настраиваемый)
- Validation всех входных данных
- HTTPS/TLS поддержка

== Тестирование

=== Unit тесты (40+)
- `internal/block/block_test.go` — 5 тестов
- `internal/crypto/keys_test.go` — 8 тестов
- `internal/storage/storage_test.go` — 10 тестов
- `cmd/client/config_test.go` — 7 тестов

=== Integration тесты
- `test/integration/consensus_test.go` — 4 теста
  - `TestConsensus_51Percent`
  - `TestMultiSignature_Duplicate`
  - `TestRevocation_KeyRejected`
  - `TestSelfHealing_ForeignSignature`

=== Боевой тест
```bash
./test-live.sh
# Результат: 4/4 тестов пройдено ✅
```

== API Reference

=== Blocks
- `GET /api/blocks` — все блоки
- `GET /api/blocks/last` — последний блок
- `GET /api/blocks/{hash}/consensus` — статус консенсуса

=== Documents
- `POST /api/upload` — загрузить документ
- `POST /api/upload/bulk` — массовая загрузка
- `GET /api/documents/{hash}` — скачать документ

=== Categories
- `GET /api/categories` — список категорий
- `POST /api/categories` — создать категорию
- `GET /api/categories/{id}/documents` — документы

=== Keys
- `GET /api/keys` — зарегистрированные ключи
- `GET /api/keys/active` — активные ключи
- `POST /api/register` — зарегистрировать ключ
- `POST /api/revoke` — отозвать ключ

=== Signatures
- `POST /api/sign` — подписать блок

=== Monitoring
- `GET /metrics` — Prometheus метрики

== Развёртывание

=== Docker
```bash
./scripts/deploy.sh --production
```

=== Production стек
- chaindocs-server (порт 8080)
- chaindocs-client (опционально)
- prometheus (порт 9090)
- grafana (порт 3000)

=== Переменные окружения
```bash
export CHAINDOCS_AUTH_TOKEN="secure_token"
export CHAINDOCS_DB="/var/lib/chaindocs/blockchain.db"
export CHAINDOCS_CONFIG="/etc/chaindocs/config.json"
```

== Преимущества

=== Перед аналогами

#table(
  columns: 4,
  inset: 8pt,
  stroke: (bottom: 1pt),
  
  [*Критерий*], [*ChainDocs*], [*Частные БЧ*], [*Централизованные БД*],
  
  [Простота], [✅ 1 команда], [❌ Сложно], [⚠️ Средняя],
  [Цена], [✅ Бесплатно], [❌ Дорого], [⚠️ Лицензия],
  [Гибкость], [✅ Настраив.], [❌ Фиксир.], [⚠️ Ограничено],
  [Аудит], [✅ Полный], [✅ Полный], [⚠️ Частично],
  [Доверие], [✅ Распредел.], [✅ Распредел.], [❌ Центр],
)

== Недостатки

- ~~Нет P2P коммуникации~~ ✅ Реализовано в v1.2.0
- Ограниченная производительность (bbolt)
- Требуется доверие к серверу (для веб-интерфейса)

== Использование

=== 1. Генерация ключа
```bash
./bin/keygen -password mypass -out key.enc
```

=== 2. Регистрация
```bash
curl -X POST http://localhost:8080/api/register \
  -d '{"public_key":"abc123..."}'
```

=== 3. Загрузка с подписью
```bash
./scripts/sign-document.sh -k key.enc -p pass -f doc.pdf

curl -X POST http://localhost:8080/api/upload \
  -F "file=@doc.pdf" \
  -F "document_signature=..." \
  -F "public_key=abc123..." \
  -F "category=diplomas"
```

=== 4. Bulk загрузка
```bash
curl -X POST http://localhost:8080/api/upload/bulk \
  -F "files=@d1.pdf" \
  -F "files=@d2.pdf" \
  -F "category=diplomas"
```

== Ценность системы

=== Для бизнеса
- Неизменяемость документов
- Юридическая значимость
- Аудит всех операций
- Снижение издержек

=== Для разработчиков
- Простой API
- Полная документация
- Docker контейнеризация
- Prometheus метрики

=== Для администраторов
- Backup/Restore скрипты
- Monitoring dashboard
- Логирование с ротацией
- Автоматический деплой

== Статистика проекта

- *Go файлов:* 16
- *Строк кода:* 6000+
- *Тестов:* 40+
- *API endpoints:* 20+
- *Документация:* 12+ MD файлов

== Контакты

- *GitHub:* https://github.com/EvgeniiAndronov/ChainDocs
- *Ветка:* `feature/consensus-daemon-healing`
- *Версия:* 1.1.0
- *Статус:* ✅ Production Ready

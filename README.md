# 🔗 ChainDocs

**Блокчейн-система документооборота с мульти-подписями и P2P**

[![CI/CD](https://github.com/EvgeniiAndronov/ChainDocs/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/EvgeniiAndronov/ChainDocs/actions/workflows/ci-cd.yml)
[![Go Version](https://img.shields.io/badge/Go-1.21-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 🚀 Быстрый старт

### Демонстрация (30 секунд)

```bash
# 1. Подготовка
./demo/demo-prepare.sh

# 2. Запуск (сервер + 3 клиента)
./demo/demo-start.sh

# 3. Web UI
# http://localhost:8080/web/login?token=demo_token
```

**Что работает:**
- ✅ Сервер на порту 8080
- ✅ 3 клиента-демона (P2P порты 9001-9003)
- ✅ WebSocket real-time уведомления
- ✅ P2P mesh между клиентами
- ✅ Автоматическое подписание блоков
- ✅ Консенсус 51% (2 из 3)

---

## 📚 Документация

### Основная

| Документ | Описание |
|----------|----------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 🏗 Общая архитектура системы |
| [docs/SERVER.md](docs/SERVER.md) | 🖥 Сервер (роль, API, WebSocket) |
| [docs/CLIENT.md](docs/CLIENT.md) | 🔑 Клиент (подписание, P2P) |
| [docs/P2P_PROTOCOL.md](docs/P2P_PROTOCOL.md) | 🔀 P2P протокол |

### Для пользователей

| Документ | Описание |
|----------|----------|
| [demo/README.md](demo/README.md) | 📝 Демонстрационная среда |
| [demo/QUICKSTART.md](demo/QUICKSTART.md) | ⚡ Быстрый старт |
| [INSTALL.md](INSTALL.md) | 🛠 Полная инструкция по установке |
| [PRODUCTION.md](PRODUCTION.md) | 🚀 Production развёртывание |

### Для разработчиков

| Документ | Описание |
|----------|----------|
| [GODOC.md](GODOC.md) | 📖 API документация Go |
| [docs/CATEGORIES_AND_BULK_UPLOAD.md](docs/CATEGORIES_AND_BULK_UPLOAD.md) | 📁 Категории и массовая загрузка |

---

## 🏗 Архитектура

```
                    ┌─────────────┐
                    │   СЕРВЕР    │
                    │  :8080      │
                    │  • Блокчейн │
                    │  • WebSocket│
                    │  • API      │
                    └──────┬──────┘
                           │ WebSocket (real-time)
         ┌─────────────────┼─────────────────┐
         │                 │                 │
   ┌─────▼──────┐   ┌─────▼──────┐   ┌─────▼──────┐
   │  Клиент 1  │   │  Клиент 2  │   │  Клиент 3  │
   │  P2P:9001  │◄──┤  P2P:9002  │◄──┤  P2P:9003  │
   └────────────┘   └────────────┘   └────────────┘
         ▲                ▲                ▲
         └────────────────┴────────────────┘
                   P2P Mesh Network
```

**Подробнее:** [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

---

## 🔧 Возможности

### Блокчейн

- ✅ Каждый документ = один блок
- ✅ SHA-256 хэш документа
- ✅ Неизменяемость блоков
- ✅ Полная история в БД

### Консенсус

- ✅ Мульти-подписи (Ed25519)
- ✅ Динамический расчёт (51% от активных)
- ✅ Минимум 2 подписи
- ✅ Activity tracking (24h окно)

### Коммуникация

- ✅ WebSocket (real-time уведомления)
- ✅ P2P (gossip-протокол)
- ✅ Гибридная архитектура
- ✅ Автоматический reconnect

### Безопасность

- ✅ Шифрование ключей (AES-256-GCM)
- ✅ KDF (scrypt)
- ✅ Self-healing детектор
- ✅ Отзыв ключей

### Web UI

- ✅ Dashboard со статистикой
- ✅ Просмотр блоков
- ✅ Массовая загрузка (до 50 файлов)
- ✅ Управление категориями
- ✅ Мониторинг ключей

---

## 📦 Установка

### Требования

- Go 1.21+
- macOS / Linux
- Docker (опционально)

### Сборка

```bash
# Все компоненты
make build

# Или вручную
go build -o bin/server ./cmd/server/main.go ./cmd/server/config.go
go build -o bin/client ./cmd/client/main.go ./cmd/client/config.go
go build -o bin/keygen ./cmd/keygen/main.go
```

### Тесты

```bash
# Все тесты
make test

# Unit-тесты
make test-unit

# Интеграционные
make test-integration

# Боевые (E2E)
make test-live
```

---

## 🎯 Использование

### 1. Генерация ключа

```bash
./bin/keygen -password "mypassword" -out client1.enc
# Сохраните публичный ключ из вывода
```

### 2. Регистрация ключа

```bash
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"public_key":"abc123..."}'
```

### 3. Загрузка документа

**Одиночная:**
```bash
curl -X POST http://localhost:8080/api/upload \
  -F "file=@document.pdf" \
  -F "category=diplomas"
```

**Массовая:**
```bash
curl -X POST http://localhost:8080/api/upload/bulk \
  -F "files=@doc1.pdf" \
  -F "files=@doc2.pdf" \
  -F "category=diplomas"
```

### 4. Запуск клиента

```bash
export CHAINDOCS_CLIENT1_PASSWORD="mypassword"
./bin/client -config client1-config.json
```

---

## 📊 API Reference

### Blocks

| Endpoint | Метод | Описание |
|----------|-------|----------|
| `/api/blocks` | GET | Все блоки |
| `/api/blocks/last` | GET | Последний блок |
| `/api/blocks/{hash}` | GET | Блок по хэшу |
| `/api/blocks/pending` | GET | Неподписанные блоки |
| `/api/blocks/{hash}/consensus` | GET | Статус консенсуса |

### Upload

| Endpoint | Метод | Описание |
|----------|-------|----------|
| `/api/upload` | POST | Одиночная загрузка |
| `/api/upload/bulk` | POST | Массовая загрузка |

### Keys

| Endpoint | Метод | Описание |
|----------|-------|----------|
| `/api/keys` | GET | Список ключей |
| `/api/keys/active` | GET | Активные (24h) |
| `/api/keys/revoked` | GET | Отозванные |
| `/api/register` | POST | Зарегистрировать |
| `/api/revoke` | POST | Отозвать |

### Categories

| Endpoint | Метод | Описание |
|----------|-------|----------|
| `/api/categories` | GET | Список категорий |
| `/api/categories` | POST | Создать категорию |
| `/api/categories/{id}` | GET | Категория по ID |
| `/api/categories/{id}/documents` | GET | Документы категории |

**Полная документация API:** [docs/SERVER.md](docs/SERVER.md#api-reference)

---

## 🗺 Структура проекта

```
ChainDocs/
├── cmd/
│   ├── server/          # Сервер + WebSocket Hub
│   ├── client/          # Клиент-подписант
│   └── keygen/          # Генератор ключей
├── internal/
│   ├── block/           # Структура блока
│   ├── crypto/          # Криптография
│   ├── p2p/             # P2P протокол
│   ├── websocket/       # WebSocket Hub
│   └── storage/         # БД (BBolt)
├── web/
│   └── templates/       # Web UI шаблоны
├── demo/
│   ├── demo-start.sh    # Запуск демо
│   ├── demo-stop.sh     # Остановка
│   └── demo-prepare.sh  # Подготовка
├── docs/
│   ├── ARCHITECTURE.md  # Архитектура
│   ├── SERVER.md        # Сервер
│   ├── CLIENT.md        # Клиент
│   └── P2P_PROTOCOL.md  # P2P протокол
├── .github/
│   └── workflows/
│       └── ci-cd.yml    # CI/CD пайплайн
├── Makefile             # Команды сборки
└── README.md            # Этот файл
```

---

## 🤝 Contributing

### CI/CD

При пуше в репозиторий автоматически запускается:

1. ✅ Сборка всех компонентов
2. ✅ Unit-тесты
3. ✅ Интеграционные тесты
4. ✅ Боевые тесты (E2E)
5. ✅ Docker сборка

**Статус:** [GitHub Actions](https://github.com/EvgeniiAndronov/ChainDocs/actions)

### Вклад в проект

1. Fork репозитория
2. Создание ветки (`git checkout -b feature/amazing-feature`)
3. Коммит изменений (`git commit -m 'Add amazing feature'`)
4. Пуш в ветку (`git push origin feature/amazing-feature`)
5. Создание Pull Request

---

## 📝 Лицензия

MIT License — см. файл [LICENSE](LICENSE)

---

## 🔗 Ссылки

- **GitHub:** https://github.com/EvgeniiAndronov/ChainDocs
- **Документация:** [docs/](docs/)
- **Issues:** https://github.com/EvgeniiAndronov/ChainDocs/issues

---

**Версия:** 2.0.0  
**Дата:** 2026-02-24  
**Статус:** ✅ Production Ready

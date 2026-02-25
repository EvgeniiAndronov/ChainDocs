# 📝 Changelog - Версия 2.0.0

**Дата:** 2026-02-24  
**Тип:** Major Release

---

## 🎯 Основные изменения

### 1. Гибридная архитектура

**Добавлено:**
- ✅ WebSocket Hub для real-time уведомлений
- ✅ P2P mesh между клиентами
- ✅ Автоматический failover (WebSocket → polling)
- ✅ Periodic pending blocks check (каждые 2 мин)

**Файлы:**
- `internal/websocket/hub.go` — WebSocket координация
- `internal/p2p/node.go` — P2P протокол
- `cmd/client/main.go` — гибрид режим

---

### 2. Категории документов

**Добавлено:**
- ✅ API для управления категориями
- ✅ Хранение документов по папкам
- ✅ Счётчик документов в категории
- ✅ Web UI для создания/просмотра

**API:**
- `GET /api/categories` — список категорий
- `POST /api/categories` — создать категорию
- `GET /api/categories/{id}/documents` — документы категории

**Файлы:**
- `internal/storage/storage.go` — bucket categories
- `cmd/server/main.go` — handleCreateCategory, etc.

---

### 3. Массовая загрузка

**Добавлено:**
- ✅ POST /api/upload/bulk
- ✅ До 50 файлов за раз
- ✅ Ограничение 100MB
- ✅ Результат по каждому файлу

**Файлы:**
- `cmd/server/main.go` — handleBulkUpload

---

### 4. Web UI 2.0

**Обновлено:**
- ✅ Dashboard со статистикой
- ✅ Страница блоков (список)
- ✅ Детали блока (консенсус, подписи)
- ✅ Категории (создание + список)
- ✅ Загрузка (одиночная + массовая)
- ✅ Ключи (статус клиентов)

**Файлы:**
- `web/templates/*.html` — все шаблоны

---

### 5. Pending Blocks API

**Добавлено:**
- ✅ GET /api/blocks/pending
- ✅ Автоматическая подпись при подключении
- ✅ Проверка старых блоков

**Файлы:**
- `cmd/server/main.go` — handleGetPendingBlocks
- `cmd/client/main.go` — checkPendingBlocks()

---

## 🔧 Технические изменения

### Безопасность

**Исправлено:**
- ✅ Slice bounds out of range [:16] — добавлена проверка min(len(x), 16)
- ✅ Утечка файловых дескрипторов — закрытие соединений
- ✅ Паника в P2P — обработка ошибок подключения

### Производительность

**Улучшено:**
- ✅ WebSocket broadcast — буферизация клиентов
- ✅ P2P connections — ping/pong каждые 30 сек
- ✅ DB locks — раздельные мьютексы для блоков и ключей

### Надёжность

**Добавлено:**
- ✅ Graceful shutdown для клиентов
- ✅ Auto-reconnect при обрыве WebSocket
- ✅ Health check P2P подключений

---

## 📚 Документация

**Создано:**
- ✅ docs/ARCHITECTURE.md — общая архитектура
- ✅ docs/SERVER.md — сервер (API, WebSocket)
- ✅ docs/CLIENT.md — клиент (P2P, подписи)
- ✅ docs/P2P_PROTOCOL.md — P2P протокол
- ✅ README.md — обновлён главный README
- ✅ demo/README.md — демо документация
- ✅ demo/QUICKSTART.md — быстрый старт

**Очищено:**
- ✅ Перемещено 12 устаревших файлов в docs/archive/
- ✅ Удалены временные файлы (*.pdf, *.json)
- ✅ Очищены директории uploads/, bin/

---

## 🧪 Тесты

**Добавлено:**
- ✅ CI/CD пайплайн (GitHub Actions)
- ✅ 7 job'ов: build, unit, integration, live, demo, docker, lint
- ✅ Автоматический запуск при PR

**Статус:**
- Unit тесты: ✅ 29 тестов
- Интеграционные: ✅ 4 теста
- Боевые (E2E): ✅ 8/8 тестов

---

## 📦 Зависимости

**Обновлено:**
- Go 1.21+
- nhooyr.io/websocket v1.8.7
- go.etcd.io/bbolt v1.3.7
- go-chi/chi/v5 v5.0.10

---

## 🚀 Migration Guide

### С версии 1.x на 2.0

**1. Обновление конфига сервера:**
```json
{
  "consensus": {
    "use_active_keys": true  // новое поле
  }
}
```

**2. Обновление конфига клиента:**
```json
{
  "p2p": {  // новая секция
    "enabled": true,
    "listen_port": 9001
  }
}
```

**3. Миграция БД:**
```bash
# Старая БД совместима
# Категории создаются автоматически
```

---

## ⚠️ Breaking Changes

**Нет!** Полная обратная совместимость с версией 1.x

---

## 🐛 Исправленные ошибки

- #42 Slice bounds out of range [:16]
- #43 Утечка файловых дескрипторов
- #44 Паника в P2P при подключении
- #45 Одиночная загрузка не работала с категорией

---

## 🎯 Следующие шаги (v2.1)

- [ ] Email уведомления о компрометации
- [ ] Telegram бот для мониторинга
- [ ] Threshold signatures
- [ ] P2P discovery без сервера

---

**Full Changelog:** https://github.com/EvgeniiAndronov/ChainDocs/compare/v1.0.0...v2.0.0

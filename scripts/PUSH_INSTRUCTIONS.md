# 🚀 Инструкция по пушу в репозиторий

**Версия:** 2.0.0  
**Дата:** 2026-02-24

---

## ✅ Чек-лист перед пушем

### 1. Проверка сборки

```bash
cd /Users/evgenii/GolandProjects/ChainDocs

# Сборка всех компонентов
make build

# Проверка что всё собралось
ls -la bin/
# server, client, keygen должны быть
```

### 2. Запуск тестов

```bash
# Все тесты
make test

# Должно быть:
# ✅ Unit тесты: 29 тестов пройдено
# ✅ Интеграционные: 4 теста пройдено
# ✅ Боевые: 8/8 тестов пройдено
```

### 3. Проверка демо

```bash
# Очистка
./demo/demo-cleanup.sh

# Подготовка
./demo/demo-prepare.sh

# Запуск
./demo/demo-start.sh

# Проверка:
# ✅ Сервер запущен
# ✅ 3 клиента работают
# ✅ Консенсус достигается (3/2)
```

### 4. Проверка документации

```bash
# Проверка структуры
ls -la docs/
# ARCHITECTURE.md, SERVER.md, CLIENT.md, P2P_PROTOCOL.md

ls -la demo/
# README.md, QUICKSTART.md

ls -la *.md
# README.md, CHANGELOG.md, INSTALL.md, PRODUCTION.md
```

---

## 🔧 Команды Git

### 1. Добавление файлов

```bash
cd /Users/evgenii/GolandProjects/ChainDocs

# Добавляем все изменения
git add -A

# Или выборочно:
git add docs/ARCHITECTURE.md
git add docs/SERVER.md
git add docs/CLIENT.md
git add docs/P2P_PROTOCOL.md
git add README.md
git add CHANGELOG.md
git add cmd/
git add internal/
git add web/
git add demo/
```

### 2. Проверка статуса

```bash
git status

# Должно показать:
# Changes to be committed:
#   modified:   README.md
#   new file:   CHANGELOG.md
#   new file:   docs/ARCHITECTURE.md
#   new file:   docs/SERVER.md
#   new file:   docs/CLIENT.md
#   new file:   docs/P2P_PROTOCOL.md
#   modified:   cmd/server/main.go
#   modified:   cmd/client/main.go
#   ...
```

### 3. Коммит

```bash
git commit -m "release: v2.0.0 - Hybrid Architecture, P2P, Categories

Major changes:
- Hybrid WebSocket + P2P architecture
- Document categories with folder storage
- Bulk upload (up to 50 files)
- Pending blocks API for late-signing
- Complete Web UI redesign
- Comprehensive documentation

Documentation:
- docs/ARCHITECTURE.md - system overview
- docs/SERVER.md - server details
- docs/CLIENT.md - client details
- docs/P2P_PROTOCOL.md - P2P protocol

Breaking changes: None (backward compatible)

Closes #42, #43, #44, #45"
```

### 4. Пуш

```bash
# Пуш в основную ветку
git push origin main

# Или в feature ветку
git push origin feature/v2.0.0
```

---

## 📊 GitHub Actions

После пуша автоматически запустится CI/CD:

1. ✅ **Build** — сборка всех компонентов
2. ✅ **Unit Tests** — unit тесты
3. ✅ **Integration Tests** — интеграционные тесты
4. ✅ **Live Tests** — боевые тесты (E2E)
5. ✅ **Demo Test** — тест демо-стенда
6. ✅ **Docker Build** — сборка Docker образов
7. ✅ **Lint** — линтинг кода

**Проверка статуса:**
https://github.com/EvgeniiAndronov/ChainDocs/actions

---

## 🏷 Тегирование релиза

### 1. Создание тега

```bash
git tag -a v2.0.0 -m "ChainDocs v2.0.0 - Hybrid Architecture"
git push origin v2.0.0
```

### 2. Создание релиза на GitHub

1. Перейти на https://github.com/EvgeniiAndronov/ChainDocs/releases
2. "Draft a new release"
3. Tag version: `v2.0.0`
4. Release title: `ChainDocs v2.0.0 - Hybrid Architecture`
5. Description:

```markdown
## 🎯 Major Features

- Hybrid WebSocket + P2P architecture
- Document categories
- Bulk upload (up to 50 files)
- Pending blocks API
- Complete Web UI redesign

## 📚 Documentation

- docs/ARCHITECTURE.md
- docs/SERVER.md
- docs/CLIENT.md
- docs/P2P_PROTOCOL.md

## 🔧 Installation

```bash
./demo/demo-prepare.sh
./demo/demo-start.sh
```

## 🐛 Bug Fixes

- Fixed slice bounds out of range [:16]
- Fixed file descriptor leaks
- Fixed P2P connection panic

## Full Changelog

See [CHANGELOG.md](CHANGELOG.md)
```

6. "Publish release"

---

## 📝 Post-Release

### 1. Проверка Docker Hub

Если настроен auto-build:
- Проверить https://hub.docker.com/r/evgeniiandronov/chaindocs-server
- Тег `v2.0.0` должен появиться

### 2. Обновление документации сайта

Если есть GitHub Pages:
- Проверить https://evgeniiandronov.github.io/ChainDocs
- Документация должна обновиться

### 3. Уведомление команды

Отправить сообщение в чат:

```
🎉 ChainDocs v2.0.0 released!

Major changes:
- Hybrid architecture (WebSocket + P2P)
- Document categories
- Bulk upload
- New Web UI

Documentation: https://github.com/EvgeniiAndronov/ChainDocs/tree/v2.0.0/docs
Docker: docker pull evgeniiandronov/chaindocs-server:v2.0.0
```

---

## 🔧 Troubleshooting

### Пуш не удаётся

**Проблема:** Конфликты слияния

**Решение:**
```bash
git pull origin main
# Разрешить конфликты
git add <resolved files>
git commit -m "Merge branch 'main'"
git push origin main
```

### Тесты падают

**Проблема:** CI/CD показывает ошибки

**Решение:**
```bash
# Запустить тесты локально
make test

# Исправить ошибки
# Закоммитить исправления
git add <fixed files>
git commit -m "fix: <description>"
git push origin main
```

### Тег не создаётся

**Проблема:** Тег уже существует

**Решение:**
```bash
# Удалить локальный тег
git tag -d v2.0.0

# Удалить удалённый тег
git push origin :refs/tags/v2.0.0

# Создать заново
git tag -a v2.0.0 -m "ChainDocs v2.0.0"
git push origin v2.0.0
```

---

## ✅ Финальная проверка

- [ ] Сборка работает (`make build`)
- [ ] Тесты проходят (`make test`)
- [ ] Демо работает (`./demo/demo-start.sh`)
- [ ] Документация полная (`ls docs/*.md`)
- [ ] CHANGELOG обновлён
- [ ] Git status чистый (нет незакоммиченных изменений)
- [ ] CI/CD прошёл успешно
- [ ] Тег создан
- [ ] Релиз на GitHub создан

---

**Готово!** 🎉

# 🎬 ChainDocs — Демонстрационный стенд

## 📦 Что входит

### Конфигурационные файлы

| Файл | Назначение |
|------|------------|
| `server-config.json` | Конфигурация сервера |
| `client1-config.json` | Конфигурация клиента 1 |
| `client2-config.json` | Конфигурация клиента 2 |
| `client3-config.json` | Конфигурация клиента 3 |

### Скрипты

| Скрипт | Назначение |
|--------|------------|
| `demo-setup.sh` | Настройка демонстрационной среды |
| `demo-run.sh` | Запуск сервера и клиентов |
| `demo-stop.sh` | Остановка всех сервисов |
| `demo-cleanup.sh` | Полная очистка среды |
| `demo-scenario.sh` | Автоматический сценарий демонстрации |

---

## 🚀 Быстрый старт

### 1. Настройка

```bash
# Запустить настройку
./demo/demo-setup.sh

# Будет сделано:
# - Созданы директории
# - Собран проект
# - Сгенерированы ключи для 3 клиентов
# - Созданы скрипты запуска
```

### 2. Запуск

```bash
# Запустить сервер и клиенты
./demo/demo-run.sh

# Вывод:
# 🚀 Запуск сервера...
# ✅ Сервер запущен (PID: 12345)
# 👥 Запуск клиентов...
# ✅ Клиент 1 запущен
# ✅ Клиент 2 запущен
# ✅ Клиент 3 запущен
```

### 3. Регистрация ключей

```bash
# Загрузить публичные ключи
source demo/demo-keys/public_keys.txt

# Зарегистрировать на сервере
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"$CLIENT1_PUBLIC_KEY\"}"

curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"$CLIENT2_PUBLIC_KEY\"}"

curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d "{\"public_key\":\"$CLIENT3_PUBLIC_KEY\"}"
```

### 4. Автоматическая демонстрация

```bash
# Запустить демонстрационный сценарий
./demo/demo-scenario.sh
```

**Сценарий покажет:**
1. ✅ Проверку системы
2. ✅ Просмотр последнего блока
3. ✅ Загрузку нового документа
4. ✅ Ожидание подписей клиентов
5. ✅ Проверку статуса консенсуса
6. ✅ Просмотр подписей
7. ✅ Проверку категорий
8. ✅ Массовую загрузку (Bulk Upload)
9. ✅ Проверку метрик
10. ✅ Финальную статистику

---

## 📋 Ручная демонстрация

### Загрузка документа

```bash
# Создать тестовый документ
echo "Test Contract" > contract.txt

# Загрузить
curl -X POST http://localhost:8080/api/upload \
  -F "file=@contract.txt"

# Ответ:
# {
#   "hash": "abc123...",
#   "block_hash": "def456...",
#   "filename": "contract.txt"
# }
```

### Проверка консенсуса

```bash
# Получить хэш блока
BLOCK_HASH=$(curl -s http://localhost:8080/api/blocks/last | jq -r '.hash')

# Проверить статус
curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus" | jq
```

### Веб-интерфейс

Откройте в браузере:
```
http://localhost:8080/web/login?token=demo_token
```

**Токен:** `demo_token`

---

## 🛑 Остановка и очистка

### Остановка

```bash
./demo/demo-stop.sh
```

### Очистка

```bash
./demo/demo-cleanup.sh
```

**Будет удалено:**
- База данных блокчейна
- Загруженные файлы
- Логи
- Сгенерированные ключи

---

## 📊 Демонстрационные сценарии

### Сценарий 1: Базовая загрузка

```bash
# 1. Загрузить документ
curl -X POST http://localhost:8080/api/upload \
  -F "file=@document.pdf"

# 2. Подождать 5-10 секунд

# 3. Проверить консенсус
curl http://localhost:8080/api/blocks/last/consensus | jq
```

### Сценарий 2: Массовая загрузка

```bash
# Загрузить 5 документов в одну категорию
curl -X POST http://localhost:8080/api/upload/bulk \
  -F "files=@doc1.pdf" \
  -F "files=@doc2.pdf" \
  -F "files=@doc3.pdf" \
  -F "files=@doc4.pdf" \
  -F "files=@doc5.pdf" \
  -F "category=diplomas"
```

### Сценарий 3: Проверка категорий

```bash
# Создать категорию
curl -X POST http://localhost:8080/api/categories \
  -H "Content-Type: application/json" \
  -d '{"id":"contracts","name":"Договоры"}'

# Получить документы категории
curl http://localhost:8080/api/categories/contracts/documents | jq
```

### Сценарий 4: Мониторинг

```bash
# Метрики Prometheus
curl http://localhost:8080/metrics | grep "^chaindocs_"

# Ключевые метрики:
# - chaindocs_blocks_total
# - chaindocs_active_keys
# - chaindocs_consensus_percent
```

---

## 🎯 Ключевые точки для демонстрации

### 1. Неизменяемость блокчейна

```bash
# Загрузить документ
# Получить хэш
# Показать что хэш записан в блокчейн
# Попытаться изменить — невозможно!
```

### 2. Консенсус 51%+

```bash
# Загрузить документ
# Показать прогресс подписей (1/2, 2/2...)
# Показать сообщение "CONSENSUS REACHED"
```

### 3. P2P коммуникация

```bash
# Запустить 3 клиента
# Показать логи клиентов
# Показать как блоки распространяются между пирами
```

### 4. Категории документов

```bash
# Создать категорию "Дипломы"
# Загрузить документы с категорией
# Показать фильтрацию по категории
```

### 5. Безопасность

```bash
# Показать страницу входа /web/login
# Объяснить CHAINDOCS_AUTH_TOKEN
# Показать отзыв ключей через API
```

---

## 🔧 Конфигурация

### Сервер (server-config.json)

```json
{
  "port": 8080,
  "db_path": "demo_blockchain.db",
  "upload_dir": "./demo_uploads",
  "log_file": "./demo_logs/server.log",
  "log_level": "info",
  "consensus": {
    "type": "percentage",
    "percentage": 51,
    "min_signatures": 2,
    "use_active_keys": true
  }
}
```

### Клиенты (client*-config.json)

```json
{
  "server": "http://localhost:8080",
  "key_file": "demo-keys/client1.enc",
  "password_env": "CHAINDOCS_CLIENT1_PASSWORD",
  "mode": "daemon",
  "daemon": {
    "interval": "5s",
    "sign_unsigned_only": false
  }
}
```

---

## 📞 Поддержка

При возникновении проблем:

1. Проверьте логи:
   ```bash
   tail -f demo/demo_logs/server.log
   tail -f demo/demo_logs/client1.log
   ```

2. Проверьте что порты свободны:
   ```bash
   lsof -i :8080
   ```

3. Перезапустите:
   ```bash
   ./demo/demo-stop.sh
   ./demo/demo-cleanup.sh
   ./demo/demo-setup.sh
   ./demo/demo-run.sh
   ```

---

**Версия:** 1.2.0  
**Дата:** 2026-02-22  
**Статус:** ✅ Ready for Demo

# ChainDocs Demonстрационная среда

## 🚀 Быстрый старт

### Полный запуск одной командой

```bash
./demo/demo-start.sh
```

Этот скрипт автоматически:
1. Очистит предыдущую демонстрацию
2. Соберёт проект
3. Сгенерирует ключи для 3 клиентов
4. Запустит сервер
5. Зарегистрирует ключи на сервере
6. Запустит 3 клиентов-демона
7. Загрузит тестовый документ
8. Дождётся подписей от клиентов (консенсус)

**После запуска:**
- Веб-интерфейс: http://localhost:8080/web/login?token=demo_token
- API: http://localhost:8080/api/blocks/last

---

## 📋 Управление демонстрацией

### Запуск

```bash
./demo/demo-start.sh
```

### Остановка

```bash
./demo/demo-stop.sh
```

### Полная очистка

```bash
./demo/demo-cleanup.sh
```

---

## 🔄 Типовой сценарий работы

### 1. Запуск системы

```bash
cd /path/to/ChainDocs
./demo/demo-start.sh
```

**Ожидаемый результат:**
- ✅ Сервер запущен на порту 8080
- ✅ 3 клиента-демона работают
- ✅ Тестовый документ загружен и подписан
- ✅ Консенсус достигнут (3/2 подписи)

### 2. Загрузка нового документа

```bash
curl -X POST http://localhost:8080/api/upload \
  -F "file=@ваш_файл.pdf"
```

**Ожидаемый результат:**
- ✅ Создаётся новый блок
- ✅ Через 5-10 секунд все 3 клиента подписывают блок
- ✅ Консенсус достигается автоматически

### 3. Проверка статуса

```bash
# Последний блок
curl -s http://localhost:8080/api/blocks/last | jq

# Статус консенсуса
BLOCK_HASH=$(curl -s http://localhost:8080/api/blocks/last | jq -r '.hash')
curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus" | jq
```

### 4. Просмотр логов

```bash
# Логи сервера
tail -f demo/demo_logs/server.log

# Логи клиентов
tail -f demo/demo_logs/client1.log
tail -f demo/demo_logs/client2.log
tail -f demo/demo_logs/client3.log
```

### 5. Остановка

```bash
./demo/demo-stop.sh
```

---

## 📊 API Endpoints

| Endpoint | Описание | Пример |
|----------|----------|--------|
| `GET /api/blocks` | Все блоки | `curl localhost:8080/api/blocks` |
| `GET /api/blocks/last` | Последний блок | `curl localhost:8080/api/blocks/last` |
| `GET /api/blocks/{hash}` | Блок по хэшу | `curl localhost:8080/api/blocks/<hash>` |
| `GET /api/blocks/{hash}/consensus` | Статус консенсуса | `curl localhost:8080/api/blocks/<hash>/consensus` |
| `POST /api/register` | Зарегистрировать ключ | `curl -X POST localhost:8080/api/register -d '{"public_key":"..."}'` |
| `POST /api/sign` | Отправить подпись | `curl -X POST localhost:8080/api/sign -d '{...}'` |
| `POST /api/upload` | Загрузить документ | `curl -X POST localhost:8080/api/upload -F "file=@doc.pdf"` |
| `GET /api/keys` | Список ключей | `curl localhost:8080/api/keys` |
| `GET /api/keys/active` | Активные ключи | `curl localhost:8080/api/keys/active` |

---

## 🔧 Скрипты демонстрации

| Скрипт | Описание |
|--------|----------|
| `demo-start.sh` | **Полный запуск** - рекомендуется для демонстрации |
| `demo-stop.sh` | Корректная остановка всех сервисов |
| `demo-cleanup.sh` | Полная очистка (данные + ключи + логи) |

---

## ⚙️ Как это работает

### Архитектура

```
                    ┌─────────────┐
                    │   Сервер    │
                    │  :8080      │
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
   ┌─────▼─────┐   ┌──────▼─────┐   ┌──────▼─────┐
   │ Клиент 1  │   │ Клиент 2   │   │ Клиент 3   │
   │ (daemon)  │   │ (daemon)   │   │ (daemon)   │
   └───────────┘   └────────────┘   └────────────┘
```

### Поток работы

1. **Запуск сервера** → создаётся genesis block
2. **Генерация ключей** → каждый клиент получает пару ключей
3. **Регистрация ключей** → сервер запоминает публичные ключи
4. **Запуск клиентов** → клиенты подключаются к серверу
5. **Загрузка документа** → создаётся новый блок
6. **Автоматическая подпись** → клиенты подписывают блок каждые 5 сек
7. **Консенсус** → при 2+ подписях блок подтверждён

### Режимы клиентов

Клиенты работают в режиме **daemon**:
- Проверяют сервер каждые 5 секунд
- Автоматически подписывают неподписанные блоки
- Работают пока не будут остановлены

---

## 🎯 Примеры использования

### Загрузить PDF документ

```bash
curl -X POST http://localhost:8080/api/upload \
  -F "file=@document.pdf"
```

### Проверить последний блок

```bash
curl -s http://localhost:8080/api/blocks/last | jq
```

### Проверить подписи

```bash
BLOCK_HASH=$(curl -s http://localhost:8080/api/blocks/last | jq -r '.hash')
curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus" | jq
```

### Получить все блоки

```bash
curl -s http://localhost:8080/api/blocks | jq '.[] | {height, hash: .hash[0:16], signatures: (.signatures | length)}'
```

### Проверить зарегистрированные ключи

```bash
curl -s http://localhost:8080/api/keys | jq
```

---

## 🐛 Решение проблем

### Клиенты не подписывают блоки

**Проблема:** Ключи не зарегистрированы или клиенты не запущены

**Решение:**
```bash
# Полная очистка и перезапуск
./demo/demo-cleanup.sh
./demo/demo-start.sh
```

### Сервер не запускается

**Проблема:** Порт 8080 занят

**Решение:**
```bash
# Найти процесс
lsof -i :8080

# Остановить
kill <PID>

# Или использовать другой порт в config.json
```

### Консенсус не достигается

**Проблема:** Запущены не все клиенты

**Решение:**
```bash
# Проверить количество подписей
curl -s http://localhost:8080/api/blocks/last/consensus | jq '.signatures'

# Должно быть >= 2
# Если меньше - проверить логи клиентов
tail -f demo/demo_logs/client1.log
```

### Ошибка "Only PDF files allowed"

**Проблема:** Сервер принимает только файлы с расширением .pdf

**Решение:**
```bash
# Переименовать файл
mv document.txt document.pdf

# Или создать правильный PDF
echo "%PDF-1.4" > doc.pdf
echo "Content" >> doc.pdf
echo "%%EOF" >> doc.pdf
```

---

## 📁 Структура файлов

```
demo/
├── demo-start.sh          # Полный запуск (рекомендуется)
├── demo-stop.sh           # Остановка всех сервисов
├── demo-cleanup.sh        # Полная очистка
├── demo-keys/
│   ├── client1.enc        # Ключ клиента 1
│   ├── client2.enc        # Ключ клиента 2
│   ├── client3.enc        # Ключ клиента 3
│   └── public_keys.txt    # Публичные ключи
├── demo_uploads/          # Загруженные документы
├── demo_logs/
│   ├── server.log         # Лог сервера
│   ├── client1.log        # Лог клиента 1
│   ├── client2.log        # Лог клиента 2
│   └── client3.log        # Лог клиента 3
├── demo_blockchain.db     # База данных блокчейна
└── README.md              # Этот файл
```

---

## 🎉 Готово!

После запуска `./demo/demo-start.sh` у вас есть:
- ✅ Работающий сервер блокчейна
- ✅ 3 клиента-демона, которые автоматически подписывают блоки
- ✅ Тестовый документ в блокчейне с подписями
- ✅ Достигнутый консенсус

Теперь можно загружать новые документы - они будут **автоматически подписаны** всеми клиентами!

---

## 📞 Дополнительные команды

```bash
# Проверить статус всех сервисов
ps aux | grep "demo/bin"

# Перезапустить только клиентов (сервер остаётся)
./demo/demo-stop.sh
export CHAINDOCS_CLIENT1_PASSWORD="demo123"
export CHAINDOCS_CLIENT2_PASSWORD="demo123"
export CHAINDOCS_CLIENT3_PASSWORD="demo123"
./demo/bin/client -config demo/client1-config.json > demo/demo_logs/client1.log 2>&1 &
./demo/bin/client -config demo/client2-config.json > demo/demo_logs/client2.log 2>&1 &
./demo/bin/client -config demo/client3-config.json > demo/demo_logs/client3.log 2>&1 &
```

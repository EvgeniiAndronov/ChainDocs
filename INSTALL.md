# 🛠️ ПОЛНАЯ ИНСТРУКЦИЯ ПО УСТАНОВКЕ И НАСТРОЙКЕ CHAINDOCS

## Содержание

1. [Быстрый старт (Демо)](#1-быстрый-старт-демо)
2. [Установка сервера](#2-установка-сервера)
3. [Установка клиента-демона](#3-установка-клиента-демона)
4. [Боевое тестирование](#4-боевое-тестирование)
5. [Мониторинг и управление](#5-мониторинг-и-управление)

---

## 1. Быстрый старт (Демо)

**Рекомендуется для знакомства с системой!**

```bash
# Клонировать репозиторий
git clone https://github.com/EvgeniiAndronov/ChainDocs.git
cd ChainDocs

# Запустить демонстрацию
make demo-start
# или
./demo/demo-start.sh
```

**Через 30 секунд:**
- ✅ Сервер запущен на `http://localhost:8080`
- ✅ 3 клиента-демона автоматически подписывают блоки
- ✅ Тестовый документ загружен и подписан (консенсус достигнут)

**Веб-интерфейс:** http://localhost:8080/web/login?token=demo_token

**Управление демо:**
```bash
make demo-stop   # Остановка
make demo-clean  # Очистка
```

**Полная документация:** [demo/README.md](demo/README.md)

---

## 2. Установка сервера

### Вариант A: Docker (рекомендуется)

```bash
# 1. Клонируем репозиторий
git clone https://github.com/EvgeniiAndronov/ChainDocs.git
cd ChainDocs

# 2. Запускаем через docker-compose
docker-compose up -d

# 3. Проверяем статус
docker-compose ps
docker-compose logs -f chaindocs-server
```

**Сервер доступен:** `http://localhost:8080`

### Вариант B: Локальный запуск

```bash
# 1. Сборка (все компоненты)
make build

# 2. Запуск сервера
make run

# Или вручную:
./bin/server
```

---

## 2. Установка клиента-демона

### Шаг 2.1: Подготовка

```bash
# 1. Сборка клиента
go build -o bin/client ./cmd/client/main.go ./cmd/client/config.go

# 2. Генерация ключа клиента
./bin/keygen -password "SecurePassword123!" -out client1.enc

# Сохраните публичный ключ из вывода!
# Пример: 5d1cb17def07135434c50169817062037fcfbf420ca9c9bf5eade6c8f39cf155
```

### Шаг 2.2: Регистрация ключа на сервере

```bash
# Зарегистрируйте публичный ключ (замените на ваш)
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"public_key":"5d1cb17def07135434c50169817062037fcfbf420ca9c9bf5eade6c8f39cf155"}'
```

### Шаг 2.3: Создание конфигурации

```bash
# 1. Генерируем пример конфига
./bin/client -gen-config > config.json

# 2. Редактируем конфиг
nano config.json
```

**Пример config.json:**
```json
{
  "server": "http://localhost:8080",
  "key_file": "client1.enc",
  "password_env": "CHAINDOCS_KEY_PASSWORD",
  "mode": "daemon",
  "daemon": {
    "interval": "10s",
    "max_blocks_per_cycle": 0,
    "sign_unsigned_only": true,
    "stop_on_consensus": true
  },
  "logging": {
    "level": "info",
    "file": "",
    "format": "text"
  },
  "self_healing": {
    "enabled": true,
    "alert_on_foreign_signature": true,
    "alert_webhook": "",
    "auto_revoke": false
  }
}
```

### Шаг 2.4: Установка как демон

#### Linux (systemd)

```bash
# 1. Создаём директорию
sudo mkdir -p /opt/chaindocs
sudo mkdir -p /opt/chaindocs/logs
sudo mkdir -p /opt/chaindocs/data

# 2. Копируем файлы
sudo cp bin/client /opt/chaindocs/chaindocs-client
sudo cp config.json /opt/chaindocs/config.json
sudo cp client1.enc /opt/chaindocs/client1.enc

# 3. Устанавливаем права
sudo chmod +x /opt/chaindocs/chaindocs-client
sudo chmod 600 /opt/chaindocs/config.json /opt/chaindocs/client1.enc
sudo chown -R root:root /opt/chaindocs

# 4. Устанавливаем systemd сервис
sudo cp scripts/install/chaindocs-client.service /etc/systemd/system/
sudo systemctl daemon-reload

# 5. Включаем и запускаем
sudo systemctl enable chaindocs-client
sudo systemctl start chaindocs-client

# 6. Проверяем статус
sudo systemctl status chaindocs-client
```

**Полезные команды:**
```bash
# Просмотр логов
sudo journalctl -u chaindocs-client -f

# Перезапуск
sudo systemctl restart chaindocs-client

# Остановка
sudo systemctl stop chaindocs-client

# Удаление
sudo systemctl disable chaindocs-client
sudo rm /etc/systemd/system/chaindocs-client.service
sudo systemctl daemon-reload
```

#### macOS (launchd)

```bash
# 1. Создаём директорию
sudo mkdir -p /opt/chaindocs
sudo mkdir -p /opt/chaindocs/logs

# 2. Копируем файлы
sudo cp bin/client /opt/chaindocs/chaindocs-client
sudo cp config.json /opt/chaindocs/config.json
sudo cp client1.enc /opt/chaindocs/client1.enc

# 3. Устанавливаем права
sudo chmod +x /opt/chaindocs/chaindocs-client
sudo chmod 600 /opt/chaindocs/config.json /opt/chaindocs/client1.enc

# 4. Устанавливаем launchd plist
sudo cp scripts/install/com.chaindocs.client.plist /Library/LaunchDaemons/

# 5. Загружаем и запускаем
sudo launchctl load /Library/LaunchDaemons/com.chaindocs.client.plist
sudo launchctl start com.chaindocs.client

# 6. Проверяем
sudo launchctl list | grep chaindocs
```

**Полезные команды:**
```bash
# Просмотр логов
tail -f /opt/chaindocs/logs/chaindocs-client.log

# Перезапуск
sudo launchctl unload /Library/LaunchDaemons/com.chaindocs.client.plist
sudo launchctl load /Library/LaunchDaemons/com.chaindocs.client.plist

# Удаление
sudo launchctl unload /Library/LaunchDaemons/com.chaindocs.client.plist
sudo rm /Library/LaunchDaemons/com.chaindocs.client.plist
```

### Альтернатива: Автоматическая установка

```bash
# Простая установка (Linux/macOS)
sudo ./scripts/install/install-client.sh \
  -b ./bin/client \
  -c ./config.json \
  -d

# Удаление
sudo ./scripts/install/install-client.sh --uninstall
```

---

## 4. Боевое тестирование

### Вариант A: Через Make (рекомендуется)

```bash
# Все тесты
make test

# Только боевые тесты
make test-live
```

### Вариант B: Ручной запуск

```bash
chmod +x test-live.sh
./test-live.sh
```

### Ожидаемый результат

```
🧪 ChainDocs Live Tests
════════════════════════════════════════
Тест 1: Запуск сервера
✅ Сервер запущен и отвечает
Тест 2: Генерация ключей
✅ Ключи сгенерированы
Тест 3: Регистрация ключей
✅ Зарегистрировано ключей: 3
Тест 4: Загрузка документа
✅ Документ загружен
Тест 5: Запуск клиентов-демонов
✅ Клиенты запущены
Тест 6: Проверка консенсуса
✅ Консенсус достигнут (3 из 2)
Тест 7: Проверка блока
✅ Блок подписан (3 подписей)
Тест 8: Второй документ (автоматическая подпись)
✅ Второй блок автоматически подписан
════════════════════════════════════════
✅ Все тесты пройдены!
```

**Тесты проверяют:**
1. Запуск сервера
2. Генерацию 3 ключей
3. Регистрацию ключей
4. Загрузку PDF документа
5. Запуск клиентов-демонов
6. Достижение консенсуса (51%+)
7. Наличие подписей в блоке
8. Автоматическую подпись новых блоков

---

## 5. Мониторинг и управление

### API для мониторинга

```bash
# Последний блок
curl http://localhost:8080/api/blocks/last | jq

# Все блоки
curl http://localhost:8080/api/blocks | jq 'length'

# Статус консенсуса для блока
curl http://localhost:8080/api/blocks/<HASH>/consensus | jq

# Зарегистрированные ключи
curl http://localhost:8080/api/keys | jq

# Отозванные ключи
curl http://localhost:8080/api/keys/revoked | jq

# Загруженные документы
ls -la uploads/
```

### Логи

```bash
# Сервер (journalctl)
sudo journalctl -u chaindocs-server -f

# Клиент (journalctl)
sudo journalctl -u chaindocs-client -f

# Клиент (файл)
tail -f /opt/chaindocs/logs/chaindocs-client.log
```

---

## 📊 Чек-лист проверки

- [ ] Сервер запущен и отвечает на `/api/blocks/last`
- [ ] Ключи сгенерированы и зарегистрированы
- [ ] Клиент установлен как демон
- [ ] Демон запускается автоматически после перезагрузки
- [ ] Блоки подписываются клиентами
- [ ] Консенсус достигается при 51%+ подписей
- [ ] Логи пишутся без ошибок

---

## 🆘 Troubleshooting

### Клиент не подключается к серверу

```bash
# Проверьте доступность сервера
curl http://localhost:8080/api/blocks/last

# Проверьте конфиг
cat /opt/chaindocs/config.json | jq '.server'

# Проверьте права на ключ
ls -la /opt/chaindocs/client1.enc
```

### Демон не запускается

```bash
# Linux: проверка статуса
sudo systemctl status chaindocs-client

# Linux: подробные логи
sudo journalctl -u chaindocs-client -n 50

# macOS: проверка
sudo launchctl list | grep chaindocs

# macOS: логи
tail -f /opt/chaindocs/logs/chaindocs-client.log
```

### Ошибка "Password required"

```bash
# Установите переменную окружения в systemd сервисе
sudo systemctl edit chaindocs-client

# Добавьте:
[Service]
Environment="CHAINDOCS_KEY_PASSWORD=your_password"

# Перезапустите
sudo systemctl daemon-reload
sudo systemctl restart chaindocs-client
```

---

**Версия:** 0.2.0  
**Дата:** 2026-02-22

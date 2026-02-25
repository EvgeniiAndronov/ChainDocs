# 🚀 ChainDocs - Шпаргалка

## Быстрый старт (30 секунд)

```bash
# Клонировать и запустить демо
git clone https://github.com/EvgeniiAndronov/ChainDocs.git
cd ChainDocs
make demo-start
```

**Готово!** Система работает:
- Сервер: http://localhost:8080
- Веб-интерфейс: http://localhost:8080/web/login?token=demo_token
- 3 клиента автоматически подписывают блоки

---

## Основные команды

### 🔨 Сборка
```bash
make build          # Собрать всё
make build-server   # Только сервер
make build-client   # Только клиент
```

### 🏃 Запуск
```bash
make run            # Сервер
make demo-start     # Демо (сервер + 3 клиента)
make demo-stop      # Остановить демо
```

### 🧪 Тесты
```bash
make test           # Все тесты
make test-live      # Боевые тесты (8 тестов)
```

### 🐳 Docker
```bash
make docker-build   # Сборка образов
make docker-up      # Запуск
make docker-down    # Остановка
```

### 🧹 Очистка
```bash
make clean          # Очистить билды
make demo-clean     # Очистить демо
make clean-all      # Полная очистка
```

---

## API (примеры)

### Загрузить документ
```bash
curl -X POST http://localhost:8080/api/upload \
  -F "file=@document.pdf"
```

### Последний блок
```bash
curl -s http://localhost:8080/api/blocks/last | jq
```

### Консенсус
```bash
BLOCK_HASH=$(curl -s http://localhost:8080/api/blocks/last | jq -r '.hash')
curl -s "http://localhost:8080/api/blocks/$BLOCK_HASH/consensus" | jq
```

### Ключи
```bash
curl -s http://localhost:8080/api/keys | jq
```

---

## Производство

### Развёртывание
```bash
# Production стек
./scripts/deploy.sh --production

# Backup
./scripts/backup.sh

# Restore
./scripts/restore.sh --file backup.db.gz
```

### Мониторинг
```bash
# Метрики
curl http://localhost:8080/metrics

# Логи
docker-compose logs -f chaindocs-server
```

---

## Troubleshooting

### Сервер не запускается
```bash
lsof -i :8080    # Найти процесс
kill <PID>       # Остановить
```

### Клиенты не подписывают
```bash
# Проверить логи
tail -f demo/demo_logs/client1.log

# Перезапустить демо
make demo-restart
```

### Тесты падают
```bash
# Очистить и запустить заново
make clean-all
make test-live
```

---

## Документация

- **README.md** - Основная документация
- **INSTALL.md** - Полная инструкция по установке
- **demo/README.md** - Демонстрационная среда
- **PRODUCTION.md** - Production развёртывание
- **AUDIT_SUMMARY.md** - Отчёт по изменениям

---

**Версия:** 1.0.0  
**Статус:** ✅ Production Ready

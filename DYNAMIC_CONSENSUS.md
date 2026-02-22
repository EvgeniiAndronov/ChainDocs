# 🎯 Реализация динамического консенсуса

## Проблема

При 50 зарегистрированных ключах и 10 активных клиентах:
- Требуется подписей (51%): 26
- Реально можно получить: 10
- **Итог: Консенсус НЕВОЗМОЖЕН** ❌

## Решение: Комбинированный подход

### 1. Activity Tracking

Отслеживание активности ключей в реальном времени:

```go
type KeyActivity struct {
    PublicKey  string `json:"public_key"`
    LastSeen   string `json:"last_seen"`   // RFC3339 timestamp
    BlockCount int64  `json:"block_count"` // количество подписей
}
```

**Хранение:**
- Bucket `activity` в bbolt БД
- Обновляется при каждой подписи блока
- Автоматическая очистка старых записей

### 2. Динамический расчёт консенсуса

```go
// Вместо: required = totalKeys / 2 + 1
// Считаем: required = activeKeys / 2 + 1

activeKeys = GetActiveKeys(last 24 hours)
required = max(2, activeKeys / 2 + 1)  // минимум 2 подписи
```

**Пример:**
```
Зарегистрировано: 50
Активных (24ч):   10
Требуется:        10/2 + 1 = 6 подписей ✅
```

### 3. Статусы блоков

| Статус | Описание |
|--------|----------|
| `pending` | Ожидает первой подписи |
| `partially_signed` | Есть подписи, но меньше порога |
| `consensus_reached` | Консенсус достигнут |
| `finalized` | Окно подписания закрыто |

---

## API

### GET /api/keys/active

Возвращает активные ключи за период:

```bash
curl http://localhost:8080/api/keys/active?window=24h
```

**Ответ:**
```json
{
  "window": "24h0m0s",
  "count": 3,
  "activities": [
    {
      "public_key": "abc123...",
      "last_seen": "2026-02-22T15:00:00Z",
      "block_count": 5
    }
  ]
}
```

### GET /api/blocks/{hash}/consensus

Обновлённый ответ с active_keys:

```json
{
  "block_hash": "abc...",
  "height": 1,
  "total_keys": 3,
  "active_keys": 3,
  "signatures": 2,
  "required": 2,
  "percent": 100,
  "consensus_reached": true,
  "signatures_list": [...]
}
```

---

## Конфигурация сервера

Создан файл `cmd/server/config.go`:

```go
type ServerConfig struct {
    Consensus ConsensusConfig `json:"consensus"`
    Activity  ActivityConfig  `json:"activity"`
}

type ConsensusConfig struct {
    Type          string `json:"type"`           // "percentage" или "fixed"
    Percentage    int    `json:"percentage"`     // 51 = 51%
    MinSignatures int    `json:"min_signatures"` // минимум 2
    MaxSignatures int    `json:"max_signatures"` // 0 = без ограничений
    UseActiveKeys bool   `json:"use_active_keys"` // true
}

type ActivityConfig struct {
    Window      Duration `json:"window"`       // "24h"
    AutoCleanup bool     `json:"auto_cleanup"` // true
}
```

**По умолчанию:**
- Консенсус: 51% от активных
- Минимум подписей: 2
- Окно активности: 24 часа

---

## Утилиты

### scripts/clean.sh

Очистка тестовых данных:

```bash
./scripts/clean.sh

# Удаляет:
# - blockchain.db
# - uploads/*
# - *.enc (ключи)
# - test_*.db
```

### test-live.sh

Боевой тест с динамическим консенсусом:

```bash
./test-live.sh

# Результат:
# ✅ 4/4 тестов пройдено
# - 3 ключа зарегистрировано
# - Блок создан
# - 2 подписи собрано
# - Консенсус достигнут
```

---

## Надёжность системы

### Защита от проблем

| Проблема | Решение |
|----------|---------|
| Мало активных клиентов | Динамический расчёт от активных |
| Ключи не активны давно | Fallback на все зарегистрированные |
| Слишком мало подписей | Минимальный порог: 2 подписи |
| Ключ скомпрометирован | API отзыва ключей (revocation) |

### Monitoring

```bash
# Проверка активности
curl http://localhost:8080/api/keys/active

# Проверка консенсуса
curl http://localhost:8080/api/blocks/last/consensus

# Логи сервера
tail -f /tmp/server.log | grep "CONSENSUS"
```

---

## Тестирование

### Сценарий 1: 3 клиента, все активны

```
Зарегистрировано: 3
Активных:         3
Требуется:        3/2 + 1 = 2 подписи

Клиент 1: подписал (1/2)
Клиент 2: подписал (2/2) ✅ КОНСЕНСУС
Клиент 3: видит консенсус, пропускает
```

### Сценарий 2: 50 клиентов, 10 активны

```
Зарегистрировано: 50
Активных:         10
Требуется:        10/2 + 1 = 6 подписей

Подписали 6 клиентов ✅ КОНСЕНСУС
Остальные 4 могут подписать для надёжности
```

### Сценарий 3: Нет активных (новая БД)

```
Зарегистрировано: 5
Активных:         0
Fallback на все:  5
Требуется:        5/2 + 1 = 3 подписи
```

---

## Рекомендации по развёртыванию

### Production настройки

```json
{
  "consensus": {
    "type": "percentage",
    "percentage": 51,
    "min_signatures": 2,
    "max_signatures": 0,
    "use_active_keys": true
  },
  "activity": {
    "window": "24h",
    "auto_cleanup": true
  }
}
```

### Monitoring dashboard

1. **Активные ключи:** `/api/keys/active`
2. **Консенсус блоков:** `/api/blocks/last/consensus`
3. **Зарегистрированные:** `/api/keys`

### Alerting

Настроить уведомления если:
- Активных ключей < 2 (риск остановки)
- Блок без подписей > 1 часа
- Консенсус не достигнут > 24 часов

---

**Версия:** 0.3.0  
**Дата:** 2026-02-22  
**Статус:** ✅ Готово к production

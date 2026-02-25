# 📁 Категории и Массовая Загрузка Документов

## 🎯 Обзор

ChainDocs поддерживает:
1. **Категории документов** - организация файлов по папкам (например, "Дипломы студентов")
2. **Массовую загрузку** - загрузка множества файлов одновременно (до 100MB)

---

## 📁 Категории документов

### Создание категории

```bash
curl -X POST http://localhost:8080/api/categories \
  -H "Content-Type: application/json" \
  -d '{
    "id": "diplomas",
    "name": "Дипломы студентов",
    "description": "Дипломы выпускников 2026 года"
  }'
```

**Ответ:**
```json
{
  "status": "created",
  "id": "diplomas"
}
```

### Получение всех категорий

```bash
curl -s http://localhost:8080/api/categories | jq
```

**Ответ:**
```json
{
  "count": 2,
  "categories": [
    {
      "id": "diplomas",
      "name": "Дипломы студентов",
      "description": "Дипломы выпускников 2026 года",
      "created": "2026-02-23T12:00:00Z",
      "doc_count": 50
    },
    {
      "id": "contracts",
      "name": "Договоры",
      "description": "Учебные договоры",
      "created": "2026-02-23T12:05:00Z",
      "doc_count": 10
    }
  ]
}
```

### Получение документов категории

```bash
curl -s http://localhost:8080/api/categories/diplomas/documents | jq
```

### Удаление категории

```bash
curl -X DELETE http://localhost:8080/api/categories/diplomas
```

---

## 📤 Загрузка с категорией

### Одиночная загрузка

```bash
curl -X POST http://localhost:8080/api/upload \
  -F "file=@diploma.pdf" \
  -F "category=diplomas"
```

**Ответ:**
```json
{
  "hash": "abc123...",
  "filename": "diploma.pdf",
  "size": 1024567,
  "uploaded": "2026-02-23T12:00:00Z",
  "block_hash": "def456...",
  "category": "diplomas"
}
```

### Массовая загрузка

```bash
curl -X POST http://localhost:8080/api/upload/bulk \
  -F "files=@diploma1.pdf" \
  -F "files=@diploma2.pdf" \
  -F "files=@diploma3.pdf" \
  -F "category=diplomas"
```

**Ответ:**
```json
{
  "total": 3,
  "success": 3,
  "failed": 0,
  "category": "diplomas",
  "results": [
    {
      "filename": "diploma1.pdf",
      "hash": "abc1...",
      "block_hash": "def1...",
      "size": 1024000,
      "success": true
    },
    {
      "filename": "diploma2.pdf",
      "hash": "abc2...",
      "block_hash": "def2...",
      "size": 1025000,
      "success": true
    },
    {
      "filename": "diploma3.pdf",
      "hash": "abc3...",
      "block_hash": "def3...",
      "size": 1026000,
      "success": true
    }
  ]
}
```

---

## 📂 Структура хранения

```
uploads/
├── diplomas/
│   ├── abc123...pdf
│   ├── def456...pdf
│   └── ...
├── contracts/
│   ├── xyz789...pdf
│   └── ...
└── other/
    └── ...
```

**Примечание:** Файлы сохраняются под своими хэшами, но в папках по категориям.

---

## 🔒 Ограничения

### Массовая загрузка

| Параметр | Значение |
|----------|----------|
| **Макс. размер** | 100 MB (общий) |
| **Макс. файлов** | Не ограничено (пока не превышен размер) |
| **Тип файлов** | Только PDF |
| **Время обработки** | ~1-2 сек на файл |

### Рекомендации

- **Оптимально:** 10-50 файлов за раз
- **Для больших объёмов:** разбивайте на пакеты по 50 файлов
- **Очень большие файлы:** загружайте по одному

---

## 🧪 Примеры использования

### Пример 1: Загрузка 50 дипломов

```bash
#!/bin/bash
# Создаём список файлов
FILES=""
for i in {1..50}; do
    FILES="$FILES -F files=@diploma_$i.pdf"
done

# Загружаем все сразу
curl -X POST http://localhost:8080/api/upload/bulk \
  -F "category=diplomas" \
  $FILES | jq
```

### Пример 2: Создание категории через Web UI

```html
<!-- Форма создания категории -->
<form id="createCategory">
  <input type="text" id="catId" placeholder="ID (например, diplomas)" required>
  <input type="text" id="catName" placeholder="Название" required>
  <textarea id="catDesc" placeholder="Описание"></textarea>
  <button type="submit">Создать категорию</button>
</form>

<script>
document.getElementById('createCategory').addEventListener('submit', async (e) => {
  e.preventDefault();
  
  const response = await fetch('/api/categories', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
      id: document.getElementById('catId').value,
      name: document.getElementById('catName').value,
      description: document.getElementById('catDesc').value
    })
  });
  
  const result = await response.json();
  alert('Категория создана: ' + result.id);
});
</script>
```

### Пример 3: Массовая загрузка с прогрессом

```javascript
async function uploadMultipleFiles(files, category) {
  const formData = new FormData();
  formData.append('category', category);
  
  for (const file of files) {
    formData.append('files', file);
  }
  
  const response = await fetch('/api/upload/bulk', {
    method: 'POST',
    body: formData
  });
  
  const result = await response.json();
  
  console.log(`Загружено: ${result.success}/${result.total}`);
  console.log(`Категория: ${result.category}`);
  
  return result;
}

// Использование
const files = document.querySelector('input[type=file]').files;
uploadMultipleFiles(files, 'diplomas');
```

---

## 📊 Мониторинг

### Статистика по категории

```bash
# Получить информацию о категории
curl -s http://localhost:8080/api/categories/diplomas | jq

# Получить документы категории
curl -s http://localhost:8080/api/categories/diplomas/documents | jq '.count'
```

### Проверка хранилища

```bash
# Размер папки категории
du -sh uploads/diplomas/

# Количество файлов
ls -1 uploads/diplomas/*.pdf | wc -l
```

---

## ⚠️ Troubleshooting

### Ошибка: "Failed to create upload directory"

**Решение:**
```bash
# Проверить права
ls -la uploads/

# Исправить права
chmod 755 uploads/
chown -R $(whoami) uploads/
```

### Ошибка: "Only PDF files allowed"

**Решение:** Убедитесь что файлы имеют расширение `.pdf`

### Массовая загрузка не работает

**Проверка:**
```bash
# Проверить размер файлов
du -sh uploads/

# Проверить логи сервера
tail -f demo_logs/server.log | grep "Bulk"
```

---

## 🎯 Best Practices

### Для категорий

1. **Используйте понятные ID:** `diplomas_2026`, `contracts_moсква`
2. **Описывайте назначение:** "Дипломы бакалавров 2026 года"
3. **Не создавайте слишком много:** 10-20 категорий достаточно

### Для массовой загрузки

1. **Разбивайте на пакеты:** по 20-50 файлов
2. **Проверяйте размер:** не более 100MB за раз
3. **Следите за прогрессом:** обрабатывайте ответ API
4. **Логируйте ошибки:** сохраняйте `result.failed`

---

**Версия:** 2.0.0  
**Дата:** 2026-02-23

# Godoc документация ChainDocs

## Генерация документации

### Локальный просмотр

```bash
# Установить godoc (если нет)
go get golang.org/x/tools/cmd/godoc

# Запустить сервер
godoc -http=:6060

# Открыть в браузере
open http://localhost:6060/pkg/ChainDocs/
```

### Структура документации

```
ChainDocs/
├── cmd/
│   ├── server    — Сервер (REST API + Web UI)
│   ├── client    — Клиент для подписи блоков
│   ├── keygen    — Генерация ключей
│   └── signer    — Утилита подписи сообщений
├── internal/
│   ├── block     — Структура блока и операции
│   ├── crypto    — Криптография (Ed25519, AES-256)
│   ├── storage   — Хранилище (bbolt)
│   └── p2p       — P2P коммуникация
└── pkg/
    ├── logger    — Логирование с ротацией
    └── metrics   — Prometheus метрики
```

### Примеры использования API

#### Создание блока

```go
import "ChainDocs/internal/block"

// Создать новый блок
prevHash := [32]byte{...}
docHash := [32]byte{...}
newBlock := block.NewBlock(1, prevHash, docHash)

// Подписать блок
keyPair, _ := crypto.GenerateKey()
newBlock.Sign(keyPair)

// Проверить подпись
valid := newBlock.VerifySignature(keyPair.PublicKey)
```

#### Подпись документа

```go
import "ChainDocs/internal/crypto"

// Сгенерировать ключи
keyPair, _ := crypto.GenerateKey()

// Подписать документ
docHash := sha256.Sum256(documentData)
signature := keyPair.Sign(docHash[:])

// Проверить подпись
valid := crypto.Verify(keyPair.PublicKey, docHash[:], signature)
```

#### P2P коммуникация

```go
import "ChainDocs/internal/p2p"

// Создать P2P узел
node := p2p.NewP2PNode(peerID, serverURL)

// Запустить
node.Start([]string{"ws://peer1:8081/p2p"})

// Транслировать блок
node.BroadcastBlock(block)
```

### Комментарии в коде

Go использует комментарии для документации. Пример:

```go
// NewBlock создает новый блок с указанной высотой и хэшами.
// height — номер блока в цепочке
// prevHash — хэш предыдущего блока
// docHash — хэш документа
// Возвращает: *Block — новый блок
func NewBlock(height int64, prevHash [32]byte, docHash [32]byte) *Block {
    // ...
}
```

### Интеграция с CI/CD

```yaml
# .github/workflows/docs.yml
name: Generate Docs
on: push
jobs:
  docs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Generate Godoc
        run: |
          go get golang.org/x/tools/cmd/godoc
          godoc -http=:6060 &
```

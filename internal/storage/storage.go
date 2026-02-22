package storage

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"ChainDocs/internal/block"

	"go.etcd.io/bbolt"
)

// RevocationInfo информация об отозванном ключе
type RevocationInfo struct {
	PublicKey string `json:"public_key"`
	Reason    string `json:"reason"`
	RevokedAt string `json:"revoked_at"`
}

var (
	// Бакеты
	bucketBlocks = []byte("blocks")    // хэш блока -> блок
	bucketHeight = []byte("height")    // высота -> хэш блока
	bucketDocs   = []byte("documents") // хэш документа -> хэш блока
	bucketMeta   = []byte("metadata")  // метаданные (последний блок и т.д.)
	bucketPubKeys = []byte("pubkeys")  // зарегистрированные публичные ключи
	bucketRevoked = []byte("revoked")  // отозванные ключи
	bucketActivity = []byte("activity") // активность ключей (key -> last_seen timestamp)
	bucketCategories = []byte("categories") // категории документов

	// Ключи метаданных
	keyLastHash   = []byte("last_hash")
	keyLastHeight = []byte("last_height")
)

// Category категория документов
type Category struct {
	ID          string `json:"id"`          // ID категории
	Name        string `json:"name"`        // Название
	Description string `json:"description"` // Описание
	Created     string `json:"created"`     // Время создания
	DocCount    int64  `json:"doc_count"`   // Количество документов
}

// KeyActivity информация об активности ключа
type KeyActivity struct {
	PublicKey string `json:"public_key"`
	LastSeen  string `json:"last_seen"` // RFC3339 timestamp
	BlockCount int64 `json:"block_count"` // количество подписанных блоков
}

// DocumentMetadata метаданные документа
type DocumentMetadata struct {
	Hash       string `json:"hash"`
	Filename   string `json:"filename"`
	Category   string `json:"category"`
	Size       int64  `json:"size"`
	Uploaded   string `json:"uploaded"`
	BlockHash  string `json:"block_hash"`
	Owner      string `json:"owner,omitempty"` // Публичный ключ владельца
}

type Storage struct {
	db *bbolt.DB
}

// New создает или открывает хранилище
func New(path string) (*Storage, error) {
	// Опции для оптимизации
	opts := &bbolt.Options{
		Timeout: 1 * time.Second,
		// NoSync: true, // Для тестов можно включить, но осторожно!
	}

	db, err := bbolt.Open(path, 0600, opts)
	if err != nil {
		return nil, err
	}

	// Создаем бакеты
	err = db.Update(func(tx *bbolt.Tx) error {
		tx.CreateBucketIfNotExists(bucketBlocks)
		tx.CreateBucketIfNotExists(bucketHeight)
		tx.CreateBucketIfNotExists(bucketDocs)
		tx.CreateBucketIfNotExists(bucketMeta)
		tx.CreateBucketIfNotExists(bucketPubKeys)
		tx.CreateBucketIfNotExists(bucketRevoked)
		tx.CreateBucketIfNotExists(bucketActivity)
		tx.CreateBucketIfNotExists(bucketCategories)
		tx.CreateBucketIfNotExists([]byte("documents_meta"))
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &Storage{db: db}, nil
}

// Close закрывает БД
func (s *Storage) Close() error {
	return s.db.Close()
}

// SaveBlock сохраняет блок в БД
func (s *Storage) SaveBlock(b *block.Block) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		// Сериализуем блок (можно использовать JSON или бинарный формат)
		data, err := b.MarshalJSON()
		if err != nil {
			return err
		}

		// Сохраняем по хэшу
		blocks := tx.Bucket(bucketBlocks)
		if err := blocks.Put(b.Hash[:], data); err != nil {
			return err
		}

		// Сохраняем индекс по высоте
		height := tx.Bucket(bucketHeight)
		heightKey := make([]byte, 8)
		binary.BigEndian.PutUint64(heightKey, uint64(b.Height))
		if err := height.Put(heightKey, b.Hash[:]); err != nil {
			return err
		}

		// Сохраняем индекс по документу
		docs := tx.Bucket(bucketDocs)
		if err := docs.Put(b.DocumentHash[:], b.Hash[:]); err != nil {
			return err
		}

		// Обновляем метаданные (последний блок)
		meta := tx.Bucket(bucketMeta)

		// Получаем текущую высоту
		var currentHeight uint64
		lastHeightData := meta.Get(keyLastHeight)
		if lastHeightData != nil {
			currentHeight = binary.BigEndian.Uint64(lastHeightData)
		}

		// Если это новый блок, обновляем метаданные
		if uint64(b.Height) > currentHeight {
			// Сохраняем хэш последнего блока
			meta.Put(keyLastHash, b.Hash[:])

			// Сохраняем высоту
			heightBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(heightBytes, uint64(b.Height))
			meta.Put(keyLastHeight, heightBytes)
		}

		return nil
	})
}

// GetBlock возвращает блок по хэшу
func (s *Storage) GetBlock(hash [32]byte) (*block.Block, error) {
	var b block.Block

	err := s.db.View(func(tx *bbolt.Tx) error {
		blocks := tx.Bucket(bucketBlocks)
		data := blocks.Get(hash[:])
		if data == nil {
			return errors.New("block not found")
		}

		return b.UnmarshalJSON(data)
	})

	if err != nil {
		return nil, err
	}

	return &b, nil
}

// GetBlockByHeight возвращает блок по высоте
func (s *Storage) GetBlockByHeight(height int64) (*block.Block, error) {
	var hash [32]byte

	err := s.db.View(func(tx *bbolt.Tx) error {
		heightBucket := tx.Bucket(bucketHeight)

		heightKey := make([]byte, 8)
		binary.BigEndian.PutUint64(heightKey, uint64(height))

		hashData := heightBucket.Get(heightKey)
		if hashData == nil {
			return errors.New("block not found")
		}

		if len(hashData) != 32 {
			return errors.New("invalid hash length")
		}

		copy(hash[:], hashData)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetBlock(hash)
}

// GetLastBlock возвращает последний блок в цепочке
func (s *Storage) GetLastBlock() (*block.Block, error) {
	var hash [32]byte

	err := s.db.View(func(tx *bbolt.Tx) error {
		meta := tx.Bucket(bucketMeta)
		lastHash := meta.Get(keyLastHash)
		if lastHash == nil {
			return errors.New("no blocks in chain")
		}

		if len(lastHash) != 32 {
			return errors.New("invalid last hash")
		}

		copy(hash[:], lastHash)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetBlock(hash)
}

// GetBlockByDocument возвращает блок, содержащий документ
func (s *Storage) GetBlockByDocument(docHash [32]byte) (*block.Block, error) {
	var hash [32]byte

	err := s.db.View(func(tx *bbolt.Tx) error {
		docs := tx.Bucket(bucketDocs)
		blockHash := docs.Get(docHash[:])
		if blockHash == nil {
			return errors.New("document not found")
		}

		if len(blockHash) != 32 {
			return errors.New("invalid block hash")
		}

		copy(hash[:], blockHash)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetBlock(hash)
}

// InitGenesis создает генезис-блок, если цепочка пуста
func (s *Storage) InitGenesis() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		// Проверяем, есть ли уже блоки
		blocks := tx.Bucket(bucketBlocks)
		stats := blocks.Stats()
		if stats.KeyN > 0 {
			return nil // Уже есть блоки
		}

		// Создаем генезис-блок
		genesis := block.NewBlock(0, [32]byte{}, [32]byte{})

		data, err := genesis.MarshalJSON()
		if err != nil {
			return err
		}

		// Сохраняем
		if err := blocks.Put(genesis.Hash[:], data); err != nil {
			return err
		}

		// Индекс по высоте
		height := tx.Bucket(bucketHeight)
		heightKey := make([]byte, 8)
		binary.BigEndian.PutUint64(heightKey, 0)
		if err := height.Put(heightKey, genesis.Hash[:]); err != nil {
			return err
		}

		// Метаданные
		meta := tx.Bucket(bucketMeta)
		meta.Put(keyLastHash, genesis.Hash[:])

		heightBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(heightBytes, 0)
		meta.Put(keyLastHeight, heightBytes)

		log.Println("✅ Genesis block created:", genesis.ShortHash())
		return nil
	})
}

// GetAllBlocks возвращает все блоки (для отладки)
func (s *Storage) GetAllBlocks() ([]*block.Block, error) {
	var blocks []*block.Block

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketBlocks)

		return b.ForEach(func(k, v []byte) error {
			var blk block.Block
			if err := blk.UnmarshalJSON(v); err != nil {
				return err
			}
			blocks = append(blocks, &blk)
			return nil
		})
	})

	return blocks, err
}

// GetHeight возвращает текущую высоту цепочки
func (s *Storage) GetHeight() (int64, error) {
	var height int64

	err := s.db.View(func(tx *bbolt.Tx) error {
		meta := tx.Bucket(bucketMeta)
		data := meta.Get(keyLastHeight)
		if data == nil {
			return nil // Нет блоков
		}

		height = int64(binary.BigEndian.Uint64(data))
		return nil
	})

	return height, err
}

// SavePublicKey сохраняет публичный ключ
func (s *Storage) SavePublicKey(pubKey string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketPubKeys)
		if bucket == nil {
			return fmt.Errorf("pubkeys bucket not found")
		}
		return bucket.Put([]byte(pubKey), []byte{1})
	})
}

// GetAllPublicKeys возвращает все ключи
func (s *Storage) GetAllPublicKeys() ([]string, error) {
	var keys []string
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketPubKeys)
		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()
		for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
			keys = append(keys, string(k))
		}
		return nil
	})
	return keys, err
}

// RevokePublicKey отзывает публичный ключ
func (s *Storage) RevokePublicKey(pubKey string, reason string, revokedAt time.Time) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		// Добавляем в bucket отозванных
		bucket, err := tx.CreateBucketIfNotExists(bucketRevoked)
		if err != nil {
			return err
		}

		revocationData := map[string]interface{}{
			"public_key": pubKey,
			"reason":     reason,
			"revoked_at": revokedAt.Format(time.RFC3339),
		}
		data, err := json.Marshal(revocationData)
		if err != nil {
			return err
		}

		if err := bucket.Put([]byte(pubKey), data); err != nil {
			return err
		}

		// Удаляем из активных ключей
		pubKeysBucket := tx.Bucket(bucketPubKeys)
		if pubKeysBucket != nil {
			if err := pubKeysBucket.Delete([]byte(pubKey)); err != nil {
				return err
			}
		}

		return nil
	})
}

// IsKeyRevoked проверяет, отозван ли ключ
func (s *Storage) IsKeyRevoked(pubKey string) (bool, *RevocationInfo, error) {
	var revoked bool
	var info *RevocationInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketRevoked)
		if bucket == nil {
			return nil
		}

		data := bucket.Get([]byte(pubKey))
		if data == nil {
			return nil
		}

		revoked = true
		info = &RevocationInfo{}
		return json.Unmarshal(data, info)
	})

	return revoked, info, err
}

// GetRevocationInfo возвращает информацию об отзыве ключа
func (s *Storage) GetRevocationInfo(pubKey string) (*RevocationInfo, error) {
	var info *RevocationInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketRevoked)
		if bucket == nil {
			return fmt.Errorf("revoked bucket not found")
		}

		data := bucket.Get([]byte(pubKey))
		if data == nil {
			return fmt.Errorf("key not found in revoked list")
		}

		info = &RevocationInfo{}
		return json.Unmarshal(data, info)
	})

	return info, err
}

// GetAllRevokedKeys возвращает все отозванные ключи
func (s *Storage) GetAllRevokedKeys() ([]RevocationInfo, error) {
	var keys []RevocationInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketRevoked)
		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var info RevocationInfo
			if err := json.Unmarshal(v, &info); err != nil {
				return err
			}
			keys = append(keys, info)
		}
		return nil
	})

	return keys, err
}

// SaveDocumentMetadata сохраняет информацию о документе
func (s *Storage) SaveDocumentMetadata(docHash string, filename string, size int64, blockHash [32]byte) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("documents_meta"))
		if err != nil {
			return err
		}

		metadata := map[string]interface{}{
			"filename":   filename,
			"size":       size,
			"block_hash": blockHash[:],
			"uploaded":   time.Now(),
		}

		data, err := json.Marshal(metadata)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(docHash), data)
	})
}

// UpdateKeyActivity обновляет активность ключа (вызывается при подписи блока)
func (s *Storage) UpdateKeyActivity(pubKey string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketActivity)
		if err != nil {
			return err
		}

		// Получаем текущую активность
		var activity KeyActivity
		data := bucket.Get([]byte(pubKey))
		if data != nil {
			json.Unmarshal(data, &activity)
		} else {
			activity = KeyActivity{
				PublicKey: pubKey,
				BlockCount: 0,
			}
		}

		// Обновляем
		activity.LastSeen = time.Now().UTC().Format(time.RFC3339)
		activity.BlockCount++

		// Сохраняем
		activityData, err := json.Marshal(activity)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(pubKey), activityData)
	})
}

// GetActiveKeys возвращает ключи, активные за последние duration
func (s *Storage) GetActiveKeys(duration time.Duration) ([]KeyActivity, error) {
	var activeKeys []KeyActivity
	cutoff := time.Now().UTC().Add(-duration)

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketActivity)
		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var activity KeyActivity
			if err := json.Unmarshal(v, &activity); err != nil {
				continue
			}

			lastSeen, err := time.Parse(time.RFC3339, activity.LastSeen)
			if err != nil {
				continue
			}

			if lastSeen.After(cutoff) {
				activeKeys = append(activeKeys, activity)
			}
		}
		return nil
	})

	return activeKeys, err
}

// GetAllKeyActivity возвращает всю активность ключей
func (s *Storage) GetAllKeyActivity() ([]KeyActivity, error) {
	var activities []KeyActivity

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketActivity)
		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var activity KeyActivity
			if err := json.Unmarshal(v, &activity); err != nil {
				continue
			}
			activities = append(activities, activity)
		}
		return nil
	})

	return activities, err
}

// ==================== Categories ====================

// CreateCategory создаёт новую категорию
func (s *Storage) CreateCategory(id, name, description string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketCategories)
		if err != nil {
			return err
		}

		category := Category{
			ID:          id,
			Name:        name,
			Description: description,
			Created:     time.Now().UTC().Format(time.RFC3339),
			DocCount:    0,
		}

		data, err := json.Marshal(category)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(id), data)
	})
}

// GetCategory возвращает категорию по ID
func (s *Storage) GetCategory(id string) (*Category, error) {
	var category Category

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketCategories)
		if bucket == nil {
			return fmt.Errorf("categories bucket not found")
		}

		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("category not found")
		}

		return json.Unmarshal(data, &category)
	})

	return &category, err
}

// GetAllCategories возвращает все категории
func (s *Storage) GetAllCategories() ([]Category, error) {
	var categories []Category

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketCategories)
		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var category Category
			if err := json.Unmarshal(v, &category); err != nil {
				continue
			}
			categories = append(categories, category)
		}
		return nil
	})

	return categories, err
}

// DeleteCategory удаляет категорию
func (s *Storage) DeleteCategory(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketCategories)
		if bucket == nil {
			return fmt.Errorf("categories bucket not found")
		}
		return bucket.Delete([]byte(id))
	})
}

// IncrementCategoryDocCount увеличивает счётчик документов
func (s *Storage) IncrementCategoryDocCount(categoryID string) error {
	if categoryID == "" {
		return nil
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketCategories)
		if bucket == nil {
			return nil
		}

		data := bucket.Get([]byte(categoryID))
		if data == nil {
			return nil
		}

		var category Category
		if err := json.Unmarshal(data, &category); err != nil {
			return err
		}

		category.DocCount++

		updatedData, err := json.Marshal(category)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(categoryID), updatedData)
	})
}

// SaveDocumentMetadataWithCategory сохраняет метаданные документа с категорией
func (s *Storage) SaveDocumentMetadataWithCategory(meta DocumentMetadata) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("documents_meta"))
		if err != nil {
			return err
		}

		data, err := json.Marshal(meta)
		if err != nil {
			return err
		}

		if err := bucket.Put([]byte(meta.Hash), data); err != nil {
			return err
		}

		// Увеличиваем счётчик категории
		if meta.Category != "" {
			catBucket := tx.Bucket(bucketCategories)
			if catBucket != nil {
				catData := catBucket.Get([]byte(meta.Category))
				if catData != nil {
					var category Category
					if err := json.Unmarshal(catData, &category); err == nil {
						category.DocCount++
						updatedData, _ := json.Marshal(category)
						catBucket.Put([]byte(meta.Category), updatedData)
					}
				}
			}
		}

		return nil
	})
}

// GetDocumentsByCategory возвращает документы категории
func (s *Storage) GetDocumentsByCategory(categoryID string) ([]DocumentMetadata, error) {
	var documents []DocumentMetadata

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("documents_meta"))
		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var meta DocumentMetadata
			if err := json.Unmarshal(v, &meta); err != nil {
				continue
			}
			if meta.Category == categoryID {
				documents = append(documents, meta)
			}
		}
		return nil
	})

	return documents, err
}

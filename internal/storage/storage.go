package storage

import (
	"encoding/binary"
	"errors"
	"log"
	"time"

	"ChainDocs/internal/block"
	"go.etcd.io/bbolt"
)

var (
	// Бакеты
	bucketBlocks = []byte("blocks")    // хэш блока -> блок
	bucketHeight = []byte("height")    // высота -> хэш блока
	bucketDocs   = []byte("documents") // хэш документа -> хэш блока
	bucketMeta   = []byte("metadata")  // метаданные (последний блок и т.д.)

	// Ключи метаданных
	keyLastHash   = []byte("last_hash")
	keyLastHeight = []byte("last_height")
)

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

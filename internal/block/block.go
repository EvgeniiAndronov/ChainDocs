package block

import (
	"ChainDocs/internal/crypto"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Block - основная структура
type Block struct {
	Height       int64     `json:"height"`        // Номер блока
	Timestamp    time.Time `json:"timestamp"`     // Время создания
	PrevHash     [32]byte  `json:"prev_hash"`     // Хэш предыдущего блока
	DocumentHash [32]byte  `json:"document_hash"` // Хэш документа
	Signature    []byte    `json:"signature"`     // Подпись клиента
	Hash         [32]byte  `json:"hash"`          // Хэш этого блока
}

// NewBlock создает новый блок
func NewBlock(height int64, prevHash [32]byte, docHash [32]byte) *Block {
	b := &Block{
		Height:       height,
		Timestamp:    time.Now().UTC(),
		PrevHash:     prevHash,
		DocumentHash: docHash,
	}
	b.Hash = b.CalculateHash()
	return b
}

// CalculateHash вычисляет хэш блока
func (b *Block) CalculateHash() [32]byte {
	// Сериализуем поля для хэширования
	data := make([]byte, 0)

	// Height (8 байт)
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, uint64(b.Height))
	data = append(data, heightBytes...)

	// PrevHash (32 байта)
	data = append(data, b.PrevHash[:]...)

	// DocumentHash (32 байта)
	data = append(data, b.DocumentHash[:]...)

	// Timestamp (через UnixNano - 8 байт)
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(b.Timestamp.UnixNano()))
	data = append(data, timeBytes...)

	return sha256.Sum256(data)
}

// Verify проверяет целостность блока
func (b *Block) Verify() bool {
	// Пересчитываем хэш и сравниваем
	return b.Hash == b.CalculateHash()
}

// MarshalJSON для красивого отображения в API
func (b *Block) MarshalJSON() ([]byte, error) {
	type Alias Block
	return json.Marshal(&struct {
		PrevHash     string `json:"prev_hash"`
		DocumentHash string `json:"document_hash"`
		Hash         string `json:"hash"`
		Signature    string `json:"signature"`
		*Alias
	}{
		PrevHash:     hex.EncodeToString(b.PrevHash[:]),
		DocumentHash: hex.EncodeToString(b.DocumentHash[:]),
		Hash:         hex.EncodeToString(b.Hash[:]),
		Signature:    hex.EncodeToString(b.Signature),
		Alias:        (*Alias)(b),
	})
}

// UnmarshalJSON для парсинга из API
func (b *Block) UnmarshalJSON(data []byte) error {
	type Alias Block
	aux := &struct {
		PrevHash     string `json:"prev_hash"`
		DocumentHash string `json:"document_hash"`
		Hash         string `json:"hash"`
		Signature    string `json:"signature"`
		*Alias
	}{
		Alias: (*Alias)(b),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Конвертируем hex строки обратно в [32]byte
	prevHash, err := hex.DecodeString(aux.PrevHash)
	if err != nil || len(prevHash) != 32 {
		return fmt.Errorf("invalid prev_hash")
	}
	copy(b.PrevHash[:], prevHash)

	docHash, err := hex.DecodeString(aux.DocumentHash)
	if err != nil || len(docHash) != 32 {
		return fmt.Errorf("invalid document_hash")
	}
	copy(b.DocumentHash[:], docHash)

	hash, err := hex.DecodeString(aux.Hash)
	if err != nil || len(hash) != 32 {
		return fmt.Errorf("invalid hash")
	}
	copy(b.Hash[:], hash)

	b.Signature, err = hex.DecodeString(aux.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// ShortHash возвращает первые 8 символов хэша для отображения
func (b *Block) ShortHash() string {
	return hex.EncodeToString(b.Hash[:])[:8]
}

// Подписать блок ключом
func (b *Block) Sign(kp *crypto.KeyPair) {
	// Подписываем хэш блока (не сам блок!)
	b.Signature = kp.Sign(b.Hash[:])
}

// Проверить подпись блока
func (b *Block) VerifySignature(publicKey []byte) bool {
	if len(b.Signature) == 0 {
		return false
	}
	return crypto.Verify(publicKey, b.Hash[:], b.Signature)
}

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

// Signature представляет одну подпись блока
type Signature struct {
	PublicKey string `json:"public_key"` // Публичный ключ подписанта (hex)
	Signature string `json:"signature"`  // Подпись (hex)
	Timestamp string `json:"timestamp"`  // Время подписи (RFC3339)
}

// Block - основная структура
type Block struct {
	Height       int64       `json:"height"`        // Номер блока
	Timestamp    time.Time   `json:"timestamp"`     // Время создания
	PrevHash     [32]byte    `json:"prev_hash"`     // Хэш предыдущего блока
	DocumentHash [32]byte    `json:"document_hash"` // Хэш документа
	Signatures   []Signature `json:"signatures"`    // Массив подписей (консенсус)
	Hash         [32]byte    `json:"hash"`          // Хэш этого блока
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
		PrevHash     string      `json:"prev_hash"`
		DocumentHash string      `json:"document_hash"`
		Hash         string      `json:"hash"`
		Signatures   []Signature `json:"signatures"`
		*Alias
	}{
		PrevHash:     hex.EncodeToString(b.PrevHash[:]),
		DocumentHash: hex.EncodeToString(b.DocumentHash[:]),
		Hash:         hex.EncodeToString(b.Hash[:]),
		Signatures:   b.Signatures,
		Alias:        (*Alias)(b),
	})
}

// UnmarshalJSON для парсинга из API
func (b *Block) UnmarshalJSON(data []byte) error {
	type Alias Block
	aux := &struct {
		PrevHash     string      `json:"prev_hash"`
		DocumentHash string      `json:"document_hash"`
		Hash         string      `json:"hash"`
		Signatures   []Signature `json:"signatures"`
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

	b.Signatures = aux.Signatures

	return nil
}

// ShortHash возвращает первые 8 символов хэша для отображения
func (b *Block) ShortHash() string {
	return hex.EncodeToString(b.Hash[:])[:8]
}

// AddSignature добавляет подпись в блок
func (b *Block) AddSignature(pubKey []byte, signature []byte) {
	b.Signatures = append(b.Signatures, Signature{
		PublicKey: hex.EncodeToString(pubKey),
		Signature: hex.EncodeToString(signature),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// HasSignature проверяет, есть ли уже подпись от данного публичного ключа
func (b *Block) HasSignature(pubKey []byte) bool {
	pubHex := hex.EncodeToString(pubKey)
	for _, sig := range b.Signatures {
		if sig.PublicKey == pubHex {
			return true
		}
	}
	return false
}

// GetSignatureCount возвращает количество подписей
func (b *Block) GetSignatureCount() int {
	return len(b.Signatures)
}

// Подписать блок ключом (добавляет подпись в массив)
func (b *Block) Sign(kp *crypto.KeyPair) {
	// Подписываем хэш блока (не сам блок!)
	signature := kp.Sign(b.Hash[:])
	b.AddSignature(kp.PublicKey, signature)
}

// Проверить все подписи блока
func (b *Block) VerifySignatures() map[string]bool {
	results := make(map[string]bool)
	
	for _, sig := range b.Signatures {
		pubKey, err := crypto.StringToPublicKey(sig.PublicKey)
		if err != nil {
			results[sig.PublicKey] = false
			continue
		}
		
		sigBytes, err := hex.DecodeString(sig.Signature)
		if err != nil {
			results[sig.PublicKey] = false
			continue
		}
		
		results[sig.PublicKey] = crypto.Verify(pubKey, b.Hash[:], sigBytes)
	}
	
	return results
}

// IsSignedBy проверяет, подписан ли блок конкретным ключом
func (b *Block) IsSignedBy(pubKey []byte) bool {
	pubHex := hex.EncodeToString(pubKey)
	for _, sig := range b.Signatures {
		if sig.PublicKey == pubHex {
			// Дополнительно проверяем валидность подписи
			key, err := crypto.StringToPublicKey(sig.PublicKey)
			if err != nil {
				return false
			}
			sigBytes, err := hex.DecodeString(sig.Signature)
			if err != nil {
				return false
			}
			return crypto.Verify(key, b.Hash[:], sigBytes)
		}
	}
	return false
}

// ConsensusReached проверяет, достигнут ли консенсус (51%+ подписей)
// registeredKeys - общее количество зарегистрированных ключей
func (b *Block) ConsensusReached(registeredKeys int) bool {
	if registeredKeys == 0 {
		return false
	}
	
	// Считаем только валидные подписи
	validSignatures := 0
	for _, sig := range b.Signatures {
		pubKey, err := crypto.StringToPublicKey(sig.PublicKey)
		if err != nil {
			continue
		}
		sigBytes, err := hex.DecodeString(sig.Signature)
		if err != nil {
			continue
		}
		if crypto.Verify(pubKey, b.Hash[:], sigBytes) {
			validSignatures++
		}
	}
	
	// 51% и более
	required := (registeredKeys / 2) + 1
	return validSignatures >= required
}

// GetConsensusProgress возвращает прогресс консенсуса
func (b *Block) GetConsensusProgress(registeredKeys int) (signed int, required int, percent float64) {
	validSignatures := 0
	for _, sig := range b.Signatures {
		pubKey, err := crypto.StringToPublicKey(sig.PublicKey)
		if err != nil {
			continue
		}
		sigBytes, err := hex.DecodeString(sig.Signature)
		if err != nil {
			continue
		}
		if crypto.Verify(pubKey, b.Hash[:], sigBytes) {
			validSignatures++
		}
	}
	
	if registeredKeys == 0 {
		return validSignatures, 0, 0
	}
	
	required = (registeredKeys / 2) + 1
	percent = float64(validSignatures) / float64(registeredKeys) * 100
	
	return validSignatures, required, percent
}

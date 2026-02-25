package block

import (
	"ChainDocs/internal/crypto"
	"testing"
)

func TestNewBlock(t *testing.T) {
	// Создаем тестовые данные
	var prevHash [32]byte
	prevHash[0] = 1

	var docHash [32]byte
	docHash[0] = 2

	// Создаем блок
	block := NewBlock(1, prevHash, docHash)

	// Проверяем поля
	if block.Height != 1 {
		t.Errorf("Height = %d, want 1", block.Height)
	}

	if block.PrevHash != prevHash {
		t.Errorf("PrevHash not set correctly")
	}

	if block.DocumentHash != docHash {
		t.Errorf("DocumentHash not set correctly")
	}

	// Проверяем, что хэш посчитался
	if block.Hash == [32]byte{} {
		t.Errorf("Hash not calculated")
	}

	// Проверяем верификацию
	if !block.Verify() {
		t.Errorf("Block should be valid")
	}
}

func TestBlockVerify(t *testing.T) {
	block := NewBlock(1, [32]byte{1}, [32]byte{2})

	// Блок должен быть валидным
	if !block.Verify() {
		t.Error("Fresh block should be valid")
	}

	// Изменяем данные - блок должен стать невалидным
	block.Height = 2
	if block.Verify() {
		t.Error("Block with changed height should be invalid")
	}
}

func TestBlockJSON(t *testing.T) {
	block := NewBlock(1, [32]byte{1, 2, 3}, [32]byte{4, 5, 6})
	
	// Добавляем тестовую подпись
	kp, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	block.Sign(kp)

	// Маршалим
	jsonData, err := block.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// Парсим обратно
	var newBlock Block
	err = newBlock.UnmarshalJSON(jsonData)
	if err != nil {
		t.Fatal(err)
	}

	// Сравниваем
	if block.Height != newBlock.Height {
		t.Error("Height mismatch")
	}

	if block.PrevHash != newBlock.PrevHash {
		t.Error("PrevHash mismatch")
	}

	if block.Hash != newBlock.Hash {
		t.Error("Hash mismatch")
	}

	// Проверяем, что блок все еще валиден
	if !newBlock.Verify() {
		t.Error("Block invalid after JSON roundtrip")
	}
}

func TestBlockMultiSignatures(t *testing.T) {
	block := NewBlock(1, [32]byte{1}, [32]byte{2})
	
	// Генерируем 3 ключа
	key1, _ := crypto.GenerateKey()
	key2, _ := crypto.GenerateKey()
	key3, _ := crypto.GenerateKey()
	
	// Подписываем блок тремя ключами
	block.Sign(key1)
	block.Sign(key2)
	block.Sign(key3)
	
	// Проверяем количество подписей
	if block.GetSignatureCount() != 3 {
		t.Errorf("Expected 3 signatures, got %d", block.GetSignatureCount())
	}
	
	// Проверяем, что каждая подпись валидна
	results := block.VerifySignatures()
	if len(results) != 3 {
		t.Errorf("Expected 3 verification results, got %d", len(results))
	}
	
	for pubKey, valid := range results {
		if !valid {
			t.Errorf("Signature from %s should be valid", pubKey[:16])
		}
	}
	
	// Проверяем IsSignedBy
	if !block.IsSignedBy(key1.PublicKey) {
		t.Error("Block should be signed by key1")
	}
	if !block.IsSignedBy(key2.PublicKey) {
		t.Error("Block should be signed by key2")
	}
	
	// Проверяем HasSignature
	if !block.HasSignature(key1.PublicKey) {
		t.Error("Block should have signature from key1")
	}
	
	// Пытаемся подписать тем же ключом ещё раз (должно добавить дубликат)
	block.Sign(key1)
	if block.GetSignatureCount() != 4 {
		t.Errorf("Expected 4 signatures after duplicate, got %d", block.GetSignatureCount())
	}
}

func TestConsensus(t *testing.T) {
	block := NewBlock(1, [32]byte{1}, [32]byte{2})
	
	// Генерируем 5 ключей (консенсус = 3 из 5)
	var keys []*crypto.KeyPair
	for i := 0; i < 5; i++ {
		kp, _ := crypto.GenerateKey()
		keys = append(keys, kp)
	}
	
	// Без подписей консенсуса нет
	if block.ConsensusReached(5) {
		t.Error("Should not reach consensus with 0 signatures")
	}
	
	// Подписываем 1 ключом (20%)
	block.Sign(keys[0])
	if block.ConsensusReached(5) {
		t.Error("Should not reach consensus with 1 signature (20%)")
	}
	
	// Подписываем 2 ключами (40%)
	block.Sign(keys[1])
	if block.ConsensusReached(5) {
		t.Error("Should not reach consensus with 2 signatures (40%)")
	}
	
	// Подписываем 3 ключами (60% - консенсус!)
	block.Sign(keys[2])
	if !block.ConsensusReached(5) {
		t.Error("Should reach consensus with 3 signatures (60%)")
	}
	
	// Проверяем прогресс
	signed, required, percent := block.GetConsensusProgress(5)
	if signed != 3 {
		t.Errorf("Expected 3 signed, got %d", signed)
	}
	if required != 3 {
		t.Errorf("Expected 3 required, got %d", required)
	}
	if percent < 59 || percent > 61 {
		t.Errorf("Expected ~60%% percent, got %.2f", percent)
	}
}

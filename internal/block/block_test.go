package block

import (
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
	block.Signature = []byte("test-signature")

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

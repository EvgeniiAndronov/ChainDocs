package storage

import (
	"os"
	"testing"

	"ChainDocs/internal/block"
)

func TestStorage(t *testing.T) {
	// Создаем временный файл для тестов
	tmpfile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Открываем хранилище
	store, err := New(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Инициализируем генезис
	err = store.InitGenesis()
	if err != nil {
		t.Fatal(err)
	}

	// Проверяем, что генезис создался
	last, err := store.GetLastBlock()
	if err != nil {
		t.Fatal(err)
	}

	if last.Height != 0 {
		t.Errorf("Genesis height = %d, want 0", last.Height)
	}

	// Создаем новый блок
	docHash := [32]byte{1, 2, 3, 4}
	newBlock := block.NewBlock(1, last.Hash, docHash)

	// Сохраняем
	err = store.SaveBlock(newBlock)
	if err != nil {
		t.Fatal(err)
	}

	// Проверяем, что последний блок обновился
	last, err = store.GetLastBlock()
	if err != nil {
		t.Fatal(err)
	}

	if last.Height != 1 {
		t.Errorf("Last height = %d, want 1", last.Height)
	}

	// Ищем по документу
	found, err := store.GetBlockByDocument(docHash)
	if err != nil {
		t.Fatal(err)
	}

	if found.Hash != newBlock.Hash {
		t.Error("Document lookup failed")
	}

	// Проверяем высоту
	height, err := store.GetHeight()
	if err != nil {
		t.Fatal(err)
	}

	if height != 1 {
		t.Errorf("Height = %d, want 1", height)
	}
}

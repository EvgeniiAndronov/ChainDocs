package storage

import (
	"os"
	"testing"
	"time"

	"ChainDocs/internal/block"
)

func setupTestStorage(t *testing.T) (*Storage, string) {
	tmpfile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatal(err)
	}

	store, err := New(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if err := store.InitGenesis(); err != nil {
		t.Fatal(err)
	}

	return store, tmpfile.Name()
}

func TestStorage_SavePublicKey(t *testing.T) {
	store, path := setupTestStorage(t)
	defer os.Remove(path)
	defer store.Close()

	pubKey := "test_public_key_hex_12345678901234567890123456789012"

	err := store.SavePublicKey(pubKey)
	if err != nil {
		t.Fatalf("Failed to save public key: %v", err)
	}

	// Проверяем, что ключ сохранился
	keys, err := store.GetAllPublicKeys()
	if err != nil {
		t.Fatalf("Failed to get public keys: %v", err)
	}

	if len(keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(keys))
	}

	if keys[0] != pubKey {
		t.Errorf("Expected key %s, got %s", pubKey, keys[0])
	}
}

func TestStorage_GetAllPublicKeys(t *testing.T) {
	store, path := setupTestStorage(t)
	defer os.Remove(path)
	defer store.Close()

	// Пустое хранилище
	keys, err := store.GetAllPublicKeys()
	if err != nil {
		t.Fatalf("Failed to get keys: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys, got %d", len(keys))
	}

	// Добавляем ключи
	key1 := "key1_hex_123456789012345678901234567890123456"
	key2 := "key2_hex_123456789012345678901234567890123456"
	key3 := "key3_hex_123456789012345678901234567890123456"

	store.SavePublicKey(key1)
	store.SavePublicKey(key2)
	store.SavePublicKey(key3)

	keys, err = store.GetAllPublicKeys()
	if err != nil {
		t.Fatalf("Failed to get keys: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}
}

func TestStorage_RevokePublicKey(t *testing.T) {
	store, path := setupTestStorage(t)
	defer os.Remove(path)
	defer store.Close()

	pubKey := "test_revoke_key_12345678901234567890123456789012"
	reason := "compromised"

	// Сохраняем ключ
	err := store.SavePublicKey(pubKey)
	if err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Проверяем, что ключ активен
	keys, _ := store.GetAllPublicKeys()
	if len(keys) != 1 {
		t.Fatal("Key should be saved")
	}

	// Отозываем ключ
	err = store.RevokePublicKey(pubKey, reason, time.Now().UTC())
	if err != nil {
		t.Fatalf("Failed to revoke key: %v", err)
	}

	// Проверяем, что ключ отозван
	isRevoked, info, err := store.IsKeyRevoked(pubKey)
	if err != nil {
		t.Fatalf("Failed to check revocation: %v", err)
	}

	if !isRevoked {
		t.Error("Key should be revoked")
	}

	if info.Reason != reason {
		t.Errorf("Expected reason %s, got %s", reason, info.Reason)
	}

	// Проверяем, что ключ удалён из активных
	keys, _ = store.GetAllPublicKeys()
	if len(keys) != 0 {
		t.Errorf("Expected 0 active keys, got %d", len(keys))
	}
}

func TestStorage_IsKeyRevoked_NotRevoked(t *testing.T) {
	store, path := setupTestStorage(t)
	defer os.Remove(path)
	defer store.Close()

	pubKey := "test_not_revoked_key_12345678901234567890123456"

	// Ключ не сохранён и не отозван
	isRevoked, info, err := store.IsKeyRevoked(pubKey)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if isRevoked {
		t.Error("Key should not be revoked")
	}

	if info != nil {
		t.Error("Info should be nil for non-revoked key")
	}
}

func TestStorage_GetAllRevokedKeys(t *testing.T) {
	store, path := setupTestStorage(t)
	defer os.Remove(path)
	defer store.Close()

	// Пустой список
	keys, err := store.GetAllRevokedKeys()
	if err != nil {
		t.Fatalf("Failed to get revoked keys: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected 0 revoked keys, got %d", len(keys))
	}

	// Добавляем отозванные ключи
	key1 := "revoked_key1_1234567890123456789012345678901"
	key2 := "revoked_key2_1234567890123456789012345678901"

	store.RevokePublicKey(key1, "reason1", time.Now().UTC())
	store.RevokePublicKey(key2, "reason2", time.Now().UTC())

	keys, err = store.GetAllRevokedKeys()
	if err != nil {
		t.Fatalf("Failed to get revoked keys: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 revoked keys, got %d", len(keys))
	}
}

func TestStorage_SaveDocumentMetadata(t *testing.T) {
	store, path := setupTestStorage(t)
	defer os.Remove(path)
	defer store.Close()

	docHash := "test_document_hash_12345678901234567890123456"
	filename := "test.pdf"
	size := int64(1024)

	var blockHash [32]byte
	for i := 0; i < 32; i++ {
		blockHash[i] = byte(i)
	}

	err := store.SaveDocumentMetadata(docHash, filename, size, blockHash)
	if err != nil {
		t.Fatalf("Failed to save document metadata: %v", err)
	}

	// TODO: Добавить метод GetDocumentMetadata для проверки
}

func TestStorage_BlockOperations(t *testing.T) {
	store, path := setupTestStorage(t)
	defer os.Remove(path)
	defer store.Close()

	// Получаем генезис блок
	genesis, err := store.GetLastBlock()
	if err != nil {
		t.Fatal(err)
	}

	if genesis.Height != 0 {
		t.Errorf("Expected genesis height 0, got %d", genesis.Height)
	}

	// Создаём новый блок
	docHash := [32]byte{1, 2, 3, 4, 5}
	newBlock := block.NewBlock(1, genesis.Hash, docHash)

	err = store.SaveBlock(newBlock)
	if err != nil {
		t.Fatalf("Failed to save block: %v", err)
	}

	// Проверяем, что блок сохранился
	last, err := store.GetLastBlock()
	if err != nil {
		t.Fatal(err)
	}

	if last.Height != 1 {
		t.Errorf("Expected height 1, got %d", last.Height)
	}

	// Получаем блок по хэшу
	retrieved, err := store.GetBlock(newBlock.Hash)
	if err != nil {
		t.Fatal(err)
	}

	if retrieved.Height != newBlock.Height {
		t.Error("Retrieved block height mismatch")
	}

	// Получаем блок по высоте
	byHeight, err := store.GetBlockByHeight(1)
	if err != nil {
		t.Fatal(err)
	}

	if byHeight.Hash != newBlock.Hash {
		t.Error("Block by height mismatch")
	}
}

func TestStorage_GetBlockByDocument(t *testing.T) {
	store, path := setupTestStorage(t)
	defer os.Remove(path)
	defer store.Close()

	genesis, _ := store.GetLastBlock()

	docHash := [32]byte{5, 4, 3, 2, 1}
	newBlock := block.NewBlock(1, genesis.Hash, docHash)

	store.SaveBlock(newBlock)

	// Ищем блок по документу
	found, err := store.GetBlockByDocument(docHash)
	if err != nil {
		t.Fatalf("Failed to get block by document: %v", err)
	}

	if found.Hash != newBlock.Hash {
		t.Error("Found block doesn't match")
	}
}

func TestStorage_GetHeight(t *testing.T) {
	store, path := setupTestStorage(t)
	defer os.Remove(path)
	defer store.Close()

	// Генезис имеет высоту 0
	height, err := store.GetHeight()
	if err != nil {
		t.Fatal(err)
	}

	if height != 0 {
		t.Errorf("Expected genesis height 0, got %d", height)
	}

	// Добавляем блок
	genesis, _ := store.GetLastBlock()
	docHash := [32]byte{1, 2, 3}
	newBlock := block.NewBlock(1, genesis.Hash, docHash)
	store.SaveBlock(newBlock)

	// Проверяем высоту
	height, err = store.GetHeight()
	if err != nil {
		t.Fatal(err)
	}

	if height != 1 {
		t.Errorf("Expected height 1, got %d", height)
	}
}

func TestStorage_GetAllBlocks(t *testing.T) {
	store, path := setupTestStorage(t)
	defer os.Remove(path)
	defer store.Close()

	// Только генезис
	blocks, err := store.GetAllBlocks()
	if err != nil {
		t.Fatal(err)
	}

	if len(blocks) != 1 {
		t.Errorf("Expected 1 block (genesis), got %d", len(blocks))
	}

	// Добавляем блоки
	genesis, _ := store.GetLastBlock()
	for i := int64(1); i <= 5; i++ {
		docHash := [32]byte{byte(i)}
		newBlock := block.NewBlock(i, genesis.Hash, docHash)
		store.SaveBlock(newBlock)
		genesis = newBlock
	}

	blocks, err = store.GetAllBlocks()
	if err != nil {
		t.Fatal(err)
	}

	if len(blocks) != 6 {
		t.Errorf("Expected 6 blocks, got %d", len(blocks))
	}
}

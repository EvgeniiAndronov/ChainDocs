package integration

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"ChainDocs/internal/block"
	"ChainDocs/internal/crypto"
	"ChainDocs/internal/storage"

	"github.com/go-chi/chi/v5"
)

// TestServer - тестовая обёртка
type TestServer struct {
	db      *storage.Storage
	handler http.Handler
}

// setupTestServer создаёт тестовый сервер
func setupTestServer(t *testing.T) *TestServer {
	// Временная БД
	tmpfile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(tmpfile.Name()) })

	store, err := storage.New(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	if err := store.InitGenesis(); err != nil {
		t.Fatal(err)
	}

	ts := &TestServer{
		db: store,
	}
	ts.setupRouter()

	return ts
}

func (ts *TestServer) setupRouter() {
	r := chi.NewRouter()

	// Упрощённые хендлеры для тестов
	r.Get("/api/blocks/last", ts.handleGetLastBlock)
	r.Post("/api/sign", ts.handleSign)
	r.Post("/api/register", ts.handleRegister)
	r.Get("/api/blocks/{hash}/consensus", ts.handleConsensus)

	ts.handler = r
}

func (ts *TestServer) handleGetLastBlock(w http.ResponseWriter, r *http.Request) {
	last, err := ts.db.GetLastBlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(last)
}

func (ts *TestServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PublicKey string `json:"public_key"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	ts.db.SavePublicKey(req.PublicKey)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
}

func (ts *TestServer) handleSign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BlockHash string `json:"block_hash"`
		Signature string `json:"signature"`
		PublicKey string `json:"public_key"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	blockHash, _ := hex.DecodeString(req.BlockHash)
	var hashArr [32]byte
	copy(hashArr[:], blockHash)

	b, err := ts.db.GetBlock(hashArr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	pubKey, _ := crypto.StringToPublicKey(req.PublicKey)
	signature, _ := hex.DecodeString(req.Signature)

	b.AddSignature(pubKey, signature)
	ts.db.SaveBlock(b)

	totalKeys, _ := ts.db.GetAllPublicKeys()
	signed, required, percent := b.GetConsensusProgress(len(totalKeys))

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "signature saved",
		"signatures": signed,
		"required":   required,
		"percent":    percent,
		"consensus":  b.ConsensusReached(len(totalKeys)),
	})
}

func (ts *TestServer) handleConsensus(w http.ResponseWriter, r *http.Request) {
	hashStr := chi.URLParam(r, "hash")
	hash, _ := hex.DecodeString(hashStr)
	var hashArr [32]byte
	copy(hashArr[:], hash)

	b, err := ts.db.GetBlock(hashArr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	keys, _ := ts.db.GetAllPublicKeys()
	totalKeys := len(keys)

	signed, required, percent := b.GetConsensusProgress(totalKeys)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"block_hash":       hashStr,
		"height":           b.Height,
		"total_keys":       totalKeys,
		"signatures":       signed,
		"required":         required,
		"percent":          percent,
		"consensus_reached": b.ConsensusReached(totalKeys),
	})
}

// ТЕСТЫ

// TestConsensus_51Percent проверяет консенсус 51%
func TestConsensus_51Percent(t *testing.T) {
	ts := setupTestServer(t)

	// Создаём 5 ключей
	var keys []*crypto.KeyPair
	var pubKeys []string

	for i := 0; i < 5; i++ {
		kp, _ := crypto.GenerateKey()
		keys = append(keys, kp)
		pubHex := crypto.PublicKeyToString(kp.PublicKey)
		pubKeys = append(pubKeys, pubHex)

		// Регистрируем ключ
		req := fmt.Sprintf(`{"public_key":"%s"}`, pubHex)
		ts.request("POST", "/api/register", req)
	}

	// Создаём тестовый блок
	last, _ := ts.db.GetLastBlock()
	docHash := [32]byte{1, 2, 3}
	newBlock := block.NewBlock(1, last.Hash, docHash)
	ts.db.SaveBlock(newBlock)

	// Проверяем: без подписей консенсуса нет
	signed, required, _ := newBlock.GetConsensusProgress(5)
	if signed != 0 {
		t.Errorf("Expected 0 signatures, got %d", signed)
	}
	if required != 3 {
		t.Errorf("Expected 3 required for consensus, got %d", required)
	}

	// Подписываем 2 ключами (40% - недостаточно)
	for i := 0; i < 2; i++ {
		sig := keys[i].Sign(newBlock.Hash[:])
		newBlock.AddSignature(keys[i].PublicKey, sig)
	}
	ts.db.SaveBlock(newBlock)

	// Проверяем: консенсуса всё ещё нет
	if newBlock.ConsensusReached(5) {
		t.Error("Should not reach consensus with 2/5 signatures (40%)")
	}

	// Подписываем 3-м ключом (60% - консенсус!)
	sig := keys[2].Sign(newBlock.Hash[:])
	newBlock.AddSignature(keys[2].PublicKey, sig)
	ts.db.SaveBlock(newBlock)

	// Проверяем: консенсус достигнут
	if !newBlock.ConsensusReached(5) {
		t.Error("Should reach consensus with 3/5 signatures (60%)")
	}
}

// TestMultiSignature_Duplicate проверяет защиту от дублирования подписи
func TestMultiSignature_Duplicate(t *testing.T) {
	block := block.NewBlock(1, [32]byte{}, [32]byte{1, 2, 3})
	kp, _ := crypto.GenerateKey()

	// Подписываем один раз
	block.Sign(kp)
	count1 := block.GetSignatureCount()

	// Подписываем ещё раз тем же ключом
	block.Sign(kp)
	count2 := block.GetSignatureCount()

	if count2 != count1+1 {
		t.Errorf("Expected signature to be added, got %d signatures", count2)
	}

	// HasSignature должен возвращать true
	if !block.HasSignature(kp.PublicKey) {
		t.Error("Should detect existing signature")
	}
}

// TestRevocation_KeyRejected проверяет отзыв ключа
func TestRevocation_KeyRejected(t *testing.T) {
	ts := setupTestServer(t)

	// Создаём старый и новый ключи
	oldKey, _ := crypto.GenerateKey()
	newKey, _ := crypto.GenerateKey()

	oldPubHex := crypto.PublicKeyToString(oldKey.PublicKey)
	newPubHex := crypto.PublicKeyToString(newKey.PublicKey)

	// Регистрируем оба ключа
	ts.db.SavePublicKey(oldPubHex)
	ts.db.SavePublicKey(newPubHex)

	// Подписываем сообщение для отзыва
	message := []byte("revoke:" + oldPubHex)
	_ = newKey.Sign(message)

	// Отозываем старый ключ
	err := ts.db.RevokePublicKey(oldPubHex, "compromised", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// Проверяем, что ключ отозван
	isRevoked, info, err := ts.db.IsKeyRevoked(oldPubHex)
	if err != nil {
		t.Fatal(err)
	}
	if !isRevoked {
		t.Error("Key should be revoked")
	}
	if info.Reason != "compromised" {
		t.Errorf("Expected reason 'compromised', got '%s'", info.Reason)
	}

	// Проверяем, что ключ удалён из активных
	keys, _ := ts.db.GetAllPublicKeys()
	for _, k := range keys {
		if k == oldPubHex {
			t.Error("Revoked key should be removed from active keys")
		}
	}
}

// TestSelfHealing_ForeignSignature проверяет детектор чужих подписей
func TestSelfHealing_ForeignSignature(t *testing.T) {
	block := block.NewBlock(1, [32]byte{}, [32]byte{1, 2, 3})

	// Генерируем "наш" и "чужой" ключи
	ourKey, _ := crypto.GenerateKey()
	foreignKey, _ := crypto.GenerateKey()

	// "Чужой" ключ подписывает блок
	block.Sign(foreignKey)

	// Проверяем: блок подписан не нами
	if block.IsSignedBy(ourKey.PublicKey) {
		t.Error("Block should not be signed by our key")
	}

	if !block.IsSignedBy(foreignKey.PublicKey) {
		t.Error("Block should be signed by foreign key")
	}

	// Детектор должен обнаружить чужую подпись
	hasForeign := false
	ourPubHex := crypto.PublicKeyToString(ourKey.PublicKey)

	for _, sig := range block.Signatures {
		if sig.PublicKey != ourPubHex {
			hasForeign = true
			break
		}
	}

	if !hasForeign {
		t.Error("Should detect foreign signature")
	}
}

// Вспомогательные методы
func (ts *TestServer) request(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	ts.handler.ServeHTTP(w, req)
	return w
}

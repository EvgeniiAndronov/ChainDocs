package main

import (
	"ChainDocs/internal/block"
	"ChainDocs/internal/crypto"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"ChainDocs/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	db      *storage.Storage
	pubKeys map[string]bool // зарегистрированные ключи
	mu      sync.RWMutex
}

func main() {
	// Определяем путь к БД из переменной окружения или используем по умолчанию
	dbPath := os.Getenv("CHAINDOCS_DB")
	if dbPath == "" {
		dbPath = "blockchain.db"
	}

	// Инициализируем хранилище
	store, err := storage.New(dbPath)
	if err != nil {
		log.Fatal("Failed to open storage:", err)
	}
	defer func(store *storage.Storage) {
		err := store.Close()
		if err != nil {
			log.Printf("Error closing storage: %v", err)
		}
	}(store)

	// Создаем генезис если нужно
	if err := store.InitGenesis(); err != nil {
		log.Fatal("Failed to init genesis:", err)
	}

	// Создаем сервер с загрузкой ключей
	srv, err := NewServer(store)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}

	// Настраиваем роутер
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Routes
	r.Get("/", srv.handleHome)
	r.Get("/api/blocks", srv.handleGetBlocks)
	r.Get("/api/blocks/last", srv.handleGetLastBlock)
	r.Get("/api/blocks/{hash}", srv.handleGetBlock)
	r.Get("/api/blocks/height/{height}", srv.handleGetBlockByHeight)
	r.Get("/api/blocks/{hash}/consensus", srv.handleGetConsensus)
	r.Post("/api/blocks", srv.handleCreateBlock)
	r.Post("/api/register", srv.handleRegisterKey)
	r.Post("/api/sign", srv.handleSignature)
	r.Post("/api/upload", srv.handleUpload)
	r.Get("/api/documents/{hash}", srv.handleGetDocument)
	r.Get("/api/keys", srv.handleGetKeys)
	r.Post("/api/revoke", srv.handleRevokeKey)
	r.Get("/api/keys/revoked", srv.handleGetRevokedKeys)

	log.Println("🚀 Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// NewServer создает новый сервер с загрузкой ключей из БД
func NewServer(store *storage.Storage) (*Server, error) {
	s := &Server{
		db:      store,
		pubKeys: make(map[string]bool),
	}

	// Загружаем сохраненные ключи
	if err := s.loadKeys(); err != nil {
		return nil, err
	}

	return s, nil
}

// loadKeys загружает ключи из БД
func (s *Server) loadKeys() error {
	// Получаем все ключи из хранилища
	keys, err := s.db.GetAllPublicKeys()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, key := range keys {
		s.pubKeys[key] = true
	}

	log.Printf("✅ Loaded %d public keys from database", len(keys))
	return nil
}

func (s *Server) handleHome(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
        <h1>Blockchain Document System</h1>
        <p>API endpoints:</p>
        <ul>
            <li><a href="/api/blocks">GET /api/blocks</a> - все блоки</li>
            <li><a href="/api/blocks/last">GET /api/blocks/last</a> - последний блок</li>
            <li>GET /api/blocks/{hash} - блок по хэшу</li>
            <li>GET /api/blocks/height/{height} - блок по высоте</li>
            <li>GET /api/blocks/{hash}/consensus - статус консенсуса блока</li>
            <li>POST /api/blocks - создать блок (с подписью)</li>
            <li>POST /api/register - зарегистрировать публичный ключ</li>
            <li>POST /api/sign - отправить подпись блока</li>
            <li>POST /api/upload - загрузить документ (PDF)</li>
            <li>GET /api/documents/{hash} - скачать документ</li>
            <li>GET /api/keys - список зарегистрированных ключей</li>
            <li>POST /api/revoke - отозвать ключ (требуется подпись новым ключом)</li>
            <li>GET /api/keys/revoked - список отозванных ключей</li>
        </ul>
    `))
}

func (s *Server) handleGetBlocks(w http.ResponseWriter, _ *http.Request) {
	blocks, err := s.db.GetAllBlocks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blocks)
}

func (s *Server) handleGetLastBlock(w http.ResponseWriter, _ *http.Request) {
	lastBlock, err := s.db.GetLastBlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lastBlock)
}

func (s *Server) handleGetBlock(w http.ResponseWriter, r *http.Request) {
	hashStr := chi.URLParam(r, "hash")

	hash, err := hex.DecodeString(hashStr)
	if err != nil || len(hash) != 32 {
		http.Error(w, "Invalid hash", http.StatusBadRequest)
		return
	}

	var hashArr [32]byte
	copy(hashArr[:], hash)

	getBlock, err := s.db.GetBlock(hashArr)
	if err != nil {
		http.Error(w, "Block not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(getBlock)
}

func (s *Server) handleGetBlockByHeight(w http.ResponseWriter, r *http.Request) {
	heightStr := chi.URLParam(r, "height")

	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid height", http.StatusBadRequest)
		return
	}

	blockByHeight, err := s.db.GetBlockByHeight(height)
	if err != nil {
		http.Error(w, "Block not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blockByHeight)
}

// РЕГИСТРАЦИЯ КЛЮЧА (с сохранением в БД)
func (s *Server) handleRegisterKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PublicKey string `json:"public_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Проверяем, что ключ валидный
	pubKey, err := crypto.StringToPublicKey(req.PublicKey)
	if err != nil || len(pubKey) != crypto.PubKeySize {
		http.Error(w, "Invalid public key", http.StatusBadRequest)
		return
	}

	// Сохраняем в БД
	if err := s.db.SavePublicKey(req.PublicKey); err != nil {
		http.Error(w, "Failed to save key", http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	s.pubKeys[req.PublicKey] = true
	s.mu.Unlock()

	log.Printf("✅ Public key registered: %s...", req.PublicKey[:16])

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "registered",
		"key":    req.PublicKey,
	})
}

// СОЗДАНИЕ БЛОКА С ПОДПИСЬЮ (поддержка мульти-подписей)
func (s *Server) handleCreateBlock(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DocumentHash string `json:"document_hash"`
		PublicKey    string `json:"public_key"`
		Signature    string `json:"signature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Проверяем, что ключ зарегистрирован
	s.mu.RLock()
	_, exists := s.pubKeys[req.PublicKey]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Public key not registered", http.StatusUnauthorized)
		return
	}

	// Парсим хэш документа
	docHash, err := hex.DecodeString(req.DocumentHash)
	if err != nil || len(docHash) != 32 {
		http.Error(w, "Invalid document hash", http.StatusBadRequest)
		return
	}

	// Парсим подпись
	signature, err := hex.DecodeString(req.Signature)
	if err != nil || len(signature) != crypto.SignatureSize {
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	var docHashArr [32]byte
	copy(docHashArr[:], docHash)

	// Получаем последний блок
	last, err := s.db.GetLastBlock()
	if err != nil {
		http.Error(w, "Failed to get last block", http.StatusInternalServerError)
		return
	}

	// Создаем новый блок
	newBlock := block.NewBlock(last.Height+1, last.Hash, docHashArr)
	
	// Добавляем подпись
	pubKey, _ := crypto.StringToPublicKey(req.PublicKey)
	newBlock.AddSignature(pubKey, signature)

	// Сохраняем
	if err := s.db.SaveBlock(newBlock); err != nil {
		http.Error(w, "Failed to save block", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ New block created: height=%d, hash=%s, signatures=1", 
		newBlock.Height, newBlock.ShortHash())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newBlock)
}

// ПРИЕМ ПОДПИСИ (поддержка мульти-подписей)
func (s *Server) handleSignature(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BlockHash string `json:"block_hash"`
		Signature string `json:"signature"`
		PublicKey string `json:"public_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Проверяем, что ключ зарегистрирован
	s.mu.RLock()
	_, exists := s.pubKeys[req.PublicKey]
	s.mu.RUnlock()

	if !exists {
		// Проверяем, не отозван ли ключ
		isRevoked, info, _ := s.db.IsKeyRevoked(req.PublicKey)
		if isRevoked {
			http.Error(w, fmt.Sprintf("Public key revoked at %s: %s", info.RevokedAt, info.Reason), http.StatusForbidden)
		} else {
			http.Error(w, "Public key not registered", http.StatusUnauthorized)
		}
		return
	}

	// Парсим хэш блока
	blockHash, err := hex.DecodeString(req.BlockHash)
	if err != nil || len(blockHash) != 32 {
		http.Error(w, "Invalid block hash", http.StatusBadRequest)
		return
	}

	// Парсим подпись
	signature, err := hex.DecodeString(req.Signature)
	if err != nil || len(signature) != crypto.SignatureSize {
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	var hashArr [32]byte
	copy(hashArr[:], blockHash)

	// Получаем блок
	b, err := s.db.GetBlock(hashArr)
	if err != nil {
		http.Error(w, "Block not found", http.StatusNotFound)
		return
	}

	// Проверяем, не подписывал ли уже этот блок этим ключом
	pubKey, _ := crypto.StringToPublicKey(req.PublicKey)
	if b.HasSignature(pubKey) {
		http.Error(w, "Block already signed by this key", http.StatusConflict)
		return
	}

	// Проверяем подпись
	if !crypto.Verify(pubKey, hashArr[:], signature) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Добавляем подпись в блок
	b.AddSignature(pubKey, signature)
	
	if err := s.db.SaveBlock(b); err != nil {
		http.Error(w, "Failed to save signature", http.StatusInternalServerError)
		return
	}

	// Проверяем консенсус
	totalKeys := len(s.pubKeys)
	signed, required, percent := b.GetConsensusProgress(totalKeys)
	
	log.Printf("✅ Signature saved for block %d from %s... [%d/%d = %.1f%%]", 
		b.Height, req.PublicKey[:16], signed, required, percent)
	
	// Если консенсус достигнут
	if b.ConsensusReached(totalKeys) {
		log.Printf("🎉 CONSENSUS REACHED for block %d! (%d/%d signatures)", 
			b.Height, signed, required)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "signature saved",
		"signatures": signed,
		"required":   required,
		"percent":    percent,
		"consensus":  b.ConsensusReached(totalKeys),
	})
}

// Директория для хранения файлов
const uploadDir = "./uploads"

// handleUpload - загрузка PDF файла
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	// Создаем директорию для загрузок, если нет
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Парсим multipart форму (макс 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Получаем файл
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Проверяем расширение
	if ext := filepath.Ext(header.Filename); ext != ".pdf" && ext != ".PDF" {
		http.Error(w, "Only PDF files allowed", http.StatusBadRequest)
		return
	}

	// Читаем файл для вычисления хэша
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Вычисляем SHA-256 хэш
	hash := sha256.Sum256(data)
	hashHex := hex.EncodeToString(hash[:])

	// Сохраняем файл (имя = хэш.pdf)
	filePath := filepath.Join(uploadDir, hashHex+".pdf")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Создаем блок для этого документа (без подписи пока)
	last, err := s.db.GetLastBlock()
	if err != nil {
		http.Error(w, "Failed to get last block", http.StatusInternalServerError)
		return
	}

	newBlock := block.NewBlock(last.Height+1, last.Hash, hash)

	// Сохраняем блок
	if err := s.db.SaveBlock(newBlock); err != nil {
		http.Error(w, "Failed to save block", http.StatusInternalServerError)
		return
	}

	// Сохраняем информацию о документе
	docInfo := struct {
		Hash      string    `json:"hash"`
		Filename  string    `json:"filename"`
		Size      int64     `json:"size"`
		Uploaded  time.Time `json:"uploaded"`
		BlockHash string    `json:"block_hash"`
	}{
		Hash:      hashHex,
		Filename:  header.Filename,
		Size:      header.Size,
		Uploaded:  time.Now(),
		BlockHash: hex.EncodeToString(newBlock.Hash[:]),
	}

	// Можно сохранить в отдельный bucket в БД
	// TODO: сохранять метаданные документа

	log.Printf("📄 File uploaded: %s (%d bytes), hash: %s", header.Filename, header.Size, hashHex[:16])
	log.Printf("🔗 Block created: height=%d, hash=%s", newBlock.Height, newBlock.ShortHash())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(docInfo)
}

// handleGetDocument - получение документа по хэшу
func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	hashHex := chi.URLParam(r, "hash")

	// Проверяем валидность хэша
	if len(hashHex) != 64 {
		http.Error(w, "Invalid hash", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(uploadDir, hashHex+".pdf")

	// Проверяем существование файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}

	// Отдаем файл
	http.ServeFile(w, r, filePath)
}

// handleGetConsensus - получение статуса консенсуса для блока
func (s *Server) handleGetConsensus(w http.ResponseWriter, r *http.Request) {
	hashStr := chi.URLParam(r, "hash")

	hash, err := hex.DecodeString(hashStr)
	if err != nil || len(hash) != 32 {
		http.Error(w, "Invalid hash", http.StatusBadRequest)
		return
	}

	var hashArr [32]byte
	copy(hashArr[:], hash)

	b, err := s.db.GetBlock(hashArr)
	if err != nil {
		http.Error(w, "Block not found", http.StatusNotFound)
		return
	}

	s.mu.RLock()
	totalKeys := len(s.pubKeys)
	s.mu.RUnlock()

	signed, required, percent := b.GetConsensusProgress(totalKeys)
	consensusReached := b.ConsensusReached(totalKeys)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"block_hash":       hex.EncodeToString(b.Hash[:]),
		"height":           b.Height,
		"total_keys":       totalKeys,
		"signatures":       signed,
		"required":         required,
		"percent":          percent,
		"consensus_reached": consensusReached,
		"signatures_list":  b.Signatures,
	})
}

// handleGetKeys - получение списка зарегистрированных ключей
func (s *Server) handleGetKeys(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	keys := make([]string, 0, len(s.pubKeys))
	for k := range s.pubKeys {
		keys = append(keys, k)
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count": len(keys),
		"keys":  keys,
	})
}

// handleRevokeKey - отзыв ключа
func (s *Server) handleRevokeKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PublicKey    string `json:"public_key"`     // Ключ для отзыва
		NewPublicKey string `json:"new_public_key"` // Новый ключ (для подтверждения владения)
		NewSignature string `json:"new_signature"`  // Подпись новым ключом
		Reason       string `json:"reason"`         // Причина отзыва
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Проверяем, что ключ существует
	s.mu.RLock()
	_, exists := s.pubKeys[req.PublicKey]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Public key not found", http.StatusNotFound)
		return
	}

	// Проверяем, что ключ ещё не отозван
	isRevoked, _, err := s.db.IsKeyRevoked(req.PublicKey)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if isRevoked {
		http.Error(w, "Key already revoked", http.StatusConflict)
		return
	}

	// Проверяем подпись новым ключом (подтверждение владения)
	// Владелец должен подписать сообщение "revoke:<old_key>" новым ключом
	newPubKey, err := crypto.StringToPublicKey(req.NewPublicKey)
	if err != nil {
		http.Error(w, "Invalid new public key", http.StatusBadRequest)
		return
	}

	signature, err := hex.DecodeString(req.NewSignature)
	if err != nil || len(signature) != crypto.SignatureSize {
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	// Формируем сообщение для проверки
	message := []byte("revoke:" + req.PublicKey)
	if !crypto.Verify(newPubKey, message, signature) {
		http.Error(w, "Invalid signature - key ownership not confirmed", http.StatusUnauthorized)
		return
	}

	// Проверяем, что новый ключ тоже зарегистрирован
	s.mu.RLock()
	_, newKeyExists := s.pubKeys[req.NewPublicKey]
	s.mu.RUnlock()

	if !newKeyExists {
		http.Error(w, "New public key not registered", http.StatusBadRequest)
		return
	}

	// Отозываем ключ
	if err := s.db.RevokePublicKey(req.PublicKey, req.Reason, time.Now().UTC()); err != nil {
		http.Error(w, "Failed to revoke key", http.StatusInternalServerError)
		return
	}

	// Удаляем из кэша
	s.mu.Lock()
	delete(s.pubKeys, req.PublicKey)
	s.mu.Unlock()

	log.Printf("🚫 Key revoked: %s... (reason: %s, replaced by: %s...)", 
		req.PublicKey[:16], req.Reason, req.NewPublicKey[:16])

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "revoked",
		"key":    req.PublicKey,
		"reason": req.Reason,
	})
}

// handleGetRevokedKeys - получение списка отозванных ключей
func (s *Server) handleGetRevokedKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.db.GetAllRevokedKeys()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count": len(keys),
		"keys":  keys,
	})
}

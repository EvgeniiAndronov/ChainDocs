package main

import (
	"ChainDocs/internal/block"
	"ChainDocs/internal/crypto"
	"ChainDocs/pkg/logger"
	"ChainDocs/pkg/metrics"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"ChainDocs/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	db      *storage.Storage
	pubKeys map[string]bool // зарегистрированные ключи
	mu      sync.RWMutex
	logger  *logger.Logger
	config  *ServerConfig
	authToken string // токен для веб-интерфейса
}

func main() {
	// Загружаем конфигурацию
	config := loadConfig()

	// Инициализируем логгер
	logConfig := logger.Config{
		Level:      "info",
		File:       config.LogFile,
		Format:     "text",
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}
	
	if err := logger.Init(logConfig); err != nil {
		logger.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Close()

	logger.Info("🚀 ChainDocs Server starting...")
	logger.Info("📄 Config loaded: port=%d, db=%s", config.Port, config.DBPath)

	// Инициализация метрик
	metrics.Init()

	// Определяем путь к БД
	dbPath := config.DBPath
	if dbPath == "" {
		dbPath = "blockchain.db"
	}

	// Инициализируем хранилище
	store, err := storage.New(dbPath)
	if err != nil {
		logger.Error("❌ Failed to open storage: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			logger.Error("Error closing storage: %v", err)
		}
	}()

	// Создаем генезис если нужно
	if err := store.InitGenesis(); err != nil {
		logger.Error("❌ Failed to init genesis: %v", err)
		os.Exit(1)
	}

	// Создаем сервер с загрузкой ключей
	srv, err := NewServer(store, config)
	if err != nil {
		logger.Error("❌ Failed to create server: %v", err)
		os.Exit(1)
	}

	// Настраиваем роутер
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	
	// Metrics middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			duration := time.Since(start).Seconds()
			if metrics.DefaultMetrics != nil {
				metrics.DefaultMetrics.ObserveRequest(duration)
			}
		})
	})
	
	// Auth middleware для веб-интерфейса
	r.Use(srv.authMiddleware)
	
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Web UI routes
	r.Get("/web/", srv.handleWebDashboard)
	r.Get("/web/login", srv.handleWebLogin)
	r.Post("/web/login", srv.handleWebLoginSubmit)
	r.Get("/web/blocks", srv.handleWebBlocks)
	r.Get("/web/blocks/{hash}", srv.handleWebBlock)
	r.Get("/web/upload", srv.handleWebUpload)
	r.Get("/web/keys", srv.handleWebKeys)

	// API routes
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
	r.Post("/api/upload/bulk", srv.handleBulkUpload)
	r.Get("/api/documents/{hash}", srv.handleGetDocument)
	r.Get("/api/keys", srv.handleGetKeys)
	r.Post("/api/revoke", srv.handleRevokeKey)
	r.Get("/api/keys/revoked", srv.handleGetRevokedKeys)
	r.Get("/api/keys/active", srv.handleGetActiveKeys)
	
	// Categories
	r.Get("/api/categories", srv.handleGetCategories)
	r.Post("/api/categories", srv.handleCreateCategory)
	r.Get("/api/categories/{id}", srv.handleGetCategory)
	r.Get("/api/categories/{id}/documents", srv.handleGetCategoryDocuments)
	r.Delete("/api/categories/{id}", srv.handleDeleteCategory)
	
	// Metrics endpoint
	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	logger.Info("🚀 Server starting on :%d", config.Port)
	logger.Fatal("%v", http.ListenAndServe(fmt.Sprintf(":%d", config.Port), r))
}

// NewServer создает новый сервер с загрузкой ключей из БД
func NewServer(store *storage.Storage, config *ServerConfig) (*Server, error) {
	s := &Server{
		db:      store,
		pubKeys: make(map[string]bool),
		logger:  logger.DefaultLogger,
		config:  config,
		authToken: os.Getenv("CHAINDOCS_AUTH_TOKEN"), // из переменной окружения
	}

	// Если токен не задан, генерируем случайный
	if s.authToken == "" {
		s.authToken = generateRandomToken()
		logger.Warn("⚠️  No auth token set. Using generated token: %s", s.authToken)
		logger.Warn("⚠️  Set CHAINDOCS_AUTH_TOKEN environment variable for production")
	}

	// Загружаем сохраненные ключи
	if err := s.loadKeys(); err != nil {
		return nil, err
	}

	logger.Info("✅ Loaded %d public keys from database", len(s.pubKeys))

	return s, nil
}

// generateRandomToken генерирует случайный токен
func generateRandomToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "default-insecure-token"
	}
	return hex.EncodeToString(bytes)
}

// authMiddleware проверяет аутентификацию для веб-интерфейса
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API endpoints не требуют аутентификации (для интеграции)
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/metrics") {
			next.ServeHTTP(w, r)
			return
		}

		// Статика не требует аутентификации
		if strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		// Веб-интерфейс требует аутентификации
		if strings.HasPrefix(r.URL.Path, "/web/") {
			// Проверяем токен из разных источников
			token := r.URL.Query().Get("token")
			
			// Проверяем cookie
			if token == "" {
				if cookie, err := r.Cookie("auth_token"); err == nil {
					token = cookie.Value
				}
			}
			
			authHeader := r.Header.Get("Authorization")

			// Проверка токена
			validToken := false
			if token != "" && token == s.authToken {
				validToken = true
				// Устанавливаем cookie если токена не было
				if r.URL.Query().Get("token") != "" {
					http.SetCookie(w, &http.Cookie{
						Name:     "auth_token",
						Value:    token,
						Path:     "/web/",
						MaxAge:   86400, // 24 часа
						HttpOnly: true,
						SameSite: http.SameSiteLaxMode,
					})
				}
			}
			if strings.HasPrefix(authHeader, "Bearer ") && authHeader[7:] == s.authToken {
				validToken = true
			}

			if !validToken {
				// Перенаправляем на страницу входа
				if r.URL.Path != "/web/login" {
					http.Redirect(w, r, "/web/login", http.StatusTemporaryRedirect)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
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

	logger.Info("✅ Loaded %d public keys from database", len(keys))
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

	logger.Info("✅ Public key registered: %s...", req.PublicKey[:16])

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

	logger.Info("✅ New block created: height=%d, hash=%s, signatures=1", 
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

	// Обновляем активность ключа
	s.db.UpdateKeyActivity(req.PublicKey)

	// Проверяем консенсус (динамический расчёт)
	activeKeys, _ := s.db.GetActiveKeys(24 * time.Hour)
	totalKeys := len(activeKeys)
	if totalKeys == 0 {
		s.mu.RLock()
		totalKeys = len(s.pubKeys)
		s.mu.RUnlock()
	}
	
	signed, required, percent := b.GetConsensusProgress(totalKeys)
	
	// Минимальный порог
	if required < 2 {
		required = 2
	}
	
	logger.Info("✅ Signature saved for block %d from %s... [%d/%d = %.1f%%]", 
		b.Height, req.PublicKey[:16], signed, required, percent)
	
	// Если консенсус достигнут
	if b.ConsensusReached(totalKeys) && signed >= required {
		logger.Info("🎉 CONSENSUS REACHED for block %d! (%d/%d signatures)", 
			b.Height, signed, required)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "signature saved",
		"signatures":  signed,
		"required":    required,
		"percent":     percent,
		"consensus":   b.ConsensusReached(totalKeys) && signed >= required,
		"active_keys": len(activeKeys),
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

	// Получаем подпись документа (опционально)
	docSignature := r.FormValue("document_signature")
	publicKey := r.FormValue("public_key")

	// Если подпись предоставлена - проверяем её
	if docSignature != "" && publicKey != "" {
		pubKey, err := crypto.StringToPublicKey(publicKey)
		if err != nil {
			http.Error(w, "Invalid public key", http.StatusBadRequest)
			return
		}

		sigBytes, err := hex.DecodeString(docSignature)
		if err != nil || len(sigBytes) != crypto.SignatureSize {
			http.Error(w, "Invalid signature format", http.StatusBadRequest)
			return
		}

		// Проверяем подпись хэша документа
		if !crypto.Verify(pubKey, hash[:], sigBytes) {
			http.Error(w, "Invalid document signature", http.StatusUnauthorized)
			return
		}

		logger.Info("✅ Document signature verified: %s...", publicKey[:16])
	}

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

	// Добавляем подпись документа если предоставлена
	if docSignature != "" && publicKey != "" {
		newBlock.DocumentSignature = &block.DocumentSignature{
			PublicKey: publicKey,
			Signature: docSignature,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}

	// Сохраняем блок
	if err := s.db.SaveBlock(newBlock); err != nil {
		http.Error(w, "Failed to save block", http.StatusInternalServerError)
		return
	}

	// Сохраняем информацию о документе
	docInfo := struct {
		Hash              string                 `json:"hash"`
		Filename          string                 `json:"filename"`
		Size              int64                  `json:"size"`
		Uploaded          time.Time              `json:"uploaded"`
		BlockHash         string                 `json:"block_hash"`
		DocumentSignature *block.DocumentSignature `json:"document_signature,omitempty"`
	}{
		Hash:              hashHex,
		Filename:          header.Filename,
		Size:              header.Size,
		Uploaded:          time.Now(),
		BlockHash:         hex.EncodeToString(newBlock.Hash[:]),
		DocumentSignature: newBlock.DocumentSignature,
	}

	// Можно сохранить в отдельный bucket в БД
	// TODO: сохранять метаданные документа

	logger.Info("📄 File uploaded: %s (%d bytes), hash: %s", header.Filename, header.Size, hashHex[:16])
	logger.Info("🔗 Block created: height=%d, hash=%s", newBlock.Height, newBlock.ShortHash())

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

// handleBulkUpload - массовая загрузка файлов
func (s *Server) handleBulkUpload(w http.ResponseWriter, r *http.Request) {
	// Создаем директорию для загрузок, если нет
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Парсим multipart форму (макс 100MB для bulk)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Получаем категорию (опционально)
	category := r.FormValue("category")
	
	// Получаем все файлы
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Error(w, "No files provided", http.StatusBadRequest)
		return
	}

	// Результат загрузки
	type UploadResult struct {
		Filename  string `json:"filename"`
		Hash      string `json:"hash"`
		BlockHash string `json:"block_hash"`
		Size      int64  `json:"size"`
		Success   bool   `json:"success"`
		Error     string `json:"error,omitempty"`
	}

	results := make([]UploadResult, 0, len(files))
	successCount := 0

	for _, fileHeader := range files {
		result := UploadResult{
			Filename: fileHeader.Filename,
			Size:     fileHeader.Size,
			Success:  false,
		}

		// Проверяем расширение
		if ext := filepath.Ext(fileHeader.Filename); ext != ".pdf" && ext != ".PDF" {
			result.Error = "Only PDF files allowed"
			results = append(results, result)
			continue
		}

		// Открываем файл
		file, err := fileHeader.Open()
		if err != nil {
			result.Error = "Failed to open file"
			results = append(results, result)
			continue
		}

		// Читаем файл
		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			result.Error = "Failed to read file"
			results = append(results, result)
			continue
		}

		// Вычисляем хэш
		hash := sha256.Sum256(data)
		hashHex := hex.EncodeToString(hash[:])
		result.Hash = hashHex

		// Сохраняем файл
		filePath := filepath.Join(uploadDir, hashHex+".pdf")
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			result.Error = "Failed to save file"
			results = append(results, result)
			continue
		}

		// Создаем блок
		last, err := s.db.GetLastBlock()
		if err != nil {
			result.Error = "Failed to get last block"
			results = append(results, result)
			continue
		}

		newBlock := block.NewBlock(last.Height+1, last.Hash, hash)

		// Сохраняем блок
		if err := s.db.SaveBlock(newBlock); err != nil {
			result.Error = "Failed to save block"
			results = append(results, result)
			continue
		}

		result.BlockHash = hex.EncodeToString(newBlock.Hash[:])
		result.Success = true
		successCount++

		// Сохраняем метаданные с категорией
		meta := storage.DocumentMetadata{
			Hash:      hashHex,
			Filename:  fileHeader.Filename,
			Category:  category,
			Size:      fileHeader.Size,
			Uploaded:  time.Now().UTC().Format(time.RFC3339),
			BlockHash: result.BlockHash,
		}
		s.db.SaveDocumentMetadataWithCategory(meta)

		if category != "" {
			s.db.IncrementCategoryDocCount(category)
		}

		results = append(results, result)
		logger.Info("📄 Bulk uploaded: %s (%d bytes), hash: %s, category: %s", 
			fileHeader.Filename, fileHeader.Size, hashHex[:16], category)
	}

	logger.Info("📦 Bulk upload completed: %d/%d files successful", successCount, len(files))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total":    len(files),
		"success":  successCount,
		"failed":   len(files) - successCount,
		"results":  results,
		"category": category,
	})
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

	// Динамический расчёт: используем активные ключи за 24 часа
	activeKeys, _ := s.db.GetActiveKeys(24 * time.Hour)
	totalKeys := len(activeKeys)
	
	// Если нет активных, используем все зарегистрированные
	if totalKeys == 0 {
		s.mu.RLock()
		totalKeys = len(s.pubKeys)
		s.mu.RUnlock()
	}

	signed, required, percent := b.GetConsensusProgress(totalKeys)
	consensusReached := b.ConsensusReached(totalKeys)
	
	// Минимальный порог - 2 подписи
	if required < 2 {
		required = 2
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"block_hash":        hex.EncodeToString(b.Hash[:]),
		"height":            b.Height,
		"total_keys":        totalKeys,
		"active_keys":       len(activeKeys),
		"signatures":        signed,
		"required":          required,
		"percent":           percent,
		"consensus_reached": consensusReached,
		"signatures_list":   b.Signatures,
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

	logger.Info("🚫 Key revoked: %s... (reason: %s, replaced by: %s...)", 
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

// handleGetActiveKeys - получение списка активных ключей
func (s *Server) handleGetActiveKeys(w http.ResponseWriter, r *http.Request) {
	// По умолчанию 24 часа
	window := 24 * time.Hour
	
	// Можно переопределить через query параметр
	if windowStr := r.URL.Query().Get("window"); windowStr != "" {
		if d, err := time.ParseDuration(windowStr); err == nil {
			window = d
		}
	}
	
	activities, err := s.db.GetActiveKeys(window)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"window":     window.String(),
		"count":      len(activities),
		"activities": activities,
	})
}

// ==================== Categories Handlers ====================

// handleGetCategories - получение всех категорий
func (s *Server) handleGetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := s.db.GetAllCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":      len(categories),
		"categories": categories,
	})
}

// handleCreateCategory - создание категории
func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.Name == "" {
		http.Error(w, "ID and name are required", http.StatusBadRequest)
		return
	}

	if err := s.db.CreateCategory(req.ID, req.Name, req.Description); err != nil {
		http.Error(w, "Failed to create category", http.StatusInternalServerError)
		return
	}

	logger.Info("📁 Category created: %s (%s)", req.Name, req.ID)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "created",
		"id":     req.ID,
	})
}

// handleGetCategory - получение категории по ID
func (s *Server) handleGetCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	category, err := s.db.GetCategory(id)
	if err != nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(category)
}

// handleGetCategoryDocuments - получение документов категории
func (s *Server) handleGetCategoryDocuments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	documents, err := s.db.GetDocumentsByCategory(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"category":  id,
		"count":     len(documents),
		"documents": documents,
	})
}

// handleDeleteCategory - удаление категории
func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.db.DeleteCategory(id); err != nil {
		http.Error(w, "Failed to delete category", http.StatusInternalServerError)
		return
	}

	logger.Info("🗑️ Category deleted: %s", id)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "deleted",
		"id":     id,
	})
}

// ========================================
// Web UI Handlers
// ========================================

// handleWebLogin — страница входа
func (s *Server) handleWebLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="ru" data-bs-theme="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Вход - ChainDocs</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        body { min-height: 100vh; display: flex; align-items: center; justify-content: center; background: linear-gradient(180deg, #1a1d20 0%, #0d0f11 100%); }
        .login-card { width: 100%; max-width: 400px; padding: 2rem; border-radius: 1rem; background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.1); }
    </style>
</head>
<body>
    <div class="card login-card">
        <div class="card-body">
            <div class="text-center mb-4">
                <h1 class="h3"><i class="bi bi-shield-lock"></i> ChainDocs</h1>
                <p class="text-muted">Вход в систему</p>
            </div>
            <form method="POST" action="/web/login">
                <div class="mb-3">
                    <label for="token" class="form-label">Токен доступа</label>
                    <input type="password" class="form-control" id="token" name="token" required 
                           placeholder="Введите токен из CHAINDOCS_AUTH_TOKEN">
                    <div class="form-text">
                        Токен задается через переменную окружения<br>
                        <code>CHAINDOCS_AUTH_TOKEN</code>
                    </div>
                </div>
                <div id="error" class="alert alert-danger d-none"></div>
                <button type="submit" class="btn btn-primary w-100">
                    <i class="bi bi-box-arrow-in-right"></i> Войти
                </button>
            </form>
        </div>
    </div>
    <script>
        const urlParams = new URLSearchParams(window.location.search);
        if (urlParams.get('error') === 'invalid') {
            document.getElementById('error').textContent = 'Неверный токен';
            document.getElementById('error').classList.remove('d-none');
        }
    </script>
</body>
</html>`))
}

// handleWebLoginSubmit — обработка входа
func (s *Server) handleWebLoginSubmit(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	
	if token == s.authToken {
		// Устанавливаем cookie на 24 часа
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    token,
			Path:     "/web/",
			MaxAge:   86400, // 24 часа
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		
		// Перенаправляем на главную
		http.Redirect(w, r, "/web/", http.StatusSeeOther)
	} else {
		// Неверный токен
		http.Redirect(w, r, "/web/login?error=invalid", http.StatusSeeOther)
	}
}

type PageData struct {
	Title string
	Page  string
	Hash  string
	Token string
}

// handleWebDashboard - главная страница
func (s *Server) handleWebDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/dashboard.html")
}

// handleWebBlocks - список блоков
func (s *Server) handleWebBlocks(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/blocks.html")
}

// handleWebBlock - детали блока
func (s *Server) handleWebBlock(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/block.html")
}

// handleWebUpload - загрузка документа
func (s *Server) handleWebUpload(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/upload.html")
}

// handleWebKeys - управление ключами
func (s *Server) handleWebKeys(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/keys.html")
}


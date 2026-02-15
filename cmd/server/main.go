package main

import (
	"ChainDocs/internal/block"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"ChainDocs/internal/storage"
)

type Server struct {
	db *storage.Storage
}

func main() {
	// Инициализируем хранилище
	store, err := storage.New("blockchain.db")
	if err != nil {
		log.Fatal("Failed to open storage:", err)
	}
	defer store.Close()

	// Создаем генезис если нужно
	if err := store.InitGenesis(); err != nil {
		log.Fatal("Failed to init genesis:", err)
	}

	srv := &Server{db: store}

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
	r.Post("/api/blocks", srv.handleCreateBlock)

	log.Println("🚀 Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
        <h1>Blockchain Document System</h1>
        <p>API endpoints:</p>
        <ul>
            <li><a href="/api/blocks">GET /api/blocks</a> - все блоки</li>
            <li><a href="/api/blocks/last">GET /api/blocks/last</a> - последний блок</li>
            <li>GET /api/blocks/{hash} - блок по хэшу</li>
            <li>GET /api/blocks/height/{height} - блок по высоте</li>
            <li>POST /api/blocks - создать блок</li>
        </ul>
    `))
}

func (s *Server) handleGetBlocks(w http.ResponseWriter, r *http.Request) {
	blocks, err := s.db.GetAllBlocks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blocks)
}

func (s *Server) handleGetLastBlock(w http.ResponseWriter, r *http.Request) {
	block, err := s.db.GetLastBlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(block)
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

	block, err := s.db.GetBlock(hashArr)
	if err != nil {
		http.Error(w, "Block not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(block)
}

func (s *Server) handleGetBlockByHeight(w http.ResponseWriter, r *http.Request) {
	heightStr := chi.URLParam(r, "height")

	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid height", http.StatusBadRequest)
		return
	}

	block, err := s.db.GetBlockByHeight(height)
	if err != nil {
		http.Error(w, "Block not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(block)
}

func (s *Server) handleCreateBlock(w http.ResponseWriter, r *http.Request) {
	// Временная заглушка для создания блока
	var req struct {
		DocumentHash string `json:"document_hash"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Парсим хэш документа
	docHash, err := hex.DecodeString(req.DocumentHash)
	if err != nil || len(docHash) != 32 {
		http.Error(w, "Invalid document hash", http.StatusBadRequest)
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

	// TODO: Добавить подпись

	// Сохраняем
	if err := s.db.SaveBlock(newBlock); err != nil {
		http.Error(w, "Failed to save block", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newBlock)
}

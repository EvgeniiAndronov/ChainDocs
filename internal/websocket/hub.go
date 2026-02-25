package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"ChainDocs/internal/block"
	"nhooyr.io/websocket"
)

// MessageType тип сообщения
type MessageType string

const (
	MsgBlockAnnounce   MessageType = "block_announce"
	MsgConsensusUpdate MessageType = "consensus_update"
	MsgPeerUpdate      MessageType = "peer_update"
	MsgError           MessageType = "error"
)

// Message структура сообщения
type Message struct {
	Type      MessageType      `json:"type"`
	Block     *block.Block     `json:"block,omitempty"`
	BlockHash string           `json:"block_hash,omitempty"`
	Consensus *ConsensusStatus `json:"consensus,omitempty"`
	Peers     []PeerInfo       `json:"peers,omitempty"`
	Error     string           `json:"error,omitempty"`
	Timestamp string           `json:"timestamp"`
}

// ConsensusStatus статус консенсуса
type ConsensusStatus struct {
	Signatures      int     `json:"signatures"`
	Required        int     `json:"required"`
	Percent         float64 `json:"percent"`
	ConsensusReached bool  `json:"consensus_reached"`
}

// PeerInfo информация о пире
type PeerInfo struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	Connected bool   `json:"connected"`
}

// ClientInfo информация о WebSocket клиенте
type ClientInfo struct {
	ID        string
	Addr      string
	PublicKey string
	Conn      *websocket.Conn
}

// Hub управляет WebSocket подключениями
type Hub struct {
	mu       sync.RWMutex
	clients  map[string]*ClientInfo
	broadcast chan Message
	serverID  string
}

// NewHub создаёт новый Hub
func NewHub(serverID string) *Hub {
	return &Hub{
		clients:   make(map[string]*ClientInfo),
		broadcast: make(chan Message, 256),
		serverID:  serverID,
	}
}

// Start запускает Hub
func (h *Hub) Start() {
	log.Println("🌐 WebSocket Hub started")
	go h.run()
	go h.periodicPendingCheck()
}

func (h *Hub) run() {
	log.Println("🌐 WebSocket Hub run() started, waiting for messages...")
	for msg := range h.broadcast {
		log.Printf("📨 Received message in run(): type=%s", msg.Type)
		h.broadcastMessage(msg)
	}
}

// periodicPendingCheck - периодическая проверка неподписанных блоков (каждые 2 минуты)
func (h *Hub) periodicPendingCheck() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		// Проверяем pending блоки через API сервера
		// Это упрощённая версия - в production лучше сделать callback
		log.Println("⏰ Periodic pending blocks check")
	}
}

func (h *Hub) broadcastMessage(msg Message) {
	log.Printf("🔵 [DEBUG 1/4] broadcastMessage called with type=%s", msg.Type)
	
	// Получаем список клиентов
	h.mu.Lock()
	clients := make([]*ClientInfo, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	clientCount := len(clients)
	h.mu.Unlock()
	
	log.Printf("🔵 [DEBUG 2/4] Broadcasting to %d clients", clientCount)
	
	for _, client := range clients {
		log.Printf("📩 Sending to client %s", client.ID)
		if err := h.sendMessage(client.Conn, msg); err != nil {
			log.Printf("⚠️  Failed to send to client %s: %v", client.ID, err)
		} else {
			log.Printf("✅ Sent to client %s", client.ID)
		}
	}
	
	log.Printf("🔵 [DEBUG 4/4] Broadcast complete")
}

func (h *Hub) sendMessage(conn *websocket.Conn, msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return conn.Write(ctx, websocket.MessageText, data)
}

// AddClient добавляет клиента
func (h *Hub) AddClient(id string, conn *websocket.Conn, addr string, publicKey string) {
	h.mu.Lock()
	h.clients[id] = &ClientInfo{
		ID:        id,
		Addr:      addr,
		PublicKey: publicKey,
		Conn:      conn,
	}
	h.mu.Unlock()

	log.Printf("✅ WebSocket client connected: %s (%s)", id, addr)

	// Отправляем список пиров новому клиенту (без блокировки)
	h.sendPeerList(id)
}

// RemoveClient удаляет клиента
func (h *Hub) RemoveClient(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client, ok := h.clients[id]; ok {
		client.Conn.Close(websocket.StatusNormalClosure, "removed")
		delete(h.clients, id)
		log.Printf("🛑 WebSocket client disconnected: %s", id)
	}
}

// GetClientCount возвращает количество клиентов
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetPeers возвращает список пиров
func (h *Hub) GetPeers() []PeerInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	peers := make([]PeerInfo, 0, len(h.clients))
	for _, client := range h.clients {
		peers = append(peers, PeerInfo{
			ID:        client.ID,
			Address:   client.Addr,
			Connected: true,
		})
	}
	return peers
}

// BroadcastBlock рассылает уведомление о новом блоке
func (h *Hub) BroadcastBlock(b *block.Block) {
	msg := Message{
		Type:      MsgBlockAnnounce,
		Block:     b,
		BlockHash: b.ShortHash(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	h.broadcast <- msg
	log.Printf("📢 Broadcasted block: height=%d, hash=%s", b.Height, b.ShortHash())
}

// BroadcastConsensus рассылает обновление консенсуса
func (h *Hub) BroadcastConsensus(blockHash string, signatures, required int, percent float64, reached bool) {
	msg := Message{
		Type:      MsgConsensusUpdate,
		BlockHash: blockHash,
		Consensus: &ConsensusStatus{
			Signatures:       signatures,
			Required:         required,
			Percent:          percent,
			ConsensusReached: reached,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	h.broadcast <- msg
}

// sendPeerList отправляет список пиров конкретному клиенту
func (h *Hub) sendPeerList(clientID string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, ok := h.clients[clientID]
	if !ok {
		return
	}

	peers := make([]PeerInfo, 0, len(h.clients)-1)
	for id, c := range h.clients {
		if id != clientID {
			peers = append(peers, PeerInfo{
				ID:        c.ID,
				Address:   c.Addr,
				Connected: true,
			})
		}
	}

	msg := Message{
		Type:      MsgPeerUpdate,
		Peers:     peers,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	h.sendMessage(client.Conn, msg)
}

// Handler создаёт HTTP handler для WebSocket
func (h *Hub) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🔌 WebSocket connection attempt from %s", r.RemoteAddr)
		
		// Принимаем WebSocket подключение
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true, // Для локальной разработки
		})
		if err != nil {
			log.Printf("⚠️  WebSocket accept error: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("✅ WebSocket accepted from %s", r.RemoteAddr)

		// Получаем адрес клиента
		addr := r.RemoteAddr
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			addr = xff
		}

		// Пытаемся получить публичный ключ из query params
		publicKey := r.URL.Query().Get("public_key")
		log.Printf("🔑 Public key from query: %s", publicKey[:50])
		
		clientID := publicKey
		if clientID == "" {
			// Генерируем временный ID
			clientID = addr
		}

		log.Printf("🆔 Client ID: %s", clientID[:50])

		// Добавляем клиента
		h.AddClient(clientID, conn, addr, publicKey)

		log.Printf("✅ Client added to hub: %s", clientID)

		// Запускаем обработчик входящих сообщений
		go h.handleClientMessages(clientID, conn)
	}
}

func (h *Hub) handleClientMessages(clientID string, conn *websocket.Conn) {
	defer func() {
		conn.Close(websocket.StatusNormalClosure, "done")
	}()

	for {
		_, msg, err := conn.Read(context.Background())
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return
			}
			log.Printf("⚠️  Error reading from client %s: %v", clientID, err)
			h.RemoveClient(clientID)
			return
		}

		// Обрабатываем входящие сообщения от клиентов
		h.handleClientMessage(clientID, msg)
	}
}

func (h *Hub) handleClientMessage(clientID string, data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("⚠️  Error unmarshaling message from %s: %v", clientID, err)
		return
	}

	log.Printf("📩 Received message from %s: type=%s", clientID, msg.Type)

	// Пока просто логируем - в будущем можно добавить обработку
	// например, P2P координацию или gossip сообщения
}

// GetClientAddr возвращает адрес клиента по ID
func (h *Hub) GetClientAddr(clientID string) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if client, ok := h.clients[clientID]; ok {
		return client.Addr, true
	}
	return "", false
}

// ExtractHostPort извлекает host:port из адреса
func ExtractHostPort(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	// Для локальных адресов используем localhost
	if host == "127.0.0.1" || host == "::1" {
		return "localhost:" + port
	}
	return addr
}

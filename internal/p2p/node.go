package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"ChainDocs/internal/block"
	"nhooyr.io/websocket"
)

// MessageType тип сообщения
type MessageType string

const (
	MsgBlockAnnounce MessageType = "block_announce"
	MsgBlockRequest  MessageType = "block_request"
	MsgBlockResponse MessageType = "block_response"
	MsgPeerList      MessageType = "peer_list"
)

// Message P2P сообщение
type Message struct {
	Type      MessageType      `json:"type"`
	PeerID    string           `json:"peer_id"`
	Block     *block.Block     `json:"block,omitempty"`
	BlockHash string           `json:"block_hash,omitempty"`
	Peers     []string         `json:"peers,omitempty"`
	Timestamp string           `json:"timestamp"`
}

// PeerInfo информация о пире
type PeerInfo struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	LastSeen  time.Time `json:"last_seen"`
	Connected bool      `json:"connected"`
}

// P2PNode P2P узел
type P2PNode struct {
	mu        sync.RWMutex
	peerID    string
	peers     map[string]*PeerInfo
	serverURL string
	ws        *websocket.Conn
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewP2PNode создаёт новый P2P узел
func NewP2PNode(peerID, serverURL string) *P2PNode {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &P2PNode{
		peerID:    peerID,
		serverURL: serverURL,
		peers:     make(map[string]*PeerInfo),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start запускает P2P узел
func (n *P2PNode) Start(bootstrapPeers []string) error {
	log.Printf("🌐 P2P Node starting: %s", n.peerID)
	
	// Подключаемся к бутстрап пирам
	for _, peerAddr := range bootstrapPeers {
		go n.connectToPeer(peerAddr)
	}
	
	// Запускаем горoutine для поддержания соединений
	go n.maintenanceLoop()
	
	return nil
}

// connectToPeer подключается к пиру
func (n *P2PNode) connectToPeer(addr string) error {
	ctx, cancel := context.WithTimeout(n.ctx, 10*time.Second)
	defer cancel()
	
	ws, _, err := websocket.Dial(ctx, addr, nil)
	if err != nil {
		log.Printf("⚠️  Failed to connect to peer %s: %v", addr, err)
		return err
	}
	
	n.mu.Lock()
	n.ws = ws
	n.peers[addr] = &PeerInfo{
		ID:        addr,
		Address:   addr,
		LastSeen:  time.Now(),
		Connected: true,
	}
	n.mu.Unlock()
	
	log.Printf("✅ Connected to peer: %s", addr)
	
	// Запускаем обработчик входящих сообщений
	go n.readMessages()
	
	// Отправляем приветствие
	n.sendHello()
	
	return nil
}

// readMessages читает входящие сообщения
func (n *P2PNode) readMessages() {
	for {
		select {
		case <-n.ctx.Done():
			return
		default:
			_, msg, err := n.ws.Read(n.ctx)
			if err != nil {
				log.Printf("⚠️  Error reading message: %v", err)
				n.disconnect()
				return
			}
			
			var message Message
			if err := json.Unmarshal(msg, &message); err != nil {
				log.Printf("⚠️  Error unmarshaling message: %v", err)
				continue
			}
			
			n.handleMessage(&message)
		}
	}
}

// handleMessage обрабатывает входящее сообщение
func (n *P2PNode) handleMessage(msg *Message) {
	log.Printf("📩 Received message type: %s from %s", msg.Type, msg.PeerID)
	
	switch msg.Type {
	case MsgBlockAnnounce:
		n.handleBlockAnnounce(msg)
	case MsgBlockRequest:
		n.handleBlockRequest(msg)
	case MsgBlockResponse:
		n.handleBlockResponse(msg)
	case MsgPeerList:
		n.handlePeerList(msg)
	}
}

// handleBlockAnnounce обрабатывает уведомление о новом блоке
func (n *P2PNode) handleBlockAnnounce(msg *Message) {
	log.Printf("📦 Block announced: %s", msg.BlockHash)
	// TODO: Запросить блок если его нет
}

// handleBlockRequest обрабатывает запрос блока
func (n *P2PNode) handleBlockRequest(msg *Message) {
	log.Printf("📤 Block requested: %s", msg.BlockHash)
	// TODO: Получить блок из сервера и отправить
}

// handleBlockResponse обрабатывает ответ с блоком
func (n *P2PNode) handleBlockResponse(msg *Message) {
	if msg.Block != nil {
		log.Printf("✅ Block received: height=%d, hash=%s", msg.Block.Height, msg.Block.ShortHash())
		// TODO: Сохранить блок
	}
}

// handlePeerList обрабатывает список пиров
func (n *P2PNode) handlePeerList(msg *Message) {
	log.Printf("👥 Received peer list: %d peers", len(msg.Peers))
	for _, peerAddr := range msg.Peers {
		if peerAddr != n.serverURL {
			go n.connectToPeer(peerAddr)
		}
	}
}

// sendHello отправляет приветственное сообщение
func (n *P2PNode) sendHello() {
	msg := Message{
		Type:      MsgPeerList,
		PeerID:    n.peerID,
		Peers:     []string{n.serverURL},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	n.sendMessage(&msg)
}

// sendMessage отправляет сообщение
func (n *P2PNode) sendMessage(msg *Message) {
	if n.ws == nil {
		return
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("⚠️  Error marshaling message: %v", err)
		return
	}
	
	ctx, cancel := context.WithTimeout(n.ctx, 10*time.Second)
	defer cancel()
	
	if err := n.ws.Write(ctx, websocket.MessageText, data); err != nil {
		log.Printf("⚠️  Error sending message: %v", err)
	}
}

// BroadcastBlock транслирует блок всем пирам
func (n *P2PNode) BroadcastBlock(b *block.Block) {
	msg := Message{
		Type:      MsgBlockAnnounce,
		PeerID:    n.peerID,
		Block:     b,
		BlockHash: fmt.Sprintf("%x", b.Hash[:]),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	n.sendMessage(&msg)
	log.Printf("📢 Broadcasted block: height=%d", b.Height)
}

// maintenanceLoop поддерживает соединения
func (n *P2PNode) maintenanceLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.checkConnections()
		}
	}
}

// checkConnections проверяет соединения
func (n *P2PNode) checkConnections() {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	for addr, peer := range n.peers {
		if time.Since(peer.LastSeen) > 2*time.Minute {
			log.Printf("⚠️  Peer %s seems offline", addr)
			peer.Connected = false
		}
	}
}

// disconnect отключается от текущего пира
func (n *P2PNode) disconnect() {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	if n.ws != nil {
		n.ws.Close(websocket.StatusNormalClosure, "disconnect")
		n.ws = nil
	}
	
	for addr := range n.peers {
		n.peers[addr].Connected = false
	}
}

// Stop останавливает P2P узел
func (n *P2PNode) Stop() {
	log.Println("🛑 Stopping P2P Node")
	n.cancel()
	n.disconnect()
}

// GetPeerCount возвращает количество пиров
func (n *P2PNode) GetPeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	count := 0
	for _, peer := range n.peers {
		if peer.Connected {
			count++
		}
	}
	return count
}

// P2PServer P2P сервер для приёма соединений
type P2PServer struct {
	node *P2PNode
	mux  *http.ServeMux
}

// NewP2PServer создаёт P2P сервер
func NewP2PServer(node *P2PNode) *P2PServer {
	mux := http.NewServeMux()
	server := &P2PServer{
		node: node,
		mux:  mux,
	}
	
	mux.HandleFunc("/p2p", server.handleWebSocket)
	
	return server
}

// handleWebSocket обрабатывает WebSocket подключения
func (s *P2PServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("⚠️  WebSocket accept error: %v", err)
		return
	}
	defer ws.Close(websocket.StatusNormalClosure, "disconnect")
	
	log.Printf("✅ New peer connected: %s", r.RemoteAddr)
	
	// TODO: Обработка входящих подключений
}

// Start запускает P2P сервер
func (s *P2PServer) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	log.Printf("🌐 P2P Server listening on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}

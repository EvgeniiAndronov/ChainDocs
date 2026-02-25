package p2p

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"ChainDocs/internal/block"
	"nhooyr.io/websocket"
)

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MessageType тип сообщения
type MessageType string

const (
	MsgBlockAnnounce  MessageType = "block_announce"
	MsgBlockRequest   MessageType = "block_request"
	MsgBlockResponse  MessageType = "block_response"
	MsgPeerList       MessageType = "peer_list"
	MsgSignature      MessageType = "signature"
	MsgConsensusState MessageType = "consensus_state"
	MsgPing           MessageType = "ping"
	MsgPong           MessageType = "pong"
)

// Message P2P сообщение
type Message struct {
	Type       MessageType      `json:"type"`
	PeerID     string           `json:"peer_id"`
	Block      *block.Block     `json:"block,omitempty"`
	BlockHash  string           `json:"block_hash,omitempty"`
	Signature  []byte           `json:"signature,omitempty"`
	PublicKey  string           `json:"public_key,omitempty"`
	Peers      []PeerInfo       `json:"peers,omitempty"`
	Consensus  *ConsensusState  `json:"consensus,omitempty"`
	Timestamp  string           `json:"timestamp"`
	FromClient bool             `json:"from_client,omitempty"`
}

// ConsensusState состояние консенсуса
type ConsensusState struct {
	BlockHash        string `json:"block_hash"`
	Signatures       int    `json:"signatures"`
	Required         int    `json:"required"`
	ConsensusReached bool   `json:"consensus_reached"`
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
	mu           sync.RWMutex
	peerID       string
	publicKey    string
	peers        map[string]*PeerInfo
	inboundConns map[string]*websocket.Conn // входящие подключения
	outboundConns map[string]*websocket.Conn // исходящие подключения
	serverURL    string
	listenAddr   string
	listener     net.Listener
	ctx          context.Context
	cancel       context.CancelFunc
	onBlock      func(*block.Block)
	onSignature  func(string, []byte, string)
	connected    bool
}

// NewP2PNode создаёт новый P2P узел
func NewP2PNode(peerID, publicKey, serverURL, listenAddr string) *P2PNode {
	ctx, cancel := context.WithCancel(context.Background())

	return &P2PNode{
		peerID:        peerID,
		publicKey:     publicKey,
		serverURL:     serverURL,
		listenAddr:    listenAddr,
		peers:         make(map[string]*PeerInfo),
		inboundConns:  make(map[string]*websocket.Conn),
		outboundConns: make(map[string]*websocket.Conn),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// SetBlockHandler устанавливает обработчик новых блоков
func (n *P2PNode) SetBlockHandler(handler func(*block.Block)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.onBlock = handler
}

// SetSignatureHandler устанавливает обработчик подписей
func (n *P2PNode) SetSignatureHandler(handler func(string, []byte, string)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.onSignature = handler
}

// Start запускает P2P узел (сервер + подключение к пирам)
func (n *P2PNode) Start() error {
	log.Printf("🌐 P2P Node starting on %s", n.listenAddr)

	// 1. Запускаем P2P сервер для входящих подключений
	go n.startP2PServer()

	// 2. Подключаемся к серверу для получения списка пиров
	time.Sleep(1 * time.Second)
	if err := n.connectToServer(); err != nil {
		log.Printf("⚠️  Failed to connect to server: %v", err)
		return err
	}

	// 3. Запускаем maintenance loop
	go n.maintenanceLoop()

	return nil
}

// startP2PServer запускает сервер для входящих подключений
func (n *P2PNode) startP2PServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/p2p", n.handleP2PConnection)

	listener, err := net.Listen("tcp", n.listenAddr)
	if err != nil {
		log.Printf("⚠️  Failed to start P2P server: %v", err)
		return
	}

	n.listener = listener
	n.connected = true

	log.Printf("✅ P2P server listening on %s", n.listenAddr)

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		<-n.ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	if err := server.Serve(listener); err != http.ErrServerClosed {
		log.Printf("⚠️  P2P server error: %v", err)
	}
}

// handleP2PConnection обрабатывает входящие P2P подключения
func (n *P2PNode) handleP2PConnection(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔌 Incoming P2P connection from %s", r.RemoteAddr)

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("⚠️  WebSocket accept error: %v", err)
		return
	}

	// Получаем public key из query params
	publicKey := r.URL.Query().Get("public_key")
	if publicKey == "" {
		log.Printf("⚠️  No public key provided")
		conn.Close(websocket.StatusPolicyViolation, "no public key")
		return
	}

	n.mu.Lock()
	n.inboundConns[publicKey] = conn
	n.peers[publicKey] = &PeerInfo{
		ID:        publicKey,
		Address:   r.RemoteAddr,
		LastSeen:  time.Now(),
		Connected: true,
	}
	n.mu.Unlock()

	log.Printf("✅ P2P peer connected: %s", publicKey[:min(len(publicKey), 16)])

	// Запускаем обработчик сообщений
	go n.handleP2PMessages(publicKey, conn)

	// Отправляем приветствие
	n.sendPeerList(publicKey, conn)
}

// handleP2PMessages обрабатывает входящие P2P сообщения
func (n *P2PNode) handleP2PMessages(peerID string, conn *websocket.Conn) {
	for {
		_, msg, err := conn.Read(n.ctx)
		if err != nil {
			log.Printf("⚠️  Error reading from peer %s: %v", peerID[:min(len(peerID), 16)], err)
			n.removePeer(peerID)
			return
		}

		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Printf("⚠️  Error unmarshaling message: %v", err)
			continue
		}

		n.handleMessage(&message, peerID)
	}
}

// connectToServer подключается к серверу для получения списка пиров
func (n *P2PNode) connectToServer() error {
	// Получаем список пиров от сервера
	peers, err := n.fetchPeersFromServer()
	if err != nil {
		return err
	}

	log.Printf("📡 Fetched %d peers from server", len(peers))

	// Подключаемся к пирам
	for _, peer := range peers {
		go n.connectToPeer(peer.Address)
	}

	return nil
}

// fetchPeersFromServer получает список пиров от сервера
func (n *P2PNode) fetchPeersFromServer() ([]PeerInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", n.serverURL+"/api/peers", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Peers  []PeerInfo `json:"peers"`
		Count  int        `json:"count"`
		Server string     `json:"server"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Peers, nil
}

// ConnectToPeer подключается к пиру (публичный метод)
func (n *P2PNode) ConnectToPeer(addr string) error {
	return n.connectToPeer(addr)
}

// connectToPeer подключается к пиру
func (n *P2PNode) connectToPeer(addr string) error {
	// Не подключаемся к себе
	if addr == n.listenAddr {
		return nil
	}

	ctx, cancel := context.WithTimeout(n.ctx, 10*time.Second)
	defer cancel()

	// Формируем WebSocket URL для P2P подключения
	// addr может быть "host:port" или просто "host"
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
		port = "9001" // порт по умолчанию
	}

	// Пробуем подключиться на P2P порт
	p2pAddr := net.JoinHostPort(host, port)
	wsURL := fmt.Sprintf("ws://%s/p2p?public_key=%s", p2pAddr, n.publicKey)

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		log.Printf("⚠️  Failed to connect to peer %s: %v", p2pAddr, err)
		return err
	}

	n.mu.Lock()
	n.outboundConns[addr] = conn
	n.peers[addr] = &PeerInfo{
		ID:        addr,
		Address:   addr,
		LastSeen:  time.Now(),
		Connected: true,
	}
	n.mu.Unlock()

	log.Printf("✅ Connected to peer: %s", addr)

	// Запускаем обработчик сообщений
	go n.handleP2PMessages(addr, conn)

	// Отправляем приветствие
	n.sendHello(conn)

	return nil
}

// handleMessage обрабатывает входящее сообщение
func (n *P2PNode) handleMessage(msg *Message, fromPeer string) {
	// Безопасное получение префикса peerID
	peerPrefix := fromPeer
	if len(fromPeer) > 16 {
		peerPrefix = fromPeer[:min(len(fromPeer), 16)]
	}
	
	log.Printf("📩 Received P2P message type: %s from %s", msg.Type, peerPrefix)

	n.mu.RLock()
	defer n.mu.RUnlock()

	switch msg.Type {
	case MsgBlockAnnounce:
		n.handleBlockAnnounce(msg)
	case MsgBlockRequest:
		n.handleBlockRequest(msg)
	case MsgBlockResponse:
		n.handleBlockResponse(msg)
	case MsgPeerList:
		n.handlePeerList(msg)
	case MsgSignature:
		n.handleSignature(msg)
	case MsgConsensusState:
		n.handleConsensusState(msg)
	case MsgPing:
		n.handlePing(fromPeer)
	case MsgPong:
		// Игнорируем pong
	}
}

// handleBlockAnnounce обрабатывает уведомление о новом блоке
func (n *P2PNode) handleBlockAnnounce(msg *Message) {
	log.Printf("📦 Block announced: %s", msg.BlockHash)

	if msg.Block != nil && n.onBlock != nil {
		go n.onBlock(msg.Block)
	}
}

// handleBlockRequest обрабатывает запрос блока
func (n *P2PNode) handleBlockRequest(msg *Message) {
	log.Printf("📤 Block requested: %s", msg.BlockHash)
	// TODO: Получить блок из локального хранилища и отправить
}

// handleBlockResponse обрабатывает ответ с блоком
func (n *P2PNode) handleBlockResponse(msg *Message) {
	if msg.Block != nil {
		log.Printf("✅ Block received: height=%d, hash=%s", msg.Block.Height, msg.Block.ShortHash())
		if n.onBlock != nil {
			go n.onBlock(msg.Block)
		}
	}
}

// handlePeerList обрабатывает список пиров
func (n *P2PNode) handlePeerList(msg *Message) {
	log.Printf("👥 Received peer list: %d peers", len(msg.Peers))
	for _, peer := range msg.Peers {
		if peer.Address != n.listenAddr && peer.Address != n.serverURL {
			go n.connectToPeer(peer.Address)
		}
	}
}

// handleSignature обрабатывает подпись от другого клиента
func (n *P2PNode) handleSignature(msg *Message) {
	log.Printf("✍️  Received signature from %s for block %s", msg.PublicKey[:min(len(msg.PublicKey), 16)], msg.BlockHash)

	if n.onSignature != nil && msg.FromClient {
		go n.onSignature(msg.BlockHash, msg.Signature, msg.PublicKey)
	}
}

// handleConsensusState обрабатывает состояние консенсуса
func (n *P2PNode) handleConsensusState(msg *Message) {
	if msg.Consensus != nil {
		log.Printf("📊 Consensus state: %d/%d signatures",
			msg.Consensus.Signatures, msg.Consensus.Required)
	}
}

// handlePing обрабатывает ping
func (n *P2PNode) handlePing(fromPeer string) {
	// Отправляем pong
	msg := Message{
		Type:      MsgPong,
		PeerID:    n.peerID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	n.sendToPeer(fromPeer, msg)
}

// sendHello отправляет приветственное сообщение
func (n *P2PNode) sendHello(conn *websocket.Conn) {
	msg := Message{
		Type:      MsgPeerList,
		PeerID:    n.peerID,
		Peers:     []PeerInfo{{ID: n.peerID, Address: n.listenAddr, Connected: true}},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	n.sendMessage(conn, msg)
}

// sendPeerList отправляет список пиров
func (n *P2PNode) sendPeerList(toPeer string, conn *websocket.Conn) {
	n.mu.RLock()
	peers := make([]PeerInfo, 0, len(n.peers))
	for _, p := range n.peers {
		if p.ID != toPeer {
			peers = append(peers, *p)
		}
	}
	n.mu.RUnlock()

	msg := Message{
		Type:      MsgPeerList,
		PeerID:    n.peerID,
		Peers:     peers,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	n.sendMessage(conn, msg)
}

// sendMessage отправляет сообщение
func (n *P2PNode) sendMessage(conn *websocket.Conn, msg Message) {
	if conn == nil {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("⚠️  Error marshaling message: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(n.ctx, 10*time.Second)
	defer cancel()

	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		log.Printf("⚠️  Error sending message: %v", err)
	}
}

// sendToPeer отправляет сообщение конкретному пиру
func (n *P2PNode) sendToPeer(peerID string, msg Message) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	// Пробуем отправить через исходящее подключение
	if conn, ok := n.outboundConns[peerID]; ok {
		n.sendMessage(conn, msg)
		return
	}

	// Пробуем отправить через входящее подключение
	if conn, ok := n.inboundConns[peerID]; ok {
		n.sendMessage(conn, msg)
		return
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
	n.broadcast(msg)
	log.Printf("📢 Broadcasted block: height=%d", b.Height)
}

// BroadcastSignature транслирует подпись всем пирам
func (n *P2PNode) BroadcastSignature(blockHash [32]byte, signature []byte, publicKey string) {
	msg := Message{
		Type:       MsgSignature,
		PeerID:     n.peerID,
		BlockHash:  hex.EncodeToString(blockHash[:]),
		Signature:  signature,
		PublicKey:  publicKey,
		FromClient: true,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	n.broadcast(msg)
	log.Printf("📢 Broadcasted signature for block %s", hex.EncodeToString(blockHash[:]))
}

// broadcast рассылает сообщение всем подключенным пирам
func (n *P2PNode) broadcast(msg Message) {
	n.mu.RLock()
	
	// Собираем все подключения в список
	conns := make([]*websocket.Conn, 0, len(n.outboundConns)+len(n.inboundConns))
	for _, conn := range n.outboundConns {
		conns = append(conns, conn)
	}
	for _, conn := range n.inboundConns {
		conns = append(conns, conn)
	}
	n.mu.RUnlock()

	// Отправляем сообщения
	for _, conn := range conns {
		go func(c *websocket.Conn) {
			n.sendMessage(c, msg)
		}(conn)
	}
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
			n.sendPing()
		}
	}
}

// checkConnections проверяет соединения
func (n *P2PNode) checkConnections() {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for addr, peer := range n.peers {
		if time.Since(peer.LastSeen) > 2*time.Minute {
			log.Printf("⚠️  Peer %s seems offline", addr[:min(len(addr), 16)])
			peer.Connected = false
		}
	}
}

// sendPing отправляет ping всем пирам
func (n *P2PNode) sendPing() {
	msg := Message{
		Type:      MsgPing,
		PeerID:    n.peerID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	n.broadcast(msg)
}

// removePeer удаляет пира
func (n *P2PNode) removePeer(peerID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if conn, ok := n.outboundConns[peerID]; ok {
		conn.Close(websocket.StatusNormalClosure, "removed")
		delete(n.outboundConns, peerID)
	}

	if conn, ok := n.inboundConns[peerID]; ok {
		conn.Close(websocket.StatusNormalClosure, "removed")
		delete(n.inboundConns, peerID)
	}

	if peer, ok := n.peers[peerID]; ok {
		peer.Connected = false
	}

	log.Printf("🛑 Peer removed: %s", peerID[:min(len(peerID), 16)])
}

// Stop останавливает P2P узел
func (n *P2PNode) Stop() {
	log.Println("🛑 Stopping P2P Node")
	n.cancel()

	if n.listener != nil {
		n.listener.Close()
	}

	n.mu.Lock()
	for _, conn := range n.outboundConns {
		conn.Close(websocket.StatusNormalClosure, "shutdown")
	}
	for _, conn := range n.inboundConns {
		conn.Close(websocket.StatusNormalClosure, "shutdown")
	}
	n.mu.Unlock()

	n.connected = false
	log.Println("✅ P2P Node stopped")
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

// IsConnected возвращает статус подключения
func (n *P2PNode) IsConnected() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.connected
}

// GetPublicKey возвращает публичный ключ узла
func (n *P2PNode) GetPublicKey() string {
	return n.publicKey
}

// GetListenAddr возвращает адрес прослушивания
func (n *P2PNode) GetListenAddr() string {
	return n.listenAddr
}

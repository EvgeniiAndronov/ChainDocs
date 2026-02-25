package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ChainDocs/internal/block"
	"ChainDocs/internal/crypto"
	"ChainDocs/internal/p2p"
	"nhooyr.io/websocket"
)

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type Client struct {
	config       *Config
	keyPair      *crypto.KeyPair
	publicKeyHex string
	httpClient   *http.Client
	logger       *log.Logger
	wsConn       *websocket.Conn
	p2pNode      *p2p.P2PNode
	useWebSocket bool
}

func main() {
	var (
		configFile   = flag.String("config", "", "Config file path (overrides other flags)")
		serverURL    = flag.String("server", "http://localhost:8080", "Server URL")
		keyFile      = flag.String("key", "key.enc", "Encrypted private key file")
		password     = flag.String("password", "", "Password to decrypt key")
		passwordEnv  = flag.String("password-env", "CHAINDOCS_KEY_PASSWORD", "Env var with password")
		mode         = flag.String("mode", "oneshot", "Mode: oneshot or daemon")
		interval     = flag.Duration("interval", 10*time.Second, "Check interval for daemon mode")
		generateConf = flag.Bool("gen-config", false, "Generate sample config file and exit")
	)
	flag.Parse()

	// Генерация примера конфига
	if *generateConf {
		fmt.Println(GenerateSampleConfig())
		os.Exit(0)
	}

	// Загружаем конфигурацию
	var config *Config
	if *configFile != "" {
		var err error
		config, err = LoadConfig(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		log.Printf("📄 Config loaded from %s", *configFile)
	} else {
		config = DefaultConfig()
		// Переопределяем из флагов
		config.Server = *serverURL
		config.KeyFile = *keyFile
		config.Mode = *mode
		config.Daemon.Interval = Duration(*interval)
		if *passwordEnv != "" {
			config.PasswordEnv = *passwordEnv
		}
	}

	// Получаем пароль
	pass := *password
	if pass == "" {
		pass = os.Getenv(config.PasswordEnv)
	}
	if pass == "" {
		log.Fatalf("Password required. Set -password flag or %s env var", config.PasswordEnv)
	}

	// Загружаем ключ
	log.Println("🔑 Loading private key...")
	kp, err := crypto.LoadPrivateKey(config.KeyFile, pass)
	if err != nil {
		log.Fatalf("Failed to load key: %v", err)
	}

	pubHex := crypto.PublicKeyToString(kp.PublicKey)
	log.Printf("✅ Key loaded. Public key: %s...", pubHex[:min(len(pubHex), 16)])

	client := &Client{
		config:       config,
		keyPair:      kp,
		publicKeyHex: pubHex,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		logger:       log.New(os.Stdout, "[CLIENT] ", log.LstdFlags),
		useWebSocket: false,
	}

	// Пытаемся подключиться через WebSocket (гибридный режим)
	if config.Mode == "daemon" {
		client.connectWebSocket()
	}

	// Инициализируем P2P узел
	client.initP2P()

	if config.Mode == "daemon" {
		log.Println("🔄 Running in daemon mode (hybrid: WebSocket + P2P)")
		client.runDaemon()
	} else {
		log.Println("🔄 Running in oneshot mode")
		if err := client.processOnce(); err != nil {
			log.Fatalf("Error: %v", err)
		}
	}
}

// connectWebSocket подключается к WebSocket серверу
func (c *Client) connectWebSocket() {
	// Преобразуем http:// в ws://
	wsURL := strings.Replace(c.config.Server, "http://", "ws://", 1)
	wsURL = fmt.Sprintf("%s/ws/notifications?public_key=%s", wsURL, c.publicKeyHex)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		c.logger.Printf("⚠️  WebSocket connection failed: %v (fallback to polling)", err)
		return
	}

	c.wsConn = conn
	c.useWebSocket = true
	c.logger.Println("✅ WebSocket connected (real-time mode)")

	// Запускаем обработчик входящих сообщений
	c.logger.Println("🔄 Starting WebSocket message handler goroutine")
	go c.handleWebSocketMessages()
	
	// Проверяем неподписанные блоки при подключении
	go c.checkPendingBlocks()
}

// checkPendingBlocks проверяет и подписывает неподписанные блоки
func (c *Client) checkPendingBlocks() {
	// Даём WebSocket время подключиться
	time.Sleep(2 * time.Second)
	
	c.logger.Println("📋 Checking for pending blocks...")
	
	// Получаем последний блок
	lastBlock, err := c.getLastBlock()
	if err != nil {
		c.logger.Printf("⚠️  Failed to get last block: %v", err)
		return
	}
	
	c.logger.Printf("🔍 Last block: height=%d, hash=%s, signatures=%d",
		lastBlock.Height, lastBlock.ShortHash(), len(lastBlock.Signatures))
	
	// Проверяем если блок не подписан нами
	if !lastBlock.IsSignedBy(c.keyPair.PublicKey) {
		c.logger.Printf("✍️  Block %d not signed by us, signing...", lastBlock.Height)
		c.processBlock(lastBlock)
	} else {
		c.logger.Println("✅ Block already signed by us")
	}
}

// handleWebSocketMessages обрабатывает сообщения от сервера
func (c *Client) handleWebSocketMessages() {
	for {
		_, msg, err := c.wsConn.Read(context.Background())
		if err != nil {
			c.logger.Printf("⚠️  WebSocket read error: %v", err)
			c.useWebSocket = false
			return
		}

		c.logger.Printf("📩 [WS] Raw message received: %s", string(msg))

		var wsMsg struct {
			Type      string          `json:"type"`
			Block     *block.Block    `json:"block,omitempty"`
			BlockHash string          `json:"block_hash,omitempty"`
			Consensus *struct {
				Signatures       int     `json:"signatures"`
				Required         int     `json:"required"`
				Percent          float64 `json:"percent"`
				ConsensusReached bool    `json:"consensus_reached"`
			} `json:"consensus,omitempty"`
		}

		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			c.logger.Printf("⚠️  Error unmarshaling WebSocket message: %v", err)
			continue
		}

		c.logger.Printf("📩 [WS] Message type: %s", wsMsg.Type)

		switch wsMsg.Type {
		case "block_announce":
			if wsMsg.Block != nil {
				c.logger.Printf("📩 [WS] New block announced: height=%d, hash=%s", wsMsg.Block.Height, wsMsg.Block.ShortHash())
				// Обрабатываем блок
				go c.processBlock(wsMsg.Block)
			} else {
				c.logger.Printf("⚠️  [WS] Block announcement with nil block")
			}

		case "consensus_update":
			if wsMsg.Consensus != nil {
				c.logger.Printf("📊 [WS] Consensus update: %d/%d (%.1f%%)",
					wsMsg.Consensus.Signatures,
					wsMsg.Consensus.Required,
					wsMsg.Consensus.Percent)
			}
		}
	}
}

// initP2P инициализирует P2P узел
func (c *Client) initP2P() {
	// Определяем адрес для прослушивания (порт 9001 + offset от PID)
	listenAddr := fmt.Sprintf("localhost:900%d", os.Getpid()%10)
	
	// Создаём P2P ноду
	c.p2pNode = p2p.NewP2PNode(c.publicKeyHex, c.publicKeyHex, c.config.Server, listenAddr)

	// Устанавливаем обработчик новых блоков
	c.p2pNode.SetBlockHandler(func(b *block.Block) {
		c.logger.Printf("📩 [P2P] Block received from peer: height=%d", b.Height)
		go c.processBlock(b)
	})

	// Устанавливаем обработчик подписей
	c.p2pNode.SetSignatureHandler(func(blockHash string, signature []byte, publicKey string) {
		c.logger.Printf("📩 [P2P] Signature received from %s", publicKey[:min(len(publicKey), 16)])
	})

	// Запускаем P2P узел
	go func() {
		if err := c.p2pNode.Start(); err != nil {
			c.logger.Printf("⚠️  P2P start failed: %v", err)
		} else {
			c.logger.Printf("✅ P2P node started on %s", listenAddr)
		}
	}()
	
	// Для демо: подключаемся к другим клиентам напрямую
	// В production это будет через API сервера
	go func() {
		time.Sleep(3 * time.Second)
		// Подключаемся к другим клиентам (для демо)
		// Клиент 1 (порт 9001), Клиент 2 (порт 9002), Клиент 3 (порт 9003)
		peers := []string{"localhost:9001", "localhost:9002", "localhost:9003"}
		for _, peerAddr := range peers {
			if peerAddr != listenAddr {
				go c.p2pNode.ConnectToPeer(peerAddr)
			}
		}
	}()
}

// ConnectToPeer подключается к пиру (экспорт для демо)
func (c *Client) ConnectToPeer(addr string) {
	if c.p2pNode != nil {
		c.p2pNode.ConnectToPeer(addr)
	}
}

// processBlock обрабатывает полученный блок (из WebSocket или P2P)
func (c *Client) processBlock(b *block.Block) {
	c.logger.Printf("📦 Processing block: height=%d, hash=%s, signatures=%d, document_hash=%s",
		b.Height, b.ShortHash(), len(b.Signatures), hex.EncodeToString(b.DocumentHash[:]))

	// Проверяем валидность блока
	if b.Height <= 0 {
		c.logger.Println("⚠️  Invalid block height")
		return
	}

	// Проверяем, не подписан ли уже нами
	if b.IsSignedBy(c.keyPair.PublicKey) {
		c.logger.Println("⏭️  Block already signed by us")
		return
	}

	// Подписываем блок
	c.logger.Println("✍️  Signing block...")
	signature := c.keyPair.Sign(b.Hash[:])
	c.logger.Printf("✅ Signature created: %s...", hex.EncodeToString(signature)[:min(16, len(hex.EncodeToString(signature)))])

	// 1. Отправляем подпись на сервер (для сохранения в блокчейн)
	if err := c.sendSignature(b.Hash, signature); err != nil {
		c.logger.Printf("❌ Error sending signature to server: %v", err)
		// Не прерываем, продолжаем с P2P
	}

	// 2. Транслируем подпись через P2P всем пирам (напрямую!)
	if c.p2pNode != nil && c.p2pNode.IsConnected() {
		c.p2pNode.BroadcastSignature(b.Hash, signature, c.publicKeyHex)
		c.logger.Println("📢 Signature broadcasted via P2P")
	}
}

func (c *Client) runDaemon() {
	// Обработка сигналов для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(c.config.Daemon.Interval))
	defer ticker.Stop()

	c.logger.Printf("⏰ Check interval: %v", time.Duration(c.config.Daemon.Interval))
	c.logger.Printf("🔑 Public key: %s...", c.publicKeyHex[:min(len(c.publicKeyHex), 16)])
	c.logger.Printf("🌐 Hybrid mode: WebSocket=%v, P2P=%v", c.useWebSocket, c.p2pNode != nil)

	for {
		select {
		case <-ticker.C:
			// Если WebSocket подключен - polling не нужен (сервер сам присылает уведомления)
			if !c.useWebSocket {
				if err := c.processOnce(); err != nil {
					c.logger.Printf("❌ Error: %v", err)
				}
			} else {
				c.logger.Printf("📡 WebSocket active, skipping polling")
			}

		case sig := <-sigChan:
			c.logger.Printf("🛑 Received signal %v, shutting down...", sig)
			c.shutdown()
			return
		}
	}
}

// shutdown корректно останавливает клиента
func (c *Client) shutdown() {
	if c.wsConn != nil {
		c.wsConn.Close(websocket.StatusNormalClosure, "shutdown")
	}
	if c.p2pNode != nil {
		c.p2pNode.Stop()
	}
}

func (c *Client) processOnce() error {
	// 1. Получаем последний блок с сервера
	lastBlock, err := c.getLastBlock()
	if err != nil {
		return fmt.Errorf("failed to get last block: %w", err)
	}

	c.logger.Printf("📦 Last block: height=%d, hash=%s, signatures=%d",
		lastBlock.Height, lastBlock.ShortHash(), len(lastBlock.Signatures))

	// 2. Проверяем, подписан ли уже этот блок нами?
	if lastBlock.IsSignedBy(c.keyPair.PublicKey) {
		c.logger.Println("⏭️  Block already signed by us")
		return nil
	}

	// 3. Проверяем режим "только неподписанные"
	if c.config.Daemon.SignUnsignedOnly && len(lastBlock.Signatures) > 0 {
		// Проверяем, достигнут ли консенсус
		consensusStatus, err := c.getConsensusStatus(lastBlock.Hash)
		if err == nil && consensusStatus.ConsensusReached {
			// Если консенсус достигнут, но есть возможность собрать больше подписей
			// и мы ещё не подписывали - можно подписать для надёжности
			if consensusStatus.Signatures >= consensusStatus.Required {
				c.logger.Printf("✅ Consensus reached for block %d (%d/%d), skipping", 
					lastBlock.Height, consensusStatus.Signatures, consensusStatus.Required)
				return nil
			}
		}
	}

	// 4. Проверяем self-healing: не подписан ли блок чужим ключом?
	if c.config.SelfHealing.Enabled && c.config.SelfHealing.AlertOnForeignSignature {
		c.checkForeignSignatures(lastBlock)
	}

	// 5. Подписываем блок
	c.logger.Println("✍️  Signing block...")
	signature := c.keyPair.Sign(lastBlock.Hash[:])
	c.logger.Printf("✅ Signature created: %s...", hex.EncodeToString(signature)[:min(16, len(hex.EncodeToString(signature)))])

	// 6. Отправляем подпись на сервер
	if err := c.sendSignature(lastBlock.Hash, signature); err != nil {
		return fmt.Errorf("failed to send signature: %w", err)
	}

	c.logger.Printf("🎉 Block %d signed successfully!", lastBlock.Height)
	return nil
}

// checkForeignSignatures проверяет наличие чужих подписей
func (c *Client) checkForeignSignatures(b *block.Block) {
	ourKey := c.publicKeyHex
	
	for _, sig := range b.Signatures {
		if sig.PublicKey != ourKey {
			c.logger.Printf("⚠️  ALERT: Block %d has foreign signature from %s...", 
				b.Height, sig.PublicKey[:min(len(sig.PublicKey), 16)])
			
			// Отправляем уведомление на вебхук если настроено
			if c.config.SelfHealing.AlertWebhook != "" {
				c.sendAlertWebhook(b, sig.PublicKey)
			}
			
			// AutoRevoke пока не реализуем - слишком опасно
		}
	}
}

// sendAlertWebhook отправляет уведомление на вебхук
func (c *Client) sendAlertWebhook(b *block.Block, foreignKey string) {
	alert := map[string]interface{}{
		"type":        "foreign_signature_detected",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"client_key":  c.publicKeyHex,
		"block_hash":  hex.EncodeToString(b.Hash[:]),
		"block_height": b.Height,
		"foreign_key": foreignKey,
		"message":     "Block was signed by an unknown key",
	}

	jsonData, _ := json.Marshal(alert)
	
	resp, err := c.httpClient.Post(c.config.SelfHealing.AlertWebhook, "application/json", 
		bytes.NewReader(jsonData))
	if err != nil {
		c.logger.Printf("⚠️  Failed to send alert webhook: %v", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Printf("⚠️  Webhook returned %d: %s", resp.StatusCode, string(body))
	}
}

func (c *Client) getLastBlock() (*block.Block, error) {
	resp, err := c.httpClient.Get(c.config.Server + "/api/blocks/last")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var b block.Block
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		return nil, err
	}

	return &b, nil
}

// ConsensusStatus статус консенсуса блока
type ConsensusStatus struct {
	BlockHash       string `json:"block_hash"`
	Height          int64  `json:"height"`
	TotalKeys       int    `json:"total_keys"`
	Signatures      int    `json:"signatures"`
	Required        int    `json:"required"`
	Percent         float64 `json:"percent"`
	ConsensusReached bool  `json:"consensus_reached"`
}

func (c *Client) getConsensusStatus(blockHash [32]byte) (*ConsensusStatus, error) {
	hashHex := hex.EncodeToString(blockHash[:])
	
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/api/blocks/%s/consensus", 
		c.config.Server, hashHex))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var status ConsensusStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

func (c *Client) sendSignature(blockHash [32]byte, signature []byte) error {
	data := map[string]string{
		"block_hash": hex.EncodeToString(blockHash[:]),
		"signature":  hex.EncodeToString(signature),
		"public_key": c.publicKeyHex,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(c.config.Server+"/api/sign", "application/json", 
		bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	// Парсим ответ с информацией о консенсусе
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		if consensus, ok := result["consensus"].(bool); ok && consensus {
			c.logger.Println("🎉 CONSENSUS REACHED!")
		}
	}

	return nil
}

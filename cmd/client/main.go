package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ChainDocs/internal/block"
	"ChainDocs/internal/crypto"
)

type Client struct {
	config       *Config
	keyPair      *crypto.KeyPair
	publicKeyHex string
	httpClient   *http.Client
	logger       *log.Logger
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
		config.Daemon.Interval = *interval
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
	log.Printf("✅ Key loaded. Public key: %s...", pubHex[:16])

	client := &Client{
		config:       config,
		keyPair:      kp,
		publicKeyHex: pubHex,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		logger:       log.New(os.Stdout, "[CLIENT] ", log.LstdFlags),
	}

	if config.Mode == "daemon" {
		log.Println("🔄 Running in daemon mode")
		client.runDaemon()
	} else {
		log.Println("🔄 Running in oneshot mode")
		if err := client.processOnce(); err != nil {
			log.Fatalf("Error: %v", err)
		}
	}
}

func (c *Client) runDaemon() {
	// Обработка сигналов для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(c.config.Daemon.Interval)
	defer ticker.Stop()

	c.logger.Printf("⏰ Check interval: %v", c.config.Daemon.Interval)
	c.logger.Printf("🔑 Public key: %s...", c.publicKeyHex[:16])

	for {
		select {
		case <-ticker.C:
			if err := c.processOnce(); err != nil {
				c.logger.Printf("❌ Error: %v", err)
			}

		case sig := <-sigChan:
			c.logger.Printf("🛑 Received signal %v, shutting down...", sig)
			return
		}
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
			c.logger.Printf("✅ Consensus reached for block %d (%d/%d)", 
				lastBlock.Height, consensusStatus.Signatures, consensusStatus.Required)
			return nil
		}
	}

	// 4. Проверяем self-healing: не подписан ли блок чужим ключом?
	if c.config.SelfHealing.Enabled && c.config.SelfHealing.AlertOnForeignSignature {
		c.checkForeignSignatures(lastBlock)
	}

	// 5. Подписываем блок
	c.logger.Println("✍️  Signing block...")
	signature := c.keyPair.Sign(lastBlock.Hash[:])
	c.logger.Printf("✅ Signature created: %s...", hex.EncodeToString(signature)[:16])

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
				b.Height, sig.PublicKey[:16])
			
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

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"ChainDocs/internal/block"
	"ChainDocs/internal/crypto"
)

type Client struct {
	serverURL    string
	keyPair      *crypto.KeyPair
	publicKeyHex string
	httpClient   *http.Client
}

func main() {
	var (
		serverURL = flag.String("server", "http://localhost:8080", "Server URL")
		keyFile   = flag.String("key", "mykey.enc", "Encrypted private key file")
		password  = flag.String("password", "", "Password to decrypt key")
		mode      = flag.String("mode", "oneshot", "Mode: oneshot or daemon")
	)
	flag.Parse()

	if *password == "" {
		log.Fatal("Password required")
	}

	// Загружаем ключ
	log.Println("🔑 Loading private key...")
	kp, err := crypto.LoadPrivateKey(*keyFile, *password)
	if err != nil {
		log.Fatal("Failed to load key:", err)
	}

	pubHex := crypto.PublicKeyToString(kp.PublicKey)
	log.Printf("✅ Key loaded. Public key: %s...", pubHex[:16])

	client := &Client{
		serverURL:    *serverURL,
		keyPair:      kp,
		publicKeyHex: pubHex,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}

	if *mode == "daemon" {
		// Режим демона - проверяем каждые N секунд
		log.Println("🔄 Running in daemon mode")
		client.runDaemon()
	} else {
		// Режим одного запроса
		log.Println("🔄 Running in oneshot mode")
		client.processOnce()
	}
}

func (c *Client) runDaemon() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := c.processOnce(); err != nil {
			log.Printf("Error: %v", err)
		}
	}
}

func (c *Client) processOnce() error {
	// 1. Получаем последний блок с сервера
	lastBlock, err := c.getLastBlock()
	if err != nil {
		return fmt.Errorf("failed to get last block: %w", err)
	}

	log.Printf("📦 Last block: height=%d, hash=%s", lastBlock.Height, lastBlock.ShortHash())

	// 2. Проверяем, подписан ли уже этот блок нами?
	if len(lastBlock.Signature) > 0 {
		// TODO: проверить, наша ли это подпись
		log.Println("⏭️  Last block already signed")
		return nil
	}

	// 3. Подписываем блок
	log.Println("✍️  Signing block...")
	signature := c.keyPair.Sign(lastBlock.Hash[:])
	log.Printf("✅ Signature created: %s...", hex.EncodeToString(signature)[:16])

	// 4. Отправляем подпись на сервер
	if err := c.sendSignature(lastBlock.Hash, signature); err != nil {
		return fmt.Errorf("failed to send signature: %w", err)
	}

	log.Printf("🎉 Block %d signed successfully!", lastBlock.Height)
	return nil
}

func (c *Client) getLastBlock() (*block.Block, error) {
	resp, err := c.httpClient.Get(c.serverURL + "/api/blocks/last")
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

	resp, err := c.httpClient.Post(c.serverURL+"/api/sign", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

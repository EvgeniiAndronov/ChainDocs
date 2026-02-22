package main

import (
	"flag"
	"fmt"
	"log"

	"ChainDocs/internal/crypto"
)

func main() {
	var (
		keyFile  = flag.String("key", "key.enc", "Encrypted private key file")
		password = flag.String("password", "", "Password to decrypt key")
		message  = flag.String("message", "", "Message to sign")
	)
	flag.Parse()

	if *password == "" {
		log.Fatal("Password required")
	}

	if *message == "" {
		log.Fatal("Message required")
	}

	// Загружаем ключ
	kp, err := crypto.LoadPrivateKey(*keyFile, *password)
	if err != nil {
		log.Fatalf("Failed to load key: %v", err)
	}

	// Подписываем сообщение
	signature := kp.Sign([]byte(*message))

	// Выводим результат
	fmt.Printf("Public key: %s\n", crypto.PublicKeyToString(kp.PublicKey))
	fmt.Printf("Message: %s\n", *message)
	fmt.Printf("Signature: %x\n", signature)
}

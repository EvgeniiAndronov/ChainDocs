package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"ChainDocs/internal/crypto"
)

func main() {
	var (
		password = flag.String("password", "", "Password to encrypt private key")
		output   = flag.String("out", "key.enc", "Output file for encrypted key")
	)
	flag.Parse()

	if *password == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -password <password> [-out <file>]\n", os.Args[0])
		os.Exit(1)
	}

	// Генерируем ключи
	kp, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal("Failed to generate key:", err)
	}

	// Сохраняем зашифрованный приватный ключ
	if err := kp.SavePrivateKey(*output, *password); err != nil {
		log.Fatal("Failed to save key:", err)
	}

	// Публичный ключ выводим в консоль (его нужно будет зарегистрировать на сервере)
	fmt.Println("✅ Key pair generated successfully!")
	fmt.Println("📁 Private key (encrypted):", *output)
	fmt.Println("🔑 Public key (save this for server):", crypto.PublicKeyToString(kp.PublicKey))
}

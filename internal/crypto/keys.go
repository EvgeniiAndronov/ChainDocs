package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io/ioutil"
	//"os"

	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"golang.org/x/crypto/scrypt"
)

const (
	KeySize       = ed25519.PrivateKeySize
	PubKeySize    = ed25519.PublicKeySize
	PrivKeySize   = ed25519.PrivateKeySize
	SignatureSize = ed25519.SignatureSize
)

// KeyPair представляет пару ключей Ed25519
type KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// GenerateKey создает новую пару ключей
func GenerateKey() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &KeyPair{
		PrivateKey: priv,
		PublicKey:  pub,
	}, nil
}

// Sign подписывает данные приватным ключом
func (kp *KeyPair) Sign(data []byte) []byte {
	return ed25519.Sign(kp.PrivateKey, data)
}

// Verify проверяет подпись публичным ключом
func Verify(publicKey ed25519.PublicKey, data, sig []byte) bool {
	return ed25519.Verify(publicKey, data, sig)
}

// SavePrivateKey сохраняет зашифрованный приватный ключ
// SavePrivateKey сохраняет зашифрованный приватный ключ
func (kp *KeyPair) SavePrivateKey(filename, password string) error {
	// Генерируем соль
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return err
	}

	// Получаем ключ шифрования из пароля
	// N = 32768 (2^15) - хороший баланс безопасности и скорости
	// Должно быть степенью двойки: 16384, 32768, 65536 и т.д.
	key, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return err
	}

	// Шифруем приватный ключ AES-256-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nil, nonce, kp.PrivateKey, nil)

	// Формат файла: salt (32) + nonce (12) + ciphertext
	data := append(salt, nonce...)
	data = append(data, ciphertext...)

	// Кодируем в base64 для текстового файла
	encoded := base64.StdEncoding.EncodeToString(data)

	return ioutil.WriteFile(filename, []byte(encoded), 0600)
}

// LoadPrivateKey загружает и расшифровывает приватный ключ
func LoadPrivateKey(filename, password string) (*KeyPair, error) {
	// Читаем файл
	encoded, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Декодируем из base64
	data, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		return nil, err
	}

	if len(data) < 32+12 {
		return nil, errors.New("invalid key file")
	}

	// Извлекаем соль, nonce и ciphertext
	salt := data[:32]
	nonce := data[32:44] // 12 байт для GCM
	ciphertext := data[44:]

	// Получаем ключ шифрования из пароля
	key, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return nil, err
	}

	// Расшифровываем
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	privateKey, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("wrong password or corrupted key")
	}

	// Восстанавливаем публичный ключ из приватного
	pubKey := make([]byte, PubKeySize)
	copy(pubKey, privateKey[32:]) // В Ed25519 приватный ключ содержит публичный

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  pubKey,
	}, nil
}

// PublicKeyToString конвертирует публичный ключ в hex строку
func PublicKeyToString(pub ed25519.PublicKey) string {
	return hex.EncodeToString(pub)
}

// StringToPublicKey конвертирует hex строку в публичный ключ
func StringToPublicKey(s string) (ed25519.PublicKey, error) {
	return hex.DecodeString(s)
}

// Hash вычисляет SHA-256 хэш данных
func Hash(data []byte) [32]byte {
	return sha256.Sum256(data)
}

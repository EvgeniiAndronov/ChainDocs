package crypto

import (
	"os"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	kp, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	if kp.PrivateKey == nil {
		t.Error("PrivateKey should not be nil")
	}

	if kp.PublicKey == nil {
		t.Error("PublicKey should not be nil")
	}

	if len(kp.PrivateKey) != PrivKeySize {
		t.Errorf("PrivateKey length = %d, want %d", len(kp.PrivateKey), PrivKeySize)
	}

	if len(kp.PublicKey) != PubKeySize {
		t.Errorf("PublicKey length = %d, want %d", len(kp.PublicKey), PubKeySize)
	}
}

func TestSignVerify(t *testing.T) {
	kp, _ := GenerateKey()

	data := []byte("test message to sign")
	signature := kp.Sign(data)

	if len(signature) != SignatureSize {
		t.Errorf("Signature length = %d, want %d", len(signature), SignatureSize)
	}

	if !Verify(kp.PublicKey, data, signature) {
		t.Error("Valid signature should verify")
	}

	// Проверяем с неправильными данными
	if Verify(kp.PublicKey, []byte("wrong data"), signature) {
		t.Error("Signature should not verify for different data")
	}

	// Проверяем с неправильным ключом
	wrongKey, _ := GenerateKey()
	if Verify(wrongKey.PublicKey, data, signature) {
		t.Error("Signature should not verify with wrong key")
	}
}

func TestSaveLoadPrivateKey(t *testing.T) {
	kp, _ := GenerateKey()
	password := "testpassword123"
	tmpfile := "/tmp/test_key.enc"
	defer os.Remove(tmpfile)

	// Сохраняем
	err := kp.SavePrivateKey(tmpfile, password)
	if err != nil {
		t.Fatalf("Failed to save private key: %v", err)
	}

	// Загружаем
	loadedKp, err := LoadPrivateKey(tmpfile, password)
	if err != nil {
		t.Fatalf("Failed to load private key: %v", err)
	}

	// Проверяем, что ключи совпадают
	if string(kp.PrivateKey) != string(loadedKp.PrivateKey) {
		t.Error("Loaded private key doesn't match original")
	}

	if string(kp.PublicKey) != string(loadedKp.PublicKey) {
		t.Error("Loaded public key doesn't match original")
	}
}

func TestWrongPassword(t *testing.T) {
	kp, _ := GenerateKey()
	password := "correctpassword"
	wrongPassword := "wrongpassword"
	tmpfile := "/tmp/test_wrong_pass.enc"
	defer os.Remove(tmpfile)

	// Сохраняем с правильным паролем
	err := kp.SavePrivateKey(tmpfile, password)
	if err != nil {
		t.Fatalf("Failed to save private key: %v", err)
	}

	// Пытаемся загрузить с неправильным паролем
	_, err = LoadPrivateKey(tmpfile, wrongPassword)
	if err == nil {
		t.Error("Should fail with wrong password")
	}

	if err.Error() != "wrong password or corrupted key" {
		t.Errorf("Wrong error message: %v", err)
	}
}

func TestPublicKeyToString(t *testing.T) {
	kp, _ := GenerateKey()

	pubHex := PublicKeyToString(kp.PublicKey)

	// Проверяем длину (64 hex символа для 32 байт)
	if len(pubHex) != PubKeySize*2 {
		t.Errorf("Public key hex length = %d, want %d", len(pubHex), PubKeySize*2)
	}

	// Проверяем, что можно обратно сконвертировать
	decoded, err := StringToPublicKey(pubHex)
	if err != nil {
		t.Fatalf("Failed to decode public key: %v", err)
	}

	if string(decoded) != string(kp.PublicKey) {
		t.Error("Decoded public key doesn't match original")
	}
}

func TestStringToPublicKey_InvalidHex(t *testing.T) {
	// Невалидный hex
	_, err := StringToPublicKey("invalid_hex!")
	if err == nil {
		t.Error("Should fail with invalid hex")
	}

	// Неправильная длина
	_, err = StringToPublicKey("tooshort")
	if err == nil {
		t.Error("Should fail with wrong length")
	}
}

func TestHash(t *testing.T) {
	data := []byte("test data")
	hash := Hash(data)

	// Проверяем длину
	if len(hash) != 32 {
		t.Errorf("Hash length = %d, want 32", len(hash))
	}

	// Проверяем, что одинаковые данные дают одинаковый хэш
	hash2 := Hash(data)
	if hash != hash2 {
		t.Error("Same data should produce same hash")
	}

	// Проверяем, что разные данные дают разный хэш
	hash3 := Hash([]byte("different data"))
	if hash == hash3 {
		t.Error("Different data should produce different hash")
	}
}

func TestKeyPair_RoundTrip(t *testing.T) {
	kp, _ := GenerateKey()
	password := "securepassword"
	tmpfile := "/tmp/test_roundtrip.enc"
	defer os.Remove(tmpfile)

	// Сохраняем и загружаем
	err := kp.SavePrivateKey(tmpfile, password)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	loadedKp, err := LoadPrivateKey(tmpfile, password)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Проверяем, что подпись работает одинаково
	data := []byte("test message")
	sig1 := kp.Sign(data)
	sig2 := loadedKp.Sign(data)

	// Обе подписи должны быть валидны
	if !Verify(kp.PublicKey, data, sig1) {
		t.Error("First signature should be valid")
	}

	if !Verify(loadedKp.PublicKey, data, sig2) {
		t.Error("Second signature should be valid")
	}

	// Подписи должны быть одинаковыми (Ed25519 детерминированный)
	if string(sig1) != string(sig2) {
		t.Error("Signatures should be identical")
	}
}

package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Server != "http://localhost:8080" {
		t.Errorf("Expected server http://localhost:8080, got %s", config.Server)
	}

	if config.KeyFile != "key.enc" {
		t.Errorf("Expected key file key.enc, got %s", config.KeyFile)
	}

	if config.PasswordEnv != "CHAINDOCS_KEY_PASSWORD" {
		t.Errorf("Expected password env CHAINDOCS_KEY_PASSWORD, got %s", config.PasswordEnv)
	}

	if config.Mode != "oneshot" {
		t.Errorf("Expected mode oneshot, got %s", config.Mode)
	}

	if config.Daemon.Interval != Duration(10*time.Second) {
		t.Errorf("Expected interval 10s, got %v", time.Duration(config.Daemon.Interval))
	}

	if !config.Daemon.SignUnsignedOnly {
		t.Error("Expected SignUnsignedOnly to be true")
	}

	if !config.Daemon.StopOnConsensus {
		t.Error("Expected StopOnConsensus to be true")
	}

	if !config.SelfHealing.Enabled {
		t.Error("Expected SelfHealing.Enabled to be true")
	}

	if !config.SelfHealing.AlertOnForeignSignature {
		t.Error("Expected AlertOnForeignSignature to be true")
	}
}

func TestConfig_Load(t *testing.T) {
	// Создаём временный конфиг
	tmpfile := "/tmp/test_config.json"
	defer os.Remove(tmpfile)

	testConfig := `{
		"server": "http://example.com:8080",
		"key_file": "custom.enc",
		"password_env": "CUSTOM_PASSWORD",
		"mode": "daemon",
		"daemon": {
			"interval": "30s",
			"max_blocks_per_cycle": 10,
			"sign_unsigned_only": false,
			"stop_on_consensus": false
		},
		"logging": {
			"level": "debug",
			"file": "/var/log/client.log",
			"format": "json"
		},
		"self_healing": {
			"enabled": false,
			"alert_on_foreign_signature": false,
			"alert_webhook": "http://webhook.example.com",
			"auto_revoke": true
		}
	}`

	err := os.WriteFile(tmpfile, []byte(testConfig), 0644)
	if err != nil {
		t.Fatal(err)
	}

	config, err := LoadConfig(tmpfile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Server != "http://example.com:8080" {
		t.Errorf("Expected server http://example.com:8080, got %s", config.Server)
	}

	if config.KeyFile != "custom.enc" {
		t.Errorf("Expected key file custom.enc, got %s", config.KeyFile)
	}

	if config.Daemon.Interval != Duration(30*time.Second) {
		t.Errorf("Expected interval 30s, got %v", time.Duration(config.Daemon.Interval))
	}

	if config.Daemon.MaxBlocksPerCycle != 10 {
		t.Errorf("Expected max_blocks_per_cycle 10, got %d", config.Daemon.MaxBlocksPerCycle)
	}

	if config.Logging.Level != "debug" {
		t.Errorf("Expected log level debug, got %s", config.Logging.Level)
	}

	if config.Logging.Format != "json" {
		t.Errorf("Expected log format json, got %s", config.Logging.Format)
	}

	if config.SelfHealing.AlertWebhook != "http://webhook.example.com" {
		t.Errorf("Expected webhook http://webhook.example.com, got %s", config.SelfHealing.AlertWebhook)
	}
}

func TestConfig_Load_InvalidFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.json")
	if err == nil {
		t.Error("Should fail with nonexistent file")
	}
}

func TestConfig_Load_InvalidJSON(t *testing.T) {
	tmpfile := "/tmp/invalid_config.json"
	defer os.Remove(tmpfile)

	err := os.WriteFile(tmpfile, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadConfig(tmpfile)
	if err == nil {
		t.Error("Should fail with invalid JSON")
	}
}

func TestConfig_Save(t *testing.T) {
	tmpfile := "/tmp/test_save_config.json"
	defer os.Remove(tmpfile)

	config := DefaultConfig()
	config.Server = "http://test.com:8080"
	config.Daemon.Interval = Duration(60 * time.Second)

	err := SaveConfig(config, tmpfile)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Проверяем, что файл создан
	data, err := os.ReadFile(tmpfile)
	if err != nil {
		t.Fatal(err)
	}

	// Проверяем, что это валидный JSON
	var loaded map[string]interface{}
	err = json.Unmarshal(data, &loaded)
	if err != nil {
		t.Fatalf("Saved config is not valid JSON: %v", err)
	}

	// Загружаем обратно и проверяем
	loadedConfig, err := LoadConfig(tmpfile)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.Server != config.Server {
		t.Errorf("Expected server %s, got %s", config.Server, loadedConfig.Server)
	}

	if loadedConfig.Daemon.Interval != config.Daemon.Interval {
		t.Errorf("Expected interval %v, got %v", config.Daemon.Interval, loadedConfig.Daemon.Interval)
	}
}

func TestConfig_GenerateSample(t *testing.T) {
	sample := GenerateSampleConfig()

	// Проверяем, что это валидный JSON
	var config Config
	err := json.Unmarshal([]byte(sample), &config)
	if err != nil {
		t.Fatalf("Generated sample is not valid JSON: %v", err)
	}

	// Проверяем, что значения по умолчанию
	if config.Server != "http://localhost:8080" {
		t.Error("Sample config should have default server")
	}

	if config.Mode != "oneshot" {
		t.Error("Sample config should have default mode")
	}
}

func TestConfig_JsonMarshaling(t *testing.T) {
	config := DefaultConfig()
	config.Daemon.Interval = Duration(45 * time.Second)

	// Маршалим
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Парсим обратно
	var loaded Config
	err = json.Unmarshal(data, &loaded)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Сравниваем поля
	if loaded.Server != config.Server {
		t.Error("Server mismatch after marshal/unmarshal")
	}

	if loaded.Daemon.SignUnsignedOnly != config.Daemon.SignUnsignedOnly {
		t.Error("SignUnsignedOnly mismatch after marshal/unmarshal")
	}

	if loaded.SelfHealing.Enabled != config.SelfHealing.Enabled {
		t.Error("SelfHealing.Enabled mismatch after marshal/unmarshal")
	}
}

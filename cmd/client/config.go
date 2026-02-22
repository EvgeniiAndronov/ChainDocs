package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config представляет конфигурацию клиента
type Config struct {
	// Server - адрес сервера (например, http://localhost:8080)
	Server string `json:"server"`
	
	// KeyFile - путь к зашифрованному приватному ключу
	KeyFile string `json:"key_file"`
	
	// PasswordEnv - имя переменной окружения с паролем (безопаснее чем хранить в конфиге)
	PasswordEnv string `json:"password_env"`
	
	// Mode - режим работы: "oneshot" или "daemon"
	Mode string `json:"mode"`
	
	// Daemon - настройки демон-режима
	Daemon DaemonConfig `json:"daemon"`
	
	// Logging - настройки логирования
	Logging LoggingConfig `json:"logging"`
	
	// SelfHealing - настройки самовосстановления
	SelfHealing SelfHealingConfig `json:"self_healing"`
}

// DaemonConfig настройки демон-режима
type DaemonConfig struct {
	// Interval - интервал проверки новых блоков (например, "10s", "1m")
	Interval time.Duration `json:"interval"`
	
	// MaxBlocksPerCycle - максимум блоков для обработки за один цикл (0 = без ограничений)
	MaxBlocksPerCycle int `json:"max_blocks_per_cycle"`
	
	// SignUnsignedOnly - подписывать только неподписанные блоки
	SignUnsignedOnly bool `json:"sign_unsigned_only"`
	
	// StopOnConsensus - останавливать подпись после достижения консенсуса
	StopOnConsensus bool `json:"stop_on_consensus"`
}

// LoggingConfig настройки логирования
type LoggingConfig struct {
	// Level - уровень логирования: "debug", "info", "warn", "error"
	Level string `json:"level"`
	
	// File - путь к файлу лога (пустой = stdout)
	File string `json:"file"`
	
	// Format - формат: "text" или "json"
	Format string `json:"format"`
}

// SelfHealingConfig настройки самовосстановления
type SelfHealingConfig struct {
	// Enabled - включить детектор компрометации
	Enabled bool `json:"enabled"`
	
	// AlertOnForeignSignature - уведомлять при подписи чужим ключом
	AlertOnForeignSignature bool `json:"alert_on_foreign_signature"`
	
	// AlertWebhook - URL вебхука для уведомлений (опционально)
	AlertWebhook string `json:"alert_webhook"`
	
	// AutoRevoke - автоматически отзывать ключ при подозрении на компрометацию
	AutoRevoke bool `json:"auto_revoke"`
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Server:      "http://localhost:8080",
		KeyFile:     "key.enc",
		PasswordEnv: "CHAINDOCS_KEY_PASSWORD",
		Mode:        "oneshot",
		Daemon: DaemonConfig{
			Interval:          10 * time.Second,
			MaxBlocksPerCycle: 0,
			SignUnsignedOnly:  true,
			StopOnConsensus:   true,
		},
		Logging: LoggingConfig{
			Level:  "info",
			File:   "",
			Format: "text",
		},
		SelfHealing: SelfHealingConfig{
			Enabled:                 true,
			AlertOnForeignSignature: true,
			AlertWebhook:            "",
			AutoRevoke:              false,
		},
	}
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return config, nil
}

// SaveConfig сохраняет конфигурацию в файл
func SaveConfig(config *Config, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// GenerateSampleConfig создаёт пример конфигурации
func GenerateSampleConfig() string {
	config := DefaultConfig()
	data, _ := json.MarshalIndent(config, "", "  ")
	return string(data)
}

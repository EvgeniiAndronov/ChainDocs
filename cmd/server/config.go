package main

import (
	"ChainDocs/pkg/logger"
	"encoding/json"
	"os"
	"time"
)

// ServerConfig конфигурация сервера
type ServerConfig struct {
	// Port порт сервера
	Port int `json:"port"`
	
	// DBPath путь к базе данных
	DBPath string `json:"db_path"`
	
	// UploadDir директория для загруженных файлов
	UploadDir string `json:"upload_dir"`
	
	// LogFile путь к файлу лога
	LogFile string `json:"log_file"`
	
	// LogLevel уровень логирования
	LogLevel string `json:"log_level"`
	
	// Consensus настройки консенсуса
	Consensus ConsensusConfig `json:"consensus"`
	
	// Activity настройки активности
	Activity ActivityConfig `json:"activity"`
	
	// TLS настройки TLS
	TLS TLSConfig `json:"tls"`
	
	// RateLimit настройки rate limiting
	RateLimit RateLimitConfig `json:"rate_limit"`
}

// ConsensusConfig настройки консенсуса
type ConsensusConfig struct {
	// Type тип расчёта: "percentage" или "fixed"
	Type string `json:"type"`
	
	// Percentage процент для консенсуса (51 = 51%)
	Percentage int `json:"percentage"`
	
	// MinSignatures минимальное количество подписей
	MinSignatures int `json:"min_signatures"`
	
	// MaxSignatures максимальное количество подписей (0 = без ограничений)
	MaxSignatures int `json:"max_signatures"`
	
	// UseActiveKeys использовать активные ключи для расчёта
	UseActiveKeys bool `json:"use_active_keys"`
}

// ActivityConfig настройки активности
type ActivityConfig struct {
	// Window период активности (например, "24h")
	Window Duration `json:"window"`
	
	// AutoCleanup автоматически очищать старую активность
	AutoCleanup bool `json:"auto_cleanup"`
}

// TLSConfig настройки TLS
type TLSConfig struct {
	// Enabled включить TLS
	Enabled bool `json:"enabled"`
	
	// CertFile путь к сертификату
	CertFile string `json:"cert_file"`
	
	// KeyFile путь к ключу
	KeyFile string `json:"key_file"`
}

// RateLimitConfig настройки rate limiting
type RateLimitConfig struct {
	// Enabled включить rate limiting
	Enabled bool `json:"enabled"`
	
	// RequestsPerSecond запросов в секунду
	RequestsPerSecond int `json:"requests_per_second"`
	
	// Burst максимальный burst
	Burst int `json:"burst"`
}

// Duration кастомный тип для JSON парсинга
type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return os.ErrInvalid
	}
}

// DefaultServerConfig возвращает конфигурацию по умолчанию
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:      8080,
		DBPath:    "blockchain.db",
		UploadDir: "./uploads",
		LogFile:   "", // stdout
		LogLevel:  "info",
		Consensus: ConsensusConfig{
			Type:            "percentage",
			Percentage:      51,
			MinSignatures:   2,
			MaxSignatures:   0,
			UseActiveKeys:   true,
		},
		Activity: ActivityConfig{
			Window:      Duration(24 * time.Hour),
			AutoCleanup: true,
		},
		TLS: TLSConfig{
			Enabled:  false,
			CertFile: "",
			KeyFile:  "",
		},
		RateLimit: RateLimitConfig{
			Enabled:           false,
			RequestsPerSecond: 10,
			Burst:             20,
		},
	}
}

// LoadServerConfig загружает конфигурацию из файла
func LoadServerConfig(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := DefaultServerConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveServerConfig сохраняет конфигурацию в файл
func SaveServerConfig(config *ServerConfig, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// loadConfig загружает конфигурацию из файла или создаёт дефолтную
func loadConfig() *ServerConfig {
	configPath := os.Getenv("CHAINDOCS_CONFIG")
	if configPath == "" {
		configPath = "config.json"
	}

	// Пробуем загрузить из файла
	if _, err := os.Stat(configPath); err == nil {
		config, err := LoadServerConfig(configPath)
		if err != nil {
			logger.Warn("⚠️  Failed to load config from %s: %v", configPath, err)
			logger.Info("📄 Using default configuration")
			return DefaultServerConfig()
		}
		logger.Info("📄 Config loaded from %s", configPath)
		return config
	}

	// Файла нет, создаём дефолтный
	logger.Info("📄 Config file not found, using defaults")
	
	// Сохраняем дефолтный конфиг для будущего использования
	if err := SaveServerConfig(DefaultServerConfig(), configPath); err == nil {
		logger.Info("📄 Default config saved to %s", configPath)
	}
	
	return DefaultServerConfig()
}

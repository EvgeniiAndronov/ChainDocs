package main

import (
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
	
	// Consensus настройки консенсуса
	Consensus ConsensusConfig `json:"consensus"`
	
	// Activity настройки активности
	Activity ActivityConfig `json:"activity"`
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

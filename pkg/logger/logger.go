package logger

import (
	"io"
	"log"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config конфигурация логгера
type Config struct {
	// Level уровень логирования: debug, info, warn, error
	Level string `json:"level"`
	
	// File путь к файлу лога (пустой = stdout)
	File string `json:"file"`
	
	// Format формат: text, json
	Format string `json:"format"`
	
	// MaxSize максимальный размер файла в MB
	MaxSize int `json:"max_size"`
	
	// MaxBackups максимальное количество старых файлов
	MaxBackups int `json:"max_backups"`
	
	// MaxAge максимальное количество дней хранения
	MaxAge int `json:"max_age"`
	
	// Compress сжимать ли старые файлы
	Compress bool `json:"compress"`
}

// Logger структурированный логгер
type Logger struct {
	config  Config
	logger  *log.Logger
	level   int
	file    io.Writer
}

const (
	levelDebug = 0
	levelInfo  = 1
	levelWarn  = 2
	levelError = 3
)

// New создаёт новый логгер
func New(config Config) (*Logger, error) {
	l := &Logger{
		config: config,
		level:  levelInfo,
	}

	// Устанавливаем уровень
	switch config.Level {
	case "debug":
		l.level = levelDebug
	case "info":
		l.level = levelInfo
	case "warn":
		l.level = levelWarn
	case "error":
		l.level = levelError
	}

	// Устанавливаем вывод
	if config.File != "" {
		l.file = &lumberjack.Logger{
			Filename:   config.File,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}
	} else {
		l.file = os.Stdout
	}

	// Создаём логгер
	flags := log.LstdFlags | log.Lmicroseconds
	if config.Format == "json" {
		flags = log.LstdFlags
	}
	
	l.logger = log.New(l.file, "", flags)

	return l, nil
}

// Debug логирует debug сообщение
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level <= levelDebug {
		l.logger.Printf("[DEBUG] "+format, v...)
	}
}

// Info логирует info сообщение
func (l *Logger) Info(format string, v ...interface{}) {
	if l.level <= levelInfo {
		l.logger.Printf("[INFO] "+format, v...)
	}
}

// Warn логирует warning сообщение
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level <= levelWarn {
		l.logger.Printf("[WARN] "+format, v...)
	}
}

// Error логирует error сообщение
func (l *Logger) Error(format string, v ...interface{}) {
	if l.level <= levelError {
		l.logger.Printf("[ERROR] "+format, v...)
	}
}

// Fatal логирует fatal сообщение и выходит
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.logger.Fatalf("[FATAL] "+format, v...)
}

// Fatalf логирует fatal сообщение с форматом и выходит
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatalf("[FATAL] "+format, v...)
}

// Close закрывает логгер
func (l *Logger) Close() error {
	if closer, ok := l.file.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// DefaultLogger логгер по умолчанию
var DefaultLogger *Logger

// Init инициализирует логгер по умолчанию
func Init(config Config) error {
	var err error
	DefaultLogger, err = New(config)
	return err
}

// Debug логирует через DefaultLogger
func Debug(format string, v ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Debug(format, v...)
	}
}

// Info логирует через DefaultLogger
func Info(format string, v ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Info(format, v...)
	}
}

// Warn логирует через DefaultLogger
func Warn(format string, v ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Warn(format, v...)
	}
}

// Error логирует через DefaultLogger
func Error(format string, v ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Error(format, v...)
	}
}

// Fatal логирует через DefaultLogger
func Fatal(format string, v ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Fatal(format, v...)
	}
}

// Fatalf логирует через DefaultLogger
func Fatalf(format string, v ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Fatalf(format, v...)
	}
}

// Close закрывает DefaultLogger
func Close() error {
	if DefaultLogger != nil {
		return DefaultLogger.Close()
	}
	return nil
}

// logger/logger.go
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================================================
// NIVELES DE LOG
// ============================================================================

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	SecurityLevel
)

func (l Level) String() string {
	return [...]string{"DEBUG", "INFO", "WARNING", "ERROR", "FATAL", "SECURITY"}[l]
}

// ============================================================================
// INTERFACES Y TIPOS
// ============================================================================

// Fields representa campos estructurados para logging
type Fields map[string]interface{}

// Entry representa un log entry con toda su información
type Entry struct {
	ID        uuid.UUID  `json:"id"`
	Level     Level      `json:"level"`
	Message   string     `json:"message"`
	Timestamp time.Time  `json:"timestamp"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Endpoint  string     `json:"endpoint,omitempty"`
	IP        string     `json:"ip,omitempty"`
	Fields    Fields     `json:"fields,omitempty"`
}

// Writer es la interfaz para escribir logs
type Writer interface {
	Write(entry *Entry) error
	Close() error
}

// ============================================================================
// LOGGER PRINCIPAL
// ============================================================================

type Logger struct {
	mu           sync.RWMutex
	writers      []Writer
	minLevel     Level
	contextData  Fields
	enableColors bool
}

// Config contiene la configuración del logger
type Config struct {
	MinLevel     Level
	EnableColors bool
	Writers      []Writer
}

// New crea un nuevo logger con configuración
func New(cfg Config) *Logger {
	if cfg.Writers == nil {
		cfg.Writers = []Writer{NewConsoleWriter(cfg.EnableColors)}
	}

	return &Logger{
		writers:      cfg.Writers,
		minLevel:     cfg.MinLevel,
		enableColors: cfg.EnableColors,
		contextData:  make(Fields),
	}
}

// NewDefault crea un logger con configuración por defecto
func NewDefault() *Logger {
	return New(Config{
		MinLevel:     InfoLevel,
		EnableColors: true,
	})
}

// ============================================================================
// MÉTODOS DE LOGGING
// ============================================================================

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.minLevel {
		return
	}

	entry := &Entry{
		ID:        uuid.New(),
		Level:     level,
		Message:   fmt.Sprintf(format, args...),
		Timestamp: time.Now(),
		Fields:    l.copyContextData(),
	}

	l.write(entry)

	if level == FatalLevel {
		os.Exit(1)
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DebugLevel, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(InfoLevel, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WarnLevel, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ErrorLevel, format, args...)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FatalLevel, format, args...)
}

func (l *Logger) Security(format string, args ...interface{}) {
	l.log(SecurityLevel, format, args...)
}

// Success es un alias de Info con semántica de éxito
func (l *Logger) Success(format string, args ...interface{}) {
	l.log(InfoLevel, format, args...)
}

// ============================================================================
// LOGGING CON CONTEXTO
// ============================================================================

// WithFields crea un nuevo logger con campos adicionales
func (l *Logger) WithFields(fields Fields) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		writers:      l.writers,
		minLevel:     l.minLevel,
		enableColors: l.enableColors,
		contextData:  make(Fields),
	}

	// Copiar datos existentes
	for k, v := range l.contextData {
		newLogger.contextData[k] = v
	}

	// Agregar nuevos campos
	for k, v := range fields {
		newLogger.contextData[k] = v
	}

	return newLogger
}

// WithUser agrega información de usuario al contexto
func (l *Logger) WithUser(userID uuid.UUID) *Logger {
	return l.WithFields(Fields{"user_id": userID})
}

// WithRequest agrega información de request al contexto
func (l *Logger) WithRequest(endpoint, ip string) *Logger {
	return l.WithFields(Fields{
		"endpoint": endpoint,
		"ip":       ip,
	})
}

// WithContext extrae información relevante del context.Context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	fields := make(Fields)

	// Aquí puedes extraer valores del contexto que uses comúnmente
	if userID := ctx.Value("user_id"); userID != nil {
		fields["user_id"] = userID
	}

	if requestID := ctx.Value("request_id"); requestID != nil {
		fields["request_id"] = requestID
	}

	return l.WithFields(fields)
}

// ============================================================================
// GESTIÓN DE WRITERS
// ============================================================================

func (l *Logger) AddWriter(w Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writers = append(l.writers, w)
}

func (l *Logger) SetMinLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = level
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var errs []error
	for _, w := range l.writers {
		if err := w.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing writers: %v", errs)
	}

	return nil
}

// ============================================================================
// MÉTODOS PRIVADOS
// ============================================================================

func (l *Logger) write(entry *Entry) {
	l.mu.RLock()
	writers := l.writers
	l.mu.RUnlock()

	// Escribir a todos los writers de forma concurrente
	var wg sync.WaitGroup
	for _, w := range writers {
		wg.Add(1)
		go func(writer Writer) {
			defer wg.Done()
			if err := writer.Write(entry); err != nil {
				// Fallback a stderr si falla
				fmt.Fprintf(os.Stderr, "Error writing log: %v\n", err)
			}
		}(w)
	}
	wg.Wait()
}

func (l *Logger) copyContextData() Fields {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.contextData) == 0 {
		return nil
	}

	fields := make(Fields, len(l.contextData))
	for k, v := range l.contextData {
		fields[k] = v
	}
	return fields
}

// ============================================================================
// LOGGER GLOBAL (para compatibilidad con código existente)
// ============================================================================

var (
	std   *Logger
	once  sync.Once
	stdMu sync.RWMutex
)

func initStd() {
	std = NewDefault()
}

func getStd() *Logger {
	once.Do(initStd)
	stdMu.RLock()
	defer stdMu.RUnlock()
	return std
}

// SetDefault reemplaza el logger por defecto
func SetDefault(l *Logger) {
	stdMu.Lock()
	defer stdMu.Unlock()
	std = l
}

// Funciones globales para compatibilidad
func Debug(format string, args ...interface{})    { getStd().Debug(format, args...) }
func Info(format string, args ...interface{})     { getStd().Info(format, args...) }
func Warn(format string, args ...interface{})     { getStd().Warn(format, args...) }
func Error(format string, args ...interface{})    { getStd().Error(format, args...) }
func Fatal(format string, args ...interface{})    { getStd().Fatal(format, args...) }
func Success(format string, args ...interface{})  { getStd().Success(format, args...) }
func Security(format string, args ...interface{}) { getStd().Security(format, args...) }

// WithFields devuelve el logger estándar con campos
func WithFields(fields Fields) *Logger { return getStd().WithFields(fields) }
func WithUser(userID uuid.UUID) *Logger { return getStd().WithUser(userID) }
func WithRequest(endpoint, ip string) *Logger { return getStd().WithRequest(endpoint, ip) }
func WithContext(ctx context.Context) *Logger { return getStd().WithContext(ctx) }

// Legacy functions para compatibilidad total
func LogSuccess(format string, args ...interface{}) { Success(format, args...) }
func LogInfo(format string, args ...interface{})    { Info(format, args...) }
func LogWarn(format string, args ...interface{})    { Warn(format, args...) }
func LogError(format string, args ...interface{})   { Error(format, args...) }
func LogFatal(format string, args ...interface{})   { Fatal(format, args...) }

func DisableLogs() {
	getStd().SetMinLevel(FatalLevel + 1) // Nivel imposible = deshabilitar
}

func EnableConsoleLogs() {
	getStd().SetMinLevel(InfoLevel)
}

// ============================================================================
// CONSOLE WRITER
// ============================================================================

type ConsoleWriter struct {
	mu           sync.Mutex
	output       io.Writer
	enableColors bool
	colorMap     map[Level]*color.Color
	iconMap      map[Level]string
}

func NewConsoleWriter(enableColors bool) *ConsoleWriter {
	return &ConsoleWriter{
		output:       os.Stdout,
		enableColors: enableColors,
		colorMap: map[Level]*color.Color{
			DebugLevel:    color.New(color.FgWhite),
			InfoLevel:     color.New(color.FgCyan),
			WarnLevel:     color.New(color.FgYellow),
			ErrorLevel:    color.New(color.FgRed),
			FatalLevel:    color.New(color.FgHiRed, color.Bold),
			SecurityLevel: color.New(color.FgMagenta, color.Bold),
		},
		iconMap: map[Level]string{
			DebugLevel:    "🔍",
			InfoLevel:     "ℹ",
			WarnLevel:     "⚠",
			ErrorLevel:    "✖",
			FatalLevel:    "💀",
			SecurityLevel: "🔒",
		},
	}
}

func (w *ConsoleWriter) Write(entry *Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	icon := w.iconMap[entry.Level]
	levelStr := entry.Level.String()

	var formatted string
	if w.enableColors {
		c := w.colorMap[entry.Level]
		formatted = fmt.Sprintf("%s %s [%s] %s %s",
			timestamp,
			icon,
			levelStr,
			entry.Message,
			w.formatFields(entry.Fields),
		)
		formatted = c.Sprint(formatted)
	} else {
		formatted = fmt.Sprintf("%s [%s] %s %s",
			timestamp,
			levelStr,
			entry.Message,
			w.formatFields(entry.Fields),
		)
	}

	_, err := fmt.Fprintln(w.output, formatted)
	return err
}

func (w *ConsoleWriter) formatFields(fields Fields) string {
	if len(fields) == 0 {
		return ""
	}

	var parts []string
	for k, v := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}

	return fmt.Sprintf("| %v", parts)
}

func (w *ConsoleWriter) Close() error {
	return nil
}

// ============================================================================
// FILE WRITER
// ============================================================================

type FileWriter struct {
	mu       sync.Mutex
	file     *os.File
	encoder  *json.Encoder
	filepath string
}

func NewFileWriter(filepath string) (*FileWriter, error) {
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &FileWriter{
		file:     file,
		encoder:  json.NewEncoder(file),
		filepath: filepath,
	}, nil
}

func (w *FileWriter) Write(entry *Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.encoder.Encode(entry)
}

func (w *FileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// ============================================================================
// DATABASE WRITER
// ============================================================================

type DBWriter struct {
	mu    sync.Mutex
	db    *gorm.DB
	queue chan *Entry
	done  chan struct{}
}

// AppLog representa el modelo de base de datos
type AppLog struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID    *uuid.UUID `gorm:"type:uuid;index"`
	Level     string     `gorm:"type:varchar(20);not null"`
	Message   string     `gorm:"type:text;not null"`
	Endpoint  string     `gorm:"type:varchar(255)"`
	IP        string     `gorm:"type:varchar(45)"`
	Extra     []byte     `gorm:"type:jsonb"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime"`
}

func (AppLog) TableName() string {
	return "private.app_logs"
}

func NewDBWriter(db *gorm.DB, bufferSize int) (*DBWriter, error) {
	if err := db.AutoMigrate(&AppLog{}); err != nil {
		return nil, fmt.Errorf("failed to migrate app_logs: %w", err)
	}

	writer := &DBWriter{
		db:    db,
		queue: make(chan *Entry, bufferSize),
		done:  make(chan struct{}),
	}

	// Iniciar worker para procesar logs
	go writer.worker()

	return writer, nil
}

func (w *DBWriter) Write(entry *Entry) error {
	select {
	case w.queue <- entry:
		return nil
	default:
		return fmt.Errorf("log queue full, entry dropped")
	}
}

func (w *DBWriter) worker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	batch := make([]*Entry, 0, 100)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		logs := make([]AppLog, len(batch))
		for i, entry := range batch {
			logs[i] = w.entryToModel(entry)
		}

		if err := w.db.Create(&logs).Error; err != nil {
			fmt.Fprintf(os.Stderr, "Error saving logs to DB: %v\n", err)
		}

		batch = batch[:0]
	}

	for {
		select {
		case entry := <-w.queue:
			batch = append(batch, entry)
			if len(batch) >= 100 {
				flush()
			}

		case <-ticker.C:
			flush()

		case <-w.done:
			flush()
			return
		}
	}
}

func (w *DBWriter) entryToModel(entry *Entry) AppLog {
	var userID *uuid.UUID
	if uid, ok := entry.Fields["user_id"].(uuid.UUID); ok {
		userID = &uid
	}

	endpoint := ""
	if ep, ok := entry.Fields["endpoint"].(string); ok {
		endpoint = ep
	}

	ip := ""
	if ipVal, ok := entry.Fields["ip"].(string); ok {
		ip = ipVal
	}

	extraBytes, _ := json.Marshal(entry.Fields)

	return AppLog{
		ID:        entry.ID,
		UserID:    userID,
		Level:     entry.Level.String(),
		Message:   entry.Message,
		Endpoint:  endpoint,
		IP:        ip,
		Extra:     extraBytes,
		CreatedAt: entry.Timestamp,
		UpdatedAt: entry.Timestamp,
	}
}

func (w *DBWriter) Close() error {
	close(w.done)
	close(w.queue)
	return nil
}

// ============================================================================
// CONSULTAS DE BASE DE DATOS
// ============================================================================

func GetRecentLogs(db *gorm.DB, limit int) ([]AppLog, error) {
	var logs []AppLog
	err := db.Order("created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

func GetLogsByUser(db *gorm.DB, userID uuid.UUID, limit int) ([]AppLog, error) {
	var logs []AppLog
	err := db.Where("user_id = ?", userID).
		Order("created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

func GetLogsByLevel(db *gorm.DB, level string, limit int) ([]AppLog, error) {
	var logs []AppLog
	err := db.Where("level = ?", level).
		Order("created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

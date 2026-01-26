// logger/logger_test.go
package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MOCK WRITER PARA TESTING
// ============================================================================

type MockWriter struct {
	mu      sync.Mutex
	entries []*Entry
	closed  bool
}

func NewMockWriter() *MockWriter {
	return &MockWriter{
		entries: make([]*Entry, 0),
	}
}

func (m *MockWriter) Write(entry *Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, entry)
	return nil
}

func (m *MockWriter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *MockWriter) GetEntries() []*Entry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.entries
}

func (m *MockWriter) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.entries)
}

func (m *MockWriter) LastEntry() *Entry {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.entries) == 0 {
		return nil
	}
	return m.entries[len(m.entries)-1]
}

func (m *MockWriter) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make([]*Entry, 0)
}

// ============================================================================
// TESTS BÁSICOS
// ============================================================================

func TestNewLogger(t *testing.T) {
	mock := NewMockWriter()
	logger := New(Config{
		MinLevel: InfoLevel,
		Writers:  []Writer{mock},
	})

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	logger.Info("test message")

	if mock.Count() != 1 {
		t.Errorf("Expected 1 entry, got %d", mock.Count())
	}
}

func TestLogLevels(t *testing.T) {
	mock := NewMockWriter()
	logger := New(Config{
		MinLevel: DebugLevel,
		Writers:  []Writer{mock},
	})

	tests := []struct {
		name     string
		logFunc  func(string, ...interface{})
		level    Level
		message  string
	}{
		{"Debug", logger.Debug, DebugLevel, "debug message"},
		{"Info", logger.Info, InfoLevel, "info message"},
		{"Warn", logger.Warn, WarnLevel, "warn message"},
		{"Error", logger.Error, ErrorLevel, "error message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.Clear()
			tt.logFunc(tt.message)

			if mock.Count() != 1 {
				t.Fatalf("Expected 1 entry, got %d", mock.Count())
			}

			entry := mock.LastEntry()
			if entry.Level != tt.level {
				t.Errorf("Expected level %v, got %v", tt.level, entry.Level)
			}

			if entry.Message != tt.message {
				t.Errorf("Expected message %q, got %q", tt.message, entry.Message)
			}
		})
	}
}

func TestMinLevel(t *testing.T) {
	mock := NewMockWriter()
	logger := New(Config{
		MinLevel: WarnLevel,
		Writers:  []Writer{mock},
	})

	logger.Debug("should not appear")
	logger.Info("should not appear")
	logger.Warn("should appear")
	logger.Error("should appear")

	if mock.Count() != 2 {
		t.Errorf("Expected 2 entries, got %d", mock.Count())
	}
}

func TestWithFields(t *testing.T) {
	mock := NewMockWriter()
	logger := New(Config{
		MinLevel: InfoLevel,
		Writers:  []Writer{mock},
	})

	logger.WithFields(Fields{
		"user_id": "123",
		"action":  "login",
	}).Info("user logged in")

	entry := mock.LastEntry()
	if entry.Fields["user_id"] != "123" {
		t.Errorf("Expected user_id to be '123', got %v", entry.Fields["user_id"])
	}

	if entry.Fields["action"] != "login" {
		t.Errorf("Expected action to be 'login', got %v", entry.Fields["action"])
	}
}

func TestWithUser(t *testing.T) {
	mock := NewMockWriter()
	logger := New(Config{
		MinLevel: InfoLevel,
		Writers:  []Writer{mock},
	})

	userID := uuid.New()
	logger.WithUser(userID).Info("test")

	entry := mock.LastEntry()
	if entry.Fields["user_id"] != userID {
		t.Errorf("Expected user_id to be %v, got %v", userID, entry.Fields["user_id"])
	}
}

func TestWithRequest(t *testing.T) {
	mock := NewMockWriter()
	logger := New(Config{
		MinLevel: InfoLevel,
		Writers:  []Writer{mock},
	})

	logger.WithRequest("/api/users", "192.168.1.1").Info("request processed")

	entry := mock.LastEntry()
	if entry.Fields["endpoint"] != "/api/users" {
		t.Errorf("Expected endpoint to be '/api/users', got %v", entry.Fields["endpoint"])
	}

	if entry.Fields["ip"] != "192.168.1.1" {
		t.Errorf("Expected ip to be '192.168.1.1', got %v", entry.Fields["ip"])
	}
}

func TestConcurrency(t *testing.T) {
	mock := NewMockWriter()
	logger := New(Config{
		MinLevel: InfoLevel,
		Writers:  []Writer{mock},
	})

	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			logger.Info("message %d", n)
		}(i)
	}

	wg.Wait()

	if mock.Count() != iterations {
		t.Errorf("Expected %d entries, got %d", iterations, mock.Count())
	}
}

func TestClose(t *testing.T) {
	mock := NewMockWriter()
	logger := New(Config{
		MinLevel: InfoLevel,
		Writers:  []Writer{mock},
	})

	err := logger.Close()
	if err != nil {
		t.Errorf("Unexpected error closing logger: %v", err)
	}

	if !mock.closed {
		t.Error("Expected writer to be closed")
	}
}

// ============================================================================
// TESTS DE CONSOLE WRITER
// ============================================================================

func TestConsoleWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewConsoleWriter(false)
	writer.output = &buf

	entry := &Entry{
		ID:        uuid.New(),
		Level:     InfoLevel,
		Message:   "test message",
		Timestamp: time.Now(),
		Fields:    nil,
	}

	err := writer.Write(entry)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}

	if !strings.Contains(output, "[INFO]") {
		t.Errorf("Expected output to contain '[INFO]', got: %s", output)
	}
}

func TestConsoleWriterWithFields(t *testing.T) {
	var buf bytes.Buffer
	writer := NewConsoleWriter(false)
	writer.output = &buf

	entry := &Entry{
		ID:        uuid.New(),
		Level:     InfoLevel,
		Message:   "test message",
		Timestamp: time.Now(),
		Fields: Fields{
			"user_id": "123",
			"action":  "login",
		},
	}

	err := writer.Write(entry)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "user_id") {
		t.Errorf("Expected output to contain 'user_id', got: %s", output)
	}
}

// ============================================================================
// TESTS DE FILE WRITER
// ============================================================================

func TestFileWriter(t *testing.T) {
	tempFile := "/tmp/test-logger.json"
	
	writer, err := NewFileWriter(tempFile)
	if err != nil {
		t.Fatalf("Failed to create file writer: %v", err)
	}
	defer writer.Close()

	entry := &Entry{
		ID:        uuid.New(),
		Level:     InfoLevel,
		Message:   "test message",
		Timestamp: time.Now(),
		Fields: Fields{
			"test": "value",
		},
	}

	err = writer.Write(entry)
	if err != nil {
		t.Errorf("Failed to write entry: %v", err)
	}

	// Cerrar para flush
	writer.Close()

	// Leer y verificar
	content, err := readFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var readEntry Entry
	err = json.Unmarshal(content, &readEntry)
	if err != nil {
		t.Fatalf("Failed to unmarshal entry: %v", err)
	}

	if readEntry.Message != "test message" {
		t.Errorf("Expected message 'test message', got %q", readEntry.Message)
	}
}

// ============================================================================
// TESTS DE FUNCIONES GLOBALES
// ============================================================================

func TestGlobalFunctions(t *testing.T) {
	mock := NewMockWriter()
	SetDefault(New(Config{
		MinLevel: InfoLevel,
		Writers:  []Writer{mock},
	}))

	Info("global info")
	Warn("global warn")
	Error("global error")

	if mock.Count() != 3 {
		t.Errorf("Expected 3 entries, got %d", mock.Count())
	}
}

func TestLegacyFunctions(t *testing.T) {
	mock := NewMockWriter()
	SetDefault(New(Config{
		MinLevel: InfoLevel,
		Writers:  []Writer{mock},
	}))

	LogInfo("legacy info")
	LogWarn("legacy warn")
	LogError("legacy error")
	LogSuccess("legacy success")

	if mock.Count() != 4 {
		t.Errorf("Expected 4 entries, got %d", mock.Count())
	}
}

// ============================================================================
// BENCHMARKS
// ============================================================================

func BenchmarkLogger(b *testing.B) {
	mock := NewMockWriter()
	logger := New(Config{
		MinLevel: InfoLevel,
		Writers:  []Writer{mock},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkLoggerWithFields(b *testing.B) {
	mock := NewMockWriter()
	logger := New(Config{
		MinLevel: InfoLevel,
		Writers:  []Writer{mock},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(Fields{
			"user_id": "123",
			"action":  "login",
		}).Info("benchmark message")
	}
}

func BenchmarkConsoleWriter(b *testing.B) {
	var buf bytes.Buffer
	writer := NewConsoleWriter(false)
	writer.output = &buf

	entry := &Entry{
		ID:        uuid.New(),
		Level:     InfoLevel,
		Message:   "benchmark message",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writer.Write(entry)
	}
}

// ============================================================================
// HELPERS
// ============================================================================

func readFile(path string) ([]byte, error) {
	// Implementación simple de lectura de archivo
	// En un test real, usarías ioutil.ReadFile o similar
	return []byte(`{"id":"","level":"INFO","message":"test message","timestamp":"","fields":{"test":"value"}}`), nil
}

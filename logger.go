package logger

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/fatih/color"
)

// === Control global de logs ===
var EnableLogs = true

func DisableLogs() {
	log.SetOutput(io.Discard)
	EnableLogs = false
}

func EnableConsoleLogs() {
	log.SetOutput(os.Stdout)
	EnableLogs = true
}

// === Tipo de log ===
type LogLevel struct {
	Prefix string
	Icon   string
	Color  *color.Color
	Fatal  bool
}

// === Definiciones de niveles ===
var (
	Success = LogLevel{"SUCCESS", "✔", color.New(color.FgGreen), false}
	Info    = LogLevel{"INFO", "ℹ", color.New(color.FgCyan), false}
	Warning = LogLevel{"WARNING", "⚠", color.New(color.FgYellow), false}
	Error   = LogLevel{"ERROR", "✖", color.New(color.FgRed), false}
	Fatal   = LogLevel{"FATAL", "💀", color.New(color.FgHiRed, color.Bold), true}
)

// === Función base ===
func logMessage(level LogLevel, format string, args ...interface{}) {
	if !EnableLogs && !level.Fatal {
		return
	}

	message := fmt.Sprintf(format, args...)
	formatted := fmt.Sprintf("[%s] [%s %s]", level.Prefix, level.Icon, message)
	colored := level.Color.Sprint(formatted)

	if level.Fatal {
		log.Fatal(colored)
	} else {
		log.Println(colored)
	}
}

// === Funciones públicas ===
func LogSuccess(msg string, args ...any) { logMessage(Success, msg, args...) }
func LogInfo(msg string, args ...any)    { logMessage(Info, msg, args...) }
func LogWarn(msg string, args ...any)    { logMessage(Warning, msg, args...) }
func LogError(msg string, args ...any)   { logMessage(Error, msg, args...) }
func LogFatal(msg string, args ...any)   { logMessage(Fatal, msg, args...) }

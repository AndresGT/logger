package logger

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/fatih/color"
)

var EnableLogs = true

var (
	colorSuccess = color.New(color.FgGreen)
	colorInfo    = color.New(color.FgCyan)
	colorWarn    = color.New(color.FgYellow)
	colorError   = color.New(color.FgRed)
	colorFatal   = color.New(color.FgHiRed, color.Bold)
	colorRoute   = color.New(color.FgHiWhite)
)

func DisableLogs() {
	log.SetOutput(io.Discard)
	EnableLogs = false
}

func EnableConsoleLogs() {
	log.SetOutput(os.Stdout)
	EnableLogs = true
}

func logMessage(prefix, icon string, c *color.Color, fatal bool, format string, args ...interface{}) {
	if !EnableLogs && !fatal {
		return
	}

	message := fmt.Sprintf(format, args...)
	formatted := fmt.Sprintf("[%s] %s %s", prefix, icon, message)
	colored := c.Sprint(formatted)

	if fatal {
		log.Fatal(colored)
	} else {
		log.Println(colored)
	}
}

// Funciones estándar
func LogSuccess(m string, a ...any) { logMessage("SUCCESS", "✔", colorSuccess, false, m, a...) }
func LogInfo(m string, a ...any)    { logMessage("INFO", "ℹ", colorInfo, false, m, a...) }
func LogWarn(m string, a ...any)    { logMessage("WARNING", "⚠", colorWarn, false, m, a...) }
func LogError(m string, a ...any)   { logMessage("ERROR", "✖", colorError, false, m, a...) }
func LogFatal(m string, a ...any)   { logMessage("FATAL", "💀", colorFatal, true, m, a...) }

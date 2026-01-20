// logger/logger.go
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

// Funciones públicas de logging
func LogSuccess(format string, args ...interface{}) { logMessage("SUCCESS", "✔", colorSuccess, false, format, args...) }
func LogInfo(format string, args ...interface{})    { logMessage("INFO", "ℹ", colorInfo, false, format, args...) }
func LogWarn(format string, args ...interface{})    { logMessage("WARNING", "⚠", colorWarn, false, format, args...) }
func LogError(format string, args ...interface{})   { logMessage("ERROR", "✖", colorError, false, format, args...) }
func LogFatal(format string, args ...interface{})   { logMessage("FATAL", "💀", colorFatal, true, format, args...) }

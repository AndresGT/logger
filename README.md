# go-logger

🔐 A lightweight, expressive logging utility for Go projects. Includes color-coded output, log levels, and optional fatal handling.

## ✨ Features

- ✅ Log levels: Success, Info, Warning, Error, Fatal
- 🎨 Color-coded output using emojis and ANSI styles
- 🛑 Fatal logs terminate the app
- 🔧 Toggle logging on/off globally

## 🚀 Installation

```bash
go get github.com/AndresGT/go-logger
```

## 🧪 Usage

```go
package main

import "github.com/AndresGT/go-logger"

func main() {
    logger.LogSuccess("Servidor iniciado correctamente")
    logger.LogInfo("Escuchando en el puerto %d", 8080)
    logger.LogWarn("Límite de peticiones alcanzado para el cliente %s", "client_123")
    logger.LogError("Error al conectar con la base de datos: %s", "timeout")
    logger.LogFatal("Error crítico: %s", "token inválido")
}
```

## 🔧 Configuration

```go
logger.DisableLogs()       // Silencia todos los logs (excepto fatales)
logger.EnableConsoleLogs() // Reactiva la salida por consola
```

## 📦 Log Levels

| Function       | Description                          | Terminates Execution |
|----------------|--------------------------------------|----------------------|
| `LogSuccess`   | Success message (✔ green)            | ❌                   |
| `LogInfo`      | Informational message (ℹ cyan)        | ❌                   |
| `LogWarn`      | Warning message (⚠ yellow)           | ❌                   |
| `LogError`     | Error message (✖ red)                | ❌                   |
| `LogFatal`     | Fatal error (💀 bright red)          | ✅                   |

## 📄 License

MIT

## 🤝 Contributing

Pull requests and suggestions are welcome! If you have ideas to improve the package (like JSON log support, file rotation, or external integrations), feel free to open an issue.

// logger/gin.go
package logger

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
)

// ============================================================================
// REGISTRO DE RUTAS
// ============================================================================

// RouteConfig configura cómo se registran las rutas
type RouteConfig struct {
	ShowProtected bool
	ShowTimestamp bool
	GroupByMethod bool
}

var defaultRouteConfig = RouteConfig{
	ShowProtected: true,
	ShowTimestamp: false,
	GroupByMethod: false,
}

// RegisterRoute registra una ruta en Gin y la logea
func RegisterRoute(group *gin.RouterGroup, method string, path string, handler gin.HandlerFunc, protected bool) {
	RegisterRouteWithConfig(group, method, path, handler, protected, defaultRouteConfig)
}

// RegisterRouteWithConfig registra una ruta con configuración personalizada
func RegisterRouteWithConfig(group *gin.RouterGroup, method, path string, handler gin.HandlerFunc, protected bool, cfg RouteConfig) {
	method = strings.ToUpper(method)
	group.Handle(method, path, handler)

	// Loguear la ruta
	logRoute(group.BasePath()+path, method, protected, cfg)
}

// RegisterRoutes registra múltiples rutas a la vez
func RegisterRoutes(group *gin.RouterGroup, routes []Route) {
	for _, route := range routes {
		RegisterRoute(group, route.Method, route.Path, route.Handler, route.Protected)
	}
}

// Route representa una ruta a registrar
type Route struct {
	Method    string
	Path      string
	Handler   gin.HandlerFunc
	Protected bool
}

// ============================================================================
// MIDDLEWARE DE LOGGING
// ============================================================================

// GinMiddleware crea un middleware para loguear requests
func GinMiddleware(logger *Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Procesar request
		c.Next()

		// Calcular latencia
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		if raw != "" {
			path = path + "?" + raw
		}

		// Determinar nivel de log basado en status code
		var logFunc func(string, ...interface{})
		switch {
		case statusCode >= 500:
			logFunc = logger.Error
		case statusCode >= 400:
			logFunc = logger.Warn
		default:
			logFunc = logger.Info
		}

		logFunc("[%s] %s %s %d - %v | IP: %s",
			method,
			path,
			c.Request.Proto,
			statusCode,
			latency,
			clientIP,
		)
	}
}

// GinMiddlewareWithFields middleware que agrega campos al logger
func GinMiddlewareWithFields(logger *Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Crear logger con contexto de request
		reqLogger := logger.WithFields(Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		})

		// Guardar en contexto para uso en handlers
		c.Set("logger", reqLogger)

		// Procesar request
		c.Next()

		// Log después del request
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		reqLogger.WithFields(Fields{
			"status":  statusCode,
			"latency": latency.String(),
		}).Info("Request completed")
	}
}

// GetLogger extrae el logger del contexto de Gin
func GetLogger(c *gin.Context) *Logger {
	if logger, exists := c.Get("logger"); exists {
		if l, ok := logger.(*Logger); ok {
			return l
		}
	}
	return getStd()
}

// ============================================================================
// FUNCIONES PRIVADAS
// ============================================================================

func logRoute(fullPath, method string, protected bool, cfg RouteConfig) {
	icon := "→"
	if protected && cfg.ShowProtected {
		icon = "🔒"
	}

	methodColor := getMethodColor(method)
	timestamp := ""
	if cfg.ShowTimestamp {
		timestamp = time.Now().Format("15:04:05") + " "
	}

	fmt.Printf("%s%s %s %-16s %s\n",
		timestamp,
		color.New(color.FgCyan).Sprint("[ROUTE]"),
		icon,
		methodColor.Sprintf("[%s]", method),
		color.New(color.FgWhite).Sprint(fullPath),
	)
}

func getMethodColor(method string) *color.Color {
	switch method {
	case "GET":
		return color.New(color.FgBlue, color.Bold)
	case "POST":
		return color.New(color.FgGreen, color.Bold)
	case "PUT":
		return color.New(color.FgYellow, color.Bold)
	case "PATCH":
		return color.New(color.FgCyan, color.Bold)
	case "DELETE":
		return color.New(color.FgRed, color.Bold)
	default:
		return color.New(color.FgWhite, color.Bold)
	}
}

// ============================================================================
// HELPERS PARA LOGGING EN HANDLERS
// ============================================================================

// LogRequest helper para loguear en handlers de Gin
func LogRequest(c *gin.Context, level Level, format string, args ...interface{}) {
	logger := GetLogger(c)
	logger.log(level, format, args...)
}

// LogRequestInfo loguea info desde un handler
func LogRequestInfo(c *gin.Context, format string, args ...interface{}) {
	LogRequest(c, InfoLevel, format, args...)
}

// LogRequestError loguea error desde un handler
func LogRequestError(c *gin.Context, format string, args ...interface{}) {
	LogRequest(c, ErrorLevel, format, args...)
}

// LogRequestWarn loguea warning desde un handler
func LogRequestWarn(c *gin.Context, format string, args ...interface{}) {
	LogRequest(c, WarnLevel, format, args...)
}

// ============================================================================
// RECOVERY MIDDLEWARE
// ============================================================================

// GinRecoveryWithLogger middleware de recovery que usa nuestro logger
func GinRecoveryWithLogger(logger *Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.WithFields(Fields{
					"error":  err,
					"method": c.Request.Method,
					"path":   c.Request.URL.Path,
					"ip":     c.ClientIP(),
				}).Error("Panic recovered: %v", err)

				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}

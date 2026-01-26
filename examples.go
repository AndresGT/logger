// examples/basic_usage.go
package main

import (
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"yourproject/logger"
)

func main() {
	// ============================================================================
	// EJEMPLO 1: Uso básico (compatible con código existente)
	// ============================================================================
	
	logger.Info("Servidor iniciando...")
	logger.Success("Conexión a DB exitosa")
	logger.Warn("Cache no disponible, usando modo sin cache")
	logger.Error("Error conectando a Redis: %v", "connection timeout")
	
	// Funciones legacy siguen funcionando
	logger.LogInfo("Esto también funciona")
	logger.LogSuccess("Backward compatible!")

	// ============================================================================
	// EJEMPLO 2: Logger con contexto
	// ============================================================================
	
	userID := uuid.New()
	userLogger := logger.WithUser(userID)
	
	userLogger.Info("Usuario autenticado")
	userLogger.Success("Perfil actualizado")
	
	// Agregar más contexto
	requestLogger := userLogger.WithRequest("/api/users/profile", "192.168.1.1")
	requestLogger.Info("Procesando request")

	// ============================================================================
	// EJEMPLO 3: Logger personalizado con múltiples destinos
	// ============================================================================
	
	// Conectar a base de datos
	db, _ := gorm.Open(postgres.Open("postgres://..."), &gorm.Config{})
	
	// Crear writer de DB
	dbWriter, _ := logger.NewDBWriter(db, 1000)
	
	// Crear writer de archivo
	fileWriter, _ := logger.NewFileWriter("app.log")
	
	// Crear logger con múltiples destinos
	customLogger := logger.New(logger.Config{
		MinLevel:     logger.DebugLevel,
		EnableColors: true,
		Writers: []logger.Writer{
			logger.NewConsoleWriter(true),
			dbWriter,
			fileWriter,
		},
	})
	
	customLogger.Info("Este log va a: consola + DB + archivo")
	customLogger.Debug("Los logs debug ahora se muestran")
	
	// Establecer como logger global
	logger.SetDefault(customLogger)
	
	// Ahora todas las funciones globales usan el customLogger
	logger.Info("Usando el nuevo logger global")
	
	// ============================================================================
	// EJEMPLO 4: Logging estructurado
	// ============================================================================
	
	logger.WithFields(logger.Fields{
		"order_id":    "ORD-12345",
		"customer_id": "CUST-789",
		"amount":      99.99,
		"currency":    "USD",
	}).Info("Orden procesada exitosamente")
	
	// ============================================================================
	// EJEMPLO 5: Niveles de log
	// ============================================================================
	
	// Solo mostrar warnings y errores en producción
	logger.SetDefault(logger.New(logger.Config{
		MinLevel: logger.WarnLevel,
	}))
	
	logger.Debug("No se mostrará")
	logger.Info("No se mostrará")
	logger.Warn("Esto SÍ se mostrará")
	logger.Error("Esto también")

	// ============================================================================
	// EJEMPLO 6: Limpieza al cerrar
	// ============================================================================
	
	defer customLogger.Close() // Asegura que todos los logs se escriban
}

// ============================================================================
// EJEMPLO 7: Integración con Gin
// ============================================================================

func ginExample() {
	router := gin.New()
	
	// Crear logger personalizado
	appLogger := logger.New(logger.Config{
		MinLevel:     logger.InfoLevel,
		EnableColors: true,
	})
	
	// Agregar middleware de logging
	router.Use(logger.GinMiddlewareWithFields(appLogger))
	router.Use(logger.GinRecoveryWithLogger(appLogger))
	
	// Registrar rutas con logging
	api := router.Group("/api")
	
	logger.RegisterRoute(api, "GET", "/users", getUsers, false)
	logger.RegisterRoute(api, "POST", "/users", createUser, true)
	logger.RegisterRoute(api, "PUT", "/users/:id", updateUser, true)
	logger.RegisterRoute(api, "DELETE", "/users/:id", deleteUser, true)
	
	// O registrar múltiples rutas a la vez
	logger.RegisterRoutes(api, []logger.Route{
		{Method: "GET", Path: "/products", Handler: getProducts, Protected: false},
		{Method: "POST", Path: "/products", Handler: createProduct, Protected: true},
	})
	
	router.Run(":8080")
}

// Handlers de ejemplo
func getUsers(c *gin.Context) {
	// Usar logger del contexto
	logger.LogRequestInfo(c, "Obteniendo lista de usuarios")
	
	// O extraer el logger completo
	reqLogger := logger.GetLogger(c)
	reqLogger.WithFields(logger.Fields{
		"limit":  10,
		"offset": 0,
	}).Info("Query ejecutado")
	
	c.JSON(200, gin.H{"users": []string{}})
}

func createUser(c *gin.Context) {
	logger.LogRequestInfo(c, "Creando nuevo usuario")
	c.JSON(201, gin.H{"message": "created"})
}

func updateUser(c *gin.Context) {
	logger.LogRequestInfo(c, "Actualizando usuario: %s", c.Param("id"))
	c.JSON(200, gin.H{"message": "updated"})
}

func deleteUser(c *gin.Context) {
	logger.LogRequestWarn(c, "Eliminando usuario: %s", c.Param("id"))
	c.JSON(200, gin.H{"message": "deleted"})
}

func getProducts(c *gin.Context) {
	c.JSON(200, gin.H{"products": []string{}})
}

func createProduct(c *gin.Context) {
	c.JSON(201, gin.H{"message": "created"})
}

// ============================================================================
// EJEMPLO 8: Logging de seguridad
// ============================================================================

func securityExample() {
	userID := uuid.New()
	
	// Log de evento de seguridad
	logger.WithFields(logger.Fields{
		"user_id":  userID,
		"ip":       "192.168.1.100",
		"endpoint": "/api/login",
		"action":   "login_attempt",
		"success":  false,
		"reason":   "invalid_password",
	}).Security("Intento de login fallido")
	
	// Login exitoso
	logger.WithFields(logger.Fields{
		"user_id":  userID,
		"ip":       "192.168.1.100",
		"endpoint": "/api/login",
		"action":   "login",
		"success":  true,
	}).Security("Usuario autenticado exitosamente")
	
	// Acceso a recurso protegido
	logger.WithFields(logger.Fields{
		"user_id":  userID,
		"resource": "/api/admin/users",
		"action":   "access_denied",
	}).Security("Acceso denegado a recurso protegido")
}

// ============================================================================
// EJEMPLO 9: Migrando código existente
// ============================================================================

// ANTES (tu código actual):
func oldCode() {
	// logger.LogInfo("Iniciando proceso")
	// logger.LogSuccess("Completado")
	// logger.LogError("Error: %v", err)
}

// DESPUÉS (sin cambios necesarios - backward compatible):
func newCode() {
	logger.LogInfo("Iniciando proceso")    // Funciona igual
	logger.LogSuccess("Completado")         // Funciona igual
	logger.LogError("Error: %v", "algún error") // Funciona igual
}

// O puedes migrar gradualmente a la nueva API:
func modernCode() {
	logger.Info("Iniciando proceso")        // Más corto
	logger.Success("Completado")            // Más corto
	logger.Error("Error: %v", "algún error") // Más corto
	
	// Con contexto adicional
	logger.WithFields(logger.Fields{
		"process_id": "PROC-123",
		"duration":   "5.2s",
	}).Success("Proceso completado")
}

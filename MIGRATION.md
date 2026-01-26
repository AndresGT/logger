# 🔄 Guía de Migración - Paso a Paso

Esta guía te ayudará a migrar tu logger actual al nuevo sistema mejorado de forma gradual y segura.

## ✅ Compatibilidad Total

**Buena noticia**: Tu código actual funcionará sin cambios. El nuevo logger es 100% backward compatible.

## 📋 Plan de Migración (3 Fases)

### Fase 1: Instalación y Setup (5 minutos)

#### 1.1 Reemplazar archivos

```bash
# Hacer backup de tu logger actual
cp logger/logger.go logger/logger.go.backup

# Copiar nuevos archivos
cp new_logger/logger.go logger/logger.go
cp new_logger/gin.go logger/gin.go
```

#### 1.2 Verificar imports

Tu código debería seguir compilando sin cambios. Todas estas funciones siguen funcionando:

```go
logger.LogInfo("mensaje")
logger.LogSuccess("éxito")
logger.LogWarn("advertencia")
logger.LogError("error")
logger.LogFatal("fatal")
logger.DisableLogs()
logger.EnableConsoleLogs()
```

#### 1.3 Ejecutar tests

```bash
go test ./...
```

Si todo pasa, ¡listo! Ya estás usando el nuevo logger.

### Fase 2: Configuración Básica (10 minutos)

#### 2.1 Agregar logging a archivos

En tu `main.go`:

```go
func main() {
    // Crear writer de archivos
    fileWriter, err := logger.NewFileWriter("logs/app.log")
    if err != nil {
        log.Fatal(err)
    }
    
    // Configurar logger
    appLogger := logger.New(logger.Config{
        MinLevel:     logger.InfoLevel,
        EnableColors: true,
        Writers: []logger.Writer{
            logger.NewConsoleWriter(true),
            fileWriter,
        },
    })
    
    // Establecer como logger global
    logger.SetDefault(appLogger)
    defer appLogger.Close()
    
    // Tu código sigue igual
    logger.LogInfo("Servidor iniciando...")
    
    // ... resto de tu aplicación
}
```

#### 2.2 Agregar logging a base de datos

```go
func main() {
    // Tu conexión DB existente
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }
    
    // Crear DB writer
    dbWriter, err := logger.NewDBWriter(db, 1000)
    if err != nil {
        log.Fatal(err)
    }
    
    // Configurar logger con DB
    appLogger := logger.New(logger.Config{
        MinLevel:     logger.InfoLevel,
        EnableColors: true,
        Writers: []logger.Writer{
            logger.NewConsoleWriter(true),
            dbWriter,
        },
    })
    
    logger.SetDefault(appLogger)
    defer appLogger.Close()
    
    // Ahora todos tus logs también van a la DB
    logger.LogInfo("Servidor iniciando...")
}
```

#### 2.3 Configurar por entorno

```go
func setupLogger(env string) *logger.Logger {
    var writers []logger.Writer
    var minLevel logger.Level
    
    switch env {
    case "production":
        minLevel = logger.WarnLevel
        fileWriter, _ := logger.NewFileWriter("logs/app.log")
        dbWriter, _ := logger.NewDBWriter(db, 1000)
        writers = []logger.Writer{
            logger.NewConsoleWriter(false), // sin colores en prod
            fileWriter,
            dbWriter,
        }
        
    case "development":
        minLevel = logger.DebugLevel
        writers = []logger.Writer{
            logger.NewConsoleWriter(true), // con colores
        }
        
    default:
        minLevel = logger.InfoLevel
        writers = []logger.Writer{
            logger.NewConsoleWriter(true),
        }
    }
    
    return logger.New(logger.Config{
        MinLevel:     minLevel,
        EnableColors: env != "production",
        Writers:      writers,
    })
}

func main() {
    env := os.Getenv("ENV")
    if env == "" {
        env = "development"
    }
    
    appLogger := setupLogger(env)
    logger.SetDefault(appLogger)
    defer appLogger.Close()
}
```

### Fase 3: Modernización Gradual (Opcional)

Una vez que tienes el nuevo logger funcionando, puedes migrar gradualmente a la nueva API.

#### 3.1 Actualizar llamadas simples

```go
// ANTES
logger.LogInfo("Usuario autenticado")
logger.LogSuccess("Orden procesada")
logger.LogError("Error: %v", err)

// DESPUÉS (más corto y moderno)
logger.Info("Usuario autenticado")
logger.Success("Orden procesada")
logger.Error("Error: %v", err)
```

#### 3.2 Agregar contexto a logs existentes

```go
// ANTES
logger.LogInfo("Usuario autenticado: %s", userID)
logger.LogInfo("Orden procesada: %s para usuario: %s", orderID, userID)

// DESPUÉS (más estructurado)
logger.WithUser(userID).Info("Usuario autenticado")

logger.WithFields(logger.Fields{
    "order_id": orderID,
    "user_id":  userID,
    "amount":   99.99,
}).Info("Orden procesada")
```

#### 3.3 Mejorar logging en handlers de Gin

```go
// ANTES
func GetUser(c *gin.Context) {
    userID := c.Param("id")
    logger.LogInfo("Obteniendo usuario: %s", userID)
    
    // ... tu código
    
    logger.LogSuccess("Usuario encontrado")
}

// DESPUÉS
func GetUser(c *gin.Context) {
    userID := c.Param("id")
    
    // Opción 1: Usar helpers
    logger.LogRequestInfo(c, "Obteniendo usuario: %s", userID)
    
    // Opción 2: Logger completo con contexto
    reqLogger := logger.GetLogger(c)
    reqLogger.WithFields(logger.Fields{
        "user_id": userID,
    }).Info("Obteniendo usuario")
    
    // ... tu código
    
    reqLogger.Success("Usuario encontrado")
}
```

#### 3.4 Agregar middleware de Gin

```go
// En tu configuración de rutas
func SetupRouter() *gin.Engine {
    router := gin.New()
    
    // Agregar middleware de logging
    appLogger := logger.NewDefault()
    router.Use(logger.GinMiddlewareWithFields(appLogger))
    router.Use(logger.GinRecoveryWithLogger(appLogger))
    
    // Tus rutas
    api := router.Group("/api")
    
    // Puedes seguir usando RegisterRoute igual
    logger.RegisterRoute(api, "GET", "/users", GetUsers, false)
    logger.RegisterRoute(api, "POST", "/users", CreateUser, true)
    
    return router
}
```

#### 3.5 Agregar logging de seguridad

```go
// En tu lógica de autenticación
func Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        logger.WithFields(logger.Fields{
            "ip":       c.ClientIP(),
            "endpoint": c.Request.URL.Path,
            "error":    err.Error(),
        }).Security("Intento de login con datos inválidos")
        
        c.JSON(400, gin.H{"error": "invalid request"})
        return
    }
    
    user, err := authenticate(req.Email, req.Password)
    if err != nil {
        logger.WithFields(logger.Fields{
            "email":    req.Email,
            "ip":       c.ClientIP(),
            "endpoint": c.Request.URL.Path,
        }).Security("Intento de login fallido")
        
        c.JSON(401, gin.H{"error": "invalid credentials"})
        return
    }
    
    logger.WithFields(logger.Fields{
        "user_id":  user.ID,
        "email":    user.Email,
        "ip":       c.ClientIP(),
        "endpoint": c.Request.URL.Path,
    }).Security("Login exitoso")
    
    c.JSON(200, user)
}
```

## 🎯 Checklist de Migración

### Setup Inicial
- [ ] Backup del logger actual
- [ ] Copiar nuevos archivos
- [ ] Verificar que compila
- [ ] Ejecutar tests

### Configuración Básica
- [ ] Configurar logger en main.go
- [ ] Agregar file writer (opcional)
- [ ] Agregar DB writer (opcional)
- [ ] Configurar por entorno
- [ ] Agregar defer logger.Close()

### Mejoras Gradual (Opcional)
- [ ] Actualizar llamadas a nueva API (LogInfo → Info)
- [ ] Agregar WithFields donde tenga sentido
- [ ] Implementar middleware de Gin
- [ ] Agregar logging de seguridad
- [ ] Refactorizar handlers con contexto

### Testing
- [ ] Tests unitarios pasan
- [ ] Tests de integración pasan
- [ ] Verificar logs en consola
- [ ] Verificar logs en archivo (si aplica)
- [ ] Verificar logs en DB (si aplica)

## 🔍 Solución de Problemas Comunes

### Problema: Logs no aparecen

**Solución**: Verificar nivel mínimo

```go
logger.SetDefault(logger.New(logger.Config{
    MinLevel: logger.DebugLevel, // Mostrar todo durante migración
}))
```

### Problema: Duplicación de logs

**Causa**: Llamando `logger.SetDefault()` múltiples veces

**Solución**: Llamar solo una vez en main.go

```go
func main() {
    appLogger := setupLogger()
    logger.SetDefault(appLogger) // Solo una vez
    defer appLogger.Close()
}
```

### Problema: Panic al cerrar

**Causa**: Cerrando logger que ya está cerrado

**Solución**: Usar defer correctamente

```go
func main() {
    appLogger := setupLogger()
    logger.SetDefault(appLogger)
    defer appLogger.Close() // Solo cerrar una vez
    
    // NO llamar appLogger.Close() manualmente después
}
```

### Problema: Colores no se ven en producción

**Solución**: Deshabilitar colores en producción

```go
if env == "production" {
    writers = []logger.Writer{
        logger.NewConsoleWriter(false), // sin colores
    }
}
```

### Problema: Logs de DB no se guardan

**Solución**: Asegurar que el logger se cierra

```go
defer appLogger.Close() // Esto hace flush de los logs pendientes
```

## 📊 Testing Post-Migración

### Test Manual

```bash
# 1. Verificar logs en consola
go run main.go

# 2. Verificar logs en archivo
tail -f logs/app.log

# 3. Verificar logs en DB
psql -d mydb -c "SELECT * FROM private.app_logs ORDER BY created_at DESC LIMIT 10;"
```

### Test Automatizado

```go
func TestLoggingIntegration(t *testing.T) {
    mock := logger.NewMockWriter()
    testLogger := logger.New(logger.Config{
        Writers: []logger.Writer{mock},
    })
    logger.SetDefault(testLogger)
    
    // Tu código que hace logging
    MyFunction()
    
    // Verificar que se logueó
    if mock.Count() == 0 {
        t.Error("Expected logs but got none")
    }
}
```

## 📈 Siguientes Pasos

1. **Revisar dashboard**: Analiza tus logs en DB para insights
2. **Alertas**: Configura alertas en logs de ERROR y SECURITY
3. **Métricas**: Exporta métricas de logs (conteos por nivel)
4. **Rotación**: Implementa rotación de archivos de log
5. **Monitoreo**: Integra con herramientas como Grafana/Prometheus

## 💡 Tips

- Migra primero en desarrollo, luego staging, luego producción
- Mantén tu logger antiguo por 1-2 semanas como backup
- Usa logging estructurado (WithFields) para mejor búsqueda
- Monitorea el impacto en performance después de migrar
- Documenta tu configuración de logging en README del proyecto

## 🆘 Ayuda

Si tienes problemas:

1. Revisa los ejemplos en `examples.go`
2. Lee el README completo
3. Chequea los tests en `logger_test.go`
4. Verifica la configuración de tu entorno
5. Asegura que todas las dependencias están instaladas

---

¡Feliz migración! 🎉

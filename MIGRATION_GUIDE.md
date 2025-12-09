# Guía de Migración a API REST Dedicada

Esta guía describe los pasos para migrar de la aplicación web actual (con vistas HTML) a una API REST pura y dedicada.

## 📋 Tabla de Contenidos

1. [Visión General](#visión-general)
2. [Arquitectura Propuesta](#arquitectura-propuesta)
3. [Plan de Migración](#plan-de-migración)
4. [Cambios en Endpoints](#cambios-en-endpoints)
5. [Autenticación y Seguridad](#autenticación-y-seguridad)
6. [Versionado de API](#versionado-de-api)
7. [CORS y Headers](#cors-y-headers)
8. [Documentación y Testing](#documentación-y-testing)

---

## Visión General

### Estado Actual
- Aplicación monolítica con Gin
- Renderizado de HTML en servidor
- Endpoints mixtos (HTML + JSON)
- Sin autenticación
- Sin versionado

### Estado Objetivo
- API REST pura (solo JSON)
- Frontend separado (React, Vue, Angular, etc.)
- Autenticación JWT
- Versionado de API (v1)
- CORS configurado
- Documentación interactiva (Swagger UI)
- Rate limiting
- Logging y monitoreo

---

## Arquitectura Propuesta

```
┌─────────────────────────────────────────────────────────┐
│                    Frontend (SPA)                       │
│              React / Vue / Angular / Svelte             │
│                  http://localhost:3000                  │
└─────────────────────────────────────────────────────────┘
                            │
                            │ HTTP/HTTPS + JWT
                            ▼
┌─────────────────────────────────────────────────────────┐
│                    API Gateway (Opcional)               │
│                   Rate Limiting / CORS                  │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                    API REST Backend                     │
│                  Gin Framework (Go)                     │
│                  http://localhost:8081                  │
│                                                         │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Auth      │  │  Business    │  │   Database   │  │
│  │ Middleware  │→ │    Logic     │→ │    Layer     │  │
│  └─────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                    PostgreSQL Database                  │
└─────────────────────────────────────────────────────────┘
```

---

## Plan de Migración

### Fase 1: Preparación (Semana 1-2)

#### 1.1 Crear estructura de proyecto separada
```bash
bolsa_gin/
├── backend/           # API REST
│   ├── cmd/
│   │   └── api/
│   │       └── main.go
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handlers/
│   │   │   ├── middleware/
│   │   │   └── routes/
│   │   ├── models/
│   │   ├── repository/
│   │   └── service/
│   ├── pkg/
│   │   ├── auth/
│   │   └── utils/
│   ├── config/
│   ├── migrations/
│   └── docs/
├── frontend/          # SPA (React/Vue/etc)
│   ├── src/
│   ├── public/
│   └── package.json
└── docs/
    ├── openapi.yaml
    └── API_README.md
```

#### 1.2 Configurar variables de entorno
```env
# .env.development
API_VERSION=v1
API_PORT=8081
DB_HOST=localhost
DB_PORT=5432
DB_NAME=investments
DB_USER=postgres
DB_PASSWORD=password
JWT_SECRET=your-secret-key
JWT_EXPIRATION=24h
CORS_ALLOWED_ORIGINS=http://localhost:3000
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_DURATION=1m
LOG_LEVEL=debug
```

### Fase 2: Refactorización del Backend (Semana 3-4)

#### 2.1 Separar lógica de negocio

**Antes (main.go monolítico)**:
```go
router.POST("/add-investment", func(c *gin.Context) {
    // Parsear formulario
    // Validar
    // Guardar en DB
    // Redirigir
})
```

**Después (arquitectura en capas)**:

```go
// internal/models/investment.go
type Investment struct {
    gorm.Model
    TickerID      uint      `json:"ticker_id"`
    PurchaseDate  time.Time `json:"purchase_date"`
    Shares        float64   `json:"shares"`
    PurchasePrice float64   `json:"purchase_price"`
    OperationCost float64   `json:"operation_cost"`
}

// internal/repository/investment_repository.go
type InvestmentRepository interface {
    Create(investment *Investment) error
    GetByID(id uint) (*Investment, error)
    Update(investment *Investment) error
    Delete(id uint) error
    List(filters map[string]interface{}) ([]*Investment, error)
}

// internal/service/investment_service.go
type InvestmentService struct {
    repo InvestmentRepository
}

func (s *InvestmentService) CreateInvestment(req CreateInvestmentRequest) (*Investment, error) {
    // Validación de negocio
    // Crear inversión
    // Retornar resultado
}

// internal/api/handlers/investment_handler.go
type InvestmentHandler struct {
    service *InvestmentService
}

func (h *InvestmentHandler) Create(c *gin.Context) {
    var req CreateInvestmentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
        return
    }
    
    investment, err := h.service.CreateInvestment(req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, investment)
}
```

#### 2.2 Implementar middleware de autenticación

```go
// pkg/auth/jwt.go
package auth

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID uint   `json:"user_id"`
    Email  string `json:"email"`
    jwt.RegisteredClaims
}

func GenerateToken(userID uint, email string, secret string) (string, error) {
    claims := Claims{
        UserID: userID,
        Email:  email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}

func ValidateToken(tokenString string, secret string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(secret), nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, jwt.ErrSignatureInvalid
}

// internal/api/middleware/auth.go
package middleware

func AuthMiddleware(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }
        
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := auth.ValidateToken(tokenString, secret)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }
        
        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Next()
    }
}
```

#### 2.3 Implementar CORS

```go
// internal/api/middleware/cors.go
package middleware

import (
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
    "time"
)

func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
    config := cors.Config{
        AllowOrigins:     allowedOrigins,
        AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }
    
    return cors.New(config)
}
```

#### 2.4 Implementar Rate Limiting

```go
// internal/api/middleware/rate_limit.go
package middleware

import (
    "github.com/gin-gonic/gin"
    "golang.org/x/time/rate"
    "sync"
)

type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rate     rate.Limit
    burst    int
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
    return &RateLimiter{
        limiters: make(map[string]*rate.Limiter),
        rate:     r,
        burst:    b,
    }
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    limiter, exists := rl.limiters[key]
    if !exists {
        limiter = rate.NewLimiter(rl.rate, rl.burst)
        rl.limiters[key] = limiter
    }
    
    return limiter
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Usar IP del cliente como clave
        key := c.ClientIP()
        limiter := rl.getLimiter(key)
        
        if !limiter.Allow() {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "Rate limit exceeded",
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

### Fase 3: Nuevos Endpoints REST (Semana 5-6)

#### 3.1 Estructura de rutas versionadas

```go
// internal/api/routes/routes.go
package routes

func SetupRoutes(router *gin.Engine, handlers *Handlers, middlewares *Middlewares) {
    // Health check (sin autenticación)
    router.GET("/health", handlers.Health.Check)
    
    // API v1
    v1 := router.Group("/api/v1")
    {
        // Auth (sin autenticación)
        auth := v1.Group("/auth")
        {
            auth.POST("/register", handlers.Auth.Register)
            auth.POST("/login", handlers.Auth.Login)
            auth.POST("/refresh", handlers.Auth.RefreshToken)
        }
        
        // Rutas protegidas
        protected := v1.Group("")
        protected.Use(middlewares.Auth)
        {
            // Tickers
            tickers := protected.Group("/tickers")
            {
                tickers.GET("", handlers.Ticker.List)
                tickers.POST("", handlers.Ticker.Create)
                tickers.GET("/:id", handlers.Ticker.GetByID)
                tickers.PUT("/:id", handlers.Ticker.Update)
                tickers.DELETE("/:id", handlers.Ticker.Delete)
            }
            
            // Investments
            investments := protected.Group("/investments")
            {
                investments.GET("", handlers.Investment.List)
                investments.POST("", handlers.Investment.Create)
                investments.GET("/:id", handlers.Investment.GetByID)
                investments.PUT("/:id", handlers.Investment.Update)
                investments.DELETE("/:id", handlers.Investment.Delete)
            }
            
            // Sales
            sales := protected.Group("/sales")
            {
                sales.GET("", handlers.Sale.List)
                sales.POST("", handlers.Sale.Create)
                sales.GET("/:id", handlers.Sale.GetByID)
                sales.PUT("/:id", handlers.Sale.Update)
                sales.DELETE("/:id", handlers.Sale.Delete)
                sales.GET("/:id/calculation", handlers.Sale.GetCalculation)
            }
            
            // Snapshots
            snapshots := protected.Group("/snapshots")
            {
                snapshots.GET("", handlers.Snapshot.List)
                snapshots.POST("", handlers.Snapshot.Create)
                snapshots.GET("/:id", handlers.Snapshot.GetByID)
                snapshots.DELETE("/:id", handlers.Snapshot.Delete)
            }
            
            // Analytics
            analytics := protected.Group("/analytics")
            {
                analytics.GET("/portfolio-utility-history", handlers.Analytics.PortfolioUtilityHistory)
                analytics.GET("/portfolio-summary", handlers.Analytics.PortfolioSummary)
                analytics.GET("/ticker-summary", handlers.Analytics.TickerSummary)
            }
        }
    }
    
    // Swagger docs
    router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
```

#### 3.2 Respuestas estandarizadas

```go
// internal/api/responses/responses.go
package responses

type SuccessResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data"`
    Message string      `json:"message,omitempty"`
}

type ErrorResponse struct {
    Success bool   `json:"success"`
    Error   string `json:"error"`
    Code    string `json:"code,omitempty"`
}

type PaginatedResponse struct {
    Success    bool        `json:"success"`
    Data       interface{} `json:"data"`
    Pagination Pagination  `json:"pagination"`
}

type Pagination struct {
    Page       int   `json:"page"`
    PerPage    int   `json:"per_page"`
    Total      int64 `json:"total"`
    TotalPages int   `json:"total_pages"`
}

func Success(data interface{}, message string) SuccessResponse {
    return SuccessResponse{
        Success: true,
        Data:    data,
        Message: message,
    }
}

func Error(err string, code string) ErrorResponse {
    return ErrorResponse{
        Success: false,
        Error:   err,
        Code:    code,
    }
}
```

### Fase 4: Frontend (Semana 7-10)

#### 4.1 Crear proyecto React (ejemplo)

```bash
npx create-react-app frontend
cd frontend
npm install axios react-router-dom @tanstack/react-query
```

#### 4.2 Configurar cliente API

```javascript
// src/api/client.js
import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8081/api/v1';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Interceptor para agregar token
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Interceptor para manejar errores
apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      // Token expirado, redirigir a login
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default apiClient;
```

#### 4.3 Servicios de API

```javascript
// src/api/services/investmentService.js
import apiClient from '../client';

export const investmentService = {
  getAll: async (params) => {
    const { data } = await apiClient.get('/investments', { params });
    return data;
  },
  
  getById: async (id) => {
    const { data } = await apiClient.get(`/investments/${id}`);
    return data;
  },
  
  create: async (investment) => {
    const { data } = await apiClient.post('/investments', investment);
    return data;
  },
  
  update: async (id, investment) => {
    const { data } = await apiClient.put(`/investments/${id}`, investment);
    return data;
  },
  
  delete: async (id) => {
    const { data } = await apiClient.delete(`/investments/${id}`);
    return data;
  },
};
```

---

## Cambios en Endpoints

### Mapeo de Endpoints Antiguos a Nuevos

| Endpoint Actual | Método | Nuevo Endpoint | Método | Cambios |
|----------------|--------|----------------|--------|---------|
| `/add-ticker` | POST | `/api/v1/tickers` | POST | JSON en lugar de form-data |
| `/update-ticker/:id` | POST | `/api/v1/tickers/:id` | PUT | JSON, sin redirección |
| `/delete-ticker` | POST | `/api/v1/tickers/:id` | DELETE | ID en path |
| `/add-investment` | POST | `/api/v1/investments` | POST | JSON, sin redirección |
| `/update/:id` | POST | `/api/v1/investments/:id` | PUT | JSON, sin redirección |
| `/delete-investment` | POST | `/api/v1/investments/:id` | DELETE | ID en path |
| `/add-sale` | POST | `/api/v1/sales` | POST | JSON, sin redirección |
| `/update-sale/:id` | POST | `/api/v1/sales/:id` | PUT | JSON, sin redirección |
| `/delete-sale` | POST | `/api/v1/sales/:id` | DELETE | ID en path |
| `/create-snapshot` | POST | `/api/v1/snapshots` | POST | Sin cambios |
| `/delete-snapshot` | POST | `/api/v1/snapshots/:id` | DELETE | ID en path |
| `/sale-calculation/:id` | GET | `/api/v1/sales/:id/calculation` | GET | Ruta anidada |
| `/api/portfolio-utility-history` | GET | `/api/v1/analytics/portfolio-utility-history` | GET | Bajo /analytics |

---

## Autenticación y Seguridad

### Implementar sistema de usuarios

```go
// internal/models/user.go
type User struct {
    gorm.Model
    Email        string `gorm:"uniqueIndex" json:"email"`
    PasswordHash string `json:"-"`
    Name         string `json:"name"`
}

// internal/api/handlers/auth_handler.go
type RegisterRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
    Name     string `json:"name" binding:"required"`
}

type LoginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
    Token string `json:"token"`
    User  User   `json:"user"`
}

func (h *AuthHandler) Register(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, responses.Error(err.Error(), "VALIDATION_ERROR"))
        return
    }
    
    user, token, err := h.service.Register(req)
    if err != nil {
        c.JSON(http.StatusBadRequest, responses.Error(err.Error(), "REGISTRATION_FAILED"))
        return
    }
    
    c.JSON(http.StatusCreated, responses.Success(AuthResponse{
        Token: token,
        User:  *user,
    }, "User registered successfully"))
}

func (h *AuthHandler) Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, responses.Error(err.Error(), "VALIDATION_ERROR"))
        return
    }
    
    user, token, err := h.service.Login(req)
    if err != nil {
        c.JSON(http.StatusUnauthorized, responses.Error("Invalid credentials", "AUTH_FAILED"))
        return
    }
    
    c.JSON(http.StatusOK, responses.Success(AuthResponse{
        Token: token,
        User:  *user,
    }, "Login successful"))
}
```

---

## Versionado de API

### Estrategia de versionado

1. **URL Path Versioning** (Recomendado)
   - `/api/v1/tickers`
   - `/api/v2/tickers`

2. **Header Versioning** (Alternativa)
   - Header: `Accept: application/vnd.bolsagin.v1+json`

### Manejo de versiones múltiples

```go
// Mantener v1 mientras se desarrolla v2
v1 := router.Group("/api/v1")
setupV1Routes(v1)

v2 := router.Group("/api/v2")
setupV2Routes(v2)
```

---

## CORS y Headers

### Configuración de producción

```go
config := cors.Config{
    AllowOrigins: []string{
        "https://bolsagin.com",
        "https://app.bolsagin.com",
    },
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}
```

---

## Documentación y Testing

### Generar documentación Swagger

```bash
# Instalar swag
go install github.com/swaggo/swag/cmd/swag@latest

# Generar docs
swag init -g cmd/api/main.go -o docs
```

### Ejemplo de anotaciones Swagger

```go
// @Summary Create investment
// @Description Create a new investment
// @Tags investments
// @Accept json
// @Produce json
// @Param investment body CreateInvestmentRequest true "Investment data"
// @Success 201 {object} responses.SuccessResponse{data=Investment}
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/investments [post]
func (h *InvestmentHandler) Create(c *gin.Context) {
    // ...
}
```

---

## Checklist de Migración

- [ ] Configurar estructura de proyecto separada
- [ ] Implementar autenticación JWT
- [ ] Implementar middleware de CORS
- [ ] Implementar rate limiting
- [ ] Refactorizar a arquitectura en capas
- [ ] Crear endpoints REST versionados
- [ ] Estandarizar respuestas JSON
- [ ] Implementar validación de entrada
- [ ] Agregar logging estructurado
- [ ] Configurar Swagger/OpenAPI
- [ ] Crear frontend separado
- [ ] Implementar tests unitarios
- [ ] Implementar tests de integración
- [ ] Configurar CI/CD
- [ ] Documentar API
- [ ] Migrar datos si es necesario
- [ ] Configurar monitoreo y alertas
- [ ] Realizar pruebas de carga
- [ ] Preparar rollback plan
- [ ] Desplegar a producción

---

## Recursos Adicionales

- [Gin Framework](https://gin-gonic.com/)
- [GORM](https://gorm.io/)
- [JWT Go](https://github.com/golang-jwt/jwt)
- [Swagger](https://swagger.io/)
- [React Query](https://tanstack.com/query)
- [Go Project Layout](https://github.com/golang-standards/project-layout)

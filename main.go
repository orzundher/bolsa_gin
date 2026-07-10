package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- MODELOS DE GORM ---

// Migration trackea las migraciones ejecutadas
type Migration struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"uniqueIndex"`
	AppliedAt time.Time
}

// TickerGroup representa un grupo de tickers (ej: Tech, Finance, etc.)
type TickerGroup struct {
	gorm.Model
	Name string `gorm:"uniqueIndex;not null"`
}

// Ticker representa un símbolo bursátil con su precio actual.
type Ticker struct {
	gorm.Model
	Name               string `gorm:"uniqueIndex"`
	CurrentPrice       float64
	YahooFinanceTicker string
	YahooEur           bool
	Active             bool        `gorm:"default:true"`
	GroupID            *uint       // Nullable foreign key
	Group              TickerGroup `gorm:"foreignKey:GroupID"`
}

// Investment representa una única compra de acciones en la BD.
type Investment struct {
	gorm.Model
	TickerID      uint
	Ticker        Ticker `gorm:"foreignKey:TickerID"`
	PurchaseDate  time.Time
	Shares        float64
	PurchasePrice float64
	OperationCost float64
}

// Sale representa una única venta de acciones en la BD.
type Sale struct {
	gorm.Model
	TickerID      uint
	Ticker        Ticker `gorm:"foreignKey:TickerID"`
	SaleDate      time.Time
	Shares        float64
	SalePrice     float64
	OperationCost float64
}

// PriceHistory representa un snapshot histórico de precio de un ticker.
type PriceHistory struct {
	gorm.Model
	SnapshotID string // UUID o timestamp para agrupar snapshots
	TickerID   uint
	Ticker     Ticker `gorm:"foreignKey:TickerID"`
	Price      float64
}

// Dividend representa un cobro de dividendo de un ticker.
type Dividend struct {
	gorm.Model
	TickerID uint
	Ticker   Ticker `gorm:"foreignKey:TickerID"`
	Date     time.Time
	Amount   float64 // Importe total recibido
	Notes    string
}

// DividendView representa los datos de dividendo para mostrar en la UI.
type DividendView struct {
	ID       uint
	TickerID uint
	Ticker   string
	Date     string
	Amount   float64
	Notes    string
}

// --- VISTAS ---

// TickerView representa los datos de un ticker para mostrar en la UI.
type TickerView struct {
	ID                      uint
	Name                    string
	CurrentPrice            float64
	UpdatedAt               string
	SnapshotChange          float64 // Cambio porcentual entre los últimos 2 snapshots
	HasSnapshotChange       bool    // Indica si hay datos suficientes para mostrar el cambio
	YahooFinanceTicker      string
	YahooEur                bool
	HasShares               bool
	GroupName               string
	GroupID                 uint
	Active                  bool
	CurrentToSnapshotPct    float64 // Cambio porcentual entre precio actual y último snapshot
	HasCurrentToSnapshotPct bool    // Indica si hay datos para mostrar el cambio actual vs snapshot
}

// InvestmentView representa los datos de inversión que se mostrarán en la página.
type InvestmentView struct {
	ID              uint
	TickerID        uint
	Ticker          string
	PurchaseDate    string
	Shares          float64
	PurchasePrice   float64
	OperationCost   float64
	InvestedCapital float64
	CurrentPrice    float64
	CurrentValue    float64
	ProfitLoss      float64
	Performance     float64
}

// TickerSummaryView representa un resumen de las inversiones por ticker.
type TickerSummaryView struct {
	TickerID          uint
	Ticker            string
	TotalShares       float64
	CurrentInvestment float64
	TotalCost         float64
	CurrentValue      float64
	ProfitLoss        float64
	Performance       float64
	SalesProfit       float64
	WeightPercentage  float64
	SnapshotChange    float64 // Cambio porcentual entre los últimos 2 snapshots
	HasSnapshotChange bool    // Indica si hay datos suficientes para mostrar el cambio
	GroupName         string
	Dividends         float64 // Total de dividendos cobrados para este ticker
}

// SaleView representa los datos de venta que se mostrarán en la página.
type SaleView struct {
	ID              uint
	TickerID        uint
	Ticker          string
	SaleDate        string
	Shares          float64
	SalePrice       float64
	OperationCost   float64
	TotalSaleValue  float64
	CurrentPrice    float64
	CurrentValue    float64
	Performance     float64
	Profit          float64
	Projection      float64
	WACAtSale       float64
	SalePerformance float64
	SaleUtility     float64
}

var db *gorm.DB
var preciosAPIKey string
var preciosBaseURL string

// Credenciales y configuración de autenticación
var authUser string
var authPassword string
var authSecret string

const (
	authSessionKey = "authenticated"
	authSessionUser = "user"
)

func loadAuthConfig() {
	authUser = os.Getenv("AUTH_USER")
	authPassword = os.Getenv("AUTH_PASSWORD")
	authSecret = os.Getenv("AUTH_SECRET")

	if authUser == "" {
		authUser = "admin"
		log.Println("AUTH_USER no configurado, usando valor por defecto: admin")
	}
	if authPassword == "" {
		authPassword = "admin"
		log.Println("AUTH_PASSWORD no configurado, usando valor por defecto: admin")
	}
	if authSecret == "" {
		authSecret = "change-me-in-production"
		log.Println("AUTH_SECRET no configurado, usando valor por defecto (inseguro)")
	}
}

func isPublicPath(path string) bool {
	return path == "/login" || path == "/logout" || strings.HasPrefix(path, "/static/")
}

func checkBasicAuth(c *gin.Context) bool {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}

	const prefix = "Basic "
	if !strings.HasPrefix(authHeader, prefix) {
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(authHeader[len(prefix):])
	if err != nil {
		return false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return false
	}

	return parts[0] == authUser && parts[1] == authPassword
}

func setAuthSession(c *gin.Context) {
	session := sessions.Default(c)
	session.Set(authSessionKey, true)
	session.Set(authSessionUser, authUser)
	if err := session.Save(); err != nil {
		log.Printf("Error guardando sesión: %v", err)
	}
}

func clearAuthSession(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete(authSessionKey)
	session.Delete(authSessionUser)
	if err := session.Save(); err != nil {
		log.Printf("Error limpiando sesión: %v", err)
	}
}

func isAuthenticated(c *gin.Context) bool {
	session := sessions.Default(c)
	auth := session.Get(authSessionKey)
	if auth == nil {
		return false
	}
	authenticated, ok := auth.(bool)
	return ok && authenticated
}

func redirectToLogin(c *gin.Context) {
	c.Redirect(http.StatusFound, "/login")
	c.Abort()
}

func unauthorizedAPI(c *gin.Context) {
	c.Header("WWW-Authenticate", `Basic realm="bolsa_gin"`)
	c.JSON(http.StatusUnauthorized, gin.H{"error": "Se requiere autenticación"})
	c.Abort()
}

func requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if isPublicPath(path) {
			c.Next()
			return
		}

		// Las rutas de API aceptan Basic Auth o sesión de cookie
		if strings.HasPrefix(path, "/api/") {
			if isAuthenticated(c) || checkBasicAuth(c) {
				c.Next()
				return
			}
			unauthorizedAPI(c)
			return
		}

		// El resto de rutas requiere sesión de cookie (redirige a login)
		if isAuthenticated(c) {
			c.Next()
			return
		}

		redirectToLogin(c)
	}
}

func handleLoginGet(c *gin.Context) {
	if isAuthenticated(c) {
		c.Redirect(http.StatusFound, "/")
		return
	}
	c.HTML(http.StatusOK, "login.html", gin.H{
		"Error": c.Query("error"),
	})
}

func handleLoginPost(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == authUser && password == authPassword {
		setAuthSession(c)
		c.Redirect(http.StatusFound, "/")
		return
	}

	c.HTML(http.StatusUnauthorized, "login.html", gin.H{
		"Error": "Usuario o contraseña incorrectos",
	})
}

func handleLogout(c *gin.Context) {
	clearAuthSession(c)
	c.Redirect(http.StatusFound, "/login")
}

func main() {

	// Cargar variables de entorno desde .env
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró archivo .env, usando variables de entorno del sistema")
	}

	loadAuthConfig()

	var err error

	// Configurar la base de datos con GORM
	db, err = setupDatabase()
	if err != nil {
		log.Fatalf("Error al configurar la base de datos: %v", err)
	}

	// Configurar Gin
	router := gin.Default()

	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./static")

	// Configurar sesiones
	store := cookie.NewStore([]byte(authSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 días
		HttpOnly: true,
		Secure:   false, // cambiar a true si se usa HTTPS
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("bolsa_gin_session", store))

	// Rutas públicas de autenticación
	router.GET("/login", handleLoginGet)
	router.POST("/login", handleLoginPost)
	router.GET("/logout", handleLogout)

	// Middleware de autenticación para todas las demás rutas
	router.Use(requireAuth())

	// Ruta principal para mostrar los datos
	router.GET("/", func(c *gin.Context) {
		investments, summaries, sales, totalCapital, netProfitLoss, totalOperationCost, _, portfolioPerformance, portfolioUtility, numPositions, totalCurrentValue, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		// Calcular utilidad neta de ventas
		totalSaleUtility := 0.0
		for _, s := range sales {
			totalSaleUtility += s.SaleUtility
		}

		// Calcular total de dividendos
		var dividends []Dividend
		db.Find(&dividends)
		totalDividends := 0.0
		for _, d := range dividends {
			totalDividends += d.Amount
		}

		// Calcular Valor de Salida: Utilidad Ventas + Utilidad Cartera + Dividendos - Costos de Operación - Número de Posiciones
		exitValue := totalSaleUtility + portfolioUtility + totalDividends - totalOperationCost - float64(numPositions)

		// Obtener notas
		var notes []Note
		db.Order("date desc").Find(&notes)

		c.HTML(http.StatusOK, "index.html", gin.H{
			"Investments":          investments,
			"Summaries":            summaries,
			"TotalCapital":         totalCapital,
			"NetProfitLoss":        netProfitLoss,
			"TotalOperationCost":   totalOperationCost,
			"TotalSaleUtility":     totalSaleUtility,
			"PortfolioPerformance": portfolioPerformance,
			"PortfolioUtility":     portfolioUtility,
			"TotalCurrentValue":    totalCurrentValue,
			"NumPositions":         numPositions,
			"ExitValue":            exitValue,
			"TotalDividends":       totalDividends,
			"Notes":                notes,
			"ActivePage":           "home",
		})
	})

	// Rutas para notas
	router.POST("/api/notes", func(c *gin.Context) {
		var input struct {
			Date    string `json:"date"`
			Content string `json:"content"`
		}

		if err := c.BindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		date, err := time.ParseInLocation("2006-01-02", input.Date, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido"})
			return
		}

		note := Note{Date: date, Content: input.Content}
		if err := db.Create(&note).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar la nota"})
			return
		}

		c.JSON(http.StatusOK, note)
	})

	router.PUT("/api/notes/:id", func(c *gin.Context) {
		id := c.Param("id")
		var input struct {
			Date    string `json:"date"`
			Content string `json:"content"`
		}

		if err := c.BindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var note Note
		if err := db.First(&note, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nota no encontrada"})
			return
		}

		date, err := time.ParseInLocation("2006-01-02", input.Date, time.Local)
		if err == nil {
			date = date.UTC()
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido"})
			return
		}

		note.Date = date
		note.Content = input.Content
		db.Save(&note)

		c.JSON(http.StatusOK, note)
	})

	router.DELETE("/api/notes/:id", func(c *gin.Context) {
		id := c.Param("id")
		if err := db.Delete(&Note{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar la nota"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	// Rutas para notas de tickers
	router.POST("/api/ticker-notes", func(c *gin.Context) {
		var input struct {
			TickerID uint   `json:"ticker_id"`
			Date     string `json:"date"`
			Content  string `json:"content"`
		}

		if err := c.BindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		date, err := time.ParseInLocation("2006-01-02", input.Date, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido"})
			return
		}

		note := TickerNote{TickerID: input.TickerID, Date: date, Content: input.Content}
		if err := db.Create(&note).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar la nota del ticker"})
			return
		}

		c.JSON(http.StatusOK, note)
	})

	router.PUT("/api/ticker-notes/:id", func(c *gin.Context) {
		id := c.Param("id")
		var input struct {
			Date    string `json:"date"`
			Content string `json:"content"`
		}

		if err := c.BindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var note TickerNote
		if err := db.First(&note, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nota del ticker no encontrada"})
			return
		}

		date, err := time.ParseInLocation("2006-01-02", input.Date, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido"})
			return
		}

		note.Date = date
		note.Content = input.Content
		db.Save(&note)

		c.JSON(http.StatusOK, note)
	})

	router.DELETE("/api/ticker-notes/:id", func(c *gin.Context) {
		id := c.Param("id")
		if err := db.Delete(&TickerNote{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar la nota del ticker"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	// Proxy: Obtener precio actual desde el servicio externo (evita CORS)
	router.GET("/api/fetch-price/:yahoo_ticker", func(c *gin.Context) {
		yahooTicker := c.Param("yahoo_ticker")
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/precio/%s", preciosBaseURL, yahooTicker), nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando la petición"})
			return
		}
		req.Header.Set("X-API-Key", preciosAPIKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "No se puede conectar con el servicio de precios"})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer la respuesta del servicio de precios"})
			return
		}

		c.Data(resp.StatusCode, "application/json", body)
	})

	// API: Obtener historial de utilidad de ventas (basado en eventos reales, no snapshots)
	router.GET("/api/sales-utility-history", func(c *gin.Context) {
		// Obtener todas las inversiones y ventas
		var allInvestments []Investment
		db.Order("purchase_date asc").Find(&allInvestments)

		var allSales []Sale
		db.Order("sale_date asc").Find(&allSales)

		if len(allSales) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"dates":     []string{},
				"utilities": []float64{},
			})
			return
		}

		// Combinar todos los eventos financieros en una lista cronológica
		type Event struct {
			Date   time.Time
			Type   string // "buy", "sell"
			Ticker uint
			Shares float64
			Price  float64
		}
		var allEvents []Event
		for _, inv := range allInvestments {
			allEvents = append(allEvents, Event{Date: inv.PurchaseDate, Type: "buy", Ticker: inv.TickerID, Shares: inv.Shares, Price: inv.PurchasePrice})
		}
		for _, sale := range allSales {
			allEvents = append(allEvents, Event{Date: sale.SaleDate, Type: "sell", Ticker: sale.TickerID, Shares: sale.Shares, Price: sale.SalePrice})
		}
		sort.Slice(allEvents, func(i, j int) bool {
			if allEvents[i].Date.Equal(allEvents[j].Date) {
				return allEvents[i].Type == "buy" // Compras antes que ventas si coinciden
			}
			return allEvents[i].Date.Before(allEvents[j].Date)
		})

		// Mapa para seguir el estado de cada ticker
		type TickerState struct {
			Shares  float64
			Capital float64
		}
		tickerStates := make(map[uint]TickerState)

		var dates []string
		var utilities []float64

		totalSalesUtility := 0.0

		// Agregar punto inicial en 0 justo antes del primer evento
		if len(allEvents) > 0 {
			dates = append(dates, allEvents[0].Date.Add(-1*time.Second).Format("02 Jan 2006 15:04:05"))
			utilities = append(utilities, 0.0)
		}

		for _, e := range allEvents {
			state := tickerStates[e.Ticker]

			if e.Type == "buy" {
				state.Shares += e.Shares
				state.Capital += e.Shares * e.Price
				tickerStates[e.Ticker] = state
			} else if e.Type == "sell" {
				wac := 0.0
				if state.Shares > 0 {
					wac = state.Capital / state.Shares
				}
				// Utilidad realizada: (Precio Venta - WAC) * Acciones Vendidas
				saleUtility := (e.Price - wac) * e.Shares
				totalSalesUtility += saleUtility

				// Reducir capital proporcionalmente
				state.Capital -= e.Shares * wac
				state.Shares -= e.Shares
				tickerStates[e.Ticker] = state

				// Registrar punto en el gráfico para cada venta (cambio de utilidad)
				dates = append(dates, e.Date.Format("02 Jan 2006 15:04:05"))
				utilities = append(utilities, totalSalesUtility)
			}
		}

		// Agregar punto final "AHORA" para extender la línea hasta el presente
		now := time.Now()
		if len(dates) > 0 {
			lastDateStr := dates[len(dates)-1]
			lastDate, _ := time.Parse("02 Jan 2006 15:04:05", lastDateStr) // Formato interno usado arriba

			if lastDate.Before(now) {
				dates = append(dates, now.Local().Format("02 Jan 2006 15:04:05"))
				utilities = append(utilities, totalSalesUtility)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"dates":     dates,
			"utilities": utilities,
		})
	})

	// Ruta para mostrar la página de resumen
	router.GET("/resumen", func(c *gin.Context) {
		_, summaries, _, _, _, _, _, _, _, _, _, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		// Obtener los dos últimos snapshots para calcular cambios
		type SnapshotInfo struct {
			SnapshotID string
			CreatedAt  time.Time
		}
		var snapshots []SnapshotInfo
		db.Model(&PriceHistory{}).
			Select("DISTINCT snapshot_id, MIN(created_at) as created_at").
			Group("snapshot_id").
			Order("created_at DESC").
			Limit(2).
			Scan(&snapshots)

		// Crear un mapa para almacenar los cambios porcentuales por ticker
		snapshotChanges := make(map[uint]*float64)

		// Si hay al menos 2 snapshots, calcular los cambios
		if len(snapshots) >= 2 {
			lastSnapshotID := snapshots[0].SnapshotID
			prevSnapshotID := snapshots[1].SnapshotID

			// Obtener precios del último snapshot
			var lastPrices []PriceHistory
			db.Where("snapshot_id = ?", lastSnapshotID).Find(&lastPrices)
			lastPriceMap := make(map[uint]float64)
			for _, p := range lastPrices {
				lastPriceMap[p.TickerID] = p.Price
			}

			// Obtener precios del snapshot anterior
			var prevPrices []PriceHistory
			db.Where("snapshot_id = ?", prevSnapshotID).Find(&prevPrices)
			prevPriceMap := make(map[uint]float64)
			for _, p := range prevPrices {
				prevPriceMap[p.TickerID] = p.Price
			}

			// Calcular cambios porcentuales
			for tickerID, lastPrice := range lastPriceMap {
				if prevPrice, exists := prevPriceMap[tickerID]; exists && prevPrice > 0 {
					change := ((lastPrice - prevPrice) / prevPrice) * 100
					snapshotChanges[tickerID] = &change
				}
			}
		}

		// Agregar información de cambios de snapshot a los summaries
		for i := range summaries {
			if changePtr, exists := snapshotChanges[summaries[i].TickerID]; exists {
				summaries[i].SnapshotChange = *changePtr
				summaries[i].HasSnapshotChange = true
			}
		}

		// Calcular dividendos por ticker
		type DividendSum struct {
			TickerID uint
			Total    float64
		}
		var divSums []DividendSum
		db.Model(&Dividend{}).Select("ticker_id, SUM(amount) as total").Group("ticker_id").Scan(&divSums)
		divMap := make(map[uint]float64)
		for _, d := range divSums {
			divMap[d.TickerID] = d.Total
		}
		for i := range summaries {
			summaries[i].Dividends = divMap[summaries[i].TickerID]
		}

		c.HTML(http.StatusOK, "resumen.html", gin.H{
			"Summaries":  summaries,
			"ActivePage": "resumen",
		})
	})

	// Ruta para exportar la tabla resumen a TOML
	router.GET("/export-resumen-toml", func(c *gin.Context) {
		_, summaries, _, _, _, _, _, _, _, _, _, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		// Cargar todos los dividendos con sus tickers
		var allDividends []Dividend
		db.Preload("Ticker").Order("date asc").Find(&allDividends)

		// Agrupar dividendos por símbolo de ticker
		type divEntry struct {
			Date   string
			Amount float64
			Notes  string
		}
		dividendsByTicker := make(map[string][]divEntry)
		for _, d := range allDividends {
			sym := d.Ticker.Name
			dividendsByTicker[sym] = append(dividendsByTicker[sym], divEntry{
				Date:   d.Date.Format("2006-01-02"),
				Amount: d.Amount,
				Notes:  d.Notes,
			})
		}

		var b strings.Builder
		b.WriteString("# Resumen de Inversiones\n")
		b.WriteString(fmt.Sprintf("# Fecha de exportación: %s\n", time.Now().Local().Format("2006-01-02 15:04:05")))
		b.WriteString("currency = \"EUR\"\n\n")

		for _, s := range summaries {
			// Solo exportar si hay acciones actualmente
			if s.TotalShares > 0 {
				b.WriteString("[[tickers]]\n")
				b.WriteString(fmt.Sprintf("symbol = \"%s\"\n", s.Ticker))
				b.WriteString(fmt.Sprintf("shares = %.6f\n", s.TotalShares))
				b.WriteString(fmt.Sprintf("invested = %.2f\n", s.CurrentInvestment))
				b.WriteString(fmt.Sprintf("cost = %.2f\n", s.TotalCost))
				b.WriteString(fmt.Sprintf("value = %.2f\n", s.CurrentValue))
				b.WriteString(fmt.Sprintf("profit_loss = %.2f\n", s.ProfitLoss))
				b.WriteString(fmt.Sprintf("performance = %.2f\n", s.Performance))
				b.WriteString(fmt.Sprintf("sales_profit = %.2f\n", s.SalesProfit))
				b.WriteString(fmt.Sprintf("dividends_total = %.2f\n", s.Dividends))
				b.WriteString(fmt.Sprintf("weight = %.2f\n", s.WeightPercentage))
				if s.GroupName != "" {
					b.WriteString(fmt.Sprintf("group = \"%s\"\n", s.GroupName))
				}
				// Recibos individuales de dividendos
				if entries, ok := dividendsByTicker[s.Ticker]; ok {
					for _, e := range entries {
						b.WriteString(fmt.Sprintf("  [[tickers.dividends]]\n"))
						b.WriteString(fmt.Sprintf("  date = \"%s\"\n", e.Date))
						b.WriteString(fmt.Sprintf("  amount = %.2f\n", e.Amount))
						if e.Notes != "" {
							b.WriteString(fmt.Sprintf("  notes = \"%s\"\n", e.Notes))
						}
					}
				}
				b.WriteString("\n")
			}
		}

		// Configurar headers para descarga
		filename := fmt.Sprintf("resumen_inversiones_%s.toml", time.Now().Format("20060102"))
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Header("Content-Type", "application/toml")
		c.String(http.StatusOK, b.String())
	})

	// Ruta para mostrar la página de compras
	router.GET("/compras", func(c *gin.Context) {
		investments, summaries, _, _, _, _, _, _, _, _, _, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		// Crear mapa de acciones actuales por ticker
		tickerShares := make(map[string]float64)
		for _, s := range summaries {
			tickerShares[s.Ticker] = s.TotalShares
		}

		// Obtener todos los tickers disponibles
		var tickers []Ticker
		db.Order("name").Find(&tickers)
		var tickerViews []TickerView
		for _, t := range tickers {
			tickerViews = append(tickerViews, TickerView{ID: t.ID, Name: t.Name, CurrentPrice: t.CurrentPrice})
		}

		c.HTML(http.StatusOK, "compras.html", gin.H{
			"Investments":  investments,
			"Tickers":      tickerViews,
			"TickerShares": tickerShares,
			"ActivePage":   "compras",
		})
	})

	// Ruta para mostrar la página de ventas
	router.GET("/ventas", func(c *gin.Context) {
		_, summaries, sales, _, _, _, _, _, _, _, _, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		// Crear mapa de acciones actuales por ticker
		tickerShares := make(map[string]float64)
		for _, s := range summaries {
			tickerShares[s.Ticker] = s.TotalShares
		}

		// Obtener todos los tickers disponibles
		var tickers []Ticker
		db.Order("name").Find(&tickers)
		var tickerViews []TickerView
		for _, t := range tickers {
			tickerViews = append(tickerViews, TickerView{ID: t.ID, Name: t.Name, CurrentPrice: t.CurrentPrice})
		}

		c.HTML(http.StatusOK, "ventas.html", gin.H{
			"Sales":        sales,
			"Tickers":      tickerViews,
			"TickerShares": tickerShares,
			"ActivePage":   "ventas",
		})
	})

	// Ruta para mostrar la página de dividendos
	router.GET("/dividendos", func(c *gin.Context) {
		var dividends []Dividend
		db.Preload("Ticker").Order("date desc").Find(&dividends)

		var dividendViews []DividendView
		totalDividends := 0.0
		for _, d := range dividends {
			dividendViews = append(dividendViews, DividendView{
				ID:       d.ID,
				TickerID: d.TickerID,
				Ticker:   d.Ticker.Name,
				Date:     d.Date.Local().Format("02 Jan 2006"),
				Amount:   d.Amount,
				Notes:    d.Notes,
			})
			totalDividends += d.Amount
		}

		var tickers []Ticker
		db.Order("name").Find(&tickers)
		var tickerViews []TickerView
		for _, t := range tickers {
			tickerViews = append(tickerViews, TickerView{ID: t.ID, Name: t.Name})
		}

		c.HTML(http.StatusOK, "dividendos.html", gin.H{
			"Dividends":      dividendViews,
			"TotalDividends": totalDividends,
			"Tickers":        tickerViews,
			"ActivePage":     "dividendos",
		})
	})

	// API: Registrar un nuevo dividendo
	router.POST("/api/dividend", func(c *gin.Context) {
		var input struct {
			TickerID uint    `json:"ticker_id"`
			Date     string  `json:"date"`
			Amount   float64 `json:"amount"`
			Notes    string  `json:"notes"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
			return
		}

		date, err := time.ParseInLocation("2006-01-02", input.Date, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido"})
			return
		}

		dividend := Dividend{
			TickerID: input.TickerID,
			Date:     date.UTC(),
			Amount:   input.Amount,
			Notes:    input.Notes,
		}
		if err := db.Create(&dividend).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar el dividendo"})
			return
		}

		var ticker Ticker
		db.First(&ticker, input.TickerID)

		c.JSON(http.StatusOK, gin.H{
			"id":        dividend.ID,
			"ticker_id": dividend.TickerID,
			"ticker":    ticker.Name,
			"date":      dividend.Date.Local().Format("02 Jan 2006"),
			"amount":    dividend.Amount,
			"notes":     dividend.Notes,
		})
	})

	// API: Obtener datos de un dividendo
	router.GET("/api/dividend/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
			return
		}

		var dividend Dividend
		if err := db.Preload("Ticker").First(&dividend, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Dividendo no encontrado"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":        dividend.ID,
			"ticker_id": dividend.TickerID,
			"ticker":    dividend.Ticker.Name,
			"date":      dividend.Date.Local().Format("2006-01-02"),
			"amount":    dividend.Amount,
			"notes":     dividend.Notes,
		})
	})

	// API: Actualizar un dividendo
	router.PUT("/api/dividend/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
			return
		}

		var dividend Dividend
		if err := db.First(&dividend, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Dividendo no encontrado"})
			return
		}

		var input struct {
			TickerID uint    `json:"ticker_id"`
			Date     string  `json:"date"`
			Amount   float64 `json:"amount"`
			Notes    string  `json:"notes"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
			return
		}

		date, err := time.ParseInLocation("2006-01-02", input.Date, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido"})
			return
		}

		db.Model(&dividend).Updates(map[string]interface{}{
			"ticker_id": input.TickerID,
			"date":      date.UTC(),
			"amount":    input.Amount,
			"notes":     input.Notes,
		})

		var ticker Ticker
		db.First(&ticker, input.TickerID)

		c.JSON(http.StatusOK, gin.H{
			"id":        id,
			"ticker_id": input.TickerID,
			"ticker":    ticker.Name,
			"date":      date.Local().Format("02 Jan 2006"),
			"amount":    input.Amount,
			"notes":     input.Notes,
		})
	})

	// API: Eliminar un dividendo
	router.DELETE("/api/dividend/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
			return
		}
		if err := db.Delete(&Dividend{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el dividendo"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	// Ruta para mostrar la página de precios
	router.GET("/precios", func(c *gin.Context) {
		var tickers []Ticker
		db.Preload("Group").Order("name").Find(&tickers)

		var groups []TickerGroup
		db.Order("name").Find(&groups)

		// Obtener los dos últimos snapshots
		type SnapshotInfo struct {
			SnapshotID string
			CreatedAt  time.Time
		}
		var snapshots []SnapshotInfo
		db.Model(&PriceHistory{}).
			Select("DISTINCT snapshot_id, MIN(created_at) as created_at").
			Group("snapshot_id").
			Order("created_at DESC").
			Limit(2).
			Scan(&snapshots)

		// Crear un mapa para almacenar los cambios porcentuales por ticker
		snapshotChanges := make(map[uint]*float64)

		// Si hay al menos 2 snapshots, calcular los cambios
		// Mapas para precios de snapshots
		lastPriceMap := make(map[uint]float64)

		// Si hay al menos 1 snapshot, cargarlo para comparar con actual
		if len(snapshots) >= 1 {
			lastSnapshotID := snapshots[0].SnapshotID
			var lastPrices []PriceHistory
			db.Where("snapshot_id = ?", lastSnapshotID).Find(&lastPrices)
			for _, p := range lastPrices {
				lastPriceMap[p.TickerID] = p.Price
			}
		}

		// Si hay al menos 2 snapshots, calcular cambios entre snapshots (SnapshotChange)
		if len(snapshots) >= 2 {
			prevSnapshotID := snapshots[1].SnapshotID
			var prevPrices []PriceHistory
			db.Where("snapshot_id = ?", prevSnapshotID).Find(&prevPrices)

			prevPriceMap := make(map[uint]float64)
			for _, p := range prevPrices {
				prevPriceMap[p.TickerID] = p.Price
			}

			// Calcular cambios porcentuales entre snapshots
			for tickerID, lastPrice := range lastPriceMap {
				if prevPrice, exists := prevPriceMap[tickerID]; exists && prevPrice > 0 {
					change := ((lastPrice - prevPrice) / prevPrice) * 100
					snapshotChanges[tickerID] = &change
				}
			}
		}

		// Obtener datos de inversión para saber si tenemos acciones
		_, summaries, _, _, _, _, _, _, _, _, _, _ := getInvestmentData()
		sharesMap := make(map[uint]float64)
		for _, s := range summaries {
			sharesMap[s.TickerID] = s.TotalShares
		}

		var tickerViews []TickerView
		for _, t := range tickers {
			changePtr := snapshotChanges[t.ID]
			hasChange := changePtr != nil
			changeVal := 0.0
			if hasChange {
				changeVal = *changePtr
			}

			gid := uint(0)
			if t.GroupID != nil {
				gid = *t.GroupID
			}

			// Calcular cambio vs último snapshot
			currentToSnapshotPct := 0.0
			hasCurrentToSnapshotPct := false
			if lastSnapPrice, ok := lastPriceMap[t.ID]; ok && lastSnapPrice > 0 {
				currentToSnapshotPct = ((t.CurrentPrice - lastSnapPrice) / lastSnapPrice) * 100
				hasCurrentToSnapshotPct = true
			}

			tickerViews = append(tickerViews, TickerView{
				ID:                      t.ID,
				Name:                    t.Name,
				CurrentPrice:            t.CurrentPrice,
				UpdatedAt:               t.UpdatedAt.Local().Format("02 Jan 2006 15:04"),
				SnapshotChange:          changeVal,
				HasSnapshotChange:       hasChange,
				YahooFinanceTicker:      t.YahooFinanceTicker,
				YahooEur:                t.YahooEur,
				HasShares:               sharesMap[t.ID] > 0.000001,
				GroupName:               t.Group.Name,
				GroupID:                 gid,
				Active:                  t.Active,
				CurrentToSnapshotPct:    currentToSnapshotPct,
				HasCurrentToSnapshotPct: hasCurrentToSnapshotPct,
			})
		}

		c.HTML(http.StatusOK, "precios.html", gin.H{
			"Tickers":    tickerViews,
			"Groups":     groups,
			"ActivePage": "precios",
		})
	})

	// Ruta para mostrar la página de snapshots
	router.GET("/snapshots", func(c *gin.Context) {
		// Obtener todos los snapshots agrupados por SnapshotID
		type SnapshotGroup struct {
			SnapshotID string
			CreatedAt  time.Time
			Count      int64
		}

		var snapshots []SnapshotGroup
		db.Model(&PriceHistory{}).
			Select("snapshot_id, MIN(created_at) as created_at, COUNT(*) as count").
			Group("snapshot_id").
			Order("created_at DESC").
			Scan(&snapshots)

		// --- Calculation for Top Gainers/Losers ---
		type TickerPerf struct {
			Ticker     string
			FirstDate  time.Time
			LastDate   time.Time
			FirstPrice float64
			LastPrice  float64
			AbsChange  float64
			PctChange  float64
		}

		var allHistory []PriceHistory
		db.Preload("Ticker").Order("created_at asc").Find(&allHistory)

		perfMap := make(map[uint]*TickerPerf)

		for _, h := range allHistory {
			if _, exists := perfMap[h.TickerID]; !exists {
				perfMap[h.TickerID] = &TickerPerf{
					Ticker:     h.Ticker.Name,
					FirstDate:  h.CreatedAt,
					FirstPrice: h.Price,
				}
			}
			// Update last entry (since we are iterating in ASC order)
			perfMap[h.TickerID].LastDate = h.CreatedAt
			perfMap[h.TickerID].LastPrice = h.Price
		}

		var perfs []*TickerPerf
		for _, p := range perfMap {
			p.AbsChange = p.LastPrice - p.FirstPrice
			if p.FirstPrice != 0 {
				p.PctChange = (p.AbsChange / p.FirstPrice) * 100
			}
			perfs = append(perfs, p)
		}

		// Sort for Gainers (Highest PctChange)
		sort.Slice(perfs, func(i, j int) bool {
			return perfs[i].PctChange > perfs[j].PctChange
		})

		var topGainers []*TickerPerf
		for i := 0; i < len(perfs) && i < 10; i++ {
			if perfs[i].PctChange > 0 {
				topGainers = append(topGainers, perfs[i])
			}
		}

		// Sort for Losers (Lowest PctChange - most negative)
		sort.Slice(perfs, func(i, j int) bool {
			return perfs[i].PctChange < perfs[j].PctChange
		})

		var topLosers []*TickerPerf
		for i := 0; i < len(perfs) && i < 10; i++ {
			if perfs[i].PctChange < 0 {
				topLosers = append(topLosers, perfs[i])
			}
		}

		c.HTML(http.StatusOK, "snapshots.html", gin.H{
			"Snapshots":  snapshots,
			"TopGainers": topGainers,
			"TopLosers":  topLosers,
			"ActivePage": "snapshots",
		})
	})

	// Ruta para agregar un nuevo ticker
	router.POST("/add-ticker", func(c *gin.Context) {
		name := strings.ToUpper(c.PostForm("name"))
		priceStr := strings.Replace(c.PostForm("current_price"), ",", ".", -1)
		yahooTicker := c.PostForm("yahoo_finance_ticker")
		yahooEur := c.PostForm("yahoo_eur") == "on"

		if name == "" {
			c.String(http.StatusBadRequest, "El nombre del ticker es obligatorio.")
			return
		}

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			price = 0
		}

		// Verificar si ya existe
		var existing Ticker
		if db.Where("name = ?", name).First(&existing).Error == nil {
			c.String(http.StatusBadRequest, "El ticker ya existe.")
			return
		}

		newTicker := Ticker{Name: name, CurrentPrice: price, YahooFinanceTicker: yahooTicker, YahooEur: yahooEur, Active: true}

		// Manejar Grupo
		groupIDStr := c.PostForm("group_id")
		newGroupName := c.PostForm("new_group_name")

		if newGroupName != "" {
			var group TickerGroup
			if db.Where("name = ?", newGroupName).First(&group).Error != nil {
				group = TickerGroup{Name: newGroupName}
				db.Create(&group)
			}
			newTicker.GroupID = &group.ID
		} else if groupIDStr != "" && groupIDStr != "0" {
			gid, _ := strconv.Atoi(groupIDStr)
			ugid := uint(gid)
			newTicker.GroupID = &ugid
		}

		db.Create(&newTicker)

		log.Printf("Nuevo ticker creado: %s", name)
		c.Redirect(http.StatusFound, "/precios")
	})

	// Ruta para actualizar un ticker (nombre y/o precio)
	router.POST("/update-ticker/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.String(http.StatusBadRequest, "ID inválido.")
			return
		}

		var ticker Ticker
		if err := db.First(&ticker, id).Error; err != nil {
			c.String(http.StatusNotFound, "Ticker no encontrado.")
			return
		}

		name := strings.ToUpper(c.PostForm("name"))
		priceStr := strings.Replace(c.PostForm("current_price"), ",", ".", -1)
		yahooTicker := c.PostForm("yahoo_finance_ticker")
		yahooEur := c.PostForm("yahoo_eur") == "on"
		active := c.PostForm("active") == "on"

		if name == "" {
			c.String(http.StatusBadRequest, "El nombre del ticker es obligatorio.")
			return
		}

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			price = ticker.CurrentPrice
		}

		// Manejar Grupo
		groupIDStr := c.PostForm("group_id")
		newGroupName := c.PostForm("new_group_name")
		var gidPtr *uint

		if newGroupName != "" {
			var group TickerGroup
			if db.Where("name = ?", newGroupName).First(&group).Error != nil {
				group = TickerGroup{Name: newGroupName}
				db.Create(&group)
			}
			gidPtr = &group.ID
		} else if groupIDStr != "" {
			if groupIDStr == "0" {
				gidPtr = nil
			} else {
				gid, _ := strconv.Atoi(groupIDStr)
				ugid := uint(gid)
				gidPtr = &ugid
			}
		} else {
			gidPtr = ticker.GroupID
		}

		// Actualizar campos usando Select para incluir campos vacíos
		db.Model(&ticker).Select("Name", "CurrentPrice", "YahooFinanceTicker", "YahooEur", "GroupID", "Active").Updates(Ticker{
			Name:               name,
			CurrentPrice:       price,
			YahooFinanceTicker: yahooTicker,
			YahooEur:           yahooEur,
			GroupID:            gidPtr,
			Active:             active,
		})

		log.Printf("Ticker %d actualizado: %s", id, name)
		c.Redirect(http.StatusFound, "/precios")
	})

	// Ruta para eliminar un ticker
	router.POST("/delete-ticker", func(c *gin.Context) {
		idStr := c.PostForm("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.String(http.StatusBadRequest, "ID inválido.")
			return
		}

		// Verificar si hay inversiones o ventas asociadas
		var investmentCount int64
		var saleCount int64
		db.Model(&Investment{}).Where("ticker_id = ?", id).Count(&investmentCount)
		db.Model(&Sale{}).Where("ticker_id = ?", id).Count(&saleCount)

		if investmentCount > 0 || saleCount > 0 {
			c.String(http.StatusBadRequest, "No se puede eliminar el ticker porque tiene inversiones o ventas asociadas.")
			return
		}

		db.Delete(&Ticker{}, id)
		log.Printf("Ticker %d eliminado", id)
		c.Redirect(http.StatusFound, "/precios")
	})

	// Ruta para activar/desactivar un ticker
	router.POST("/toggle-ticker-active/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, _ := strconv.Atoi(idStr)

		var ticker Ticker
		if err := db.First(&ticker, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Ticker no encontrado"})
			return
		}

		ticker.Active = !ticker.Active
		db.Save(&ticker)

		c.JSON(http.StatusOK, gin.H{"success": true, "active": ticker.Active})
	})

	// Ruta para actualizar múltiples precios de tickers que tengan Yahoo Finance Ticker
	router.POST("/precios", func(c *gin.Context) {
		var input struct {
			Tickers []string `json:"tickers"`
		}
		if err := c.BindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido: " + err.Error()})
			return
		}

		if len(input.Tickers) == 0 {
			c.JSON(http.StatusOK, gin.H{"success": true, "updated_count": 0, "results": []string{}})
			return
		}

		// Preparar el cuerpo para el servicio externo
		jsonData, err := json.Marshal(input)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al preparar la solicitud"})
			return
		}

		// Llamar al servicio externo de precios (batch)
		reqBatch, err := http.NewRequest("POST", preciosBaseURL+"/precios", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("Error creando petición al servicio de precios: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando la petición"})
			return
		}
		reqBatch.Header.Set("Content-Type", "application/json")
		reqBatch.Header.Set("X-API-Key", preciosAPIKey)
		resp, err := http.DefaultClient.Do(reqBatch)
		if err != nil {
			log.Printf("Error conectando con servicio de precios: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "No se puede conectar con el servicio de precios"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			c.JSON(resp.StatusCode, gin.H{"error": "El servicio de precios devolvió un error"})
			return
		}

		var priceResults []interface{}
		if err := json.NewDecoder(resp.Body).Decode(&priceResults); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al decodificar la respuesta del servicio"})
			return
		}

		results := []gin.H{}
		updatedCount := 0

		for i, res := range priceResults {
			yahooTickerName := input.Tickers[i]

			// Si el servicio devolvió false para este ticker
			if res == nil || res == false {
				results = append(results, gin.H{
					"ticker":  yahooTickerName,
					"success": false,
					"error":   "No se pudo obtener el precio",
				})
				continue
			}

			// Mapear el resultado
			m, ok := res.(map[string]interface{})
			if !ok {
				results = append(results, gin.H{
					"ticker":  yahooTickerName,
					"success": false,
					"error":   "Formato de respuesta inválido",
				})
				continue
			}

			// Buscar el ticker en nuestra DB (usando el campo YahooFinanceTicker)
			var dbTicker Ticker
			if err := db.Where("yahoo_finance_ticker = ?", yahooTickerName).First(&dbTicker).Error; err != nil {
				results = append(results, gin.H{
					"ticker":  yahooTickerName,
					"success": false,
					"error":   "Ticker no encontrado en la base de datos",
				})
				continue
			}

			// Lógica de precio según moneda (siguiendo el patrón existente)
			var price float64
			if dbTicker.YahooEur {
				if v, ok := m["precio_usd"].(float64); ok {
					price = v
				}
			} else {
				if v, ok := m["precio_eur"].(float64); ok {
					price = v
				}
			}

			if price > 0 {
				dbTicker.CurrentPrice = price
				db.Save(&dbTicker)
				updatedCount++
				results = append(results, gin.H{
					"ticker":  yahooTickerName,
					"success": true,
					"price":   price,
				})
			} else {
				results = append(results, gin.H{
					"ticker":  yahooTickerName,
					"success": false,
					"error":   "Precio inválido recibido",
				})
			}
		}

		log.Printf("Actualización masiva completada: %d tickers actualizados", updatedCount)
		c.JSON(http.StatusOK, gin.H{
			"success":       true,
			"updated_count": updatedCount,
			"results":       results,
		})
	})

	// Ruta para crear un snapshot de precios
	router.POST("/create-snapshot", func(c *gin.Context) {
		// Obtener todos los tickers
		var tickers []Ticker
		db.Find(&tickers)

		if len(tickers) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "No hay tickers para crear un snapshot",
			})
			return
		}

		// Generar un ID único para este snapshot usando timestamp
		snapshotID := time.Now().Format("20060102-150405")

		// Crear un registro de precio para cada ticker
		var priceHistories []PriceHistory
		for _, ticker := range tickers {
			priceHistories = append(priceHistories, PriceHistory{
				SnapshotID: snapshotID,
				TickerID:   ticker.ID,
				Price:      ticker.CurrentPrice,
			})
		}

		// Guardar todos los registros en la base de datos
		if err := db.Create(&priceHistories).Error; err != nil {
			log.Printf("Error al crear snapshot: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Error al crear el snapshot",
			})
			return
		}

		log.Printf("Snapshot creado: %s con %d precios", snapshotID, len(priceHistories))
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"message":    fmt.Sprintf("Snapshot creado exitosamente con %d precios", len(priceHistories)),
			"snapshotID": snapshotID,
			"count":      len(priceHistories),
		})
	})

	// Ruta para eliminar un snapshot
	router.POST("/delete-snapshot", func(c *gin.Context) {
		snapshotID := c.PostForm("snapshot_id")

		if snapshotID == "" {
			c.Redirect(http.StatusFound, "/snapshots")
			return
		}

		// Eliminar todos los registros con este snapshot_id
		if err := db.Where("snapshot_id = ?", snapshotID).Delete(&PriceHistory{}).Error; err != nil {
			log.Printf("Error al eliminar snapshot %s: %v", snapshotID, err)
		} else {
			log.Printf("Snapshot eliminado: %s", snapshotID)
		}

		c.Redirect(http.StatusFound, "/snapshots")
	})

	// Ruta para registrar una nueva compra
	router.POST("/add-investment", func(c *gin.Context) {
		// Parsear valores del formulario
		tickerIDStr := c.PostForm("ticker_id")
		purchaseDateStr := c.PostForm("purchase_date")
		sharesStr := strings.Replace(c.PostForm("shares"), ",", ".", -1)
		purchasePriceStr := strings.Replace(c.PostForm("purchase_price"), ",", ".", -1)
		operationCostStr := strings.Replace(c.PostForm("operation_cost"), ",", ".", -1)
		redirectTo := c.PostForm("redirect_to")
		if redirectTo == "" {
			redirectTo = "/"
		}

		// Validar y convertir tipos
		tickerID, err := strconv.Atoi(tickerIDStr)
		if err != nil || tickerID <= 0 {
			c.String(http.StatusBadRequest, "Debe seleccionar un ticker válido.")
			return
		}

		if purchaseDateStr == "" {
			c.String(http.StatusBadRequest, "La fecha es obligatoria.")
			return
		}

		shares, err := strconv.ParseFloat(sharesStr, 64)
		if err != nil || shares <= 0 {
			c.String(http.StatusBadRequest, "La cantidad de acciones debe ser un número positivo.")
			return
		}

		purchasePrice, err := strconv.ParseFloat(purchasePriceStr, 64)
		if err != nil || purchasePrice <= 0 {
			c.String(http.StatusBadRequest, "El precio de compra debe ser un número positivo.")
			return
		}

		operationCost, err := strconv.ParseFloat(operationCostStr, 64)
		if err != nil {
			operationCost = 0 // Default to 0 if empty or invalid
		}

		purchaseDate, err := time.ParseInLocation("2006-01-02T15:04", purchaseDateStr, time.Local)
		if err != nil {
			// Intentar formato sin hora para compatibilidad
			purchaseDate, err = time.ParseInLocation("2006-01-02", purchaseDateStr, time.Local)
			if err != nil {
				c.String(http.StatusBadRequest, "Formato de fecha inválido.")
				return
			}
		}
		purchaseDate = purchaseDate.UTC()

		// Verificar que el ticker existe
		var ticker Ticker
		if err := db.First(&ticker, tickerID).Error; err != nil {
			c.String(http.StatusBadRequest, "El ticker seleccionado no existe.")
			return
		}

		// Crear la nueva inversión
		newInvestment := Investment{
			TickerID:      uint(tickerID),
			PurchaseDate:  purchaseDate,
			Shares:        shares,
			PurchasePrice: purchasePrice,
			OperationCost: operationCost,
		}
		db.Create(&newInvestment)

		log.Printf("Nueva compra registrada para ticker ID %d", tickerID)
		c.Redirect(http.StatusFound, redirectTo)
	})

	// Ruta para registrar una nueva venta
	router.POST("/add-sale", func(c *gin.Context) {
		// Parsear valores del formulario
		tickerIDStr := c.PostForm("ticker_id")
		saleDateStr := c.PostForm("sale_date")
		sharesStr := strings.Replace(c.PostForm("shares"), ",", ".", -1)
		salePriceStr := strings.Replace(c.PostForm("sale_price"), ",", ".", -1)
		operationCostStr := strings.Replace(c.PostForm("operation_cost"), ",", ".", -1)
		redirectTo := c.PostForm("redirect_to")
		if redirectTo == "" {
			redirectTo = "/"
		}

		// Validar y convertir tipos
		tickerID, err := strconv.Atoi(tickerIDStr)
		if err != nil || tickerID <= 0 {
			c.String(http.StatusBadRequest, "Debe seleccionar un ticker válido.")
			return
		}

		if saleDateStr == "" {
			c.String(http.StatusBadRequest, "La fecha de venta es obligatoria.")
			return
		}

		shares, err := strconv.ParseFloat(sharesStr, 64)
		if err != nil || shares <= 0 {
			c.String(http.StatusBadRequest, "La cantidad de acciones debe ser un número positivo.")
			return
		}

		salePrice, err := strconv.ParseFloat(salePriceStr, 64)
		if err != nil || salePrice <= 0 {
			c.String(http.StatusBadRequest, "El precio de venta debe ser un número positivo.")
			return
		}

		operationCost, err := strconv.ParseFloat(operationCostStr, 64)
		if err != nil {
			operationCost = 0 // Default to 0 if empty or invalid
		}

		saleDate, err := time.ParseInLocation("2006-01-02T15:04", saleDateStr, time.Local)
		if err != nil {
			// Intentar formato DD/MM/YYYY para compatibilidad
			saleDate, err = time.ParseInLocation("02/01/2006", saleDateStr, time.Local)
			if err != nil {
				c.String(http.StatusBadRequest, "Formato de fecha inválido.")
				return
			}
		}
		saleDate = saleDate.UTC()

		// Verificar que el ticker existe
		var ticker Ticker
		if err := db.First(&ticker, tickerID).Error; err != nil {
			c.String(http.StatusBadRequest, "El ticker seleccionado no existe.")
			return
		}

		// Crear la nueva venta
		newSale := Sale{
			TickerID:      uint(tickerID),
			SaleDate:      saleDate,
			Shares:        shares,
			SalePrice:     salePrice,
			OperationCost: operationCost,
		}
		db.Create(&newSale)

		log.Printf("Nueva venta registrada para ticker ID %d", tickerID)
		c.Redirect(http.StatusFound, redirectTo)
	})

	// Ruta para eliminar una venta
	router.POST("/delete-sale", func(c *gin.Context) {
		idStr := c.PostForm("id")
		redirectTo := c.PostForm("redirect_to")
		if redirectTo == "" {
			redirectTo = "/"
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.String(http.StatusBadRequest, "ID inválido.")
			return
		}

		db.Delete(&Sale{}, id)

		log.Printf("Registro de venta con ID %d marcado como eliminado", id)
		c.Redirect(http.StatusFound, redirectTo)
	})

	// Ruta para obtener detalles del cálculo de utilidad
	router.GET("/sale-calculation/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
			return
		}

		var sale Sale
		if err := db.Preload("Ticker").First(&sale, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Venta no encontrada"})
			return
		}

		// Obtener todas las inversiones y ventas anteriores para este ticker
		var investments []Investment
		db.Where("ticker_id = ? AND purchase_date <= ?", sale.TickerID, sale.SaleDate).Order("purchase_date asc").Find(&investments)

		var sales []Sale
		db.Where("ticker_id = ? AND sale_date <= ?", sale.TickerID, sale.SaleDate).Order("sale_date asc").Find(&sales)

		// Reconstruir la historia para calcular el WAC en el momento de la venta
		type Event struct {
			Date   time.Time
			Type   string // "buy", "sell"
			Shares float64
			Price  float64
			ID     uint
		}

		var events []Event
		for _, inv := range investments {
			events = append(events, Event{Date: inv.PurchaseDate, Type: "buy", Shares: inv.Shares, Price: inv.PurchasePrice, ID: inv.ID})
		}
		for _, s := range sales {
			// Excluir la venta actual del cálculo histórico (queremos el estado JUSTO ANTES)
			if s.ID == sale.ID {
				continue
			}
			events = append(events, Event{Date: s.SaleDate, Type: "sell", Shares: s.Shares, Price: s.SalePrice, ID: s.ID})
		}

		// Ordenar eventos
		sort.Slice(events, func(i, j int) bool {
			if events[i].Date.Equal(events[j].Date) {
				return events[i].Type == "buy"
			}
			return events[i].Date.Before(events[j].Date)
		})

		currentShares := 0.0
		currentCapital := 0.0

		for _, e := range events {
			if e.Type == "buy" {
				currentShares += e.Shares
				currentCapital += e.Shares * e.Price
			} else if e.Type == "sell" {
				wac := 0.0
				if currentShares > 0 {
					wac = currentCapital / currentShares
				}
				currentCapital -= e.Shares * wac
				currentShares -= e.Shares
			}
		}

		// Calcular WAC final
		wac := 0.0
		if currentShares > 0 {
			wac = currentCapital / currentShares
		}

		// Preparar respuesta
		type PurchaseInfo struct {
			Date   string  `json:"date"`
			Shares float64 `json:"shares"`
			Price  float64 `json:"price"`
			Total  float64 `json:"total"`
		}

		var purchasesList []PurchaseInfo
		for _, inv := range investments {
			purchasesList = append(purchasesList, PurchaseInfo{
				Date:   inv.PurchaseDate.Local().Format("02 Jan 2006 15:04"),
				Shares: inv.Shares,
				Price:  inv.PurchasePrice,
				Total:  inv.Shares * inv.PurchasePrice,
			})
		}

		// Utilidad calculada solo con precios
		profit := (sale.SalePrice - wac) * sale.Shares

		c.JSON(http.StatusOK, gin.H{
			"ticker":        sale.Ticker.Name,
			"sale_date":     sale.SaleDate.Local().Format("02 Jan 2006 15:04"),
			"shares":        sale.Shares,
			"sale_price":    sale.SalePrice,
			"purchases":     purchasesList,
			"total_capital": currentCapital, // Capital acumulado antes de la venta
			"total_shares":  currentShares,  // Acciones acumuladas antes de la venta
			"wac":           wac,
			"profit":        profit,
		})
	})

	// Ruta para mostrar el formulario de edición
	router.GET("/edit/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.String(http.StatusBadRequest, "ID inválido.")
			return
		}

		var investment Investment
		if err := db.Preload("Ticker").First(&investment, id).Error; err != nil {
			c.String(http.StatusNotFound, "Registro no encontrado.")
			return
		}

		// Obtener todos los tickers disponibles
		var tickers []Ticker
		db.Order("name").Find(&tickers)
		var tickerViews []TickerView
		for _, t := range tickers {
			tickerViews = append(tickerViews, TickerView{ID: t.ID, Name: t.Name, CurrentPrice: t.CurrentPrice})
		}

		c.HTML(http.StatusOK, "edit.html", gin.H{
			"Investment": investment,
			"Tickers":    tickerViews,
			"ActivePage": "compras",
		})
	})

	// Ruta para actualizar una compra
	router.POST("/update/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.String(http.StatusBadRequest, "ID inválido.")
			return
		}

		var investment Investment
		if err := db.First(&investment, id).Error; err != nil {
			c.String(http.StatusNotFound, "Registro no encontrado.")
			return
		}

		// Parsear y validar datos del formulario
		tickerIDStr := c.PostForm("ticker_id")
		purchaseDateStr := c.PostForm("purchase_date")
		sharesStr := strings.Replace(c.PostForm("shares"), ",", ".", -1)
		purchasePriceStr := strings.Replace(c.PostForm("purchase_price"), ",", ".", -1)
		operationCostStr := strings.Replace(c.PostForm("operation_cost"), ",", ".", -1)

		tickerID, _ := strconv.Atoi(tickerIDStr)
		shares, _ := strconv.ParseFloat(sharesStr, 64)
		purchasePrice, _ := strconv.ParseFloat(purchasePriceStr, 64)
		operationCost, _ := strconv.ParseFloat(operationCostStr, 64)
		purchaseDate, err := time.ParseInLocation("2006-01-02T15:04", purchaseDateStr, time.Local)
		if err != nil {
			purchaseDate, _ = time.ParseInLocation("2006-01-02", purchaseDateStr, time.Local)
		}
		purchaseDate = purchaseDate.UTC()

		// Actualizar el registro
		db.Model(&investment).Updates(map[string]interface{}{
			"ticker_id":      tickerID,
			"purchase_date":  purchaseDate,
			"shares":         shares,
			"purchase_price": purchasePrice,
			"operation_cost": operationCost,
		})

		log.Printf("Registro de compra con ID %d actualizado", id)
		c.Redirect(http.StatusFound, "/compras")
	})

	// API: Actualizar una compra (devuelve JSON)
	router.PUT("/api/investment/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
			return
		}

		var investment Investment
		if err := db.First(&investment, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registro no encontrado"})
			return
		}

		// Parsear JSON del body
		var input struct {
			TickerID      uint    `json:"ticker_id"`
			PurchaseDate  string  `json:"purchase_date"`
			Shares        float64 `json:"shares"`
			PurchasePrice float64 `json:"purchase_price"`
			OperationCost float64 `json:"operation_cost"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
			return
		}

		purchaseDate, err := time.ParseInLocation("2006-01-02T15:04", input.PurchaseDate, time.Local)
		if err != nil {
			purchaseDate, _ = time.ParseInLocation("2006-01-02", input.PurchaseDate, time.Local)
		}
		purchaseDate = purchaseDate.UTC()

		// Actualizar el registro
		db.Model(&investment).Updates(map[string]interface{}{
			"ticker_id":      input.TickerID,
			"purchase_date":  purchaseDate,
			"shares":         input.Shares,
			"purchase_price": input.PurchasePrice,
			"operation_cost": input.OperationCost,
		})

		// Obtener el ticker actualizado para devolver los datos completos
		var ticker Ticker
		db.First(&ticker, input.TickerID)

		investedCapital := input.Shares * input.PurchasePrice
		currentValue := input.Shares * ticker.CurrentPrice
		profitLoss := currentValue - (investedCapital + input.OperationCost)

		log.Printf("Registro de compra con ID %d actualizado via API", id)
		c.JSON(http.StatusOK, gin.H{
			"id":               id,
			"ticker_id":        input.TickerID,
			"ticker":           ticker.Name,
			"purchase_date":    purchaseDate.Local().Format("02 Jan 2006 15:04"),
			"shares":           input.Shares,
			"purchase_price":   input.PurchasePrice,
			"operation_cost":   input.OperationCost,
			"invested_capital": investedCapital,
			"current_price":    ticker.CurrentPrice,
			"current_value":    currentValue,
			"profit_loss":      profitLoss,
		})
	})

	// API: Obtener datos de una compra (devuelve JSON)
	router.GET("/api/investment/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
			return
		}

		var investment Investment
		if err := db.Preload("Ticker").First(&investment, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registro no encontrado"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":             investment.ID,
			"ticker_id":      investment.TickerID,
			"ticker":         investment.Ticker.Name,
			"purchase_date":  investment.PurchaseDate.Local().Format("2006-01-02T15:04"),
			"shares":         investment.Shares,
			"purchase_price": investment.PurchasePrice,
			"operation_cost": investment.OperationCost,
		})
	})

	// API: Actualizar una venta (devuelve JSON)
	router.PUT("/api/sale/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
			return
		}

		var sale Sale
		if err := db.First(&sale, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Venta no encontrada"})
			return
		}

		// Parsear JSON del body
		var input struct {
			TickerID      uint    `json:"ticker_id"`
			SaleDate      string  `json:"sale_date"`
			Shares        float64 `json:"shares"`
			SalePrice     float64 `json:"sale_price"`
			OperationCost float64 `json:"operation_cost"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
			return
		}

		saleDate, err := time.ParseInLocation("2006-01-02T15:04", input.SaleDate, time.Local)
		if err != nil {
			saleDate, _ = time.ParseInLocation("2006-01-02", input.SaleDate, time.Local)
		}
		saleDate = saleDate.UTC()

		// Actualizar el registro
		db.Model(&sale).Updates(map[string]interface{}{
			"ticker_id":      input.TickerID,
			"sale_date":      saleDate,
			"shares":         input.Shares,
			"sale_price":     input.SalePrice,
			"operation_cost": input.OperationCost,
		})

		// Obtener el ticker actualizado
		var ticker Ticker
		db.First(&ticker, input.TickerID)

		// Calcular valores para la respuesta
		totalSaleValue := input.Shares * input.SalePrice

		// Calcular WAC y utilidad (similar a sale-calculation)
		var investments []Investment
		db.Where("ticker_id = ? AND purchase_date <= ?", input.TickerID, saleDate).Order("purchase_date asc").Find(&investments)

		var previousSales []Sale
		db.Where("ticker_id = ? AND sale_date <= ? AND id != ?", input.TickerID, saleDate, id).Order("sale_date asc").Find(&previousSales)

		type Event struct {
			Date   time.Time
			Type   string
			Shares float64
			Price  float64
		}

		var events []Event
		for _, inv := range investments {
			events = append(events, Event{Date: inv.PurchaseDate, Type: "buy", Shares: inv.Shares, Price: inv.PurchasePrice})
		}
		for _, s := range previousSales {
			events = append(events, Event{Date: s.SaleDate, Type: "sell", Shares: s.Shares, Price: s.SalePrice})
		}

		sort.Slice(events, func(i, j int) bool {
			if events[i].Date.Equal(events[j].Date) {
				return events[i].Type == "buy"
			}
			return events[i].Date.Before(events[j].Date)
		})

		currentShares := 0.0
		currentCapital := 0.0

		for _, e := range events {
			if e.Type == "buy" {
				currentShares += e.Shares
				currentCapital += e.Shares * e.Price
			} else if e.Type == "sell" {
				wac := 0.0
				if currentShares > 0 {
					wac = currentCapital / currentShares
				}
				currentCapital -= e.Shares * wac
				currentShares -= e.Shares
			}
		}

		wac := 0.0
		if currentShares > 0 {
			wac = currentCapital / currentShares
		}

		profit := (input.SalePrice - wac) * input.Shares
		performance := 0.0
		if wac > 0 {
			performance = ((input.SalePrice - wac) / wac) * 100
		}

		log.Printf("Registro de venta con ID %d actualizado via API", id)
		c.JSON(http.StatusOK, gin.H{
			"id":               id,
			"ticker_id":        input.TickerID,
			"ticker":           ticker.Name,
			"sale_date":        saleDate.Local().Format("02 Jan 2006 15:04"),
			"shares":           input.Shares,
			"sale_price":       input.SalePrice,
			"operation_cost":   input.OperationCost,
			"total_sale_value": totalSaleValue,
			"performance":      performance,
			"profit":           profit,
		})
	})

	// API: Obtener datos de una venta (devuelve JSON)
	router.GET("/api/sale/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
			return
		}

		var sale Sale
		if err := db.Preload("Ticker").First(&sale, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Venta no encontrada"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":             sale.ID,
			"ticker_id":      sale.TickerID,
			"ticker":         sale.Ticker.Name,
			"sale_date":      sale.SaleDate.Local().Format("2006-01-02T15:04"),
			"shares":         sale.Shares,
			"sale_price":     sale.SalePrice,
			"operation_cost": sale.OperationCost,
		})
	})

	// Ruta para eliminar una compra
	router.POST("/delete-investment", func(c *gin.Context) {
		idStr := c.PostForm("id")
		redirectTo := c.PostForm("redirect_to")
		if redirectTo == "" {
			redirectTo = "/"
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.String(http.StatusBadRequest, "ID inválido.")
			return
		}

		// GORM usa borrado suave (soft delete) porque gorm.Model tiene el campo DeletedAt
		db.Delete(&Investment{}, id)

		log.Printf("Registro de compra con ID %d marcado como eliminado", id)
		c.Redirect(http.StatusFound, redirectTo)
	})

	// Ruta para mostrar el detalle de un ticker
	router.GET("/ticker/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		tickerID, err := strconv.Atoi(idStr)
		if err != nil {
			c.String(http.StatusBadRequest, "ID inválido.")
			return
		}

		// Obtener el ticker
		var ticker Ticker
		if err := db.First(&ticker, tickerID).Error; err != nil {
			c.String(http.StatusNotFound, "Ticker no encontrado.")
			return
		}

		// Obtener las compras del ticker
		var investments []Investment
		db.Where("ticker_id = ?", tickerID).Order("purchase_date desc").Find(&investments)

		var investmentViews []InvestmentView
		var totalInvested float64
		var totalCostBuy float64
		for _, i := range investments {
			investedCapital := i.Shares * i.PurchasePrice
			currentValue := i.Shares * ticker.CurrentPrice
			profitLoss := currentValue - (investedCapital + i.OperationCost)
			performance := 0.0
			if i.PurchasePrice > 0 {
				performance = (ticker.CurrentPrice - i.PurchasePrice) / i.PurchasePrice * 100
			}

			view := InvestmentView{
				ID:              i.ID,
				TickerID:        i.TickerID,
				Ticker:          ticker.Name,
				PurchaseDate:    i.PurchaseDate.Local().Format("02 Jan 2006 15:04:05"),
				Shares:          i.Shares,
				PurchasePrice:   i.PurchasePrice,
				OperationCost:   i.OperationCost,
				InvestedCapital: investedCapital,
				CurrentPrice:    ticker.CurrentPrice,
				CurrentValue:    currentValue,
				ProfitLoss:      profitLoss,
				Performance:     performance,
			}
			investmentViews = append(investmentViews, view)
			totalInvested += investedCapital
			totalCostBuy += i.OperationCost
		}

		// Obtener las ventas del ticker
		var sales []Sale
		db.Where("ticker_id = ?", tickerID).Order("sale_date desc").Find(&sales)

		// Calcular WAC (Weighted Average Cost) para cada venta
		// Crear eventos ordenados cronológicamente
		type Event struct {
			Date   time.Time
			Type   string // "buy", "sell"
			Shares float64
			Price  float64
			SaleID uint // Para identificar la venta
		}

		var events []Event
		for _, i := range investments {
			events = append(events, Event{
				Date:   i.PurchaseDate,
				Type:   "buy",
				Shares: i.Shares,
				Price:  i.PurchasePrice,
			})
		}
		for _, s := range sales {
			events = append(events, Event{
				Date:   s.SaleDate,
				Type:   "sell",
				Shares: s.Shares,
				Price:  s.SalePrice,
				SaleID: s.ID,
			})
		}

		// Ordenar eventos por fecha
		sort.Slice(events, func(i, j int) bool {
			if events[i].Date.Equal(events[j].Date) {
				// Si la fecha es igual, procesar compras antes que ventas
				return events[i].Type == "buy"
			}
			return events[i].Date.Before(events[j].Date)
		})

		// Mapa para guardar el WAC al momento de cada venta
		saleWACMap := make(map[uint]float64)

		currentShares := 0.0
		currentCapital := 0.0

		// Datos para el gráfico de WAC
		var wacChartDates []string
		var wacChartValues []float64

		for _, e := range events {
			if e.Type == "buy" {
				currentShares += e.Shares
				currentCapital += e.Shares * e.Price

				// Calcular nuevo WAC y registrar punto
				if currentShares > 0 {
					wac := currentCapital / currentShares
					wacChartDates = append(wacChartDates, e.Date.Format("02 Jan 2006 15:04:05"))
					wacChartValues = append(wacChartValues, wac)
				}
			} else if e.Type == "sell" {
				wac := 0.0
				if currentShares > 0 {
					wac = currentCapital / currentShares
				}
				// Guardar el WAC para esta venta
				saleWACMap[e.SaleID] = wac
				// Reducir capital proporcionalmente al WAC
				currentCapital -= e.Shares * wac
				currentShares -= e.Shares

				// Registrar punto también en la venta para mantener la línea
				if wac > 0 {
					wacChartDates = append(wacChartDates, e.Date.Format("02 Jan 2006 15:04:05"))
					wacChartValues = append(wacChartValues, wac)
				}
			}
		}

		// WAC final de las acciones en cartera
		// Usar un umbral pequeño (epsilon) para evitar problemas de precisión de punto flotante
		const epsilon = 0.000001
		portfolioWAC := 0.0

		// Calcular WAC final
		if currentShares > epsilon {
			portfolioWAC = currentCapital / currentShares
		} else {
			// Si las acciones son efectivamente cero, resetear también el capital
			currentShares = 0.0
			currentCapital = 0.0
		}

		// Agregar punto final "NOW" para que la línea se extienda hasta el presente si tenemos acciones
		if currentShares > epsilon && portfolioWAC > 0 {
			wacChartDates = append(wacChartDates, time.Now().Local().Format("02 Jan 2006 15:04:05"))
			wacChartValues = append(wacChartValues, portfolioWAC)
		}
		if currentShares > epsilon {
			portfolioWAC = currentCapital / currentShares
		} else {
			// Si las acciones son efectivamente cero, resetear también el capital
			currentShares = 0.0
			currentCapital = 0.0
		}

		// Construir saleViews con el WAC calculado
		var saleViews []SaleView
		var totalSold float64
		var totalCostSell float64
		var totalSaleUtility float64
		for _, s := range sales {
			totalSaleValue := s.Shares * s.SalePrice
			wacAtSale := saleWACMap[s.ID]
			salePerformance := 0.0
			if wacAtSale > 0 {
				salePerformance = ((s.SalePrice - wacAtSale) / wacAtSale) * 100
			}
			saleUtility := (s.SalePrice - wacAtSale) * s.Shares

			view := SaleView{
				ID:              s.ID,
				TickerID:        s.TickerID,
				Ticker:          ticker.Name,
				SaleDate:        s.SaleDate.Local().Format("02 Jan 2006 15:04:05"),
				Shares:          s.Shares,
				SalePrice:       s.SalePrice,
				OperationCost:   s.OperationCost,
				TotalSaleValue:  totalSaleValue,
				WACAtSale:       wacAtSale,
				SalePerformance: salePerformance,
				SaleUtility:     saleUtility,
			}
			saleViews = append(saleViews, view)
			totalSold += totalSaleValue
			totalCostSell += s.OperationCost
			totalSaleUtility += saleUtility
		}

		// Rendimiento porcentual vs precio ponderado
		wacPerformance := 0.0
		if portfolioWAC > 0 {
			wacPerformance = ((ticker.CurrentPrice - portfolioWAC) / portfolioWAC) * 100
		}

		// Utilidad: diferencia entre valor actual y valor ponderado del portafolio
		utilidad := (ticker.CurrentPrice * currentShares) - (portfolioWAC * currentShares)

		// Calcular distancia al WAC (cuánto falta para llegar al precio ponderado)
		distanceToWAC := 0.0
		if ticker.CurrentPrice > 0 && portfolioWAC > 0 {
			distanceToWAC = ((portfolioWAC - ticker.CurrentPrice) / ticker.CurrentPrice) * 100
		}

		// Obtener historial de precios del ticker
		var priceHistories []PriceHistory
		db.Where("ticker_id = ?", tickerID).Order("created_at asc").Find(&priceHistories)

		// Preparar datos para el gráfico
		var priceChartDates []string
		var priceChartValues []float64
		for _, ph := range priceHistories {
			priceChartDates = append(priceChartDates, ph.CreatedAt.Format("02 Jan 2006 15:04:05"))
			priceChartValues = append(priceChartValues, ph.Price)
		}

		// Preparar datos de compras para el gráfico
		var purchaseChartDates []string
		var purchaseChartPrices []float64
		for _, inv := range investmentViews {
			purchaseChartDates = append(purchaseChartDates, inv.PurchaseDate)
			purchaseChartPrices = append(purchaseChartPrices, inv.PurchasePrice)
		}

		// Preparar datos de ventas para el gráfico
		var saleChartDates []string
		var saleChartPrices []float64
		for _, s := range saleViews {
			saleChartDates = append(saleChartDates, s.SaleDate)
			saleChartPrices = append(saleChartPrices, s.SalePrice)
		}

		// Obtener notas del ticker
		var tickerNotes []TickerNote
		db.Where("ticker_id = ?", tickerID).Order("date desc").Find(&tickerNotes)

		c.HTML(http.StatusOK, "ticker_detail.html", gin.H{
			"Ticker":              ticker,
			"Investments":         investmentViews,
			"Sales":               saleViews,
			"TotalInvested":       totalInvested,
			"TotalCostBuy":        totalCostBuy,
			"TotalSold":           totalSold,
			"TotalCostSell":       totalCostSell,
			"TotalCosts":          totalCostBuy + totalCostSell,
			"SharesInPortfolio":   currentShares,
			"CurrentValue":        ticker.CurrentPrice * currentShares,
			"PortfolioWAC":        portfolioWAC,
			"PortfolioWACValue":   currentCapital,
			"WACPerformance":      wacPerformance,
			"Utilidad":            utilidad,
			"DistanceToWAC":       distanceToWAC,
			"TotalSaleUtility":    totalSaleUtility,
			"PriceChartDates":     priceChartDates,
			"PriceChartValues":    priceChartValues,
			"PurchaseChartDates":  purchaseChartDates,
			"PurchaseChartPrices": purchaseChartPrices,
			"SaleChartDates":      saleChartDates,
			"SaleChartPrices":     saleChartPrices,
			"WACChartDates":       wacChartDates,
			"WACChartValues":      wacChartValues,
			"Notes":               tickerNotes,
			"ActivePage":          "resumen",
		})
	})

	// API: Obtener historial de utilidad de la cartera por snapshot
	router.GET("/api/portfolio-utility-history", func(c *gin.Context) {
		// Obtener todos los snapshots ordenados por fecha
		type SnapshotInfo struct {
			SnapshotID string
			CreatedAt  time.Time
		}
		var snapshots []SnapshotInfo
		db.Model(&PriceHistory{}).
			Select("snapshot_id, MIN(created_at) as created_at").
			Group("snapshot_id").
			Order("created_at ASC").
			Scan(&snapshots)

		if len(snapshots) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"dates":     []string{},
				"utilities": []float64{},
			})
			return
		}

		// Obtener todas las inversiones y ventas para seguir el WAC
		var allInvestments []Investment
		db.Order("purchase_date asc").Find(&allInvestments)

		var allSales []Sale
		db.Order("sale_date asc").Find(&allSales)

		// Combinar eventos
		type Event struct {
			Date   time.Time
			Type   string // "buy", "sell"
			Ticker uint
			Shares float64
			Price  float64
		}
		var allEvents []Event
		for _, inv := range allInvestments {
			allEvents = append(allEvents, Event{Date: inv.PurchaseDate, Type: "buy", Ticker: inv.TickerID, Shares: inv.Shares, Price: inv.PurchasePrice})
		}
		for _, sale := range allSales {
			allEvents = append(allEvents, Event{Date: sale.SaleDate, Type: "sell", Ticker: sale.TickerID, Shares: sale.Shares, Price: sale.SalePrice})
		}
		sort.Slice(allEvents, func(i, j int) bool {
			if allEvents[i].Date.Equal(allEvents[j].Date) {
				return allEvents[i].Type == "buy"
			}
			return allEvents[i].Date.Before(allEvents[j].Date)
		})

		// Mapa para seguir el estado de cada ticker
		type TickerState struct {
			Shares  float64
			Capital float64
		}
		tickerStates := make(map[uint]TickerState)

		// Cargar todos los price histories en una sola query y agrupar en memoria
		// para evitar N queries (una por snapshot) contra Supabase.
		var allPriceHistories []PriceHistory
		db.Select("snapshot_id, ticker_id, price").Find(&allPriceHistories)
		pricesBySnapshot := make(map[string][]PriceHistory, len(snapshots))
		for _, ph := range allPriceHistories {
			pricesBySnapshot[ph.SnapshotID] = append(pricesBySnapshot[ph.SnapshotID], ph)
		}

		var dates []string
		var utilities []float64
		eventIdx := 0

		for _, snapshot := range snapshots {
			// Procesar eventos hasta este snapshot
			for eventIdx < len(allEvents) && (allEvents[eventIdx].Date.Before(snapshot.CreatedAt) || allEvents[eventIdx].Date.Equal(snapshot.CreatedAt)) {
				e := allEvents[eventIdx]
				state := tickerStates[e.Ticker]

				if e.Type == "buy" {
					state.Shares += e.Shares
					state.Capital += e.Shares * e.Price
				} else if e.Type == "sell" {
					wac := 0.0
					if state.Shares > 0 {
						wac = state.Capital / state.Shares
					}
					state.Capital -= e.Shares * wac
					state.Shares -= e.Shares
				}

				tickerStates[e.Ticker] = state
				eventIdx++
			}

			totalUtility := 0.0
			for _, ph := range pricesBySnapshot[snapshot.SnapshotID] {
				state := tickerStates[ph.TickerID]
				if state.Shares > 0.000001 {
					wac := state.Capital / state.Shares
					totalUtility += (ph.Price - wac) * state.Shares
				}
			}

			dates = append(dates, snapshot.CreatedAt.Format("02 Jan 2006 15:04:05"))
			utilities = append(utilities, totalUtility)
		}

		c.JSON(http.StatusOK, gin.H{
			"dates":     dates,
			"utilities": utilities,
		})
	})

	preciosAPIKey = os.Getenv("PRECIOS_API_KEY")
	preciosBaseURL = os.Getenv("PRECIOS_BASE_URL")
	if preciosBaseURL == "" {
		preciosBaseURL = "http://localhost:3010"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("Servidor iniciado en http://localhost:%s", port)
	router.Run(":" + port)
}

func setupDatabase() (*gorm.DB, error) {
	// Usar connection string directo de Supabase
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("falta la variable de entorno DATABASE_URL")
	}

	// Agregar parámetros necesarios
	if !strings.Contains(dsn, "prepare") {
		if strings.Contains(dsn, "?") {
			dsn += "&prepare=false"
		} else {
			dsn += "?prepare=false"
		}
	}
	if !strings.Contains(dsn, "TimeZone") {
		if strings.Contains(dsn, "?") {
			dsn += "&TimeZone=UTC"
		} else {
			dsn += "?TimeZone=UTC"
		}
	}

	database, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // Desactiva prepared statements
	}), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Ejecutar migraciones
	if err := runMigrations(database); err != nil {
		return nil, fmt.Errorf("error ejecutando migraciones: %v", err)
	}

	return database, nil
}

// runMigrations ejecuta todas las migraciones pendientes en orden
func runMigrations(database *gorm.DB) error {
	// Crear tabla de migraciones si no existe
	database.AutoMigrate(&Migration{})

	// Definir todas las migraciones disponibles
	migrations := map[string]func(*gorm.DB) error{
		"001_create_initial_schema":           migration001CreateInitialSchema,
		"002_migrate_to_ticker_id_schema":     migration002MigrateToTickerIDSchema,
		"003_create_price_history_table":      migration003CreatePriceHistoryTable,
		"004_add_yahoo_finance_ticker_column": migration004AddYahooFinanceTickerColumn,
		"005_create_notes_table":              migration005CreateNotesTable,
		"006_add_usdeur_column":               migration006AddUsdEurColumn,
		"007_create_ticker_notes_table":       migration007CreateTickerNotesTable,
		"008_drop_tax_column":                 migration008DropTaxColumn,
		"009_create_ticker_groups_table":      migration009CreateTickerGroupsTable,
		"010_add_active_column_to_tickers":    migration010AddActiveColumnToTickers,
		"011_rename_usdeur_to_yahooeur":       migration011RenameUsdEurToYahooEur,
		"012_fix_yahoo_eur_column":            migration012FixYahooEurColumn,
		"013_create_dividends_table":          migration013CreateDividendsTable,
	}

	// Obtener migraciones ya aplicadas
	var appliedMigrations []Migration
	database.Find(&appliedMigrations)
	appliedMap := make(map[string]bool)
	for _, m := range appliedMigrations {
		appliedMap[m.Name] = true
	}

	// Ordenar las migraciones por nombre
	var migrationNames []string
	for name := range migrations {
		migrationNames = append(migrationNames, name)
	}
	sort.Strings(migrationNames)

	// Ejecutar migraciones pendientes
	for _, name := range migrationNames {
		if appliedMap[name] {
			continue
		}

		log.Printf("Ejecutando migración: %s", name)
		if err := migrations[name](database); err != nil {
			return fmt.Errorf("error en migración %s: %v", name, err)
		}

		// Registrar migración como aplicada
		database.Create(&Migration{Name: name, AppliedAt: time.Now()})
		log.Printf("Migración completada: %s", name)
	}

	return nil
}

// migration001CreateInitialSchema crea el esquema inicial con la tabla Ticker
func migration001CreateInitialSchema(database *gorm.DB) error {
	// Verificar si estamos migrando desde esquema antiguo
	hasOldSchema := database.Migrator().HasTable("market_data")
	hasInvestments := database.Migrator().HasTable("investments")

	// Crear tabla tickers si no existe
	if !database.Migrator().HasTable("tickers") {
		log.Println("Creando tabla tickers...")
		if err := database.Exec(`CREATE TABLE "tickers" (
			"id" bigserial PRIMARY KEY,
			"created_at" timestamptz,
			"updated_at" timestamptz,
			"deleted_at" timestamptz,
			"name" text,
			"current_price" decimal
		)`).Error; err != nil {
			return err
		}
		database.Exec(`CREATE UNIQUE INDEX "idx_tickers_name" ON "tickers" ("name")`)
		database.Exec(`CREATE INDEX "idx_tickers_deleted_at" ON "tickers" ("deleted_at")`)
	}

	if hasOldSchema && hasInvestments {
		// Esquema antiguo existe
		log.Println("Detectado esquema antiguo, preparando para migración...")

		// Agregar columna ticker_id a investments si no existe
		if !database.Migrator().HasColumn(&Investment{}, "ticker_id") {
			log.Println("Agregando columna ticker_id a investments...")
			database.Exec("ALTER TABLE investments ADD COLUMN ticker_id bigint")
		}
		// Agregar columna ticker_id a sales si no existe
		if database.Migrator().HasTable("sales") && !database.Migrator().HasColumn(&Sale{}, "ticker_id") {
			log.Println("Agregando columna ticker_id a sales...")
			database.Exec("ALTER TABLE sales ADD COLUMN ticker_id bigint")
		}
		return nil
	}

	// Base de datos nueva - crear tablas investments y sales si no existen
	if !database.Migrator().HasTable("investments") {
		log.Println("Creando tabla investments...")
		database.AutoMigrate(&Investment{})
	}
	if !database.Migrator().HasTable("sales") {
		log.Println("Creando tabla sales...")
		database.AutoMigrate(&Sale{})
	}

	return nil
}

// migration002MigrateToTickerIDSchema migra datos del esquema antiguo al nuevo
func migration002MigrateToTickerIDSchema(database *gorm.DB) error {
	// Verificar si existe la tabla market_data (esquema antiguo)
	if !database.Migrator().HasTable("market_data") {
		log.Println("No se encontró esquema antiguo, saltando migración de datos")

		// Si no hay datos, insertar datos de ejemplo
		var count int64
		database.Model(&Ticker{}).Count(&count)
		if count == 0 {
			log.Println("Insertando datos de ejemplo...")
			tickers := []Ticker{
				{Name: "AAPL", CurrentPrice: 195.50},
				{Name: "GOOGL", CurrentPrice: 2850.00},
				{Name: "MSFT", CurrentPrice: 340.80},
			}
			database.Create(&tickers)

			var aaplTicker, googlTicker, msftTicker Ticker
			database.Where("name = ?", "AAPL").First(&aaplTicker)
			database.Where("name = ?", "GOOGL").First(&googlTicker)
			database.Where("name = ?", "MSFT").First(&msftTicker)

			investments := []Investment{
				{TickerID: aaplTicker.ID, PurchaseDate: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC), Shares: 10, PurchasePrice: 150.75, OperationCost: 5.50},
				{TickerID: googlTicker.ID, PurchaseDate: time.Date(2023, 2, 20, 0, 0, 0, 0, time.UTC), Shares: 5, PurchasePrice: 2750.50, OperationCost: 12.00},
				{TickerID: msftTicker.ID, PurchaseDate: time.Date(2023, 3, 10, 0, 0, 0, 0, time.UTC), Shares: 8, PurchasePrice: 305.20, OperationCost: 7.25},
				{TickerID: aaplTicker.ID, PurchaseDate: time.Date(2023, 5, 22, 0, 0, 0, 0, time.UTC), Shares: 5, PurchasePrice: 172.25, OperationCost: 5.50},
			}
			database.Create(&investments)
		}
		return nil
	}

	log.Println("Migrando datos del esquema antiguo...")

	// 1. Migrar datos de market_data a tickers
	type OldMarketData struct {
		Ticker       string `gorm:"primaryKey"`
		CurrentPrice float64
	}

	var oldMarketData []OldMarketData
	database.Table("market_data").Find(&oldMarketData)

	tickerMap := make(map[string]uint) // mapa de nombre -> ID

	for _, md := range oldMarketData {
		ticker := Ticker{Name: md.Ticker, CurrentPrice: md.CurrentPrice}
		database.Create(&ticker)
		tickerMap[md.Ticker] = ticker.ID
		log.Printf("  Ticker migrado: %s (ID: %d)", md.Ticker, ticker.ID)
	}

	// 2. Verificar si hay columna 'ticker' en investments (esquema antiguo)
	if database.Migrator().HasColumn(&Investment{}, "ticker") {
		// Migrar investments: actualizar ticker_id basado en el nombre del ticker
		type OldInvestment struct {
			ID     uint
			Ticker string
		}
		var oldInvestments []OldInvestment
		database.Table("investments").Select("id, ticker").Find(&oldInvestments)

		for _, oi := range oldInvestments {
			if tickerID, ok := tickerMap[oi.Ticker]; ok {
				database.Table("investments").Where("id = ?", oi.ID).Update("ticker_id", tickerID)
			}
		}
		log.Printf("  Migradas %d inversiones", len(oldInvestments))

		// Eliminar columna ticker antigua de investments
		database.Migrator().DropColumn(&Investment{}, "ticker")
	}

	// 3. Verificar si hay columna 'ticker' en sales (esquema antiguo)
	if database.Migrator().HasColumn(&Sale{}, "ticker") {
		type OldSale struct {
			ID     uint
			Ticker string
		}
		var oldSales []OldSale
		database.Table("sales").Select("id, ticker").Find(&oldSales)

		for _, os := range oldSales {
			if tickerID, ok := tickerMap[os.Ticker]; ok {
				database.Table("sales").Where("id = ?", os.ID).Update("ticker_id", tickerID)
			}
		}
		log.Printf("  Migradas %d ventas", len(oldSales))

		// Eliminar columna ticker antigua de sales
		database.Migrator().DropColumn(&Sale{}, "ticker")
	}

	// 4. Eliminar tabla market_data antigua
	database.Migrator().DropTable("market_data")
	log.Println("  Tabla market_data eliminada")

	log.Println("Migración de datos completada")
	return nil
}

// migration003CreatePriceHistoryTable crea la tabla price_histories
func migration003CreatePriceHistoryTable(database *gorm.DB) error {
	log.Println("Creando tabla price_histories...")

	if !database.Migrator().HasTable("price_histories") {
		if err := database.AutoMigrate(&PriceHistory{}); err != nil {
			return err
		}
		log.Println("  Tabla price_histories creada exitosamente")

		// Crear índices para mejorar el rendimiento
		database.Exec("CREATE INDEX idx_price_histories_snapshot_id ON price_histories(snapshot_id)")
		database.Exec("CREATE INDEX idx_price_histories_ticker_id_created_at ON price_histories(ticker_id, created_at)")
		log.Println("  Índices creados en price_histories")
	} else {
		log.Println("  Tabla price_histories ya existe")
	}

	return nil
}

// migration004AddYahooFinanceTickerColumn agrega la columna yahoo_finance_ticker a la tabla tickers
func migration004AddYahooFinanceTickerColumn(database *gorm.DB) error {
	log.Println("Agregando columna yahoo_finance_ticker a tabla tickers...")

	if !database.Migrator().HasColumn(&Ticker{}, "YahooFinanceTicker") {
		if err := database.Migrator().AddColumn(&Ticker{}, "YahooFinanceTicker"); err != nil {
			return fmt.Errorf("error al agregar columna yahoo_finance_ticker: %v", err)
		}
		log.Println("  Columna yahoo_finance_ticker agregada exitosamente")
	} else {
		log.Println("  Columna yahoo_finance_ticker ya existe")
	}

	return nil
}

// migration006AddUsdEurColumn agrega la columna usdeur a la tabla tickers
func migration006AddUsdEurColumn(database *gorm.DB) error {
	log.Println("Agregando columna usdeur a tabla tickers...")

	if !database.Migrator().HasColumn(&Ticker{}, "YahooEur") {
		if err := database.Migrator().AddColumn(&Ticker{}, "YahooEur"); err != nil {
			return fmt.Errorf("error al agregar columna yahoo_eur: %v", err)
		}
		log.Println("  Columna yahoo_eur agregada exitosamente")
	} else {
		log.Println("  Columna yahoo_eur ya existe")
	}

	return nil
}

// migration008DropTaxColumn elimina la columna withheld_tax de la tabla sales
func migration008DropTaxColumn(database *gorm.DB) error {
	log.Println("Eliminando columna withheld_tax de tabla sales...")

	// Verificar si la columna existe en la tabla sales
	if database.Migrator().HasColumn("sales", "withheld_tax") {
		if err := database.Migrator().DropColumn("sales", "withheld_tax"); err != nil {
			return fmt.Errorf("error al eliminar columna withheld_tax: %v", err)
		}
		log.Println("  Columna withheld_tax eliminada exitosamente")
	} else {
		log.Println("  Columna withheld_tax no existe")
	}

	return nil
}

func getInvestmentData() ([]InvestmentView, []TickerSummaryView, []SaleView, float64, float64, float64, map[uint]float64, float64, float64, int, float64, error) {
	// 1. Obtener todos los tickers con sus precios y sus grupos
	var tickers []Ticker
	db.Preload("Group").Find(&tickers)

	tickerPrices := make(map[uint]float64)
	tickerNames := make(map[uint]string)
	tickerGroups := make(map[uint]string)
	for _, t := range tickers {
		tickerPrices[t.ID] = t.CurrentPrice
		tickerNames[t.ID] = t.Name
		if t.Group.Name != "" {
			tickerGroups[t.ID] = t.Group.Name
		}
	}

	// 2. Obtener todas las inversiones de la BD con preload del ticker
	var investments []Investment
	db.Preload("Ticker").Order("purchase_date desc").Find(&investments)

	// 3. Construir la vista detallada de inversiones y calcular totales
	var investmentViews []InvestmentView
	var totalCapital float64
	var netProfitLoss float64
	var totalOperationCost float64

	for _, i := range investments {
		currentPrice := tickerPrices[i.TickerID]
		tickerName := tickerNames[i.TickerID]
		investedCapital := i.Shares * i.PurchasePrice
		currentValue := i.Shares * currentPrice
		profitLoss := currentValue - (investedCapital + i.OperationCost)
		performance := 0.0
		if i.PurchasePrice > 0 {
			performance = (currentPrice - i.PurchasePrice) / i.PurchasePrice * 100
		}

		view := InvestmentView{
			ID:              i.ID,
			TickerID:        i.TickerID,
			Ticker:          tickerName,
			PurchaseDate:    i.PurchaseDate.Local().Format("02 Jan 2006 15:04"),
			Shares:          i.Shares,
			PurchasePrice:   i.PurchasePrice,
			OperationCost:   i.OperationCost,
			InvestedCapital: investedCapital,
			CurrentPrice:    currentPrice,
			CurrentValue:    currentValue,
			ProfitLoss:      profitLoss,
			Performance:     performance,
		}

		totalCapital += investedCapital + i.OperationCost
		totalOperationCost += i.OperationCost
		netProfitLoss += profitLoss
		investmentViews = append(investmentViews, view)
	}

	// 4. Construir la vista de resumen por ticker
	summaries := make(map[uint]*TickerSummaryView)
	for _, view := range investmentViews {
		summary, ok := summaries[view.TickerID]
		if !ok {
			summary = &TickerSummaryView{
				TickerID:  view.TickerID,
				Ticker:    view.Ticker,
				GroupName: tickerGroups[view.TickerID],
			}
			summaries[view.TickerID] = summary
		}

		summary.TotalShares += view.Shares
		summary.CurrentInvestment += view.InvestedCapital
		summary.TotalCost += view.OperationCost
		summary.CurrentValue += view.CurrentValue
		summary.ProfitLoss += view.ProfitLoss
	}

	// 5. Obtener todas las ventas de la BD con preload del ticker
	var sales []Sale
	db.Preload("Ticker").Order("sale_date desc").Find(&sales)

	// Calcular el monto total de ventas por ticker y agregar costos de operación de ventas
	tickerSalesAmount := make(map[uint]float64)
	for _, s := range sales {
		tickerSalesAmount[s.TickerID] += s.Shares * s.SalePrice
		totalOperationCost += s.OperationCost
		if summary, ok := summaries[s.TickerID]; ok {
			summary.TotalCost += s.OperationCost
		}
	}

	var summaryViews []TickerSummaryView
	for _, summary := range summaries {
		summaryViews = append(summaryViews, *summary)
	}

	// Calcular WAC (Weighted Average Cost) histórico para cada venta
	type Event struct {
		Date   time.Time
		Type   string // "buy", "sell"
		Shares float64
		Price  float64
		SaleID uint
	}

	tickerEvents := make(map[uint][]Event)

	// Agregar compras a eventos (sin incluir costos de operación)
	for _, inv := range investments {
		tickerEvents[inv.TickerID] = append(tickerEvents[inv.TickerID], Event{
			Date:   inv.PurchaseDate,
			Type:   "buy",
			Shares: inv.Shares,
			Price:  inv.PurchasePrice,
		})
	}

	// Agregar ventas a eventos
	for _, s := range sales {
		tickerEvents[s.TickerID] = append(tickerEvents[s.TickerID], Event{
			Date:   s.SaleDate,
			Type:   "sell",
			Shares: s.Shares,
			Price:  s.SalePrice,
			SaleID: s.ID,
		})
	}

	saleWACs := make(map[uint]float64)
	tickerFinalState := make(map[uint]struct {
		Shares  float64
		Capital float64
	})

	for tickerID, events := range tickerEvents {
		// Ordenar eventos por fecha
		sort.Slice(events, func(i, j int) bool {
			if events[i].Date.Equal(events[j].Date) {
				// Si la fecha es igual, procesar compras antes que ventas
				return events[i].Type == "buy"
			}
			return events[i].Date.Before(events[j].Date)
		})

		currentShares := 0.0
		currentCapital := 0.0

		for _, e := range events {
			if e.Type == "buy" {
				currentShares += e.Shares
				currentCapital += e.Shares * e.Price
			} else if e.Type == "sell" {
				wac := 0.0
				if currentShares > 0 {
					wac = currentCapital / currentShares
				}
				saleWACs[e.SaleID] = wac

				// Actualizar posición después de la venta (reducir capital proporcionalmente)
				currentCapital -= e.Shares * wac
				currentShares -= e.Shares
			}
		}
		// Guardar estado final del ticker
		tickerFinalState[tickerID] = struct {
			Shares  float64
			Capital float64
		}{currentShares, currentCapital}
	}

	// Calcular rendimiento del portafolio completo
	totalPortfolioCurrentValue := 0.0
	totalPortfolioWACValue := 0.0
	for tickerID, state := range tickerFinalState {
		if state.Shares > 0 {
			currentPrice := tickerPrices[tickerID]
			totalPortfolioCurrentValue += state.Shares * currentPrice
			totalPortfolioWACValue += state.Capital // Capital ya es shares * WAC
		}
	}
	portfolioPerformance := 0.0
	if totalPortfolioWACValue > 0 {
		portfolioPerformance = ((totalPortfolioCurrentValue - totalPortfolioWACValue) / totalPortfolioWACValue) * 100
	}
	portfolioUtility := totalPortfolioCurrentValue - totalPortfolioWACValue

	// Calcular utilidad de ventas por ticker
	tickerSalesProfit := make(map[uint]float64)
	for _, s := range sales {
		wac := saleWACs[s.ID]
		saleUtility := (s.SalePrice - wac) * s.Shares
		tickerSalesProfit[s.TickerID] += saleUtility
	}

	// Actualizar summaries con el cálculo correcto basado en WAC y utilidad de ventas
	for i := range summaryViews {
		tickerID := summaryViews[i].TickerID

		// Asignar utilidad de ventas si existe para este ticker
		if profit, ok := tickerSalesProfit[tickerID]; ok {
			summaryViews[i].SalesProfit = profit
		}

		if state, ok := tickerFinalState[tickerID]; ok && state.Shares > 0 {
			currentPrice := tickerPrices[tickerID]
			wac := 0.0
			if state.Shares > 0 {
				wac = state.Capital / state.Shares
			}
			// Utilidad = (Precio Actual * Acciones) - (WAC * Acciones)
			summaryViews[i].TotalShares = state.Shares
			summaryViews[i].CurrentInvestment = state.Capital // Inversión actual = Capital restante (Cost basis)
			summaryViews[i].CurrentValue = state.Shares * currentPrice
			summaryViews[i].ProfitLoss = (currentPrice * state.Shares) - (wac * state.Shares)
			// Rendimiento = ((Precio Actual - WAC) / WAC) * 100
			if wac > 0 {
				summaryViews[i].Performance = ((currentPrice - wac) / wac) * 100
			}
		} else {
			// Si no hay acciones en cartera, poner todo en 0 excepto lo ya calculado
			summaryViews[i].TotalShares = 0
			summaryViews[i].CurrentInvestment = 0
			summaryViews[i].CurrentValue = 0
			summaryViews[i].ProfitLoss = 0
			summaryViews[i].Performance = 0
		}

		// Calcular peso porcentual en la cartera
		if totalPortfolioCurrentValue > 0 {
			summaryViews[i].WeightPercentage = (summaryViews[i].CurrentValue / totalPortfolioCurrentValue) * 100
		} else {
			summaryViews[i].WeightPercentage = 0
		}
	}

	// Contar número de posiciones (tickers con acciones > 0)
	numPositions := 0
	for _, state := range tickerFinalState {
		if state.Shares > 0 {
			numPositions++
		}
	}

	// Ordenar summaryViews por peso porcentual (de mayor a menor)
	sort.Slice(summaryViews, func(i, j int) bool {
		return summaryViews[i].WeightPercentage > summaryViews[j].WeightPercentage
	})

	var saleViews []SaleView
	for _, s := range sales {
		tickerName := tickerNames[s.TickerID]
		currentPrice := tickerPrices[s.TickerID]
		totalSaleValue := s.Shares * s.SalePrice
		currentValue := s.Shares * currentPrice

		wac := saleWACs[s.ID]
		// Utilidad calculada solo con precios, sin costos de operación ni impuestos
		profit := (s.SalePrice - wac) * s.Shares
		performance := 0.0
		if s.SalePrice > 0 {
			performance = (currentPrice - s.SalePrice) / s.SalePrice * 100
		}
		// Proyección: diferencia entre monto actual y monto de venta
		projection := currentValue - totalSaleValue

		// Rendimiento de la venta vs WAC
		salePerformance := 0.0
		if wac > 0 {
			salePerformance = ((s.SalePrice - wac) / wac) * 100
		}
		// Utilidad de la venta
		saleUtility := (s.SalePrice - wac) * s.Shares

		view := SaleView{
			ID:              s.ID,
			TickerID:        s.TickerID,
			Ticker:          tickerName,
			SaleDate:        s.SaleDate.Local().Format("02 Jan 2006 15:04"),
			Shares:          s.Shares,
			SalePrice:       s.SalePrice,
			OperationCost:   s.OperationCost,
			TotalSaleValue:  totalSaleValue,
			CurrentPrice:    currentPrice,
			CurrentValue:    currentValue,
			Performance:     performance,
			Profit:          profit,
			Projection:      projection,
			WACAtSale:       wac,
			SalePerformance: salePerformance,
			SaleUtility:     saleUtility,
		}
		saleViews = append(saleViews, view)
	}

	return investmentViews, summaryViews, saleViews, totalCapital, netProfitLoss, totalOperationCost, tickerPrices, portfolioPerformance, portfolioUtility, numPositions, totalPortfolioCurrentValue, nil
}

// migration009CreateTickerGroupsTable crea la tabla ticker_groups y agrega la columna group_id a tickers
func migration009CreateTickerGroupsTable(database *gorm.DB) error {
	log.Println("Creando tabla ticker_groups y actualizando tabla tickers...")

	// Crear tabla ticker_groups si no existe
	if !database.Migrator().HasTable("ticker_groups") {
		if err := database.AutoMigrate(&TickerGroup{}); err != nil {
			return err
		}
		log.Println("  Tabla ticker_groups creada exitosamente")
	}

	// Agregar columna group_id a tickers si no existe
	if !database.Migrator().HasColumn(&Ticker{}, "GroupID") {
		if err := database.Migrator().AddColumn(&Ticker{}, "GroupID"); err != nil {
			return fmt.Errorf("error al agregar columna group_id: %v", err)
		}
		log.Println("  Columna group_id agregada exitosamente")

		// Agregar llave foránea manualmente si es necesario (GORM suele hacerlo con AutoMigrate pero AddColumn no)
		database.Exec("ALTER TABLE tickers ADD CONSTRAINT fk_tickers_group FOREIGN KEY (group_id) REFERENCES ticker_groups(id)")
	}

	return nil
}

// migration010AddActiveColumnToTickers agrega la columna active a la tabla tickers y la inicializa en true
func migration010AddActiveColumnToTickers(database *gorm.DB) error {
	log.Println("Agregando columna active a tabla tickers...")

	if !database.Migrator().HasColumn(&Ticker{}, "Active") {
		if err := database.Migrator().AddColumn(&Ticker{}, "Active"); err != nil {
			return fmt.Errorf("error al agregar columna active: %v", err)
		}
		log.Println("  Columna active agregada exitosamente")
	} else {
		log.Println("  Columna active ya existe")
	}

	// Asegurarse de que todos los registros actuales estén activos
	log.Println("Activando todos los tickers existentes...")
	if err := database.Model(&Ticker{}).Where("1=1").Update("active", true).Error; err != nil {
		return fmt.Errorf("error al activar tickers: %v", err)
	}

	return nil
}

// migration011RenameUsdEurToYahooEur renombra la columna y voltea el valor booleano
// Antes: true = USD, false = EUR. Ahora: true = EUR, false = USD.
func migration011RenameUsdEurToYahooEur(database *gorm.DB) error {
	log.Println("Renombrando columna usdeur a yahoo_eur e invirtiendo lógica...")

	// 1. Renombrar si existe la antigua y no la nueva
	if database.Migrator().HasColumn(&Ticker{}, "UsdEur") {
		// En PostgreSQL GORM suele usar snake_case: UsdEur -> usd_eur
		if err := database.Exec("ALTER TABLE tickers RENAME COLUMN usd_eur TO yahoo_eur").Error; err != nil {
			log.Printf("Aviso: No se pudo renombrar via SQL directo (tal vez ya se renombro): %v", err)
		} else {
			log.Println("  Columna renombrada a yahoo_eur")
		}

		// 2. Invertir los valores: true -> false (era USD, ahora no es EUR), false -> true (era EUR, ahora es EUR)
		if err := database.Exec("UPDATE tickers SET yahoo_eur = NOT yahoo_eur").Error; err != nil {
			return fmt.Errorf("error al invertir valores booleanos: %v", err)
		}
		log.Println("  Valores invertidos (True ahora significa EUR)")
	} else {
		log.Println("  La migración ya parece haber sido aplicada o la columna no existe")
	}

	return nil
}

// migration012FixYahooEurColumn asegura que la columna se llame yahoo_eur y los valores estén invertidos correctamente.
func migration012FixYahooEurColumn(database *gorm.DB) error {
	log.Println("Ejecutando migración 012: Asegurando columna yahoo_eur...")

	// 1. Verificar si existe la columna antigua usd_eur
	var hasUsdEur bool
	database.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'usd_eur')").Scan(&hasUsdEur)

	if hasUsdEur {
		log.Println("  Detectada columna antigua 'usd_eur'. Renombrando e invirtiendo valores...")
		if err := database.Exec("ALTER TABLE tickers RENAME COLUMN usd_eur TO yahoo_eur").Error; err != nil {
			log.Printf("Aviso: No se pudo renombrar via SQL directo (tal vez ya se renombro): %v", err)
		} else {
			// Invertir: lo que era USD (true) ahora es NO EUR (false). Lo que era EUR (false) ahora es EUR (true).
			if err := database.Exec("UPDATE tickers SET yahoo_eur = NOT yahoo_eur").Error; err != nil {
				return fmt.Errorf("error al invertir valores: %v", err)
			}
			log.Println("  Columna 'usd_eur' migrada a 'yahoo_eur' satisfactoriamente")
		}
	} else {
		// Verificar si existe yahoo_eur. Si no existe, crearla.
		var hasYahooEur bool
		database.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tickers' AND column_name = 'yahoo_eur')").Scan(&hasYahooEur)

		if !hasYahooEur {
			log.Println("  No se encontró 'usd_eur' ni 'yahoo_eur'. Creando 'yahoo_eur'...")
			if err := database.Migrator().AddColumn(&Ticker{}, "YahooEur"); err != nil {
				return err
			}
		} else {
			log.Println("  La columna 'yahoo_eur' ya existe.")
		}
	}

	return nil
}

// migration013CreateDividendsTable crea la tabla dividends
func migration013CreateDividendsTable(database *gorm.DB) error {
	log.Println("Creando tabla dividends...")
	if !database.Migrator().HasTable("dividends") {
		if err := database.AutoMigrate(&Dividend{}); err != nil {
			return fmt.Errorf("error al crear tabla dividends: %v", err)
		}
		log.Println("  Tabla dividends creada exitosamente")
	} else {
		log.Println("  Tabla dividends ya existe")
	}
	return nil
}

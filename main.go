package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

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

// Ticker representa un símbolo bursátil con su precio actual.
type Ticker struct {
	gorm.Model
	Name               string `gorm:"uniqueIndex"`
	CurrentPrice       float64
	YahooFinanceTicker string
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
	WithheldTax   float64
}

// PriceHistory representa un snapshot histórico de precio de un ticker.
type PriceHistory struct {
	gorm.Model
	SnapshotID string // UUID o timestamp para agrupar snapshots
	TickerID   uint
	Ticker     Ticker `gorm:"foreignKey:TickerID"`
	Price      float64
}

// --- VISTAS ---

// TickerView representa los datos de un ticker para mostrar en la UI.
type TickerView struct {
	ID                 uint
	Name               string
	CurrentPrice       float64
	UpdatedAt          string
	SnapshotChange     float64 // Cambio porcentual entre los últimos 2 snapshots
	HasSnapshotChange  bool    // Indica si hay datos suficientes para mostrar el cambio
	YahooFinanceTicker string
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
	WithheldTax     float64
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

func main() {
	// Cargar variables de entorno desde .env
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró archivo .env, usando variables de entorno del sistema")
	}

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

	// Ruta principal para mostrar los datos
	router.GET("/", func(c *gin.Context) {
		investments, summaries, sales, totalCapital, netProfitLoss, totalOperationCost, _, portfolioPerformance, portfolioUtility, numPositions, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		// Calcular utilidad neta de ventas
		totalSaleUtility := 0.0
		for _, s := range sales {
			totalSaleUtility += s.SaleUtility
		}

		// Calcular Valor de Salida: Utilidad Ventas + Utilidad Cartera - Costos de Operación - Número de Posiciones
		exitValue := totalSaleUtility + portfolioUtility - totalOperationCost - float64(numPositions)

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
			"NumPositions":         numPositions,
			"ExitValue":            exitValue,
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

		date, err := time.Parse("2006-01-02", input.Date)
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

		date, err := time.Parse("2006-01-02", input.Date)
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

	// Ruta para mostrar la página de resumen
	router.GET("/resumen", func(c *gin.Context) {
		_, summaries, _, _, _, _, _, _, _, _, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		c.HTML(http.StatusOK, "resumen.html", gin.H{
			"Summaries":  summaries,
			"ActivePage": "resumen",
		})
	})

	// Ruta para mostrar la página de compras
	router.GET("/compras", func(c *gin.Context) {
		investments, summaries, _, _, _, _, _, _, _, _, err := getInvestmentData()
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
		_, summaries, sales, _, _, _, _, _, _, _, err := getInvestmentData()
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

	// Ruta para mostrar la página de precios
	router.GET("/precios", func(c *gin.Context) {
		var tickers []Ticker
		db.Order("name").Find(&tickers)

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

		var tickerViews []TickerView
		for _, t := range tickers {
			changePtr := snapshotChanges[t.ID]
			hasChange := changePtr != nil
			changeVal := 0.0
			if hasChange {
				changeVal = *changePtr
			}

			tickerViews = append(tickerViews, TickerView{
				ID:                 t.ID,
				Name:               t.Name,
				CurrentPrice:       t.CurrentPrice,
				UpdatedAt:          t.UpdatedAt.Format("02 Jan 2006 15:04"),
				SnapshotChange:     changeVal,
				HasSnapshotChange:  hasChange,
				YahooFinanceTicker: t.YahooFinanceTicker,
			})
		}

		c.HTML(http.StatusOK, "precios.html", gin.H{
			"Tickers":    tickerViews,
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

		newTicker := Ticker{Name: name, CurrentPrice: price, YahooFinanceTicker: yahooTicker}
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

		if name == "" {
			c.String(http.StatusBadRequest, "El nombre del ticker es obligatorio.")
			return
		}

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			price = ticker.CurrentPrice
		}

		// Actualizar campos usando Select para incluir campos vacíos
		db.Model(&ticker).Select("Name", "CurrentPrice", "YahooFinanceTicker").Updates(Ticker{
			Name:               name,
			CurrentPrice:       price,
			YahooFinanceTicker: yahooTicker,
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

		purchaseDate, err := time.Parse("2006-01-02T15:04", purchaseDateStr)
		if err != nil {
			// Intentar formato sin hora para compatibilidad
			purchaseDate, err = time.Parse("2006-01-02", purchaseDateStr)
			if err != nil {
				c.String(http.StatusBadRequest, "Formato de fecha inválido.")
				return
			}
		}

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
		withheldTaxStr := strings.Replace(c.PostForm("withheld_tax"), ",", ".", -1)
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

		withheldTax, err := strconv.ParseFloat(withheldTaxStr, 64)
		if err != nil {
			withheldTax = 0 // Default to 0 if empty or invalid
		}

		saleDate, err := time.Parse("2006-01-02T15:04", saleDateStr)
		if err != nil {
			// Intentar formato DD/MM/YYYY para compatibilidad
			saleDate, err = time.Parse("02/01/2006", saleDateStr)
			if err != nil {
				c.String(http.StatusBadRequest, "Formato de fecha inválido.")
				return
			}
		}

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
			WithheldTax:   withheldTax,
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

	// Ruta para actualizar una venta
	router.POST("/update-sale/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.String(http.StatusBadRequest, "ID inválido.")
			return
		}

		var sale Sale
		if err := db.First(&sale, id).Error; err != nil {
			c.String(http.StatusNotFound, "Venta no encontrada.")
			return
		}

		// Parsear y validar datos del formulario
		tickerIDStr := c.PostForm("ticker_id")
		saleDateStr := c.PostForm("sale_date")
		sharesStr := strings.Replace(c.PostForm("shares"), ",", ".", -1)
		salePriceStr := strings.Replace(c.PostForm("sale_price"), ",", ".", -1)
		operationCostStr := strings.Replace(c.PostForm("operation_cost"), ",", ".", -1)
		withheldTaxStr := strings.Replace(c.PostForm("withheld_tax"), ",", ".", -1)
		redirectTo := c.PostForm("redirect_to")
		if redirectTo == "" {
			redirectTo = "/ventas"
		}

		tickerID, _ := strconv.Atoi(tickerIDStr)
		shares, _ := strconv.ParseFloat(sharesStr, 64)
		salePrice, _ := strconv.ParseFloat(salePriceStr, 64)
		operationCost, _ := strconv.ParseFloat(operationCostStr, 64)
		withheldTax, _ := strconv.ParseFloat(withheldTaxStr, 64)
		saleDate, err := time.Parse("2006-01-02T15:04", saleDateStr)
		if err != nil {
			saleDate, _ = time.Parse("02/01/2006", saleDateStr)
		}

		// Actualizar el registro
		db.Model(&sale).Updates(map[string]interface{}{
			"ticker_id":      tickerID,
			"sale_date":      saleDate,
			"shares":         shares,
			"sale_price":     salePrice,
			"operation_cost": operationCost,
			"withheld_tax":   withheldTax,
		})

		log.Printf("Registro de venta con ID %d actualizado", id)
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
				Date:   inv.PurchaseDate.Format("02 Jan 2006 15:04"),
				Shares: inv.Shares,
				Price:  inv.PurchasePrice,
				Total:  inv.Shares * inv.PurchasePrice,
			})
		}

		// Utilidad calculada solo con precios
		profit := (sale.SalePrice - wac) * sale.Shares

		c.JSON(http.StatusOK, gin.H{
			"ticker":        sale.Ticker.Name,
			"sale_date":     sale.SaleDate.Format("02 Jan 2006 15:04"),
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
		purchaseDate, err := time.Parse("2006-01-02T15:04", purchaseDateStr)
		if err != nil {
			purchaseDate, _ = time.Parse("2006-01-02", purchaseDateStr)
		}

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

		purchaseDate, err := time.Parse("2006-01-02T15:04", input.PurchaseDate)
		if err != nil {
			purchaseDate, _ = time.Parse("2006-01-02", input.PurchaseDate)
		}

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
			"purchase_date":    purchaseDate.Format("02 Jan 2006 15:04"),
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
			"purchase_date":  investment.PurchaseDate.Format("2006-01-02T15:04"),
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
			WithheldTax   float64 `json:"withheld_tax"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
			return
		}

		saleDate, err := time.Parse("2006-01-02T15:04", input.SaleDate)
		if err != nil {
			saleDate, _ = time.Parse("2006-01-02", input.SaleDate)
		}

		// Actualizar el registro
		db.Model(&sale).Updates(map[string]interface{}{
			"ticker_id":      input.TickerID,
			"sale_date":      saleDate,
			"shares":         input.Shares,
			"sale_price":     input.SalePrice,
			"operation_cost": input.OperationCost,
			"withheld_tax":   input.WithheldTax,
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
			"sale_date":        saleDate.Format("02 Jan 2006 15:04"),
			"shares":           input.Shares,
			"sale_price":       input.SalePrice,
			"operation_cost":   input.OperationCost,
			"withheld_tax":     input.WithheldTax,
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
			"sale_date":      sale.SaleDate.Format("2006-01-02T15:04"),
			"shares":         sale.Shares,
			"sale_price":     sale.SalePrice,
			"operation_cost": sale.OperationCost,
			"withheld_tax":   sale.WithheldTax,
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
				PurchaseDate:    i.PurchaseDate.Format("02 Jan 2006 15:04"),
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

		for _, e := range events {
			if e.Type == "buy" {
				currentShares += e.Shares
				currentCapital += e.Shares * e.Price
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
			}
		}

		// WAC final de las acciones en cartera
		portfolioWAC := 0.0
		if currentShares > 0 {
			portfolioWAC = currentCapital / currentShares
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
				SaleDate:        s.SaleDate.Format("02 Jan 2006 15:04"),
				Shares:          s.Shares,
				SalePrice:       s.SalePrice,
				OperationCost:   s.OperationCost,
				WithheldTax:     s.WithheldTax,
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

		// Obtener historial de precios del ticker
		var priceHistories []PriceHistory
		db.Where("ticker_id = ?", tickerID).Order("created_at asc").Find(&priceHistories)

		// Preparar datos para el gráfico
		var priceChartDates []string
		var priceChartValues []float64
		for _, ph := range priceHistories {
			priceChartDates = append(priceChartDates, ph.CreatedAt.Format("02 Jan 2006 15:04"))
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
			"PortfolioWAC":        portfolioWAC,
			"WACPerformance":      wacPerformance,
			"Utilidad":            utilidad,
			"TotalSaleUtility":    totalSaleUtility,
			"PriceChartDates":     priceChartDates,
			"PriceChartValues":    priceChartValues,
			"PurchaseChartDates":  purchaseChartDates,
			"PurchaseChartPrices": purchaseChartPrices,
			"SaleChartDates":      saleChartDates,
			"SaleChartPrices":     saleChartPrices,
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
			Select("DISTINCT snapshot_id, MIN(created_at) as created_at").
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

		// Obtener todas las inversiones y ventas
		var allInvestments []Investment
		db.Preload("Ticker").Order("purchase_date asc").Find(&allInvestments)

		var allSales []Sale
		db.Preload("Ticker").Order("sale_date asc").Find(&allSales)

		// Para cada snapshot, calcular la utilidad de la cartera en ese momento
		var dates []string
		var utilities []float64

		for _, snapshot := range snapshots {
			// Obtener los precios de este snapshot
			var priceHistories []PriceHistory
			db.Where("snapshot_id = ?", snapshot.SnapshotID).Find(&priceHistories)

			// Crear mapa de precios del snapshot
			snapshotPrices := make(map[uint]float64)
			for _, ph := range priceHistories {
				snapshotPrices[ph.TickerID] = ph.Price
			}

			// Filtrar inversiones y ventas hasta la fecha del snapshot
			type Event struct {
				Date   time.Time
				Type   string
				Shares float64
				Price  float64
			}

			tickerEvents := make(map[uint][]Event)

			// Agregar compras hasta la fecha del snapshot
			for _, inv := range allInvestments {
				if inv.PurchaseDate.Before(snapshot.CreatedAt) || inv.PurchaseDate.Equal(snapshot.CreatedAt) {
					tickerEvents[inv.TickerID] = append(tickerEvents[inv.TickerID], Event{
						Date:   inv.PurchaseDate,
						Type:   "buy",
						Shares: inv.Shares,
						Price:  inv.PurchasePrice,
					})
				}
			}

			// Agregar ventas hasta la fecha del snapshot
			for _, sale := range allSales {
				if sale.SaleDate.Before(snapshot.CreatedAt) || sale.SaleDate.Equal(snapshot.CreatedAt) {
					tickerEvents[sale.TickerID] = append(tickerEvents[sale.TickerID], Event{
						Date:   sale.SaleDate,
						Type:   "sell",
						Shares: sale.Shares,
						Price:  sale.SalePrice,
					})
				}
			}

			// Calcular el estado de la cartera en este snapshot
			totalUtility := 0.0

			for tickerID, events := range tickerEvents {
				// Ordenar eventos por fecha
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

				// Calcular utilidad para este ticker
				if currentShares > 0 {
					snapshotPrice, exists := snapshotPrices[tickerID]
					if exists {
						wac := 0.0
						if currentShares > 0 {
							wac = currentCapital / currentShares
						}
						utility := (snapshotPrice - wac) * currentShares
						totalUtility += utility
					}
				}
			}

			dates = append(dates, snapshot.CreatedAt.Format("02 Jan 2006 15:04"))
			utilities = append(utilities, totalUtility)
		}

		c.JSON(http.StatusOK, gin.H{
			"dates":     dates,
			"utilities": utilities,
		})
	})

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

	// Agregar parámetro para desactivar prepared statements (necesario para connection pooler de Supabase)
	if !strings.Contains(dsn, "prepared_statements") {
		if strings.Contains(dsn, "?") {
			dsn += "&prepare=false"
		} else {
			dsn += "?prepare=false"
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

func getInvestmentData() ([]InvestmentView, []TickerSummaryView, []SaleView, float64, float64, float64, map[uint]float64, float64, float64, int, error) {
	// 1. Obtener todos los tickers con sus precios
	var tickers []Ticker
	db.Find(&tickers)

	tickerPrices := make(map[uint]float64)
	tickerNames := make(map[uint]string)
	for _, t := range tickers {
		tickerPrices[t.ID] = t.CurrentPrice
		tickerNames[t.ID] = t.Name
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
			PurchaseDate:    i.PurchaseDate.Format("02 Jan 2006 15:04"),
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
			summary = &TickerSummaryView{TickerID: view.TickerID, Ticker: view.Ticker}
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

	// Calcular el monto total de ventas por ticker
	tickerSalesAmount := make(map[uint]float64)
	for _, s := range sales {
		tickerSalesAmount[s.TickerID] += s.Shares * s.SalePrice
	}

	// Restar el monto de ventas de CurrentInvestment
	for tickerID, salesAmount := range tickerSalesAmount {
		if summary, ok := summaries[tickerID]; ok {
			summary.CurrentInvestment -= salesAmount
		}
	}

	var summaryViews []TickerSummaryView
	for _, summary := range summaries {
		summaryViews = append(summaryViews, *summary)
	}

	// Ordenar summaryViews por nombre del ticker
	sort.Slice(summaryViews, func(i, j int) bool {
		return summaryViews[i].Ticker < summaryViews[j].Ticker
	})

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

	// Actualizar summaries con el cálculo correcto basado en WAC
	for i := range summaryViews {
		tickerID := summaryViews[i].TickerID
		if state, ok := tickerFinalState[tickerID]; ok && state.Shares > 0 {
			currentPrice := tickerPrices[tickerID]
			wac := 0.0
			if state.Shares > 0 {
				wac = state.Capital / state.Shares
			}
			// Utilidad = (Precio Actual * Acciones) - (WAC * Acciones)
			summaryViews[i].TotalShares = state.Shares
			summaryViews[i].CurrentValue = state.Shares * currentPrice
			summaryViews[i].ProfitLoss = (currentPrice * state.Shares) - (wac * state.Shares)
			// Rendimiento = ((Precio Actual - WAC) / WAC) * 100
			if wac > 0 {
				summaryViews[i].Performance = ((currentPrice - wac) / wac) * 100
			}
		} else {
			// Si no hay acciones en cartera, poner todo en 0
			summaryViews[i].TotalShares = 0
			summaryViews[i].CurrentValue = 0
			summaryViews[i].ProfitLoss = 0
			summaryViews[i].Performance = 0
		}
	}

	// Contar número de posiciones (tickers con acciones > 0)
	numPositions := 0
	for _, state := range tickerFinalState {
		if state.Shares > 0 {
			numPositions++
		}
	}

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
			SaleDate:        s.SaleDate.Format("02 Jan 2006 15:04"),
			Shares:          s.Shares,
			SalePrice:       s.SalePrice,
			OperationCost:   s.OperationCost,
			WithheldTax:     s.WithheldTax,
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

	return investmentViews, summaryViews, saleViews, totalCapital, netProfitLoss, totalOperationCost, tickerPrices, portfolioPerformance, portfolioUtility, numPositions, nil
}

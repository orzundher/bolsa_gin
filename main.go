package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// --- MODELOS DE GORM ---

// Investment representa una única compra de acciones en la BD.
type Investment struct {
	gorm.Model
	Ticker        string
	PurchaseDate  time.Time
	Shares        float64
	PurchasePrice float64
	OperationCost float64
}

// MarketData almacena el precio de mercado actual para un ticker.
type MarketData struct {
	Ticker       string `gorm:"primaryKey"`
	CurrentPrice float64
}

// --- VISTAS ---

// InvestmentView representa los datos de inversión que se mostrarán en la página.
type InvestmentView struct {
	ID              uint
	Ticker          string
	PurchaseDate    string
	Shares          float64
	PurchasePrice   float64
	OperationCost   float64
	InvestedCapital float64
	CurrentPrice    float64
	CurrentValue    float64
	ProfitLoss      float64
}

// TickerSummaryView representa un resumen de las inversiones por ticker.
type TickerSummaryView struct {
	Ticker          string
	TotalShares     float64
	InvestedCapital float64
	TotalCost       float64
	CurrentValue    float64
	ProfitLoss      float64
}

var db *gorm.DB

func main() {
	var err error
	// Configurar la base de datos con GORM
	db, err = setupDatabase()
	if err != nil {
		log.Fatalf("Error al configurar la base de datos: %v", err)
	}

	// Configurar Gin
	router := gin.Default()
	router.LoadHTMLGlob("templates/*")

	// Ruta principal para mostrar los datos
	router.GET("/", func(c *gin.Context) {
		investments, summaries, totalCapital, netProfitLoss, currentPrices, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"Investments":   investments,
			"Summaries":     summaries,
			"TotalCapital":  totalCapital,
			"NetProfitLoss": netProfitLoss,
			"CurrentPrices": currentPrices,
		})
	})

	// Ruta para actualizar los precios
	router.POST("/update-prices", func(c *gin.Context) {
		err := c.Request.ParseForm()
		if err != nil {
			c.String(http.StatusBadRequest, "Error al parsear el formulario: %v", err)
			return
		}

		for ticker, newPriceStr := range c.Request.PostForm {
			// Reemplazar coma por punto para asegurar el parseo correcto
			priceStrWithDot := strings.Replace(newPriceStr[0], ",", ".", -1)
			if newPrice, err := strconv.ParseFloat(priceStrWithDot, 64); err == nil {
				log.Printf("Actualizando precio para %s en la BD: %.2f", ticker, newPrice)
				db.Save(&MarketData{Ticker: ticker, CurrentPrice: newPrice})
			}
		}
		c.Redirect(http.StatusFound, "/")
	})

	// Ruta para registrar una nueva compra
	router.POST("/add-investment", func(c *gin.Context) {
		// Parsear valores del formulario
		ticker := strings.ToUpper(c.PostForm("ticker"))
		purchaseDateStr := c.PostForm("purchase_date")
		sharesStr := strings.Replace(c.PostForm("shares"), ",", ".", -1)
		purchasePriceStr := strings.Replace(c.PostForm("purchase_price"), ",", ".", -1)
		operationCostStr := strings.Replace(c.PostForm("operation_cost"), ",", ".", -1)

		// Validar y convertir tipos
		if ticker == "" || purchaseDateStr == "" {
			c.String(http.StatusBadRequest, "Ticker y fecha son obligatorios.")
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

		purchaseDate, err := time.Parse("2006-01-02", purchaseDateStr) // El formato de input type=date
		if err != nil {
			c.String(http.StatusBadRequest, "Formato de fecha inválido.")
			return
		}

		// Crear la nueva inversión
		newInvestment := Investment{
			Ticker:        ticker,
			PurchaseDate:  purchaseDate,
			Shares:        shares,
			PurchasePrice: purchasePrice,
			OperationCost: operationCost,
		}
		db.Create(&newInvestment)

		// Si el ticker es nuevo, añadirlo a MarketData con el precio de compra
		db.FirstOrCreate(&MarketData{Ticker: ticker}, &MarketData{Ticker: ticker, CurrentPrice: purchasePrice})

		log.Printf("Nueva compra registrada para %s", ticker)
		c.Redirect(http.StatusFound, "/")
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
		if err := db.First(&investment, id).Error; err != nil {
			c.String(http.StatusNotFound, "Registro no encontrado.")
			return
		}

		c.HTML(http.StatusOK, "edit.html", gin.H{
			"Investment": investment,
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
		ticker := strings.ToUpper(c.PostForm("ticker"))
		purchaseDateStr := c.PostForm("purchase_date")
		sharesStr := strings.Replace(c.PostForm("shares"), ",", ".", -1)
		purchasePriceStr := strings.Replace(c.PostForm("purchase_price"), ",", ".", -1)
		operationCostStr := strings.Replace(c.PostForm("operation_cost"), ",", ".", -1)

		shares, _ := strconv.ParseFloat(sharesStr, 64)
		purchasePrice, _ := strconv.ParseFloat(purchasePriceStr, 64)
		operationCost, _ := strconv.ParseFloat(operationCostStr, 64)
		purchaseDate, _ := time.Parse("2006-01-02", purchaseDateStr)

		// Actualizar el registro
		db.Model(&investment).Updates(map[string]interface{}{
			"ticker":         ticker,
			"purchase_date":  purchaseDate,
			"shares":         shares,
			"purchase_price": purchasePrice,
			"operation_cost": operationCost,
		})

		log.Printf("Registro de compra con ID %d actualizado", id)
		c.Redirect(http.StatusFound, "/")
	})

	// Ruta para eliminar una compra
	router.POST("/delete-investment", func(c *gin.Context) {
		idStr := c.PostForm("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.String(http.StatusBadRequest, "ID inválido.")
			return
		}

		// GORM usa borrado suave (soft delete) porque gorm.Model tiene el campo DeletedAt
		db.Delete(&Investment{}, id)

		log.Printf("Registro de compra con ID %d marcado como eliminado", id)
		c.Redirect(http.StatusFound, "/")
	})

	log.Println("Servidor iniciado en http://localhost:8080")
	router.Run(":8080")
}

func setupDatabase() (*gorm.DB, error) {
	database, err := gorm.Open(sqlite.Open("investments.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// AutoMigrate creará las tablas basadas en los structs de Go
	log.Println("Migrando base de datos...")
	database.AutoMigrate(&Investment{}, &MarketData{})

	// Insertar datos de ejemplo si la tabla está vacía
	var count int64
	database.Model(&Investment{}).Count(&count)
	if count == 0 {
		log.Println("Insertando datos de ejemplo en la base de datos...")
		investments := []Investment{
			{Ticker: "AAPL", PurchaseDate: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC), Shares: 10, PurchasePrice: 150.75, OperationCost: 5.50},
			{Ticker: "GOOGL", PurchaseDate: time.Date(2023, 2, 20, 0, 0, 0, 0, time.UTC), Shares: 5, PurchasePrice: 2750.50, OperationCost: 12.00},
			{Ticker: "MSFT", PurchaseDate: time.Date(2023, 3, 10, 0, 0, 0, 0, time.UTC), Shares: 8, PurchasePrice: 305.20, OperationCost: 7.25},
			{Ticker: "AAPL", PurchaseDate: time.Date(2023, 5, 22, 0, 0, 0, 0, time.UTC), Shares: 5, PurchasePrice: 172.25, OperationCost: 5.50},
		}
		database.Create(&investments)

		marketData := []MarketData{
			{Ticker: "AAPL", CurrentPrice: 195.50},
			{Ticker: "GOOGL", CurrentPrice: 2850.00},
			{Ticker: "MSFT", CurrentPrice: 340.80},
		}
		for _, md := range marketData {
            database.Save(&md) // Usamos Save para crear o actualizar
        }
	}

	return database, nil
}

func getInvestmentData() ([]InvestmentView, []TickerSummaryView, float64, float64, map[string]float64, error) {
	// 1. Obtener todos los precios de mercado de la BD
	var marketDataItems []MarketData
	db.Find(&marketDataItems)

	currentPrices := make(map[string]float64)
	for _, item := range marketDataItems {
		currentPrices[item.Ticker] = item.CurrentPrice
	}

	// 2. Obtener todas las inversiones de la BD
	var investments []Investment
	db.Order("purchase_date desc").Find(&investments)

	// 3. Construir la vista detallada y calcular totales
	var investmentViews []InvestmentView
	var totalCapital float64
	var netProfitLoss float64

	for _, i := range investments {
		currentPrice := currentPrices[i.Ticker]
		investedCapital := i.Shares * i.PurchasePrice
		currentValue := i.Shares * currentPrice
		profitLoss := currentValue - (investedCapital + i.OperationCost)

		view := InvestmentView{
			ID:              i.ID,
			Ticker:          i.Ticker,
			PurchaseDate:    i.PurchaseDate.Format("02 Jan 2006"),
			Shares:          i.Shares,
			PurchasePrice:   i.PurchasePrice,
			OperationCost:   i.OperationCost,
			InvestedCapital: investedCapital,
			CurrentPrice:    currentPrice,
			CurrentValue:    currentValue,
			ProfitLoss:      profitLoss,
		}

		totalCapital += investedCapital + i.OperationCost // El capital total sí debe incluir los costos
		netProfitLoss += profitLoss
		investmentViews = append(investmentViews, view)
	}

	// 4. Construir la vista de resumen por ticker
	summaries := make(map[string]*TickerSummaryView)
	for _, view := range investmentViews {
		summary, ok := summaries[view.Ticker]
		if !ok {
			summary = &TickerSummaryView{Ticker: view.Ticker}
			summaries[view.Ticker] = summary
		}

		summary.TotalShares += view.Shares
		summary.InvestedCapital += view.InvestedCapital
		summary.TotalCost += view.OperationCost
		summary.CurrentValue += view.CurrentValue
		summary.ProfitLoss += view.ProfitLoss
	}

	var summaryViews []TickerSummaryView
	for _, summary := range summaries {
		summaryViews = append(summaryViews, *summary)
	}

	return investmentViews, summaryViews, totalCapital, netProfitLoss, currentPrices, nil
}
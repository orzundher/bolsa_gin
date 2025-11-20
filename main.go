package main

import (
	"log"
	"net/http"
	"os"
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

// Sale representa una única venta de acciones en la BD.
type Sale struct {
	gorm.Model
	Ticker        string
	SaleDate      time.Time
	Shares        float64
	SalePrice     float64
	OperationCost float64
	WithheldTax   float64
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

// SaleView representa los datos de venta que se mostrarán en la página.
type SaleView struct {
	ID             uint
	Ticker         string
	SaleDate       string
	Shares         float64
	SalePrice      float64
	OperationCost  float64
	WithheldTax    float64
	TotalSaleValue float64
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
		investments, summaries, _, totalCapital, netProfitLoss, _, uniqueTickers, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"Investments":   investments,
			"Summaries":     summaries,
			"TotalCapital":  totalCapital,
			"NetProfitLoss": netProfitLoss,
			"UniqueTickers": uniqueTickers,
			"ActivePage":    "home",
		})
		log.Printf("Unique Tickers passed to template: %v", uniqueTickers)
	})

	// Ruta para mostrar la página de compras
	router.GET("/compras", func(c *gin.Context) {
		investments, _, _, _, _, _, uniqueTickers, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		c.HTML(http.StatusOK, "compras.html", gin.H{
			"Investments":   investments,
			"UniqueTickers": uniqueTickers,
			"ActivePage":    "compras",
		})
	})

	// Ruta para mostrar la página de ventas
	router.GET("/ventas", func(c *gin.Context) {
		_, _, sales, _, _, _, uniqueTickers, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		c.HTML(http.StatusOK, "ventas.html", gin.H{
			"Sales":         sales,
			"UniqueTickers": uniqueTickers,
			"ActivePage":    "ventas",
		})
	})

	// Ruta para mostrar la página de precios
	router.GET("/precios", func(c *gin.Context) {
		_, _, _, _, _, currentPrices, _, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		c.HTML(http.StatusOK, "precios.html", gin.H{
			"CurrentPrices": currentPrices,
			"ActivePage":    "precios",
		})
	})

	// Ruta para actualizar los precios
	router.POST("/update-prices", func(c *gin.Context) {
		err := c.Request.ParseForm()
		if err != nil {
			c.String(http.StatusBadRequest, "Error al parsear el formulario: %v", err)
			return
		}

		redirectTo := c.PostForm("redirect_to")
		if redirectTo == "" {
			redirectTo = "/"
		}

		for ticker, newPriceStr := range c.Request.PostForm {
			// Ignorar el campo redirect_to
			if ticker == "redirect_to" {
				continue
			}
			// Reemplazar coma por punto para asegurar el parseo correcto
			priceStrWithDot := strings.Replace(newPriceStr[0], ",", ".", -1)
			if newPrice, err := strconv.ParseFloat(priceStrWithDot, 64); err == nil {
				log.Printf("Actualizando precio para %s en la BD: %.2f", ticker, newPrice)
				db.Save(&MarketData{Ticker: ticker, CurrentPrice: newPrice})
			}
		}
		c.Redirect(http.StatusFound, redirectTo)
	})

	// Ruta para registrar una nueva compra
	router.POST("/add-investment", func(c *gin.Context) {
		// Parsear valores del formulario
		ticker := strings.ToUpper(c.PostForm("ticker"))
		purchaseDateStr := c.PostForm("purchase_date")
		sharesStr := strings.Replace(c.PostForm("shares"), ",", ".", -1)
		purchasePriceStr := strings.Replace(c.PostForm("purchase_price"), ",", ".", -1)
		operationCostStr := strings.Replace(c.PostForm("operation_cost"), ",", ".", -1)
		redirectTo := c.PostForm("redirect_to")
		if redirectTo == "" {
			redirectTo = "/"
		}

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
		c.Redirect(http.StatusFound, redirectTo)
	})

	// Ruta para registrar una nueva venta
	router.POST("/add-sale", func(c *gin.Context) {
		// Parsear valores del formulario
		ticker := strings.ToUpper(c.PostForm("ticker"))
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
		if ticker == "" || saleDateStr == "" {
			c.String(http.StatusBadRequest, "Ticker y fecha de venta son obligatorios.")
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

		saleDate, err := time.Parse("2006-01-02", saleDateStr) // El formato de input type=date
		if err != nil {
			c.String(http.StatusBadRequest, "Formato de fecha inválido.")
			return
		}

		// Crear la nueva venta
		newSale := Sale{
			Ticker:        ticker,
			SaleDate:      saleDate,
			Shares:        shares,
			SalePrice:     salePrice,
			OperationCost: operationCost,
			WithheldTax:   withheldTax,
		}
		db.Create(&newSale)

		log.Printf("Nueva venta registrada para %s", ticker)
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("Servidor iniciado en http://localhost:%s", port)
	router.Run(":" + port)
}

func setupDatabase() (*gorm.DB, error) {
	database, err := gorm.Open(sqlite.Open("investments.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// AutoMigrate creará las tablas basadas en los structs de Go
	log.Println("Migrando base de datos...")
	database.AutoMigrate(&Investment{}, &MarketData{}, &Sale{})

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

func getInvestmentData() ([]InvestmentView, []TickerSummaryView, []SaleView, float64, float64, map[string]float64, []string, error) {
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

	// 3. Construir la vista detallada de inversiones y calcular totales
	var investmentViews []InvestmentView
	var totalCapital float64
	var netProfitLoss float64

	// Para almacenar tickers únicos
	uniqueTickersMap := make(map[string]bool)

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
		uniqueTickersMap[i.Ticker] = true
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

	// 5. Obtener todas las ventas de la BD y construir la vista detallada
	var sales []Sale
	db.Order("sale_date desc").Find(&sales)

	var saleViews []SaleView
	for _, s := range sales {
		totalSaleValue := s.Shares * s.SalePrice
		view := SaleView{
			ID:             s.ID,
			Ticker:         s.Ticker,
			SaleDate:       s.SaleDate.Format("02 Jan 2006"),
			Shares:         s.Shares,
			SalePrice:      s.SalePrice,
			OperationCost:  s.OperationCost,
			WithheldTax:    s.WithheldTax,
			TotalSaleValue: totalSaleValue,
		}
		saleViews = append(saleViews, view)
	}

	// Convertir el mapa de tickers únicos a un slice
	var uniqueTickers []string
	for ticker := range uniqueTickersMap {
		uniqueTickers = append(uniqueTickers, ticker)
	}

	return investmentViews, summaryViews, saleViews, totalCapital, netProfitLoss, currentPrices, uniqueTickers, nil
}

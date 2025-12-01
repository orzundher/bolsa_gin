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
	Name         string `gorm:"uniqueIndex"`
	CurrentPrice float64
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

// --- VISTAS ---

// TickerView representa los datos de un ticker para mostrar en la UI.
type TickerView struct {
	ID           uint
	Name         string
	CurrentPrice float64
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
	TickerID       uint
	Ticker         string
	SaleDate       string
	Shares         float64
	SalePrice      float64
	OperationCost  float64
	WithheldTax    float64
	TotalSaleValue float64
	Performance    float64
	Profit         float64
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
		investments, summaries, _, totalCapital, netProfitLoss, totalOperationCost, _, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"Investments":        investments,
			"Summaries":          summaries,
			"TotalCapital":       totalCapital,
			"NetProfitLoss":      netProfitLoss,
			"TotalOperationCost": totalOperationCost,
			"ActivePage":         "home",
		})
	})

	// Ruta para mostrar la página de resumen
	router.GET("/resumen", func(c *gin.Context) {
		_, summaries, _, _, _, _, _, err := getInvestmentData()
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
		investments, _, _, _, _, _, _, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		// Obtener todos los tickers disponibles
		var tickers []Ticker
		db.Order("name").Find(&tickers)
		var tickerViews []TickerView
		for _, t := range tickers {
			tickerViews = append(tickerViews, TickerView{ID: t.ID, Name: t.Name, CurrentPrice: t.CurrentPrice})
		}

		c.HTML(http.StatusOK, "compras.html", gin.H{
			"Investments": investments,
			"Tickers":     tickerViews,
			"ActivePage":  "compras",
		})
	})

	// Ruta para mostrar la página de ventas
	router.GET("/ventas", func(c *gin.Context) {
		_, _, sales, _, _, _, _, err := getInvestmentData()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al obtener los datos: %v", err)
			return
		}

		// Obtener todos los tickers disponibles
		var tickers []Ticker
		db.Order("name").Find(&tickers)
		var tickerViews []TickerView
		for _, t := range tickers {
			tickerViews = append(tickerViews, TickerView{ID: t.ID, Name: t.Name, CurrentPrice: t.CurrentPrice})
		}

		c.HTML(http.StatusOK, "ventas.html", gin.H{
			"Sales":      sales,
			"Tickers":    tickerViews,
			"ActivePage": "ventas",
		})
	})

	// Ruta para mostrar la página de precios
	router.GET("/precios", func(c *gin.Context) {
		var tickers []Ticker
		db.Order("name").Find(&tickers)

		var tickerViews []TickerView
		for _, t := range tickers {
			tickerViews = append(tickerViews, TickerView{
				ID:           t.ID,
				Name:         t.Name,
				CurrentPrice: t.CurrentPrice,
			})
		}

		c.HTML(http.StatusOK, "precios.html", gin.H{
			"Tickers":    tickerViews,
			"ActivePage": "precios",
		})
	})

	// Ruta para agregar un nuevo ticker
	router.POST("/add-ticker", func(c *gin.Context) {
		name := strings.ToUpper(c.PostForm("name"))
		priceStr := strings.Replace(c.PostForm("current_price"), ",", ".", -1)

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

		newTicker := Ticker{Name: name, CurrentPrice: price}
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

		if name == "" {
			c.String(http.StatusBadRequest, "El nombre del ticker es obligatorio.")
			return
		}

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			price = ticker.CurrentPrice
		}

		db.Model(&ticker).Updates(map[string]interface{}{
			"name":          name,
			"current_price": price,
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

		purchaseDate, err := time.Parse("2006-01-02", purchaseDateStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Formato de fecha inválido.")
			return
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

		saleDate, err := time.Parse("02/01/2006", saleDateStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Formato de fecha inválido. Use DD/MM/YYYY")
			return
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
		saleDate, _ := time.Parse("02/01/2006", saleDateStr)

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
				Date:   inv.PurchaseDate.Format("02 Jan 2006"),
				Shares: inv.Shares,
				Price:  inv.PurchasePrice,
				Total:  inv.Shares * inv.PurchasePrice,
			})
		}

		// Utilidad calculada solo con precios
		profit := (sale.SalePrice - wac) * sale.Shares

		c.JSON(http.StatusOK, gin.H{
			"ticker":        sale.Ticker.Name,
			"sale_date":     sale.SaleDate.Format("02 Jan 2006"),
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
		purchaseDate, _ := time.Parse("2006-01-02", purchaseDateStr)

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
		"001_create_initial_schema":       migration001CreateInitialSchema,
		"002_migrate_to_ticker_id_schema": migration002MigrateToTickerIDSchema,
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

func getInvestmentData() ([]InvestmentView, []TickerSummaryView, []SaleView, float64, float64, float64, map[uint]float64, error) {
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

		view := InvestmentView{
			ID:              i.ID,
			TickerID:        i.TickerID,
			Ticker:          tickerName,
			PurchaseDate:    i.PurchaseDate.Format("02 Jan 2006"),
			Shares:          i.Shares,
			PurchasePrice:   i.PurchasePrice,
			OperationCost:   i.OperationCost,
			InvestedCapital: investedCapital,
			CurrentPrice:    currentPrice,
			CurrentValue:    currentValue,
			ProfitLoss:      profitLoss,
		}

		totalCapital += investedCapital + i.OperationCost
		totalOperationCost += i.OperationCost
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

	// 5. Obtener todas las ventas de la BD con preload del ticker
	var sales []Sale
	db.Preload("Ticker").Order("sale_date desc").Find(&sales)

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

	for _, events := range tickerEvents {
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
	}

	var saleViews []SaleView
	for _, s := range sales {
		tickerName := tickerNames[s.TickerID]
		totalSaleValue := s.Shares * s.SalePrice

		wac := saleWACs[s.ID]
		// Utilidad calculada solo con precios, sin costos de operación ni impuestos
		profit := (s.SalePrice - wac) * s.Shares
		performance := 0.0
		if wac > 0 {
			performance = (s.SalePrice - wac) / wac * 100
		}

		view := SaleView{
			ID:             s.ID,
			TickerID:       s.TickerID,
			Ticker:         tickerName,
			SaleDate:       s.SaleDate.Format("02 Jan 2006"),
			Shares:         s.Shares,
			SalePrice:      s.SalePrice,
			OperationCost:  s.OperationCost,
			WithheldTax:    s.WithheldTax,
			TotalSaleValue: totalSaleValue,
			Performance:    performance,
			Profit:         profit,
		}
		saleViews = append(saleViews, view)
	}

	return investmentViews, summaryViews, saleViews, totalCapital, netProfitLoss, totalOperationCost, tickerPrices, nil
}

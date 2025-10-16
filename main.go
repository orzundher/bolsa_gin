package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// Transaction representa una transacción de compra de acciones
type Transaction struct {
	ID            int
	Symbol        string
	Shares        int
	PurchasePrice float64
}

// PortfolioItem representa un resumen de una acción en el portafolio
type PortfolioItem struct {
	Symbol          string
	Shares          int
	PurchasePrice   float64
	CapitalInvested float64
	CurrentPrice    float64
	CurrentValue    float64
	ProfitLoss      float64
}

var db *sql.DB

// initDB inicializa la base de datos SQLite
func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "./portfolio.db")
	if err != nil {
		return err
	}

	// Crear tabla de transacciones
	createTableSQL := `CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		symbol TEXT NOT NULL,
		shares INTEGER NOT NULL,
		purchase_price REAL NOT NULL
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return err
	}

	// Verificar si hay datos
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&count)
	if err != nil {
		return err
	}

	// Si no hay datos, insertar datos de ejemplo
	if count == 0 {
		err = insertSampleData()
		if err != nil {
			return err
		}
	}

	return nil
}

// insertSampleData inserta datos de ejemplo en la base de datos
func insertSampleData() error {
	sampleData := []Transaction{
		{Symbol: "AAPL", Shares: 10, PurchasePrice: 150.00},
		{Symbol: "GOOGL", Shares: 5, PurchasePrice: 2800.00},
		{Symbol: "MSFT", Shares: 15, PurchasePrice: 300.00},
		{Symbol: "TSLA", Shares: 8, PurchasePrice: 700.00},
		{Symbol: "AMZN", Shares: 12, PurchasePrice: 3200.00},
	}

	for _, t := range sampleData {
		_, err := db.Exec("INSERT INTO transactions (symbol, shares, purchase_price) VALUES (?, ?, ?)",
			t.Symbol, t.Shares, t.PurchasePrice)
		if err != nil {
			return err
		}
	}

	return nil
}

// getSimulatedPrice simula el precio actual de una acción
func getSimulatedPrice(symbol string) float64 {
	// Precios simulados de mercado
	prices := map[string]float64{
		"AAPL":  175.00,
		"GOOGL": 2900.00,
		"MSFT":  330.00,
		"TSLA":  750.00,
		"AMZN":  3300.00,
	}

	if price, ok := prices[symbol]; ok {
		return price
	}
	return 100.00 // Precio por defecto
}

// getTransactions obtiene todas las transacciones de la base de datos
func getTransactions() ([]Transaction, error) {
	rows, err := db.Query("SELECT id, symbol, shares, purchase_price FROM transactions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		err := rows.Scan(&t.ID, &t.Symbol, &t.Shares, &t.PurchasePrice)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	return transactions, nil
}

// calculatePortfolio calcula el resumen del portafolio
func calculatePortfolio() ([]PortfolioItem, float64, error) {
	transactions, err := getTransactions()
	if err != nil {
		return nil, 0, err
	}

	var portfolio []PortfolioItem
	var totalCapital float64

	for _, t := range transactions {
		currentPrice := getSimulatedPrice(t.Symbol)
		capitalInvested := float64(t.Shares) * t.PurchasePrice
		currentValue := float64(t.Shares) * currentPrice
		profitLoss := currentValue - capitalInvested

		portfolio = append(portfolio, PortfolioItem{
			Symbol:          t.Symbol,
			Shares:          t.Shares,
			PurchasePrice:   t.PurchasePrice,
			CapitalInvested: capitalInvested,
			CurrentPrice:    currentPrice,
			CurrentValue:    currentValue,
			ProfitLoss:      profitLoss,
		})

		totalCapital += capitalInvested
	}

	return portfolio, totalCapital, nil
}

func main() {
	// Inicializar la base de datos
	err := initDB()
	if err != nil {
		log.Fatal("Error al inicializar la base de datos:", err)
	}
	defer db.Close()

	// Configurar Gin
	r := gin.Default()

	// Cargar plantillas HTML
	r.SetFuncMap(template.FuncMap{
		"formatMoney": func(value float64) string {
			return fmt.Sprintf("$%.2f", value)
		},
		"isPositive": func(value float64) bool {
			return value >= 0
		},
	})
	r.LoadHTMLGlob("templates/*")

	// Ruta principal
	r.GET("/", func(c *gin.Context) {
		portfolio, totalCapital, err := calculatePortfolio()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error al calcular el portafolio")
			return
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"Portfolio":    portfolio,
			"TotalCapital": totalCapital,
		})
	})

	// Iniciar el servidor
	fmt.Println("Servidor iniciado en http://localhost:8080")
	r.Run(":8080")
}

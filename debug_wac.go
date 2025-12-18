package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Copiamos las estructuras necesarias
type Ticker struct {
	gorm.Model
	Name               string
	CurrentPrice       float64
	YahooFinanceTicker string
	UsdEur             bool
}

type Investment struct {
	gorm.Model
	TickerID      uint
	PurchaseDate  string // Simplificado para este script
	Shares        float64
	PurchasePrice float64
	OperationCost float64
}

type Sale struct {
	gorm.Model
	TickerID  uint
	SaleDate  string
	Shares    float64
	SalePrice float64
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}

	// Fix for supabase connection pooling if needed (copied from main.go)
	if !strings.Contains(dsn, "prepared_statements") {
		if strings.Contains(dsn, "?") {
			dsn += "&prepare=false"
		} else {
			dsn += "?prepare=false"
		}
	}

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Buscar el ticker
	var tickers []Ticker
	// Buscamos algo que parezca "gold"
	db.Where("name ILIKE ?", "%gold%").Find(&tickers)

	for _, t := range tickers {
		fmt.Printf("\n=== ANALYSIS FOR TICKER: %s (ID: %d) ===\n", t.Name, t.ID)
		fmt.Printf("Current Price: %f, UsdEur: %v\n", t.CurrentPrice, t.UsdEur)

		var investments []Investment
		db.Where("ticker_id = ?", t.ID).Order("purchase_date asc").Find(&investments)

		fmt.Println("\n--- INVESTMENTS ---")
		totalShares := 0.0
		weightedSum := 0.0
		for _, inv := range investments {
			cost := inv.Shares * inv.PurchasePrice
			fmt.Printf("ID: %d, Shares: %f, Price: %f, TotalCost: %f\n", inv.ID, inv.Shares, inv.PurchasePrice, cost)
			totalShares += inv.Shares
			weightedSum += cost
		}

		fmt.Printf("\nRaw Average Cost (TotalCost / TotalShares): %f / %f = %f\n", weightedSum, totalShares, weightedSum/totalShares)

		// Check Sales
		var sales []Sale
		db.Where("ticker_id = ?", t.ID).Find(&sales)
		if len(sales) > 0 {
			fmt.Println("\n--- SALES ---")
			for _, s := range sales {
				fmt.Printf("ID: %d, Shares: %f, Price: %f\n", s.ID, s.Shares, s.SalePrice)
			}
		} else {
			fmt.Println("\nNo sales found.")
		}
	}
}

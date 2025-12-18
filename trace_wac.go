package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Investment struct {
	gorm.Model
	TickerID      uint
	PurchaseDate  time.Time
	Shares        float64
	PurchasePrice float64
}

type Sale struct {
	gorm.Model
	TickerID  uint
	SaleDate  time.Time
	Shares    float64
	SalePrice float64
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env")
	}
	dsn := os.Getenv("DATABASE_URL")
	if !strings.Contains(dsn, "prepared_statements") {
		if strings.Contains(dsn, "?") {
			dsn += "&prepare=false"
		} else {
			dsn += "?prepare=false"
		}
	}
	db, _ := gorm.Open(postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true}), &gorm.Config{})

	// ID 18 is Physical Swiss Gold based on previous output
	tickerID := 18

	var investments []Investment
	db.Where("ticker_id = ?", tickerID).Find(&investments)
	var sales []Sale
	db.Where("ticker_id = ?", tickerID).Find(&sales)

	type Event struct {
		Date   time.Time
		Type   string
		Shares float64
		Price  float64
	}
	var events []Event
	for _, i := range investments {
		events = append(events, Event{Date: i.PurchaseDate, Type: "buy", Shares: i.Shares, Price: i.PurchasePrice})
	}
	for _, s := range sales {
		events = append(events, Event{Date: s.SaleDate, Type: "sell", Shares: s.Shares, Price: s.SalePrice})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Date.Before(events[j].Date)
	})

	fmt.Println("Date | Type | Shares | Price | CurrentShares | CurrentCapital | WAC")
	currShares := 0.0
	currCap := 0.0
	for _, e := range events {
		if e.Type == "buy" {
			currShares += e.Shares
			currCap += e.Shares * e.Price
		} else {
			wac := 0.0
			if currShares > 0 {
				wac = currCap / currShares
			}
			currCap -= e.Shares * wac
			currShares -= e.Shares
		}

		wacDisplay := 0.0
		if currShares > 0 {
			wacDisplay = currCap / currShares
		}

		fmt.Printf("%s | %s | %.4f | %.2f | %.4f | %.2f | %.4f\n",
			e.Date.Format("2006-01-02"), e.Type, e.Shares, e.Price, currShares, currCap, wacDisplay)
	}
}

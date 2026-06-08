package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Ticker struct {
	ID           uint
	Name         string
	CurrentPrice float64
}

type Investment struct {
	ID            uint
	TickerID      uint
	Shares        float64
	PurchasePrice float64
	PurchaseDate  time.Time
}

type Sale struct {
	ID        uint
	TickerID  uint
	Shares    float64
	SalePrice float64
	SaleDate  time.Time
}

type PriceHistory struct {
	ID         uint
	SnapshotID string
	TickerID   uint
	Price      float64
	CreatedAt  time.Time
}

func main() {
	godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	// 1. Current Utility
	var tickers []Ticker
	db.Find(&tickers)
	tickerPrices := make(map[uint]float64)
	for _, t := range tickers {
		tickerPrices[t.ID] = t.CurrentPrice
	}

	var investments []Investment
	db.Find(&investments)
	var sales []Sale
	db.Find(&sales)

	type Event struct {
		Date   time.Time
		Type   string
		Ticker uint
		Shares float64
		Price  float64
	}
	var allEvents []Event
	for _, inv := range investments {
		allEvents = append(allEvents, Event{inv.PurchaseDate, "buy", inv.TickerID, inv.Shares, inv.PurchasePrice})
	}
	for _, s := range sales {
		allEvents = append(allEvents, Event{s.SaleDate, "sell", s.TickerID, s.Shares, s.SalePrice})
	}
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Date.Before(allEvents[j].Date)
	})

	type TickerState struct {
		Shares  float64
		Capital float64
	}
	states := make(map[uint]TickerState)
	for _, e := range allEvents {
		state := states[e.Ticker]
		if e.Type == "buy" {
			state.Shares += e.Shares
			state.Capital += e.Shares * e.Price
		} else {
			wac := 0.0
			if state.Shares > 0 {
				wac = state.Capital / state.Shares
			}
			state.Capital -= e.Shares * wac
			state.Shares -= e.Shares
		}
		states[e.Ticker] = state
	}

	currentUtility := 0.0
	for id, state := range states {
		if state.Shares > 0 {
			currentUtility += (tickerPrices[id] * state.Shares) - state.Capital
		}
	}
	fmt.Printf("Current Card Utility: %.2f\n", currentUtility)

	// 2. Last Snapshot Utility
	var lastPH PriceHistory
	db.Order("created_at desc").First(&lastPH)
	snapshotID := lastPH.SnapshotID
	
	var snapshotPHs []PriceHistory
	db.Where("snapshot_id = ?", snapshotID).Find(&snapshotPHs)
	
	var lastSnapshotDate time.Time
	db.Model(&PriceHistory{}).Where("snapshot_id = ?", snapshotID).Select("MIN(created_at)").Scan(&lastSnapshotDate)
	
	statesAtSnapshot := make(map[uint]TickerState)
	for _, e := range allEvents {
		if e.Date.After(lastSnapshotDate) {
			continue
		}
		state := statesAtSnapshot[e.Ticker]
		if e.Type == "buy" {
			state.Shares += e.Shares
			state.Capital += e.Shares * e.Price
		} else {
			wac := 0.0
			if state.Shares > 0 {
				wac = state.Capital / state.Shares
			}
			state.Capital -= e.Shares * wac
			state.Shares -= e.Shares
		}
		statesAtSnapshot[e.Ticker] = state
	}

	snapshotUtility := 0.0
	for _, ph := range snapshotPHs {
		state := statesAtSnapshot[ph.TickerID]
		if state.Shares > 0 {
			wac := state.Capital / state.Shares
			snapshotUtility += (ph.Price - wac) * state.Shares
		}
	}
	fmt.Printf("Last Snapshot Utility (%s): %.2f\n", snapshotID, snapshotUtility)
	fmt.Printf("Snapshot Date: %v\n", lastSnapshotDate)
}

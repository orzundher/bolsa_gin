package main

import (
	"time"

	"gorm.io/gorm"
)

// Note representa una nota o recordatorio guardado en la BD
type Note struct {
	gorm.Model
	Date    time.Time
	Content string
}

// TickerNote representa una nota específica para un ticker
type TickerNote struct {
	gorm.Model
	TickerID uint      `gorm:"index"`
	Ticker   Ticker    `gorm:"foreignKey:TickerID"`
	Date     time.Time `gorm:"index"`
	Content  string
}

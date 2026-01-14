package main

import (
	"log"
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

// migration005CreateNotesTable crea la tabla notes
func migration005CreateNotesTable(database *gorm.DB) error {
	log.Println("Creando tabla notes...")

	if !database.Migrator().HasTable("notes") {
		if err := database.AutoMigrate(&Note{}); err != nil {
			return err
		}
		log.Println("  Tabla notes creada exitosamente")
	} else {
		log.Println("  Tabla notes ya existe")
	}

	return nil
}

// migration007CreateTickerNotesTable crea la tabla ticker_notes
func migration007CreateTickerNotesTable(database *gorm.DB) error {
	log.Println("Creando tabla ticker_notes...")

	if !database.Migrator().HasTable("ticker_notes") {
		if err := database.AutoMigrate(&TickerNote{}); err != nil {
			return err
		}
		log.Println("  Tabla ticker_notes creada exitosamente")
	} else {
		log.Println("  Tabla ticker_notes ya existe")
	}

	return nil
}

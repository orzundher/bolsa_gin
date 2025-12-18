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

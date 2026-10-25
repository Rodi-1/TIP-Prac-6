package main

import (
	"log"
	"net/http"

	"example.com/pz6-gorm/internal/db"
	httpapi "example.com/pz6-gorm/internal/http"
	"example.com/pz6-gorm/internal/models"
)

func main() {
	d, err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, err := d.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
	if err := d.AutoMigrate(&models.User{}, &models.Note{}, &models.Tag{}); err != nil {
		log.Fatal("migrate: ", err)
	}
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", httpapi.BuildRouter(d)))
}

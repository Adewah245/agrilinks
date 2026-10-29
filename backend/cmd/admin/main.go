package main

import (
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/repository"
	"backend/internal/services"
	"context"
	"flag"
	"log"
	"time"
)

func main() {
	phone := flag.String("phone", "", "Phone of the existing account to promote to the first Super Admin")
	flag.Parse()
	normalized := services.NormalizeNigerianPhone(*phone)
	if normalized == "" {
		log.Fatal("provide a valid existing account phone with -phone")
	}
	url, err := config.LoadDatabaseURL()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.ConnectDB(url)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err = database.RunMigration(db); err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = repository.NewAdminRepository(db).Bootstrap(ctx, normalized); err != nil {
		log.Fatal(err)
	}
	log.Println("First Super Admin is ready. Log in and open /admin.")
}

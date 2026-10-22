package main

import (
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/legacycutover"
	"log"
	"os"
)

func main() {
	if os.Getenv("CONFIRM_LEGACY_CUTOVER") != "archive-and-backfill" {
		log.Fatal("refusing to cut over without CONFIRM_LEGACY_CUTOVER=archive-and-backfill")
	}
	databaseURL, err := config.LoadDatabaseURL()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.ConnectDB(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if os.Getenv("PREFLIGHT_ONLY") == "legacy-preflight-only" {
		if err = legacycutover.Preflight(db); err != nil {
			log.Fatal(err)
		}
		log.Println("Legacy preflight passed. No data or tables were changed.")
		return
	}
	if err = legacycutover.Run(db); err != nil {
		log.Fatal(err)
	}
	log.Println("Legacy transfer completed. The old records remain in legacy_agro_shield.")
}

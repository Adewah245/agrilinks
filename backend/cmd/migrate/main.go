package main

import (
	"log"

	"backend/internal/config"
	"backend/internal/database"
)

// The migrate command applies every pending numbered migration and exits. It is
// separate from the web server so deployment can verify schema changes without
// opening an HTTP listener or starting payment and AI integrations.
func main() {
	databaseURL, err := config.LoadDatabaseURL()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.ConnectDB(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := database.RunMigration(db); err != nil {
		log.Fatal(err)
	}
}

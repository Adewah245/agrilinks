package main

import (
	"log"
	"net/http"
	"os"

	app "backend/internal"
	"backend/internal/database"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/routes"
)

func main() {
	// Establishes a connection between Agro-Shield and the database.
	db, err := database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Makes sure the database has the required tables/structure.
	if err := database.RunMigration(db); err != nil {
		log.Fatal(err)
	}
	// Create repositories
	farmerRepo := repository.NewFarmerRepository(db)
	cropRepo := repository.NewCropRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	diagnosisRepo := repository.NewDiagnosisRepository(db)
	negotiationRepo := repository.NewNegotiationRepository(db)
	negotiationMsgRepo := repository.NewNegotiationMessageRepository(db)
	// Creates the Gemini AI provider that Agro-Shield will use for AI operations.
	aiProvider, err := services.NewGeminiProvider(
		os.Getenv("GEMINI_API_KEY"),
	)
	if err != nil {
		log.Fatal(err)
	}

	container := app.NewContainer(
		farmerRepo,
		cropRepo,
		orderRepo,
		diagnosisRepo,
		negotiationRepo,
		negotiationMsgRepo,
		aiProvider,
	)
	// Tells the HTTP server which URL should go to which handler.
	routes.RegisterRoutes(container)

	log.Println("Server Starting on: http://localhost:8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Println(err)
	}
}

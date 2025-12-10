package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/dwikikusuma/atlas/pkg/database"
)

const (
	postgresURI = "postgres://atlas:atlaspassword@localhost:5432/atlas_db?sslmode=disable"
)

func main() {
	ctx := context.Background()

	dbConfig := database.PostgresConfig{
		ConnectionURL: postgresURI,
		MaxConn:       10,
		MinConn:       2,
		MaxIdleTime:   300,
		HealthCheck:   true,
	}

	db, err := database.NewPostgresPool(ctx, dbConfig)
	if err != nil {
		log.Fatalf("❌ cannot connect to Postgres: %v", err)
	}
	defer db.Close()
	log.Println("✅ Connected to Postgres")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("🛑 Shutting down server...")

	db.Close()
	log.Println("✅ Postgres connection closed")
}

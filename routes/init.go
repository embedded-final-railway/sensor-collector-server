package routes

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var dbPool *pgxpool.Pool
var dbClient *mongo.Client

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	dbPool, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_CONNECTION_STRING"))
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}
	dbClient, _ = mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_CONNECTION_STRING")))
}

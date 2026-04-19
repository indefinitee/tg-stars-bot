package main

import (
	"context"
	"log"
	"os"

	"tg-stars-bot/internal/domain"
	"tg-stars-bot/internal/infrastructure/bitrix"
	"tg-stars-bot/internal/infrastructure/db"
	"tg-stars-bot/internal/transport/handlers"
	"tg-stars-bot/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Database connection
	dbURL := getEnv("DATABASE_URL", "postgres://stars_user:stars_pass@localhost:5432/stars_db?sslmode=disable")

	// Bot token for Telegram auth validation
	botToken := getEnv("TELEGRAM_BOT_TOKEN", "")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	// Bitrix24 configuration
	bitrixURL := getEnv("BITRIX_URL", "")
	bitrixWebhook := getEnv("BITRIX_WEBHOOK", "")

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// Initialize repositories
	userRepo := db.NewUserRepository(pool)
	periodRepo := db.NewPeriodRepository(pool)
	voteRepo := db.NewVoteRepository(pool)

	// Initialize Bitrix client
	var bitrixClient domain.BitrixClient
	if bitrixURL != "" && bitrixWebhook != "" {
		bitrixClient = bitrix.NewClient(bitrix.Config{
			BaseURL: bitrixURL,
			Webhook: bitrixWebhook,
		})
	}

	// Initialize use cases using the usecase package constructors
	voteUC := usecase.NewVoteUseCase(userRepo, periodRepo, voteRepo, bitrixClient)
	reportUC := usecase.NewReportUseCase(userRepo, periodRepo, voteRepo)
	adminUC := usecase.NewAdminUseCase(userRepo, periodRepo, voteRepo, bitrixClient)

	// Initialize handler
	handler := handlers.NewHandler(voteUC, reportUC, adminUC)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Register routes
	handler.RegisterRoutes(r, botToken)

	// Start server
	port := getEnv("PORT", "8080")
	log.Printf("Starting server on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

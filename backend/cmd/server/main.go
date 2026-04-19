package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"tg-stars-bot/internal/domain"
	"tg-stars-bot/internal/infrastructure/bitrix"
	"tg-stars-bot/internal/infrastructure/db"
	"tg-stars-bot/internal/transport/handlers"

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

	// Initialize use cases
	voteUC := newVoteUseCase(userRepo, periodRepo, voteRepo, bitrixClient)
	reportUC := newReportUseCase(userRepo, periodRepo, voteRepo)
	adminUC := newAdminUseCase(userRepo, periodRepo, voteRepo, bitrixClient)

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

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// Import use cases (placeholder - will be created via wire or manual DI)
func newVoteUseCase(
	userRepo domain.UserRepository,
	periodRepo domain.PeriodRepository,
	voteRepo domain.VoteRepository,
	bitrixClient domain.BitrixClient,
) *voteUseCase {
	return &voteUseCase{
		userRepo:     userRepo,
		periodRepo:   periodRepo,
		voteRepo:     voteRepo,
		bitrixClient: bitrixClient,
	}
}

func newReportUseCase(
	userRepo domain.UserRepository,
	periodRepo domain.PeriodRepository,
	voteRepo domain.VoteRepository,
) *reportUseCase {
	return &reportUseCase{
		userRepo:   userRepo,
		periodRepo: periodRepo,
		voteRepo:   voteRepo,
	}
}

func newAdminUseCase(
	userRepo domain.UserRepository,
	periodRepo domain.PeriodRepository,
	voteRepo domain.VoteRepository,
	bitrixClient domain.BitrixClient,
) *adminUseCase {
	return &adminUseCase{
		userRepo:     userRepo,
		periodRepo:   periodRepo,
		voteRepo:     voteRepo,
		bitrixClient: bitrixClient,
	}
}

// Local type aliases for use case construction
type voteUseCase struct {
	userRepo     domain.UserRepository
	periodRepo   domain.PeriodRepository
	voteRepo     domain.VoteRepository
	bitrixClient domain.BitrixClient
}

type reportUseCase struct {
	userRepo   domain.UserRepository
	periodRepo domain.PeriodRepository
	voteRepo   domain.VoteRepository
}

type adminUseCase struct {
	userRepo     domain.UserRepository
	periodRepo   domain.PeriodRepository
	voteRepo     domain.VoteRepository
	bitrixClient domain.BitrixClient
}

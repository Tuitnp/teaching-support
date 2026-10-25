package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/config"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/queue"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/storage"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/router"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/logger"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/middleware"
)

func main() {
	// 1. Initialize Configurations and Databases
	config.LoadEnv()
	if _, err := config.JWTSecret(); err != nil {
		log.Fatal(err)
	}
	logger.Init() // Moved up to ensure logging starts immediately
	database.Connect()
	if err := storage.Init(); err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// 2. Initialize Redis/Queue
	redisAddr := config.GetEnv("REDIS_ADDRESS")
	redisPassword := config.GetEnv("REDIS_PASSWORD")
	queue.Init(redisAddr, redisPassword)

	// 3. Initialize Fiber App
	app := fiber.New(fiber.Config{
		BodyLimit: 60 * 1024 * 1024, // 60 MB total request body
	})

	// 4. Register Middlewares
	app.Use(cors.New())
	
	// Log Middleware
	app.Use(middleware.RequestID())
	app.Use(middleware.Recover()) 
	app.Use(middleware.Logging())

	// 5. Setup Routes
	router.SetupRoutes(app)

	// 6. Start Server
	log.Println("Server running on :8080")
	log.Fatal(app.Listen(":8080"))
}

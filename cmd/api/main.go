package main

import (
	"log"

	"cotizaciones-api/internal/config"
	"cotizaciones-api/internal/database"
	"cotizaciones-api/internal/handlers"
	"cotizaciones-api/internal/middleware"
	"cotizaciones-api/internal/repository"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func main() {
	cfg := config.Load()

	pool, err := database.NewPool(cfg)
	if err != nil {
		log.Fatalf("error al conectar a la base de datos: %v", err)
	}
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	authHandler := handlers.NewAuthHandler(cfg, userRepo)

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{cfg.CORSOrigins},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	app.Get("/api/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := app.Group("/api")
	api.Post("/auth/login", authHandler.Login)
	api.Get("/auth/me", authHandler.Me, middleware.Protected(cfg.JWTSecret))

	log.Printf("servidor escuchando en el puerto %s (%s)", cfg.AppPort, cfg.AppEnv)
	if err := app.Listen(":" + cfg.AppPort); err != nil {
		log.Fatalf("error al iniciar el servidor: %v", err)
	}
}

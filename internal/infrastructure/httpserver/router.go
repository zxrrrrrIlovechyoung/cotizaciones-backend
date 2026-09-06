package httpserver

import (
	"cotizaciones-api/internal/infrastructure/httpserver/handlers"
	"cotizaciones-api/internal/infrastructure/httpserver/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

type RouterDeps struct {
	AuthHandler *handlers.AuthHandler
	JWTSecret   string
	CORSOrigin  string
}

// NewRouter arma el árbol de rutas de Fiber. Es el único lugar que sabe
// que existe Fiber "como framework HTTP" — nada más depende de él.
func NewRouter(deps RouterDeps) *fiber.App {
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{deps.CORSOrigin},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	app.Get("/api/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := app.Group("/api")
	api.Post("/auth/login", deps.AuthHandler.Login)
	api.Get("/auth/me", deps.AuthHandler.Me, middleware.Protected(deps.JWTSecret))

	return app
}

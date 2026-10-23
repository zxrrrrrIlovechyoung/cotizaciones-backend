package httpserver

import (
	"time"

	"cotizaciones-api/internal/infrastructure/httpserver/handlers"
	"cotizaciones-api/internal/infrastructure/httpserver/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

type RouterDeps struct {
	AuthHandler    *handlers.AuthHandler
	ClienteHandler *handlers.ClienteHandler
	JWTSecret      string
	CORSOrigin     string
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

	// Rate-limit específico para el login: frena ataques de fuerza bruta sin
	// afectar al resto de la API.
	loginLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: time.Minute,
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "demasiados intentos, espera un minuto",
			})
		},
	})

	api := app.Group("/api")
	api.Post("/auth/login", loginLimiter, deps.AuthHandler.Login)
	api.Get("/auth/me", middleware.Protected(deps.JWTSecret), deps.AuthHandler.Me)

	// Clientes: todo el módulo requiere sesión iniciada (token válido).
	protected := middleware.Protected(deps.JWTSecret)
	api.Get("/clientes", protected, deps.ClienteHandler.List)
	api.Post("/clientes", protected, deps.ClienteHandler.Create)
	api.Delete("/clientes/:id", protected, deps.ClienteHandler.Delete)

	return app
}

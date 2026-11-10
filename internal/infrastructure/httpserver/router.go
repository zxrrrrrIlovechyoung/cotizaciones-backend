package httpserver

import (
	"strconv"
	"strings"
	"time"

	"cotizaciones-api/internal/domain"
	"cotizaciones-api/internal/infrastructure/httpserver/handlers"
	"cotizaciones-api/internal/infrastructure/httpserver/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RouterDeps struct {
	SessionDB         *pgxpool.Pool
	PerfilHandler     *handlers.PerfilHandler
	AuthHandler       *handlers.AuthHandler
	Recuperacion      *handlers.RecuperacionHandler
	EliminacionCuenta *handlers.EliminacionCuentaHandler
	ClienteHandler    *handlers.ClienteHandler
	CotizacionHandler *handlers.CotizacionHandler
	JWTSecret         string
	CORSOrigins       string
}

// NewRouter arma el árbol de rutas de Fiber. Es el único lugar que sabe
// que existe Fiber "como framework HTTP" — nada más depende de él.
func NewRouter(deps RouterDeps) *fiber.App {
	app := fiber.New(fiber.Config{BodyLimit: 16 << 20})

	app.Use(cors.New(cors.Config{
		AllowOrigins:     parseCORSOrigins(deps.CORSOrigins),
		AllowCredentials: true,
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
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
	api.Use(func(c fiber.Ctx) error {
		if c.Get("X-Requested-With") != "Cotizaciones" {
			return c.SendStatus(403)
		}
		origin := c.Get("Origin")
		if origin != "" {
			allowed := false
			for _, v := range parseCORSOrigins(deps.CORSOrigins) {
				if origin == v {
					allowed = true
				}
			}
			if !allowed {
				return c.SendStatus(403)
			}
		}
		return c.Next()
	})
	api.Get("/auth/session", deps.AuthHandler.Session)
	api.Post("/auth/logout", deps.AuthHandler.Logout)
	api.Use(func(c fiber.Ctx) error {
		if c.Path() == "/api/auth/me" || c.Path() == "/api/auth/perfil" || c.Path() == "/api/auth/logo" || strings.HasPrefix(c.Path(), "/api/clientes") || strings.HasPrefix(c.Path(), "/api/cotizaciones") {
			return middleware.SessionRequired(deps.SessionDB)(c)
		}
		return c.Next()
	})
	api.Post("/auth/login", loginLimiter, deps.AuthHandler.Login)
	recuperacion := api.Group("/auth/recuperacion")
	recuperacion.Post("/solicitar", limiter.New(limiter.Config{Max: 6, Expiration: time.Minute}), deps.Recuperacion.Solicitar)
	limiteRecuperacion := limiter.New(limiter.Config{Max: 20, Expiration: time.Minute})
	recuperacion.Post("/verificar", limiteRecuperacion, deps.Recuperacion.Verificar)
	recuperacion.Post("/restablecer", limiteRecuperacion, deps.Recuperacion.Restablecer)
	api.Post("/auth/registro", limiter.New(limiter.Config{Max: 10, Expiration: time.Minute}), deps.PerfilHandler.Register)
	api.Get("/auth/logo", middleware.Protected(deps.JWTSecret), deps.PerfilHandler.Logo)
	api.Put("/auth/logo", middleware.Protected(deps.JWTSecret), deps.PerfilHandler.SaveLogo)
	api.Delete("/auth/logo", middleware.Protected(deps.JWTSecret), deps.PerfilHandler.DeleteLogo)
	api.Post("/auth/validar-actividad", limiter.New(limiter.Config{Max: 30, Expiration: time.Minute}), deps.PerfilHandler.Validate)
	api.Put("/auth/perfil", middleware.Protected(deps.JWTSecret), deps.PerfilHandler.Save)
	api.Get("/auth/perfil", middleware.Protected(deps.JWTSecret), deps.PerfilHandler.Get)
	api.Post("/auth/validar-registro", limiter.New(limiter.Config{Max: 30, Expiration: time.Minute}), deps.AuthHandler.ValidateRegistration)
	api.Post("/auth/email-disponible", limiter.New(limiter.Config{Max: 30, Expiration: time.Minute}), deps.AuthHandler.EmailAvailable)
	api.Get("/auth/me", middleware.Protected(deps.JWTSecret), deps.AuthHandler.Me)

	// Clientes y Cotizaciones: todo requiere sesión iniciada (token válido).
	protected := middleware.Protected(deps.JWTSecret)
	eliminacion := api.Group("/auth/eliminacion", middleware.SessionRequired(deps.SessionDB), protected, limiter.New(limiter.Config{
		Max: 12, Expiration: time.Minute,
		KeyGenerator: func(c fiber.Ctx) string { return strconv.FormatInt(domain.Owner(c.Context()), 10) },
	}))
	eliminacion.Post("/solicitar", deps.EliminacionCuenta.Solicitar)
	eliminacion.Post("/verificar", deps.EliminacionCuenta.Verificar)
	eliminacion.Delete("", deps.EliminacionCuenta.Eliminar)
	contrasena := api.Group("/auth/password", middleware.SessionRequired(deps.SessionDB), protected)
	limiteContrasena := limiter.New(limiter.Config{
		Max:        12,
		Expiration: time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			return strconv.FormatInt(domain.Owner(c.Context()), 10)
		},
	})
	contrasena.Post("/verificar", limiteContrasena, deps.AuthHandler.VerifyCurrentPassword)
	contrasena.Put("", limiteContrasena, deps.AuthHandler.ChangePassword)
	nombreUsuario := api.Group("/auth/nombre", middleware.SessionRequired(deps.SessionDB), protected)
	nombreUsuario.Get("", deps.AuthHandler.EstadoNombre)
	nombreUsuario.Put("", deps.AuthHandler.ActualizarNombre)
	fotoPerfil := api.Group("/auth/foto", middleware.SessionRequired(deps.SessionDB), protected)
	fotoPerfil.Get("", deps.PerfilHandler.FotoPerfil)
	fotoPerfil.Put("", deps.PerfilHandler.GuardarFotoPerfil)
	configuracion := api.Group("/configuracion", middleware.SessionRequired(deps.SessionDB), protected)
	configuracion.Get("/alertas", deps.PerfilHandler.GetConfiguracionAlertas)
	configuracion.Put("/alertas", deps.PerfilHandler.SaveConfiguracionAlertas)
	configuracion.Get("/cotizaciones", deps.PerfilHandler.GetPreferenciasCotizacion)
	configuracion.Put("/cotizaciones", deps.PerfilHandler.SavePreferenciasCotizacion)
	configuracion.Get("/documento", deps.PerfilHandler.GetConfiguracionDocumento)
	configuracion.Put("/documento", deps.PerfilHandler.SaveConfiguracionDocumento)
	api.Get("/clientes", protected, deps.ClienteHandler.List)
	api.Get("/clientes/:id/cotizaciones", protected, deps.ClienteHandler.Cotizaciones)
	api.Post("/clientes", protected, deps.ClienteHandler.Create)

	api.Get("/cotizaciones", protected, deps.CotizacionHandler.List)
	api.Get("/cotizaciones/por-vencer", protected, deps.PerfilHandler.AlertasVencimiento)
	api.Get("/cotizaciones/:id/documento", protected, deps.CotizacionHandler.Documento)
	api.Get("/cotizaciones/partidas/:id/imagenes/:slot", protected, deps.CotizacionHandler.ImagenPartida)
	api.Post("/cotizaciones", protected, deps.CotizacionHandler.Create)
	api.Put("/cotizaciones/:id", protected, deps.CotizacionHandler.Update)
	api.Put("/cotizaciones/:id/archivo", protected, deps.CotizacionHandler.Archive)
	api.Delete("/cotizaciones/:id", protected, deps.CotizacionHandler.Delete)

	return app
}

func parseCORSOrigins(value string) []string {
	origins := strings.Split(value, ",")
	allowed := make([]string, 0, len(origins))
	for _, origin := range origins {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			allowed = append(allowed, trimmed)
		}
	}
	return allowed
}

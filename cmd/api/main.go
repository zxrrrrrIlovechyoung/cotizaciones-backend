package main

import (
	"log"

	"cotizaciones-api/internal/config"
	"cotizaciones-api/internal/infrastructure/documentos"
	"cotizaciones-api/internal/infrastructure/httpserver"
	"cotizaciones-api/internal/infrastructure/httpserver/handlers"
	"cotizaciones-api/internal/infrastructure/mail"
	"cotizaciones-api/internal/infrastructure/postgres"
	"cotizaciones-api/internal/infrastructure/security"
	"cotizaciones-api/internal/usecase"
)

// main es el "composition root": el único punto del programa donde se
// conocen todas las implementaciones concretas (Postgres, JWT, bcrypt, Fiber)
// y se inyectan unas dentro de otras. El resto del código solo conoce interfaces.
func main() {
	cfg := config.Load()

	pool, err := postgres.NewPool(cfg)
	if err != nil {
		log.Fatalf("error al conectar a la base de datos: %v", err)
	}
	defer pool.Close()

	userRepo := postgres.NewUserRepository(pool)
	clienteRepo := postgres.NewClienteRepository(pool)
	hasher := security.NewBcryptHasher()
	tokens := security.NewJWTGenerator(cfg.JWTSecret, cfg.JWTExpirationHours)

	authUseCase := usecase.NewAuthUseCase(userRepo, hasher, tokens)
	authHandler := handlers.NewAuthHandler(authUseCase, pool)
	var enviarCodigo, enviarCodigoEliminacion func(string, string) error
	modoLocal := cfg.AppEnv == "development" && (cfg.SMTPHost == "127.0.0.1" || cfg.SMTPHost == "localhost" || cfg.SMTPHost == "::1")
	if cfg.SMTPHost != "" && cfg.SMTPFrom != "" &&
		(cfg.SMTPMode == "starttls" || cfg.SMTPMode == "tls" || (cfg.SMTPMode == "none" && modoLocal && cfg.SMTPUser == "")) {
		correo := mail.SMTP{Host: cfg.SMTPHost, Port: cfg.SMTPPort, From: cfg.SMTPFrom, User: cfg.SMTPUser, Password: cfg.SMTPPassword, Mode: cfg.SMTPMode}
		enviarCodigo = correo.EnviarCodigo
		enviarCodigoEliminacion = correo.EnviarCodigoEliminacion
	}
	recuperacionHandler := handlers.NewRecuperacionHandler(pool, hasher, cfg.JWTSecret, enviarCodigo)

	clienteUseCase := usecase.NewClienteUseCase(clienteRepo)
	clienteHandler := handlers.NewClienteHandler(clienteUseCase)

	cotizacionRepo := postgres.NewCotizacionRepository(pool)
	cotizacionUseCase := usecase.NewCotizacionUseCase(cotizacionRepo, userRepo)
	cotizacionHandler := handlers.NewCotizacionHandler(cotizacionUseCase, documentos.NuevoServicio(pool, cfg.PDFChromiumPath))

	app := httpserver.NewRouter(httpserver.RouterDeps{
		SessionDB:         pool,
		PerfilHandler:     handlers.NewPerfilHandler(pool),
		AuthHandler:       authHandler,
		Recuperacion:      recuperacionHandler,
		EliminacionCuenta: handlers.NewEliminacionCuentaHandler(pool, hasher, cfg.JWTSecret, enviarCodigoEliminacion),
		ClienteHandler:    clienteHandler,
		CotizacionHandler: cotizacionHandler,
		JWTSecret:         cfg.JWTSecret,
		CORSOrigins:       cfg.CORSOrigins,
	})

	log.Printf("servidor escuchando en el puerto %s (%s)", cfg.AppPort, cfg.AppEnv)
	if err := app.Listen(":" + cfg.AppPort); err != nil {
		log.Fatalf("error al iniciar el servidor: %v", err)
	}
}

package main

import (
	"log"

	"cotizaciones-api/internal/config"
	"cotizaciones-api/internal/infrastructure/httpserver"
	"cotizaciones-api/internal/infrastructure/httpserver/handlers"
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
	authHandler := handlers.NewAuthHandler(authUseCase)

	clienteUseCase := usecase.NewClienteUseCase(clienteRepo)
	clienteHandler := handlers.NewClienteHandler(clienteUseCase)

	app := httpserver.NewRouter(httpserver.RouterDeps{
		AuthHandler:    authHandler,
		ClienteHandler: clienteHandler,
		JWTSecret:      cfg.JWTSecret,
		CORSOrigin:     cfg.CORSOrigins,
	})

	log.Printf("servidor escuchando en el puerto %s (%s)", cfg.AppPort, cfg.AppEnv)
	if err := app.Listen(":" + cfg.AppPort); err != nil {
		log.Fatalf("error al iniciar el servidor: %v", err)
	}
}

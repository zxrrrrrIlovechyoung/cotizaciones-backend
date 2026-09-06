package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort string
	AppEnv  string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	JWTSecret          string
	JWTExpirationHours int

	CORSOrigins string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró archivo .env, se usarán variables de entorno del sistema")
	}

	expHours, err := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24"))
	if err != nil {
		expHours = 24
	}

	return &Config{
		AppPort: getEnv("APP_PORT", "3000"),
		AppEnv:  getEnv("APP_ENV", "development"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "cotizaciones_user"),
		DBPassword: getEnv("DB_PASSWORD", "cotizaciones_pass"),
		DBName:     getEnv("DB_NAME", "cotizaciones_db"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		JWTSecret:          getEnv("JWT_SECRET", "changeme"),
		JWTExpirationHours: expHours,

		CORSOrigins: getEnv("CORS_ORIGINS", "http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

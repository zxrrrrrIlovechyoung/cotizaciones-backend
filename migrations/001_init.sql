-- Migración inicial: tabla de usuarios
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    nombre        VARCHAR(150) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Usuario de prueba: admin@cotizaciones.com / 123456
-- Hash generado con bcrypt (costo 10) — ver `go run ./cmd/hashpw <contraseña>`.
-- SOLO para desarrollo local: nunca sembrar esta contraseña en producción.
INSERT INTO users (nombre, email, password_hash)
VALUES (
    'Administrador',
    'admin@cotizaciones.com',
    '$2a$10$C505d96ILivuOpuTGeNgYekwWq5R2QNKoxDD9PexC9MsCKGd0V6kC'
)
ON CONFLICT (email) DO NOTHING;

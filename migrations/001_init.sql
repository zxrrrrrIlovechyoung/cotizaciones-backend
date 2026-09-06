-- Migración inicial: tabla de usuarios
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    nombre        VARCHAR(150) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Usuario de prueba: admin@cotizaciones.com / admin123
-- Hash generado con bcrypt (costo 10)
INSERT INTO users (nombre, email, password_hash)
VALUES (
    'Administrador',
    'admin@cotizaciones.com',
    '$2a$10$K3jleLM7x/HB4Ag8AV9QDuCF/kgDDgADhyaBd4NVcVmZXebisrLlC'
)
ON CONFLICT (email) DO NOTHING;

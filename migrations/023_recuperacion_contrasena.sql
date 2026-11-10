CREATE TABLE recuperacion_contrasena (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    solicitud_hash BYTEA NOT NULL UNIQUE,
    codigo_hash BYTEA,
    permiso_hash BYTEA UNIQUE,
    clave_base BYTEA NOT NULL,
    intentos SMALLINT NOT NULL DEFAULT 0 CHECK (intentos BETWEEN 0 AND 5),
    expira_en TIMESTAMPTZ NOT NULL,
    reenviar_desde TIMESTAMPTZ NOT NULL,
    CHECK ((codigo_hash IS NULL) <> (permiso_hash IS NULL))
);

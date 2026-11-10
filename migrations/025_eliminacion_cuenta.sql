CREATE TABLE eliminacion_cuenta (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    sesion_hash CHAR(64) NOT NULL,
    codigo_hash BYTEA NOT NULL,
    permiso_hash BYTEA,
    intentos SMALLINT NOT NULL DEFAULT 0 CHECK (intentos BETWEEN 0 AND 5),
    expira_en TIMESTAMPTZ NOT NULL,
    reenviar_desde TIMESTAMPTZ NOT NULL
);

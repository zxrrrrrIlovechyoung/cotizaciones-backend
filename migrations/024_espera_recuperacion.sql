ALTER TABLE recuperacion_contrasena
    ADD COLUMN permiso_expira_en TIMESTAMPTZ,
    DROP CONSTRAINT recuperacion_contrasena_check,
    DROP COLUMN reenviar_desde;

UPDATE recuperacion_contrasena SET permiso_expira_en = expira_en WHERE permiso_hash IS NOT NULL;

ALTER TABLE recuperacion_contrasena
    ADD CHECK ((permiso_hash IS NULL) = (permiso_expira_en IS NULL));

CREATE TABLE recuperacion_envios (
    correo_hash BYTEA PRIMARY KEY,
    envios SMALLINT NOT NULL CHECK (envios BETWEEN 1 AND 4),
    ultimo_envio TIMESTAMPTZ NOT NULL,
    reenviar_desde TIMESTAMPTZ NOT NULL
);

CREATE INDEX recuperacion_envios_ultimo_idx ON recuperacion_envios(ultimo_envio);

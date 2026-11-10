CREATE TABLE historial_nombre_usuario (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    nombre_anterior VARCHAR(60) NOT NULL,
    nombre_nuevo VARCHAR(60) NOT NULL,
    cambiado_en TIMESTAMPTZ NOT NULL,
    CHECK (nombre_anterior <> nombre_nuevo)
);

CREATE INDEX historial_nombre_usuario_ultimo_idx
    ON historial_nombre_usuario (user_id, cambiado_en DESC);

CREATE FUNCTION registrar_cambio_nombre_usuario() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    ultimo_cambio TIMESTAMPTZ;
    instante TIMESTAMPTZ;
BEGIN
    SELECT cambiado_en INTO ultimo_cambio
    FROM historial_nombre_usuario
    WHERE user_id = OLD.id
    ORDER BY cambiado_en DESC
    LIMIT 1;

    instante := clock_timestamp();
    IF ultimo_cambio + interval '1440 hours' > instante THEN
        RAISE EXCEPTION USING ERRCODE = 'P0060', MESSAGE = 'Solo puedes cambiar tu nombre cada 60 días';
    END IF;

    INSERT INTO historial_nombre_usuario (user_id, nombre_anterior, nombre_nuevo, cambiado_en)
    VALUES (OLD.id, OLD.nombre, NEW.nombre, instante);
    RETURN NEW;
END;
$$;

CREATE TRIGGER users_registrar_cambio_nombre
    BEFORE UPDATE OF nombre ON users
    FOR EACH ROW
    WHEN (OLD.nombre IS DISTINCT FROM NEW.nombre)
    EXECUTE FUNCTION registrar_cambio_nombre_usuario();

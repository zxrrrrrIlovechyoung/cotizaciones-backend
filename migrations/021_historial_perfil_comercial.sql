CREATE TABLE historial_perfil_comercial (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES perfiles_cuenta(user_id),
    datos_anteriores JSONB NOT NULL,
    datos_nuevos JSONB NOT NULL,
    cambiado_en TIMESTAMPTZ NOT NULL
);

CREATE INDEX historial_perfil_comercial_ultimo_idx
    ON historial_perfil_comercial (user_id, cambiado_en DESC);

CREATE FUNCTION registrar_cambio_perfil_comercial() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    ultimo_cambio TIMESTAMPTZ;
    instante TIMESTAMPTZ;
BEGIN
    IF OLD.tipo IS DISTINCT FROM NEW.tipo THEN
        RAISE EXCEPTION USING ERRCODE = 'P0061', MESSAGE = 'El tipo de cuenta no se puede cambiar';
    END IF;

    SELECT cambiado_en INTO ultimo_cambio
    FROM historial_perfil_comercial
    WHERE user_id = OLD.user_id
    ORDER BY cambiado_en DESC
    LIMIT 1;

    instante := clock_timestamp();
    IF ultimo_cambio + interval '1440 hours' > instante THEN
        RAISE EXCEPTION USING ERRCODE = 'P0060', MESSAGE = 'Solo puedes editar tus datos comerciales cada 60 días';
    END IF;

    INSERT INTO historial_perfil_comercial (user_id, datos_anteriores, datos_nuevos, cambiado_en)
    VALUES (OLD.user_id,
        jsonb_build_object('comercial', OLD.comercial, 'razonSocial', OLD.razon_social, 'rfc', OLD.rfc, 'telefono', OLD.telefono, 'direccion', OLD.direccion),
        jsonb_build_object('comercial', NEW.comercial, 'razonSocial', NEW.razon_social, 'rfc', NEW.rfc, 'telefono', NEW.telefono, 'direccion', NEW.direccion),
        instante);
    RETURN NEW;
END;
$$;

CREATE TRIGGER perfiles_cuenta_registrar_cambio
    BEFORE UPDATE OF tipo, comercial, razon_social, rfc, telefono, direccion ON perfiles_cuenta
    FOR EACH ROW
    WHEN ((OLD.tipo, OLD.comercial, OLD.razon_social, OLD.rfc, OLD.telefono, OLD.direccion)
        IS DISTINCT FROM (NEW.tipo, NEW.comercial, NEW.razon_social, NEW.rfc, NEW.telefono, NEW.direccion))
    EXECUTE FUNCTION registrar_cambio_perfil_comercial();

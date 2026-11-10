CREATE INDEX IF NOT EXISTS clientes_identidad_idx
    ON clientes(owner_id, lower(btrim(nombre_comercial)), upper(btrim(rfc)));

CREATE OR REPLACE FUNCTION asegurar_cliente_cotizacion(
    cuenta bigint, nombre text, tipo text, documento text,
    correo text, telefono text, contacto text
) RETURNS bigint LANGUAGE plpgsql AS $$
DECLARE cliente_id bigint;
BEGIN
    IF cuenta IS NULL OR cuenta < 1 OR btrim(nombre) = '' THEN
        RAISE EXCEPTION 'Cuenta o nombre de cliente inválido';
    END IF;
    -- Serialize matching and insertion within the same account.
    PERFORM pg_advisory_xact_lock(cuenta);
    SELECT id INTO cliente_id FROM clientes
    WHERE owner_id = cuenta
      AND lower(btrim(nombre_comercial)) = lower(btrim(nombre))
      AND upper(btrim(rfc)) = upper(btrim(coalesce(documento, '')))
    ORDER BY id LIMIT 1;
    IF cliente_id IS NULL THEN
        INSERT INTO clientes(owner_id, nombre_comercial, tipo_cliente, rfc,
                             correo_contacto, telefono_contacto, atencion_a)
        VALUES (cuenta, btrim(nombre), coalesce(nullif(tipo, ''), 'Empresa'),
                upper(btrim(coalesce(documento, ''))), coalesce(correo, ''),
                coalesce(telefono, ''), coalesce(contacto, ''))
        RETURNING id INTO cliente_id;
    END IF;
    RETURN cliente_id;
END $$;

DO $$
DECLARE cot record;
BEGIN
    FOR cot IN SELECT * FROM cotizaciones
               WHERE id_cliente IS NULL AND estado <> 'Eliminada' ORDER BY owner_id, id
    LOOP
        UPDATE cotizaciones SET id_cliente = asegurar_cliente_cotizacion(
            cot.owner_id, cot.nombre_comercial, cot.tipo_cliente, cot.rfc,
            cot.correo_contacto, cot.telefono_contacto, cot.atencion_a)
        WHERE id = cot.id;
    END LOOP;
END $$;

-- Límites de los datos del emisor en el documento de cotización.
ALTER TABLE perfiles_cuenta
    DROP CONSTRAINT IF EXISTS perfiles_cuenta_comercial_max_50,
    ALTER COLUMN comercial TYPE VARCHAR(53) USING (
        CASE WHEN char_length(comercial) > 53 THEN rtrim(left(ltrim(comercial), 53)) ELSE comercial END
    ),
    ALTER COLUMN razon_social TYPE VARCHAR(38) USING (
        CASE WHEN char_length(razon_social) > 38 THEN rtrim(left(ltrim(razon_social), 38)) ELSE razon_social END
    ),
    ALTER COLUMN direccion TYPE VARCHAR(40) USING (
        CASE WHEN char_length(direccion) > 40 THEN rtrim(left(ltrim(direccion), 40)) ELSE direccion END
    );

ALTER TABLE users
    ADD COLUMN moneda_cotizacion VARCHAR(3) NOT NULL DEFAULT 'MXN'
        CHECK (moneda_cotizacion IN ('MXN', 'USD')),
    ADD COLUMN vigencia_cotizacion_dias SMALLINT
        CHECK (vigencia_cotizacion_dias BETWEEN 1 AND 365);

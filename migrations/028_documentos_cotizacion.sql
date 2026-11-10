CREATE TABLE documentos_cotizacion (
    cotizacion_id BIGINT PRIMARY KEY REFERENCES cotizaciones(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    instantanea JSONB,
    pdf BYTEA,
    generado_en TIMESTAMPTZ,
    CHECK ((instantanea IS NULL) <> (pdf IS NULL)),
    CHECK ((pdf IS NULL) = (generado_en IS NULL))
);

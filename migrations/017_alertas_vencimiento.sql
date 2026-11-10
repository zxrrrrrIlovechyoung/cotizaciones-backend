ALTER TABLE users
    ADD COLUMN IF NOT EXISTS dias_alerta_vencimiento SMALLINT NOT NULL DEFAULT 10
    CHECK (dias_alerta_vencimiento BETWEEN 0 AND 365);

CREATE INDEX IF NOT EXISTS idx_cotizaciones_owner_vigencia
    ON cotizaciones (owner_id, vigencia_hasta, id)
    WHERE estado <> 'Eliminada';

ALTER TABLE users
    ADD COLUMN fuente_documento_cotizacion VARCHAR(12) NOT NULL DEFAULT 'arial'
    CHECK (fuente_documento_cotizacion IN ('arial', 'verdana', 'georgia', 'times'));

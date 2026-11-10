ALTER TABLE users
    ADD COLUMN color_documento_cotizacion VARCHAR(7) NOT NULL DEFAULT '#1b4ed8'
    CHECK (color_documento_cotizacion ~ '^#[0-9a-f]{6}$');

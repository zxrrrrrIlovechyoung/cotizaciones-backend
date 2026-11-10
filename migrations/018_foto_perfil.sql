ALTER TABLE users
    ADD COLUMN IF NOT EXISTS foto_perfil BYTEA,
    ADD COLUMN IF NOT EXISTS foto_perfil_tipo VARCHAR(10),
    ADD CONSTRAINT users_foto_perfil_valida CHECK (
        (foto_perfil IS NULL AND foto_perfil_tipo IS NULL)
        OR (foto_perfil IS NOT NULL AND foto_perfil_tipo IS NOT NULL
            AND octet_length(foto_perfil) BETWEEN 1 AND 2097152
            AND foto_perfil_tipo IN ('image/jpeg', 'image/png'))
    );

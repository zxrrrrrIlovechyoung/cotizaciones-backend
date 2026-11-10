ALTER TABLE partidas
    ADD COLUMN imagen_1 bytea CHECK (octet_length(imagen_1) BETWEEN 1 AND 2097152),
    ADD COLUMN imagen_2 bytea CHECK (octet_length(imagen_2) BETWEEN 1 AND 2097152);

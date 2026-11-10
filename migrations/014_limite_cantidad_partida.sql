ALTER TABLE partidas ADD CONSTRAINT partidas_cantidad_valida
    CHECK (cantidad BETWEEN 1 AND 99999 AND cantidad = trunc(cantidad));

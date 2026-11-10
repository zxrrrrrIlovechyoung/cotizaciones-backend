ALTER TABLE cotizaciones ADD CONSTRAINT cotizaciones_metodos_pago_longitud
    CHECK (char_length(metodos_pago) <= 100);

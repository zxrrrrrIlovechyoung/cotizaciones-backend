ALTER TABLE cotizaciones DROP CONSTRAINT cotizaciones_textos_longitud;
ALTER TABLE cotizaciones ADD CONSTRAINT cotizaciones_textos_longitud CHECK (
    char_length(tiempo_entrega) <= 100
    AND char_length(condiciones_pago) <= 100
    AND char_length(notas_comerciales) <= 100
);

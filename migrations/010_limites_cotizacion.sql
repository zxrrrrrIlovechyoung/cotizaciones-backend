ALTER TABLE partidas ADD CONSTRAINT partidas_precios_validos CHECK (
    precio_unitario_compra BETWEEN 0.01 AND 9999999.99
    AND precio_unitario_venta BETWEEN precio_unitario_compra AND 9999999.99
    AND precio_unitario_compra = round(precio_unitario_compra::numeric, 2)::double precision
    AND precio_unitario_venta = round(precio_unitario_venta::numeric, 2)::double precision
);

ALTER TABLE cotizaciones ADD CONSTRAINT cotizaciones_porcentajes_validos CHECK (
    porcentaje_iva BETWEEN 0 AND 100
    AND porcentaje_iva = round(porcentaje_iva::numeric, 2)::double precision
    AND (porcentaje_anticipo IS NULL OR (
        porcentaje_anticipo BETWEEN 0 AND 100
        AND porcentaje_anticipo = round(porcentaje_anticipo::numeric, 2)::double precision
    ))
);

ALTER TABLE cotizaciones ADD CONSTRAINT cotizaciones_textos_longitud CHECK (
    char_length(tiempo_entrega) <= 100
    AND char_length(condiciones_pago) <= 1000
    AND char_length(notas_comerciales) <= 1000
);

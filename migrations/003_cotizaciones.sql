-- Migración: cotizaciones y sus partidas (relación 1→N)
CREATE TABLE IF NOT EXISTS cotizaciones (
    id                   BIGSERIAL PRIMARY KEY,
    folio                VARCHAR(12)  NOT NULL,
    version              INT          NOT NULL DEFAULT 1,
    estado               VARCHAR(20)  NOT NULL DEFAULT 'Aceptada',
    fecha_creacion       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    id_cliente           BIGINT,
    tipo_cliente         VARCHAR(50)  NOT NULL DEFAULT 'Empresa',
    razon_social         VARCHAR(150) NOT NULL DEFAULT '',
    rfc                  VARCHAR(13)  NOT NULL DEFAULT '',
    nombre_cliente       VARCHAR(150) NOT NULL DEFAULT '',
    nombre_comercial     VARCHAR(150) NOT NULL DEFAULT '',
    atencion_a           VARCHAR(150) NOT NULL DEFAULT '',
    nombre_asesor        VARCHAR(150) NOT NULL DEFAULT '',
    moneda               VARCHAR(10)  NOT NULL DEFAULT 'MXN',
    vigencia_hasta       DATE,
    correo_contacto      VARCHAR(120) NOT NULL DEFAULT '',
    telefono_contacto    VARCHAR(20)  NOT NULL DEFAULT '',
    precios_incluyen_iva BOOLEAN          NOT NULL DEFAULT true,
    porcentaje_iva       DOUBLE PRECISION NOT NULL DEFAULT 16,
    porcentaje_anticipo  DOUBLE PRECISION,
    tiempo_entrega       VARCHAR(150) NOT NULL DEFAULT '',
    condiciones_pago     TEXT         NOT NULL DEFAULT '',
    notas_comerciales    TEXT         NOT NULL DEFAULT '',
    metodos_pago         TEXT         NOT NULL DEFAULT '',
    costo                DOUBLE PRECISION NOT NULL DEFAULT 0,
    subtotal             DOUBLE PRECISION NOT NULL DEFAULT 0,
    impuestos            DOUBLE PRECISION NOT NULL DEFAULT 0,
    total                DOUBLE PRECISION NOT NULL DEFAULT 0,
    utilidad_estimada    DOUBLE PRECISION NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS partidas (
    id                        BIGSERIAL PRIMARY KEY,
    cotizacion_id             BIGINT       NOT NULL REFERENCES cotizaciones(id) ON DELETE CASCADE,
    numero_partida            INT          NOT NULL,
    descripcion               VARCHAR(200) NOT NULL DEFAULT '',
    especificaciones_tecnicas TEXT         NOT NULL DEFAULT '',
    sku                       VARCHAR(50)  NOT NULL DEFAULT '',
    cantidad                  DOUBLE PRECISION NOT NULL DEFAULT 0,
    unidad                    VARCHAR(20)  NOT NULL DEFAULT 'PZA',
    precio_unitario_compra    DOUBLE PRECISION NOT NULL DEFAULT 0,
    precio_unitario_venta     DOUBLE PRECISION NOT NULL DEFAULT 0,
    porcentaje_descuento      DOUBLE PRECISION NOT NULL DEFAULT 0,
    importe                   DOUBLE PRECISION NOT NULL DEFAULT 0,
    requiere_compra           BOOLEAN      NOT NULL DEFAULT false,
    estado_compra             VARCHAR(20)
);

CREATE INDEX IF NOT EXISTS idx_partidas_cotizacion ON partidas(cotizacion_id);

-- Datos de ejemplo (solo si la tabla está vacía). Totales precalculados de forma
-- consistente con la regla de cálculo del sistema.
INSERT INTO cotizaciones (
    folio, version, estado, tipo_cliente, razon_social, rfc, nombre_cliente, nombre_comercial,
    atencion_a, nombre_asesor, moneda, vigencia_hasta, precios_incluyen_iva, porcentaje_iva,
    tiempo_entrega, condiciones_pago, metodos_pago, costo, subtotal, impuestos, total, utilidad_estimada
)
SELECT * FROM (VALUES
    ('0001', 1, 'Aceptada', 'Empresa', 'Distribuidora El Roble S.A. de C.V.', 'DER950101AB1', 'Ing. Marcela Ruiz', 'Distribuidora El Roble S.A. de C.V.', 'Ing. Marcela Ruiz', 'Administrador', 'MXN', DATE '2026-11-30', true,  16.0, '3 dias habiles', 'Pago de contado.', 'Transferencia SPEI.', 1500.0, 2500.0, 0.0,   2500.0, 1000.0),
    ('0002', 1, 'Aceptada', 'Empresa', 'Constructora Vega y Asociados',       'CVA010203XY9', 'Lic. Hugo Vega',    'Constructora Vega y Asociados',       'Lic. Hugo Vega',    'Administrador', 'MXN', DATE '2026-11-30', false, 16.0, '5 dias habiles', '50% de anticipo y 50% contra entrega.', 'Transferencia SPEI.', 4000.0, 5800.0, 928.0, 6728.0, 1800.0)
) AS seed
WHERE NOT EXISTS (SELECT 1 FROM cotizaciones);

INSERT INTO partidas (
    cotizacion_id, numero_partida, descripcion, especificaciones_tecnicas, sku,
    cantidad, unidad, precio_unitario_compra, precio_unitario_venta, porcentaje_descuento, importe, requiere_compra
)
SELECT c.id, v.numero_partida, v.descripcion, v.especificaciones, v.sku,
       v.cantidad, v.unidad, v.compra, v.venta, v.descuento, v.importe, v.requiere
FROM (VALUES
    ('0001', 1, 'Cambio de aceite y filtro',        'Aceite sintetico 5W-30, 4 litros',   '',             2.0, 'SERVICIO', 300.0,  500.0,  0.0, 1000.0, false),
    ('0001', 2, 'Balatas ceramicas delanteras',     'Juego completo, incluye instalacion','BAL-CER-DEL',  1.0, 'PZA',      900.0,  1500.0, 0.0, 1500.0, true),
    ('0002', 1, 'Llantas 195/65 R15',               'Radial, indice de carga 91H',        'LLA-195-65-R15',4.0,'PZA',      1000.0, 1450.0, 0.0, 5800.0, true)
) AS v(folio, numero_partida, descripcion, especificaciones, sku, cantidad, unidad, compra, venta, descuento, importe, requiere)
JOIN cotizaciones c ON c.folio = v.folio
WHERE NOT EXISTS (SELECT 1 FROM partidas);

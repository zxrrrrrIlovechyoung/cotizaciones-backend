-- Migración: tabla del directorio de clientes
CREATE TABLE IF NOT EXISTS clientes (
    id                BIGSERIAL PRIMARY KEY,
    nombre_comercial  VARCHAR(150) NOT NULL,
    tipo_cliente      VARCHAR(50)  NOT NULL DEFAULT 'Empresa',
    rfc               VARCHAR(13)  NOT NULL DEFAULT '',
    correo_contacto   VARCHAR(120) NOT NULL DEFAULT '',
    telefono_contacto VARCHAR(20)  NOT NULL DEFAULT '',
    atencion_a        VARCHAR(150) NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Datos de ejemplo (los mismos que mostraba la maqueta) para que la pantalla
-- no arranque vacía. Solo se insertan si la tabla está vacía.
INSERT INTO clientes (nombre_comercial, tipo_cliente, rfc, correo_contacto, telefono_contacto, atencion_a)
SELECT * FROM (VALUES
    ('Distribuidora El Roble S.A. de C.V.', 'Empresa', 'DER950101AB1', 'compras@elroble.mx',           '773 185 1363', 'Ing. Marcela Ruiz'),
    ('Constructora Vega y Asociados',       'Empresa', 'CVA010203XY9', 'admin@vega.com.mx',             '771 204 8890', 'Lic. Hugo Vega'),
    ('Colegio San Nicolás S.C.',            'Escuela', 'CSN880412K34', 'direccion@sannicolas.edu.mx',   '775 112 7745', 'Mtra. Elena Sáenz'),
    ('Clínica Santa Elena',                 'Empresa', 'CSE030915QQ2', 'contacto@santaelena.mx',        '773 900 4412', 'Dra. Paulina Ortiz'),
    ('Transportes del Bajío',               'Empresa', 'TBA150720JJ4', 'flotilla@transportesbajio.mx',  '461 118 3390', 'C.P. Julio Navarro')
) AS seed(nombre_comercial, tipo_cliente, rfc, correo_contacto, telefono_contacto, atencion_a)
WHERE NOT EXISTS (SELECT 1 FROM clientes);

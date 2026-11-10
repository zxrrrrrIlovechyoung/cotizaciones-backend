package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"cotizaciones-api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ domain.CotizacionRepository = (*CotizacionRepository)(nil)

type CotizacionRepository struct {
	db *pgxpool.Pool
}

func NewCotizacionRepository(db *pgxpool.Pool) *CotizacionRepository {
	return &CotizacionRepository{db: db}
}

func (r *CotizacionRepository) ImagenPartida(ctx context.Context, partidaID int64, slot int) ([]byte, error) {
	var data []byte
	err := r.db.QueryRow(ctx, `
		SELECT CASE WHEN $3 = 1 THEN p.imagen_1 ELSE p.imagen_2 END
		FROM partidas p JOIN cotizaciones c ON c.id = p.cotizacion_id
		WHERE p.id = $1 AND c.owner_id = $2 AND c.estado <> 'Eliminada'
	`, partidaID, domain.Owner(ctx), slot).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && len(data) == 0) {
		return nil, domain.ErrCotizacionNotFound
	}
	return data, err
}

// List devuelve las cotizaciones no eliminadas, cada una con sus partidas.
// Usa dos consultas (cabeceras y partidas) para evitar el problema N+1.
func (r *CotizacionRepository) List(ctx context.Context) ([]domain.Cotizacion, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, folio, version, estado, fecha_creacion, tipo_cliente, razon_social, rfc,
		       nombre_cliente, nombre_comercial, atencion_a, nombre_asesor, moneda, vigencia_hasta,
		       costo, subtotal, impuestos, total, utilidad_estimada,
		       correo_contacto, telefono_contacto, precios_incluyen_iva, porcentaje_iva,
		       porcentaje_anticipo, tiempo_entrega, condiciones_pago, notas_comerciales, metodos_pago, id_cliente,
		       vigencia_hasta - calendario.hoy,
		       estado <> 'Archivada' AND COALESCE(vigencia_hasta - calendario.hoy BETWEEN 0 AND calendario.margen, false)
		FROM cotizaciones
		CROSS JOIN (
		    SELECT (now() AT TIME ZONE 'America/Mexico_City')::date AS hoy, dias_alerta_vencimiento AS margen
		    FROM users WHERE id=$1
		) calendario
		WHERE estado <> 'Eliminada' AND owner_id=$1
		ORDER BY id DESC`, domain.Owner(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cots := make([]domain.Cotizacion, 0)
	indice := make(map[int64]int)
	for rows.Next() {
		var c domain.Cotizacion
		if err := rows.Scan(
			&c.ID, &c.Folio, &c.Version, &c.Estado, &c.FechaCreacion, &c.TipoCliente, &c.RazonSocial, &c.RFC,
			&c.NombreCliente, &c.NombreComercial, &c.AtencionA, &c.NombreAsesor, &c.Moneda, &c.VigenciaHasta,
			&c.Costo, &c.Subtotal, &c.Impuestos, &c.Total, &c.UtilidadEstimada,
			&c.CorreoContacto, &c.TelefonoContacto, &c.PreciosIncluyenIva, &c.PorcentajeIva,
			&c.PorcentajeAnticipo, &c.TiempoEntrega, &c.CondicionesPago, &c.NotasComerciales, &c.MetodosPago,
			&c.IDCliente, &c.DiasParaVencer, &c.ProximaAVencer,
		); err != nil {
			return nil, err
		}
		c.Partidas = make([]domain.Partida, 0)
		indice[c.ID] = len(cots)
		cots = append(cots, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(cots) == 0 {
		return cots, nil
	}

	ids := make([]int64, 0, len(cots))
	for _, c := range cots {
		ids = append(ids, c.ID)
	}

	prows, err := r.db.Query(ctx, `
		SELECT id, cotizacion_id, numero_partida, descripcion, especificaciones_tecnicas, sku,
		       cantidad, unidad, precio_unitario_venta, importe, requiere_compra, estado_compra,
		       imagen_1 IS NOT NULL, imagen_2 IS NOT NULL, porcentaje_descuento, precio_unitario_compra
		FROM partidas
		WHERE cotizacion_id = ANY($1)
		ORDER BY cotizacion_id, numero_partida`, ids)
	if err != nil {
		return nil, err
	}
	defer prows.Close()
	for prows.Next() {
		var cotID int64
		var p domain.Partida
		if err := prows.Scan(
			&p.ID, &cotID, &p.NumeroPartida, &p.Descripcion, &p.EspecificacionesTecnicas, &p.SKU,
			&p.Cantidad, &p.Unidad, &p.PrecioUnitarioVenta, &p.Importe, &p.RequiereCompra, &p.EstadoCompra,
			&p.TieneImagen[0], &p.TieneImagen[1],
			&p.PorcentajeDescuento,
			&p.PrecioUnitarioCompra,
		); err != nil {
			return nil, err
		}
		if i, ok := indice[cotID]; ok {
			cots[i].Partidas = append(cots[i].Partidas, p)
		}
	}
	return cots, prows.Err()
}

// Create inserta la cotización y sus partidas en una transacción, y asigna el
// folio correlativo (siguiente al mayor existente) dentro de la misma.
func (r *CotizacionRepository) Create(ctx context.Context, c *domain.Cotizacion) (*domain.Cotizacion, error) {
	return r.save(ctx, c, false)
}

func (r *CotizacionRepository) Update(ctx context.Context, c *domain.Cotizacion) (*domain.Cotizacion, error) {
	return r.save(ctx, c, true)
}

func (r *CotizacionRepository) save(ctx context.Context, c *domain.Cotizacion, editing bool) (*domain.Cotizacion, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if c.IDCliente != nil {
		var allowed bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM clientes WHERE id=$1 AND owner_id=$2)`, *c.IDCliente, domain.Owner(ctx)).Scan(&allowed); err != nil {
			return nil, err
		}
		if !allowed {
			return nil, domain.ErrClienteNotFound
		}
	}
	if _, err := tx.Exec(ctx, `LOCK TABLE cotizaciones IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return nil, err
	}
	if editing {
		err := comprobarCotizacionEditable(tx.QueryRow(ctx, consultaCotizacionArchivada, c.ID, domain.Owner(ctx)))
		if err != nil {
			return nil, err
		}
	}
	if c.IDCliente == nil {
		var clienteID int64
		if err := tx.QueryRow(ctx, `SELECT asegurar_cliente_cotizacion($1,$2,$3,$4,$5,$6,$7)`,
			domain.Owner(ctx), c.NombreComercial, c.TipoCliente, c.RFC, c.CorreoContacto,
			c.TelefonoContacto, c.AtencionA).Scan(&clienteID); err != nil {
			return nil, err
		}
		c.IDCliente = &clienteID
	}

	if editing {
		for i := range c.Partidas {
			p := &c.Partidas[i]
			if p.OrigenID == 0 {
				continue
			}
			err := tx.QueryRow(ctx, `SELECT estado_compra FROM partidas WHERE id=$1 AND cotizacion_id=$2 AND EXISTS(SELECT 1 FROM cotizaciones WHERE id=$2 AND owner_id=$3)`, p.OrigenID, c.ID, domain.Owner(ctx)).Scan(&p.EstadoCompra)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, domain.ErrCotizacionNotFound
			}
			if err != nil {
				return nil, err
			}
			if !p.RequiereCompra {
				p.EstadoCompra = nil
			}
		}
		err := tx.QueryRow(ctx, `UPDATE cotizaciones SET
			id_cliente=$1,tipo_cliente=$2,razon_social=$3,rfc=$4,nombre_cliente=$5,nombre_comercial=$6,
			atencion_a=$7,moneda=$8,vigencia_hasta=$9,correo_contacto=$10,telefono_contacto=$11,
			precios_incluyen_iva=$12,porcentaje_iva=$13,porcentaje_anticipo=$14,tiempo_entrega=$15,
			condiciones_pago=$16,notas_comerciales=$17,metodos_pago=$18,costo=$19,subtotal=$20,
			impuestos=$21,total=$22,utilidad_estimada=$23,version=version+1
			WHERE id=$24 AND owner_id=$25 AND estado<>'Eliminada'
			RETURNING folio,version,fecha_creacion,nombre_asesor`,
			c.IDCliente, c.TipoCliente, c.RazonSocial, c.RFC, c.NombreCliente, c.NombreComercial,
			c.AtencionA, c.Moneda, c.VigenciaHasta, c.CorreoContacto, c.TelefonoContacto, c.PreciosIncluyenIva,
			c.PorcentajeIva, c.PorcentajeAnticipo, c.TiempoEntrega, c.CondicionesPago, c.NotasComerciales, c.MetodosPago,
			c.Costo, c.Subtotal, c.Impuestos, c.Total, c.UtilidadEstimada, c.ID, domain.Owner(ctx),
		).Scan(&c.Folio, &c.Version, &c.FechaCreacion, &c.NombreAsesor)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCotizacionNotFound
		}
		if err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM partidas WHERE cotizacion_id=$1`, c.ID); err != nil {
			return nil, err
		}
	} else {
		var maxFolio int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(folio::int), 0) FROM cotizaciones`).Scan(&maxFolio); err != nil {
			return nil, err
		}
		c.Folio = fmt.Sprintf("%04d", maxFolio+1)

		if err := tx.QueryRow(ctx, `
		INSERT INTO cotizaciones (
			folio, version, estado, id_cliente, tipo_cliente, razon_social, rfc, nombre_cliente,
			nombre_comercial, atencion_a, nombre_asesor, moneda, vigencia_hasta, correo_contacto,
			telefono_contacto, precios_incluyen_iva, porcentaje_iva, porcentaje_anticipo,
			tiempo_entrega, condiciones_pago, notas_comerciales, metodos_pago,
			costo, subtotal, impuestos, total, utilidad_estimada,owner_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28)
		RETURNING id, fecha_creacion`,
			c.Folio, c.Version, c.Estado, c.IDCliente, c.TipoCliente, c.RazonSocial, c.RFC, c.NombreCliente,
			c.NombreComercial, c.AtencionA, c.NombreAsesor, c.Moneda, c.VigenciaHasta, c.CorreoContacto,
			c.TelefonoContacto, c.PreciosIncluyenIva, c.PorcentajeIva, c.PorcentajeAnticipo,
			c.TiempoEntrega, c.CondicionesPago, c.NotasComerciales, c.MetodosPago,
			c.Costo, c.Subtotal, c.Impuestos, c.Total, c.UtilidadEstimada, domain.Owner(ctx),
		).Scan(&c.ID, &c.FechaCreacion); err != nil {
			return nil, err
		}

	}
	for i := range c.Partidas {
		p := &c.Partidas[i]
		if err := tx.QueryRow(ctx, `
			INSERT INTO partidas (
				cotizacion_id, numero_partida, descripcion, especificaciones_tecnicas, sku,
				cantidad, unidad, precio_unitario_compra, precio_unitario_venta, porcentaje_descuento,
				importe, requiere_compra, estado_compra, imagen_1, imagen_2
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
			RETURNING id`,
			c.ID, p.NumeroPartida, p.Descripcion, p.EspecificacionesTecnicas, p.SKU,
			p.Cantidad, p.Unidad, p.PrecioUnitarioCompra, p.PrecioUnitarioVenta, p.PorcentajeDescuento,
			p.Importe, p.RequiereCompra, p.EstadoCompra, p.Imagenes[0], p.Imagenes[1],
		).Scan(&p.ID); err != nil {
			return nil, err
		}
	}

	documento := domain.DocumentoCotizacion{Cotizacion: *c}
	emisor := &documento.Emisor
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(p.comercial,u.nombre), COALESCE(p.razon_social,''), COALESCE(p.rfc,''),
		       COALESCE(p.telefono,''), COALESCE(p.direccion,''), u.color_documento_cotizacion,
		       u.fuente_documento_cotizacion, p.logo
		FROM users u LEFT JOIN perfiles_cuenta p ON p.user_id=u.id WHERE u.id=$1
	`, domain.Owner(ctx)).Scan(&emisor.Comercial, &emisor.RazonSocial, &emisor.RFC, &emisor.Telefono,
		&emisor.Direccion, &emisor.Color, &emisor.Fuente, &emisor.Logo); err != nil {
		return nil, err
	}
	instantanea, err := json.Marshal(documento)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO documentos_cotizacion (cotizacion_id,version,instantanea) VALUES ($1,$2,$3)
		ON CONFLICT (cotizacion_id) DO UPDATE SET version=excluded.version,
		  instantanea=excluded.instantanea, pdf=NULL, generado_en=NULL
	`, c.ID, c.Version, instantanea); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// SoftDelete marca la cotización como 'Eliminada' (no la borra físicamente).
func (r *CotizacionRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.cambiarEstadoActiva(ctx, id, "Eliminada")
}

func (r *CotizacionRepository) Archive(ctx context.Context, id int64) error {
	return r.cambiarEstadoActiva(ctx, id, "Archivada")
}

func (r *CotizacionRepository) cambiarEstadoActiva(ctx context.Context, id int64, estado string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE cotizaciones SET estado=$3
		WHERE id=$1 AND owner_id=$2 AND estado<>'Eliminada'
		  AND (estado<>'Archivada' OR estado=$3)
		  AND (vigencia_hasta IS NULL OR vigencia_hasta >= (now() AT TIME ZONE 'America/Mexico_City')::date)
	`, id, domain.Owner(ctx), estado)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if err := comprobarCotizacionEditable(r.db.QueryRow(ctx, consultaCotizacionArchivada, id, domain.Owner(ctx))); err != nil {
			return err
		}
		return domain.ErrCotizacionNotFound
	}
	return nil
}

const consultaCotizacionArchivada = `
	SELECT estado='Archivada' OR COALESCE(vigencia_hasta < (now() AT TIME ZONE 'America/Mexico_City')::date, false)
	FROM cotizaciones WHERE id=$1 AND owner_id=$2 AND estado<>'Eliminada'
`

func comprobarCotizacionEditable(fila pgx.Row) error {
	var archivada bool
	err := fila.Scan(&archivada)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrCotizacionNotFound
	}
	if err != nil {
		return err
	}
	if archivada {
		return domain.ErrCotizacionArchivada
	}
	return nil
}

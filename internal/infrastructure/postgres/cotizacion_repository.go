package postgres

import (
	"context"
	"fmt"

	"cotizaciones-api/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

var _ domain.CotizacionRepository = (*CotizacionRepository)(nil)

type CotizacionRepository struct {
	db *pgxpool.Pool
}

func NewCotizacionRepository(db *pgxpool.Pool) *CotizacionRepository {
	return &CotizacionRepository{db: db}
}

// List devuelve las cotizaciones no eliminadas, cada una con sus partidas.
// Usa dos consultas (cabeceras y partidas) para evitar el problema N+1.
func (r *CotizacionRepository) List(ctx context.Context) ([]domain.Cotizacion, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, folio, version, estado, fecha_creacion, tipo_cliente, razon_social, rfc,
		       nombre_cliente, nombre_comercial, atencion_a, nombre_asesor, moneda, vigencia_hasta,
		       costo, subtotal, impuestos, total, utilidad_estimada
		FROM cotizaciones
		WHERE estado <> 'Eliminada'
		ORDER BY id DESC`)
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
		       cantidad, unidad, precio_unitario_venta, importe, requiere_compra, estado_compra
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
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

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
			costo, subtotal, impuestos, total, utilidad_estimada
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)
		RETURNING id, fecha_creacion`,
		c.Folio, c.Version, c.Estado, c.IDCliente, c.TipoCliente, c.RazonSocial, c.RFC, c.NombreCliente,
		c.NombreComercial, c.AtencionA, c.NombreAsesor, c.Moneda, c.VigenciaHasta, c.CorreoContacto,
		c.TelefonoContacto, c.PreciosIncluyenIva, c.PorcentajeIva, c.PorcentajeAnticipo,
		c.TiempoEntrega, c.CondicionesPago, c.NotasComerciales, c.MetodosPago,
		c.Costo, c.Subtotal, c.Impuestos, c.Total, c.UtilidadEstimada,
	).Scan(&c.ID, &c.FechaCreacion); err != nil {
		return nil, err
	}

	for i := range c.Partidas {
		p := &c.Partidas[i]
		if err := tx.QueryRow(ctx, `
			INSERT INTO partidas (
				cotizacion_id, numero_partida, descripcion, especificaciones_tecnicas, sku,
				cantidad, unidad, precio_unitario_compra, precio_unitario_venta, porcentaje_descuento,
				importe, requiere_compra, estado_compra
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
			RETURNING id`,
			c.ID, p.NumeroPartida, p.Descripcion, p.EspecificacionesTecnicas, p.SKU,
			p.Cantidad, p.Unidad, p.PrecioUnitarioCompra, p.PrecioUnitarioVenta, p.PorcentajeDescuento,
			p.Importe, p.RequiereCompra, p.EstadoCompra,
		).Scan(&p.ID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// SoftDelete marca la cotización como 'Eliminada' (no la borra físicamente).
func (r *CotizacionRepository) SoftDelete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `UPDATE cotizaciones SET estado = 'Eliminada' WHERE id = $1 AND estado <> 'Eliminada'`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCotizacionNotFound
	}
	return nil
}

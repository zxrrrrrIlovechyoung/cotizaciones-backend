package postgres

import (
	"context"
	"errors"

	"cotizaciones-api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Verificación en tiempo de compilación de que implementamos el puerto.
var _ domain.ClienteRepository = (*ClienteRepository)(nil)

type ClienteRepository struct {
	db *pgxpool.Pool
}

func NewClienteRepository(db *pgxpool.Pool) *ClienteRepository {
	return &ClienteRepository{db: db}
}

func (r *ClienteRepository) List(ctx context.Context) ([]domain.Cliente, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, nombre_comercial, tipo_cliente, rfc, correo_contacto, telefono_contacto, atencion_a, created_at
		 FROM clientes WHERE owner_id=$1
		 ORDER BY id DESC`,
		domain.Owner(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Lista vacía (no nil) para que el JSON sea [] y no null cuando no haya filas.
	clientes := make([]domain.Cliente, 0)
	for rows.Next() {
		var c domain.Cliente
		if err := rows.Scan(
			&c.ID, &c.NombreComercial, &c.TipoCliente, &c.RFC,
			&c.CorreoContacto, &c.TelefonoContacto, &c.AtencionA, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		clientes = append(clientes, c)
	}
	return clientes, rows.Err()
}

func (r *ClienteRepository) Create(ctx context.Context, c *domain.Cliente) (*domain.Cliente, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO clientes (nombre_comercial, tipo_cliente, rfc, correo_contacto, telefono_contacto, atencion_a,owner_id)
		 VALUES ($1, $2, $3, $4, $5, $6,$7)
		 RETURNING id, created_at`,
		c.NombreComercial, c.TipoCliente, c.RFC, c.CorreoContacto, c.TelefonoContacto, c.AtencionA, domain.Owner(ctx),
	)
	if err := row.Scan(&c.ID, &c.CreatedAt); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *ClienteRepository) Cotizaciones(ctx context.Context, id int64) (domain.HistorialCliente, error) {
	owner := domain.Owner(ctx)
	historial := domain.HistorialCliente{FoliosEliminados: make([]string, 0)}
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE((SELECT MIN(c.fecha_creacion) FROM cotizaciones c
		                 WHERE c.id_cliente=cl.id AND c.owner_id=cl.owner_id), cl.created_at)
		FROM clientes cl WHERE cl.id=$1 AND cl.owner_id=$2
	`, id, owner).Scan(&historial.FechaAlta)
	if errors.Is(err, pgx.ErrNoRows) {
		return historial, domain.ErrClienteNotFound
	}
	if err != nil {
		return historial, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, folio, fecha_creacion, moneda, total, estado='Eliminada',
		       estado='Archivada' OR COALESCE(vigencia_hasta < (now() AT TIME ZONE 'America/Mexico_City')::date, false)
		FROM cotizaciones
		WHERE id_cliente=$1 AND owner_id=$2
		ORDER BY fecha_creacion DESC, id DESC
	`, id, owner)
	if err != nil {
		return historial, err
	}
	defer rows.Close()
	for rows.Next() {
		var cotizacion domain.CotizacionCliente
		var eliminada bool
		if err := rows.Scan(&cotizacion.ID, &cotizacion.Folio, &cotizacion.FechaCreacion, &cotizacion.Moneda,
			&cotizacion.Total, &eliminada, &cotizacion.Archivada); err != nil {
			return historial, err
		}
		if eliminada {
			historial.FoliosEliminados = append(historial.FoliosEliminados, cotizacion.Folio)
		} else {
			historial.Cotizaciones = append(historial.Cotizaciones, cotizacion)
		}
	}
	return historial, rows.Err()
}

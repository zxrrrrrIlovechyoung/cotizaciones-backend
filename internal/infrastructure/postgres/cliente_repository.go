package postgres

import (
	"context"

	"cotizaciones-api/internal/domain"

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
		 FROM clientes
		 ORDER BY id DESC`,
	)
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
		`INSERT INTO clientes (nombre_comercial, tipo_cliente, rfc, correo_contacto, telefono_contacto, atencion_a)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		c.NombreComercial, c.TipoCliente, c.RFC, c.CorreoContacto, c.TelefonoContacto, c.AtencionA,
	)
	if err := row.Scan(&c.ID, &c.CreatedAt); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *ClienteRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM clientes WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrClienteNotFound
	}
	return nil
}

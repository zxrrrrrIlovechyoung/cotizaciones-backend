package domain

import "context"

// ClienteRepository es el puerto de salida del módulo de clientes. El usecase
// depende de esta interfaz, nunca de pgx.
type ClienteRepository interface {
	List(ctx context.Context) ([]Cliente, error)
	Create(ctx context.Context, c *Cliente) (*Cliente, error)
	Delete(ctx context.Context, id int64) error
}

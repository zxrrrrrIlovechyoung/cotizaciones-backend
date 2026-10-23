package domain

import "context"

// CotizacionRepository es el puerto de salida del módulo de cotizaciones.
type CotizacionRepository interface {
	List(ctx context.Context) ([]Cotizacion, error)
	Create(ctx context.Context, c *Cotizacion) (*Cotizacion, error)
	SoftDelete(ctx context.Context, id int64) error
}

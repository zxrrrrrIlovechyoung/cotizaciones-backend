package domain

import "context"

// CotizacionRepository es el puerto de salida del módulo de cotizaciones.
type CotizacionRepository interface {
	List(ctx context.Context) ([]Cotizacion, error)
	Create(ctx context.Context, c *Cotizacion) (*Cotizacion, error)
	Update(ctx context.Context, c *Cotizacion) (*Cotizacion, error)
	SoftDelete(ctx context.Context, id int64) error
	Archive(ctx context.Context, id int64) error
	ImagenPartida(ctx context.Context, partidaID int64, slot int) ([]byte, error)
}

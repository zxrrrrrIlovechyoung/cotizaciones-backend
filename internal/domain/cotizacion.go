package domain

import "time"

// Partida es un renglón (producto o servicio) de una cotización.
type Partida struct {
	ID                       int64
	NumeroPartida            int
	Descripcion              string
	EspecificacionesTecnicas string
	SKU                      string
	Cantidad                 float64
	Unidad                   string
	PrecioUnitarioCompra     float64
	PrecioUnitarioVenta      float64
	PorcentajeDescuento      float64
	Importe                  float64
	RequiereCompra           bool
	EstadoCompra             *string // null = sin compra agendada
}

// Cotizacion es la entidad de negocio con su lista de partidas (relación 1→N).
type Cotizacion struct {
	ID                 int64
	Folio              string
	Version            int
	Estado             string
	FechaCreacion      time.Time
	TipoCliente        string
	RazonSocial        string
	RFC                string
	NombreCliente      string
	NombreComercial    string
	AtencionA          string
	NombreAsesor       string
	Moneda             string
	VigenciaHasta      *time.Time
	CorreoContacto     string
	TelefonoContacto   string
	PreciosIncluyenIva bool
	PorcentajeIva      float64
	PorcentajeAnticipo *float64
	TiempoEntrega      string
	CondicionesPago    string
	NotasComerciales   string
	MetodosPago        string
	IDCliente          *int64
	Costo              float64
	Subtotal           float64
	Impuestos          float64
	Total              float64
	UtilidadEstimada   float64
	Partidas           []Partida
}

// PartidaNueva y CotizacionNueva son la entrada de creación: lo que captura el
// formulario. El backend calcula folio, totales, asesor, etc. a partir de esto.
type PartidaNueva struct {
	SKU                      string
	Descripcion              string
	EspecificacionesTecnicas string
	Unidad                   string
	Cantidad                 float64
	PrecioUnitarioCompra     float64
	PrecioUnitarioVenta      float64
	PorcentajeDescuento      float64
	RequiereCompra           bool
}

type CotizacionNueva struct {
	IDCliente          *int64
	TipoCliente        string
	NombreComercial    string
	AtencionA          string
	RFC                string
	CorreoContacto     string
	TelefonoContacto   string
	VigenciaHasta      *time.Time
	Moneda             string
	PreciosIncluyenIva bool
	PorcentajeIva      float64
	PorcentajeAnticipo *float64
	TiempoEntrega      string
	CondicionesPago    string
	NotasComerciales   string
	MetodosPago        string
	Partidas           []PartidaNueva
}

package domain

import "time"

// Cliente es la entidad de negocio pura del directorio de clientes. Igual que
// User, no conoce JSON ni SQL: eso vive en las capas externas.
type Cliente struct {
	ID               int64
	NombreComercial  string
	TipoCliente      string
	RFC              string
	CorreoContacto   string
	TelefonoContacto string
	AtencionA        string
	CreatedAt        time.Time
}

type CotizacionCliente struct {
	ID            int64
	Folio         string
	FechaCreacion time.Time
	Moneda        string
	Total         float64
	Archivada     bool
}

type HistorialCliente struct {
	FechaAlta        time.Time
	Cotizaciones     []CotizacionCliente
	FoliosEliminados []string
}

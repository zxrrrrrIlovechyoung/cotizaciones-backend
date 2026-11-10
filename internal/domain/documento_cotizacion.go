package domain

// DocumentoCotizacion reúne la cotización y la identidad del emisor en el
// momento del guardado, incluidos los adjuntos que aparecerán en el PDF.
type DocumentoCotizacion struct {
	Cotizacion Cotizacion
	Emisor     EmisorDocumento
}

type EmisorDocumento struct {
	Comercial, RazonSocial, RFC, Telefono, Direccion string
	Color, Fuente                                    string
	Logo                                             []byte
}

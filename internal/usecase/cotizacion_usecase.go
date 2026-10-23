package usecase

import (
	"context"
	"strings"

	"cotizaciones-api/internal/domain"
)

// CotizacionUseCase orquesta el alta, listado y borrado suave de cotizaciones.
// Depende de su repositorio y del de usuarios (para resolver el nombre del asesor
// a partir del usuario logueado).
type CotizacionUseCase struct {
	repo     domain.CotizacionRepository
	userRepo domain.UserRepository
}

func NewCotizacionUseCase(repo domain.CotizacionRepository, userRepo domain.UserRepository) *CotizacionUseCase {
	return &CotizacionUseCase{repo: repo, userRepo: userRepo}
}

func (uc *CotizacionUseCase) List(ctx context.Context) ([]domain.Cotizacion, error) {
	return uc.repo.List(ctx)
}

// Create arma la cotización final a partir de lo capturado: calcula los totales
// (misma regla que el frontend), asigna estado/versión y el asesor (nombre del
// usuario logueado). El folio lo asigna el repositorio dentro de la transacción.
func (uc *CotizacionUseCase) Create(ctx context.Context, asesorUserID int64, n domain.CotizacionNueva) (*domain.Cotizacion, error) {
	if strings.TrimSpace(n.NombreComercial) == "" || len(n.Partidas) == 0 {
		return nil, domain.ErrCotizacionInvalida
	}

	asesor := ""
	if u, err := uc.userRepo.FindByID(ctx, asesorUserID); err == nil {
		asesor = u.Nombre
	}

	c := &domain.Cotizacion{
		Version:            1,
		Estado:             "Aceptada",
		TipoCliente:        valorODefecto(n.TipoCliente, "Empresa"),
		RazonSocial:        n.NombreComercial, // decisión: razonSocial = nombreComercial
		RFC:                n.RFC,
		NombreCliente:      n.AtencionA, // decisión: nombreCliente = atencionA
		NombreComercial:    n.NombreComercial,
		AtencionA:          n.AtencionA,
		NombreAsesor:       asesor,
		Moneda:             valorODefecto(n.Moneda, "MXN"),
		VigenciaHasta:      n.VigenciaHasta,
		CorreoContacto:     n.CorreoContacto,
		TelefonoContacto:   n.TelefonoContacto,
		PreciosIncluyenIva: n.PreciosIncluyenIva,
		PorcentajeIva:      n.PorcentajeIva,
		PorcentajeAnticipo: n.PorcentajeAnticipo,
		TiempoEntrega:      n.TiempoEntrega,
		CondicionesPago:    n.CondicionesPago,
		NotasComerciales:   n.NotasComerciales,
		MetodosPago:        n.MetodosPago,
		IDCliente:          n.IDCliente,
	}

	// Totales: réplica de calcularTotales() del frontend. Si los precios ya
	// incluyen IVA, el IVA no se desglosa (queda en 0).
	for i, p := range n.Partidas {
		descuento := p.Cantidad * p.PrecioUnitarioVenta * p.PorcentajeDescuento / 100
		base := p.Cantidad*p.PrecioUnitarioVenta - descuento
		impuesto := 0.0
		if !n.PreciosIncluyenIva {
			impuesto = base * n.PorcentajeIva / 100
		}

		c.Costo += p.Cantidad * p.PrecioUnitarioCompra
		c.Subtotal += base
		c.Impuestos += impuesto
		c.Total += base + impuesto

		c.Partidas = append(c.Partidas, domain.Partida{
			NumeroPartida:            i + 1,
			Descripcion:              p.Descripcion,
			EspecificacionesTecnicas: p.EspecificacionesTecnicas,
			SKU:                      p.SKU,
			Cantidad:                 p.Cantidad,
			Unidad:                   valorODefecto(p.Unidad, "PZA"),
			PrecioUnitarioCompra:     p.PrecioUnitarioCompra,
			PrecioUnitarioVenta:      p.PrecioUnitarioVenta,
			PorcentajeDescuento:      p.PorcentajeDescuento,
			Importe:                  base,
			RequiereCompra:           p.RequiereCompra,
			EstadoCompra:             nil,
		})
	}
	c.UtilidadEstimada = c.Subtotal - c.Costo

	return uc.repo.Create(ctx, c)
}

func (uc *CotizacionUseCase) Delete(ctx context.Context, id int64) error {
	return uc.repo.SoftDelete(ctx, id)
}

func valorODefecto(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

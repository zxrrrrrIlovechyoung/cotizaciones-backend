package usecase_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"cotizaciones-api/internal/domain"
	"cotizaciones-api/internal/usecase"
)

// fakeCotizacionRepo implementa domain.CotizacionRepository en memoria. Create
// solo guarda la cotización que arma el usecase, para poder inspeccionar los
// totales calculados sin tocar la base de datos. Reutiliza el fakeUserRepo que
// ya vive en auth_usecase_test.go (mismo paquete de pruebas).
type fakeCotizacionRepo struct {
	creada *domain.Cotizacion
	err    error
}

func (f *fakeCotizacionRepo) List(_ context.Context) ([]domain.Cotizacion, error) {
	return nil, f.err
}

func (f *fakeCotizacionRepo) Create(_ context.Context, c *domain.Cotizacion) (*domain.Cotizacion, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.creada = c
	return c, nil
}

func (f *fakeCotizacionRepo) SoftDelete(_ context.Context, _ int64) error { return f.err }

// aprox compara flotantes con una tolerancia pequeña (evita falsos negativos por
// la aritmética de punto flotante).
func aprox(a, b float64) bool { return math.Abs(a-b) < 0.0001 }

func TestCrearCotizacion_CalculaTotales(t *testing.T) {
	repo := &fakeCotizacionRepo{}
	userRepo := fakeUserRepo{user: &domain.User{ID: 1, Nombre: "Administrador"}}
	uc := usecase.NewCotizacionUseCase(repo, userRepo)

	// IVA 16% no incluido. P1: 2×1000. P2: 1×500 con 10% de descuento.
	nueva := domain.CotizacionNueva{
		NombreComercial:    "Taller QA",
		AtencionA:          "Ing. Prueba",
		PreciosIncluyenIva: false,
		PorcentajeIva:      16,
		Partidas: []domain.PartidaNueva{
			{Descripcion: "Producto A", Cantidad: 2, PrecioUnitarioCompra: 600, PrecioUnitarioVenta: 1000, PorcentajeDescuento: 0},
			{Descripcion: "Producto B", Cantidad: 1, PrecioUnitarioCompra: 300, PrecioUnitarioVenta: 500, PorcentajeDescuento: 10},
		},
	}

	c, err := uc.Create(context.Background(), 1, nueva)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}

	if !aprox(c.Costo, 1500) || !aprox(c.Subtotal, 2450) || !aprox(c.Impuestos, 392) ||
		!aprox(c.Total, 2842) || !aprox(c.UtilidadEstimada, 950) {
		t.Fatalf("totales incorrectos: costo=%.2f subtotal=%.2f iva=%.2f total=%.2f utilidad=%.2f",
			c.Costo, c.Subtotal, c.Impuestos, c.Total, c.UtilidadEstimada)
	}
	if len(c.Partidas) != 2 || !aprox(c.Partidas[0].Importe, 2000) || !aprox(c.Partidas[1].Importe, 450) {
		t.Fatalf("importes de partida incorrectos: %+v", c.Partidas)
	}
	if c.NombreAsesor != "Administrador" {
		t.Fatalf("asesor esperado 'Administrador', se obtuvo %q", c.NombreAsesor)
	}
	if c.Estado != "Aceptada" || c.Version != 1 {
		t.Fatalf("estado/version por defecto incorrectos: estado=%q version=%d", c.Estado, c.Version)
	}
}

func TestCrearCotizacion_PreciosIncluyenIva_SinImpuestos(t *testing.T) {
	repo := &fakeCotizacionRepo{}
	uc := usecase.NewCotizacionUseCase(repo, fakeUserRepo{user: &domain.User{Nombre: "Administrador"}})

	nueva := domain.CotizacionNueva{
		NombreComercial:    "Taller QA",
		PreciosIncluyenIva: true, // los precios ya incluyen IVA: no se desglosa
		PorcentajeIva:      16,
		Partidas:           []domain.PartidaNueva{{Descripcion: "X", Cantidad: 1, PrecioUnitarioVenta: 1000, PrecioUnitarioCompra: 700}},
	}

	c, err := uc.Create(context.Background(), 1, nueva)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if !aprox(c.Impuestos, 0) || !aprox(c.Subtotal, 1000) || !aprox(c.Total, 1000) {
		t.Fatalf("con precios que ya incluyen IVA el impuesto debe ser 0: iva=%.2f subtotal=%.2f total=%.2f",
			c.Impuestos, c.Subtotal, c.Total)
	}
}

func TestCrearCotizacion_Invalida(t *testing.T) {
	uc := usecase.NewCotizacionUseCase(&fakeCotizacionRepo{}, fakeUserRepo{user: &domain.User{}})

	// Sin nombre comercial.
	_, err := uc.Create(context.Background(), 1, domain.CotizacionNueva{
		Partidas: []domain.PartidaNueva{{Descripcion: "X", Cantidad: 1, PrecioUnitarioVenta: 1}},
	})
	if !errors.Is(err, domain.ErrCotizacionInvalida) {
		t.Fatalf("se esperaba ErrCotizacionInvalida (sin nombre), se obtuvo: %v", err)
	}

	// Sin partidas.
	_, err = uc.Create(context.Background(), 1, domain.CotizacionNueva{NombreComercial: "Taller"})
	if !errors.Is(err, domain.ErrCotizacionInvalida) {
		t.Fatalf("se esperaba ErrCotizacionInvalida (sin partidas), se obtuvo: %v", err)
	}
}

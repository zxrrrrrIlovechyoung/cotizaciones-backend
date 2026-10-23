package usecase

import (
	"context"
	"strings"

	"cotizaciones-api/internal/domain"
)

// ClienteUseCase orquesta las operaciones del directorio de clientes. No sabe
// de dónde vienen los datos ni cómo se exponen por HTTP.
type ClienteUseCase struct {
	repo domain.ClienteRepository
}

func NewClienteUseCase(repo domain.ClienteRepository) *ClienteUseCase {
	return &ClienteUseCase{repo: repo}
}

func (uc *ClienteUseCase) List(ctx context.Context) ([]domain.Cliente, error) {
	return uc.repo.List(ctx)
}

func (uc *ClienteUseCase) Create(ctx context.Context, c *domain.Cliente) (*domain.Cliente, error) {
	// Normalización/validación mínima de negocio. El nombre es lo único
	// obligatorio; el resto son datos de contacto opcionales.
	c.NombreComercial = strings.TrimSpace(c.NombreComercial)
	if c.NombreComercial == "" {
		return nil, domain.ErrClienteInvalido
	}
	if strings.TrimSpace(c.TipoCliente) == "" {
		c.TipoCliente = "Empresa"
	}
	return uc.repo.Create(ctx, c)
}

func (uc *ClienteUseCase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

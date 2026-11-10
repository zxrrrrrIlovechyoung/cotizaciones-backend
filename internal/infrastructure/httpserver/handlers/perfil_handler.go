package handlers

import (
	"errors"
	"time"

	"cotizaciones-api/internal/domain"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Profile writes are scoped to the authenticated user, never a request user ID.
type PerfilHandler struct{ db *pgxpool.Pool }

func NewPerfilHandler(db *pgxpool.Pool) *PerfilHandler { return &PerfilHandler{db: db} }
func (h *PerfilHandler) Get(c fiber.Ctx) error {
	return h.responderPerfil(c, fiber.StatusOK)
}

func (h *PerfilHandler) responderPerfil(c fiber.Ctx, status int) error {
	var p struct {
		domain.PerfilCuenta
		TieneLogo       bool       `json:"tieneLogo"`
		PuedeEditar     bool       `json:"puedeEditar"`
		DisponibleDesde *time.Time `json:"disponibleDesde"`
	}
	err := h.db.QueryRow(c.Context(), `
		SELECT p.tipo, p.comercial, p.razon_social, p.rfc, p.telefono, p.direccion, p.logo IS NOT NULL,
		       COALESCE(h.disponible_desde <= now(), true), h.disponible_desde
		FROM perfiles_cuenta p
		LEFT JOIN LATERAL (
			SELECT cambiado_en + interval '1440 hours' AS disponible_desde
			FROM historial_perfil_comercial
			WHERE user_id = p.user_id
			ORDER BY cambiado_en DESC LIMIT 1
		) h ON true
		WHERE p.user_id = $1
	`, domain.Owner(c.Context())).Scan(&p.Tipo, &p.Comercial, &p.RazonSocial, &p.RFC, &p.Telefono, &p.Direccion, &p.TieneLogo, &p.PuedeEditar, &p.DisponibleDesde)
	if errors.Is(err, pgx.ErrNoRows) {
		return c.SendStatus(404)
	}
	if err != nil {
		return internalError(c, "leer perfil comercial", err)
	}
	c.Set("Cache-Control", "private, no-store")
	return c.Status(status).JSON(p)
}
func (h *PerfilHandler) Validate(c fiber.Ctx) error {
	var p domain.PerfilCuenta
	if err := c.Bind().Body(&p); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Solicitud inválida"})
	}
	if err := p.Validate(); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}
func (h *PerfilHandler) Save(c fiber.Ctx) error {
	var p domain.PerfilCuenta
	if err := c.Bind().Body(&p); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Solicitud inválida"})
	}
	id := domain.Owner(c.Context())
	var tipoActual string
	err := h.db.QueryRow(c.Context(), `SELECT tipo FROM perfiles_cuenta WHERE user_id=$1`, id).Scan(&tipoActual)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return internalError(c, "consultar tipo de cuenta", err)
	}
	if tipoActual != "" {
		if p.Tipo != "" && p.Tipo != tipoActual {
			return c.Status(400).JSON(fiber.Map{"error": "El tipo de cuenta no se puede cambiar."})
		}
		p.Tipo = tipoActual
	}
	if err := p.Validate(); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	_, err = h.db.Exec(c.Context(), `
		INSERT INTO perfiles_cuenta (user_id,tipo,comercial,razon_social,rfc,telefono,direccion)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (user_id) DO UPDATE SET tipo=excluded.tipo, comercial=excluded.comercial,
		  razon_social=excluded.razon_social, rfc=excluded.rfc, telefono=excluded.telefono, direccion=excluded.direccion
	`, id, p.Tipo, p.Comercial, p.RazonSocial, p.RFC, p.Telefono, p.Direccion)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "P0060":
			return h.responderPerfil(c, fiber.StatusConflict)
		case "P0061":
			return c.Status(400).JSON(fiber.Map{"error": "El tipo de cuenta no se puede cambiar."})
		}
	}
	if err != nil {
		return internalError(c, "guardar perfil", err)
	}
	return h.responderPerfil(c, fiber.StatusOK)
}

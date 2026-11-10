package handlers

import (
	"cotizaciones-api/internal/domain"

	"github.com/gofiber/fiber/v3"
)

type preferenciasCotizacion struct {
	Moneda       string `json:"moneda"`
	VigenciaDias *int   `json:"vigenciaDias"`
}

func (h *PerfilHandler) GetPreferenciasCotizacion(c fiber.Ctx) error {
	var preferencias preferenciasCotizacion
	err := h.db.QueryRow(c.Context(), `SELECT moneda_cotizacion, vigencia_cotizacion_dias FROM users WHERE id = $1`,
		domain.Owner(c.Context())).Scan(&preferencias.Moneda, &preferencias.VigenciaDias)
	if err != nil {
		return internalError(c, "consultar preferencias de cotización", err)
	}
	c.Set("Cache-Control", "private, no-store")
	return c.JSON(preferencias)
}

func (h *PerfilHandler) SavePreferenciasCotizacion(c fiber.Ctx) error {
	var preferencias preferenciasCotizacion
	if err := c.Bind().Body(&preferencias); err != nil ||
		(preferencias.Moneda != "MXN" && preferencias.Moneda != "USD") ||
		(preferencias.VigenciaDias != nil && (*preferencias.VigenciaDias < 1 || *preferencias.VigenciaDias > 365)) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Selecciona MXN o USD y una vigencia de 1 a 365 días, o déjala vacía."})
	}
	_, err := h.db.Exec(c.Context(), `UPDATE users SET moneda_cotizacion = $1, vigencia_cotizacion_dias = $2 WHERE id = $3`,
		preferencias.Moneda, preferencias.VigenciaDias, domain.Owner(c.Context()))
	if err != nil {
		return internalError(c, "guardar preferencias de cotización", err)
	}
	return c.JSON(preferencias)
}

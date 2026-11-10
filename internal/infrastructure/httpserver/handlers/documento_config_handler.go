package handlers

import (
	"regexp"
	"strings"

	"cotizaciones-api/internal/domain"

	"github.com/gofiber/fiber/v3"
)

var colorDocumentoHex = regexp.MustCompile(`^#[0-9a-f]{6}$`)

type configuracionDocumento struct {
	Color  string `json:"color"`
	Fuente string `json:"fuente"`
}

func (h *PerfilHandler) GetConfiguracionDocumento(c fiber.Ctx) error {
	var datos configuracionDocumento
	err := h.db.QueryRow(c.Context(), `SELECT color_documento_cotizacion, fuente_documento_cotizacion FROM users WHERE id=$1`,
		domain.Owner(c.Context())).Scan(&datos.Color, &datos.Fuente)
	if err != nil {
		return internalError(c, "leer preferencias del documento de cotización", err)
	}
	c.Set("Cache-Control", "private, no-store")
	return c.JSON(datos)
}

func (h *PerfilHandler) SaveConfiguracionDocumento(c fiber.Ctx) error {
	var req struct {
		Color  string  `json:"color"`
		Fuente *string `json:"fuente"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Solicitud inválida."})
	}
	req.Color = strings.ToLower(strings.TrimSpace(req.Color))
	if !colorDocumentoHex.MatchString(req.Color) {
		return c.Status(400).JSON(fiber.Map{"error": "Ingresa un color hexadecimal válido, como #1B4ED8."})
	}
	if req.Fuente != nil {
		switch *req.Fuente {
		case "arial", "verdana", "georgia", "times":
		default:
			return c.Status(400).JSON(fiber.Map{"error": "Selecciona una fuente disponible."})
		}
	}
	var datos configuracionDocumento
	err := h.db.QueryRow(c.Context(), `UPDATE users SET color_documento_cotizacion=$1,
		fuente_documento_cotizacion=COALESCE($2, fuente_documento_cotizacion) WHERE id=$3
		RETURNING color_documento_cotizacion, fuente_documento_cotizacion`, req.Color, req.Fuente,
		domain.Owner(c.Context())).Scan(&datos.Color, &datos.Fuente)
	if err != nil {
		return internalError(c, "guardar preferencias del documento de cotización", err)
	}
	return c.JSON(datos)
}

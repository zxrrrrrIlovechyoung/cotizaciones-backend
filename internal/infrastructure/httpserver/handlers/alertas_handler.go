package handlers

import (
	"cotizaciones-api/internal/domain"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
)

type configuracionAlertas struct {
	Dias *int `json:"dias"`
}

type alertaVencimiento struct {
	ID            int64  `json:"id"`
	Folio         string `json:"folio"`
	Cliente       string `json:"cliente"`
	DiasRestantes int    `json:"diasRestantes"`
}

func (h *PerfilHandler) GetConfiguracionAlertas(c fiber.Ctx) error {
	var dias int
	err := h.db.QueryRow(c.Context(), `SELECT dias_alerta_vencimiento FROM users WHERE id = $1`,
		domain.Owner(c.Context())).Scan(&dias)
	if err != nil {
		return internalError(c, "leer configuración de alertas", err)
	}
	c.Set("Cache-Control", "private, no-store")
	return c.JSON(configuracionAlertas{Dias: &dias})
}

func (h *PerfilHandler) SaveConfiguracionAlertas(c fiber.Ctx) error {
	var req configuracionAlertas
	if err := c.Bind().Body(&req); err != nil || req.Dias == nil || *req.Dias < 0 || *req.Dias > 365 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Ingresa un número entero entre 0 y 365 días."})
	}
	_, err := h.db.Exec(c.Context(), `UPDATE users SET dias_alerta_vencimiento = $1 WHERE id = $2`,
		*req.Dias, domain.Owner(c.Context()))
	if err != nil {
		return internalError(c, "guardar configuración de alertas", err)
	}
	return c.JSON(req)
}

func (h *PerfilHandler) AlertasVencimiento(c fiber.Ctx) error {
	rows, err := h.db.Query(c.Context(), `
		SELECT c.id, c.folio, c.nombre_comercial, c.vigencia_hasta - calendario.hoy
		FROM users u
		JOIN cotizaciones c ON c.owner_id = u.id
		CROSS JOIN (SELECT (now() AT TIME ZONE 'America/Mexico_City')::date AS hoy) calendario
		WHERE u.id = $1 AND c.estado NOT IN ('Eliminada', 'Archivada')
		  AND c.vigencia_hasta BETWEEN calendario.hoy AND calendario.hoy + u.dias_alerta_vencimiento
		ORDER BY c.vigencia_hasta, c.id
	`, domain.Owner(c.Context()))
	if err != nil {
		return internalError(c, "listar alertas de vencimiento", err)
	}
	alertas, err := pgx.AppendRows(make([]alertaVencimiento, 0), rows, pgx.RowToStructByPos[alertaVencimiento])
	if err != nil {
		return internalError(c, "leer alertas de vencimiento", err)
	}
	c.Set("Cache-Control", "private, no-store")
	return c.JSON(alertas)
}

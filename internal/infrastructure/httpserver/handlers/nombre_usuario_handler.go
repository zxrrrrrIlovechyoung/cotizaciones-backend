package handlers

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"cotizaciones-api/internal/domain"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type estadoNombreUsuario struct {
	Nombre          string     `json:"nombre"`
	PuedeEditar     bool       `json:"puedeEditar"`
	DisponibleDesde *time.Time `json:"disponibleDesde"`
}

func (h *AuthHandler) EstadoNombre(c fiber.Ctx) error {
	return h.responderEstadoNombre(c, fiber.StatusOK)
}

func (h *AuthHandler) ActualizarNombre(c fiber.Ctx) error {
	var req struct {
		Nombre string `json:"nombre"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Solicitud inválida."})
	}
	nombre := strings.TrimSpace(req.Nombre)
	if longitud := utf8.RuneCountInString(nombre); longitud < 2 || longitud > 60 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "El nombre debe tener entre 2 y 60 caracteres."})
	}

	_, err := h.db.Exec(c.Context(), `UPDATE users SET nombre = $1 WHERE id = $2`,
		nombre, domain.Owner(c.Context()))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "P0060" {
		return h.responderEstadoNombre(c, fiber.StatusConflict)
	}
	if err != nil {
		return internalError(c, "actualizar nombre de usuario", err)
	}
	return h.responderEstadoNombre(c, fiber.StatusOK)
}

func (h *AuthHandler) responderEstadoNombre(c fiber.Ctx, status int) error {
	var estado estadoNombreUsuario
	err := h.db.QueryRow(c.Context(), `
		SELECT u.nombre, COALESCE(h.disponible_desde <= now(), true), h.disponible_desde
		FROM users u
		LEFT JOIN LATERAL (
			SELECT cambiado_en + interval '1440 hours' AS disponible_desde
			FROM historial_nombre_usuario
			WHERE user_id = u.id
			ORDER BY cambiado_en DESC LIMIT 1
		) h ON true
		WHERE u.id = $1
	`, domain.Owner(c.Context())).Scan(&estado.Nombre, &estado.PuedeEditar, &estado.DisponibleDesde)
	if errors.Is(err, pgx.ErrNoRows) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if err != nil {
		return internalError(c, "consultar disponibilidad de cambio de nombre", err)
	}
	c.Set("Cache-Control", "private, no-store")
	return c.Status(status).JSON(estado)
}

package handlers

import (
	"bytes"
	"errors"
	"image"
	"io"

	"cotizaciones-api/internal/domain"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
)

const maxFotoPerfil = 2 << 20

func (h *PerfilHandler) FotoPerfil(c fiber.Ctx) error {
	var foto []byte
	var mime string
	err := h.db.QueryRow(c.Context(), `SELECT foto_perfil, foto_perfil_tipo FROM users
		WHERE id = $1 AND foto_perfil IS NOT NULL`, domain.Owner(c.Context())).Scan(&foto, &mime)
	c.Set("Cache-Control", "private, no-store")
	if errors.Is(err, pgx.ErrNoRows) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if err != nil {
		return internalError(c, "leer foto de perfil", err)
	}
	c.Set("Content-Type", mime)
	c.Set("X-Content-Type-Options", "nosniff")
	return c.Send(foto)
}

func (h *PerfilHandler) GuardarFotoPerfil(c fiber.Ctx) error {
	invalid := func() error {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Selecciona una foto PNG o JPG de hasta 2 MB y 4096 píxeles por lado."})
	}
	form, err := c.MultipartForm()
	if err != nil {
		return invalid()
	}
	files := form.File["foto"]
	if len(files) != 1 || files[0].Size > maxFotoPerfil {
		return invalid()
	}
	file, err := files[0].Open()
	if err != nil {
		return internalError(c, "abrir foto de perfil", err)
	}
	defer file.Close()
	foto, err := io.ReadAll(io.LimitReader(file, maxFotoPerfil+1))
	if err != nil {
		return internalError(c, "leer archivo de foto de perfil", err)
	}
	cfg, formato, err := image.DecodeConfig(bytes.NewReader(foto))
	if err != nil || len(foto) > maxFotoPerfil || (formato != "png" && formato != "jpeg") || cfg.Width > 4096 || cfg.Height > 4096 {
		return invalid()
	}
	if _, _, err := image.Decode(bytes.NewReader(foto)); err != nil {
		return invalid()
	}
	_, err = h.db.Exec(c.Context(), `UPDATE users SET foto_perfil = $1, foto_perfil_tipo = $2 WHERE id = $3`,
		foto, "image/"+formato, domain.Owner(c.Context()))
	if err != nil {
		return internalError(c, "guardar foto de perfil", err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

package handlers

import (
	"bytes"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"

	"cotizaciones-api/internal/domain"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
)

const maxLogoCuenta = 2 << 20

var errLogoInvalido = errors.New("Selecciona un logo PNG o JPG válido, de hasta 2 MB y 8000 píxeles por lado.")

func leerLogoCuenta(files []*multipart.FileHeader) ([]byte, string, error) {
	if len(files) != 1 || files[0].Size > maxLogoCuenta {
		return nil, "", errLogoInvalido
	}
	archivo, err := files[0].Open()
	if err != nil {
		return nil, "", err
	}
	defer archivo.Close()
	logo, err := io.ReadAll(io.LimitReader(archivo, maxLogoCuenta+1))
	if err != nil {
		return nil, "", err
	}
	mime := http.DetectContentType(logo)
	cfg, _, decodeErr := image.DecodeConfig(bytes.NewReader(logo))
	if len(logo) > maxLogoCuenta || (mime != "image/png" && mime != "image/jpeg") || decodeErr != nil ||
		cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 8000 || cfg.Height > 8000 {
		return nil, "", errLogoInvalido
	}
	return logo, mime, nil
}

func (h *PerfilHandler) Logo(c fiber.Ctx) error {
	var logo []byte
	var mime string
	err := h.db.QueryRow(c.Context(), `SELECT logo, logo_tipo FROM perfiles_cuenta WHERE user_id=$1 AND logo IS NOT NULL`,
		domain.Owner(c.Context())).Scan(&logo, &mime)
	if errors.Is(err, pgx.ErrNoRows) {
		return c.SendStatus(404)
	}
	if err != nil {
		return internalError(c, "leer logo comercial", err)
	}
	c.Set("Content-Type", mime)
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Cache-Control", "private, no-store")
	return c.Send(logo)
}

func (h *PerfilHandler) SaveLogo(c fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": errLogoInvalido.Error()})
	}
	logo, mime, err := leerLogoCuenta(form.File["logo"])
	if errors.Is(err, errLogoInvalido) {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if err != nil {
		return internalError(c, "leer archivo del logo comercial", err)
	}
	tag, err := h.db.Exec(c.Context(), `UPDATE perfiles_cuenta SET logo=$1, logo_tipo=$2 WHERE user_id=$3`,
		logo, mime, domain.Owner(c.Context()))
	if err != nil {
		return internalError(c, "guardar logo comercial", err)
	}
	if tag.RowsAffected() == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Tu cuenta todavía no tiene un perfil comercial registrado."})
	}
	return c.SendStatus(204)
}

func (h *PerfilHandler) DeleteLogo(c fiber.Ctx) error {
	_, err := h.db.Exec(c.Context(), `UPDATE perfiles_cuenta SET logo=NULL, logo_tipo='' WHERE user_id=$1 AND logo IS NOT NULL`,
		domain.Owner(c.Context()))
	if err != nil {
		return internalError(c, "eliminar logo comercial", err)
	}
	return c.SendStatus(204)
}

package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"strconv"

	"cotizaciones-api/internal/domain"
	"github.com/gofiber/fiber/v3"
)

const maxImagenPartida = 2 << 20

func leerCotizacion(c fiber.Ctx, req *cotizacionRequest) error {
	contentType, _, _ := mime.ParseMediaType(c.Get("Content-Type"))
	if contentType != "multipart/form-data" {
		return c.Bind().Body(req)
	}
	form, err := c.MultipartForm()
	if err != nil || len(form.Value["datos"]) != 1 {
		return fmt.Errorf("formulario de cotización inválido")
	}
	if err := json.Unmarshal([]byte(form.Value["datos"][0]), req); err != nil {
		return fmt.Errorf("datos de cotización inválidos")
	}
	var total int64
	for key, files := range form.File {
		var partida, slot int
		if _, err := fmt.Sscanf(key, "imagen_%d_%d", &partida, &slot); err != nil ||
			key != fmt.Sprintf("imagen_%d_%d", partida, slot) || partida < 0 ||
			partida >= len(req.Partidas) || slot < 1 || slot > 2 || len(files) != 1 {
			return fmt.Errorf("cada partida admite hasta dos imágenes")
		}
		file := files[0]
		if file.Size <= 0 || file.Size > maxImagenPartida {
			return fmt.Errorf("partida %d: cada imagen puede pesar hasta 2 MB", partida+1)
		}
		total += file.Size
		if total > 15<<20 {
			return fmt.Errorf("las imágenes de la cotización no pueden superar 15 MB en total")
		}
		f, err := file.Open()
		if err != nil {
			return fmt.Errorf("partida %d: no se pudo leer la imagen", partida+1)
		}
		data, err := io.ReadAll(io.LimitReader(f, maxImagenPartida+1))
		f.Close()
		cfg, format, configErr := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || len(data) > maxImagenPartida || configErr != nil ||
			(format != "jpeg" && format != "png") || cfg.Width < 1 || cfg.Height < 1 ||
			cfg.Width > 4096 || cfg.Height > 4096 {
			return fmt.Errorf("partida %d: selecciona un PNG o JPG válido, de hasta 4096 px por lado y 2 MB", partida+1)
		}
		if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
			return fmt.Errorf("partida %d: la imagen está dañada", partida+1)
		}
		req.Partidas[partida].imagenes[slot-1] = data
	}
	return nil
}

func (h *CotizacionHandler) ImagenPartida(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	slot, slotErr := strconv.Atoi(c.Params("slot"))
	if err != nil || id < 1 || slotErr != nil || slot < 1 || slot > 2 {
		return c.SendStatus(404)
	}
	data, err := h.uc.ImagenPartida(c.Context(), id, slot)
	if errors.Is(err, domain.ErrCotizacionNotFound) {
		return c.SendStatus(404)
	}
	if err != nil {
		return internalError(c, "leer imagen de partida", err)
	}
	c.Set("Content-Type", http.DetectContentType(data))
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Cache-Control", "private, no-store")
	return c.Send(data)
}

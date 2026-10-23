package handlers

import (
	"errors"
	"strconv"
	"strings"

	"cotizaciones-api/internal/domain"
	"cotizaciones-api/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

type ClienteHandler struct {
	uc *usecase.ClienteUseCase
}

func NewClienteHandler(uc *usecase.ClienteUseCase) *ClienteHandler {
	return &ClienteHandler{uc: uc}
}

// El JSON expuesto usa exactamente los mismos nombres de campo que el frontend
// (idCliente, nombreComercial, ...), para que la pantalla consuma la API sin
// transformar nada.
type clienteResponse struct {
	IDCliente        int64  `json:"idCliente"`
	NombreComercial  string `json:"nombreComercial"`
	TipoCliente      string `json:"tipoCliente"`
	RFC              string `json:"rfc"`
	CorreoContacto   string `json:"correoContacto"`
	TelefonoContacto string `json:"telefonoContacto"`
	AtencionA        string `json:"atencionA"`
}

type clienteRequest struct {
	NombreComercial  string `json:"nombreComercial"`
	TipoCliente      string `json:"tipoCliente"`
	RFC              string `json:"rfc"`
	CorreoContacto   string `json:"correoContacto"`
	TelefonoContacto string `json:"telefonoContacto"`
	AtencionA        string `json:"atencionA"`
}

func toClienteResponse(c domain.Cliente) clienteResponse {
	return clienteResponse{
		IDCliente:        c.ID,
		NombreComercial:  c.NombreComercial,
		TipoCliente:      c.TipoCliente,
		RFC:              c.RFC,
		CorreoContacto:   c.CorreoContacto,
		TelefonoContacto: c.TelefonoContacto,
		AtencionA:        c.AtencionA,
	}
}

func (h *ClienteHandler) List(c fiber.Ctx) error {
	clientes, err := h.uc.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error interno"})
	}

	res := make([]clienteResponse, 0, len(clientes))
	for _, cl := range clientes {
		res = append(res, toClienteResponse(cl))
	}
	return c.JSON(res)
}

func (h *ClienteHandler) Create(c fiber.Ctx) error {
	var req clienteRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cuerpo de la solicitud inválido"})
	}

	if strings.TrimSpace(req.NombreComercial) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "el nombre del cliente es obligatorio"})
	}

	cliente := &domain.Cliente{
		NombreComercial:  req.NombreComercial,
		TipoCliente:      req.TipoCliente,
		RFC:              strings.TrimSpace(req.RFC),
		CorreoContacto:   strings.TrimSpace(req.CorreoContacto),
		TelefonoContacto: strings.TrimSpace(req.TelefonoContacto),
		AtencionA:        strings.TrimSpace(req.AtencionA),
	}

	creado, err := h.uc.Create(c.Context(), cliente)
	if err != nil {
		if errors.Is(err, domain.ErrClienteInvalido) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "datos de cliente inválidos"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error interno"})
	}

	return c.Status(fiber.StatusCreated).JSON(toClienteResponse(*creado))
}

func (h *ClienteHandler) Delete(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id inválido"})
	}

	if err := h.uc.Delete(c.Context(), id); err != nil {
		if errors.Is(err, domain.ErrClienteNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "cliente no encontrado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error interno"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

package handlers

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"cotizaciones-api/internal/domain"
	"cotizaciones-api/internal/infrastructure/documentos"
	"cotizaciones-api/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

type CotizacionHandler struct {
	uc         *usecase.CotizacionUseCase
	documentos *documentos.Servicio
}

func NewCotizacionHandler(uc *usecase.CotizacionUseCase, documentos *documentos.Servicio) *CotizacionHandler {
	return &CotizacionHandler{uc: uc, documentos: documentos}
}

func (h *CotizacionHandler) Documento(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Id inválido"})
	}
	pdf, err := h.documentos.Obtener(c.Context(), id)
	if errors.Is(err, domain.ErrCotizacionNotFound) {
		return c.SendStatus(404)
	}
	if err != nil {
		log.Printf("PDF de cotización %d: %v", id, err)
		return c.Status(503).JSON(fiber.Map{"error": "No se pudo generar el PDF. Intenta de nuevo."})
	}
	c.Set("Cache-Control", "private, no-store")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Vary("Accept")
	if c.Get("Accept") == "application/json" {
		return c.JSON(fiber.Map{"pdf": pdf})
	}
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", `inline; filename="cotizacion-`+strconv.FormatInt(id, 10)+`.pdf"`)
	return c.Send(pdf)
}

// ---- Respuesta (mismos nombres de campo que el tipo Cotizacion del frontend) ----

type partidaResponse struct {
	PrecioUnitarioCompra     float64 `json:"precioUnitarioCompra"`
	PorcentajeDescuento      float64 `json:"porcentajeDescuento"`
	TieneImagen              [2]bool `json:"tieneImagen"`
	ID                       int64   `json:"id"`
	NumeroPartida            int     `json:"numeroPartida"`
	Descripcion              string  `json:"descripcion"`
	EspecificacionesTecnicas string  `json:"especificacionesTecnicas"`
	SKU                      string  `json:"sku"`
	Cantidad                 float64 `json:"cantidad"`
	Unidad                   string  `json:"unidad"`
	PrecioUnitarioVenta      float64 `json:"precioUnitarioVenta"`
	Importe                  float64 `json:"importe"`
	RequiereCompra           bool    `json:"requiereCompra"`
	EstadoCompra             *string `json:"estadoCompra"`
}

type cotizacionResponse struct {
	IDCliente          *int64            `json:"idCliente"`
	CorreoContacto     string            `json:"correoContacto"`
	TelefonoContacto   string            `json:"telefonoContacto"`
	PreciosIncluyenIva bool              `json:"preciosIncluyenIva"`
	PorcentajeIva      float64           `json:"porcentajeIva"`
	PorcentajeAnticipo *float64          `json:"porcentajeAnticipo"`
	TiempoEntrega      string            `json:"tiempoEntrega"`
	CondicionesPago    string            `json:"condicionesPago"`
	NotasComerciales   string            `json:"notasComerciales"`
	MetodosPago        string            `json:"metodosPago"`
	ID                 int64             `json:"id"`
	Folio              string            `json:"folio"`
	Version            int               `json:"version"`
	Estado             string            `json:"estado"`
	FechaCreacion      string            `json:"fechaCreacion"`
	TipoCliente        string            `json:"tipoCliente"`
	RazonSocial        string            `json:"razonSocial"`
	RFC                string            `json:"rfc"`
	NombreCliente      string            `json:"nombreCliente"`
	NombreComercial    string            `json:"nombreComercial"`
	AtencionA          string            `json:"atencionA"`
	NombreAsesor       string            `json:"nombreAsesor"`
	Moneda             string            `json:"moneda"`
	VigenciaHasta      string            `json:"vigenciaHasta"`
	DiasParaVencer     *int              `json:"diasParaVencer"`
	ProximaAVencer     bool              `json:"proximaAVencer"`
	Costo              float64           `json:"costo"`
	Subtotal           float64           `json:"subtotal"`
	Impuestos          float64           `json:"impuestos"`
	Total              float64           `json:"total"`
	UtilidadEstimada   float64           `json:"utilidadEstimada"`
	Partidas           []partidaResponse `json:"partidas"`
}

func toCotizacionResponse(c domain.Cotizacion) cotizacionResponse {
	partidas := make([]partidaResponse, 0, len(c.Partidas))
	for _, p := range c.Partidas {
		partidas = append(partidas, partidaResponse{
			PrecioUnitarioCompra:     p.PrecioUnitarioCompra,
			PorcentajeDescuento:      p.PorcentajeDescuento,
			TieneImagen:              p.TieneImagen,
			ID:                       p.ID,
			NumeroPartida:            p.NumeroPartida,
			Descripcion:              p.Descripcion,
			EspecificacionesTecnicas: p.EspecificacionesTecnicas,
			SKU:                      p.SKU,
			Cantidad:                 p.Cantidad,
			Unidad:                   p.Unidad,
			PrecioUnitarioVenta:      p.PrecioUnitarioVenta,
			Importe:                  p.Importe,
			RequiereCompra:           p.RequiereCompra,
			EstadoCompra:             p.EstadoCompra,
		})
	}
	vigencia := ""
	if c.VigenciaHasta != nil {
		vigencia = c.VigenciaHasta.Format("2006-01-02")
	}
	return cotizacionResponse{
		IDCliente:          c.IDCliente,
		CorreoContacto:     c.CorreoContacto,
		TelefonoContacto:   c.TelefonoContacto,
		PreciosIncluyenIva: c.PreciosIncluyenIva,
		PorcentajeIva:      c.PorcentajeIva,
		PorcentajeAnticipo: c.PorcentajeAnticipo,
		TiempoEntrega:      c.TiempoEntrega,
		CondicionesPago:    c.CondicionesPago,
		NotasComerciales:   c.NotasComerciales,
		MetodosPago:        c.MetodosPago,
		ID:                 c.ID,
		Folio:              c.Folio,
		Version:            c.Version,
		Estado:             c.Estado,
		FechaCreacion:      c.FechaCreacion.Format("2006-01-02"),
		TipoCliente:        c.TipoCliente,
		RazonSocial:        c.RazonSocial,
		RFC:                c.RFC,
		NombreCliente:      c.NombreCliente,
		NombreComercial:    c.NombreComercial,
		AtencionA:          c.AtencionA,
		NombreAsesor:       c.NombreAsesor,
		Moneda:             c.Moneda,
		VigenciaHasta:      vigencia,
		DiasParaVencer:     c.DiasParaVencer,
		ProximaAVencer:     c.ProximaAVencer,
		Costo:              c.Costo,
		Subtotal:           c.Subtotal,
		Impuestos:          c.Impuestos,
		Total:              c.Total,
		UtilidadEstimada:   c.UtilidadEstimada,
		Partidas:           partidas,
	}
}

// ---- Petición de creación (lo que manda el formulario) ----

type partidaRequest struct {
	OrigenID                 int64 `json:"origenId"`
	imagenes                 [2][]byte
	SKU                      string  `json:"sku"`
	Descripcion              string  `json:"descripcion"`
	EspecificacionesTecnicas string  `json:"especificacionesTecnicas"`
	Unidad                   string  `json:"unidad"`
	Cantidad                 float64 `json:"cantidad"`
	PrecioUnitarioCompra     float64 `json:"precioUnitarioCompra"`
	PrecioUnitarioVenta      float64 `json:"precioUnitarioVenta"`
	PorcentajeDescuento      float64 `json:"porcentajeDescuento"`
	RequiereCompra           bool    `json:"requiereCompra"`
}

type cotizacionRequest struct {
	IDCliente          *int64           `json:"idCliente"`
	TipoCliente        string           `json:"tipoCliente"`
	NombreComercial    string           `json:"nombreComercial"`
	AtencionA          string           `json:"atencionA"`
	RFC                string           `json:"rfc"`
	CorreoContacto     string           `json:"correoContacto"`
	TelefonoContacto   string           `json:"telefonoContacto"`
	VigenciaHasta      string           `json:"vigenciaHasta"`
	Moneda             string           `json:"moneda"`
	PreciosIncluyenIva bool             `json:"preciosIncluyenIva"`
	PorcentajeIva      float64          `json:"porcentajeIva"`
	PorcentajeAnticipo *float64         `json:"porcentajeAnticipo"`
	TiempoEntrega      string           `json:"tiempoEntrega"`
	CondicionesPago    string           `json:"condicionesPago"`
	NotasComerciales   string           `json:"notasComerciales"`
	MetodosPago        string           `json:"metodosPago"`
	Partidas           []partidaRequest `json:"partidas"`
}

func (h *CotizacionHandler) List(c fiber.Ctx) error {
	cots, err := h.uc.List(c.Context())
	if err != nil {
		return internalError(c, "listar cotizaciones", err)
	}
	res := make([]cotizacionResponse, 0, len(cots))
	for _, ct := range cots {
		res = append(res, toCotizacionResponse(ct))
	}
	return c.JSON(res)
}

func (h *CotizacionHandler) Create(c fiber.Ctx) error {
	return h.save(c, false)
}

func (h *CotizacionHandler) Update(c fiber.Ctx) error {
	return h.save(c, true)
}

func (h *CotizacionHandler) save(c fiber.Ctx, editing bool) error {
	var req cotizacionRequest
	if err := leerCotizacion(c, &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if strings.TrimSpace(req.NombreComercial) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "el nombre del cliente es obligatorio"})
	}
	if len(req.Partidas) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "la cotización debe tener al menos una partida"})
	}

	var vigencia *time.Time
	if err := validarLimitesCotizacion(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if s := strings.TrimSpace(req.VigenciaHasta); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			vigencia = &t
		} else {
			return c.Status(400).JSON(fiber.Map{"error": "Fecha de vigencia inválida"})
		}
	}

	nueva := domain.CotizacionNueva{
		IDCliente:          req.IDCliente,
		TipoCliente:        req.TipoCliente,
		NombreComercial:    strings.TrimSpace(req.NombreComercial),
		AtencionA:          strings.TrimSpace(req.AtencionA),
		RFC:                strings.TrimSpace(req.RFC),
		CorreoContacto:     strings.TrimSpace(req.CorreoContacto),
		TelefonoContacto:   strings.TrimSpace(req.TelefonoContacto),
		VigenciaHasta:      vigencia,
		Moneda:             req.Moneda,
		PreciosIncluyenIva: req.PreciosIncluyenIva,
		PorcentajeIva:      req.PorcentajeIva,
		PorcentajeAnticipo: req.PorcentajeAnticipo,
		TiempoEntrega:      req.TiempoEntrega,
		CondicionesPago:    req.CondicionesPago,
		NotasComerciales:   req.NotasComerciales,
		MetodosPago:        req.MetodosPago,
	}
	for _, p := range req.Partidas {
		nueva.Partidas = append(nueva.Partidas, domain.PartidaNueva{
			OrigenID:                 p.OrigenID,
			Imagenes:                 p.imagenes,
			SKU:                      p.SKU,
			Descripcion:              p.Descripcion,
			EspecificacionesTecnicas: p.EspecificacionesTecnicas,
			Unidad:                   p.Unidad,
			Cantidad:                 p.Cantidad,
			PrecioUnitarioCompra:     p.PrecioUnitarioCompra,
			PrecioUnitarioVenta:      p.PrecioUnitarioVenta,
			PorcentajeDescuento:      p.PorcentajeDescuento,
			RequiereCompra:           p.RequiereCompra,
		})
	}

	userID, _ := c.Locals("user_id").(int64)
	var creada *domain.Cotizacion
	var err error
	if editing {
		id, parseErr := strconv.ParseInt(c.Params("id"), 10, 64)
		if parseErr != nil || id < 1 {
			return c.Status(400).JSON(fiber.Map{"error": "Id inválido"})
		}
		creada, err = h.uc.Update(c.Context(), userID, id, nueva)
	} else {
		creada, err = h.uc.Create(c.Context(), userID, nueva)
	}
	if err != nil {
		if errors.Is(err, domain.ErrCotizacionArchivada) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, domain.ErrCotizacionNotFound) || errors.Is(err, domain.ErrClienteNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "Cotización o cliente no encontrado"})
		}
		if errors.Is(err, domain.ErrCotizacionInvalida) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "datos de cotización inválidos"})
		}
		return internalError(c, "crear cotización", err)
	}
	if _, err := h.documentos.Obtener(c.Context(), creada.ID); err != nil {
		log.Printf("PDF pendiente de la cotización %d: %v", creada.ID, err)
	}
	if editing {
		return c.JSON(toCotizacionResponse(*creada))
	}
	return c.Status(fiber.StatusCreated).JSON(toCotizacionResponse(*creada))
}

func (h *CotizacionHandler) Delete(c fiber.Ctx) error {
	return cambiarEstadoCotizacion(c, h.uc.Delete, "eliminar cotización")
}

func (h *CotizacionHandler) Archive(c fiber.Ctx) error {
	return cambiarEstadoCotizacion(c, h.uc.Archive, "archivar cotización")
}

func cambiarEstadoCotizacion(c fiber.Ctx, cambiar func(context.Context, int64) error, operacion string) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id inválido"})
	}
	if err := cambiar(c.Context(), id); err != nil {
		if errors.Is(err, domain.ErrCotizacionArchivada) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, domain.ErrCotizacionNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "cotización no encontrada"})
		}
		return internalError(c, operacion, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

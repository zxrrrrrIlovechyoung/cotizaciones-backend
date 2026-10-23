package handlers

import (
	"log"

	"github.com/gofiber/fiber/v3"
)

// internalError centraliza la respuesta de los errores 500. Registra en el log
// del SERVIDOR la causa real (con método y ruta) para poder diagnosticar, pero
// al cliente solo le responde un mensaje genérico: el detalle técnico nunca se
// expone a quien hace la petición.
//
// Importante: aquí NO se registran datos sensibles (contraseñas, tokens ni el
// cuerpo de la solicitud), solo el error y la ruta donde ocurrió.
func internalError(c fiber.Ctx, contexto string, err error) error {
	log.Printf("[ERROR 500] %s %s — %s: %v", c.Method(), c.Path(), contexto, err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error interno"})
}

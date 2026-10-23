package domain

import "errors"

// Errores de negocio. Cualquier capa externa (handlers, usecase) los reconoce
// con errors.Is, sin acoplarse a errores específicos de pgx o de Fiber.
var (
	ErrUserNotFound       = errors.New("usuario no encontrado")
	ErrInvalidCredentials = errors.New("credenciales inválidas")

	ErrClienteNotFound = errors.New("cliente no encontrado")
	ErrClienteInvalido = errors.New("datos de cliente inválidos")

	ErrCotizacionNotFound = errors.New("cotización no encontrada")
	ErrCotizacionInvalida = errors.New("datos de cotización inválidos")
)

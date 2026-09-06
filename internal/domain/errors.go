package domain

import "errors"

// Errores de negocio. Cualquier capa externa (handlers, usecase) los reconoce
// con errors.Is, sin acoplarse a errores específicos de pgx o de Fiber.
var (
	ErrUserNotFound       = errors.New("usuario no encontrado")
	ErrInvalidCredentials = errors.New("credenciales inválidas")
)

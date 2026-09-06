package domain

import "context"

// UserRepository es un "puerto de salida": el usecase depende de esta interfaz,
// nunca de una implementación concreta. Quien la implementa (infrastructure/postgres)
// vive fuera del dominio.
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
}

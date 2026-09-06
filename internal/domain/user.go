package domain

import "time"

// User es la entidad de negocio pura. No tiene tags de JSON ni sabe nada
// de HTTP o de la base de datos: eso es responsabilidad de las capas externas.
type User struct {
	ID           int64
	Nombre       string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

package usecase

// PasswordHasher y TokenGenerator son puertos de salida: el usecase los necesita
// para hacer su trabajo, pero no le importa si detrás hay bcrypt+JWT o cualquier
// otra tecnología. Eso lo decide infrastructure/security.

type PasswordHasher interface {
	Compare(hash, password string) error
}

type TokenGenerator interface {
	Generate(userID int64, email string) (string, error)
}

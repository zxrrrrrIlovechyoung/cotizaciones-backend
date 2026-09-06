package security

import (
	"cotizaciones-api/internal/usecase"

	"golang.org/x/crypto/bcrypt"
)

var _ usecase.PasswordHasher = (*BcryptHasher)(nil)

type BcryptHasher struct{}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{}
}

func (BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

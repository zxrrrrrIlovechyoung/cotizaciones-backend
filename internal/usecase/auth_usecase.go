package usecase

import (
	"context"
	"errors"

	"cotizaciones-api/internal/domain"
)

type AuthUseCase struct {
	userRepo domain.UserRepository
	hasher   PasswordHasher
	tokens   TokenGenerator
}

func NewAuthUseCase(userRepo domain.UserRepository, hasher PasswordHasher, tokens TokenGenerator) *AuthUseCase {
	return &AuthUseCase{userRepo: userRepo, hasher: hasher, tokens: tokens}
}

type LoginResult struct {
	Token string
	User  *domain.User
}

// Login orquesta el caso de uso: busca el usuario, valida la contraseña y
// genera el token. No sabe si el usuario viene de Postgres ni si el token es JWT.
func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := uc.hasher.Compare(user.PasswordHash, password); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	token, err := uc.tokens.Generate(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &LoginResult{Token: token, User: user}, nil
}

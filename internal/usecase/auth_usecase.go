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

func (uc *AuthUseCase) EmailAvailable(ctx context.Context, email string) (bool, error) {
	exists, err := uc.userRepo.EmailExists(ctx, email)
	return !exists, err
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

// Profile devuelve el usuario actual a partir de su ID (el que viene en el token ya
// validado). Consulta la BD para reflejar el estado real: si el usuario fue borrado,
// el token deja de servir aunque todavía no haya expirado.
func (uc *AuthUseCase) Profile(ctx context.Context, userID int64) (*domain.User, error) {
	return uc.userRepo.FindByID(ctx, userID)
}

func (uc *AuthUseCase) VerificarContrasenaActual(ctx context.Context, userID int64, actual string) error {
	_, err := uc.usuarioConContrasenaActual(ctx, userID, actual)
	return err
}

func (uc *AuthUseCase) usuarioConContrasenaActual(ctx context.Context, userID int64, actual string) (*domain.User, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := uc.hasher.Compare(user.PasswordHash, actual); err != nil {
		return nil, domain.ErrContrasenaActualIncorrecta
	}
	return user, nil
}

// ChangePassword cambia la contraseña del usuario: primero verifica que la
// contraseña actual sea correcta, luego guarda el hash de la nueva. Nunca se
// guarda la contraseña en texto plano.
func (uc *AuthUseCase) ChangePassword(ctx context.Context, userID int64, actual, nueva string) error {
	if err := domain.ValidarContrasena(nueva); err != nil {
		return err
	}
	user, err := uc.usuarioConContrasenaActual(ctx, userID, actual)
	if err != nil {
		return err
	}
	if actual == nueva || uc.hasher.Compare(user.PasswordHash, nueva) == nil {
		return domain.ErrContrasenaSinCambios
	}
	nuevoHash, err := uc.hasher.Hash(nueva)
	if err != nil {
		return err
	}
	return uc.userRepo.UpdatePassword(ctx, userID, user.PasswordHash, nuevoHash)
}

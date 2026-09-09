package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"cotizaciones-api/internal/domain"
	"cotizaciones-api/internal/usecase"
)

// --- Dobles de prueba: implementan los puertos sin tocar Postgres/JWT/bcrypt ---

type fakeUserRepo struct {
	user *domain.User
	err  error
}

func (f fakeUserRepo) FindByEmail(_ context.Context, _ string) (*domain.User, error) {
	return f.user, f.err
}

func (f fakeUserRepo) FindByID(_ context.Context, _ int64) (*domain.User, error) {
	return f.user, f.err
}

type fakeHasher struct{ err error }

func (f fakeHasher) Compare(_, _ string) error { return f.err }

type fakeTokens struct {
	token string
	err   error
}

func (f fakeTokens) Generate(_ int64, _ string) (string, error) { return f.token, f.err }

// --- Tests ---

func TestLogin_OK(t *testing.T) {
	repo := fakeUserRepo{user: &domain.User{ID: 1, Email: "a@b.com", CreatedAt: time.Now()}}
	uc := usecase.NewAuthUseCase(repo, fakeHasher{}, fakeTokens{token: "tok-123"})

	res, err := uc.Login(context.Background(), "a@b.com", "secret")
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if res.Token != "tok-123" || res.User.ID != 1 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
}

func TestLogin_UsuarioNoExiste_DevuelveCredencialesInvalidas(t *testing.T) {
	repo := fakeUserRepo{err: domain.ErrUserNotFound}
	uc := usecase.NewAuthUseCase(repo, fakeHasher{}, fakeTokens{})

	_, err := uc.Login(context.Background(), "x@y.com", "secret")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("se esperaba ErrInvalidCredentials, se obtuvo: %v", err)
	}
}

func TestLogin_PasswordIncorrecta_DevuelveCredencialesInvalidas(t *testing.T) {
	repo := fakeUserRepo{user: &domain.User{ID: 1, Email: "a@b.com"}}
	uc := usecase.NewAuthUseCase(repo, fakeHasher{err: errors.New("mismatch")}, fakeTokens{})

	_, err := uc.Login(context.Background(), "a@b.com", "mala")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("se esperaba ErrInvalidCredentials, se obtuvo: %v", err)
	}
}

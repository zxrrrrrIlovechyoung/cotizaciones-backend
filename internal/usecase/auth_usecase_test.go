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

func (f fakeUserRepo) EmailExists(_ context.Context, _ string) (bool, error) { return false, nil }

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

func (f fakeUserRepo) UpdatePassword(_ context.Context, _ int64, _, _ string) error {
	return f.err
}

type fakeHasher struct {
	err      error
	password string
}

func (f fakeHasher) Compare(_, password string) error {
	if f.password != "" && password != f.password {
		return errors.New("mismatch")
	}
	return f.err
}

func (f fakeHasher) Hash(password string) (string, error) { return "hash-de:" + password, nil }

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

func TestChangePassword_OK(t *testing.T) {
	repo := fakeUserRepo{user: &domain.User{ID: 1, PasswordHash: "hash-viejo"}}
	uc := usecase.NewAuthUseCase(repo, fakeHasher{password: "actual"}, fakeTokens{})
	if err := uc.ChangePassword(context.Background(), 1, "actual", "NuevaSegura!1"); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
}

func TestChangePassword_ActualIncorrecta(t *testing.T) {
	repo := fakeUserRepo{user: &domain.User{ID: 1, PasswordHash: "hash-viejo"}}
	uc := usecase.NewAuthUseCase(repo, fakeHasher{err: errors.New("mismatch")}, fakeTokens{})

	err := uc.ChangePassword(context.Background(), 1, "mala", "NuevaSegura!1")
	if !errors.Is(err, domain.ErrContrasenaActualIncorrecta) {
		t.Fatalf("se esperaba ErrContrasenaActualIncorrecta, se obtuvo: %v", err)
	}
}

func TestChangePassword_RechazaClaveRepetidaODebil(t *testing.T) {
	const actual = "ActualSegura!1"
	repo := fakeUserRepo{user: &domain.User{ID: 1, PasswordHash: "hash-viejo"}}
	uc := usecase.NewAuthUseCase(repo, fakeHasher{password: actual}, fakeTokens{})
	for _, caso := range []struct {
		nueva string
		error error
	}{
		{actual, domain.ErrContrasenaSinCambios},
		{"NuevaSegura", domain.ErrContrasenaInvalida},
	} {
		if err := uc.ChangePassword(context.Background(), 1, actual, caso.nueva); !errors.Is(err, caso.error) {
			t.Fatalf("se esperaba %v, se obtuvo %v", caso.error, err)
		}
	}
}

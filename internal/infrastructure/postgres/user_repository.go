package postgres

import (
	"context"
	"errors"

	"cotizaciones-api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository implementa domain.UserRepository usando pgx.
// El compilador verifica esta línea: si un método no coincide con la interfaz, no compila.
var _ domain.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	row := r.db.QueryRow(ctx,
		`SELECT id, nombre, email, password_hash, created_at FROM users WHERE email = $1`,
		email,
	)
	if err := row.Scan(&u.ID, &u.Nombre, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	row := r.db.QueryRow(ctx,
		`SELECT id, nombre, email, password_hash, created_at FROM users WHERE id = $1`,
		id,
	)
	if err := row.Scan(&u.ID, &u.Nombre, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

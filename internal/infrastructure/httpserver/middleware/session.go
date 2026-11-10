package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func SessionRequired(db *pgxpool.Pool) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := c.Cookies("cotizaciones_session")
		if token == "" || subtle.ConstantTimeCompare([]byte(c.Get("Authorization")), []byte("Bearer "+token)) != 1 {
			return c.SendStatus(401)
		}
		var valid bool
		if err := db.QueryRow(c.Context(), `SELECT EXISTS(SELECT 1 FROM auth_sessions s JOIN users u ON u.id=s.user_id WHERE token_hash=$1 AND expires_at>now())`, TokenHash(token)).Scan(&valid); err != nil || !valid {
			return c.SendStatus(401)
		}
		return c.Next()
	}
}

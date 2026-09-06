package security

import (
	"time"

	"cotizaciones-api/internal/usecase"

	"github.com/golang-jwt/jwt/v5"
)

var _ usecase.TokenGenerator = (*JWTGenerator)(nil)

type Claims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type JWTGenerator struct {
	secret          string
	expirationHours int
}

func NewJWTGenerator(secret string, expirationHours int) *JWTGenerator {
	return &JWTGenerator{secret: secret, expirationHours: expirationHours}
}

func (g *JWTGenerator) Generate(userID int64, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(g.expirationHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(g.secret))
}

// ParseToken se usa desde el middleware HTTP para validar el token entrante.
// No forma parte de la interfaz TokenGenerator (que solo genera) porque
// validar un token es una responsabilidad distinta, usada por otra capa.
func ParseToken(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}

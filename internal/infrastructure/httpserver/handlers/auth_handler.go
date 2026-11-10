package handlers

import (
	"cotizaciones-api/internal/infrastructure/httpserver/middleware"
	"cotizaciones-api/internal/infrastructure/security"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"cotizaciones-api/internal/domain"
	"cotizaciones-api/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

// emailRegex es una validación de formato deliberadamente laxa: solo descarta
// entradas obviamente inválidas. La verdad sobre si el email existe la tiene la BD.
var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type AuthHandler struct {
	authUseCase *usecase.AuthUseCase
	db          *pgxpool.Pool
}

func NewAuthHandler(authUseCase *usecase.AuthUseCase, db *pgxpool.Pool) *AuthHandler {
	return &AuthHandler{authUseCase: authUseCase, db: db}
}

func sessionCookie(value string, expires time.Time) *fiber.Cookie {
	return &fiber.Cookie{Name: "cotizaciones_session", Value: value, Path: "/api", HTTPOnly: true, Secure: os.Getenv("APP_ENV") == "production", SameSite: "Strict", Expires: expires}
}
func (h *AuthHandler) Session(c fiber.Ctx) error {
	token := c.Cookies("cotizaciones_session")
	if token == "" {
		return c.SendStatus(401)
	}
	var id int64
	if err := h.db.QueryRow(c.Context(), `SELECT user_id FROM auth_sessions WHERE token_hash=$1 AND expires_at>now()`, middleware.TokenHash(token)).Scan(&id); err != nil {
		return c.SendStatus(401)
	}
	user, err := h.authUseCase.Profile(c.Context(), id)
	if err != nil {
		return c.SendStatus(401)
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(fiber.Map{"token": token, "user": userResponse{ID: user.ID, Nombre: user.Nombre, Email: user.Email}})
}
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	_, err := h.db.Exec(c.Context(), `DELETE FROM auth_sessions WHERE token_hash=$1`, middleware.TokenHash(c.Cookies("cotizaciones_session")))
	if err != nil {
		return internalError(c, "cerrar sesión", err)
	}
	c.Cookie(sessionCookie("", time.Now().Add(-time.Hour)))
	return c.SendStatus(204)
}

func (h *AuthHandler) EmailAvailable(c fiber.Ctx) error {
	c.Set("Cache-Control", "no-store")
	var req struct {
		Email string `json:"email"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Solicitud inválida"})
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if utf8.RuneCountInString(email) > 100 || !emailRegex.MatchString(email) {
		return c.Status(400).JSON(fiber.Map{"error": "Correo inválido"})
	}
	available, err := h.authUseCase.EmailAvailable(c.Context(), email)
	if err != nil {
		return internalError(c, "disponibilidad de correo", err)
	}
	return c.JSON(fiber.Map{"disponible": available})
}

// Los DTOs de request/response viven aquí, en el borde HTTP, y no en domain.
// Así el shape del JSON puede cambiar sin tocar la entidad de negocio.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) ValidateRegistration(c fiber.Ctx) error {
	var req struct {
		Nombre   string `json:"nombre"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Solicitud inválida"})
	}
	n := utf8.RuneCountInString(req.Nombre)
	if n < 2 || n > 60 || strings.TrimSpace(req.Nombre) == "" || utf8.RuneCountInString(req.Email) > 100 || !emailRegex.MatchString(strings.TrimSpace(req.Email)) {
		return c.Status(400).JSON(fiber.Map{"error": "Nombre: 2 a 60 caracteres; correo: máximo 100"})
	}
	if err := domain.ValidarContrasena(req.Password); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

type userResponse struct {
	ID        int64  `json:"id"`
	Nombre    string `json:"nombre"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req loginRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cuerpo de la solicitud inválido",
		})
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if utf8.RuneCountInString(req.Email) > 100 || !emailRegex.MatchString(req.Email) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "el email no tiene un formato válido",
		})
	}
	if utf8.RuneCountInString(req.Password) < 6 || utf8.RuneCountInString(req.Password) > 20 || len(req.Password) > 72 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "la contraseña debe tener al menos 6 caracteres",
		})
	}

	result, err := h.authUseCase.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "credenciales inválidas",
			})
		}
		return internalError(c, "login", err)
	}

	claims, err := security.ParseToken(os.Getenv("JWT_SECRET"), result.Token)
	if err != nil {
		return internalError(c, "sesión", err)
	}
	tag, err := h.db.Exec(c.Context(), `INSERT INTO auth_sessions(token_hash,user_id,expires_at)
		SELECT $1,id,$3 FROM users WHERE id=$2 AND password_hash=$4 FOR SHARE`,
		middleware.TokenHash(result.Token), result.User.ID, claims.ExpiresAt.Time, result.User.PasswordHash)
	if err != nil {
		return internalError(c, "guardar sesión", err)
	}
	if tag.RowsAffected() == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "credenciales inválidas"})
	}
	c.Cookie(sessionCookie(result.Token, claims.ExpiresAt.Time))
	c.Set("Cache-Control", "no-store")
	return c.JSON(loginResponse{
		Token: result.Token,
		User: userResponse{
			ID:        result.User.ID,
			Nombre:    result.User.Nombre,
			Email:     result.User.Email,
			CreatedAt: result.User.CreatedAt.Format(time.RFC3339),
		},
	})
}

type cambiarPasswordRequest struct {
	Actual    string `json:"actual"`
	Nueva     string `json:"nueva"`
	Confirmar string `json:"confirmar"`
}

func contrasenaActualAdmitida(actual string) bool {
	longitud := utf8.RuneCountInString(actual)
	return longitud >= 6 && longitud <= 20 && len(actual) <= 72
}

func (h *AuthHandler) VerifyCurrentPassword(c fiber.Ctx) error {
	var req struct {
		Actual string `json:"actual"`
	}
	if err := c.Bind().Body(&req); err != nil || !contrasenaActualAdmitida(req.Actual) {
		return c.Status(400).JSON(fiber.Map{"error": "Ingresa una contraseña actual válida"})
	}
	c.Set("Cache-Control", "no-store")
	err := h.authUseCase.VerificarContrasenaActual(c.Context(), domain.Owner(c.Context()), req.Actual)
	if errors.Is(err, domain.ErrContrasenaActualIncorrecta) {
		return c.JSON(fiber.Map{"valida": false})
	}
	if errors.Is(err, domain.ErrUserNotFound) {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	if err != nil {
		return internalError(c, "verificar contraseña actual", err)
	}
	return c.JSON(fiber.Map{"valida": true})
}

func (h *AuthHandler) ChangePassword(c fiber.Ctx) error {
	var req cambiarPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cuerpo de la solicitud inválido"})
	}
	if !contrasenaActualAdmitida(req.Actual) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Ingresa una contraseña actual válida"})
	}
	if req.Nueva != req.Confirmar {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Las contraseñas nuevas no coinciden"})
	}

	userID, _ := c.Locals("user_id").(int64)
	if err := h.authUseCase.ChangePassword(c.Context(), userID, req.Actual, req.Nueva); err != nil {
		if errors.Is(err, domain.ErrContrasenaInvalida) || errors.Is(err, domain.ErrContrasenaSinCambios) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, domain.ErrContrasenaActualIncorrecta) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "la contraseña actual es incorrecta"})
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "el usuario ya no existe"})
		}
		return internalError(c, "cambiar contraseña", err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *AuthHandler) Me(c fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(int64)

	user, err := h.authUseCase.Profile(c.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "el usuario ya no existe",
			})
		}
		return internalError(c, "perfil", err)
	}

	return c.JSON(userResponse{
		ID:        user.ID,
		Nombre:    user.Nombre,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	})
}

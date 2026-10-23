package handlers

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"cotizaciones-api/internal/domain"
	"cotizaciones-api/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

// emailRegex es una validación de formato deliberadamente laxa: solo descarta
// entradas obviamente inválidas. La verdad sobre si el email existe la tiene la BD.
var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type AuthHandler struct {
	authUseCase *usecase.AuthUseCase
}

func NewAuthHandler(authUseCase *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{authUseCase: authUseCase}
}

// Los DTOs de request/response viven aquí, en el borde HTTP, y no en domain.
// Así el shape del JSON puede cambiar sin tocar la entidad de negocio.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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

	if !emailRegex.MatchString(req.Email) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "el email no tiene un formato válido",
		})
	}
	if len(req.Password) < 6 {
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
	Actual string `json:"actual"`
	Nueva  string `json:"nueva"`
}

func (h *AuthHandler) ChangePassword(c fiber.Ctx) error {
	var req cambiarPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cuerpo de la solicitud inválido"})
	}
	if len(req.Nueva) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "la nueva contraseña debe tener al menos 6 caracteres"})
	}

	userID, _ := c.Locals("user_id").(int64)
	if err := h.authUseCase.ChangePassword(c.Context(), userID, req.Actual, req.Nueva); err != nil {
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

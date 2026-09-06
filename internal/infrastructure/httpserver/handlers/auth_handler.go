package handlers

import (
	"errors"
	"time"

	"cotizaciones-api/internal/domain"
	"cotizaciones-api/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

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

	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "email y password son obligatorios",
		})
	}

	result, err := h.authUseCase.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "credenciales inválidas",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "error interno",
		})
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

func (h *AuthHandler) Me(c fiber.Ctx) error {
	email, _ := c.Locals("email").(string)
	userID, _ := c.Locals("user_id").(int64)

	return c.JSON(fiber.Map{
		"user_id": userID,
		"email":   email,
	})
}

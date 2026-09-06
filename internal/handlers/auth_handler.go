package handlers

import (
	"errors"

	"cotizaciones-api/internal/auth"
	"cotizaciones-api/internal/config"
	"cotizaciones-api/internal/models"
	"cotizaciones-api/internal/repository"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	cfg      *config.Config
	userRepo *repository.UserRepository
}

func NewAuthHandler(cfg *config.Config, userRepo *repository.UserRepository) *AuthHandler {
	return &AuthHandler{cfg: cfg, userRepo: userRepo}
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req models.LoginRequest
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

	user, err := h.userRepo.FindByEmail(c.Context(), req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "credenciales inválidas",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "error al consultar el usuario",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "credenciales inválidas",
		})
	}

	token, err := auth.GenerateToken(h.cfg.JWTSecret, h.cfg.JWTExpirationHours, user.ID, user.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "error al generar el token",
		})
	}

	return c.JSON(models.LoginResponse{
		Token: token,
		User:  *user,
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

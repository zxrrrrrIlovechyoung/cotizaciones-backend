package handlers

import (
	"cotizaciones-api/internal/domain"
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"unicode/utf8"
)

func (h *PerfilHandler) Register(c fiber.Ctx) error {
	var req struct {
		Nombre   string `json:"nombre"`
		Email    string `json:"email"`
		Password string `json:"password"`
		domain.PerfilCuenta
	}
	if err := json.Unmarshal([]byte(c.FormValue("datos")), &req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Datos inválidos"})
	}
	req.Nombre = strings.TrimSpace(req.Nombre)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	n := utf8.RuneCountInString(req.Nombre)
	if n < 2 || n > 60 || len(req.Email) > 100 || !emailRegex.MatchString(req.Email) {
		return c.Status(400).JSON(fiber.Map{"error": "Revisa el nombre y el correo"})
	}
	if err := domain.ValidarContrasena(req.Password); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if err := req.PerfilCuenta.Validate(); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	var logo []byte
	mime := ""
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Formulario inválido"})
	}
	if files := form.File["logo"]; len(files) > 0 {
		logo, mime, err = leerLogoCuenta(files)
		if errors.Is(err, errLogoInvalido) {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		if err != nil {
			return internalError(c, "leer logo", err)
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return internalError(c, "proteger contraseña", err)
	}
	tx, err := h.db.Begin(c.Context())
	if err != nil {
		return internalError(c, "iniciar registro", err)
	}
	defer tx.Rollback(c.Context())
	var id int64
	err = tx.QueryRow(c.Context(), `INSERT INTO users(nombre,email,password_hash) VALUES($1,$2,$3) RETURNING id`, req.Nombre, req.Email, string(hash)).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return c.Status(409).JSON(fiber.Map{"error": "Este correo ya tiene una cuenta"})
		}
		return internalError(c, "crear usuario", err)
	}
	profile := req.PerfilCuenta
	_, err = tx.Exec(c.Context(), `INSERT INTO perfiles_cuenta(user_id,tipo,comercial,razon_social,rfc,telefono,direccion,logo,logo_tipo) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, profile.Tipo, profile.Comercial, profile.RazonSocial, profile.RFC, profile.Telefono, profile.Direccion, logo, mime)
	if err != nil {
		return internalError(c, "crear perfil", err)
	}
	if err = tx.Commit(c.Context()); err != nil {
		return internalError(c, "confirmar registro", err)
	}
	return c.Status(201).JSON(fiber.Map{"id": id, "message": "Cuenta creada correctamente"})
}

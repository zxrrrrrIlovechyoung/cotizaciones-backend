package handlers

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"cotizaciones-api/internal/domain"
	"cotizaciones-api/internal/infrastructure/httpserver/middleware"
	"cotizaciones-api/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EliminacionCuentaHandler struct {
	db     *pgxpool.Pool
	hasher usecase.PasswordHasher
	secret []byte
	enviar func(string, string) error
}

func NewEliminacionCuentaHandler(db *pgxpool.Pool, hasher usecase.PasswordHasher, secret string, enviar func(string, string) error) *EliminacionCuentaHandler {
	return &EliminacionCuentaHandler{db: db, hasher: hasher, secret: []byte(secret), enviar: enviar}
}

func (h *EliminacionCuentaHandler) Solicitar(c fiber.Ctx) error {
	if h.enviar == nil {
		return c.Status(503).JSON(fiber.Map{"error": "El envío de códigos no está configurado."})
	}
	codigo, err := generarCodigo()
	if err != nil {
		return internalError(c, "generar código de eliminación", err)
	}
	ctx := c.Context()
	id := domain.Owner(ctx)
	sesion := middleware.TokenHash(c.Cookies("cotizaciones_session"))
	hash := h.hashCodigo(sesion, codigo)
	var destino string
	err = h.db.QueryRow(ctx, `
		WITH solicitud AS (
			INSERT INTO eliminacion_cuenta AS e (user_id, sesion_hash, codigo_hash, expira_en, reenviar_desde)
			SELECT id, $2, $3, now()+interval '10 minutes', now()+interval '1 minute'
			FROM users WHERE id=$1
			ON CONFLICT (user_id) DO UPDATE SET sesion_hash=excluded.sesion_hash, codigo_hash=excluded.codigo_hash,
			  permiso_hash=NULL, intentos=0, expira_en=excluded.expira_en, reenviar_desde=excluded.reenviar_desde
			WHERE e.reenviar_desde <= now()
			RETURNING user_id
		)
		SELECT u.email FROM solicitud s JOIN users u ON u.id=s.user_id
	`, id, sesion, hash).Scan(&destino)
	if errors.Is(err, pgx.ErrNoRows) {
		return c.Status(429).JSON(fiber.Map{"error": "Espera un minuto antes de solicitar otro código."})
	}
	if err != nil {
		return internalError(c, "registrar código de eliminación", err)
	}
	if err := h.enviar(destino, codigo); err != nil {
		log.Printf("el servidor SMTP no pudo enviar un código de eliminación (%v)", err)
		limpieza, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_, _ = h.db.Exec(limpieza, `DELETE FROM eliminacion_cuenta WHERE user_id=$1 AND sesion_hash=$2 AND codigo_hash=$3 AND permiso_hash IS NULL`, id, sesion, hash)
		return c.Status(503).JSON(fiber.Map{"error": "No se pudo enviar el código. Intenta de nuevo."})
	}
	c.Set("Cache-Control", "no-store")
	return c.SendStatus(204)
}

func (h *EliminacionCuentaHandler) Verificar(c fiber.Ctx) error {
	var req struct {
		Codigo string `json:"codigo"`
	}
	if err := c.Bind().Body(&req); err != nil || !codigoRecuperacion.MatchString(req.Codigo) {
		return c.Status(400).JSON(fiber.Map{"error": "Ingresa un código de seis dígitos."})
	}
	permiso, err := generarPermiso()
	if err != nil {
		return internalError(c, "generar permiso de eliminación", err)
	}
	ctx := c.Context()
	sesion := middleware.TokenHash(c.Cookies("cotizaciones_session"))
	var aceptado bool
	err = h.db.QueryRow(ctx, `
		UPDATE eliminacion_cuenta SET intentos=intentos+1,
		  permiso_hash=CASE WHEN codigo_hash=$3 THEN $4::bytea ELSE NULL END,
		  expira_en=CASE WHEN codigo_hash=$3 THEN now()+interval '10 minutes' ELSE expira_en END
		WHERE user_id=$1 AND sesion_hash=$2 AND permiso_hash IS NULL AND expira_en>now() AND intentos<5
		RETURNING permiso_hash IS NOT NULL
	`, domain.Owner(ctx), sesion, h.hashCodigo(sesion, req.Codigo), hashRecuperacion(permiso)).Scan(&aceptado)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !aceptado) {
		return c.Status(400).JSON(fiber.Map{"error": "El código es incorrecto, caducó o agotó sus intentos."})
	}
	if err != nil {
		return internalError(c, "verificar código de eliminación", err)
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(fiber.Map{"permiso": permiso})
}

func (h *EliminacionCuentaHandler) Eliminar(c fiber.Ctx) error {
	var req struct {
		Permiso           string `json:"permiso"`
		Actual            string `json:"actual"`
		AceptaCondiciones bool   `json:"aceptaCondiciones"`
	}
	if err := c.Bind().Body(&req); err != nil || !permisoValido(req.Permiso) ||
		!contrasenaActualAdmitida(req.Actual) || !req.AceptaCondiciones {
		return c.Status(400).JSON(fiber.Map{"error": "Verifica el código, ingresa tu contraseña y acepta las condiciones."})
	}

	ctx := c.Context()
	id := domain.Owner(ctx)
	sesion := middleware.TokenHash(c.Cookies("cotizaciones_session"))
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return internalError(c, "iniciar eliminación de cuenta", err)
	}
	defer tx.Rollback(ctx)
	var claveHash, email string
	err = tx.QueryRow(ctx, `
		SELECT u.password_hash, u.email
		FROM users u JOIN eliminacion_cuenta e ON e.user_id=u.id
		WHERE u.id=$1 AND e.sesion_hash=$2 AND e.permiso_hash=$3 AND e.expira_en>now()
		FOR UPDATE OF u,e
	`, id, sesion, hashRecuperacion(req.Permiso)).Scan(&claveHash, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		return c.Status(410).JSON(fiber.Map{"error": "La verificación caducó o ya se usó. Solicita otro código."})
	}
	if err != nil {
		return internalError(c, "validar permiso de eliminación", err)
	}
	if h.hasher.Compare(claveHash, req.Actual) != nil {
		return c.Status(400).JSON(fiber.Map{"error": "La contraseña actual es incorrecta."})
	}

	for _, consulta := range []string{
		`DELETE FROM historial_nombre_usuario WHERE user_id=$1`,
		`DELETE FROM historial_perfil_comercial WHERE user_id=$1`,
		`DELETE FROM cotizaciones WHERE owner_id=$1`,
		`DELETE FROM clientes WHERE owner_id=$1`,
		`DELETE FROM users WHERE id=$1`,
	} {
		if _, err := tx.Exec(ctx, consulta, id); err != nil {
			return internalError(c, "eliminar datos de la cuenta", err)
		}
	}
	correoHash := hashSecreto(h.secret, "recuperacion-envio:"+strings.ToLower(strings.TrimSpace(email)))
	if _, err := tx.Exec(ctx, `DELETE FROM recuperacion_envios WHERE correo_hash=$1`, correoHash); err != nil {
		return internalError(c, "eliminar datos de recuperación", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return internalError(c, "confirmar eliminación de cuenta", err)
	}
	c.Cookie(sessionCookie("", time.Now().Add(-time.Hour)))
	c.Set("Cache-Control", "no-store")
	return c.SendStatus(204)
}

func (h *EliminacionCuentaHandler) hashCodigo(sesion, codigo string) []byte {
	return hashSecreto(h.secret, "eliminacion-cuenta:"+sesion+":"+codigo)
}

package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cotizaciones-api/internal/domain"
	"cotizaciones-api/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var codigoRecuperacion = regexp.MustCompile(`^[0-9]{6}$`)

type solicitudRecuperacion struct{ email, id string }

type RecuperacionHandler struct {
	db     *pgxpool.Pool
	hasher usecase.PasswordHasher
	secret []byte
	cola   chan solicitudRecuperacion
	cupos  chan struct{}
}

func NewRecuperacionHandler(db *pgxpool.Pool, hasher usecase.PasswordHasher, secret string, enviar func(string, string) error) *RecuperacionHandler {
	h := &RecuperacionHandler{db: db, hasher: hasher, secret: []byte(secret)}
	if enviar != nil {
		h.cola = make(chan solicitudRecuperacion, 24)
		h.cupos = make(chan struct{}, cap(h.cola))
		go func() {
			limpieza := time.NewTicker(time.Hour)
			defer limpieza.Stop()
			for {
				select {
				case solicitud := <-h.cola:
					h.procesarSolicitud(solicitud, enviar)
					<-h.cupos
				case <-limpieza.C:
					ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
					_, _ = h.db.Exec(ctx, `DELETE FROM recuperacion_envios WHERE ultimo_envio <= now()-interval '24 hours'`)
					cancelar()
				}
			}
		}()
	}
	return h
}

func (h *RecuperacionHandler) Solicitar(c fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Ingresa un correo válido."})
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if len(email) > 100 || !emailRegex.MatchString(email) {
		return c.Status(400).JSON(fiber.Map{"error": "Ingresa un correo válido."})
	}
	c.Set("Cache-Control", "no-store")
	if h.cola == nil {
		return c.Status(503).JSON(fiber.Map{"error": "El envío de correos de recuperación no está configurado."})
	}
	select {
	case h.cupos <- struct{}{}:
	default:
		return c.Status(503).JSON(fiber.Map{"error": "No se pudo iniciar el envío. Intenta de nuevo en un momento."})
	}
	enCola := false
	defer func() {
		if !enCola {
			<-h.cupos
		}
	}()
	id, err := generarPermiso()
	if err != nil {
		return internalError(c, "preparar recuperación", err)
	}
	espera, aceptado, err := h.reservarEnvio(c.Context(), email)
	if err != nil {
		return internalError(c, "reservar envío de recuperación", err)
	}
	if !aceptado {
		c.Set("Retry-After", strconv.Itoa(espera))
		return c.Status(429).JSON(fiber.Map{"esperaSegundos": espera})
	}
	h.cola <- solicitudRecuperacion{email, id}
	enCola = true
	return c.Status(202).JSON(fiber.Map{"solicitud": id, "esperaSegundos": espera})
}

func (h *RecuperacionHandler) reservarEnvio(ctx context.Context, email string) (int, bool, error) {
	hash := h.hashSecreto("recuperacion-envio:" + email)
	var espera int
	err := h.db.QueryRow(ctx, `
		INSERT INTO recuperacion_envios AS e (correo_hash, envios, ultimo_envio, reenviar_desde)
		VALUES ($1, 1, now(), now()+interval '1 minute')
		ON CONFLICT (correo_hash) DO UPDATE SET
		  envios = CASE WHEN e.ultimo_envio <= now()-interval '24 hours' THEN 1 ELSE LEAST(e.envios+1, 4) END,
		  ultimo_envio = now(),
		  reenviar_desde = now() + CASE
		    WHEN e.ultimo_envio <= now()-interval '24 hours' THEN interval '1 minute'
		    WHEN e.envios = 1 THEN interval '5 minutes'
		    WHEN e.envios = 2 THEN interval '10 minutes'
		    ELSE interval '24 hours' END
		WHERE e.reenviar_desde <= now()
		RETURNING ceil(extract(epoch FROM reenviar_desde-now()))::int
	`, hash).Scan(&espera)
	if errors.Is(err, pgx.ErrNoRows) {
		err = h.db.QueryRow(ctx, `SELECT GREATEST(0, ceil(extract(epoch FROM reenviar_desde-now())))::int
			FROM recuperacion_envios WHERE correo_hash=$1`, hash).Scan(&espera)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return espera, false, err
	}
	return espera, err == nil, err
}

func (h *RecuperacionHandler) procesarSolicitud(solicitud solicitudRecuperacion, enviar func(string, string) error) {
	codigo, err := generarCodigo()
	if err != nil {
		log.Print("no se pudo generar el código de recuperación")
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	var destino string
	err = h.db.QueryRow(ctx, `
		WITH creada AS (
			INSERT INTO recuperacion_contrasena (user_id, solicitud_hash, codigo_hash, clave_base, expira_en)
			SELECT id, $2, $3, sha256(convert_to(password_hash,'UTF8')), now()+interval '10 minutes'
			FROM users WHERE lower(btrim(email))=$1 ORDER BY id LIMIT 1
			ON CONFLICT (user_id) DO UPDATE SET solicitud_hash=excluded.solicitud_hash, codigo_hash=excluded.codigo_hash,
			  permiso_hash=CASE WHEN recuperacion_contrasena.clave_base=excluded.clave_base THEN recuperacion_contrasena.permiso_hash END,
			  permiso_expira_en=CASE WHEN recuperacion_contrasena.clave_base=excluded.clave_base THEN recuperacion_contrasena.permiso_expira_en END,
			  clave_base=excluded.clave_base, intentos=0, expira_en=excluded.expira_en
			RETURNING user_id
		)
		SELECT u.email FROM creada JOIN users u ON u.id=creada.user_id
	`, solicitud.email, hashRecuperacion(solicitud.id), h.hashCodigo(solicitud.id, codigo)).Scan(&destino)
	cancelar()
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		log.Print("no se pudo registrar la solicitud de recuperación")
		return
	}
	if err := enviar(destino, codigo); err != nil {
		log.Printf("el servidor SMTP no pudo enviar un código de recuperación (%v)", err)
		ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_, _ = h.db.Exec(ctx, `UPDATE recuperacion_contrasena SET expira_en=now() WHERE solicitud_hash=$1`, hashRecuperacion(solicitud.id))
	}
}

func generarCodigo() (string, error) {
	numero, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", numero.Int64()), nil
}

func (h *RecuperacionHandler) Verificar(c fiber.Ctx) error {
	var req struct {
		Solicitud string `json:"solicitud"`
		Codigo    string `json:"codigo"`
	}
	if err := c.Bind().Body(&req); err != nil || !permisoValido(req.Solicitud) || !codigoRecuperacion.MatchString(req.Codigo) {
		return c.Status(400).JSON(fiber.Map{"error": "Ingresa un código de seis dígitos."})
	}
	permiso, err := generarPermiso()
	if err != nil {
		return internalError(c, "generar permiso de recuperación", err)
	}
	var aceptado bool
	err = h.db.QueryRow(c.Context(), `
		UPDATE recuperacion_contrasena
		SET intentos=intentos+1,
		    permiso_hash=CASE WHEN codigo_hash=$2 THEN $3::bytea ELSE permiso_hash END,
		    permiso_expira_en=CASE WHEN codigo_hash=$2 THEN now()+interval '10 minutes' ELSE permiso_expira_en END,
		    codigo_hash=CASE WHEN codigo_hash=$2 THEN NULL ELSE codigo_hash END
		WHERE solicitud_hash=$1 AND codigo_hash IS NOT NULL AND expira_en>now() AND intentos<5
		RETURNING codigo_hash IS NULL
	`, hashRecuperacion(req.Solicitud), h.hashCodigo(req.Solicitud, req.Codigo), hashRecuperacion(permiso)).Scan(&aceptado)
	c.Set("Cache-Control", "no-store")
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !aceptado) {
		return c.Status(400).JSON(fiber.Map{"error": "El código es incorrecto, caducó o agotó sus intentos. Puedes solicitar otro."})
	}
	if err != nil {
		return internalError(c, "verificar código de recuperación", err)
	}
	return c.JSON(fiber.Map{"permiso": permiso})
}

func (h *RecuperacionHandler) Restablecer(c fiber.Ctx) error {
	var req struct {
		Permiso   string `json:"permiso"`
		Nueva     string `json:"nueva"`
		Confirmar string `json:"confirmar"`
	}
	if err := c.Bind().Body(&req); err != nil || !permisoValido(req.Permiso) || req.Nueva != req.Confirmar {
		return c.Status(400).JSON(fiber.Map{"error": "Revisa que ambas contraseñas coincidan."})
	}
	if err := domain.ValidarContrasena(req.Nueva); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	ctx := c.Context()
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return internalError(c, "iniciar restablecimiento", err)
	}
	defer tx.Rollback(ctx)

	var id int64
	var base []byte
	err = tx.QueryRow(ctx, `
		DELETE FROM recuperacion_contrasena WHERE permiso_hash=$1 AND permiso_expira_en>now()
		RETURNING user_id, clave_base
	`, hashRecuperacion(req.Permiso)).Scan(&id, &base)
	expirado := func() error {
		return c.Status(410).JSON(fiber.Map{"error": "El permiso caducó o ya se usó. Solicita otro código."})
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return expirado()
	}
	if err != nil {
		return internalError(c, "validar permiso de recuperación", err)
	}
	var claveActual string
	err = tx.QueryRow(ctx, `SELECT password_hash FROM users WHERE id=$1 AND sha256(convert_to(password_hash,'UTF8'))=$2 FOR UPDATE`, id, base).Scan(&claveActual)
	if errors.Is(err, pgx.ErrNoRows) {
		return expirado()
	}
	if err != nil {
		return internalError(c, "validar cuenta para restablecimiento", err)
	}
	if h.hasher.Compare(claveActual, req.Nueva) == nil {
		return c.Status(400).JSON(fiber.Map{"error": domain.ErrContrasenaSinCambios.Error()})
	}
	hash, err := h.hasher.Hash(req.Nueva)
	if err != nil {
		return internalError(c, "proteger nueva contraseña", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash=$1 WHERE id=$2`, hash, id); err != nil {
		return internalError(c, "actualizar contraseña recuperada", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM auth_sessions WHERE user_id=$1`, id); err != nil {
		return internalError(c, "invalidar sesiones anteriores", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return internalError(c, "confirmar restablecimiento", err)
	}
	c.Set("Cache-Control", "no-store")
	return c.SendStatus(204)
}

func generarPermiso() (string, error) {
	var bytes [32]byte
	_, err := rand.Read(bytes[:])
	return hex.EncodeToString(bytes[:]), err
}

func permisoValido(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	_, err := hex.DecodeString(valor)
	return err == nil
}

func hashRecuperacion(valor string) []byte {
	hash := sha256.Sum256([]byte(valor))
	return hash[:]
}

func (h *RecuperacionHandler) hashCodigo(solicitud, codigo string) []byte {
	return h.hashSecreto("recuperacion:" + solicitud + ":" + codigo)
}

func (h *RecuperacionHandler) hashSecreto(valor string) []byte {
	return hashSecreto(h.secret, valor)
}

func hashSecreto(secret []byte, valor string) []byte {
	hash := hmac.New(sha256.New, secret)
	hash.Write([]byte(valor))
	return hash.Sum(nil)
}

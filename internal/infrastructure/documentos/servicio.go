package documentos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"cotizaciones-api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Servicio struct {
	db       *pgxpool.Pool
	chromium string
	cupos    chan struct{}
}

func NuevoServicio(db *pgxpool.Pool, chromium string) *Servicio {
	if chromium == "" {
		chromium = buscarChromium()
	}
	return &Servicio{db: db, chromium: chromium, cupos: make(chan struct{}, 2)}
}

func (s *Servicio) Obtener(ctx context.Context, id int64) ([]byte, error) {
	owner := domain.Owner(ctx)
	for intento := 0; intento < 2; intento++ {
		var version int
		var pdf, instantanea []byte
		err := s.db.QueryRow(ctx, `
			SELECT d.version, d.pdf, d.instantanea
			FROM documentos_cotizacion d JOIN cotizaciones c ON c.id=d.cotizacion_id
			WHERE c.id=$1 AND c.owner_id=$2 AND c.estado<>'Eliminada'
		`, id, owner).Scan(&version, &pdf, &instantanea)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCotizacionNotFound
		}
		if err != nil || len(pdf) > 0 {
			return pdf, err
		}
		var documento domain.DocumentoCotizacion
		if err := json.Unmarshal(instantanea, &documento); err != nil {
			return nil, err
		}
		pdf, err = s.generar(ctx, documento)
		if err != nil {
			return nil, err
		}
		tag, err := s.db.Exec(ctx, `
			UPDATE documentos_cotizacion d SET pdf=$1, instantanea=NULL, generado_en=now()
			FROM cotizaciones c
			WHERE d.cotizacion_id=$2 AND d.version=$3 AND d.pdf IS NULL AND c.id=d.cotizacion_id
			  AND c.owner_id=$4 AND c.estado<>'Eliminada'
		`, pdf, id, version, owner)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 1 {
			return pdf, nil
		}
	}
	return nil, errors.New("el documento cambió durante la generación")
}

func (s *Servicio) generar(ctx context.Context, documento domain.DocumentoCotizacion) ([]byte, error) {
	if s.chromium == "" {
		return nil, errors.New("no se encontró Chrome, Edge o Chromium para generar el PDF")
	}
	ctx, cancelar := context.WithTimeout(ctx, 20*time.Second)
	defer cancelar()
	select {
	case s.cupos <- struct{}{}:
		defer func() { <-s.cupos }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	html, err := renderHTML(documento)
	if err != nil {
		return nil, err
	}
	carpeta, err := os.MkdirTemp("", "cotizacion-pdf-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(carpeta)
	entrada := filepath.Join(carpeta, "cotizacion.html")
	salida := filepath.Join(carpeta, "cotizacion.pdf")
	if err := os.WriteFile(entrada, html, 0o600); err != nil {
		return nil, err
	}
	ruta := filepath.ToSlash(entrada)
	if !strings.HasPrefix(ruta, "/") {
		ruta = "/" + ruta
	}
	origen := (&url.URL{Scheme: "file", Path: ruta}).String()
	comando := exec.CommandContext(ctx, s.chromium,
		"--headless", "--disable-gpu", "--disable-background-networking", "--disable-extensions",
		"--no-first-run", "--no-default-browser-check", "--no-pdf-header-footer",
		"--user-data-dir="+filepath.Join(carpeta, "perfil"), "--print-to-pdf="+salida, origen)
	if err := comando.Run(); err != nil {
		return nil, fmt.Errorf("generar PDF de cotización %d: %w", documento.Cotizacion.ID, err)
	}
	pdf, err := os.ReadFile(salida)
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return nil, errors.New("Chromium no generó un archivo PDF válido")
	}
	return pdf, nil
}

func buscarChromium() string {
	for _, nombre := range []string{"chromium", "chromium-browser", "google-chrome", "msedge"} {
		if ruta, err := exec.LookPath(nombre); err == nil {
			return ruta
		}
	}
	for _, variable := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
		base := os.Getenv(variable)
		if base == "" {
			continue
		}
		for _, ruta := range []string{
			filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"),
		} {
			if info, err := os.Stat(ruta); err == nil && !info.IsDir() {
				return ruta
			}
		}
	}
	return ""
}

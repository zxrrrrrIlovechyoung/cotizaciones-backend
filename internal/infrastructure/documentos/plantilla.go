package documentos

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cotizaciones-api/internal/domain"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//go:embed cotizacion.html
var plantillaHTML string

//go:embed logo.png
var logoAplicacion []byte

var logoPredeterminado = imagen(logoAplicacion)

var colorValido = regexp.MustCompile(`^#[0-9a-f]{6}$`)
var montos = message.NewPrinter(language.MustParse("es-MX"))
var meses = [...]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"}
var fuentes = map[string]template.CSS{
	"arial": "Arial, sans-serif", "verdana": "Verdana, Arial, sans-serif",
	"georgia": "Georgia, serif", "times": `"Times New Roman", Times, serif`,
}

func abreviar(valor string, limite int) string {
	runas := []rune(strings.TrimSpace(valor))
	if len(runas) <= limite {
		return string(runas)
	}
	return strings.TrimSpace(string(runas[:limite-1])) + "…"
}

func fecha(valor *time.Time) string {
	if valor == nil || valor.IsZero() {
		return "Sin definir"
	}
	return fmt.Sprintf("%02d %s %d", valor.Day(), meses[valor.Month()-1], valor.Year())
}

func imagen(datos []byte) template.URL {
	mime := http.DetectContentType(datos)
	if mime != "image/png" && mime != "image/jpeg" {
		return ""
	}
	return template.URL("data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(datos))
}

func luminanciaCanal(canal float64) float64 {
	if canal <= .04045 {
		return canal / 12.92
	}
	return math.Pow((canal+.055)/1.055, 2.4)
}

var plantilla = template.Must(template.New("cotizacion").Funcs(template.FuncMap{
	"abreviar": abreviar,
	"fecha":    fecha,
	"monto": func(valor float64, moneda string) string {
		prefijo := "$"
		if moneda != "" && moneda != "MXN" {
			prefijo = moneda + " "
		}
		if valor < 0 {
			return "-" + prefijo + montos.Sprintf("%.2f", -valor)
		}
		return prefijo + montos.Sprintf("%.2f", valor)
	},
	"imagen": imagen,
	"logo": func(datos []byte) template.URL {
		if personalizado := imagen(datos); personalizado != "" {
			return personalizado
		}
		return logoPredeterminado
	},
}).Parse(plantillaHTML))

func renderHTML(documento domain.DocumentoCotizacion) ([]byte, error) {
	color := documento.Emisor.Color
	if !colorValido.MatchString(color) {
		color = "#1b4ed8"
	}
	familia, existe := fuentes[documento.Emisor.Fuente]
	if !existe {
		familia = fuentes["arial"]
	}
	luminancia := 0.0
	for i, peso := range [...]float64{.2126, .7152, .0722} {
		canal, _ := strconv.ParseUint(color[1+i*2:3+i*2], 16, 8)
		luminancia += peso * luminanciaCanal(float64(canal)/255)
	}
	textoFranja, textoAcento := "#fff", color
	if luminancia > .179 {
		textoFranja = "#000"
	}
	if luminancia > .15 {
		textoAcento = "#192d48"
	}
	datos := struct {
		domain.DocumentoCotizacion
		Acento, TextoFranja, TextoAcento string
		Fuente                           template.CSS
		FechaCreacion                    *time.Time
	}{documento, color, textoFranja, textoAcento, familia, &documento.Cotizacion.FechaCreacion}

	var resultado bytes.Buffer
	if err := plantilla.Execute(&resultado, datos); err != nil {
		return nil, err
	}
	return resultado.Bytes(), nil
}

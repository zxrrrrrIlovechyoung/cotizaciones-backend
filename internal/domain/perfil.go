package domain

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	maxNombreComercial = 53
	maxRazonSocial     = 38
	maxDireccion       = 40
)

type PerfilCuenta struct {
	Tipo        string `json:"tipo"`
	Comercial   string `json:"comercial"`
	RazonSocial string `json:"razonSocial"`
	RFC         string `json:"rfc"`
	Telefono    string `json:"telefono"`
	Direccion   string `json:"direccion"`
}

func (p *PerfilCuenta) Validate() error {
	p.Comercial = strings.TrimSpace(p.Comercial)
	p.RazonSocial = strings.TrimSpace(p.RazonSocial)
	p.RFC = strings.ToUpper(strings.TrimSpace(p.RFC))
	p.Telefono = strings.TrimSpace(p.Telefono)
	p.Direccion = strings.TrimSpace(p.Direccion)
	if p.Tipo != "Empresa" && p.Tipo != "Persona" {
		return fmt.Errorf("tipo de cuenta inválido")
	}
	for _, field := range []struct {
		name, value string
		min, max    int
	}{
		{"nombre comercial", p.Comercial, 2, maxNombreComercial},
		{"razón social", p.RazonSocial, 0, maxRazonSocial},
		{"dirección", p.Direccion, 0, maxDireccion},
	} {
		n := utf8.RuneCountInString(field.value)
		if n < field.min || n > field.max {
			return fmt.Errorf("%s: entre %d y %d caracteres", field.name, field.min, field.max)
		}
	}
	if p.Tipo == "Empresa" && utf8.RuneCountInString(p.RazonSocial) < 2 {
		return fmt.Errorf("razón social obligatoria")
	}
	if p.Tipo == "Persona" && p.RazonSocial != "" {
		return fmt.Errorf("una cuenta personal no lleva razón social")
	}
	if utf8.RuneCountInString(p.RFC) > 13 {
		return fmt.Errorf("RFC: máximo 13 caracteres")
	}
	if p.Telefono != "" && !regexp.MustCompile(`^[0-9]{10}$`).MatchString(p.Telefono) {
		return fmt.Errorf("el teléfono debe tener 10 dígitos")
	}
	return nil
}

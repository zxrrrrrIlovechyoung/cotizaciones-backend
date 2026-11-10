package domain

import (
	"unicode"
	"unicode/utf8"
)

func ValidarContrasena(contrasena string) error {
	longitud := utf8.RuneCountInString(contrasena)
	if longitud < 8 || longitud > 20 || len(contrasena) > 72 {
		return ErrContrasenaInvalida
	}

	var mayuscula, minuscula, digito, especial bool
	for _, r := range contrasena {
		mayuscula = mayuscula || unicode.IsUpper(r)
		minuscula = minuscula || unicode.IsLower(r)
		digito = digito || unicode.IsDigit(r)
		especial = especial || (!unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r))
	}
	if !mayuscula || !minuscula || !digito || !especial {
		return ErrContrasenaInvalida
	}
	return nil
}

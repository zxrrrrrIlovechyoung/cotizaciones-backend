package handlers

import (
	"fmt"
	"math"
	"unicode/utf8"
)

func decimalEnRango(valor, minimo, maximo float64) bool {
	return valor >= minimo && valor <= maximo && math.Round(valor*100)/100 == valor
}

func validarLimitesCotizacion(req cotizacionRequest) error {
	if !decimalEnRango(req.PorcentajeIva, 0, 100) {
		return fmt.Errorf("IVA: ingresa de 0 a 100%% con hasta dos decimales")
	}
	if req.PorcentajeAnticipo != nil && !decimalEnRango(*req.PorcentajeAnticipo, 0, 100) {
		return fmt.Errorf("anticipo: ingresa de 0 a 100%% con hasta dos decimales")
	}
	for _, campo := range []struct {
		label, value string
		max          int
	}{
		{"plazo de entrega", req.TiempoEntrega, 100},
		{"condiciones de pago", req.CondicionesPago, 100},
		{"notas adicionales", req.NotasComerciales, 100},
		{"métodos de pago", req.MetodosPago, 100},
	} {
		if utf8.RuneCountInString(campo.value) > campo.max {
			return fmt.Errorf("%s: máximo %d caracteres", campo.label, campo.max)
		}
	}
	for i, p := range req.Partidas {
		if !decimalEnRango(p.PorcentajeDescuento, 0, 100) {
			return fmt.Errorf("partida %d: descuento entre 0 y 100%%", i+1)
		}
		if utf8.RuneCountInString(p.EspecificacionesTecnicas) > 500 {
			return fmt.Errorf("partida %d: especificaciones máximo 500 caracteres", i+1)
		}
		if !(p.Cantidad >= 1 && p.Cantidad <= 99999 && p.Cantidad == math.Trunc(p.Cantidad)) {
			return fmt.Errorf("partida %d: la cantidad debe ser un entero de 1 a 99,999", i+1)
		}
		if !decimalEnRango(p.PrecioUnitarioCompra, .01, 9999999.99) || !decimalEnRango(p.PrecioUnitarioVenta, .01, 9999999.99) {
			return fmt.Errorf("partida %d: costo y precio de venta deben estar entre 0.01 y 9,999,999.99, con hasta dos decimales", i+1)
		}
		if p.PrecioUnitarioVenta < p.PrecioUnitarioCompra {
			return fmt.Errorf("partida %d: el precio de venta no puede ser menor al costo", i+1)
		}
	}
	return nil
}

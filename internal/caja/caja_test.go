// Archivo: caja_test.go
// Qué hace: pruebas automáticas del corte de caja. No necesitan base de datos.

package caja

import (
	"testing"

	"github.com/GustavoQuinteroC/pv/internal/venta"
)

// Prueba el cálculo del efectivo esperado en el cajón.
func TestEsperado(t *testing.T) {
	ventas := []venta.Venta{
		// Cobró $50 en efectivo por una venta de $45: el cajón se quedó con $45.
		{Estado: venta.Activa, Total: 4500, Cambio: 500, Pagos: []venta.Pago{{Forma: venta.Efectivo, Monto: 5000}}},
		// La tarjeta no entra al cajón.
		{Estado: venta.Activa, Total: 1000, Pagos: []venta.Pago{{Forma: venta.Tarjeta, Monto: 1000}}},
		// Una venta cancelada tampoco.
		{Estado: venta.Cancelada, Total: 2000, Pagos: []venta.Pago{{Forma: venta.Efectivo, Monto: 2000}}},
	}
	movimientos := []Movimiento{
		// Entraron $10 al cajón.
		{Tipo: Entrada, Monto: 1000},
		// Salieron $30.
		{Tipo: Retiro, Monto: 3000},
	}

	// Fondo $200 + efectivo $45 + entrada $10 - retiro $30 = $225
	esperado := Esperado(20000, ventas, movimientos)
	if esperado != 22500 {
		t.Errorf("Esperado = %d, se esperaba 22500", esperado)
	}
}

// Prueba la diferencia entre lo contado y lo esperado: cuadra, sobra o falta.
func TestDiferencia(t *testing.T) {
	casos := []struct {
		nombre   string
		turno    Turno
		esperado int64
	}{
		{"cuadra", Turno{Contado: 22500, Esperado: 22500}, 0},
		{"sobra", Turno{Contado: 23000, Esperado: 22500}, 500},
		{"falta", Turno{Contado: 22000, Esperado: 22500}, -500},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if obtenida := caso.turno.Diferencia(); obtenida != caso.esperado {
				t.Errorf("Diferencia = %d, se esperaba %d", obtenida, caso.esperado)
			}
		})
	}
}

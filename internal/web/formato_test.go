// Archivo: formato_test.go
// Qué hace: pruebas automáticas de las funciones de formato (pesos, cantidades
// y lectura de montos escritos por una persona).

package web

import (
	"errors"
	"testing"
)

// Prueba que los centavos se muestren como pesos con comas y signo.
func TestPesos(t *testing.T) {
	// Mapa: entrada (centavos) → texto esperado.
	casos := map[int64]string{
		0:       "$0.00",
		5:       "$0.05",
		1800:    "$18.00",
		123450:  "$1,234.50",
		9999999: "$99,999.99",
		-2500:   "-$25.00",
	}

	for centavos, esperado := range casos {
		if obtenido := pesos(centavos); obtenido != esperado {
			t.Errorf("pesos(%d) = %q, se esperaba %q", centavos, obtenido, esperado)
		}
	}
}

// Prueba que las milésimas se muestren sin ceros sobrantes (y las piezas sin decimales).
func TestCantidad(t *testing.T) {
	casos := []struct {
		milesimas int64
		unidad    string
		esperado  string
	}{
		{3000, "pieza", "3"},
		{1000, "kg", "1"},
		{1250, "kg", "1.25"},
		{833, "kg", "0.833"},
		{500, "litro", "0.5"},
	}

	for _, caso := range casos {
		if obtenida := cantidad(caso.milesimas, caso.unidad); obtenida != caso.esperado {
			t.Errorf("cantidad(%d, %q) = %q, se esperaba %q", caso.milesimas, caso.unidad, obtenida, caso.esperado)
		}
	}
}

// Prueba la lectura de montos: los válidos dan centavos y los inválidos dan ErrMonto.
func TestACentavos(t *testing.T) {
	// Textos válidos y los centavos que deben dar.
	validos := map[string]int64{
		"":          0,
		"12":        1200,
		"12.5":      1250,
		"12.50":     1250,
		"0.05":      5,
		" 1,200 ":   120000,
		"999999999": 99999999900,
	}
	for texto, esperado := range validos {
		obtenido, err := aCentavos(texto)
		if err != nil || obtenido != esperado {
			t.Errorf("aCentavos(%q) = %d, %v; se esperaba %d", texto, obtenido, err, esperado)
		}
	}

	// Textos que deben rechazarse: negativos, letras, demasiados decimales, notación científica, muy grandes.
	for _, texto := range []string{"-5", "abc", "1.234", "NaN", "Inf", "1e5", "10000000000"} {
		if _, err := aCentavos(texto); !errors.Is(err, ErrMonto) {
			t.Errorf("aCentavos(%q) debía rechazarse con ErrMonto, dio %v", texto, err)
		}
	}
}

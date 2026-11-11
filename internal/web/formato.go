package web

import (
	"fmt"
	"html/template"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/producto"
)

var ErrMonto = aviso.Nuevo("Escribe un monto válido, por ejemplo 12.50.")

var patronMonto = regexp.MustCompile(`^(\d{1,9})(?:\.(\d{1,2}))?$`)

// funciones son los ayudantes que las plantillas pueden usar.
var funciones = template.FuncMap{
	"pesos":    pesos,
	"decimal":  decimal,
	"cantidad": cantidad,
	"fecha":    fecha,
}

// decimal convierte centavos a un número editable: 1250 → "12.50".
func decimal(centavos int64) string {
	return fmt.Sprintf("%d.%02d", centavos/100, centavos%100)
}

// pesos convierte centavos a texto: 123450 → "$1,234.50".
func pesos(centavos int64) string {
	signo := ""
	if centavos < 0 {
		signo = "-"
		centavos = -centavos
	}
	return fmt.Sprintf("%s$%s.%02d", signo, conComas(strconv.FormatInt(centavos/100, 10)), centavos%100)
}

func conComas(enteros string) string {
	for i := len(enteros) - 3; i > 0; i -= 3 {
		enteros = enteros[:i] + "," + enteros[i:]
	}
	return enteros
}

// cantidad convierte milésimas a texto: 1250 → "1.25", y las piezas sin decimales.
func cantidad(milesimas int64, unidad string) string {
	if unidad == producto.Pieza {
		return strconv.FormatInt(milesimas/1000, 10)
	}

	texto := fmt.Sprintf("%d.%03d", milesimas/1000, milesimas%1000)
	return strings.TrimRight(strings.TrimRight(texto, "0"), ".")
}

func fecha(momento time.Time) string {
	return momento.Local().Format("02/01/2006 15:04")
}

// aCentavos lee un monto escrito por una persona ("12.5", "1,200") sin pasar por decimales.
// Un texto vacío cuenta como cero.
func aCentavos(texto string) (int64, error) {
	texto = strings.ReplaceAll(strings.TrimSpace(texto), ",", "")
	if texto == "" {
		return 0, nil
	}

	partes := patronMonto.FindStringSubmatch(texto)
	if partes == nil {
		return 0, ErrMonto
	}

	enteros, _ := strconv.ParseInt(partes[1], 10, 64)
	centavos, _ := strconv.ParseInt((partes[2] + "00")[:2], 10, 64) // "5" → 50, "" → 0
	return enteros*100 + centavos, nil
}

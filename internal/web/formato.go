// Archivo: formato.go
// Qué hace: convierte números a texto bonito y al revés. De centavos a
// "$1,234.50", de milésimas a "1.25 kg", de fecha a "12/11/2026 15:30", y de lo
// que escribe una persona ("12.5") a centavos. Estas funciones también las
// usan las plantillas HTML.

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

// ErrMonto es el aviso cuando el monto escrito no se entiende.
var ErrMonto = aviso.Nuevo("Escribe un monto válido, por ejemplo 12.50.")

// patronMonto describe un monto válido: hasta 9 dígitos, y opcionalmente un punto con 1 o 2 decimales.
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
	// %d = entero; %02d = entero con al menos 2 dígitos (rellena con cero).
	return fmt.Sprintf("%d.%02d", centavos/100, centavos%100)
}

// pesos convierte centavos a texto: 123450 → "$1,234.50".
func pesos(centavos int64) string {
	signo := ""
	// Si es negativo, guarda el signo aparte y trabaja con el valor positivo.
	if centavos < 0 {
		signo = "-"
		centavos = -centavos
	}
	// Pesos enteros con comas, y centavos con dos dígitos.
	return fmt.Sprintf("%s$%s.%02d", signo, conComas(strconv.FormatInt(centavos/100, 10)), centavos%100)
}

// conComas separa los miles: "1234567" → "1,234,567".
func conComas(enteros string) string {
	// Empieza 3 dígitos antes del final y va metiendo una coma cada 3 hacia la izquierda.
	for i := len(enteros) - 3; i > 0; i -= 3 {
		enteros = enteros[:i] + "," + enteros[i:]
	}
	return enteros
}

// cantidad convierte milésimas a texto: 1250 → "1.25", y las piezas sin decimales.
func cantidad(milesimas int64, unidad string) string {
	// Las piezas son números enteros: 3000 milésimas = 3 piezas.
	if unidad == producto.Pieza {
		return strconv.FormatInt(milesimas/1000, 10)
	}

	// Kilos y litros llevan hasta 3 decimales; después se quitan los ceros y el punto sobrantes.
	texto := fmt.Sprintf("%d.%03d", milesimas/1000, milesimas%1000)
	return strings.TrimRight(strings.TrimRight(texto, "0"), ".")
}

// fecha muestra un momento en la hora local, como "12/11/2026 15:30".
func fecha(momento time.Time) string {
	return momento.Local().Format("02/01/2006 15:04")
}

// aCentavos lee un monto escrito por una persona ("12.5", "1,200") sin pasar por decimales.
// Un texto vacío cuenta como cero.
func aCentavos(texto string) (int64, error) {
	// Quita espacios y comas de miles.
	texto = strings.ReplaceAll(strings.TrimSpace(texto), ",", "")
	if texto == "" {
		return 0, nil
	}

	// Comprueba que tenga la forma correcta y separa enteros y decimales.
	partes := patronMonto.FindStringSubmatch(texto)
	if partes == nil {
		return 0, ErrMonto
	}

	// El patrón ya garantiza que son dígitos, por eso se ignora el error.
	enteros, _ := strconv.ParseInt(partes[1], 10, 64)
	// Rellena los decimales a 2 dígitos con ceros y toma los primeros 2.
	centavos, _ := strconv.ParseInt((partes[2] + "00")[:2], 10, 64) // "5" → 50, "" → 0
	return enteros*100 + centavos, nil
}

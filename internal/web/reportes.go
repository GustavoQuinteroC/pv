// Archivo: reportes.go
// Qué hace: pantalla de reportes (solo administración). Resume las ventas de un
// rango de días con totales por forma de pago y productos más vendidos, y puede
// descargar esas ventas en un archivo CSV para abrir en Excel.

package web

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/GustavoQuinteroC/pv/internal/usuario"
	"github.com/GustavoQuinteroC/pv/internal/venta"
)

// formatoFecha es el formato de fecha que usa Go para "año-mes-día" (2026-11-12).
const formatoFecha = "2006-01-02"

// reportes resume las ventas de un rango de días (hoy, si no se elige otro).
// Con ?formato=csv descarga las mismas ventas para abrirlas en una hoja de cálculo.
func (a *App) reportes(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	// Lee ?desde= y ?hasta= de la URL.
	primerDia := diaDeLaConsulta(r, "desde")
	ultimoDia := diaDeLaConsulta(r, "hasta")
	hasta := ultimoDia.AddDate(0, 0, 1) // el rango excluye el límite, así que se suma un día

	ventas, err := a.Ventas.Entre(r.Context(), primerDia, hasta)
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	// Si piden CSV, se descarga el archivo y no se pinta la pantalla.
	if r.URL.Query().Get("formato") == "csv" {
		a.descargarCSV(w, r, ventas)
		return
	}

	masVendidos, err := a.Ventas.MasVendidos(r.Context(), primerDia, hasta)
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	a.mostrar(w, r, "reportes", u, vista{
		"Titulo":      "Reportes",
		"Desde":       primerDia.Format(formatoFecha),
		"Hasta":       ultimoDia.Format(formatoFecha),
		"Resumen":     venta.Resumir(ventas),
		"MasVendidos": masVendidos,
		"Ventas":      ventas,
	})
}

// diaDeLaConsulta lee una fecha de la URL; si falta o es inválida, usa hoy.
func diaDeLaConsulta(r *http.Request, nombre string) time.Time {
	dia, err := time.ParseInLocation(formatoFecha, r.URL.Query().Get(nombre), time.Local)
	if err != nil {
		// Hoy a las 00:00 en la hora local.
		hoy := time.Now()
		return time.Date(hoy.Year(), hoy.Month(), hoy.Day(), 0, 0, 0, 0, time.Local)
	}
	return dia
}

// descargarCSV envía las ventas como archivo ventas.csv.
func (a *App) descargarCSV(w http.ResponseWriter, r *http.Request, ventas []venta.Venta) {
	// Encabezados que hacen que el navegador descargue el archivo en vez de mostrarlo.
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="ventas.csv"`)
	w.Write([]byte("\xEF\xBB\xBF")) // marca para que Excel respete los acentos

	// El escritor de CSV se ocupa de comas y comillas.
	salida := csv.NewWriter(w)
	// Primera fila: títulos de las columnas.
	salida.Write([]string{"Folio", "Fecha", "Cajero", "Total", "Estado"})
	// Una fila por venta.
	for _, v := range ventas {
		salida.Write([]string{
			strconv.FormatInt(v.Folio, 10),
			v.Fecha.Local().Format("2006-01-02 15:04"),
			v.CajeroNombre,
			decimal(v.Total),
			v.Estado,
		})
	}
	// Vacía lo pendiente hacia el navegador.
	salida.Flush()
}

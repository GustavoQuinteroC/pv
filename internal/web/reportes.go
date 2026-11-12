// Archivo: reportes.go
// Qué hace: reportes de ventas (solo administración). Resume las ventas de un
// rango de días con totales por forma de pago y productos más vendidos, y
// puede descargar esas ventas en un archivo CSV para abrir en Excel.

package web

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/GustavoQuinteroC/pv/internal/usuario"
	"github.com/GustavoQuinteroC/pv/internal/venta"
)

// formatoFecha es el formato de fecha que usa Go para "año-mes-día" (2026-11-12).
const formatoFecha = "2006-01-02"

// ventasDeLaConsulta lee ?desde= y ?hasta= (hoy, si faltan) y trae las ventas de ese rango.
func (a *App) ventasDeLaConsulta(r *http.Request) ([]venta.Venta, time.Time, time.Time, error) {
	primerDia := diaDeLaConsulta(r, "desde")
	hasta := diaDeLaConsulta(r, "hasta").AddDate(0, 0, 1) // el rango excluye el límite, así que se suma un día

	ventas, err := a.Ventas.Entre(r.Context(), primerDia, hasta)
	return ventas, primerDia, hasta, err
}

// reportes devuelve el resumen, lo más vendido y las ventas del rango.
func (a *App) reportes(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	ventas, desde, hasta, err := a.ventasDeLaConsulta(r)
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	masVendidos, err := a.Ventas.MasVendidos(r.Context(), desde, hasta)
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	responder(w, http.StatusOK, map[string]any{
		"resumen":     venta.Resumir(ventas),
		"masVendidos": masVendidos,
		"ventas":      ventas,
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

// descargarCSV envía las ventas del rango como archivo ventas.csv.
func (a *App) descargarCSV(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	ventas, _, _, err := a.ventasDeLaConsulta(r)
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	// Encabezados que hacen que el navegador descargue el archivo en vez de mostrarlo.
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="ventas.csv"`)
	w.Write([]byte("\xEF\xBB\xBF")) // marca para que Excel respete los acentos

	salida := csv.NewWriter(w)
	salida.Write([]string{"Folio", "Fecha", "Cajero", "Total", "Estado"})
	for _, v := range ventas {
		salida.Write([]string{
			strconv.FormatInt(v.Folio, 10),
			v.Fecha.Local().Format("2006-01-02 15:04"),
			v.CajeroNombre,
			fmt.Sprintf("%d.%02d", v.Total/100, v.Total%100),
			v.Estado,
		})
	}
	salida.Flush()
}

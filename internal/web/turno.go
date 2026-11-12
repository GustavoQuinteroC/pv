// Archivo: turno.go
// Qué hace: endpoints del turno de caja. Abrir turno con fondo, ver el turno
// en curso, registrar entradas y retiros de efectivo, hacer el corte ciego al
// cerrar, y consultar el historial de cortes.

package web

import (
	"net/http"

	"github.com/GustavoQuinteroC/pv/internal/caja"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
	"github.com/GustavoQuinteroC/pv/internal/venta"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// caja devuelve lo que necesita la pantalla de venta: el turno abierto (si hay),
// si es de esta persona y, en ese caso, los productos activos.
func (a *App) caja(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	turno, err := a.Caja.TurnoAbierto(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	respuesta := map[string]any{"turno": turno, "propio": turno != nil && turno.Cajero == u.ID}
	if respuesta["propio"] == true {
		productos, err := a.Productos.Listar(r.Context(), true)
		if err != nil {
			a.fallo(w, r, err)
			return
		}
		respuesta["productos"] = productos
	}
	responder(w, http.StatusOK, respuesta)
}

// abrirTurno abre el turno de la persona con el fondo indicado (centavos).
func (a *App) abrirTurno(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	var datos struct {
		Fondo int64 `json:"fondo"`
	}
	if !a.leer(w, r, &datos) {
		return
	}

	if err := a.Caja.Abrir(r.Context(), u.ID, u.Nombre, datos.Fondo); err != nil {
		a.fallo(w, r, err)
		return
	}
	sinContenido(w)
}

// turnoDeLaPersona devuelve el turno abierto si esta persona puede operarlo.
// Cuando no puede, ya respondió al navegador y devuelve nil.
func (a *App) turnoDeLaPersona(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) *caja.Turno {
	turno, err := a.Caja.TurnoAbierto(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return nil
	}
	if turno == nil {
		a.fallo(w, r, caja.ErrSinTurno)
		return nil
	}
	// Un turno solo lo opera quien lo abrió (o la administración).
	if turno.Cajero != u.ID && u.Rol != usuario.Admin {
		responder(w, http.StatusForbidden, map[string]string{"error": "Este turno es de otra persona."})
		return nil
	}
	return turno
}

// verTurno devuelve el turno en curso con sus movimientos de efectivo.
func (a *App) verTurno(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	turno := a.turnoDeLaPersona(w, r, u)
	if turno == nil {
		return
	}

	movimientos, err := a.Caja.Movimientos(r.Context(), turno.ID)
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	responder(w, http.StatusOK, map[string]any{"turno": turno, "movimientos": movimientos})
}

// registrarMovimiento guarda una entrada o un retiro de efectivo.
func (a *App) registrarMovimiento(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	turno := a.turnoDeLaPersona(w, r, u)
	if turno == nil {
		return
	}

	var datos struct {
		Tipo   string `json:"tipo"`
		Monto  int64  `json:"monto"`
		Motivo string `json:"motivo"`
	}
	if !a.leer(w, r, &datos) {
		return
	}

	if err := a.Caja.Registrar(r.Context(), turno.ID, datos.Tipo, datos.Monto, datos.Motivo); err != nil {
		a.fallo(w, r, err)
		return
	}
	sinContenido(w)
}

// cerrarTurno es el corte ciego: la persona cuenta el efectivo sin ver cuánto debería haber.
func (a *App) cerrarTurno(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	turno := a.turnoDeLaPersona(w, r, u)
	if turno == nil {
		return
	}

	var datos struct {
		Contado int64 `json:"contado"`
	}
	if !a.leer(w, r, &datos) {
		return
	}

	// Junta las ventas y los movimientos del turno para calcular lo esperado.
	ventas, err := a.Ventas.DeTurno(r.Context(), turno.ID)
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	movimientos, err := a.Caja.Movimientos(r.Context(), turno.ID)
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	// Calcula cuánto debía haber y cierra el turno guardando ambos números.
	esperado := caja.Esperado(turno.Fondo, ventas, movimientos)
	if err := a.Caja.Cerrar(r.Context(), turno.ID, datos.Contado, esperado); err != nil {
		a.fallo(w, r, err)
		return
	}
	responder(w, http.StatusOK, map[string]string{"id": turno.ID.Hex()})
}

// listaTurnos devuelve todos los cortes (solo administración).
func (a *App) listaTurnos(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	turnos, err := a.Caja.Listar(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	responder(w, http.StatusOK, turnos)
}

// detalleTurno devuelve el corte: lo vendido, los movimientos y la diferencia.
func (a *App) detalleTurno(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	turno, err := a.Caja.Obtener(r.Context(), id)
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	// Un cajero solo puede ver sus propios cortes.
	if turno.Cajero != u.ID && u.Rol != usuario.Admin {
		responder(w, http.StatusForbidden, map[string]string{"error": "Este turno es de otra persona."})
		return
	}

	ventas, err := a.Ventas.DeTurno(r.Context(), id)
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	movimientos, err := a.Caja.Movimientos(r.Context(), id)
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	responder(w, http.StatusOK, map[string]any{
		"turno":       turno,
		"diferencia":  turno.Diferencia(),
		"resumen":     venta.Resumir(ventas),
		"movimientos": movimientos,
	})
}

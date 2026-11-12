// Archivo: turno.go
// Qué hace: pantallas del turno de caja. Abrir turno con fondo, ver el turno
// en curso, registrar entradas y retiros de efectivo, hacer el corte ciego al
// cerrar, y ver el historial de cortes y el detalle de cada uno.

package web

import (
	"net/http"

	"github.com/GustavoQuinteroC/pv/internal/caja"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
	"github.com/GustavoQuinteroC/pv/internal/venta"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// abrirTurno abre el turno de la persona con el fondo capturado.
func (a *App) abrirTurno(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	// Convierte el fondo escrito ("200") a centavos.
	fondo, err := aCentavos(r.FormValue("fondo"))
	// Solo intenta abrir si el fondo se entendió bien.
	if err == nil {
		err = a.Caja.Abrir(r.Context(), u.ID, u.Nombre, fondo)
	}

	if err != nil {
		a.volver(w, r, "/caja", err)
		return
	}
	http.Redirect(w, r, "/caja", http.StatusSeeOther)
}

// turnoDeLaPersona devuelve el turno abierto si esta persona puede operarlo.
// Cuando no puede, ya respondió al navegador y devuelve nil.
func (a *App) turnoDeLaPersona(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) *caja.Turno {
	turno, err := a.Caja.TurnoAbierto(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return nil
	}

	// Sin turno abierto, manda a la caja para que lo abra.
	if turno == nil {
		http.Redirect(w, r, "/caja", http.StatusSeeOther)
		return nil
	}
	// Un turno solo lo opera quien lo abrió (o la administración).
	if turno.Cajero != u.ID && u.Rol != usuario.Admin {
		http.Error(w, "Este turno es de otra persona.", http.StatusForbidden)
		return nil
	}
	return turno
}

// verTurno muestra el turno en curso con sus movimientos y el formulario de cierre.
func (a *App) verTurno(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	turno := a.turnoDeLaPersona(w, r, u)
	// nil = ya se respondió al navegador, no hay nada más que hacer.
	if turno == nil {
		return
	}

	movimientos, err := a.Caja.Movimientos(r.Context(), turno.ID)
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	a.mostrar(w, r, "turno", u, vista{"Titulo": "Turno", "Turno": turno, "Movimientos": movimientos})
}

// registrarMovimiento guarda una entrada o un retiro de efectivo.
func (a *App) registrarMovimiento(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	turno := a.turnoDeLaPersona(w, r, u)
	if turno == nil {
		return
	}

	monto, err := aCentavos(r.FormValue("monto"))
	if err == nil {
		err = a.Caja.Registrar(r.Context(), turno.ID, r.FormValue("tipo"), monto, r.FormValue("motivo"))
	}

	if err != nil {
		a.volver(w, r, "/turno", err)
		return
	}
	http.Redirect(w, r, "/turno", http.StatusSeeOther)
}

// cerrarTurno es el corte ciego: la persona cuenta el efectivo sin ver cuánto debería haber.
func (a *App) cerrarTurno(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	turno := a.turnoDeLaPersona(w, r, u)
	if turno == nil {
		return
	}

	// Lo que la persona contó en el cajón.
	contado, err := aCentavos(r.FormValue("contado"))
	if err != nil {
		a.volver(w, r, "/turno", err)
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
	if err := a.Caja.Cerrar(r.Context(), turno.ID, contado, esperado); err != nil {
		a.volver(w, r, "/turno", err)
		return
	}
	// Muestra el resultado del corte.
	http.Redirect(w, r, "/turnos/"+turno.ID.Hex(), http.StatusSeeOther)
}

// listaTurnos muestra todos los cortes (solo administración).
func (a *App) listaTurnos(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	turnos, err := a.Caja.Listar(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	a.mostrar(w, r, "turnos", u, vista{"Titulo": "Cortes", "Turnos": turnos})
}

// detalleTurno muestra el corte: lo vendido, los movimientos y la diferencia.
func (a *App) detalleTurno(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	// Lee el id de la URL.
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
		http.Error(w, "Este turno es de otra persona.", http.StatusForbidden)
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

	a.mostrar(w, r, "turno_detalle", u, vista{
		"Titulo":      "Corte",
		"Turno":       turno,
		"Resumen":     venta.Resumir(ventas),
		"Movimientos": movimientos,
		// Clase de color para la diferencia (verde, rojo…).
		"Clase": claseDeDiferencia(turno.Diferencia()),
	})
}

// claseDeDiferencia traduce la diferencia a un nombre de clase CSS para pintarla.
func claseDeDiferencia(diferencia int64) string {
	switch {
	case diferencia > 0:
		return "sobra"
	case diferencia < 0:
		return "falta"
	}
	return "cuadra"
}

// Archivo: ventas.go
// Qué hace: endpoints de la venta. Recibe la venta armada en el navegador y la
// cobra, devuelve el ticket y permite cancelar una venta (solo administración).

package web

import (
	"net/http"

	"github.com/GustavoQuinteroC/pv/internal/caja"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
	"github.com/GustavoQuinteroC/pv/internal/venta"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// cobrar recibe la venta completa que armó la pantalla de caja, la valida
// contra el catálogo y la guarda.
func (a *App) cobrar(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	// Debe haber un turno abierto y ser de esta persona.
	turno, err := a.Caja.TurnoAbierto(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	if turno == nil || turno.Cajero != u.ID {
		a.fallo(w, r, caja.ErrSinTurno)
		return
	}

	var pedido venta.Pedido
	if !a.leer(w, r, &pedido) {
		return
	}

	// Trae el catálogo: los precios SIEMPRE salen de aquí, nunca del navegador.
	catalogo, err := a.Productos.Catalogo(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	// Construye y valida la venta (renglones, totales, pagos, cambio).
	nueva, err := venta.Nueva(pedido, catalogo)
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	// Anota a qué turno y a qué cajero pertenece.
	nueva.Turno, nueva.Cajero, nueva.CajeroNombre = turno.ID, u.ID, u.Nombre

	// Guarda en MongoDB (si el token ya existía, devuelve la misma venta sin duplicar).
	if err := a.Ventas.Registrar(r.Context(), nueva); err != nil {
		a.fallo(w, r, err)
		return
	}
	responder(w, http.StatusCreated, map[string]string{"id": nueva.ID.Hex()})
}

// verVenta devuelve una venta completa, para dibujar su ticket.
func (a *App) verVenta(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	v, err := a.Ventas.Obtener(r.Context(), id)
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	responder(w, http.StatusOK, v)
}

// cancelarVenta anula una venta del turno abierto, guardando el motivo.
func (a *App) cancelarVenta(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var datos struct {
		Motivo string `json:"motivo"`
	}
	if !a.leer(w, r, &datos) {
		return
	}

	turno, err := a.Caja.TurnoAbierto(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	// Sin turno abierto no se puede cancelar.
	if turno == nil {
		a.fallo(w, r, venta.ErrNoCancelable)
		return
	}

	if err := a.Ventas.Cancelar(r.Context(), id, turno.ID, datos.Motivo); err != nil {
		a.fallo(w, r, err)
		return
	}
	sinContenido(w)
}

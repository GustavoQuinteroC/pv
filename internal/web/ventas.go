// Archivo: ventas.go
// Qué hace: pantallas y endpoints de la venta. Muestra la pantalla de caja,
// recibe la venta armada en el navegador (JSON) y la cobra, muestra el ticket
// imprimible y permite cancelar una venta (solo administración).

package web

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/caja"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
	"github.com/GustavoQuinteroC/pv/internal/venta"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ErrPedido es el aviso cuando el JSON de la venta no se pudo leer.
var ErrPedido = aviso.Nuevo("No se entendió la venta. Recarga la pantalla de caja.")

// pantallaCaja es la pantalla de venta. Si la persona no tiene turno, pide abrirlo.
func (a *App) pantallaCaja(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	turno, err := a.Caja.TurnoAbierto(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	// Sin turno, o con el turno de otra persona: muestra la pantalla para abrir turno.
	if turno == nil || turno.Cajero != u.ID {
		a.mostrar(w, r, "turno_abrir", u, vista{"Titulo": "Caja", "Turno": turno})
		return
	}

	// true = solo productos activos, para venderlos.
	productos, err := a.Productos.Listar(r.Context(), true)
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	a.mostrar(w, r, "caja", u, vista{"Titulo": "Caja", "Productos": productos})
}

// cobrar recibe la venta completa que armó la pantalla de caja, la valida
// contra el catálogo y la guarda. Responde JSON porque lo llama el JavaScript.
func (a *App) cobrar(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	// Debe haber un turno abierto y ser de esta persona.
	turno, err := a.Caja.TurnoAbierto(r.Context())
	if err != nil {
		a.errorJSON(w, r, err)
		return
	}
	if turno == nil || turno.Cajero != u.ID {
		a.errorJSON(w, r, caja.ErrSinTurno)
		return
	}

	// Lee el JSON del cuerpo de la petición (máximo 1 MB: 1<<20 bytes).
	var pedido venta.Pedido
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&pedido); err != nil {
		a.errorJSON(w, r, ErrPedido)
		return
	}

	// Trae el catálogo: los precios SIEMPRE salen de aquí, nunca del navegador.
	catalogo, err := a.Productos.Catalogo(r.Context())
	if err != nil {
		a.errorJSON(w, r, err)
		return
	}

	// Construye y valida la venta (renglones, totales, pagos, cambio).
	nueva, err := venta.Nueva(pedido, catalogo)
	if err != nil {
		a.errorJSON(w, r, err)
		return
	}
	// Anota a qué turno y a qué cajero pertenece.
	nueva.Turno, nueva.Cajero, nueva.CajeroNombre = turno.ID, u.ID, u.Nombre

	// Guarda en MongoDB (si el token ya existía, devuelve la misma venta sin duplicar).
	if err := a.Ventas.Registrar(r.Context(), nueva); err != nil {
		a.errorJSON(w, r, err)
		return
	}
	// 201 Creado, con el id para abrir el ticket.
	responderJSON(w, http.StatusCreated, map[string]string{"id": nueva.ID.Hex()})
}

// verTicket muestra el ticket de una venta.
func (a *App) verTicket(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
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

	a.mostrar(w, r, "ticket", u, vista{
		"Titulo":  "Ticket",
		"Venta":   v,
		"SinMenu": true,
		// Con ?imprimir=1 abre directo el diálogo de impresión del navegador.
		"Imprimir": r.URL.Query().Get("imprimir") == "1",
	})
}

// cancelarVenta anula una venta del turno abierto, guardando el motivo.
func (a *App) cancelarVenta(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// Después de cancelar (o fallar) vuelve al ticket.
	destino := "/ventas/" + id.Hex()

	turno, err := a.Caja.TurnoAbierto(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	// Sin turno abierto no se puede cancelar.
	if turno == nil {
		a.volver(w, r, destino, venta.ErrNoCancelable)
		return
	}

	if err := a.Ventas.Cancelar(r.Context(), id, turno.ID, r.FormValue("motivo")); err != nil {
		a.volver(w, r, destino, err)
		return
	}
	http.Redirect(w, r, destino, http.StatusSeeOther)
}

// responderJSON envía una respuesta en formato JSON con el código HTTP indicado.
func responderJSON(w http.ResponseWriter, estado int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	json.NewEncoder(w).Encode(cuerpo)
}

// errorJSON responde un error en JSON: los avisos tal cual, los errores internos con un mensaje genérico.
func (a *App) errorJSON(w http.ResponseWriter, r *http.Request, err error) {
	// 422: la persona mandó algo que no se puede procesar; el mensaje sí se le muestra.
	if mensaje, esAviso := aviso.De(err); esAviso {
		responderJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": mensaje})
		return
	}

	// 500: error interno; se registra y no se muestran detalles.
	slog.Error("error interno", "ruta", r.URL.Path, "error", err)
	responderJSON(w, http.StatusInternalServerError, map[string]string{"error": "Algo salió mal. Intenta de nuevo."})
}

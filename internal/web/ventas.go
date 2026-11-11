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

var ErrPedido = aviso.Nuevo("No se entendió la venta. Recarga la pantalla de caja.")

// pantallaCaja es la pantalla de venta. Si la persona no tiene turno, pide abrirlo.
func (a *App) pantallaCaja(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	turno, err := a.Caja.TurnoAbierto(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	if turno == nil || turno.Cajero != u.ID {
		a.mostrar(w, r, "turno_abrir", u, vista{"Titulo": "Caja", "Turno": turno})
		return
	}

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
	turno, err := a.Caja.TurnoAbierto(r.Context())
	if err != nil {
		a.errorJSON(w, r, err)
		return
	}
	if turno == nil || turno.Cajero != u.ID {
		a.errorJSON(w, r, caja.ErrSinTurno)
		return
	}

	var pedido venta.Pedido
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&pedido); err != nil {
		a.errorJSON(w, r, ErrPedido)
		return
	}

	catalogo, err := a.Productos.Catalogo(r.Context())
	if err != nil {
		a.errorJSON(w, r, err)
		return
	}

	nueva, err := venta.Nueva(pedido, catalogo)
	if err != nil {
		a.errorJSON(w, r, err)
		return
	}
	nueva.Turno, nueva.Cajero, nueva.CajeroNombre = turno.ID, u.ID, u.Nombre

	if err := a.Ventas.Registrar(r.Context(), nueva); err != nil {
		a.errorJSON(w, r, err)
		return
	}
	responderJSON(w, http.StatusCreated, map[string]string{"id": nueva.ID.Hex()})
}

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
		"Titulo":   "Ticket",
		"Venta":    v,
		"SinMenu":  true,
		"Imprimir": r.URL.Query().Get("imprimir") == "1",
	})
}

func (a *App) cancelarVenta(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	destino := "/ventas/" + id.Hex()

	turno, err := a.Caja.TurnoAbierto(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return
	}
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

func responderJSON(w http.ResponseWriter, estado int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	json.NewEncoder(w).Encode(cuerpo)
}

func (a *App) errorJSON(w http.ResponseWriter, r *http.Request, err error) {
	if mensaje, esAviso := aviso.De(err); esAviso {
		responderJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": mensaje})
		return
	}

	slog.Error("error interno", "ruta", r.URL.Path, "error", err)
	responderJSON(w, http.StatusInternalServerError, map[string]string{"error": "Algo salió mal. Intenta de nuevo."})
}

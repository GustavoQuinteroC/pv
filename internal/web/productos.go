// Archivo: productos.go
// Qué hace: endpoints del catálogo (solo administración). Lista los productos
// y crea o actualiza uno.

package web

import (
	"net/http"

	"github.com/GustavoQuinteroC/pv/internal/producto"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// listaProductos devuelve todos los productos, también los inactivos.
func (a *App) listaProductos(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	productos, err := a.Productos.Listar(r.Context(), false)
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	responder(w, http.StatusOK, productos)
}

// guardarProducto crea un producto (POST) o actualiza el de la ruta (PUT).
// La validación (nombre, precio, unidad…) la hace el paquete producto.
func (a *App) guardarProducto(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	var p producto.Producto
	if !a.leer(w, r, &p) {
		return
	}

	// El id nunca se toma del cuerpo: viene de la ruta o no existe (producto nuevo).
	p.ID = bson.ObjectID{}
	if texto := r.PathValue("id"); texto != "" {
		id, err := bson.ObjectIDFromHex(texto)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		p.ID = id
	}

	if err := a.Productos.Guardar(r.Context(), p); err != nil {
		a.fallo(w, r, err)
		return
	}
	sinContenido(w)
}

// Archivo: productos.go
// Qué hace: pantallas del catálogo (solo administración). Lista los productos,
// muestra el formulario para crear o editar uno y guarda lo que se capture.

package web

import (
	"net/http"
	"strings"

	"github.com/GustavoQuinteroC/pv/internal/producto"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// listaProductos muestra la tabla con todos los productos.
func (a *App) listaProductos(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	// false = incluir también los productos inactivos.
	productos, err := a.Productos.Listar(r.Context(), false)
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	a.mostrar(w, r, "productos", u, vista{"Titulo": "Productos", "Productos": productos})
}

// formularioProducto sirve para crear (sin id en la ruta) y para editar (con id).
func (a *App) formularioProducto(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	// Valores iniciales de un producto nuevo: por piezas y activo.
	p := producto.Producto{Unidad: producto.Pieza, Activo: true}
	// El formulario se envía a /productos (crear).
	accion := "/productos"

	// Si la ruta trae un id, se está editando un producto existente.
	if r.PathValue("id") != "" {
		// Convierte el texto del id a un ObjectID de MongoDB.
		id, err := bson.ObjectIDFromHex(r.PathValue("id"))
		if err != nil {
			// Id con formato inválido: página no encontrada.
			http.NotFound(w, r)
			return
		}

		// Carga el producto guardado.
		p, err = a.Productos.Obtener(r.Context(), id)
		if err != nil {
			a.fallo(w, r, err)
			return
		}
		// Ahora el formulario se envía a /productos/<id> (editar).
		accion = "/productos/" + id.Hex()
	}

	a.mostrar(w, r, "producto_form", u, vista{
		"Titulo":   "Producto",
		"Producto": p,
		"Accion":   accion,
		"Unidades": producto.Unidades,
	})
}

// guardarProducto recibe el formulario y crea o actualiza el producto.
func (a *App) guardarProducto(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	// Convierte los campos del formulario en un producto.
	p, err := productoDelFormulario(r)
	if err != nil {
		a.volver(w, r, "/productos", err)
		return
	}

	// Si la ruta trae id, se edita ese producto; si no, se crea uno nuevo.
	volverA := "/productos/nuevo"
	if r.PathValue("id") != "" {
		p.ID, err = bson.ObjectIDFromHex(r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		// Si hay un error de validación, regresa al formulario de este producto.
		volverA = "/productos/" + p.ID.Hex()
	}

	// Guarda; la validación (nombre, precio, unidad…) la hace el paquete producto.
	if err := a.Productos.Guardar(r.Context(), p); err != nil {
		a.volver(w, r, volverA, err)
		return
	}
	http.Redirect(w, r, "/productos", http.StatusSeeOther)
}

// productoDelFormulario lee los campos que mandó el navegador y arma un Producto.
func productoDelFormulario(r *http.Request) (producto.Producto, error) {
	// El precio llega como texto ("12.50") y se convierte a centavos.
	precio, err := aCentavos(r.FormValue("precio"))
	return producto.Producto{
		// TrimSpace quita espacios sobrantes al inicio y al final.
		Nombre: strings.TrimSpace(r.FormValue("nombre")),
		Codigo: strings.TrimSpace(r.FormValue("codigo")),
		Precio: precio,
		Unidad: r.FormValue("unidad"),
		// Una casilla marcada llega como "on"; sin marcar, no llega.
		Favorito: r.FormValue("favorito") == "on",
		Activo:   r.FormValue("activo") == "on",
	}, err
}

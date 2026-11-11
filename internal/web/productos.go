package web

import (
	"net/http"
	"strings"

	"github.com/GustavoQuinteroC/pv/internal/producto"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (a *App) listaProductos(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	productos, err := a.Productos.Listar(r.Context(), false)
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	a.mostrar(w, r, "productos", u, vista{"Titulo": "Productos", "Productos": productos})
}

// formularioProducto sirve para crear (sin id en la ruta) y para editar (con id).
func (a *App) formularioProducto(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	p := producto.Producto{Unidad: producto.Pieza, Activo: true}
	accion := "/productos"

	if r.PathValue("id") != "" {
		id, err := bson.ObjectIDFromHex(r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		p, err = a.Productos.Obtener(r.Context(), id)
		if err != nil {
			a.fallo(w, r, err)
			return
		}
		accion = "/productos/" + id.Hex()
	}

	a.mostrar(w, r, "producto_form", u, vista{
		"Titulo":   "Producto",
		"Producto": p,
		"Accion":   accion,
		"Unidades": producto.Unidades,
	})
}

func (a *App) guardarProducto(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
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
		volverA = "/productos/" + p.ID.Hex()
	}

	if err := a.Productos.Guardar(r.Context(), p); err != nil {
		a.volver(w, r, volverA, err)
		return
	}
	http.Redirect(w, r, "/productos", http.StatusSeeOther)
}

func productoDelFormulario(r *http.Request) (producto.Producto, error) {
	precio, err := aCentavos(r.FormValue("precio"))
	return producto.Producto{
		Nombre:   strings.TrimSpace(r.FormValue("nombre")),
		Codigo:   strings.TrimSpace(r.FormValue("codigo")),
		Precio:   precio,
		Unidad:   r.FormValue("unidad"),
		Favorito: r.FormValue("favorito") == "on",
		Activo:   r.FormValue("activo") == "on",
	}, err
}

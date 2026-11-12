// Archivo: ajustes.go
// Qué hace: pantalla de ajustes (solo administración). Permite cambiar el
// nombre del negocio y el mensaje que sale al final del ticket.

package web

import (
	"net/http"
	"strings"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/negocio"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
)

// ErrNombreNegocio es el aviso cuando se intenta guardar sin nombre.
var ErrNombreNegocio = aviso.Nuevo("Escribe el nombre del negocio.")

// formularioAjustes muestra el formulario (los datos del negocio los agrega mostrar).
func (a *App) formularioAjustes(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	a.mostrar(w, r, "ajustes", u, vista{"Titulo": "Ajustes"})
}

// guardarAjustes valida y guarda los datos del negocio.
func (a *App) guardarAjustes(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	ajustes := negocio.Negocio{
		Nombre:  strings.TrimSpace(r.FormValue("nombre")),
		Mensaje: strings.TrimSpace(r.FormValue("mensaje")),
	}
	// El nombre es obligatorio; el mensaje puede ir vacío.
	if ajustes.Nombre == "" {
		a.volver(w, r, "/ajustes", ErrNombreNegocio)
		return
	}

	if err := a.Negocio.Guardar(r.Context(), ajustes); err != nil {
		a.fallo(w, r, err)
		return
	}
	http.Redirect(w, r, "/ajustes", http.StatusSeeOther)
}

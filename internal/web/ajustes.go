package web

import (
	"net/http"
	"strings"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/negocio"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
)

var ErrNombreNegocio = aviso.Nuevo("Escribe el nombre del negocio.")

func (a *App) formularioAjustes(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	a.mostrar(w, r, "ajustes", u, vista{"Titulo": "Ajustes"})
}

func (a *App) guardarAjustes(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	ajustes := negocio.Negocio{
		Nombre:  strings.TrimSpace(r.FormValue("nombre")),
		Mensaje: strings.TrimSpace(r.FormValue("mensaje")),
	}
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

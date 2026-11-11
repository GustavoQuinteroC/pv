package web

import (
	"net/http"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var ErrAutoDesactivar = aviso.Nuevo("No puedes desactivar tu propia cuenta.")

func (a *App) listaUsuarios(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	usuarios, err := a.Usuarios.Listar(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	a.mostrar(w, r, "usuarios", u, vista{"Titulo": "Usuarios", "Usuarios": usuarios})
}

func (a *App) crearUsuario(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	rol := usuario.Cajero
	if r.FormValue("rol") == usuario.Admin {
		rol = usuario.Admin
	}

	err := a.Usuarios.Crear(r.Context(), r.FormValue("nombre"), r.FormValue("correo"), r.FormValue("clave"), rol)
	if err != nil {
		a.volver(w, r, "/usuarios", err)
		return
	}
	http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
}

func (a *App) cambiarActivo(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if id == u.ID {
		a.volver(w, r, "/usuarios", ErrAutoDesactivar)
		return
	}

	if err := a.Usuarios.CambiarActivo(r.Context(), id, r.FormValue("activo") == "true"); err != nil {
		a.fallo(w, r, err)
		return
	}
	http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
}

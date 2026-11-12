// Archivo: usuarios.go
// Qué hace: pantallas de administración de usuarios. Lista las cuentas, crea
// cajeros o administradores nuevos y activa o desactiva cuentas.

package web

import (
	"net/http"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ErrAutoDesactivar evita que una persona se desactive a sí misma y se quede sin acceso.
var ErrAutoDesactivar = aviso.Nuevo("No puedes desactivar tu propia cuenta.")

// listaUsuarios muestra todas las cuentas.
func (a *App) listaUsuarios(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	usuarios, err := a.Usuarios.Listar(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	a.mostrar(w, r, "usuarios", u, vista{"Titulo": "Usuarios", "Usuarios": usuarios})
}

// crearUsuario da de alta una cuenta nueva con el formulario.
func (a *App) crearUsuario(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	// Por defecto es cajero; solo es administrador si se eligió ese rol exactamente.
	rol := usuario.Cajero
	if r.FormValue("rol") == usuario.Admin {
		rol = usuario.Admin
	}

	// La validación (correo repetido, contraseña corta…) la hace el paquete usuario.
	err := a.Usuarios.Crear(r.Context(), r.FormValue("nombre"), r.FormValue("correo"), r.FormValue("clave"), rol)
	if err != nil {
		a.volver(w, r, "/usuarios", err)
		return
	}
	http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
}

// cambiarActivo activa o desactiva una cuenta.
func (a *App) cambiarActivo(w http.ResponseWriter, r *http.Request, u *usuario.Usuario) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// Nadie puede desactivar su propia cuenta.
	if id == u.ID {
		a.volver(w, r, "/usuarios", ErrAutoDesactivar)
		return
	}

	// El formulario manda "true" para activar y otro valor para desactivar.
	if err := a.Usuarios.CambiarActivo(r.Context(), id, r.FormValue("activo") == "true"); err != nil {
		a.fallo(w, r, err)
		return
	}
	http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
}

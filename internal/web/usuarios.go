// Archivo: usuarios.go
// Qué hace: administración de usuarios (solo administración). Lista las
// cuentas, crea cajeros o administradores nuevos y activa o desactiva cuentas.

package web

import (
	"net/http"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ErrAutoDesactivar evita que una persona se desactive a sí misma y se quede sin acceso.
var ErrAutoDesactivar = aviso.Nuevo("No puedes desactivar tu propia cuenta.")

// listaUsuarios devuelve todas las cuentas (sin contraseñas).
func (a *App) listaUsuarios(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	usuarios, err := a.Usuarios.Listar(r.Context())
	if err != nil {
		a.fallo(w, r, err)
		return
	}
	responder(w, http.StatusOK, usuarios)
}

// crearUsuario da de alta una cuenta nueva.
func (a *App) crearUsuario(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	var datos struct {
		Nombre string `json:"nombre"`
		Correo string `json:"correo"`
		Clave  string `json:"clave"`
		Rol    string `json:"rol"`
	}
	if !a.leer(w, r, &datos) {
		return
	}

	// Por defecto es cajero; solo es administrador si se eligió ese rol exactamente.
	rol := usuario.Cajero
	if datos.Rol == usuario.Admin {
		rol = usuario.Admin
	}

	// La validación (correo repetido, contraseña corta…) la hace el paquete usuario.
	if err := a.Usuarios.Crear(r.Context(), datos.Nombre, datos.Correo, datos.Clave, rol); err != nil {
		a.fallo(w, r, err)
		return
	}
	sinContenido(w)
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
		a.fallo(w, r, ErrAutoDesactivar)
		return
	}

	var datos struct {
		Activo bool `json:"activo"`
	}
	if !a.leer(w, r, &datos) {
		return
	}

	if err := a.Usuarios.CambiarActivo(r.Context(), id, datos.Activo); err != nil {
		a.fallo(w, r, err)
		return
	}
	sinContenido(w)
}

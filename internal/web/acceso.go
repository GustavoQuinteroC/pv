// Archivo: acceso.go
// Qué hace: sesión de la persona. Dice quién es (y los datos del negocio),
// valida correo y contraseña, crea la cookie de sesión y la borra al salir.

package web

import (
	"log/slog"
	"net/http"

	"github.com/GustavoQuinteroC/pv/internal/usuario"
)

// yo es lo primero que pide la interfaz al abrirse: los datos del negocio y la
// persona con sesión (null si no ha entrado).
func (a *App) yo(w http.ResponseWriter, r *http.Request) {
	responder(w, http.StatusOK, map[string]any{
		"usuario": a.persona(r),
		"negocio": a.Negocio.Obtener(r.Context()),
	})
}

// iniciarSesion revisa correo y contraseña y, si son correctos, abre la sesión.
func (a *App) iniciarSesion(w http.ResponseWriter, r *http.Request) {
	var datos struct {
		Correo string `json:"correo"`
		Clave  string `json:"clave"`
	}
	if !a.leer(w, r, &datos) {
		return
	}

	u, err := a.Usuarios.Autenticar(r.Context(), datos.Correo, datos.Clave)
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	// Crea una sesión en la base de datos y obtiene su token secreto.
	token, err := a.Usuarios.IniciarSesion(r.Context(), u)
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	// Guarda el token en una cookie del navegador.
	http.SetCookie(w, &http.Cookie{
		Name:  galletaSesion,
		Value: token,
		Path:  "/",
		// HttpOnly: el JavaScript de la página no puede leer la cookie.
		HttpOnly: true,
		// Secure: solo viaja por HTTPS (cuando se usa HTTPS).
		Secure: esHTTPS(r),
		// Lax: no se manda en envíos desde otros sitios.
		SameSite: http.SameSiteLaxMode,
	})
	responder(w, http.StatusOK, u)
}

// cerrarSesion borra la sesión de la base de datos y la cookie del navegador.
func (a *App) cerrarSesion(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	if galleta, err := r.Cookie(galletaSesion); err == nil {
		if err := a.Usuarios.CerrarSesion(r.Context(), galleta.Value); err != nil {
			// Si falla no se detiene: solo se anota, y la cookie se borra de todos modos.
			slog.Warn("no se pudo borrar la sesión", "error", err)
		}
	}

	// MaxAge -1 le dice al navegador que borre la cookie.
	http.SetCookie(w, &http.Cookie{Name: galletaSesion, Path: "/", MaxAge: -1})
	sinContenido(w)
}

// esHTTPS también mira el encabezado del proxy: en Coolify el TLS lo termina Traefik.
func esHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

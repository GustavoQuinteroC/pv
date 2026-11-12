// Archivo: acceso.go
// Qué hace: pantallas de entrada y salida. Muestra el formulario de login,
// valida correo y contraseña, crea la cookie de sesión y la borra al salir.

package web

import (
	"log/slog"
	"net/http"

	"github.com/GustavoQuinteroC/pv/internal/usuario"
)

// formularioLogin muestra la pantalla para escribir correo y contraseña.
func (a *App) formularioLogin(w http.ResponseWriter, r *http.Request) {
	// SinMenu oculta la barra de navegación, porque aún no hay sesión.
	a.mostrar(w, r, "login", nil, vista{"Titulo": "Entrar", "SinMenu": true})
}

// iniciarSesion recibe el formulario de login y, si es correcto, abre la sesión.
func (a *App) iniciarSesion(w http.ResponseWriter, r *http.Request) {
	// Revisa correo y contraseña.
	u, err := a.Usuarios.Autenticar(r.Context(), r.FormValue("correo"), r.FormValue("clave"))
	if err != nil {
		// Datos incorrectos: regresa al login con el aviso.
		a.volver(w, r, "/login", err)
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
	// Redirige a la caja.
	http.Redirect(w, r, "/caja", http.StatusSeeOther)
}

// cerrarSesion borra la sesión de la base de datos y la cookie del navegador.
func (a *App) cerrarSesion(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	// Si el navegador trae la cookie, borra esa sesión de la base de datos.
	if galleta, err := r.Cookie(galletaSesion); err == nil {
		if err := a.Usuarios.CerrarSesion(r.Context(), galleta.Value); err != nil {
			// Si falla no se detiene: solo se anota, y la cookie se borra de todos modos.
			slog.Warn("no se pudo borrar la sesión", "error", err)
		}
	}

	// MaxAge -1 le dice al navegador que borre la cookie.
	http.SetCookie(w, &http.Cookie{Name: galletaSesion, Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// esHTTPS también mira el encabezado del proxy: en Coolify el TLS lo termina Traefik.
func esHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

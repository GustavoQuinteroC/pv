package web

import (
	"log/slog"
	"net/http"

	"github.com/GustavoQuinteroC/pv/internal/usuario"
)

func (a *App) formularioLogin(w http.ResponseWriter, r *http.Request) {
	a.mostrar(w, r, "login", nil, vista{"Titulo": "Entrar", "SinMenu": true})
}

func (a *App) iniciarSesion(w http.ResponseWriter, r *http.Request) {
	u, err := a.Usuarios.Autenticar(r.Context(), r.FormValue("correo"), r.FormValue("clave"))
	if err != nil {
		a.volver(w, r, "/login", err)
		return
	}

	token, err := a.Usuarios.IniciarSesion(r.Context(), u)
	if err != nil {
		a.fallo(w, r, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     galletaSesion,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   esHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/caja", http.StatusSeeOther)
}

func (a *App) cerrarSesion(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	if galleta, err := r.Cookie(galletaSesion); err == nil {
		if err := a.Usuarios.CerrarSesion(r.Context(), galleta.Value); err != nil {
			slog.Warn("no se pudo borrar la sesión", "error", err)
		}
	}

	http.SetCookie(w, &http.Cookie{Name: galletaSesion, Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// esHTTPS también mira el encabezado del proxy: en Coolify el TLS lo termina Traefik.
func esHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

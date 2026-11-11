// Package web contiene las pantallas: rutas, formularios y plantillas.
package web

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/caja"
	"github.com/GustavoQuinteroC/pv/internal/negocio"
	"github.com/GustavoQuinteroC/pv/internal/producto"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
	"github.com/GustavoQuinteroC/pv/internal/venta"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

//go:embed plantillas static
var recursos embed.FS

const (
	galletaSesion = "sesion"
	galletaAviso  = "aviso"
	cualquiera    = "" // rol que pide una ruta abierta a todas las personas con sesión
)

type App struct {
	Usuarios   *usuario.Repo
	Productos  *producto.Repo
	Ventas     *venta.Repo
	Caja       *caja.Repo
	Negocio    *negocio.Repo
	plantillas map[string]*template.Template
}

func Nueva(db *mongo.Database) *App {
	return &App{
		Usuarios:   usuario.NuevoRepo(db),
		Productos:  producto.NuevoRepo(db),
		Ventas:     venta.NuevoRepo(db),
		Caja:       caja.NuevoRepo(db),
		Negocio:    negocio.NuevoRepo(db),
		plantillas: cargarPlantillas(),
	}
}

func (a *App) Rutas() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServerFS(recursos))
	mux.HandleFunc("GET /salud", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/caja", http.StatusSeeOther)
	})

	mux.HandleFunc("GET /login", a.formularioLogin)
	mux.HandleFunc("POST /login", a.iniciarSesion)
	mux.HandleFunc("POST /salir", a.con(cualquiera, a.cerrarSesion))

	mux.HandleFunc("GET /productos", a.con(usuario.Admin, a.listaProductos))
	mux.HandleFunc("GET /productos/nuevo", a.con(usuario.Admin, a.formularioProducto))
	mux.HandleFunc("GET /productos/{id}", a.con(usuario.Admin, a.formularioProducto))
	mux.HandleFunc("POST /productos", a.con(usuario.Admin, a.guardarProducto))
	mux.HandleFunc("POST /productos/{id}", a.con(usuario.Admin, a.guardarProducto))

	mux.HandleFunc("POST /turno/abrir", a.con(cualquiera, a.abrirTurno))
	mux.HandleFunc("GET /turno", a.con(cualquiera, a.verTurno))
	mux.HandleFunc("POST /turno/movimientos", a.con(cualquiera, a.registrarMovimiento))
	mux.HandleFunc("POST /turno/cerrar", a.con(cualquiera, a.cerrarTurno))
	mux.HandleFunc("GET /turnos", a.con(usuario.Admin, a.listaTurnos))
	mux.HandleFunc("GET /turnos/{id}", a.con(cualquiera, a.detalleTurno))

	mux.HandleFunc("GET /caja", a.con(cualquiera, a.pantallaCaja))
	mux.HandleFunc("POST /api/ventas", a.con(cualquiera, a.cobrar))
	mux.HandleFunc("GET /ventas/{id}", a.con(cualquiera, a.verTicket))
	mux.HandleFunc("POST /ventas/{id}/cancelar", a.con(usuario.Admin, a.cancelarVenta))

	mux.HandleFunc("GET /reportes", a.con(usuario.Admin, a.reportes))

	mux.HandleFunc("GET /usuarios", a.con(usuario.Admin, a.listaUsuarios))
	mux.HandleFunc("POST /usuarios", a.con(usuario.Admin, a.crearUsuario))
	mux.HandleFunc("POST /usuarios/{id}/activo", a.con(usuario.Admin, a.cambiarActivo))

	mux.HandleFunc("GET /ajustes", a.con(usuario.Admin, a.formularioAjustes))
	mux.HandleFunc("POST /ajustes", a.con(usuario.Admin, a.guardarAjustes))

	// Rechaza los envíos que vienen de otro sitio (protección contra CSRF).
	return cabeceras(http.NewCrossOriginProtection().Handler(mux))
}

func cabeceras(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		siguiente.ServeHTTP(w, r)
	})
}

// manejador es un handler que ya sabe quién es la persona con sesión.
type manejador func(w http.ResponseWriter, r *http.Request, u *usuario.Usuario)

// con exige sesión y, si se pide, un rol. Sin sesión manda al login.
func (a *App) con(rol string, siguiente manejador) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		galleta, err := r.Cookie(galletaSesion)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		u, err := a.Usuarios.PorSesion(r.Context(), galleta.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if rol != cualquiera && u.Rol != rol {
			http.Error(w, "No tienes permiso para esta pantalla.", http.StatusForbidden)
			return
		}
		siguiente(w, r, u)
	}
}

type vista map[string]any

// mostrar pinta una página dentro del diseño común.
func (a *App) mostrar(w http.ResponseWriter, r *http.Request, pagina string, u *usuario.Usuario, datos vista) {
	datos["Usuario"] = u
	datos["Negocio"] = a.Negocio.Obtener(r.Context())

	if galleta, err := r.Cookie(galletaAviso); err == nil {
		datos["Error"], _ = url.QueryUnescape(galleta.Value)
		http.SetCookie(w, &http.Cookie{Name: galletaAviso, Path: "/", MaxAge: -1})
	}

	// Se pinta primero en memoria: si la plantilla falla, no sale media página.
	var cuerpo bytes.Buffer
	if err := a.plantillas[pagina].ExecuteTemplate(&cuerpo, "base.html", datos); err != nil {
		a.fallo(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	cuerpo.WriteTo(w)
}

// volver regresa a una pantalla mostrando el aviso de lo que salió mal.
// Los errores internos no se muestran: se registran y la pantalla dice que algo falló.
func (a *App) volver(w http.ResponseWriter, r *http.Request, destino string, err error) {
	mensaje, esAviso := aviso.De(err)
	if !esAviso {
		a.fallo(w, r, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     galletaAviso,
		Value:    url.QueryEscape(mensaje),
		Path:     "/",
		MaxAge:   30,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, destino, http.StatusSeeOther)
}

func (a *App) fallo(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("error interno", "ruta", r.URL.Path, "error", err)
	http.Error(w, "Algo salió mal. Intenta de nuevo.", http.StatusInternalServerError)
}

func cargarPlantillas() map[string]*template.Template {
	archivos, err := fs.Glob(recursos, "plantillas/*.html")
	if err != nil {
		panic(err)
	}

	plantillas := map[string]*template.Template{}
	for _, archivo := range archivos {
		nombre := strings.TrimSuffix(path.Base(archivo), ".html")
		if nombre == "base" {
			continue
		}

		// Cada página se junta con el diseño común (base.html) en su propia plantilla.
		plantillas[nombre] = template.Must(
			template.New("base.html").Funcs(funciones).ParseFS(recursos, "plantillas/base.html", archivo),
		)
	}
	return plantillas
}

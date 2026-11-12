// Archivo: app.go
// Qué hace: es el centro de la capa web. Define la estructura App (que reúne
// todos los repositorios), registra todas las rutas (URL → función), agrega los
// encabezados de seguridad, revisa la sesión y el rol antes de cada pantalla,
// y sabe pintar las plantillas HTML y manejar los errores.

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

// Esta línea especial mete las carpetas "plantillas" y "static" dentro del
// programa compilado, así no hay que copiar archivos aparte al servidor.
//
//go:embed plantillas static
var recursos embed.FS

// Nombres fijos usados en varios lugares.
const (
	// Nombre de la cookie que guarda la sesión.
	galletaSesion = "sesion"
	// Nombre de la cookie que lleva un aviso de una pantalla a la siguiente.
	galletaAviso = "aviso"
	// Rol que pide una ruta abierta a todas las personas con sesión.
	cualquiera = ""
)

// App reúne todo lo que las pantallas necesitan.
type App struct {
	Usuarios  *usuario.Repo
	Productos *producto.Repo
	Ventas    *venta.Repo
	Caja      *caja.Repo
	Negocio   *negocio.Repo
	// Una plantilla ya preparada por cada página.
	plantillas map[string]*template.Template
}

// Nueva crea la App conectando cada repositorio con la base de datos.
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

// Rutas dice qué función atiende cada dirección (método + URL).
func (a *App) Rutas() http.Handler {
	mux := http.NewServeMux()

	// Archivos estáticos (CSS y JavaScript).
	mux.Handle("GET /static/", http.FileServerFS(recursos))
	// Ruta de salud: sirve para comprobar que el servidor está vivo.
	mux.HandleFunc("GET /salud", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	// La raíz "/" manda a la caja. {$} significa "exactamente la raíz".
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/caja", http.StatusSeeOther)
	})

	// Entrar y salir. a.con(...) revisa la sesión (y el rol) antes de llamar a la función.
	mux.HandleFunc("GET /login", a.formularioLogin)
	mux.HandleFunc("POST /login", a.iniciarSesion)
	mux.HandleFunc("POST /salir", a.con(cualquiera, a.cerrarSesion))

	// Catálogo de productos (solo administración).
	mux.HandleFunc("GET /productos", a.con(usuario.Admin, a.listaProductos))
	mux.HandleFunc("GET /productos/nuevo", a.con(usuario.Admin, a.formularioProducto))
	mux.HandleFunc("GET /productos/{id}", a.con(usuario.Admin, a.formularioProducto))
	mux.HandleFunc("POST /productos", a.con(usuario.Admin, a.guardarProducto))
	mux.HandleFunc("POST /productos/{id}", a.con(usuario.Admin, a.guardarProducto))

	// Turnos y cortes de caja.
	mux.HandleFunc("POST /turno/abrir", a.con(cualquiera, a.abrirTurno))
	mux.HandleFunc("GET /turno", a.con(cualquiera, a.verTurno))
	mux.HandleFunc("POST /turno/movimientos", a.con(cualquiera, a.registrarMovimiento))
	mux.HandleFunc("POST /turno/cerrar", a.con(cualquiera, a.cerrarTurno))
	mux.HandleFunc("GET /turnos", a.con(usuario.Admin, a.listaTurnos))
	mux.HandleFunc("GET /turnos/{id}", a.con(cualquiera, a.detalleTurno))

	// Venta: pantalla de caja, cobro, ticket y cancelación.
	mux.HandleFunc("GET /caja", a.con(cualquiera, a.pantallaCaja))
	mux.HandleFunc("POST /api/ventas", a.con(cualquiera, a.cobrar))
	mux.HandleFunc("GET /ventas/{id}", a.con(cualquiera, a.verTicket))
	mux.HandleFunc("POST /ventas/{id}/cancelar", a.con(usuario.Admin, a.cancelarVenta))

	// Reportes de ventas.
	mux.HandleFunc("GET /reportes", a.con(usuario.Admin, a.reportes))

	// Administración de usuarios.
	mux.HandleFunc("GET /usuarios", a.con(usuario.Admin, a.listaUsuarios))
	mux.HandleFunc("POST /usuarios", a.con(usuario.Admin, a.crearUsuario))
	mux.HandleFunc("POST /usuarios/{id}/activo", a.con(usuario.Admin, a.cambiarActivo))

	// Datos del negocio.
	mux.HandleFunc("GET /ajustes", a.con(usuario.Admin, a.formularioAjustes))
	mux.HandleFunc("POST /ajustes", a.con(usuario.Admin, a.guardarAjustes))

	// Rechaza los envíos que vienen de otro sitio (protección contra CSRF).
	return cabeceras(http.NewCrossOriginProtection().Handler(mux))
}

// cabeceras agrega encabezados de seguridad a todas las respuestas.
func cabeceras(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Solo permite cargar recursos del mismo sitio y prohíbe meter la página en un iframe.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
		// Evita que el navegador "adivine" el tipo de archivo.
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Solo manda la dirección de origen a páginas del mismo sitio.
		w.Header().Set("Referrer-Policy", "same-origin")
		// Sigue con la petición normal.
		siguiente.ServeHTTP(w, r)
	})
}

// manejador es un handler que ya sabe quién es la persona con sesión.
type manejador func(w http.ResponseWriter, r *http.Request, u *usuario.Usuario)

// con exige sesión y, si se pide, un rol. Sin sesión manda al login.
func (a *App) con(rol string, siguiente manejador) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Sin cookie de sesión: al login.
		galleta, err := r.Cookie(galletaSesion)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Busca la persona dueña de esa sesión; si no existe o venció: al login.
		u, err := a.Usuarios.PorSesion(r.Context(), galleta.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Si la ruta pide un rol y la persona no lo tiene: prohibido.
		if rol != cualquiera && u.Rol != rol {
			http.Error(w, "No tienes permiso para esta pantalla.", http.StatusForbidden)
			return
		}
		// Todo en orden: ejecuta la pantalla.
		siguiente(w, r, u)
	}
}

// vista son los datos que se le pasan a una plantilla: nombre → valor.
type vista map[string]any

// mostrar pinta una página dentro del diseño común.
func (a *App) mostrar(w http.ResponseWriter, r *http.Request, pagina string, u *usuario.Usuario, datos vista) {
	// Datos que toda página necesita: quién es la persona y los datos del negocio.
	datos["Usuario"] = u
	datos["Negocio"] = a.Negocio.Obtener(r.Context())

	// Si la pantalla anterior dejó un aviso en una cookie, lo muestra y borra la cookie.
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

	// Ya salió bien: ahora sí se envía al navegador.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	cuerpo.WriteTo(w)
}

// volver regresa a una pantalla mostrando el aviso de lo que salió mal.
// Los errores internos no se muestran: se registran y la pantalla dice que algo falló.
func (a *App) volver(w http.ResponseWriter, r *http.Request, destino string, err error) {
	// ¿Es un aviso pensado para la persona, o un error interno?
	mensaje, esAviso := aviso.De(err)
	if !esAviso {
		a.fallo(w, r, err)
		return
	}

	// Guarda el mensaje 30 segundos en una cookie para mostrarlo en la pantalla de destino.
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

// fallo registra un error interno y responde un mensaje genérico (sin detalles técnicos).
func (a *App) fallo(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("error interno", "ruta", r.URL.Path, "error", err)
	http.Error(w, "Algo salió mal. Intenta de nuevo.", http.StatusInternalServerError)
}

// cargarPlantillas prepara una plantilla por cada página HTML.
func cargarPlantillas() map[string]*template.Template {
	// Busca todos los archivos .html de la carpeta plantillas.
	archivos, err := fs.Glob(recursos, "plantillas/*.html")
	if err != nil {
		// Solo falla si el patrón está mal escrito: es un error del programador.
		panic(err)
	}

	plantillas := map[string]*template.Template{}
	for _, archivo := range archivos {
		// El nombre de la página es el del archivo sin ".html".
		nombre := strings.TrimSuffix(path.Base(archivo), ".html")
		// base.html es el diseño común, no una página.
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

// Archivo: app.go
// Qué hace: es el centro de la capa web. Define la estructura App (que reúne
// todos los repositorios), registra las rutas de la API JSON y sirve la
// aplicación React ya compilada. También revisa la sesión y el rol antes de
// cada endpoint y traduce los errores a respuestas JSON.

// Package web expone la API JSON y sirve la interfaz React.
package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
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

// Mete la carpeta "dist" (la interfaz React compilada con `npm run build`)
// dentro del programa. "all:" incluye también archivos que empiezan con punto.
//
//go:embed all:dist
var interfaz embed.FS

// Nombres fijos usados en varios lugares.
const (
	// Nombre de la cookie que guarda la sesión.
	galletaSesion = "sesion"
	// Rol que pide una ruta abierta a todas las personas con sesión.
	cualquiera = ""
)

// ErrSolicitud es el aviso cuando el JSON recibido no se pudo leer.
var ErrSolicitud = aviso.Nuevo("No se entendió la solicitud. Recarga la pantalla e intenta de nuevo.")

// App reúne todo lo que los endpoints necesitan.
type App struct {
	Usuarios  *usuario.Repo
	Productos *producto.Repo
	Ventas    *venta.Repo
	Caja      *caja.Repo
	Negocio   *negocio.Repo
}

// Nueva crea la App conectando cada repositorio con la base de datos.
func Nueva(db *mongo.Database) *App {
	return &App{
		Usuarios:  usuario.NuevoRepo(db),
		Productos: producto.NuevoRepo(db),
		Ventas:    venta.NuevoRepo(db),
		Caja:      caja.NuevoRepo(db),
		Negocio:   negocio.NuevoRepo(db),
	}
}

// Rutas dice qué función atiende cada dirección (método + URL).
// a.con(...) revisa la sesión (y el rol) antes de llamar a la función.
func (a *App) Rutas() http.Handler {
	mux := http.NewServeMux()

	// Salud: sirve para comprobar que el servidor está vivo.
	mux.HandleFunc("GET /api/salud", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	// Sesión: quién soy, entrar y salir.
	mux.HandleFunc("GET /api/yo", a.yo)
	mux.HandleFunc("POST /api/login", a.iniciarSesion)
	mux.HandleFunc("POST /api/salir", a.con(cualquiera, a.cerrarSesion))

	// Catálogo de productos (solo administración).
	mux.HandleFunc("GET /api/productos", a.con(usuario.Admin, a.listaProductos))
	mux.HandleFunc("POST /api/productos", a.con(usuario.Admin, a.guardarProducto))
	mux.HandleFunc("PUT /api/productos/{id}", a.con(usuario.Admin, a.guardarProducto))

	// Turnos y cortes de caja.
	mux.HandleFunc("GET /api/caja", a.con(cualquiera, a.caja))
	mux.HandleFunc("POST /api/turno/abrir", a.con(cualquiera, a.abrirTurno))
	mux.HandleFunc("GET /api/turno", a.con(cualquiera, a.verTurno))
	mux.HandleFunc("POST /api/turno/movimientos", a.con(cualquiera, a.registrarMovimiento))
	mux.HandleFunc("POST /api/turno/cerrar", a.con(cualquiera, a.cerrarTurno))
	mux.HandleFunc("GET /api/turnos", a.con(usuario.Admin, a.listaTurnos))
	mux.HandleFunc("GET /api/turnos/{id}", a.con(cualquiera, a.detalleTurno))

	// Ventas: cobro, ticket y cancelación.
	mux.HandleFunc("POST /api/ventas", a.con(cualquiera, a.cobrar))
	mux.HandleFunc("GET /api/ventas/{id}", a.con(cualquiera, a.verVenta))
	mux.HandleFunc("POST /api/ventas/{id}/cancelar", a.con(usuario.Admin, a.cancelarVenta))

	// Reportes (solo administración).
	mux.HandleFunc("GET /api/reportes", a.con(usuario.Admin, a.reportes))
	mux.HandleFunc("GET /api/reportes/csv", a.con(usuario.Admin, a.descargarCSV))

	// Usuarios y datos del negocio (solo administración).
	mux.HandleFunc("GET /api/usuarios", a.con(usuario.Admin, a.listaUsuarios))
	mux.HandleFunc("POST /api/usuarios", a.con(usuario.Admin, a.crearUsuario))
	mux.HandleFunc("POST /api/usuarios/{id}/activo", a.con(usuario.Admin, a.cambiarActivo))
	mux.HandleFunc("PUT /api/ajustes", a.con(usuario.Admin, a.guardarAjustes))

	// Cualquier otra dirección es una pantalla de React.
	mux.Handle("/", pantallas())

	// Rechaza los envíos que vienen de otro sitio (protección contra CSRF).
	return cabeceras(http.NewCrossOriginProtection().Handler(mux))
}

// pantallas sirve los archivos de React. Si la dirección no es un archivo
// (por ejemplo /productos), entrega index.html y React decide qué pantalla mostrar.
func pantallas() http.Handler {
	dist, _ := fs.Sub(interfaz, "dist")
	archivos := http.FileServerFS(dist)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(dist, strings.TrimPrefix(path.Clean(r.URL.Path), "/")); err != nil {
			r.URL.Path = "/"
		}
		archivos.ServeHTTP(w, r)
	})
}

// cabeceras agrega encabezados de seguridad a todas las respuestas.
func cabeceras(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Solo permite cargar recursos del mismo sitio y prohíbe meter la página en un iframe.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		// Evita que el navegador "adivine" el tipo de archivo.
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Solo manda la dirección de origen a páginas del mismo sitio.
		w.Header().Set("Referrer-Policy", "same-origin")
		siguiente.ServeHTTP(w, r)
	})
}

// manejador es un handler que ya sabe quién es la persona con sesión.
type manejador func(w http.ResponseWriter, r *http.Request, u *usuario.Usuario)

// persona busca a quien tiene la sesión de la cookie (nil si no hay sesión válida).
func (a *App) persona(r *http.Request) *usuario.Usuario {
	galleta, err := r.Cookie(galletaSesion)
	if err != nil {
		return nil
	}
	u, err := a.Usuarios.PorSesion(r.Context(), galleta.Value)
	if err != nil {
		return nil
	}
	return u
}

// con exige sesión y, si se pide, un rol. Sin sesión responde 401.
func (a *App) con(rol string, siguiente manejador) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := a.persona(r)
		if u == nil {
			responder(w, http.StatusUnauthorized, map[string]string{"error": "Inicia sesión para continuar."})
			return
		}
		// Si la ruta pide un rol y la persona no lo tiene: prohibido.
		if rol != cualquiera && u.Rol != rol {
			responder(w, http.StatusForbidden, map[string]string{"error": "No tienes permiso para esta pantalla."})
			return
		}
		siguiente(w, r, u)
	}
}

// responder envía una respuesta en formato JSON con el código HTTP indicado.
func responder(w http.ResponseWriter, estado int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	json.NewEncoder(w).Encode(cuerpo)
}

// leer decodifica el JSON del cuerpo (máximo 1 MB). Si no se entiende, ya respondió el error.
func (a *App) leer(w http.ResponseWriter, r *http.Request, destino any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(destino); err != nil {
		a.fallo(w, r, ErrSolicitud)
		return false
	}
	return true
}

// fallo responde un error en JSON: los avisos tal cual (422), los errores internos con un mensaje genérico (500).
func (a *App) fallo(w http.ResponseWriter, r *http.Request, err error) {
	if mensaje, esAviso := aviso.De(err); esAviso {
		responder(w, http.StatusUnprocessableEntity, map[string]string{"error": mensaje})
		return
	}

	slog.Error("error interno", "ruta", r.URL.Path, "error", err)
	responder(w, http.StatusInternalServerError, map[string]string{"error": "Algo salió mal. Intenta de nuevo."})
}

// sinContenido responde 204: la acción salió bien y no hay nada que devolver.
func sinContenido(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

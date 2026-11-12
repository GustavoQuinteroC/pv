// Archivo: ajustes.go
// Qué hace: guarda los datos del negocio (solo administración): el nombre y
// el mensaje que sale al final del ticket.

package web

import (
	"net/http"
	"strings"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/negocio"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
)

// ErrNombreNegocio es el aviso cuando se intenta guardar sin nombre.
var ErrNombreNegocio = aviso.Nuevo("Escribe el nombre del negocio.")

// guardarAjustes valida y guarda los datos del negocio.
func (a *App) guardarAjustes(w http.ResponseWriter, r *http.Request, _ *usuario.Usuario) {
	var ajustes negocio.Negocio
	if !a.leer(w, r, &ajustes) {
		return
	}

	ajustes.Nombre = strings.TrimSpace(ajustes.Nombre)
	ajustes.Mensaje = strings.TrimSpace(ajustes.Mensaje)
	// El nombre es obligatorio; el mensaje puede ir vacío.
	if ajustes.Nombre == "" {
		a.fallo(w, r, ErrNombreNegocio)
		return
	}

	if err := a.Negocio.Guardar(r.Context(), ajustes); err != nil {
		a.fallo(w, r, err)
		return
	}
	sinContenido(w)
}

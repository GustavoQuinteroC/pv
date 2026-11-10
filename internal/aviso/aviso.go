// Package aviso define los errores que se le muestran a la persona usuaria.
//
// Cualquier otro error (la base de datos caída, por ejemplo) es interno:
// se registra en el log y la pantalla solo dice que algo salió mal.
package aviso

import "errors"

type Error struct {
	Mensaje string
}

func (e *Error) Error() string {
	return e.Mensaje
}

func Nuevo(mensaje string) error {
	return &Error{Mensaje: mensaje}
}

// De devuelve el mensaje si el error es un aviso para la persona usuaria.
func De(err error) (string, bool) {
	var aviso *Error
	if errors.As(err, &aviso) {
		return aviso.Mensaje, true
	}
	return "", false
}

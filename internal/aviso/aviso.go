// Archivo: aviso.go
// Qué hace: define un tipo de error especial para los mensajes que SÍ se le
// muestran a la persona ("Correo o contraseña incorrectos"). Cualquier otro error
// (por ejemplo, la base de datos caída) es interno: se guarda en el log y la
// pantalla solo dice "algo salió mal", para no revelar detalles técnicos.

// Package aviso define los errores que se le muestran a la persona usuaria.
//
// Cualquier otro error (la base de datos caída, por ejemplo) es interno:
// se registra en el log y la pantalla solo dice que algo salió mal.
package aviso

// "errors" es la biblioteca estándar de Go para trabajar con errores.
import "errors"

// Error es una ficha con un solo campo: el mensaje que verá la persona.
type Error struct {
	// El texto del aviso.
	Mensaje string
}

// Error() es un método (función pegada a una ficha). Go exige este método
// para que un tipo cuente como "error". Devuelve el mensaje.
func (e *Error) Error() string {
	// "e" es el aviso sobre el que se llamó el método.
	return e.Mensaje
}

// Nuevo crea un aviso con el texto dado. Devuelve un "error" para usarlo como cualquier otro.
func Nuevo(mensaje string) error {
	// "&" entrega la dirección de la ficha en memoria (un puntero), no una copia.
	return &Error{Mensaje: mensaje}
}

// De revisa si un error es un aviso para la persona. Devuelve dos cosas:
// el mensaje y un true/false que dice si de verdad era un aviso.
func De(err error) (string, bool) {
	// Variable vacía donde se guardará el aviso si se encuentra.
	var aviso *Error
	// errors.As recorre el error (y los que lo envuelven) buscando uno del tipo *Error.
	if errors.As(err, &aviso) {
		// Era un aviso: se devuelve su texto y "true".
		return aviso.Mensaje, true
	}
	// No era un aviso: texto vacío y "false".
	return "", false
}

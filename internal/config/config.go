// Archivo: config.go
// Qué hace: lee los ajustes del programa (dónde está la base de datos, en qué
// puerto escuchar, quién es el primer administrador) desde variables de entorno.
// Una variable de entorno es un dato que el sistema operativo le pasa al programa
// al arrancar, así las contraseñas no se escriben dentro del código.

// Package config lee la configuración del programa desde variables de entorno.
// (En Go, cada carpeta es un "paquete": un grupo de código con un solo propósito.)
package config

// "import" trae código ya hecho. "os" es una biblioteca de Go que habla con el sistema operativo.
import "os"

// Config es una "ficha" (struct) con todos los ajustes juntos.
// Cada línea es un campo: nombre a la izquierda, tipo de dato a la derecha.
type Config struct {
	// Dirección de MongoDB.
	MongoURI string
	// Nombre de la base de datos dentro de MongoDB.
	MongoBD string
	// Puerto por el que el programa recibe visitas.
	Puerto string
	// Correo del primer administrador.
	AdminCorreo string
	// Contraseña del primer administrador.
	AdminClave string
	// true o false: ¿cargar productos de ejemplo la primera vez?
	SembrarDemo bool
}

// Cargar arma la ficha Config leyendo las variables de entorno.
// Una función se declara con "func", lleva un nombre y dice qué devuelve (aquí, una Config).
func Cargar() Config {
	// "return" devuelve el resultado. Config{...} crea una ficha rellenando sus campos.
	return Config{
		// Si no existe MONGO_URI, se usa la dirección de la derecha (MongoDB en la misma máquina).
		MongoURI: texto("MONGO_URI", "mongodb://localhost:27017"),
		// Base de datos llamada "pv" por omisión.
		MongoBD: texto("MONGO_BD", "pv"),
		// Puerto 8080 por omisión.
		Puerto: texto("PUERTO", "8080"),
		// Sin correo de administrador por omisión (queda vacío).
		AdminCorreo: texto("ADMIN_CORREO", ""),
		// Sin contraseña de administrador por omisión.
		AdminClave: texto("ADMIN_CLAVE", ""),
		// Solo se siembran datos de ejemplo si la variable vale exactamente "true".
		SembrarDemo: texto("SEMBRAR_DEMO", "false") == "true",
	}
}

// texto lee una variable de entorno; si está vacía o no existe, devuelve el valor por omisión.
// Empieza con minúscula: así solo se puede usar dentro de este paquete.
func texto(nombre, porDefecto string) string {
	// Lee la variable y, en la misma línea, pregunta si trae algo.
	if valor := os.Getenv(nombre); valor != "" {
		// Sí trae algo: ese es el valor que se usa.
		return valor
	}
	// No trae nada: se usa el valor por omisión.
	return porDefecto
}

// Package config lee la configuración del programa desde variables de entorno.
package config

import "os"

type Config struct {
	MongoURI    string
	MongoBD     string
	Puerto      string
	AdminCorreo string
	AdminClave  string
	SembrarDemo bool
}

func Cargar() Config {
	return Config{
		MongoURI:    texto("MONGO_URI", "mongodb://localhost:27017"),
		MongoBD:     texto("MONGO_BD", "pv"),
		Puerto:      texto("PUERTO", "8080"),
		AdminCorreo: texto("ADMIN_CORREO", ""),
		AdminClave:  texto("ADMIN_CLAVE", ""),
		SembrarDemo: texto("SEMBRAR_DEMO", "false") == "true",
	}
}

func texto(nombre, porDefecto string) string {
	if valor := os.Getenv(nombre); valor != "" {
		return valor
	}
	return porDefecto
}

// Archivo: main.go
// Qué hace: es la puerta de entrada del programa. Lee la configuración,
// se conecta a MongoDB, crea los índices, deja listos los datos iniciales,
// arranca el servidor web y lo apaga con cuidado al recibir Ctrl+C o la señal de Docker.

// Command pv es el punto de venta: un servidor web con MongoDB.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	// El guion bajo importa el paquete solo por su efecto: incluye las zonas horarias
	// dentro del programa, para que TZ funcione en una imagen de Docker sin sistema.
	_ "time/tzdata"

	"github.com/GustavoQuinteroC/pv/internal/bd"
	"github.com/GustavoQuinteroC/pv/internal/config"
	"github.com/GustavoQuinteroC/pv/internal/semilla"
	"github.com/GustavoQuinteroC/pv/internal/web"
)

// main es lo primero que Go ejecuta. Solo llama a ejecutar y reporta si hubo error.
func main() {
	if err := ejecutar(); err != nil {
		// Escribe el error en el log y sale con código 1 (señal de fallo).
		slog.Error("el programa terminó con error", "error", err)
		os.Exit(1)
	}
}

// ejecutar hace todo el trabajo y devuelve un error si algo falla.
func ejecutar() error {
	// Lee la configuración (puerto, MongoDB, administrador…) de las variables de entorno.
	cfg := config.Cargar()

	// ctx se cancela con Ctrl+C o con la señal de apagado de Docker.
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// defer: al terminar la función, libera lo que creó signal.
	defer cancelar()

	// Se conecta a MongoDB.
	db, err := bd.Conectar(ctx, cfg.MongoURI, cfg.MongoBD)
	if err != nil {
		return err
	}
	// Crea los índices (reglas de unicidad y búsquedas rápidas) si no existen.
	if err := bd.CrearIndices(ctx, db); err != nil {
		return err
	}

	// Arma la aplicación web con todos sus repositorios.
	app := web.Nueva(db)
	// Crea el primer administrador y los productos de ejemplo, si hace falta.
	err = semilla.Aplicar(ctx, app.Usuarios, app.Productos, cfg.AdminCorreo, cfg.AdminClave, cfg.SembrarDemo)
	if err != nil {
		return err
	}

	// Configura el servidor HTTP.
	servidor := &http.Server{
		// Escucha en este puerto.
		Addr: ":" + cfg.Puerto,
		// Todas las rutas de la aplicación.
		Handler: app.Rutas(),
		// Si alguien tarda más de 5 s en mandar sus encabezados, se corta (evita conexiones colgadas).
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Un "canal" es un tubo para pasar datos entre tareas. Aquí llegará un error si el servidor falla.
	errores := make(chan error, 1)
	// "go" lanza el servidor en paralelo, para poder seguir esperando la señal de apagado.
	go func() { errores <- servidor.ListenAndServe() }()
	slog.Info("punto de venta listo", "puerto", cfg.Puerto)

	// Espera lo que ocurra primero: un error del servidor o la orden de apagar.
	select {
	case err := <-errores:
		return err
	case <-ctx.Done():
		return apagar(servidor)
	}
}

// apagar deja terminar las peticiones en curso antes de cerrar.
func apagar(servidor *http.Server) error {
	// Da hasta 10 segundos para terminar lo pendiente.
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()

	err := servidor.Shutdown(ctx)
	// ErrServerClosed significa "cerró bien": no es un error real.
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

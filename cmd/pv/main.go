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
	_ "time/tzdata" // zonas horarias incluidas, para que TZ funcione en una imagen sin sistema

	"github.com/GustavoQuinteroC/pv/internal/bd"
	"github.com/GustavoQuinteroC/pv/internal/config"
	"github.com/GustavoQuinteroC/pv/internal/semilla"
	"github.com/GustavoQuinteroC/pv/internal/web"
)

func main() {
	if err := ejecutar(); err != nil {
		slog.Error("el programa terminó con error", "error", err)
		os.Exit(1)
	}
}

func ejecutar() error {
	cfg := config.Cargar()

	// ctx se cancela con Ctrl+C o con la señal de apagado de Docker.
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelar()

	db, err := bd.Conectar(ctx, cfg.MongoURI, cfg.MongoBD)
	if err != nil {
		return err
	}
	if err := bd.CrearIndices(ctx, db); err != nil {
		return err
	}

	app := web.Nueva(db)
	err = semilla.Aplicar(ctx, app.Usuarios, app.Productos, cfg.AdminCorreo, cfg.AdminClave, cfg.SembrarDemo)
	if err != nil {
		return err
	}

	servidor := &http.Server{
		Addr:              ":" + cfg.Puerto,
		Handler:           app.Rutas(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errores := make(chan error, 1)
	go func() { errores <- servidor.ListenAndServe() }()
	slog.Info("punto de venta listo", "puerto", cfg.Puerto)

	select {
	case err := <-errores:
		return err
	case <-ctx.Done():
		return apagar(servidor)
	}
}

// apagar deja terminar las peticiones en curso antes de cerrar.
func apagar(servidor *http.Server) error {
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()

	err := servidor.Shutdown(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

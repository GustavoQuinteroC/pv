// Package semilla deja listo un sistema recién instalado: el primer administrador
// y, si se pide, unos productos de ejemplo para probarlo.
package semilla

import (
	"context"
	"log/slog"

	"github.com/GustavoQuinteroC/pv/internal/producto"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
)

// Aplicar solo actúa sobre colecciones vacías, así que es seguro correrlo en cada arranque.
func Aplicar(ctx context.Context, usuarios *usuario.Repo, productos *producto.Repo, adminCorreo, adminClave string, conDemo bool) error {
	if err := crearAdministrador(ctx, usuarios, adminCorreo, adminClave); err != nil {
		return err
	}
	if conDemo {
		return crearProductosDeEjemplo(ctx, productos)
	}
	return nil
}

func crearAdministrador(ctx context.Context, usuarios *usuario.Repo, correo, clave string) error {
	hay, err := usuarios.Hay(ctx)
	if err != nil || hay {
		return err
	}
	if correo == "" || clave == "" {
		slog.Warn("no hay usuarios y faltan ADMIN_CORREO y ADMIN_CLAVE: nadie podrá entrar")
		return nil
	}

	slog.Info("creando el primer administrador", "correo", correo)
	return usuarios.Crear(ctx, "Administración", correo, clave, usuario.Admin)
}

// Productos inventados, para mostrar cada giro: granel por kilo y litro, y piezas.
func crearProductosDeEjemplo(ctx context.Context, productos *producto.Repo) error {
	hay, err := productos.Hay(ctx)
	if err != nil || hay {
		return err
	}

	ejemplos := []producto.Producto{
		{Nombre: "Tortilla de maíz", Precio: 2400, Unidad: "kg", Favorito: true},
		{Nombre: "Queso Oaxaca", Precio: 16000, Unidad: "kg", Favorito: true},
		{Nombre: "Crema fresca", Precio: 7500, Unidad: "litro", Favorito: true},
		{Nombre: "Bistec de res", Precio: 19500, Unidad: "kg", Favorito: true},
		{Nombre: "Pechuga de pollo", Precio: 9800, Unidad: "kg"},
		{Nombre: "Chicharrón", Precio: 18000, Unidad: "kg"},
		{Nombre: "Gomitas a granel", Precio: 9000, Unidad: "kg", Favorito: true},
		{Nombre: "Paleta de caramelo", Precio: 500, Unidad: "pieza", Codigo: "7500000000011", Favorito: true},
		{Nombre: "Chicle de menta", Precio: 300, Unidad: "pieza", Codigo: "7500000000028", Favorito: true},
		{Nombre: "Refresco de cola 600 ml", Precio: 1800, Unidad: "pieza", Codigo: "7500000000035", Favorito: true},
		{Nombre: "Huevo (pieza)", Precio: 350, Unidad: "pieza"},
		{Nombre: "Pan de dulce", Precio: 1000, Unidad: "pieza", Favorito: true},
	}

	for _, ejemplo := range ejemplos {
		ejemplo.Activo = true
		if err := productos.Guardar(ctx, ejemplo); err != nil {
			return err
		}
	}
	slog.Info("productos de ejemplo creados", "cantidad", len(ejemplos))
	return nil
}

// Archivo: semilla.go
// Qué hace: deja listo un sistema recién instalado. Crea al primer
// administrador (con el correo y la contraseña de las variables de entorno) y,
// si se pide, carga unos productos de ejemplo para poder probar enseguida.
// "Semilla" es el nombre que se le da a los datos iniciales.

// Package semilla deja listo un sistema recién instalado: el primer administrador
// y, si se pide, unos productos de ejemplo para probarlo.
package semilla

import (
	"context"
	// "slog" escribe mensajes en el registro (log) del programa.
	"log/slog"

	"github.com/GustavoQuinteroC/pv/internal/producto"
	"github.com/GustavoQuinteroC/pv/internal/usuario"
)

// Aplicar solo actúa sobre colecciones vacías, así que es seguro correrlo en cada arranque.
func Aplicar(ctx context.Context, usuarios *usuario.Repo, productos *producto.Repo, adminCorreo, adminClave string, conDemo bool) error {
	// Primero el administrador.
	if err := crearAdministrador(ctx, usuarios, adminCorreo, adminClave); err != nil {
		return err
	}
	// Después, si se pidió, los productos de ejemplo.
	if conDemo {
		return crearProductosDeEjemplo(ctx, productos)
	}
	return nil
}

// crearAdministrador crea la primera cuenta, solo si todavía no hay ninguna.
func crearAdministrador(ctx context.Context, usuarios *usuario.Repo, correo, clave string) error {
	// ¿Ya hay usuarios?
	hay, err := usuarios.Hay(ctx)
	// Si falló la consulta, o si ya hay usuarios, no se hace nada.
	if err != nil || hay {
		return err
	}
	// Sin correo o sin contraseña no se puede crear la cuenta: se avisa en el log.
	if correo == "" || clave == "" {
		slog.Warn("no hay usuarios y faltan ADMIN_CORREO y ADMIN_CLAVE: nadie podrá entrar")
		return nil
	}

	// Deja constancia en el log (sin escribir la contraseña).
	slog.Info("creando el primer administrador", "correo", correo)
	return usuarios.Crear(ctx, "Administración", correo, clave, usuario.Admin)
}

// Productos inventados, para mostrar cada giro: granel por kilo y litro, y piezas.
func crearProductosDeEjemplo(ctx context.Context, productos *producto.Repo) error {
	// Si ya hay productos, no se agrega nada (no se duplican).
	hay, err := productos.Hay(ctx)
	if err != nil || hay {
		return err
	}

	// Lista de productos de ejemplo. Precio en centavos (2400 = $24.00 por kilo).
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

	// Guarda uno por uno.
	for _, ejemplo := range ejemplos {
		// Todos quedan activos.
		ejemplo.Activo = true
		if err := productos.Guardar(ctx, ejemplo); err != nil {
			return err
		}
	}
	// Deja constancia de cuántos se crearon.
	slog.Info("productos de ejemplo creados", "cantidad", len(ejemplos))
	return nil
}

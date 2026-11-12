// Archivo: producto.go
// Qué hace: es el catálogo de lo que se vende. Define qué datos tiene un
// producto (nombre, precio, unidad, favorito...), valida que estén bien y
// permite listarlos, buscarlos y guardarlos en MongoDB.

// Package producto es el catálogo de lo que se vende.
package producto

import (
	// Para cancelar o limitar operaciones.
	"context"
	// Funciones para trabajar con listas (aquí, saber si algo está en una lista).
	"slices"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/bd"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Pieza es la unidad de lo que se cuenta de uno en uno (un refresco, una paleta).
const Pieza = "pieza"

// Unidades en que se vende un producto. Todo lo que no es pieza se vende a granel.
var Unidades = []string{Pieza, "kg", "litro"}

// Errores que se le muestran a la persona al capturar un producto.
var (
	ErrNombre         = aviso.Nuevo("Escribe el nombre del producto.")
	ErrPrecio         = aviso.Nuevo("El precio debe ser mayor a cero.")
	ErrUnidad         = aviso.Nuevo("Elige una unidad válida.")
	ErrCodigoRepetido = aviso.Nuevo("Ya existe otro producto con ese código de barras.")
)

// Producto guarda el precio en centavos por unidad: nunca se usan decimales para dinero.
// Las etiquetas `bson` son los nombres en MongoDB; las `json` son los nombres que
// ve el JavaScript del navegador.
type Producto struct {
	// Identificador único.
	ID bson.ObjectID `bson:"_id,omitempty" json:"id"`
	// Nombre que se muestra en la caja.
	Nombre string `bson:"nombre" json:"nombre"`
	// Código de barras (opcional; si está vacío no se guarda).
	Codigo string `bson:"codigo,omitempty" json:"codigo"`
	// Precio en centavos por cada unidad (1850 = $18.50).
	Precio int64 `bson:"precio" json:"precio"`
	// "pieza", "kg" o "litro".
	Unidad string `bson:"unidad" json:"unidad"`
	// true = aparece en los mosaicos de la caja.
	Favorito bool `bson:"favorito" json:"favorito"`
	// false = ya no se puede vender.
	Activo bool `bson:"activo" json:"activo"`
}

// Granel dice si se vende por peso o volumen (cantidades con decimales).
func (p Producto) Granel() bool {
	return p.Unidad != Pieza
}

// validar revisa que el producto tenga datos correctos antes de guardarlo.
func (p Producto) validar() error {
	// "switch" elige el primer caso que sea verdadero.
	switch {
	case p.Nombre == "":
		return ErrNombre
	case p.Precio <= 0:
		return ErrPrecio
	// La unidad debe estar en la lista de unidades válidas.
	case !slices.Contains(Unidades, p.Unidad):
		return ErrUnidad
	}
	// Ningún caso falló: el producto es válido.
	return nil
}

// Repo guarda el acceso a la colección de productos.
type Repo struct {
	productos *mongo.Collection
}

// NuevoRepo prepara el acceso a la colección "productos".
func NuevoRepo(db *mongo.Database) *Repo {
	return &Repo{productos: db.Collection("productos")}
}

// Listar devuelve los productos ordenados por nombre; si se pide, solo los activos.
func (r *Repo) Listar(ctx context.Context, soloActivos bool) ([]Producto, error) {
	// Filtro vacío = todos.
	filtro := bson.M{}
	// Si solo se quieren los activos, se agrega esa condición.
	if soloActivos {
		filtro["activo"] = true
	}
	return bd.Todos[Producto](ctx, r.productos, filtro, bson.M{"nombre": 1})
}

// Catalogo devuelve los productos activos por id, listos para armar una venta.
// Un "map" es un diccionario: se busca por la llave (el id) y devuelve el producto.
func (r *Repo) Catalogo(ctx context.Context) (map[bson.ObjectID]Producto, error) {
	// Primero obtiene la lista de activos.
	activos, err := r.Listar(ctx, true)
	if err != nil {
		return nil, err
	}

	// Crea el diccionario vacío, con espacio reservado para todos.
	catalogo := make(map[bson.ObjectID]Producto, len(activos))
	// Mete cada producto en el diccionario usando su id como llave.
	for _, p := range activos {
		catalogo[p.ID] = p
	}
	return catalogo, nil
}

// Obtener busca un producto por su id.
func (r *Repo) Obtener(ctx context.Context, id bson.ObjectID) (Producto, error) {
	var p Producto
	err := r.productos.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	return p, err
}

// Guardar crea el producto si no tiene id, y si lo tiene lo reemplaza.
func (r *Repo) Guardar(ctx context.Context, p Producto) error {
	// Antes de guardar, valida.
	if err := p.validar(); err != nil {
		return err
	}

	var err error
	// Sin id = producto nuevo.
	if p.ID.IsZero() {
		_, err = r.productos.InsertOne(ctx, p)
	} else {
		// Con id = se reemplaza el documento existente.
		_, err = r.productos.ReplaceOne(ctx, bson.M{"_id": p.ID}, p)
	}

	// Si el índice único del código se queja, ya existe otro con ese código.
	if mongo.IsDuplicateKeyError(err) {
		return ErrCodigoRepetido
	}
	return err
}

// Hay dice si ya existe al menos un producto.
func (r *Repo) Hay(ctx context.Context) (bool, error) {
	cantidad, err := r.productos.CountDocuments(ctx, bson.M{})
	return cantidad > 0, err
}

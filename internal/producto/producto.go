// Package producto es el catálogo de lo que se vende.
package producto

import (
	"context"
	"slices"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/bd"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const Pieza = "pieza"

// Unidades en que se vende un producto. Todo lo que no es pieza se vende a granel.
var Unidades = []string{Pieza, "kg", "litro"}

var (
	ErrNombre         = aviso.Nuevo("Escribe el nombre del producto.")
	ErrPrecio         = aviso.Nuevo("El precio debe ser mayor a cero.")
	ErrUnidad         = aviso.Nuevo("Elige una unidad válida.")
	ErrCodigoRepetido = aviso.Nuevo("Ya existe otro producto con ese código de barras.")
)

// Producto guarda el precio en centavos por unidad: nunca se usan decimales para dinero.
type Producto struct {
	ID       bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Nombre   string        `bson:"nombre" json:"nombre"`
	Codigo   string        `bson:"codigo,omitempty" json:"codigo"`
	Precio   int64         `bson:"precio" json:"precio"`
	Unidad   string        `bson:"unidad" json:"unidad"`
	Favorito bool          `bson:"favorito" json:"favorito"`
	Activo   bool          `bson:"activo" json:"activo"`
}

func (p Producto) Granel() bool {
	return p.Unidad != Pieza
}

func (p Producto) validar() error {
	switch {
	case p.Nombre == "":
		return ErrNombre
	case p.Precio <= 0:
		return ErrPrecio
	case !slices.Contains(Unidades, p.Unidad):
		return ErrUnidad
	}
	return nil
}

type Repo struct {
	productos *mongo.Collection
}

func NuevoRepo(db *mongo.Database) *Repo {
	return &Repo{productos: db.Collection("productos")}
}

func (r *Repo) Listar(ctx context.Context, soloActivos bool) ([]Producto, error) {
	filtro := bson.M{}
	if soloActivos {
		filtro["activo"] = true
	}
	return bd.Todos[Producto](ctx, r.productos, filtro, bson.M{"nombre": 1})
}

// Catalogo devuelve los productos activos por id, listos para armar una venta.
func (r *Repo) Catalogo(ctx context.Context) (map[bson.ObjectID]Producto, error) {
	activos, err := r.Listar(ctx, true)
	if err != nil {
		return nil, err
	}

	catalogo := make(map[bson.ObjectID]Producto, len(activos))
	for _, p := range activos {
		catalogo[p.ID] = p
	}
	return catalogo, nil
}

func (r *Repo) Obtener(ctx context.Context, id bson.ObjectID) (Producto, error) {
	var p Producto
	err := r.productos.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	return p, err
}

// Guardar crea el producto si no tiene id, y si lo tiene lo reemplaza.
func (r *Repo) Guardar(ctx context.Context, p Producto) error {
	if err := p.validar(); err != nil {
		return err
	}

	var err error
	if p.ID.IsZero() {
		_, err = r.productos.InsertOne(ctx, p)
	} else {
		_, err = r.productos.ReplaceOne(ctx, bson.M{"_id": p.ID}, p)
	}

	if mongo.IsDuplicateKeyError(err) {
		return ErrCodigoRepetido
	}
	return err
}

func (r *Repo) Hay(ctx context.Context) (bool, error) {
	cantidad, err := r.productos.CountDocuments(ctx, bson.M{})
	return cantidad > 0, err
}

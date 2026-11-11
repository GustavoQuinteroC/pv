package venta

import (
	"context"
	"time"

	"github.com/GustavoQuinteroC/pv/internal/bd"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Repo struct {
	ventas     *mongo.Collection
	contadores *mongo.Collection
}

func NuevoRepo(db *mongo.Database) *Repo {
	return &Repo{ventas: db.Collection("ventas"), contadores: db.Collection("contadores")}
}

// Registrar guarda la venta con su folio. Es idempotente: si el mismo token ya se
// guardó (un doble clic, un reintento de red), devuelve la venta que ya existía.
func (r *Repo) Registrar(ctx context.Context, v *Venta) error {
	folio, err := r.siguienteFolio(ctx)
	if err != nil {
		return err
	}

	v.ID = bson.NewObjectID()
	v.Folio = folio
	v.Fecha = time.Now()

	_, err = r.ventas.InsertOne(ctx, v)
	if mongo.IsDuplicateKeyError(err) {
		return r.ventas.FindOne(ctx, bson.M{"token": v.Token}).Decode(v)
	}
	return err
}

// siguienteFolio es atómico: Mongo incrementa y devuelve el valor en una sola operación.
// Si después falla el guardado de la venta, ese folio queda sin usar.
func (r *Repo) siguienteFolio(ctx context.Context) (int64, error) {
	var contador struct {
		Valor int64 `bson:"valor"`
	}

	err := r.contadores.FindOneAndUpdate(ctx,
		bson.M{"_id": "folio"},
		bson.M{"$inc": bson.M{"valor": 1}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&contador)

	return contador.Valor, err
}

func (r *Repo) Obtener(ctx context.Context, id bson.ObjectID) (Venta, error) {
	var v Venta
	err := r.ventas.FindOne(ctx, bson.M{"_id": id}).Decode(&v)
	return v, err
}

func (r *Repo) DeTurno(ctx context.Context, turno bson.ObjectID) ([]Venta, error) {
	return bd.Todos[Venta](ctx, r.ventas, bson.M{"turno": turno}, bson.M{"folio": 1})
}

// Entre devuelve las ventas de un rango de fechas: desde incluido, hasta excluido.
func (r *Repo) Entre(ctx context.Context, desde, hasta time.Time) ([]Venta, error) {
	filtro := bson.M{"fecha": bson.M{"$gte": desde, "$lt": hasta}}
	return bd.Todos[Venta](ctx, r.ventas, filtro, bson.M{"folio": -1})
}

// Cancelar solo funciona si la venta sigue activa y su turno sigue abierto.
// El filtro y el cambio van en una sola operación, así dos cancelaciones
// simultáneas no se pisan: solo una encuentra la venta activa.
func (r *Repo) Cancelar(ctx context.Context, id, turnoAbierto bson.ObjectID, motivo string) error {
	if motivo == "" {
		return ErrMotivo
	}

	resultado, err := r.ventas.UpdateOne(ctx,
		bson.M{"_id": id, "turno": turnoAbierto, "estado": Activa},
		bson.M{"$set": bson.M{"estado": Cancelada, "motivo_cancelacion": motivo}},
	)
	if err != nil {
		return err
	}
	if resultado.MatchedCount == 0 {
		return ErrNoCancelable
	}
	return nil
}

type MasVendido struct {
	Nombre   string `bson:"_id"`
	Cantidad int64  `bson:"cantidad"`
	Importe  int64  `bson:"importe"`
}

// MasVendidos agrupa los renglones de las ventas activas por producto.
func (r *Repo) MasVendidos(ctx context.Context, desde, hasta time.Time) ([]MasVendido, error) {
	etapas := []bson.M{
		{"$match": bson.M{"estado": Activa, "fecha": bson.M{"$gte": desde, "$lt": hasta}}},
		{"$unwind": "$renglones"},
		{"$group": bson.M{
			"_id":      "$renglones.nombre",
			"cantidad": bson.M{"$sum": "$renglones.cantidad"},
			"importe":  bson.M{"$sum": "$renglones.importe"},
		}},
		{"$sort": bson.M{"importe": -1}},
		{"$limit": 10},
	}

	cursor, err := r.ventas.Aggregate(ctx, etapas)
	if err != nil {
		return nil, err
	}

	masVendidos := []MasVendido{}
	err = cursor.All(ctx, &masVendidos)
	return masVendidos, err
}

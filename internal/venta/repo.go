// Archivo: repo.go (venta)
// Qué hace: guarda y consulta las ventas en MongoDB. Se encarga de dar el folio
// consecutivo, de que un doble clic no cobre dos veces, de cancelar de forma
// segura y de armar el reporte de lo más vendido.

package venta

import (
	"context"
	"time"

	"github.com/GustavoQuinteroC/pv/internal/bd"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repo agrupa las dos colecciones que usa: las ventas y los contadores (para el folio).
type Repo struct {
	ventas     *mongo.Collection
	contadores *mongo.Collection
}

// NuevoRepo prepara el acceso a las colecciones "ventas" y "contadores".
func NuevoRepo(db *mongo.Database) *Repo {
	return &Repo{ventas: db.Collection("ventas"), contadores: db.Collection("contadores")}
}

// Registrar guarda la venta con su folio. Es idempotente: si el mismo token ya se
// guardó (un doble clic, un reintento de red), devuelve la venta que ya existía.
func (r *Repo) Registrar(ctx context.Context, v *Venta) error {
	// Pide el siguiente número de ticket.
	folio, err := r.siguienteFolio(ctx)
	if err != nil {
		return err
	}

	// Completa los datos que faltaban: id nuevo, folio y hora actual.
	v.ID = bson.NewObjectID()
	v.Folio = folio
	v.Fecha = time.Now()

	// Intenta guardarla.
	_, err = r.ventas.InsertOne(ctx, v)
	// Si el índice único del token se queja, esa venta ya estaba guardada.
	if mongo.IsDuplicateKeyError(err) {
		// Entonces se lee la que ya existe y se devuelve esa (no se cobra dos veces).
		return r.ventas.FindOne(ctx, bson.M{"token": v.Token}).Decode(v)
	}
	return err
}

// siguienteFolio es atómico: Mongo incrementa y devuelve el valor en una sola operación.
// Si después falla el guardado de la venta, ese folio queda sin usar.
func (r *Repo) siguienteFolio(ctx context.Context) (int64, error) {
	// Ficha mínima para leer el contador.
	var contador struct {
		Valor int64 `bson:"valor"`
	}

	// Busca el contador "folio", le suma 1 y devuelve el valor ya actualizado.
	err := r.contadores.FindOneAndUpdate(ctx,
		// Qué documento buscar.
		bson.M{"_id": "folio"},
		// Qué cambio hacer: sumar 1 al campo "valor".
		bson.M{"$inc": bson.M{"valor": 1}},
		// Si no existe, se crea (upsert); y se devuelve el documento DESPUÉS del cambio.
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&contador)

	return contador.Valor, err
}

// Obtener busca una venta por su id.
func (r *Repo) Obtener(ctx context.Context, id bson.ObjectID) (Venta, error) {
	var v Venta
	err := r.ventas.FindOne(ctx, bson.M{"_id": id}).Decode(&v)
	return v, err
}

// DeTurno devuelve las ventas de un turno, ordenadas por folio.
func (r *Repo) DeTurno(ctx context.Context, turno bson.ObjectID) ([]Venta, error) {
	return bd.Todos[Venta](ctx, r.ventas, bson.M{"turno": turno}, bson.M{"folio": 1})
}

// Entre devuelve las ventas de un rango de fechas: desde incluido, hasta excluido.
func (r *Repo) Entre(ctx context.Context, desde, hasta time.Time) ([]Venta, error) {
	// $gte = "mayor o igual"; $lt = "menor que".
	filtro := bson.M{"fecha": bson.M{"$gte": desde, "$lt": hasta}}
	// Las más recientes primero (folio descendente).
	return bd.Todos[Venta](ctx, r.ventas, filtro, bson.M{"folio": -1})
}

// Cancelar solo funciona si la venta sigue activa y su turno sigue abierto.
// El filtro y el cambio van en una sola operación, así dos cancelaciones
// simultáneas no se pisan: solo una encuentra la venta activa.
func (r *Repo) Cancelar(ctx context.Context, id, turnoAbierto bson.ObjectID, motivo string) error {
	// Sin motivo no se cancela.
	if motivo == "" {
		return ErrMotivo
	}

	// Busca la venta con ese id, de ese turno y que siga activa; si la encuentra, la marca cancelada.
	resultado, err := r.ventas.UpdateOne(ctx,
		bson.M{"_id": id, "turno": turnoAbierto, "estado": Activa},
		bson.M{"$set": bson.M{"estado": Cancelada, "motivo_cancelacion": motivo}},
	)
	if err != nil {
		return err
	}
	// Si ningún documento coincidió, no había nada que cancelar.
	if resultado.MatchedCount == 0 {
		return ErrNoCancelable
	}
	return nil
}

// MasVendido es una fila del reporte: un producto y cuánto dinero dejó.
type MasVendido struct {
	// Nombre del producto (MongoDB lo llama "_id" en el resultado agrupado).
	Nombre string `bson:"_id"`
	// Dinero total vendido (centavos).
	Importe int64 `bson:"importe"`
}

// MasVendidos agrupa los renglones de las ventas activas por producto.
func (r *Repo) MasVendidos(ctx context.Context, desde, hasta time.Time) ([]MasVendido, error) {
	// Un "pipeline" es una cadena de pasos: cada paso transforma el resultado del anterior.
	etapas := []bson.M{
		// 1) Quedarse solo con ventas activas dentro del rango de fechas.
		{"$match": bson.M{"estado": Activa, "fecha": bson.M{"$gte": desde, "$lt": hasta}}},
		// 2) "Desenrollar": convertir cada renglón de cada venta en un documento aparte.
		{"$unwind": "$renglones"},
		// 3) Agrupar por nombre de producto y sumar sus importes.
		{"$group": bson.M{
			"_id":     "$renglones.nombre",
			"importe": bson.M{"$sum": "$renglones.importe"},
		}},
		// 4) Ordenar del que más dinero dejó al que menos.
		{"$sort": bson.M{"importe": -1}},
		// 5) Quedarse con los 10 primeros.
		{"$limit": 10},
	}

	// Ejecuta la cadena de pasos.
	cursor, err := r.ventas.Aggregate(ctx, etapas)
	if err != nil {
		return nil, err
	}

	// Lista vacía (y no nula) para recibir los resultados.
	masVendidos := []MasVendido{}
	err = cursor.All(ctx, &masVendidos)
	return masVendidos, err
}

// Package negocio guarda los datos que salen impresos en el ticket.
package negocio

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Negocio struct {
	Nombre  string `bson:"nombre"`
	Mensaje string `bson:"mensaje"`
}

type Repo struct {
	ajustes *mongo.Collection
}

func NuevoRepo(db *mongo.Database) *Repo {
	return &Repo{ajustes: db.Collection("ajustes")}
}

// Obtener nunca falla: sin datos guardados, usa unos de ejemplo.
func (r *Repo) Obtener(ctx context.Context) Negocio {
	negocio := Negocio{Nombre: "Mi negocio", Mensaje: "¡Gracias por su compra!"}

	err := r.ajustes.FindOne(ctx, bson.M{"_id": "negocio"}).Decode(&negocio)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return Negocio{Nombre: "Mi negocio"}
	}
	return negocio
}

func (r *Repo) Guardar(ctx context.Context, negocio Negocio) error {
	_, err := r.ajustes.ReplaceOne(ctx,
		bson.M{"_id": "negocio"},
		negocio,
		options.Replace().SetUpsert(true),
	)
	return err
}

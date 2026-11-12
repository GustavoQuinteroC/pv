// Archivo: negocio.go
// Qué hace: guarda los datos del negocio que aparecen en pantalla y en el
// ticket: el nombre y el mensaje de agradecimiento del final.

// Package negocio guarda los datos que salen impresos en el ticket.
package negocio

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Negocio son los dos datos que se pueden cambiar en "Ajustes".
type Negocio struct {
	// Nombre del negocio.
	Nombre string `bson:"nombre"`
	// Mensaje al final del ticket.
	Mensaje string `bson:"mensaje"`
}

// Repo guarda el acceso a la colección "ajustes".
type Repo struct {
	ajustes *mongo.Collection
}

// NuevoRepo prepara el acceso a la colección "ajustes".
func NuevoRepo(db *mongo.Database) *Repo {
	return &Repo{ajustes: db.Collection("ajustes")}
}

// Obtener nunca falla: sin datos guardados, usa unos de ejemplo.
func (r *Repo) Obtener(ctx context.Context) Negocio {
	// Valores de ejemplo por si todavía no se ha guardado nada.
	negocio := Negocio{Nombre: "Mi negocio", Mensaje: "¡Gracias por su compra!"}

	// Intenta leer lo guardado; si lo encuentra, sobrescribe los valores de ejemplo.
	err := r.ajustes.FindOne(ctx, bson.M{"_id": "negocio"}).Decode(&negocio)
	// Que no exista nada guardado es normal; cualquier otro error sí es un problema.
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		// Ante un problema real, se muestra al menos un nombre para que la pantalla no se rompa.
		return Negocio{Nombre: "Mi negocio"}
	}
	return negocio
}

// Guardar reemplaza los datos del negocio (o los crea la primera vez).
func (r *Repo) Guardar(ctx context.Context, negocio Negocio) error {
	_, err := r.ajustes.ReplaceOne(ctx,
		// Siempre es el mismo documento, con id "negocio".
		bson.M{"_id": "negocio"},
		// Lo nuevo que se guarda.
		negocio,
		// Upsert: si no existe, lo crea.
		options.Replace().SetUpsert(true),
	)
	return err
}

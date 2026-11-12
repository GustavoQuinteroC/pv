// Package bdtest da una base de MongoDB vacía y desechable a las pruebas de integración.
package bdtest

import (
	"context"
	"os"
	"testing"

	"github.com/GustavoQuinteroC/pv/internal/bd"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Nueva crea una base con nombre único, con los índices puestos, y la borra al terminar.
// Si MONGO_URI_PRUEBAS no está definida, la prueba se salta.
func Nueva(t *testing.T) *mongo.Database {
	t.Helper()

	uri := os.Getenv("MONGO_URI_PRUEBAS")
	if uri == "" {
		t.Skip("define MONGO_URI_PRUEBAS para correr las pruebas con MongoDB")
	}

	ctx := context.Background()
	db, err := bd.Conectar(ctx, uri, "pv_prueba_"+bson.NewObjectID().Hex())
	if err != nil {
		t.Fatalf("no se pudo conectar a MongoDB: %v", err)
	}
	t.Cleanup(func() { db.Drop(ctx) })

	if err := bd.CrearIndices(ctx, db); err != nil {
		t.Fatalf("no se pudieron crear los índices: %v", err)
	}
	return db
}

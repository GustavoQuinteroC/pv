// Archivo: bdtest.go
// Qué hace: ayuda a las pruebas que necesitan MongoDB de verdad. Crea una base
// de datos nueva y desechable para cada prueba y la borra al terminar, así las
// pruebas no se estorban entre sí ni tocan datos reales.

// Package bdtest da una base de MongoDB vacía y desechable a las pruebas de integración.
package bdtest

import (
	// Para operaciones con tiempo límite y cancelación.
	"context"
	// Para leer variables de entorno.
	"os"
	// La biblioteca de pruebas de Go.
	"testing"

	// Nuestro paquete que conecta con MongoDB y crea índices.
	"github.com/GustavoQuinteroC/pv/internal/bd"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Nueva crea una base con nombre único, con los índices puestos, y la borra al terminar.
// Si MONGO_URI_PRUEBAS no está definida, la prueba se salta.
func Nueva(t *testing.T) *mongo.Database {
	// Marca esta función como "ayudante": si algo falla, el error apunta a la prueba que la llamó.
	t.Helper()

	// Dirección de la MongoDB de pruebas.
	uri := os.Getenv("MONGO_URI_PRUEBAS")
	// Si no está definida, no hay con qué probar...
	if uri == "" {
		// ...así que la prueba se salta (no cuenta como fallo).
		t.Skip("define MONGO_URI_PRUEBAS para correr las pruebas con MongoDB")
	}

	// Contexto vacío: sin tiempo límite.
	ctx := context.Background()
	// El nombre lleva un identificador aleatorio para que cada prueba tenga su propia base.
	db, err := bd.Conectar(ctx, uri, "pv_prueba_"+bson.NewObjectID().Hex())
	if err != nil {
		// Fatalf detiene la prueba con un mensaje de error.
		t.Fatalf("no se pudo conectar a MongoDB: %v", err)
	}
	// Cleanup: al terminar la prueba, borra la base de datos completa.
	t.Cleanup(func() { db.Drop(ctx) })

	// Crea los mismos índices que en producción, para que las reglas también se prueben.
	if err := bd.CrearIndices(ctx, db); err != nil {
		t.Fatalf("no se pudieron crear los índices: %v", err)
	}
	// Entrega la base lista para usar.
	return db
}

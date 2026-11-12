// Archivo: repo_test.go (caja)
// Qué hace: pruebas con MongoDB real y desechable. Comprueban que el sistema
// nunca permita dos turnos abiertos a la vez y que un turno cerrado no se
// pueda cerrar otra vez. Se saltan solas si no hay MongoDB de pruebas.

package caja

import (
	"context"
	"errors"
	"testing"

	"github.com/GustavoQuinteroC/pv/internal/bd/bdtest"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Prueba la regla "solo un turno abierto a la vez".
func TestSoloPuedeHaberUnTurnoAbierto(t *testing.T) {
	repo := NuevoRepo(bdtest.Nueva(t))
	ctx := context.Background()
	// Dos personas inventadas.
	ana, luis := bson.NewObjectID(), bson.NewObjectID()

	// Ana abre turno con $200 de fondo.
	if err := repo.Abrir(ctx, ana, "Ana", 20000); err != nil {
		t.Fatalf("Ana debía poder abrir: %v", err)
	}
	// Luis intenta abrir con el de Ana todavía abierto: debe rechazarse.
	if err := repo.Abrir(ctx, luis, "Luis", 10000); !errors.Is(err, ErrTurnoAbierto) {
		t.Errorf("Luis con el turno de Ana abierto: error = %v, se esperaba ErrTurnoAbierto", err)
	}

	// Recupera el turno abierto de Ana.
	turno, err := repo.TurnoAbierto(ctx)
	if err != nil || turno == nil {
		t.Fatalf("debía haber un turno abierto: %v", err)
	}
	// Ana lo cierra.
	if err := repo.Cerrar(ctx, turno.ID, 20000, 20000); err != nil {
		t.Fatalf("cerrar: %v", err)
	}
	// Cerrarlo una segunda vez debe fallar.
	if err := repo.Cerrar(ctx, turno.ID, 20000, 20000); !errors.Is(err, ErrTurnoCerrado) {
		t.Errorf("cerrar dos veces: error = %v, se esperaba ErrTurnoCerrado", err)
	}

	// Ahora que el de Ana está cerrado, Luis sí puede abrir.
	if err := repo.Abrir(ctx, luis, "Luis", 10000); err != nil {
		t.Errorf("con el turno de Ana cerrado, Luis debía poder abrir: %v", err)
	}
}

// Prueba que, sin turnos, TurnoAbierto devuelva "nada" y no un error.
func TestSinTurnoNoHayTurnoAbierto(t *testing.T) {
	repo := NuevoRepo(bdtest.Nueva(t))

	turno, err := repo.TurnoAbierto(context.Background())
	if err != nil || turno != nil {
		t.Errorf("TurnoAbierto = %v, %v; se esperaba nil, nil", turno, err)
	}
}

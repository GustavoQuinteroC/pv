package caja

import (
	"context"
	"errors"
	"testing"

	"github.com/GustavoQuinteroC/pv/internal/bd/bdtest"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestSoloPuedeHaberUnTurnoAbierto(t *testing.T) {
	repo := NuevoRepo(bdtest.Nueva(t))
	ctx := context.Background()
	ana, luis := bson.NewObjectID(), bson.NewObjectID()

	if err := repo.Abrir(ctx, ana, "Ana", 20000); err != nil {
		t.Fatalf("Ana debía poder abrir: %v", err)
	}
	if err := repo.Abrir(ctx, luis, "Luis", 10000); !errors.Is(err, ErrTurnoAbierto) {
		t.Errorf("Luis con el turno de Ana abierto: error = %v, se esperaba ErrTurnoAbierto", err)
	}

	turno, err := repo.TurnoAbierto(ctx)
	if err != nil || turno == nil {
		t.Fatalf("debía haber un turno abierto: %v", err)
	}
	if err := repo.Cerrar(ctx, turno.ID, 20000, 20000); err != nil {
		t.Fatalf("cerrar: %v", err)
	}
	if err := repo.Cerrar(ctx, turno.ID, 20000, 20000); !errors.Is(err, ErrTurnoCerrado) {
		t.Errorf("cerrar dos veces: error = %v, se esperaba ErrTurnoCerrado", err)
	}

	if err := repo.Abrir(ctx, luis, "Luis", 10000); err != nil {
		t.Errorf("con el turno de Ana cerrado, Luis debía poder abrir: %v", err)
	}
}

func TestSinTurnoNoHayTurnoAbierto(t *testing.T) {
	repo := NuevoRepo(bdtest.Nueva(t))

	turno, err := repo.TurnoAbierto(context.Background())
	if err != nil || turno != nil {
		t.Errorf("TurnoAbierto = %v, %v; se esperaba nil, nil", turno, err)
	}
}

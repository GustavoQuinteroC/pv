package venta

import (
	"context"
	"errors"
	"testing"

	"github.com/GustavoQuinteroC/pv/internal/bd/bdtest"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func ventaDePrueba(token string, turno bson.ObjectID) *Venta {
	return &Venta{Token: token, Turno: turno, Total: 1000, Estado: Activa}
}

func TestRegistrarEsIdempotente(t *testing.T) {
	repo := NuevoRepo(bdtest.Nueva(t))
	ctx := context.Background()
	turno := bson.NewObjectID()

	primera := ventaDePrueba("token-1", turno)
	repetida := ventaDePrueba("token-1", turno) // el mismo doble clic, otra petición

	if err := repo.Registrar(ctx, primera); err != nil {
		t.Fatalf("primer registro: %v", err)
	}
	if err := repo.Registrar(ctx, repetida); err != nil {
		t.Fatalf("registro repetido: %v", err)
	}

	if repetida.ID != primera.ID || repetida.Folio != primera.Folio {
		t.Errorf("el repetido debía devolver la venta original: %v/%d contra %v/%d",
			repetida.ID, repetida.Folio, primera.ID, primera.Folio)
	}

	ventas, err := repo.DeTurno(ctx, turno)
	if err != nil || len(ventas) != 1 {
		t.Errorf("debía haber 1 venta guardada, hay %d (error: %v)", len(ventas), err)
	}
}

func TestLosFoliosSonConsecutivos(t *testing.T) {
	repo := NuevoRepo(bdtest.Nueva(t))
	ctx := context.Background()
	turno := bson.NewObjectID()

	for esperado := int64(1); esperado <= 3; esperado++ {
		v := ventaDePrueba(bson.NewObjectID().Hex(), turno)
		if err := repo.Registrar(ctx, v); err != nil {
			t.Fatal(err)
		}
		if v.Folio != esperado {
			t.Errorf("folio = %d, se esperaba %d", v.Folio, esperado)
		}
	}
}

func TestCancelarSoloUnaVezYSoloEnSuTurno(t *testing.T) {
	repo := NuevoRepo(bdtest.Nueva(t))
	ctx := context.Background()
	turno := bson.NewObjectID()

	v := ventaDePrueba("token-1", turno)
	if err := repo.Registrar(ctx, v); err != nil {
		t.Fatal(err)
	}

	otroTurno := bson.NewObjectID()
	if err := repo.Cancelar(ctx, v.ID, otroTurno, "error"); !errors.Is(err, ErrNoCancelable) {
		t.Errorf("cancelar desde otro turno: error = %v, se esperaba ErrNoCancelable", err)
	}
	if err := repo.Cancelar(ctx, v.ID, turno, ""); !errors.Is(err, ErrMotivo) {
		t.Errorf("cancelar sin motivo: error = %v, se esperaba ErrMotivo", err)
	}
	if err := repo.Cancelar(ctx, v.ID, turno, "se equivocó el cajero"); err != nil {
		t.Errorf("la primera cancelación debía funcionar: %v", err)
	}
	if err := repo.Cancelar(ctx, v.ID, turno, "otra vez"); !errors.Is(err, ErrNoCancelable) {
		t.Errorf("la segunda cancelación: error = %v, se esperaba ErrNoCancelable", err)
	}
}

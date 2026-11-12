// Archivo: repo_test.go (venta)
// Qué hace: pruebas de integración. Usan una MongoDB real y desechable para
// comprobar que el guardado, los folios y la cancelación se comporten como deben.
// Si no hay MongoDB de pruebas disponible, se saltan solas.

package venta

import (
	"context"
	"errors"
	"testing"

	// Nos da una base de datos limpia para cada prueba.
	"github.com/GustavoQuinteroC/pv/internal/bd/bdtest"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ventaDePrueba arma una venta mínima para guardar.
func ventaDePrueba(token string, turno bson.ObjectID) *Venta {
	return &Venta{Token: token, Turno: turno, Total: 1000, Estado: Activa}
}

// Prueba que guardar dos veces la misma venta (mismo token) no la duplique.
func TestRegistrarEsIdempotente(t *testing.T) {
	repo := NuevoRepo(bdtest.Nueva(t))
	ctx := context.Background()
	turno := bson.NewObjectID()

	primera := ventaDePrueba("token-1", turno)
	// El mismo doble clic, otra petición.
	repetida := ventaDePrueba("token-1", turno)

	if err := repo.Registrar(ctx, primera); err != nil {
		t.Fatalf("primer registro: %v", err)
	}
	if err := repo.Registrar(ctx, repetida); err != nil {
		t.Fatalf("registro repetido: %v", err)
	}

	// La repetida debe haber recibido los datos de la original.
	if repetida.ID != primera.ID || repetida.Folio != primera.Folio {
		t.Errorf("el repetido debía devolver la venta original: %v/%d contra %v/%d",
			repetida.ID, repetida.Folio, primera.ID, primera.Folio)
	}

	// Y en la base solo debe existir una.
	ventas, err := repo.DeTurno(ctx, turno)
	if err != nil || len(ventas) != 1 {
		t.Errorf("debía haber 1 venta guardada, hay %d (error: %v)", len(ventas), err)
	}
}

// Prueba que los folios sean 1, 2, 3...
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

// Prueba las reglas de cancelación: solo una vez, solo en su turno, y con motivo.
func TestCancelarSoloUnaVezYSoloEnSuTurno(t *testing.T) {
	repo := NuevoRepo(bdtest.Nueva(t))
	ctx := context.Background()
	turno := bson.NewObjectID()

	v := ventaDePrueba("token-1", turno)
	if err := repo.Registrar(ctx, v); err != nil {
		t.Fatal(err)
	}

	// Desde otro turno no se puede cancelar.
	otroTurno := bson.NewObjectID()
	if err := repo.Cancelar(ctx, v.ID, otroTurno, "error"); !errors.Is(err, ErrNoCancelable) {
		t.Errorf("cancelar desde otro turno: error = %v, se esperaba ErrNoCancelable", err)
	}
	// Sin motivo tampoco.
	if err := repo.Cancelar(ctx, v.ID, turno, ""); !errors.Is(err, ErrMotivo) {
		t.Errorf("cancelar sin motivo: error = %v, se esperaba ErrMotivo", err)
	}
	// Con turno correcto y motivo, sí.
	if err := repo.Cancelar(ctx, v.ID, turno, "se equivocó el cajero"); err != nil {
		t.Errorf("la primera cancelación debía funcionar: %v", err)
	}
	// Una segunda vez ya no, porque ya no está activa.
	if err := repo.Cancelar(ctx, v.ID, turno, "otra vez"); !errors.Is(err, ErrNoCancelable) {
		t.Errorf("la segunda cancelación: error = %v, se esperaba ErrNoCancelable", err)
	}
}

// Archivo: caja.go
// Qué hace: controla el turno del cajero. Abrir turno (con fondo de caja),
// registrar entradas y retiros de efectivo, y cerrar con el "corte": calcula
// cuánto efectivo debería haber y lo compara con lo que la persona contó.

// Package caja controla el turno del cajero: fondo, entradas, retiros y corte.
package caja

import (
	"context"
	"errors"
	"time"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/bd"
	"github.com/GustavoQuinteroC/pv/internal/venta"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Estados del turno y tipos de movimiento de efectivo.
const (
	Abierto = "abierto"
	Cerrado = "cerrado"

	Entrada = "entrada"
	Retiro  = "retiro"
)

// Errores que se le muestran a la persona.
var (
	ErrSinTurno     = aviso.Nuevo("Abre tu turno en la caja antes de vender.")
	ErrTurnoAbierto = aviso.Nuevo("Ya hay un turno abierto en la caja.")
	ErrFondo        = aviso.Nuevo("El fondo no puede ser negativo.")
	ErrMovimiento   = aviso.Nuevo("Elige entrada o retiro, un monto mayor a cero y escribe el motivo.")
	ErrConteo       = aviso.Nuevo("El efectivo contado no puede ser negativo.")
	ErrTurnoCerrado = aviso.Nuevo("Ese turno ya está cerrado.")
)

// Turno es el periodo en que una persona atiende la caja.
type Turno struct {
	// Identificador único.
	ID bson.ObjectID `bson:"_id,omitempty" json:"id"`
	// Quién abrió el turno (id).
	Cajero bson.ObjectID `bson:"cajero" json:"cajero"`
	// Quién abrió el turno (nombre).
	CajeroNombre string `bson:"cajero_nombre" json:"cajero_nombre"`
	// Efectivo con el que empezó la caja (centavos).
	Fondo int64 `bson:"fondo" json:"fondo"`
	// "abierto" o "cerrado".
	Estado string `bson:"estado" json:"estado"`
	// Cuándo se abrió.
	Abierto time.Time `bson:"abierto" json:"abierto"`
	// Cuándo se cerró.
	Cerrado time.Time `bson:"cerrado" json:"cerrado"`
	// Efectivo que la persona contó al cerrar (centavos).
	Contado int64 `bson:"contado" json:"contado"`
	// Efectivo que el sistema calculó que debía haber (centavos).
	Esperado int64 `bson:"esperado" json:"esperado"`
}

// Diferencia es lo que sobra (positivo) o falta (negativo) en el cajón.
func (t Turno) Diferencia() int64 {
	return t.Contado - t.Esperado
}

// Movimiento es una entrada o retiro de efectivo del cajón (no es una venta).
type Movimiento struct {
	// Identificador único.
	ID bson.ObjectID `bson:"_id,omitempty" json:"id"`
	// A qué turno pertenece.
	Turno bson.ObjectID `bson:"turno" json:"turno"`
	// "entrada" o "retiro".
	Tipo string `bson:"tipo" json:"tipo"`
	// Cuánto dinero (centavos).
	Monto int64 `bson:"monto" json:"monto"`
	// Para qué fue.
	Motivo string `bson:"motivo" json:"motivo"`
	// Cuándo se registró.
	Fecha time.Time `bson:"fecha" json:"fecha"`
}

// Esperado es el efectivo que debe haber en el cajón al cerrar el turno:
// el fondo, más lo cobrado en efectivo (sin el cambio), más las entradas, menos los retiros.
func Esperado(fondo int64, ventas []venta.Venta, movimientos []Movimiento) int64 {
	// Empieza con el fondo más el efectivo neto de las ventas (la tarjeta y las transferencias no entran al cajón).
	esperado := fondo + venta.Resumir(ventas).PorForma[venta.Efectivo]
	// Recorre los movimientos de efectivo.
	for _, m := range movimientos {
		if m.Tipo == Entrada {
			// Una entrada suma.
			esperado += m.Monto
		} else {
			// Un retiro resta.
			esperado -= m.Monto
		}
	}
	return esperado
}

// Repo agrupa las colecciones de turnos y movimientos.
type Repo struct {
	turnos      *mongo.Collection
	movimientos *mongo.Collection
}

// NuevoRepo prepara el acceso a las colecciones "turnos" y "movimientos".
func NuevoRepo(db *mongo.Database) *Repo {
	return &Repo{turnos: db.Collection("turnos"), movimientos: db.Collection("movimientos")}
}

// TurnoAbierto devuelve nil, sin error, cuando la caja está cerrada.
func (r *Repo) TurnoAbierto(ctx context.Context) (*Turno, error) {
	var t Turno
	// Busca el turno cuyo estado es "abierto".
	err := r.turnos.FindOne(ctx, bson.M{"estado": Abierto}).Decode(&t)
	// Que no exista no es un error: simplemente la caja está cerrada.
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &t, err
}

// Abrir depende del índice único parcial: si ya hay un turno abierto, Mongo lo rechaza.
func (r *Repo) Abrir(ctx context.Context, cajero bson.ObjectID, nombre string, fondo int64) error {
	// El fondo no puede ser negativo.
	if fondo < 0 {
		return ErrFondo
	}

	// Arma el turno nuevo, abierto desde este momento.
	turno := Turno{Cajero: cajero, CajeroNombre: nombre, Fondo: fondo, Estado: Abierto, Abierto: time.Now()}
	_, err := r.turnos.InsertOne(ctx, turno)
	// Si MongoDB rechaza por duplicado, es que ya había un turno abierto.
	if mongo.IsDuplicateKeyError(err) {
		return ErrTurnoAbierto
	}
	return err
}

// Cerrar guarda el resultado del corte. Solo funciona si el turno sigue abierto.
func (r *Repo) Cerrar(ctx context.Context, id bson.ObjectID, contado, esperado int64) error {
	// Lo contado no puede ser negativo.
	if contado < 0 {
		return ErrConteo
	}

	// Los cambios que se guardan al cerrar.
	cambios := bson.M{"estado": Cerrado, "cerrado": time.Now(), "contado": contado, "esperado": esperado}
	// Busca ese turno SOLO si sigue abierto y lo actualiza, todo en una operación.
	resultado, err := r.turnos.UpdateOne(ctx, bson.M{"_id": id, "estado": Abierto}, bson.M{"$set": cambios})
	if err != nil {
		return err
	}
	// Si no coincidió ningún documento, el turno ya estaba cerrado.
	if resultado.MatchedCount == 0 {
		return ErrTurnoCerrado
	}
	return nil
}

// Obtener busca un turno por su id.
func (r *Repo) Obtener(ctx context.Context, id bson.ObjectID) (Turno, error) {
	var t Turno
	err := r.turnos.FindOne(ctx, bson.M{"_id": id}).Decode(&t)
	return t, err
}

// Listar devuelve todos los turnos, los más recientes primero.
func (r *Repo) Listar(ctx context.Context) ([]Turno, error) {
	return bd.Todos[Turno](ctx, r.turnos, bson.M{}, bson.M{"abierto": -1})
}

// Registrar guarda una entrada o un retiro de efectivo en el turno.
func (r *Repo) Registrar(ctx context.Context, turno bson.ObjectID, tipo string, monto int64, motivo string) error {
	// Debe ser entrada o retiro, con monto positivo y con motivo escrito.
	if (tipo != Entrada && tipo != Retiro) || monto <= 0 || motivo == "" {
		return ErrMovimiento
	}

	movimiento := Movimiento{Turno: turno, Tipo: tipo, Monto: monto, Motivo: motivo, Fecha: time.Now()}
	_, err := r.movimientos.InsertOne(ctx, movimiento)
	return err
}

// Movimientos devuelve las entradas y retiros de un turno, en orden de hora.
func (r *Repo) Movimientos(ctx context.Context, turno bson.ObjectID) ([]Movimiento, error) {
	return bd.Todos[Movimiento](ctx, r.movimientos, bson.M{"turno": turno}, bson.M{"fecha": 1})
}

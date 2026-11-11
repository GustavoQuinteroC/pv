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

const (
	Abierto = "abierto"
	Cerrado = "cerrado"

	Entrada = "entrada"
	Retiro  = "retiro"
)

var (
	ErrSinTurno     = aviso.Nuevo("Abre tu turno en la caja antes de vender.")
	ErrTurnoAbierto = aviso.Nuevo("Ya hay un turno abierto en la caja.")
	ErrFondo        = aviso.Nuevo("El fondo no puede ser negativo.")
	ErrMovimiento   = aviso.Nuevo("Elige entrada o retiro, un monto mayor a cero y escribe el motivo.")
	ErrConteo       = aviso.Nuevo("El efectivo contado no puede ser negativo.")
	ErrTurnoCerrado = aviso.Nuevo("Ese turno ya está cerrado.")
)

type Turno struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	Cajero       bson.ObjectID `bson:"cajero"`
	CajeroNombre string        `bson:"cajero_nombre"`
	Fondo        int64         `bson:"fondo"`
	Estado       string        `bson:"estado"`
	Abierto      time.Time     `bson:"abierto"`
	Cerrado      time.Time     `bson:"cerrado"`
	Contado      int64         `bson:"contado"`
	Esperado     int64         `bson:"esperado"`
}

// Diferencia es lo que sobra (positivo) o falta (negativo) en el cajón.
func (t Turno) Diferencia() int64 {
	return t.Contado - t.Esperado
}

type Movimiento struct {
	ID     bson.ObjectID `bson:"_id,omitempty"`
	Turno  bson.ObjectID `bson:"turno"`
	Tipo   string        `bson:"tipo"`
	Monto  int64         `bson:"monto"`
	Motivo string        `bson:"motivo"`
	Fecha  time.Time     `bson:"fecha"`
}

// Esperado es el efectivo que debe haber en el cajón al cerrar el turno:
// el fondo, más lo cobrado en efectivo (sin el cambio), más las entradas, menos los retiros.
func Esperado(fondo int64, ventas []venta.Venta, movimientos []Movimiento) int64 {
	esperado := fondo + venta.Resumir(ventas).PorForma[venta.Efectivo]
	for _, m := range movimientos {
		if m.Tipo == Entrada {
			esperado += m.Monto
		} else {
			esperado -= m.Monto
		}
	}
	return esperado
}

type Repo struct {
	turnos      *mongo.Collection
	movimientos *mongo.Collection
}

func NuevoRepo(db *mongo.Database) *Repo {
	return &Repo{turnos: db.Collection("turnos"), movimientos: db.Collection("movimientos")}
}

// TurnoAbierto devuelve nil, sin error, cuando la caja está cerrada.
func (r *Repo) TurnoAbierto(ctx context.Context) (*Turno, error) {
	var t Turno
	err := r.turnos.FindOne(ctx, bson.M{"estado": Abierto}).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &t, err
}

// Abrir depende del índice único parcial: si ya hay un turno abierto, Mongo lo rechaza.
func (r *Repo) Abrir(ctx context.Context, cajero bson.ObjectID, nombre string, fondo int64) error {
	if fondo < 0 {
		return ErrFondo
	}

	turno := Turno{Cajero: cajero, CajeroNombre: nombre, Fondo: fondo, Estado: Abierto, Abierto: time.Now()}
	_, err := r.turnos.InsertOne(ctx, turno)
	if mongo.IsDuplicateKeyError(err) {
		return ErrTurnoAbierto
	}
	return err
}

func (r *Repo) Cerrar(ctx context.Context, id bson.ObjectID, contado, esperado int64) error {
	if contado < 0 {
		return ErrConteo
	}

	cambios := bson.M{"estado": Cerrado, "cerrado": time.Now(), "contado": contado, "esperado": esperado}
	resultado, err := r.turnos.UpdateOne(ctx, bson.M{"_id": id, "estado": Abierto}, bson.M{"$set": cambios})
	if err != nil {
		return err
	}
	if resultado.MatchedCount == 0 {
		return ErrTurnoCerrado
	}
	return nil
}

func (r *Repo) Obtener(ctx context.Context, id bson.ObjectID) (Turno, error) {
	var t Turno
	err := r.turnos.FindOne(ctx, bson.M{"_id": id}).Decode(&t)
	return t, err
}

func (r *Repo) Listar(ctx context.Context) ([]Turno, error) {
	return bd.Todos[Turno](ctx, r.turnos, bson.M{}, bson.M{"abierto": -1})
}

func (r *Repo) Registrar(ctx context.Context, turno bson.ObjectID, tipo string, monto int64, motivo string) error {
	if (tipo != Entrada && tipo != Retiro) || monto <= 0 || motivo == "" {
		return ErrMovimiento
	}

	movimiento := Movimiento{Turno: turno, Tipo: tipo, Monto: monto, Motivo: motivo, Fecha: time.Now()}
	_, err := r.movimientos.InsertOne(ctx, movimiento)
	return err
}

func (r *Repo) Movimientos(ctx context.Context, turno bson.ObjectID) ([]Movimiento, error) {
	return bd.Todos[Movimiento](ctx, r.movimientos, bson.M{"turno": turno}, bson.M{"fecha": 1})
}

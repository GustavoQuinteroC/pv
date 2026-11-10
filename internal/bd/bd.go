// Package bd conecta con MongoDB y deja listos los índices que hacen cumplir las reglas.
package bd

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Conectar(ctx context.Context, uri, nombre string) (*mongo.Database, error) {
	cliente, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := cliente.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return cliente.Database(nombre), nil
}

type indice struct {
	coleccion string
	nombre    string
	claves    bson.M
	unico     bool
	filtro    bson.M // si lo trae, solo los documentos que lo cumplen entran al índice
	caduca    bool   // Mongo borra solo cada documento cuando llega la fecha de su campo
}

var indices = []indice{
	{coleccion: "usuarios", nombre: "correo_unico", claves: bson.M{"correo": 1}, unico: true},

	// Solo los productos con código entran al índice, así que el código es opcional.
	{
		coleccion: "productos",
		nombre:    "codigo_unico",
		claves:    bson.M{"codigo": 1},
		unico:     true,
		filtro:    bson.M{"codigo": bson.M{"$exists": true}},
	},

	// Solo los turnos abiertos entran al índice: nunca puede haber dos a la vez.
	{
		coleccion: "turnos",
		nombre:    "un_solo_turno_abierto",
		claves:    bson.M{"estado": 1},
		unico:     true,
		filtro:    bson.M{"estado": "abierto"},
	},

	{coleccion: "ventas", nombre: "token_unico", claves: bson.M{"token": 1}, unico: true},
	{coleccion: "ventas", nombre: "folio_unico", claves: bson.M{"folio": 1}, unico: true},
	{coleccion: "ventas", nombre: "por_turno", claves: bson.M{"turno": 1}},
	{coleccion: "ventas", nombre: "por_fecha", claves: bson.M{"fecha": -1}},

	{coleccion: "sesiones", nombre: "sesion_caduca", claves: bson.M{"expira": 1}, caduca: true},
}

// CrearIndices es seguro de repetir: si el índice ya existe, no hace nada.
func CrearIndices(ctx context.Context, db *mongo.Database) error {
	for _, i := range indices {
		// Cada índice lleva su propio constructor de opciones: el driver lo modifica al usarlo.
		opciones := options.Index().SetName(i.nombre)
		if i.unico {
			opciones.SetUnique(true)
		}
		if i.filtro != nil {
			opciones.SetPartialFilterExpression(i.filtro)
		}
		if i.caduca {
			opciones.SetExpireAfterSeconds(0)
		}

		modelo := mongo.IndexModel{Keys: i.claves, Options: opciones}
		if _, err := db.Collection(i.coleccion).Indexes().CreateOne(ctx, modelo); err != nil {
			return err
		}
	}
	return nil
}

// Todos devuelve todos los documentos que cumplen el filtro, ya ordenados.
func Todos[T any](ctx context.Context, coleccion *mongo.Collection, filtro, orden bson.M) ([]T, error) {
	cursor, err := coleccion.Find(ctx, filtro, options.Find().SetSort(orden))
	if err != nil {
		return nil, err
	}

	documentos := []T{} // vacío y no nulo, para que en JSON salga [] y no null
	err = cursor.All(ctx, &documentos)
	return documentos, err
}

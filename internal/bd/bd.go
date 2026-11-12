// Archivo: bd.go
// Qué hace: conecta el programa con MongoDB (la base de datos) y crea los
// "índices". Un índice es como el índice de un libro: acelera las búsquedas, y
// los índices "únicos" además impiden guardar dos cosas repetidas (dos correos
// iguales, dos turnos abiertos a la vez, etc.).

// Package bd conecta con MongoDB y deja listos los índices que hacen cumplir las reglas.
package bd

import (
	// "context" permite cancelar o poner tiempo límite a las operaciones lentas.
	"context"

	// bson es el formato en que MongoDB guarda los datos (parecido a JSON).
	"go.mongodb.org/mongo-driver/v2/bson"
	// mongo es el driver oficial: el código que sabe hablar con MongoDB.
	"go.mongodb.org/mongo-driver/v2/mongo"
	// options sirve para configurar cada operación (orden, índices, etc.).
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Conectar abre la conexión con MongoDB y devuelve la base de datos lista para usar.
func Conectar(ctx context.Context, uri, nombre string) (*mongo.Database, error) {
	// Crea el cliente con la dirección indicada.
	cliente, err := mongo.Connect(options.Client().ApplyURI(uri))
	// En Go los errores se revisan a mano: si hubo uno, se devuelve de inmediato.
	if err != nil {
		// "nil" significa "nada": no hay base de datos que devolver.
		return nil, err
	}
	// Ping: pregunta "¿estás ahí?" para confirmar que la conexión funciona de verdad.
	if err := cliente.Ping(ctx, nil); err != nil {
		return nil, err
	}
	// Todo bien: se devuelve la base de datos pedida y "nil" como error (sin error).
	return cliente.Database(nombre), nil
}

// indice describe un índice: en qué colección va, cómo se llama y qué reglas tiene.
type indice struct {
	// Colección = "carpeta" de MongoDB donde viven los documentos (ej. "ventas").
	coleccion string
	// Nombre del índice (se pone a propósito para que sea claro y estable).
	nombre string
	// Campo(s) por los que se indexa. 1 = orden ascendente, -1 = descendente.
	claves bson.M
	// Si es true, MongoDB rechaza valores repetidos.
	unico bool
	// Si lo trae, solo los documentos que cumplen el filtro entran al índice.
	filtro bson.M
	// Si es true, MongoDB borra solo cada documento cuando llega la fecha de su campo.
	caduca bool
}

// indices es la lista completa de índices que necesita el sistema.
var indices = []indice{
	// Un correo no puede repetirse entre usuarios.
	{coleccion: "usuarios", nombre: "correo_unico", claves: bson.M{"correo": 1}, unico: true},

	// Solo los productos con código entran al índice, así que el código es opcional.
	{
		// Colección de los productos.
		coleccion: "productos",
		// Nombre del índice.
		nombre: "codigo_unico",
		// Se indexa el campo "codigo".
		claves: bson.M{"codigo": 1},
		// No se permiten códigos repetidos...
		unico: true,
		// ...pero solo entre los productos que sí tienen código.
		filtro: bson.M{"codigo": bson.M{"$exists": true}},
	},

	// Solo los turnos abiertos entran al índice: nunca puede haber dos a la vez.
	{
		// Colección de los turnos.
		coleccion: "turnos",
		// Nombre del índice.
		nombre: "un_solo_turno_abierto",
		// Se indexa el estado del turno.
		claves: bson.M{"estado": 1},
		// Único: no puede haber dos con el mismo valor...
		unico: true,
		// ...y solo cuentan los turnos cuyo estado es "abierto".
		filtro: bson.M{"estado": "abierto"},
	},

	// Cada venta lleva un "token" único: evita cobrar dos veces con un doble clic.
	{coleccion: "ventas", nombre: "token_unico", claves: bson.M{"token": 1}, unico: true},
	// El folio (número del ticket) tampoco se repite.
	{coleccion: "ventas", nombre: "folio_unico", claves: bson.M{"folio": 1}, unico: true},
	// Índice para buscar rápido las ventas de un turno.
	{coleccion: "ventas", nombre: "por_turno", claves: bson.M{"turno": 1}},
	// Índice para buscar rápido por fecha (-1 = de la más nueva a la más vieja).
	{coleccion: "ventas", nombre: "por_fecha", claves: bson.M{"fecha": -1}},

	// Las sesiones se borran solas cuando llega la fecha del campo "expira".
	{coleccion: "sesiones", nombre: "sesion_caduca", claves: bson.M{"expira": 1}, caduca: true},
}

// CrearIndices es seguro de repetir: si el índice ya existe, no hace nada.
func CrearIndices(ctx context.Context, db *mongo.Database) error {
	// "for ... range" recorre la lista. "_" ignora el número de vuelta; "i" es cada índice.
	for _, i := range indices {
		// Cada índice lleva su propio constructor de opciones: el driver lo modifica al usarlo.
		opciones := options.Index().SetName(i.nombre)
		// Si el índice es único, se activa esa regla.
		if i.unico {
			opciones.SetUnique(true)
		}
		// Si trae filtro, se limita a qué documentos aplica.
		if i.filtro != nil {
			opciones.SetPartialFilterExpression(i.filtro)
		}
		// Si debe caducar, MongoDB borra el documento en cuanto llega su fecha (0 segundos de espera).
		if i.caduca {
			opciones.SetExpireAfterSeconds(0)
		}

		// Junta las claves y las opciones en el "modelo" que entiende MongoDB.
		modelo := mongo.IndexModel{Keys: i.claves, Options: opciones}
		// Crea el índice en su colección; si falla, se devuelve el error.
		if _, err := db.Collection(i.coleccion).Indexes().CreateOne(ctx, modelo); err != nil {
			return err
		}
	}
	// Terminó sin problemas: "nil" = sin error.
	return nil
}

// Todos devuelve todos los documentos que cumplen el filtro, ya ordenados.
// La "[T any]" significa que sirve para cualquier tipo de ficha (productos, ventas, etc.).
func Todos[T any](ctx context.Context, coleccion *mongo.Collection, filtro, orden bson.M) ([]T, error) {
	// Find busca los documentos; SetSort indica en qué orden.
	cursor, err := coleccion.Find(ctx, filtro, options.Find().SetSort(orden))
	if err != nil {
		return nil, err
	}

	// Lista vacía (y no nula), para que en JSON salga [] y no null.
	documentos := []T{}
	// All lee todos los resultados y los mete en la lista.
	err = cursor.All(ctx, &documentos)
	// Devuelve la lista y el error (que será nil si todo salió bien).
	return documentos, err
}

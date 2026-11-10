// Package usuario guarda a las personas que usan el sistema y sus sesiones.
package usuario

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"time"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/bd"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

const (
	Admin  = "admin"
	Cajero = "cajero"

	intentosMaximos = 5
	tiempoBloqueo   = 15 * time.Minute
	duracionSesion  = 12 * time.Hour
)

var (
	ErrCredenciales   = aviso.Nuevo("Correo o contraseña incorrectos.")
	ErrBloqueado      = aviso.Nuevo("Demasiados intentos. Espera 15 minutos.")
	ErrDatos          = aviso.Nuevo("Escribe el nombre y un correo válido.")
	ErrClaveCorta     = aviso.Nuevo("La contraseña debe tener al menos 8 caracteres.")
	ErrCorreoRepetido = aviso.Nuevo("Ya existe una persona con ese correo.")
)

type Usuario struct {
	ID             bson.ObjectID `bson:"_id,omitempty"`
	Nombre         string        `bson:"nombre"`
	Correo         string        `bson:"correo"`
	Clave          string        `bson:"clave"` // siempre el hash, nunca la contraseña
	Rol            string        `bson:"rol"`
	Activo         bool          `bson:"activo"`
	Intentos       int           `bson:"intentos"`
	BloqueadoHasta time.Time     `bson:"bloqueado_hasta"`
}

type sesion struct {
	Token   string        `bson:"_id"`
	Usuario bson.ObjectID `bson:"usuario"`
	Expira  time.Time     `bson:"expira"`
}

type Repo struct {
	usuarios *mongo.Collection
	sesiones *mongo.Collection
}

func NuevoRepo(db *mongo.Database) *Repo {
	return &Repo{usuarios: db.Collection("usuarios"), sesiones: db.Collection("sesiones")}
}

func (r *Repo) Crear(ctx context.Context, nombre, correo, clave, rol string) error {
	nombre = strings.TrimSpace(nombre)
	correo = strings.ToLower(strings.TrimSpace(correo))
	if nombre == "" || !strings.Contains(correo, "@") {
		return ErrDatos
	}
	if len(clave) < 8 {
		return ErrClaveCorta
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(clave), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	nuevo := Usuario{Nombre: nombre, Correo: correo, Clave: string(hash), Rol: rol, Activo: true}
	_, err = r.usuarios.InsertOne(ctx, nuevo)
	if mongo.IsDuplicateKeyError(err) {
		return ErrCorreoRepetido
	}
	return err
}

func (r *Repo) Autenticar(ctx context.Context, correo, clave string) (*Usuario, error) {
	filtro := bson.M{"correo": strings.ToLower(strings.TrimSpace(correo)), "activo": true}

	var u Usuario
	err := r.usuarios.FindOne(ctx, filtro).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrCredenciales
	}
	if err != nil {
		return nil, err
	}

	if time.Now().Before(u.BloqueadoHasta) {
		return nil, ErrBloqueado
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Clave), []byte(clave)) != nil {
		return nil, r.registrarFallo(ctx, u)
	}

	_, err = r.usuarios.UpdateByID(ctx, u.ID, bson.M{"$set": bson.M{"intentos": 0}})
	return &u, err
}

// registrarFallo cuenta el intento y bloquea la cuenta al llegar al máximo.
func (r *Repo) registrarFallo(ctx context.Context, u Usuario) error {
	cambios := bson.M{"intentos": u.Intentos + 1}
	if u.Intentos+1 >= intentosMaximos {
		cambios = bson.M{"intentos": 0, "bloqueado_hasta": time.Now().Add(tiempoBloqueo)}
	}

	if _, err := r.usuarios.UpdateByID(ctx, u.ID, bson.M{"$set": cambios}); err != nil {
		return err
	}
	return ErrCredenciales
}

func (r *Repo) IniciarSesion(ctx context.Context, u *Usuario) (string, error) {
	nueva := sesion{Token: rand.Text(), Usuario: u.ID, Expira: time.Now().Add(duracionSesion)}
	_, err := r.sesiones.InsertOne(ctx, nueva)
	return nueva.Token, err
}

func (r *Repo) PorSesion(ctx context.Context, token string) (*Usuario, error) {
	var s sesion
	vigente := bson.M{"_id": token, "expira": bson.M{"$gt": time.Now()}}
	if err := r.sesiones.FindOne(ctx, vigente).Decode(&s); err != nil {
		return nil, err
	}

	var u Usuario
	err := r.usuarios.FindOne(ctx, bson.M{"_id": s.Usuario, "activo": true}).Decode(&u)
	return &u, err
}

func (r *Repo) CerrarSesion(ctx context.Context, token string) error {
	_, err := r.sesiones.DeleteOne(ctx, bson.M{"_id": token})
	return err
}

func (r *Repo) Listar(ctx context.Context) ([]Usuario, error) {
	return bd.Todos[Usuario](ctx, r.usuarios, bson.M{}, bson.M{"nombre": 1})
}

func (r *Repo) CambiarActivo(ctx context.Context, id bson.ObjectID, activo bool) error {
	_, err := r.usuarios.UpdateByID(ctx, id, bson.M{"$set": bson.M{"activo": activo}})
	return err
}

func (r *Repo) Hay(ctx context.Context) (bool, error) {
	cantidad, err := r.usuarios.CountDocuments(ctx, bson.M{})
	return cantidad > 0, err
}

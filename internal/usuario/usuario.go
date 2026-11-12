// Archivo: usuario.go
// Qué hace: maneja a las personas que usan el sistema. Crea cuentas guardando la
// contraseña "revuelta" (hash, nunca en texto claro), verifica el acceso con
// bloqueo tras varios intentos fallidos, y administra las sesiones (el "ya
// entraste" que se recuerda en una cookie del navegador).

// Package usuario guarda a las personas que usan el sistema y sus sesiones.
package usuario

import (
	// Para cancelar o limitar operaciones.
	"context"
	// Genera textos aleatorios seguros (sirven como identificador de sesión).
	"crypto/rand"
	// Para trabajar con errores.
	"errors"
	// Para limpiar y comparar textos.
	"strings"
	// Para fechas y duraciones.
	"time"

	// Nuestro tipo de errores que sí se le muestran a la persona.
	"github.com/GustavoQuinteroC/pv/internal/aviso"
	// Ayudante para leer muchos documentos de MongoDB.
	"github.com/GustavoQuinteroC/pv/internal/bd"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	// Biblioteca que "revuelve" contraseñas de forma segura.
	"golang.org/x/crypto/bcrypt"
)

// Constantes: valores fijos que nunca cambian mientras el programa corre.
const (
	// Los dos roles posibles.
	Admin  = "admin"
	Cajero = "cajero"

	// Cuántas contraseñas incorrectas seguidas se permiten.
	intentosMaximos = 5
	// Cuánto dura el bloqueo al llegar al máximo.
	tiempoBloqueo = 15 * time.Minute
	// Cuánto dura una sesión abierta.
	duracionSesion = 12 * time.Hour
)

// Errores que sí se le muestran a la persona (por eso se crean con aviso.Nuevo).
var (
	ErrCredenciales   = aviso.Nuevo("Correo o contraseña incorrectos.")
	ErrBloqueado      = aviso.Nuevo("Demasiados intentos. Espera 15 minutos.")
	ErrDatos          = aviso.Nuevo("Escribe el nombre y un correo válido.")
	ErrClaveCorta     = aviso.Nuevo("La contraseña debe tener al menos 8 caracteres.")
	ErrCorreoRepetido = aviso.Nuevo("Ya existe una persona con ese correo.")
)

// Usuario es la ficha de una persona. Las etiquetas `bson:"..."` dicen cómo se llama
// cada campo dentro de MongoDB.
type Usuario struct {
	// Identificador único que MongoDB asigna a cada documento ("omitempty": si está vacío, MongoDB lo genera).
	ID bson.ObjectID `bson:"_id,omitempty"`
	// Nombre para mostrar.
	Nombre string `bson:"nombre"`
	// Correo (en minúsculas) con el que entra.
	Correo string `bson:"correo"`
	// Siempre el hash, nunca la contraseña.
	Clave string `bson:"clave"`
	// "admin" o "cajero".
	Rol string `bson:"rol"`
	// false = cuenta desactivada, ya no puede entrar.
	Activo bool `bson:"activo"`
	// Contraseñas incorrectas seguidas.
	Intentos int `bson:"intentos"`
	// Hasta cuándo está bloqueada la cuenta.
	BloqueadoHasta time.Time `bson:"bloqueado_hasta"`
}

// sesion es el registro de "esta persona ya entró". Su identificador es el propio token.
type sesion struct {
	// Texto aleatorio que viaja en la cookie del navegador.
	Token string `bson:"_id"`
	// A qué persona pertenece.
	Usuario bson.ObjectID `bson:"usuario"`
	// Cuándo deja de valer (MongoDB la borra sola).
	Expira time.Time `bson:"expira"`
}

// Repo agrupa las dos colecciones que usa este paquete.
type Repo struct {
	// Colección de usuarios.
	usuarios *mongo.Collection
	// Colección de sesiones.
	sesiones *mongo.Collection
}

// NuevoRepo prepara el acceso a las colecciones "usuarios" y "sesiones".
func NuevoRepo(db *mongo.Database) *Repo {
	return &Repo{usuarios: db.Collection("usuarios"), sesiones: db.Collection("sesiones")}
}

// Crear da de alta a una persona nueva.
func (r *Repo) Crear(ctx context.Context, nombre, correo, clave, rol string) error {
	// Quita espacios sobrantes del nombre.
	nombre = strings.TrimSpace(nombre)
	// El correo se guarda siempre en minúsculas y sin espacios.
	correo = strings.ToLower(strings.TrimSpace(correo))
	// Valida: nombre obligatorio y un correo que al menos tenga arroba.
	if nombre == "" || !strings.Contains(correo, "@") {
		return ErrDatos
	}
	// La contraseña debe tener mínimo 8 caracteres.
	if len(clave) < 8 {
		return ErrClaveCorta
	}

	// "Revuelve" la contraseña: del hash no se puede recuperar el original.
	hash, err := bcrypt.GenerateFromPassword([]byte(clave), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Arma la ficha de la persona nueva (activa desde el inicio).
	nuevo := Usuario{Nombre: nombre, Correo: correo, Clave: string(hash), Rol: rol, Activo: true}
	// La guarda en MongoDB.
	_, err = r.usuarios.InsertOne(ctx, nuevo)
	// Si el índice único del correo se queja, es que ya existía.
	if mongo.IsDuplicateKeyError(err) {
		return ErrCorreoRepetido
	}
	return err
}

// Autenticar revisa correo y contraseña. Devuelve a la persona si todo está bien.
func (r *Repo) Autenticar(ctx context.Context, correo, clave string) (*Usuario, error) {
	// Busca por correo y que la cuenta esté activa.
	filtro := bson.M{"correo": strings.ToLower(strings.TrimSpace(correo)), "activo": true}

	// Variable donde se guardará lo que se encuentre.
	var u Usuario
	// FindOne busca un documento; Decode lo copia a la variable.
	err := r.usuarios.FindOne(ctx, filtro).Decode(&u)
	// No hay nadie con ese correo: mismo mensaje que si la contraseña fuera mala (no se revela cuál falló).
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrCredenciales
	}
	// Cualquier otro error es interno.
	if err != nil {
		return nil, err
	}

	// ¿Sigue bloqueada la cuenta por intentos fallidos?
	if time.Now().Before(u.BloqueadoHasta) {
		return nil, ErrBloqueado
	}
	// Compara la contraseña escrita contra el hash guardado.
	if bcrypt.CompareHashAndPassword([]byte(u.Clave), []byte(clave)) != nil {
		// Contraseña mala: se cuenta el intento fallido.
		return nil, r.registrarFallo(ctx, u)
	}

	// Contraseña buena: se reinicia el contador de intentos.
	_, err = r.usuarios.UpdateByID(ctx, u.ID, bson.M{"$set": bson.M{"intentos": 0}})
	return &u, err
}

// registrarFallo cuenta el intento y bloquea la cuenta al llegar al máximo.
func (r *Repo) registrarFallo(ctx context.Context, u Usuario) error {
	// Por omisión solo suma uno al contador.
	cambios := bson.M{"intentos": u.Intentos + 1}
	// Si con este fallo se llega al máximo...
	if u.Intentos+1 >= intentosMaximos {
		// ...se reinicia el contador y se bloquea la cuenta por un rato.
		cambios = bson.M{"intentos": 0, "bloqueado_hasta": time.Now().Add(tiempoBloqueo)}
	}

	// Guarda el cambio en MongoDB.
	if _, err := r.usuarios.UpdateByID(ctx, u.ID, bson.M{"$set": cambios}); err != nil {
		return err
	}
	// Siempre se responde lo mismo: credenciales incorrectas.
	return ErrCredenciales
}

// IniciarSesion crea una sesión nueva y devuelve su token (para guardarlo en la cookie).
func (r *Repo) IniciarSesion(ctx context.Context, u *Usuario) (string, error) {
	// rand.Text() genera un texto aleatorio imposible de adivinar.
	nueva := sesion{Token: rand.Text(), Usuario: u.ID, Expira: time.Now().Add(duracionSesion)}
	// Guarda la sesión.
	_, err := r.sesiones.InsertOne(ctx, nueva)
	// Devuelve el token (y el error, si lo hubo).
	return nueva.Token, err
}

// PorSesion busca quién es la persona dueña de un token. Falla si la sesión no existe o ya expiró.
func (r *Repo) PorSesion(ctx context.Context, token string) (*Usuario, error) {
	// Variable para la sesión encontrada.
	var s sesion
	// La sesión debe existir y su fecha de expiración debe ser futura ($gt = "mayor que").
	vigente := bson.M{"_id": token, "expira": bson.M{"$gt": time.Now()}}
	if err := r.sesiones.FindOne(ctx, vigente).Decode(&s); err != nil {
		return nil, err
	}

	// Con la sesión en mano, busca a la persona (que siga activa).
	var u Usuario
	err := r.usuarios.FindOne(ctx, bson.M{"_id": s.Usuario, "activo": true}).Decode(&u)
	return &u, err
}

// CerrarSesion borra la sesión (es el "Salir").
func (r *Repo) CerrarSesion(ctx context.Context, token string) error {
	_, err := r.sesiones.DeleteOne(ctx, bson.M{"_id": token})
	return err
}

// Listar devuelve todas las personas ordenadas por nombre.
func (r *Repo) Listar(ctx context.Context) ([]Usuario, error) {
	// {} = sin filtro (todas); {"nombre": 1} = orden alfabético.
	return bd.Todos[Usuario](ctx, r.usuarios, bson.M{}, bson.M{"nombre": 1})
}

// CambiarActivo activa o desactiva una cuenta.
func (r *Repo) CambiarActivo(ctx context.Context, id bson.ObjectID, activo bool) error {
	// "$set" cambia solo ese campo y deja el resto igual.
	_, err := r.usuarios.UpdateByID(ctx, id, bson.M{"$set": bson.M{"activo": activo}})
	return err
}

// Hay dice si ya existe al menos una persona registrada.
func (r *Repo) Hay(ctx context.Context) (bool, error) {
	// Cuenta los documentos de la colección.
	cantidad, err := r.usuarios.CountDocuments(ctx, bson.M{})
	// "cantidad > 0" es true o false.
	return cantidad > 0, err
}

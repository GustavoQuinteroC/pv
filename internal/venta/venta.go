// Package venta arma, cobra y guarda las ventas del mostrador.
//
// Todo el dinero va en centavos y las cantidades en milésimas (1.250 kg = 1250),
// así nunca se suman decimales con error de redondeo.
package venta

import (
	"slices"
	"time"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/producto"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	Activa    = "activa"
	Cancelada = "cancelada"
)

const (
	Efectivo      = "efectivo"
	Tarjeta       = "tarjeta"
	Transferencia = "transferencia"
)

var Formas = []string{Efectivo, Tarjeta, Transferencia}

var (
	ErrToken            = aviso.Nuevo("La venta no trae identificador. Recarga la pantalla de caja.")
	ErrVacia            = aviso.Nuevo("Agrega al menos un producto.")
	ErrProducto         = aviso.Nuevo("Un producto de la venta ya no existe o está inactivo.")
	ErrCantidad         = aviso.Nuevo("Revisa las cantidades: las piezas van enteras y todo debe ser mayor a cero.")
	ErrDescuento        = aviso.Nuevo("El descuento no puede ser negativo ni cubrir toda la venta.")
	ErrFormaPago        = aviso.Nuevo("Revisa los pagos: cada uno necesita una forma válida y un monto mayor a cero.")
	ErrPagoInsuficiente = aviso.Nuevo("El pago no alcanza para cubrir el total.")
	ErrCambio           = aviso.Nuevo("Solo se puede dar cambio si hay pago en efectivo suficiente.")
	ErrNoCancelable     = aviso.Nuevo("Solo se cancela una venta activa del turno que sigue abierto.")
	ErrMotivo           = aviso.Nuevo("Escribe el motivo de la cancelación.")
)

// Renglon es una línea de la venta con el nombre y el precio que tenían al venderse:
// cambiar el catálogo después no altera las ventas pasadas.
type Renglon struct {
	Producto bson.ObjectID `bson:"producto"`
	Nombre   string        `bson:"nombre"`
	Unidad   string        `bson:"unidad"`
	Precio   int64         `bson:"precio"`
	Cantidad int64         `bson:"cantidad"`
	Importe  int64         `bson:"importe"`
}

type Pago struct {
	Forma string `bson:"forma" json:"forma"`
	Monto int64  `bson:"monto" json:"monto"`
}

type Venta struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	Folio        int64         `bson:"folio"`
	Token        string        `bson:"token"`
	Turno        bson.ObjectID `bson:"turno"`
	Cajero       bson.ObjectID `bson:"cajero"`
	CajeroNombre string        `bson:"cajero_nombre"`
	Renglones    []Renglon     `bson:"renglones"`
	Descuento    int64         `bson:"descuento"`
	Total        int64         `bson:"total"`
	Pagos        []Pago        `bson:"pagos"`
	Cambio       int64         `bson:"cambio"`
	Estado       string        `bson:"estado"`
	Motivo       string        `bson:"motivo_cancelacion,omitempty"`
	Fecha        time.Time     `bson:"fecha"`
}

// Linea es lo que manda la pantalla de caja por cada producto.
// Si trae Monto ("$20 de tortilla"), la cantidad se calcula a partir de él.
type Linea struct {
	Producto string `json:"producto"`
	Cantidad int64  `json:"cantidad"`
	Monto    int64  `json:"monto"`
}

// Pedido es la venta completa tal como la manda la pantalla de caja.
// Los precios no viajan: el servidor siempre usa los del catálogo.
type Pedido struct {
	Token     string  `json:"token"`
	Renglones []Linea `json:"renglones"`
	Descuento int64   `json:"descuento"`
	Pagos     []Pago  `json:"pagos"`
}

func ImporteRenglon(precio, cantidad int64) int64 {
	return (precio*cantidad + 500) / 1000
}

func CantidadPorMonto(precio, monto int64) int64 {
	return (monto*1000 + precio/2) / precio
}

// Nueva valida el pedido contra el catálogo y devuelve la venta lista para guardar.
// Quien la llama completa el turno y el cajero.
func Nueva(pedido Pedido, catalogo map[bson.ObjectID]producto.Producto) (*Venta, error) {
	if pedido.Token == "" {
		return nil, ErrToken
	}
	if len(pedido.Renglones) == 0 {
		return nil, ErrVacia
	}

	renglones := make([]Renglon, 0, len(pedido.Renglones))
	var subtotal int64
	for _, linea := range pedido.Renglones {
		renglon, err := nuevoRenglon(linea, catalogo)
		if err != nil {
			return nil, err
		}
		renglones = append(renglones, renglon)
		subtotal += renglon.Importe
	}

	if pedido.Descuento < 0 || pedido.Descuento >= subtotal {
		return nil, ErrDescuento
	}
	total := subtotal - pedido.Descuento

	cambio, err := calcularCambio(total, pedido.Pagos)
	if err != nil {
		return nil, err
	}

	return &Venta{
		Token:     pedido.Token,
		Renglones: renglones,
		Descuento: pedido.Descuento,
		Total:     total,
		Pagos:     pedido.Pagos,
		Cambio:    cambio,
		Estado:    Activa,
	}, nil
}

func nuevoRenglon(linea Linea, catalogo map[bson.ObjectID]producto.Producto) (Renglon, error) {
	id, err := bson.ObjectIDFromHex(linea.Producto)
	p, existe := catalogo[id]
	if err != nil || !existe {
		return Renglon{}, ErrProducto
	}

	renglon := Renglon{
		Producto: p.ID,
		Nombre:   p.Nombre,
		Unidad:   p.Unidad,
		Precio:   p.Precio,
		Cantidad: linea.Cantidad,
	}

	if linea.Monto > 0 && p.Granel() {
		renglon.Cantidad = CantidadPorMonto(p.Precio, linea.Monto)
		renglon.Importe = linea.Monto
	} else {
		renglon.Importe = ImporteRenglon(p.Precio, renglon.Cantidad)
	}

	piezaPartida := !p.Granel() && renglon.Cantidad%1000 != 0
	if renglon.Cantidad <= 0 || piezaPartida {
		return Renglon{}, ErrCantidad
	}
	return renglon, nil
}

// calcularCambio revisa los pagos y devuelve lo que se regresa al cliente.
// El cambio solo sale del efectivo: no se devuelve en tarjeta ni transferencia.
func calcularCambio(total int64, pagos []Pago) (int64, error) {
	var pagado, efectivo int64
	for _, pago := range pagos {
		if pago.Monto <= 0 || !slices.Contains(Formas, pago.Forma) {
			return 0, ErrFormaPago
		}
		pagado += pago.Monto
		if pago.Forma == Efectivo {
			efectivo += pago.Monto
		}
	}

	if pagado < total {
		return 0, ErrPagoInsuficiente
	}
	cambio := pagado - total
	if cambio > efectivo {
		return 0, ErrCambio
	}
	return cambio, nil
}

type Resumen struct {
	Cantidad int
	Total    int64
	PorForma map[string]int64
}

// Resumir suma las ventas activas. Lo cobrado en efectivo ya va sin el cambio.
func Resumir(ventas []Venta) Resumen {
	resumen := Resumen{PorForma: map[string]int64{}}
	for _, v := range ventas {
		if v.Estado != Activa {
			continue
		}
		resumen.Cantidad++
		resumen.Total += v.Total
		for _, pago := range v.Pagos {
			resumen.PorForma[pago.Forma] += pago.Monto
		}
		resumen.PorForma[Efectivo] -= v.Cambio
	}
	return resumen
}

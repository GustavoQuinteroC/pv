// Archivo: venta.go
// Qué hace: es el corazón de las cuentas. Define cómo es una venta, valida el
// pedido que manda la pantalla de caja, calcula importes, total y cambio, y
// resume las ventas. No habla con la base de datos (eso está en repo.go), por
// eso es fácil de probar.
//
// Regla clave: todo el dinero va en CENTAVOS ($18.50 = 1850) y las cantidades en
// MILÉSIMAS (1.250 kg = 1250). Así nunca se suman decimales con error de redondeo.

// Package venta arma, cobra y guarda las ventas del mostrador.
//
// Todo el dinero va en centavos y las cantidades en milésimas (1.250 kg = 1250),
// así nunca se suman decimales con error de redondeo.
package venta

import (
	// Funciones para trabajar con listas.
	"slices"
	// Para fechas.
	"time"

	"github.com/GustavoQuinteroC/pv/internal/aviso"
	"github.com/GustavoQuinteroC/pv/internal/producto"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Los dos estados posibles de una venta.
const (
	Activa    = "activa"
	Cancelada = "cancelada"
)

// Las tres formas de pago.
const (
	Efectivo      = "efectivo"
	Tarjeta       = "tarjeta"
	Transferencia = "transferencia"
)

// Formas es la lista de formas de pago válidas.
var Formas = []string{Efectivo, Tarjeta, Transferencia}

// Errores que se le muestran a la persona al cobrar o cancelar.
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
	// Qué producto era (su id).
	Producto bson.ObjectID `bson:"producto"`
	// Nombre que tenía en ese momento.
	Nombre string `bson:"nombre"`
	// Unidad en que se vendió.
	Unidad string `bson:"unidad"`
	// Precio por unidad en ese momento (centavos).
	Precio int64 `bson:"precio"`
	// Cantidad vendida en milésimas.
	Cantidad int64 `bson:"cantidad"`
	// Lo que costó esta línea (centavos).
	Importe int64 `bson:"importe"`
}

// Pago es una parte de lo que pagó el cliente (puede pagar con varias formas a la vez).
type Pago struct {
	// "efectivo", "tarjeta" o "transferencia".
	Forma string `bson:"forma" json:"forma"`
	// Cuánto pagó con esa forma (centavos).
	Monto int64 `bson:"monto" json:"monto"`
}

// Venta es el ticket completo: se guarda como un solo documento, con renglones y pagos adentro.
type Venta struct {
	// Identificador único.
	ID bson.ObjectID `bson:"_id,omitempty"`
	// Número de ticket (1, 2, 3...).
	Folio int64 `bson:"folio"`
	// Identificador de esta venta, para que un doble clic no cobre dos veces.
	Token string `bson:"token"`
	// A qué turno pertenece.
	Turno bson.ObjectID `bson:"turno"`
	// Quién cobró (id).
	Cajero bson.ObjectID `bson:"cajero"`
	// Quién cobró (nombre, guardado tal cual en ese momento).
	CajeroNombre string `bson:"cajero_nombre"`
	// Las líneas de la venta.
	Renglones []Renglon `bson:"renglones"`
	// Descuento aplicado (centavos).
	Descuento int64 `bson:"descuento"`
	// Total a pagar (centavos).
	Total int64 `bson:"total"`
	// Cómo pagó el cliente.
	Pagos []Pago `bson:"pagos"`
	// Cambio que se le regresó (centavos).
	Cambio int64 `bson:"cambio"`
	// "activa" o "cancelada".
	Estado string `bson:"estado"`
	// Por qué se canceló (solo si se canceló).
	Motivo string `bson:"motivo_cancelacion,omitempty"`
	// Cuándo se hizo.
	Fecha time.Time `bson:"fecha"`
}

// Linea es lo que manda la pantalla de caja por cada producto.
// Si trae Monto ("$20 de tortilla"), la cantidad se calcula a partir de él.
type Linea struct {
	// Id del producto (en texto).
	Producto string `json:"producto"`
	// Cantidad en milésimas.
	Cantidad int64 `json:"cantidad"`
	// Monto en centavos, si se pidió "por dinero".
	Monto int64 `json:"monto"`
}

// Pedido es la venta completa tal como la manda la pantalla de caja.
// Los precios no viajan: el servidor siempre usa los del catálogo.
type Pedido struct {
	// Identificador de la venta (para no cobrar dos veces).
	Token string `json:"token"`
	// Productos y cantidades.
	Renglones []Linea `json:"renglones"`
	// Descuento en centavos.
	Descuento int64 `json:"descuento"`
	// Formas de pago.
	Pagos []Pago `json:"pagos"`
}

// ImporteRenglon calcula precio × cantidad redondeado al centavo.
// Ejemplo: $24.00 el kilo × 1.250 kg = 3000 centavos. El "+ 500" es para redondear hacia arriba desde la mitad.
func ImporteRenglon(precio, cantidad int64) int64 {
	return (precio*cantidad + 500) / 1000
}

// CantidadPorMonto es lo contrario: cuántas milésimas se llevan con cierto dinero.
// Ejemplo: $20 de algo que cuesta $24 el kilo = 833 milésimas (0.833 kg).
func CantidadPorMonto(precio, monto int64) int64 {
	return (monto*1000 + precio/2) / precio
}

// Nueva valida el pedido contra el catálogo y devuelve la venta lista para guardar.
// Quien la llama completa el turno y el cajero.
func Nueva(pedido Pedido, catalogo map[bson.ObjectID]producto.Producto) (*Venta, error) {
	// Sin identificador no se puede proteger contra el doble cobro.
	if pedido.Token == "" {
		return nil, ErrToken
	}
	// Una venta sin productos no tiene sentido.
	if len(pedido.Renglones) == 0 {
		return nil, ErrVacia
	}

	// Lista donde se irán armando los renglones (con espacio reservado).
	renglones := make([]Renglon, 0, len(pedido.Renglones))
	// Suma de los importes de todos los renglones.
	var subtotal int64
	// Recorre cada línea que mandó la pantalla.
	for _, linea := range pedido.Renglones {
		// Convierte la línea en un renglón con datos reales del catálogo.
		renglon, err := nuevoRenglon(linea, catalogo)
		if err != nil {
			return nil, err
		}
		// Lo agrega a la lista.
		renglones = append(renglones, renglon)
		// Y suma su importe al subtotal.
		subtotal += renglon.Importe
	}

	// El descuento no puede ser negativo ni igual o mayor que lo que se vende.
	if pedido.Descuento < 0 || pedido.Descuento >= subtotal {
		return nil, ErrDescuento
	}
	// Total = subtotal menos descuento.
	total := subtotal - pedido.Descuento

	// Revisa los pagos y calcula el cambio.
	cambio, err := calcularCambio(total, pedido.Pagos)
	if err != nil {
		return nil, err
	}

	// Todo en orden: devuelve la venta armada (sin id, folio, turno ni fecha: eso lo completan otros).
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

// nuevoRenglon convierte una línea del pedido en un renglón, usando el catálogo.
func nuevoRenglon(linea Linea, catalogo map[bson.ObjectID]producto.Producto) (Renglon, error) {
	// Convierte el id de texto a su tipo real.
	id, err := bson.ObjectIDFromHex(linea.Producto)
	// Busca el producto en el catálogo; "existe" dice si lo encontró.
	p, existe := catalogo[id]
	// Si el id era inválido o el producto no está (o está inactivo), se rechaza.
	if err != nil || !existe {
		return Renglon{}, ErrProducto
	}

	// Copia los datos del producto tal como están ahora.
	renglon := Renglon{
		Producto: p.ID,
		Nombre:   p.Nombre,
		Unidad:   p.Unidad,
		Precio:   p.Precio,
		Cantidad: linea.Cantidad,
	}

	// Si pidieron "por monto" y el producto es a granel, se calcula la cantidad a partir del dinero.
	if linea.Monto > 0 && p.Granel() {
		renglon.Cantidad = CantidadPorMonto(p.Precio, linea.Monto)
		// El importe es exactamente el dinero pedido.
		renglon.Importe = linea.Monto
	} else {
		// Caso normal: el importe sale de precio × cantidad.
		renglon.Importe = ImporteRenglon(p.Precio, renglon.Cantidad)
	}

	// Una pieza no puede venderse partida (por ejemplo 1.5 refrescos).
	piezaPartida := !p.Granel() && renglon.Cantidad%1000 != 0
	// La cantidad debe ser positiva y, si es pieza, entera.
	if renglon.Cantidad <= 0 || piezaPartida {
		return Renglon{}, ErrCantidad
	}
	return renglon, nil
}

// calcularCambio revisa los pagos y devuelve lo que se regresa al cliente.
// El cambio solo sale del efectivo: no se devuelve en tarjeta ni transferencia.
func calcularCambio(total int64, pagos []Pago) (int64, error) {
	// "pagado" = suma de todos los pagos; "efectivo" = solo lo pagado en efectivo.
	var pagado, efectivo int64
	// Revisa cada pago.
	for _, pago := range pagos {
		// Cada pago necesita monto positivo y una forma válida.
		if pago.Monto <= 0 || !slices.Contains(Formas, pago.Forma) {
			return 0, ErrFormaPago
		}
		// Suma al total pagado.
		pagado += pago.Monto
		// Si fue efectivo, también se suma aparte.
		if pago.Forma == Efectivo {
			efectivo += pago.Monto
		}
	}

	// Si no alcanza para cubrir el total, se rechaza.
	if pagado < total {
		return 0, ErrPagoInsuficiente
	}
	// Lo que sobra es el cambio.
	cambio := pagado - total
	// Pero solo se puede regresar si hubo suficiente efectivo.
	if cambio > efectivo {
		return 0, ErrCambio
	}
	return cambio, nil
}

// Resumen son las cifras de un grupo de ventas.
type Resumen struct {
	// Cuántas ventas activas hay.
	Cantidad int
	// Suma de sus totales (centavos).
	Total int64
	// Cuánto entró por cada forma de pago (centavos).
	PorForma map[string]int64
}

// Resumir suma las ventas activas. Lo cobrado en efectivo ya va sin el cambio.
func Resumir(ventas []Venta) Resumen {
	// Empieza en ceros, con el diccionario de formas de pago vacío.
	resumen := Resumen{PorForma: map[string]int64{}}
	for _, v := range ventas {
		// Las ventas canceladas no cuentan.
		if v.Estado != Activa {
			continue
		}
		// Cuenta la venta.
		resumen.Cantidad++
		// Suma su total.
		resumen.Total += v.Total
		// Suma cada pago a su forma de pago.
		for _, pago := range v.Pagos {
			resumen.PorForma[pago.Forma] += pago.Monto
		}
		// Del efectivo se resta el cambio entregado: ese dinero salió del cajón.
		resumen.PorForma[Efectivo] -= v.Cambio
	}
	return resumen
}

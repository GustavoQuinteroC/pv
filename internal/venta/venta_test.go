// Archivo: venta_test.go
// Qué hace: pruebas automáticas de las cuentas de venta. No necesitan base de
// datos. Se corren con "go test ./...": si alguna regla se rompe, la prueba falla.

package venta

import (
	"errors"
	// Biblioteca de pruebas de Go.
	"testing"

	"github.com/GustavoQuinteroC/pv/internal/producto"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Prueba que el importe de un renglón se calcule y redondee bien.
func TestImporteRenglon(t *testing.T) {
	// Lista de casos: cada uno trae datos de entrada y el resultado esperado.
	casos := []struct {
		nombre   string
		precio   int64
		cantidad int64
		esperado int64
	}{
		{"una pieza", 1800, 1000, 1800},
		{"kilo y cuarto", 2400, 1250, 3000},
		{"redondea hacia arriba desde la mitad", 1999, 500, 1000},
		{"redondea hacia abajo", 1001, 500, 501},
	}

	// Corre cada caso como una sub-prueba con su nombre.
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			// Calcula con la función real.
			obtenido := ImporteRenglon(caso.precio, caso.cantidad)
			// Si no coincide con lo esperado, la prueba falla con un mensaje.
			if obtenido != caso.esperado {
				t.Errorf("ImporteRenglon(%d, %d) = %d, se esperaba %d",
					caso.precio, caso.cantidad, obtenido, caso.esperado)
			}
		})
	}
}

// Prueba la conversión de dinero a cantidad.
func TestCantidadPorMonto(t *testing.T) {
	// Con $24 el kilo, $20 son 833 gramos (0.833 kg).
	obtenida := CantidadPorMonto(2400, 2000)
	if obtenida != 833 {
		t.Errorf("CantidadPorMonto(2400, 2000) = %d, se esperaba 833", obtenida)
	}
}

// catalogoDePrueba arma un catálogo pequeño: una tortilla a granel y una paleta por pieza.
func catalogoDePrueba() (map[bson.ObjectID]producto.Producto, producto.Producto, producto.Producto) {
	tortilla := producto.Producto{ID: bson.NewObjectID(), Nombre: "Tortilla", Precio: 2400, Unidad: "kg", Activo: true}
	paleta := producto.Producto{ID: bson.NewObjectID(), Nombre: "Paleta", Precio: 500, Unidad: producto.Pieza, Activo: true}

	catalogo := map[bson.ObjectID]producto.Producto{tortilla.ID: tortilla, paleta.ID: paleta}
	return catalogo, tortilla, paleta
}

// Prueba una venta normal: calcula bien el total y el cambio.
func TestNuevaCalculaTotalYCambio(t *testing.T) {
	catalogo, tortilla, paleta := catalogoDePrueba()
	pedido := Pedido{
		Token: "abc",
		Renglones: []Linea{
			{Producto: tortilla.ID.Hex(), Cantidad: 1500}, // 1.5 kg = $36.00
			{Producto: paleta.ID.Hex(), Cantidad: 2000},   // 2 piezas = $10.00
		},
		Descuento: 100,
		Pagos:     []Pago{{Forma: Efectivo, Monto: 5000}},
	}

	venta, err := Nueva(pedido, catalogo)
	// Si dio error, no tiene caso seguir.
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	// $36 + $10 - $1 de descuento = $45.
	if venta.Total != 4500 {
		t.Errorf("total = %d, se esperaba 4500", venta.Total)
	}
	// Pagó $50, debe regresarse $5.
	if venta.Cambio != 500 {
		t.Errorf("cambio = %d, se esperaba 500", venta.Cambio)
	}
}

// Prueba que "$20 de tortilla" cobre exactamente $20.
func TestNuevaPorMontoRespetaElImporte(t *testing.T) {
	catalogo, tortilla, _ := catalogoDePrueba()
	pedido := Pedido{
		Token:     "abc",
		Renglones: []Linea{{Producto: tortilla.ID.Hex(), Monto: 2000}},
		Pagos:     []Pago{{Forma: Efectivo, Monto: 2000}},
	}

	venta, err := Nueva(pedido, catalogo)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if venta.Total != 2000 {
		t.Errorf("total = %d, se esperaba exactamente 2000", venta.Total)
	}
}

// Prueba que los pedidos con errores se rechacen con el mensaje correcto.
func TestNuevaRechazaPedidosInvalidos(t *testing.T) {
	catalogo, tortilla, paleta := catalogoDePrueba()
	// Atajo para armar "pago en efectivo por tanto".
	efectivo := func(monto int64) []Pago { return []Pago{{Forma: Efectivo, Monto: monto}} }

	// Cada caso: un pedido malo y el error que debe producir.
	casos := []struct {
		nombre   string
		pedido   Pedido
		esperado error
	}{
		{
			"sin token",
			Pedido{Renglones: []Linea{{Producto: paleta.ID.Hex(), Cantidad: 1000}}, Pagos: efectivo(500)},
			ErrToken,
		},
		{
			"sin renglones",
			Pedido{Token: "a", Pagos: efectivo(500)},
			ErrVacia,
		},
		{
			"producto inexistente",
			Pedido{Token: "a", Renglones: []Linea{{Producto: bson.NewObjectID().Hex(), Cantidad: 1000}}, Pagos: efectivo(500)},
			ErrProducto,
		},
		{
			"pieza partida",
			Pedido{Token: "a", Renglones: []Linea{{Producto: paleta.ID.Hex(), Cantidad: 1500}}, Pagos: efectivo(1000)},
			ErrCantidad,
		},
		{
			"cantidad en cero",
			Pedido{Token: "a", Renglones: []Linea{{Producto: tortilla.ID.Hex(), Cantidad: 0}}, Pagos: efectivo(1000)},
			ErrCantidad,
		},
		{
			"descuento que cubre todo",
			Pedido{Token: "a", Renglones: []Linea{{Producto: paleta.ID.Hex(), Cantidad: 1000}}, Descuento: 500, Pagos: efectivo(500)},
			ErrDescuento,
		},
		{
			"pago insuficiente",
			Pedido{Token: "a", Renglones: []Linea{{Producto: paleta.ID.Hex(), Cantidad: 1000}}, Pagos: efectivo(400)},
			ErrPagoInsuficiente,
		},
		{
			"forma de pago inválida",
			Pedido{Token: "a", Renglones: []Linea{{Producto: paleta.ID.Hex(), Cantidad: 1000}}, Pagos: []Pago{{Forma: "cheque", Monto: 500}}},
			ErrFormaPago,
		},
		{
			"cambio sin efectivo",
			Pedido{Token: "a", Renglones: []Linea{{Producto: paleta.ID.Hex(), Cantidad: 1000}}, Pagos: []Pago{{Forma: Tarjeta, Monto: 800}}},
			ErrCambio,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := Nueva(caso.pedido, catalogo)
			// errors.Is compara el error obtenido contra el esperado.
			if !errors.Is(err, caso.esperado) {
				t.Errorf("error = %v, se esperaba %v", err, caso.esperado)
			}
		})
	}
}

// Prueba el resumen: el cambio se descuenta del efectivo y las canceladas no cuentan.
func TestResumirDescuentaElCambioDelEfectivo(t *testing.T) {
	ventas := []Venta{
		// Venta de $45 pagada con $50 en efectivo (cambio $5).
		{Estado: Activa, Total: 4500, Cambio: 500, Pagos: []Pago{{Forma: Efectivo, Monto: 5000}}},
		// Venta de $10 con tarjeta.
		{Estado: Activa, Total: 1000, Pagos: []Pago{{Forma: Tarjeta, Monto: 1000}}},
		// Venta cancelada: no debe contar.
		{Estado: Cancelada, Total: 9999, Pagos: []Pago{{Forma: Efectivo, Monto: 9999}}},
	}

	resumen := Resumir(ventas)

	// Dos ventas activas que suman $55.
	if resumen.Cantidad != 2 || resumen.Total != 5500 {
		t.Errorf("cantidad = %d, total = %d; se esperaba 2 y 5500", resumen.Cantidad, resumen.Total)
	}
	// Efectivo: $50 que dieron menos $5 de cambio = $45.
	if resumen.PorForma[Efectivo] != 4500 {
		t.Errorf("efectivo = %d, se esperaba 4500", resumen.PorForma[Efectivo])
	}
	// Tarjeta: $10.
	if resumen.PorForma[Tarjeta] != 1000 {
		t.Errorf("tarjeta = %d, se esperaba 1000", resumen.PorForma[Tarjeta])
	}
}

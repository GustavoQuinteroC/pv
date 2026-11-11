package venta

import (
	"errors"
	"testing"

	"github.com/GustavoQuinteroC/pv/internal/producto"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestImporteRenglon(t *testing.T) {
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

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			obtenido := ImporteRenglon(caso.precio, caso.cantidad)
			if obtenido != caso.esperado {
				t.Errorf("ImporteRenglon(%d, %d) = %d, se esperaba %d",
					caso.precio, caso.cantidad, obtenido, caso.esperado)
			}
		})
	}
}

func TestCantidadPorMonto(t *testing.T) {
	// Con $24 el kilo, $20 son 833 gramos (0.833 kg).
	obtenida := CantidadPorMonto(2400, 2000)
	if obtenida != 833 {
		t.Errorf("CantidadPorMonto(2400, 2000) = %d, se esperaba 833", obtenida)
	}
}

func catalogoDePrueba() (map[bson.ObjectID]producto.Producto, producto.Producto, producto.Producto) {
	tortilla := producto.Producto{ID: bson.NewObjectID(), Nombre: "Tortilla", Precio: 2400, Unidad: "kg", Activo: true}
	paleta := producto.Producto{ID: bson.NewObjectID(), Nombre: "Paleta", Precio: 500, Unidad: producto.Pieza, Activo: true}

	catalogo := map[bson.ObjectID]producto.Producto{tortilla.ID: tortilla, paleta.ID: paleta}
	return catalogo, tortilla, paleta
}

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
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if venta.Total != 4500 {
		t.Errorf("total = %d, se esperaba 4500", venta.Total)
	}
	if venta.Cambio != 500 {
		t.Errorf("cambio = %d, se esperaba 500", venta.Cambio)
	}
}

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

func TestNuevaRechazaPedidosInvalidos(t *testing.T) {
	catalogo, tortilla, paleta := catalogoDePrueba()
	efectivo := func(monto int64) []Pago { return []Pago{{Forma: Efectivo, Monto: monto}} }

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
			if !errors.Is(err, caso.esperado) {
				t.Errorf("error = %v, se esperaba %v", err, caso.esperado)
			}
		})
	}
}

func TestResumirDescuentaElCambioDelEfectivo(t *testing.T) {
	ventas := []Venta{
		{Estado: Activa, Total: 4500, Cambio: 500, Pagos: []Pago{{Forma: Efectivo, Monto: 5000}}},
		{Estado: Activa, Total: 1000, Pagos: []Pago{{Forma: Tarjeta, Monto: 1000}}},
		{Estado: Cancelada, Total: 9999, Pagos: []Pago{{Forma: Efectivo, Monto: 9999}}},
	}

	resumen := Resumir(ventas)

	if resumen.Cantidad != 2 || resumen.Total != 5500 {
		t.Errorf("cantidad = %d, total = %d; se esperaba 2 y 5500", resumen.Cantidad, resumen.Total)
	}
	if resumen.PorForma[Efectivo] != 4500 {
		t.Errorf("efectivo = %d, se esperaba 4500", resumen.PorForma[Efectivo])
	}
	if resumen.PorForma[Tarjeta] != 1000 {
		t.Errorf("tarjeta = %d, se esperaba 1000", resumen.PorForma[Tarjeta])
	}
}

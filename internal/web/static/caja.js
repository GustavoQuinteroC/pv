"use strict";

// Archivo: caja.js
// Qué hace: toda la lógica de la pantalla de caja, en el navegador.
//
// El carrito vive aquí, en el navegador: agregar, cambiar o quitar productos no
// toca el servidor, así que escanear es instantáneo. El servidor solo recibe la
// venta completa al cobrar, y vuelve a calcular todo con los precios del catálogo.
//
// Partes del archivo:
//   1. Referencias a elementos de la página.
//   2. Dinero: convertir y calcular en centavos.
//   3. Almacenamiento del navegador (el carrito sobrevive a recargar).
//   4. Catálogo: mosaicos, búsqueda y escáner de códigos.
//   5. Carrito: agregar, cambiar cantidad, quitar.
//   6. Tickets en espera.
//   7. Cobro: diálogo de pago y envío al servidor.

// Nombres con los que se guardan los datos en el navegador (localStorage).
const claveCarrito = "pv.carrito";
const claveEspera = "pv.espera";

// Lee la lista de productos que el servidor dejó dentro de la página como JSON.
const productos = JSON.parse(document.getElementById("productos").textContent);

// ---------- Elementos de la página ----------

// Se buscan una sola vez y se guardan en constantes.
const campoBuscar = document.getElementById("buscar");
const campoDescuento = document.getElementById("descuento");
const campoTotal = document.getElementById("total");
const mosaicos = document.getElementById("mosaicos");
const listaRenglones = document.getElementById("renglones");
const panelEspera = document.getElementById("espera");
const plantillaRenglon = document.getElementById("plantilla-renglon");

// Elementos del diálogo de cobro.
const dialogoCobro = document.getElementById("cobro");
const formularioCobro = document.getElementById("formulario-cobro");
const campoCobroTotal = document.getElementById("cobro-total");
const pagoEfectivo = document.getElementById("pago-efectivo");
const pagoTarjeta = document.getElementById("pago-tarjeta");
const pagoTransferencia = document.getElementById("pago-transferencia");
const etiquetaCambio = document.getElementById("cambio");
const avisoCobro = document.getElementById("cobro-error");
const botonConfirmar = document.getElementById("confirmar");

// Cada renglón: { id, nombre, unidad, precio, cantidad, monto }
// El dinero va en centavos y las cantidades en milésimas, igual que en el servidor.
// Empieza con lo que haya guardado en el navegador (por si se recargó la página).
let renglones = leer(claveCarrito);

// Identifica esta venta. Cambia cuando cambia el carrito, y así un doble clic en
// "Confirmar" no cobra dos veces pero una venta distinta sí se cobra.
let token = crypto.randomUUID();

// ---------- Dinero ----------

// Convierte centavos a texto con formato de pesos mexicanos: 12345 → "$123.45".
const pesos = (centavos) =>
  (centavos / 100).toLocaleString("es-MX", { style: "currency", currency: "MXN" });

// Convierte lo que se escribió ("12.50") a centavos. Si no es número, da 0.
const aCentavos = (texto) => Math.round(parseFloat(texto.replaceAll(",", "")) * 100) || 0;
// Convierte lo que se escribió ("1.5") a milésimas. Si no es número, da 0.
const aMilesimas = (texto) => Math.round(parseFloat(texto.replaceAll(",", "")) * 1000) || 0;

// Importe de un renglón: el monto que se pidió, o (precio × cantidad).
// Se divide entre 1000 porque la cantidad está en milésimas.
const importe = (renglon) =>
  renglon.monto || Math.round((renglon.precio * renglon.cantidad) / 1000);

// Suma de todos los importes.
const subtotal = () => renglones.reduce((suma, renglon) => suma + importe(renglon), 0);
// Descuento escrito; nunca puede ser mayor que el subtotal.
const descuento = () => Math.min(aCentavos(campoDescuento.value), subtotal());
// Lo que debe pagar el cliente.
const total = () => subtotal() - descuento();

// ---------- Almacenamiento del navegador ----------

// Lee una lista guardada. Si no hay nada, o está dañada, devuelve una lista vacía.
function leer(clave) {
  try {
    return JSON.parse(localStorage.getItem(clave)) ?? [];
  } catch {
    return [];
  }
}

// Guarda un valor en el navegador como texto JSON.
function guardar(clave, valor) {
  localStorage.setItem(clave, JSON.stringify(valor));
}

// ---------- Catálogo ----------

// ¿El producto coincide con lo escrito? Por parte del nombre o por código de barras exacto.
const coincide = (producto, texto) =>
  producto.nombre.toLowerCase().includes(texto) || producto.codigo === texto;

// Dibuja los botones de productos: los favoritos, o los que coinciden con la búsqueda.
function mostrarMosaicos() {
  const texto = campoBuscar.value.trim().toLowerCase();
  const favoritos = productos.filter((producto) => producto.favorito);

  // Sin búsqueda: favoritos (o todos, si no hay favoritos).
  let lista = favoritos.length > 0 ? favoritos : productos;
  // Con búsqueda: cualquier producto que coincida.
  if (texto) {
    lista = productos.filter((producto) => coincide(producto, texto));
  }
  // Reemplaza los mosaicos anteriores por los nuevos.
  mosaicos.replaceChildren(...lista.map(crearMosaico));
}

// Crea el botón de un producto: nombre y precio.
function crearMosaico(producto) {
  const nombre = document.createElement("strong");
  nombre.textContent = producto.nombre;

  const precio = document.createElement("small");
  precio.textContent = `${pesos(producto.precio)} / ${producto.unidad}`;

  const mosaico = document.createElement("button");
  mosaico.type = "button";
  mosaico.className = "mosaico";
  mosaico.append(nombre, precio);
  // Al tocarlo, se agrega al carrito.
  mosaico.addEventListener("click", () => agregar(producto));
  return mosaico;
}

// Con el escáner: el código termina con Enter y el producto se agrega solo.
campoBuscar.addEventListener("keydown", (evento) => {
  if (evento.key !== "Enter") {
    return;
  }

  const texto = campoBuscar.value.trim().toLowerCase();
  const encontrados = productos.filter((producto) => coincide(producto, texto));
  // Prefiere el producto cuyo código es exactamente lo escrito.
  const porCodigo = encontrados.find((producto) => producto.codigo === texto);
  // Si no, y solo hay un resultado, ese.
  const unico = encontrados.length === 1 ? encontrados[0] : null;

  const elegido = porCodigo ?? unico;
  if (elegido) {
    agregar(elegido);
  }
});

// Cada vez que se escribe en el buscador, se actualizan los mosaicos.
campoBuscar.addEventListener("input", mostrarMosaicos);

// ---------- Carrito ----------

// Agrega un producto al carrito (o suma 1 si ya estaba).
function agregar(producto) {
  const existente = renglones.find((renglon) => renglon.id === producto.id);

  if (existente) {
    // 1000 milésimas = 1 unidad.
    existente.cantidad += 1000;
    existente.monto = 0;
  } else {
    renglones.push({
      id: producto.id,
      nombre: producto.nombre,
      unidad: producto.unidad,
      precio: producto.precio,
      cantidad: 1000,
      monto: 0,
    });
  }

  // Limpia el buscador y deja todo listo para el siguiente producto.
  campoBuscar.value = "";
  mostrarMosaicos();
  pintar();
  campoBuscar.focus();
}

// Redibuja el carrito, el total y lo guarda en el navegador.
function pintar() {
  // El carrito cambió: es una venta distinta, así que cambia el token.
  token = crypto.randomUUID();
  listaRenglones.replaceChildren(...renglones.map(crearRenglon));
  campoTotal.textContent = pesos(total());
  guardar(claveCarrito, renglones);
}

// Crea la fila visual de un renglón a partir de la plantilla HTML.
function crearRenglon(renglon, posicion) {
  const fila = plantillaRenglon.content.cloneNode(true);
  const esPieza = renglon.unidad === "pieza";

  fila.querySelector(".nombre").textContent = renglon.nombre;
  fila.querySelector(".importe").textContent = pesos(importe(renglon));
  fila.querySelector(".unidad").textContent = renglon.unidad;

  // Campo de cantidad: se muestra en unidades (milésimas / 1000).
  const cantidad = fila.querySelector(".cantidad");
  cantidad.value = renglon.cantidad / 1000;
  cantidad.addEventListener("change", () => {
    const milesimas = aMilesimas(cantidad.value);
    // Las piezas se redondean a números enteros; kilos y litros aceptan decimales.
    renglon.cantidad = esPieza ? Math.round(milesimas / 1000) * 1000 : milesimas;
    renglon.monto = 0;
    pintar();
  });

  // Solo lo que se vende a granel se puede pedir por monto: "$20 de tortilla".
  const monto = fila.querySelector(".monto");
  if (esPieza) {
    // Las piezas no tienen campo de monto.
    monto.remove();
  } else {
    monto.value = renglon.monto ? (renglon.monto / 100).toFixed(2) : "";
    monto.addEventListener("change", () => {
      renglon.monto = aCentavos(monto.value);
      if (renglon.monto) {
        // Calcula cuánta cantidad equivale a ese monto.
        renglon.cantidad = Math.round((renglon.monto * 1000) / renglon.precio);
      }
      pintar();
    });
  }

  // Botón "Quitar": saca este renglón del carrito.
  fila.querySelector(".quitar").addEventListener("click", () => {
    renglones.splice(posicion, 1);
    pintar();
  });
  return fila;
}

// Al cambiar el descuento, se recalcula el total.
campoDescuento.addEventListener("change", pintar);

// ---------- Tickets en espera ----------

// Hora actual como "03:45 p. m." para identificar cada ticket en espera.
const horaActual = () =>
  new Date().toLocaleTimeString("es-MX", { hour: "2-digit", minute: "2-digit" });

// Dibuja un botón por cada ticket en espera.
function pintarEspera() {
  const enEspera = leer(claveEspera);

  panelEspera.replaceChildren(
    ...enEspera.map((ticket, posicion) => {
      const boton = document.createElement("button");
      boton.type = "button";
      boton.textContent = `Recuperar el de las ${ticket.hora} (${ticket.renglones.length} productos)`;
      boton.addEventListener("click", () => recuperar(posicion));
      return boton;
    }),
  );
}

// Guarda el carrito actual "en espera" y deja el carrito vacío.
function ponerEnEspera() {
  // Un carrito vacío no se guarda.
  if (renglones.length === 0) {
    return;
  }

  const enEspera = leer(claveEspera);
  enEspera.push({ hora: horaActual(), renglones });
  guardar(claveEspera, enEspera);

  renglones = [];
  campoDescuento.value = "";
  pintar();
  pintarEspera();
}

// Al recuperar un ticket, el que se estaba capturando pasa a la espera.
function recuperar(posicion) {
  const enEspera = leer(claveEspera);
  // splice saca ese ticket de la lista y lo devuelve.
  const [ticket] = enEspera.splice(posicion, 1);

  // Si había algo en el carrito, no se pierde: pasa a la espera.
  if (renglones.length > 0) {
    enEspera.push({ hora: horaActual(), renglones });
  }
  guardar(claveEspera, enEspera);

  renglones = ticket.renglones;
  pintar();
  pintarEspera();
}

document.getElementById("esperar").addEventListener("click", ponerEnEspera);

// ---------- Cobro ----------

// Suma de lo que el cliente entrega en efectivo, tarjeta y transferencia.
const pagado = () =>
  aCentavos(pagoEfectivo.value) + aCentavos(pagoTarjeta.value) + aCentavos(pagoTransferencia.value);

// Muestra el cambio, o cuánto falta por pagar.
function actualizarCambio() {
  const diferencia = pagado() - total();

  etiquetaCambio.textContent =
    diferencia >= 0 ? `Cambio ${pesos(diferencia)}` : `Falta ${pesos(-diferencia)}`;
  // Si falta dinero, la clase "falta" lo pinta en rojo.
  etiquetaCambio.classList.toggle("falta", diferencia < 0);
}

// Abre el diálogo de cobro con los campos limpios.
function abrirCobro() {
  // No se cobra un carrito vacío.
  if (renglones.length === 0) {
    return;
  }

  for (const campo of [pagoEfectivo, pagoTarjeta, pagoTransferencia]) {
    campo.value = "";
  }
  campoCobroTotal.textContent = pesos(total());
  avisoCobro.hidden = true;
  actualizarCambio();

  dialogoCobro.showModal();
  pagoEfectivo.focus();
}

// Muestra un mensaje de error dentro del diálogo de cobro.
function mostrarErrorDeCobro(mensaje) {
  avisoCobro.textContent = mensaje;
  avisoCobro.hidden = false;
}

// Manda la venta al servidor. "async/await" espera la respuesta sin congelar la pantalla.
async function confirmarCobro(evento) {
  // Evita que el formulario recargue la página.
  evento.preventDefault();

  // Arma el pedido. Solo manda ids y cantidades: el servidor pone los precios.
  const pedido = {
    token,
    renglones: renglones.map((renglon) => ({
      producto: renglon.id,
      cantidad: renglon.cantidad,
      monto: renglon.monto,
    })),
    descuento: descuento(),
    // Solo incluye las formas de pago donde se puso algún monto.
    pagos: [
      { forma: "efectivo", monto: aCentavos(pagoEfectivo.value) },
      { forma: "tarjeta", monto: aCentavos(pagoTarjeta.value) },
      { forma: "transferencia", monto: aCentavos(pagoTransferencia.value) },
    ].filter((pago) => pago.monto > 0),
  };

  // Bloquea el botón para evitar cobrar dos veces con doble clic.
  botonConfirmar.disabled = true;
  try {
    const respuesta = await fetch("/api/ventas", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(pedido),
    });
    const resultado = await respuesta.json();

    // Si el servidor rechazó la venta, muestra su mensaje y se queda en el diálogo.
    if (!respuesta.ok) {
      mostrarErrorDeCobro(resultado.error);
      return;
    }

    // Venta guardada: vacía el carrito y abre el ticket para imprimir.
    renglones = [];
    guardar(claveCarrito, renglones);
    window.location.href = `/ventas/${resultado.id}?imprimir=1`;
  } catch {
    // Falla de red: no se pudo hablar con el servidor.
    mostrarErrorDeCobro("No hay conexión con el servidor. Revisa la red e intenta de nuevo.");
  } finally {
    // Pase lo que pase, vuelve a habilitar el botón.
    botonConfirmar.disabled = false;
  }
}

// Conecta cada botón con su función.
document.getElementById("cobrar").addEventListener("click", abrirCobro);
document.getElementById("volver").addEventListener("click", () => dialogoCobro.close());
// Al cerrar el diálogo, el cursor regresa al buscador.
dialogoCobro.addEventListener("close", () => campoBuscar.focus());
formularioCobro.addEventListener("submit", confirmarCobro);

// Al escribir en cualquier campo de pago, se recalcula el cambio.
for (const campo of [pagoEfectivo, pagoTarjeta, pagoTransferencia]) {
  campo.addEventListener("input", actualizarCambio);
}

// Botones de billetes: llenan el efectivo con $50, $100…, o con el total exacto.
document.querySelectorAll("[data-billete]").forEach((boton) => {
  boton.addEventListener("click", () => {
    const centavos = boton.dataset.billete === "exacto" ? total() : Number(boton.dataset.billete) * 100;
    pagoEfectivo.value = (centavos / 100).toFixed(2);
    actualizarCambio();
  });
});

// ---------- Arranque ----------

// Al cargar la página: dibuja todo y deja el cursor en el buscador.
mostrarMosaicos();
pintar();
pintarEspera();
campoBuscar.focus();

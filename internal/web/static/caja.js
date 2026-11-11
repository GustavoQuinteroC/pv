"use strict";

// Pantalla de caja.
//
// El carrito vive aquí, en el navegador: agregar, cambiar o quitar productos no
// toca el servidor, así que escanear es instantáneo. El servidor solo recibe la
// venta completa al cobrar, y vuelve a calcular todo con los precios del catálogo.

const claveCarrito = "pv.carrito";
const claveEspera = "pv.espera";

const productos = JSON.parse(document.getElementById("productos").textContent);

const campoBuscar = document.getElementById("buscar");
const campoDescuento = document.getElementById("descuento");
const campoTotal = document.getElementById("total");
const mosaicos = document.getElementById("mosaicos");
const listaRenglones = document.getElementById("renglones");
const panelEspera = document.getElementById("espera");
const plantillaRenglon = document.getElementById("plantilla-renglon");

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
let renglones = leer(claveCarrito);

// Identifica esta venta. Cambia cuando cambia el carrito, y así un doble clic en
// "Confirmar" no cobra dos veces pero una venta distinta sí se cobra.
let token = crypto.randomUUID();

// ---------- Dinero ----------

const pesos = (centavos) =>
  (centavos / 100).toLocaleString("es-MX", { style: "currency", currency: "MXN" });

const aCentavos = (texto) => Math.round(parseFloat(texto.replaceAll(",", "")) * 100) || 0;
const aMilesimas = (texto) => Math.round(parseFloat(texto.replaceAll(",", "")) * 1000) || 0;

const importe = (renglon) =>
  renglon.monto || Math.round((renglon.precio * renglon.cantidad) / 1000);

const subtotal = () => renglones.reduce((suma, renglon) => suma + importe(renglon), 0);
const descuento = () => Math.min(aCentavos(campoDescuento.value), subtotal());
const total = () => subtotal() - descuento();

// ---------- Almacenamiento del navegador ----------

function leer(clave) {
  try {
    return JSON.parse(localStorage.getItem(clave)) ?? [];
  } catch {
    return [];
  }
}

function guardar(clave, valor) {
  localStorage.setItem(clave, JSON.stringify(valor));
}

// ---------- Catálogo ----------

const coincide = (producto, texto) =>
  producto.nombre.toLowerCase().includes(texto) || producto.codigo === texto;

function mostrarMosaicos() {
  const texto = campoBuscar.value.trim().toLowerCase();
  const favoritos = productos.filter((producto) => producto.favorito);

  let lista = favoritos.length > 0 ? favoritos : productos;
  if (texto) {
    lista = productos.filter((producto) => coincide(producto, texto));
  }
  mosaicos.replaceChildren(...lista.map(crearMosaico));
}

function crearMosaico(producto) {
  const nombre = document.createElement("strong");
  nombre.textContent = producto.nombre;

  const precio = document.createElement("small");
  precio.textContent = `${pesos(producto.precio)} / ${producto.unidad}`;

  const mosaico = document.createElement("button");
  mosaico.type = "button";
  mosaico.className = "mosaico";
  mosaico.append(nombre, precio);
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
  const porCodigo = encontrados.find((producto) => producto.codigo === texto);
  const unico = encontrados.length === 1 ? encontrados[0] : null;

  const elegido = porCodigo ?? unico;
  if (elegido) {
    agregar(elegido);
  }
});

campoBuscar.addEventListener("input", mostrarMosaicos);

// ---------- Carrito ----------

function agregar(producto) {
  const existente = renglones.find((renglon) => renglon.id === producto.id);

  if (existente) {
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

  campoBuscar.value = "";
  mostrarMosaicos();
  pintar();
  campoBuscar.focus();
}

function pintar() {
  token = crypto.randomUUID();
  listaRenglones.replaceChildren(...renglones.map(crearRenglon));
  campoTotal.textContent = pesos(total());
  guardar(claveCarrito, renglones);
}

function crearRenglon(renglon, posicion) {
  const fila = plantillaRenglon.content.cloneNode(true);
  const esPieza = renglon.unidad === "pieza";

  fila.querySelector(".nombre").textContent = renglon.nombre;
  fila.querySelector(".importe").textContent = pesos(importe(renglon));
  fila.querySelector(".unidad").textContent = renglon.unidad;

  const cantidad = fila.querySelector(".cantidad");
  cantidad.value = renglon.cantidad / 1000;
  cantidad.addEventListener("change", () => {
    const milesimas = aMilesimas(cantidad.value);
    renglon.cantidad = esPieza ? Math.round(milesimas / 1000) * 1000 : milesimas;
    renglon.monto = 0;
    pintar();
  });

  // Solo lo que se vende a granel se puede pedir por monto: "$20 de tortilla".
  const monto = fila.querySelector(".monto");
  if (esPieza) {
    monto.remove();
  } else {
    monto.value = renglon.monto ? (renglon.monto / 100).toFixed(2) : "";
    monto.addEventListener("change", () => {
      renglon.monto = aCentavos(monto.value);
      if (renglon.monto) {
        renglon.cantidad = Math.round((renglon.monto * 1000) / renglon.precio);
      }
      pintar();
    });
  }

  fila.querySelector(".quitar").addEventListener("click", () => {
    renglones.splice(posicion, 1);
    pintar();
  });
  return fila;
}

campoDescuento.addEventListener("change", pintar);

// ---------- Tickets en espera ----------

const horaActual = () =>
  new Date().toLocaleTimeString("es-MX", { hour: "2-digit", minute: "2-digit" });

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

function ponerEnEspera() {
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
  const [ticket] = enEspera.splice(posicion, 1);

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

const pagado = () =>
  aCentavos(pagoEfectivo.value) + aCentavos(pagoTarjeta.value) + aCentavos(pagoTransferencia.value);

function actualizarCambio() {
  const diferencia = pagado() - total();

  etiquetaCambio.textContent =
    diferencia >= 0 ? `Cambio ${pesos(diferencia)}` : `Falta ${pesos(-diferencia)}`;
  etiquetaCambio.classList.toggle("falta", diferencia < 0);
}

function abrirCobro() {
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

function mostrarErrorDeCobro(mensaje) {
  avisoCobro.textContent = mensaje;
  avisoCobro.hidden = false;
}

async function confirmarCobro(evento) {
  evento.preventDefault();

  const pedido = {
    token,
    renglones: renglones.map((renglon) => ({
      producto: renglon.id,
      cantidad: renglon.cantidad,
      monto: renglon.monto,
    })),
    descuento: descuento(),
    pagos: [
      { forma: "efectivo", monto: aCentavos(pagoEfectivo.value) },
      { forma: "tarjeta", monto: aCentavos(pagoTarjeta.value) },
      { forma: "transferencia", monto: aCentavos(pagoTransferencia.value) },
    ].filter((pago) => pago.monto > 0),
  };

  botonConfirmar.disabled = true;
  try {
    const respuesta = await fetch("/api/ventas", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(pedido),
    });
    const resultado = await respuesta.json();

    if (!respuesta.ok) {
      mostrarErrorDeCobro(resultado.error);
      return;
    }

    renglones = [];
    guardar(claveCarrito, renglones);
    window.location.href = `/ventas/${resultado.id}?imprimir=1`;
  } catch {
    mostrarErrorDeCobro("No hay conexión con el servidor. Revisa la red e intenta de nuevo.");
  } finally {
    botonConfirmar.disabled = false;
  }
}

document.getElementById("cobrar").addEventListener("click", abrirCobro);
document.getElementById("volver").addEventListener("click", () => dialogoCobro.close());
dialogoCobro.addEventListener("close", () => campoBuscar.focus());
formularioCobro.addEventListener("submit", confirmarCobro);

for (const campo of [pagoEfectivo, pagoTarjeta, pagoTransferencia]) {
  campo.addEventListener("input", actualizarCambio);
}

document.querySelectorAll("[data-billete]").forEach((boton) => {
  boton.addEventListener("click", () => {
    const centavos = boton.dataset.billete === "exacto" ? total() : Number(boton.dataset.billete) * 100;
    pagoEfectivo.value = (centavos / 100).toFixed(2);
    actualizarCambio();
  });
});

// ---------- Arranque ----------

mostrarMosaicos();
pintar();
pintarEspera();
campoBuscar.focus();

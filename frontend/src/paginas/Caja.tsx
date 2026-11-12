// Archivo: Caja.tsx
// Qué hace: la pantalla de venta. Busca productos (también con lector de códigos
// de barras), arma el ticket actual, permite descuento, tickets en espera y el
// cobro con efectivo, tarjeta y transferencia.
//
// El carrito vive aquí, en el navegador: agregar, cambiar o quitar productos no
// toca el servidor, así que escanear es instantáneo. El servidor solo recibe la
// venta completa al cobrar, y vuelve a calcular todo con los precios del catálogo.
// Si no hay turno abierto, esta pantalla pide abrirlo.

import { useMemo, useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";
import { aCentavos, aMilesimas, api, fecha, pesos, useApi, useGuardado } from "../lib";
import type { Producto, Turno } from "../tipos";
import { Aviso, Boton, Campo, Encabezado, Etiqueta, Tarjeta, Ventana } from "../ui";

// Una línea del carrito. Dinero en centavos y cantidades en milésimas, igual que en el servidor.
// `monto` > 0 significa "vender por monto": "$20 de tortilla".
interface Linea {
  id: string;
  nombre: string;
  unidad: string;
  precio: number;
  cantidad: number;
  monto: number;
}

interface Estado {
  turno: Turno | null;
  propio: boolean;
  productos?: Producto[];
}

// Importe de una línea: el monto pedido, o precio × cantidad (la cantidad está en milésimas).
const importe = (l: Linea) => l.monto || Math.round((l.precio * l.cantidad) / 1000);

export default function Caja() {
  const { datos, error, recargar } = useApi<Estado>("/caja");
  if (!datos) return <Aviso texto={error} />;

  // Sin turno propio: abrir turno, o avisar que la caja la usa otra persona.
  if (!datos.propio) return <AbrirTurno turno={datos.turno} alAbrir={recargar} />;
  return <Venta productos={datos.productos!} />;
}

// ---------- Abrir turno ----------

function AbrirTurno({ turno, alAbrir }: { turno: Turno | null; alAbrir: () => void }) {
  const [fallo, setFallo] = useState("");

  async function abrir(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    try {
      await api("/turno/abrir", { fondo: aCentavos(String(new FormData(evento.currentTarget).get("fondo"))) });
      alAbrir();
    } catch (e) {
      setFallo((e as Error).message);
    }
  }

  if (turno) {
    return (
      <Tarjeta titulo="La caja está en uso" className="max-w-xl">
        <p className="text-gray-700">
          {turno.cajero_nombre} abrió el turno el {fecha(turno.abierto)}.
        </p>
        <p className="my-3 text-sm text-gray-500">Solo se puede vender desde el turno propio. Hasta que esa persona cierre, no se abre otro.</p>
        <Link className="text-brand-500 hover:underline" to="/turno">
          Ver el turno
        </Link>
      </Tarjeta>
    );
  }

  return (
    <Tarjeta titulo="Abrir turno" className="max-w-xl">
      <p className="mb-4 text-sm text-gray-500">Cuenta el efectivo con el que empieza la caja.</p>
      <Aviso texto={fallo} />
      <form onSubmit={abrir} className="space-y-4">
        <Etiqueta texto="Fondo de caja">
          <Campo name="fondo" inputMode="decimal" placeholder="0.00" autoFocus />
        </Etiqueta>
        <Boton tipo="primario">Abrir turno</Boton>
      </form>
    </Tarjeta>
  );
}

// ---------- Venta ----------

function Venta({ productos }: { productos: Producto[] }) {
  // El carrito y los tickets en espera se guardan en el navegador: sobreviven a recargar.
  const [lineas, setLineas] = useGuardado<Linea[]>("pv.carrito", []);
  const [espera, setEspera] = useGuardado<{ hora: string; lineas: Linea[] }[]>("pv.espera", []);
  const [buscar, setBuscar] = useState("");
  const [descuentoTexto, setDescuentoTexto] = useState("");
  const [cobrando, setCobrando] = useState(false);

  const subtotal = lineas.reduce((suma, l) => suma + importe(l), 0);
  // El descuento nunca puede ser mayor que el subtotal.
  const descuento = Math.min(aCentavos(descuentoTexto), subtotal);
  const total = subtotal - descuento;

  // Identifica esta venta. Cambia cuando cambia el carrito, y así un doble clic en
  // "Confirmar" no cobra dos veces pero una venta distinta sí se cobra.
  const token = useMemo(() => crypto.randomUUID(), [lineas, descuento]);

  // ¿El producto coincide con lo escrito? Por parte del nombre o por código de barras exacto.
  const texto = buscar.trim().toLowerCase();
  const coincide = (p: Producto) => p.nombre.toLowerCase().includes(texto) || p.codigo === texto;
  const favoritos = productos.filter((p) => p.favorito);
  const mosaicos = texto ? productos.filter(coincide) : favoritos.length > 0 ? favoritos : productos;

  function agregar(p: Producto) {
    const existente = lineas.find((l) => l.id === p.id);
    setLineas(
      existente
        ? lineas.map((l) => (l === existente ? { ...l, cantidad: l.cantidad + 1000, monto: 0 } : l))
        : [...lineas, { id: p.id, nombre: p.nombre, unidad: p.unidad, precio: p.precio, cantidad: 1000, monto: 0 }],
    );
    setBuscar("");
  }

  // Con el lector de códigos: el código termina con Enter y el producto se agrega solo.
  function alTeclear(evento: React.KeyboardEvent) {
    if (evento.key !== "Enter") return;
    const encontrados = productos.filter(coincide);
    const elegido = encontrados.find((p) => p.codigo === texto) ?? (encontrados.length === 1 ? encontrados[0] : undefined);
    if (elegido) agregar(elegido);
  }

  const cambiar = (i: number, parche: Partial<Linea>) => setLineas(lineas.map((l, j) => (j === i ? { ...l, ...parche } : l)));

  // Guarda el carrito "en espera" y lo vacía.
  function ponerEnEspera() {
    if (lineas.length === 0) return;
    setEspera([...espera, { hora: new Date().toLocaleTimeString("es-MX", { timeStyle: "short" }), lineas }]);
    setLineas([]);
    setDescuentoTexto("");
  }

  // Al recuperar un ticket, el que se estaba capturando pasa a la espera.
  function recuperar(i: number) {
    const resto = espera.filter((_, j) => j !== i);
    const actual = lineas.length > 0 ? [{ hora: new Date().toLocaleTimeString("es-MX", { timeStyle: "short" }), lineas }] : [];
    setEspera([...resto, ...actual]);
    setLineas(espera[i].lineas);
  }

  return (
    <>
      <Encabezado titulo="Caja" />
      <div className="grid gap-6 lg:grid-cols-[1fr_24rem] lg:items-start">
        <div>
          <Campo
            type="search"
            value={buscar}
            onChange={(e) => setBuscar(e.target.value)}
            onKeyDown={alTeclear}
            placeholder="Escanea el código o escribe el nombre"
            autoComplete="off"
            autoFocus
          />
          <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-4">
            {mosaicos.map((p) => (
              <button
                key={p.id}
                onClick={() => agregar(p)}
                className="flex min-h-20 flex-col justify-between rounded-xl border border-gray-200 bg-white p-3 text-left hover:border-brand-300 hover:shadow-theme-sm"
              >
                <strong className="text-sm font-medium text-gray-800">{p.nombre}</strong>
                <small className="text-gray-500">
                  {pesos(p.precio)} / {p.unidad}
                </small>
              </button>
            ))}
          </div>
        </div>

        <Tarjeta className="lg:sticky lg:top-20">
          <ul className="mb-4 space-y-3">
            {lineas.map((l, i) => (
              <li key={l.id} className="border-b border-gray-100 pb-3">
                <div className="flex justify-between gap-2 font-medium text-gray-800">
                  <span>{l.nombre}</span>
                  <strong>{pesos(importe(l))}</strong>
                </div>
                <div className="mt-2 flex items-center gap-2">
                  {/* La cantidad se confirma al salir del campo (onBlur). Las piezas se redondean a enteros. */}
                  <input
                    key={l.cantidad}
                    defaultValue={l.cantidad / 1000}
                    inputMode="decimal"
                    aria-label="Cantidad"
                    className="h-9 w-20 rounded-lg border border-gray-300 px-2 text-sm"
                    onBlur={(e) => {
                      const milesimas = aMilesimas(e.target.value);
                      cambiar(i, { cantidad: l.unidad === "pieza" ? Math.round(milesimas / 1000) * 1000 : milesimas, monto: 0 });
                    }}
                  />
                  <span className="text-sm text-gray-500">{l.unidad}</span>
                  {/* Solo lo que se vende a granel se puede pedir por monto: "$20 de tortilla". */}
                  {l.unidad !== "pieza" && (
                    <input
                      key={l.monto}
                      defaultValue={l.monto ? (l.monto / 100).toFixed(2) : ""}
                      inputMode="decimal"
                      placeholder="o monto $"
                      aria-label="Importe en pesos"
                      className="h-9 w-24 rounded-lg border border-gray-300 px-2 text-sm"
                      onBlur={(e) => {
                        const monto = aCentavos(e.target.value);
                        cambiar(i, monto ? { monto, cantidad: Math.round((monto * 1000) / l.precio) } : { monto: 0 });
                      }}
                    />
                  )}
                  <button className="ml-auto text-sm text-red-600 hover:underline" onClick={() => setLineas(lineas.filter((_, j) => j !== i))}>
                    Quitar
                  </button>
                </div>
              </li>
            ))}
          </ul>

          <Etiqueta texto="Descuento">
            <Campo value={descuentoTexto} onChange={(e) => setDescuentoTexto(e.target.value)} inputMode="decimal" placeholder="0.00" />
          </Etiqueta>

          <p className="my-4 flex items-baseline justify-between">
            <span className="text-gray-500">Total</span>
            <strong className="text-4xl font-semibold text-gray-800">{pesos(total)}</strong>
          </p>

          <div className="grid grid-cols-[1fr_2fr] gap-3">
            <Boton className="h-12" onClick={ponerEnEspera}>
              En espera
            </Boton>
            <Boton tipo="primario" className="h-12" disabled={lineas.length === 0} onClick={() => setCobrando(true)}>
              Cobrar
            </Boton>
          </div>

          <div className="mt-4 space-y-2">
            {espera.map((t, i) => (
              <Boton key={i} className="w-full" onClick={() => recuperar(i)}>
                Recuperar el de las {t.hora} ({t.lineas.length} productos)
              </Boton>
            ))}
          </div>
        </Tarjeta>
      </div>

      <Cobro abierto={cobrando} total={total} lineas={lineas} descuento={descuento} token={token} alCerrar={() => setCobrando(false)} alCobrar={() => setLineas([])} />
    </>
  );
}

// ---------- Cobro ----------

interface PropsCobro {
  abierto: boolean;
  total: number;
  lineas: Linea[];
  descuento: number;
  token: string;
  alCerrar: () => void;
  alCobrar: () => void;
}

function Cobro({ abierto, total, lineas, descuento, token, alCerrar, alCobrar }: PropsCobro) {
  const navegar = useNavigate();
  const [pagos, setPagos] = useState({ efectivo: "", tarjeta: "", transferencia: "" });
  const [fallo, setFallo] = useState("");
  const [enviando, setEnviando] = useState(false);

  const pagado = aCentavos(pagos.efectivo) + aCentavos(pagos.tarjeta) + aCentavos(pagos.transferencia);
  const diferencia = pagado - total;

  // Manda la venta al servidor. Solo envía ids y cantidades: el servidor pone los precios.
  async function confirmar(evento: FormEvent) {
    evento.preventDefault();
    setEnviando(true);
    try {
      const { id } = await api<{ id: string }>("/ventas", {
        token,
        renglones: lineas.map((l) => ({ producto: l.id, cantidad: l.cantidad, monto: l.monto })),
        descuento,
        // Solo incluye las formas de pago donde se puso algún monto.
        pagos: Object.entries(pagos)
          .map(([forma, texto]) => ({ forma, monto: aCentavos(texto) }))
          .filter((p) => p.monto > 0),
      });
      alCobrar();
      setPagos({ efectivo: "", tarjeta: "", transferencia: "" });
      navegar(`/ventas/${id}?imprimir=1`);
    } catch (e) {
      setFallo((e as Error).message);
    } finally {
      setEnviando(false);
    }
  }

  const poner = (forma: keyof typeof pagos) => (e: { target: { value: string } }) => setPagos({ ...pagos, [forma]: e.target.value });

  return (
    <Ventana abierta={abierto} alCerrar={alCerrar} titulo="Cobrar">
      <form onSubmit={confirmar} className="space-y-4">
        <p className="flex items-baseline justify-between">
          <span className="text-gray-500">Total</span>
          <strong className="text-3xl">{pesos(total)}</strong>
        </p>

        <Etiqueta texto="Efectivo">
          <Campo value={pagos.efectivo} onChange={poner("efectivo")} inputMode="decimal" autoFocus />
        </Etiqueta>
        <div className="flex flex-wrap gap-2">
          {/* Atajos para llenar el efectivo: el total exacto o un billete. */}
          <Boton type="button" onClick={() => setPagos({ ...pagos, efectivo: (total / 100).toFixed(2) })}>
            Exacto
          </Boton>
          {[50, 100, 200, 500].map((billete) => (
            <Boton key={billete} type="button" onClick={() => setPagos({ ...pagos, efectivo: String(billete) })}>
              ${billete}
            </Boton>
          ))}
        </div>

        <Etiqueta texto="Tarjeta">
          <Campo value={pagos.tarjeta} onChange={poner("tarjeta")} inputMode="decimal" />
        </Etiqueta>
        <Etiqueta texto="Transferencia">
          <Campo value={pagos.transferencia} onChange={poner("transferencia")} inputMode="decimal" />
        </Etiqueta>

        <p className={`text-2xl font-semibold ${diferencia < 0 ? "text-red-600" : "text-gray-800"}`}>
          {diferencia >= 0 ? `Cambio ${pesos(diferencia)}` : `Falta ${pesos(-diferencia)}`}
        </p>
        <Aviso texto={fallo} />

        <div className="flex justify-end gap-3">
          <Boton type="button" onClick={alCerrar}>
            Volver
          </Boton>
          <Boton tipo="primario" disabled={enviando}>
            Confirmar cobro
          </Boton>
        </div>
      </form>
    </Ventana>
  );
}

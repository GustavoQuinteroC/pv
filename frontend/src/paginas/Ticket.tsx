// Archivo: Ticket.tsx
// Qué hace: ticket de una venta, listo para imprimir (58 mm, 80 mm o carta). Al
// llegar del cobro (?imprimir=1) abre solo el diálogo de impresión. Debajo del
// ticket, fuera del papel, hay botones para nueva venta, imprimir y cancelar.

import { Printer } from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
import { Link, useParams, useSearchParams } from "react-router";
import { api, cantidad, fecha, pesos, useApi } from "../lib";
import { useSesion } from "../sesion";
import type { Venta } from "../tipos";
import { Aviso, Boton, Campo, Etiqueta } from "../ui";

export default function Ticket() {
  const { id } = useParams();
  const { negocio, usuario } = useSesion();
  const [parametros, setParametros] = useSearchParams();
  const { datos: venta, error, recargar } = useApi<Venta>(`/ventas/${id}`);
  const [fallo, setFallo] = useState("");

  // Llegando del cobro, imprime una sola vez y limpia el aviso de la dirección.
  useEffect(() => {
    if (venta && parametros.get("imprimir")) {
      setParametros({}, { replace: true });
      window.print();
    }
  }, [venta, parametros, setParametros]);

  async function cancelar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    if (!confirm("¿Cancelar esta venta?")) return;
    try {
      await api(`/ventas/${id}/cancelar`, { motivo: new FormData(evento.currentTarget).get("motivo") });
      await recargar();
    } catch (e) {
      setFallo((e as Error).message);
    }
  }

  if (!venta) return <Aviso texto={error} />;

  return (
    <>
      {/* Esto es lo único que sale en papel. */}
      <article className="mx-auto max-w-[80mm] bg-white p-4 font-mono text-[13px] leading-snug print:max-w-none print:p-0">
        <h1 className="text-center text-lg font-semibold">{negocio.nombre}</h1>
        <p className="mb-2 text-center">
          Folio {venta.folio} · {fecha(venta.fecha)}
          <br />
          Atendió: {venta.cajero_nombre}
        </p>
        {venta.estado === "cancelada" && (
          <p className="mb-2 text-center font-bold">
            VENTA CANCELADA
            <br />
            {venta.motivo_cancelacion}
          </p>
        )}

        <hr className="border-dashed border-gray-800" />
        {venta.renglones.map((r, i) => (
          <div key={i} className="py-0.5">
            <p>{r.nombre}</p>
            <p className="flex justify-between">
              <span>
                {cantidad(r.cantidad)} {r.unidad} × {pesos(r.precio)}
              </span>
              <span>{pesos(r.importe)}</span>
            </p>
          </div>
        ))}

        <hr className="my-1 border-dashed border-gray-800" />
        {venta.descuento > 0 && <Linea texto="Descuento" valor={`-${pesos(venta.descuento)}`} />}
        <Linea texto="Total" valor={pesos(venta.total)} fuerte />
        {venta.pagos.map((p) => (
          <Linea key={p.forma} texto={`Pago en ${p.forma}`} valor={pesos(p.monto)} />
        ))}
        {venta.cambio > 0 && <Linea texto="Cambio" valor={pesos(venta.cambio)} />}

        <hr className="my-2 border-dashed border-gray-800" />
        <p className="text-center">{negocio.mensaje}</p>
      </article>

      {/* Fuera del papel: acciones. */}
      <div className="mx-auto mt-6 max-w-xl space-y-4 print:hidden">
        <Aviso texto={fallo} />
        <div className="flex gap-3">
          <Link to="/caja" className="inline-flex h-10 items-center rounded-[10px] bg-brand-500 px-4 text-sm font-medium text-white hover:bg-brand-600">
            Nueva venta
          </Link>
          <Boton onClick={() => window.print()}>
            <Printer className="size-4" /> Imprimir
          </Boton>
        </div>

        {usuario?.rol === "admin" && venta.estado === "activa" && (
          <form onSubmit={cancelar} className="flex flex-wrap items-end gap-3">
            <Etiqueta texto="Motivo de la cancelación" className="flex-1">
              <Campo name="motivo" required />
            </Etiqueta>
            <Boton tipo="peligro">Cancelar venta</Boton>
          </form>
        )}
      </div>
    </>
  );
}

// Un renglón del ticket con el texto a la izquierda y el valor a la derecha.
const Linea = ({ texto, valor, fuerte }: { texto: string; valor: string; fuerte?: boolean }) => (
  <p className={`flex justify-between ${fuerte ? "font-bold" : ""}`}>
    <span>{texto}</span>
    <span>{valor}</span>
  </p>
);

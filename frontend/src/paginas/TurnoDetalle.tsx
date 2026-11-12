// Archivo: TurnoDetalle.tsx
// Qué hace: detalle de un corte: fondo, efectivo esperado y contado, diferencia,
// resumen de ventas y entradas y retiros.

import { useParams } from "react-router";
import { fecha, pesos, useApi } from "../lib";
import type { Movimiento, Resumen, Turno } from "../tipos";
import { Aviso, Celda, Encabezado, Fila, Tabla, Tarjeta } from "../ui";

// Una cifra grande con su título, para los resúmenes.
export const Cifra = ({ titulo, valor, color = "" }: { titulo: string; valor: string | number; color?: string }) => (
  <div>
    <p className="text-sm text-gray-500">{titulo}</p>
    <p className={`text-2xl font-semibold ${color || "text-gray-800"}`}>{valor}</p>
  </div>
);

export default function TurnoDetalle() {
  const { id } = useParams();
  const { datos, error } = useApi<{ turno: Turno; diferencia: number; resumen: Resumen; movimientos: Movimiento[] }>(`/turnos/${id}`);
  if (!datos) return <Aviso texto={error} />;

  const { turno, diferencia, resumen, movimientos } = datos;
  // Verde si cuadra, ámbar si sobra, rojo si falta.
  const color = diferencia === 0 ? "text-green-600" : diferencia > 0 ? "text-amber-600" : "text-red-600";

  return (
    <>
      <Encabezado titulo={`Corte de ${turno.cajero_nombre}`} />
      <p className="mb-6 text-sm text-gray-500">
        Abierto el {fecha(turno.abierto)}
        {turno.estado === "cerrado" && ` y cerrado el ${fecha(turno.cerrado)}`}.
      </p>

      {turno.estado === "cerrado" ? (
        <Tarjeta className="mb-6">
          <div className="grid gap-6 sm:grid-cols-4">
            <Cifra titulo="Fondo" valor={pesos(turno.fondo)} />
            <Cifra titulo="Efectivo esperado" valor={pesos(turno.esperado)} />
            <Cifra titulo="Efectivo contado" valor={pesos(turno.contado)} />
            <Cifra titulo="Diferencia" valor={pesos(diferencia)} color={color} />
          </div>
        </Tarjeta>
      ) : (
        <Aviso tipo="info" texto="El turno sigue abierto. El corte se muestra al cerrarlo." />
      )}

      <Tarjeta titulo="Ventas del turno" className="mb-6">
        <div className="grid gap-6 sm:grid-cols-4">
          <Cifra titulo="Ventas" valor={resumen.cantidad} />
          <Cifra titulo="Total" valor={pesos(resumen.total)} />
          {Object.entries(resumen.porForma).map(([forma, monto]) => (
            <Cifra key={forma} titulo={forma} valor={pesos(monto)} />
          ))}
        </div>
      </Tarjeta>

      {movimientos.length > 0 && (
        <Tarjeta titulo="Entradas y retiros">
          <Tabla columnas={["Hora", "Tipo", "Monto", "Motivo"]} vacia="">
            {movimientos.map((m) => (
              <Fila key={m.id}>
                <Celda>{fecha(m.fecha)}</Celda>
                <Celda>{m.tipo}</Celda>
                <Celda>{pesos(m.monto)}</Celda>
                <Celda>{m.motivo}</Celda>
              </Fila>
            ))}
          </Tabla>
        </Tarjeta>
      )}
    </>
  );
}

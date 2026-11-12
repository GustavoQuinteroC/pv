// Archivo: Reportes.tsx
// Qué hace: reportes de ventas (solo administración). Muestra las ventas de un
// rango de fechas con totales por forma de pago, lo más vendido y descarga CSV.

import { Download } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { fecha, pesos, useApi } from "../lib";
import type { Resumen, Venta } from "../tipos";
import { Aviso, Campo, Celda, Encabezado, Etiqueta, Fila, Insignia, Tabla, Tarjeta } from "../ui";
import { Cifra } from "./TurnoDetalle";

// Hoy como "2026-11-12", el formato que entiende el campo de fecha.
const hoy = () => new Date().toLocaleDateString("sv");

export default function Reportes() {
  const [desde, setDesde] = useState(hoy());
  const [hasta, setHasta] = useState(hoy());
  const consulta = `desde=${desde}&hasta=${hasta}`;
  const { datos, error } = useApi<{ resumen: Resumen; masVendidos: { nombre: string; importe: number }[]; ventas: Venta[] }>(`/reportes?${consulta}`);

  // Lo más vendido se dibuja como barras: la más larga es el 100 %.
  const maximo = Math.max(1, ...(datos?.masVendidos.map((m) => m.importe) ?? []));

  return (
    <>
      <Encabezado titulo="Reportes">
        <a href={`/api/reportes/csv?${consulta}`} className="inline-flex h-10 items-center gap-2 rounded-[10px] border border-gray-300 bg-white px-4 text-sm font-medium text-gray-700 hover:bg-gray-50">
          <Download className="size-4" /> Descargar CSV
        </a>
      </Encabezado>

      <div className="mb-6 flex flex-wrap gap-4">
        <Etiqueta texto="Desde">
          <Campo type="date" value={desde} onChange={(e) => setDesde(e.target.value)} />
        </Etiqueta>
        <Etiqueta texto="Hasta">
          <Campo type="date" value={hasta} onChange={(e) => setHasta(e.target.value)} />
        </Etiqueta>
      </div>
      <Aviso texto={error} />

      {datos && (
        <>
          <Tarjeta className="mb-6">
            <div className="grid gap-6 sm:grid-cols-4">
              <Cifra titulo="Ventas" valor={datos.resumen.cantidad} />
              <Cifra titulo="Total" valor={pesos(datos.resumen.total)} />
              {Object.entries(datos.resumen.porForma).map(([forma, monto]) => (
                <Cifra key={forma} titulo={forma} valor={pesos(monto)} />
              ))}
            </div>
          </Tarjeta>

          <Tarjeta titulo="Lo más vendido" className="mb-6">
            {datos.masVendidos.length === 0 && <p className="text-sm text-gray-500">Sin ventas en estas fechas.</p>}
            <ul className="space-y-3">
              {datos.masVendidos.map((m) => (
                <li key={m.nombre}>
                  <div className="mb-1 flex justify-between text-sm text-gray-700">
                    <span>{m.nombre}</span>
                    <span>{pesos(m.importe)}</span>
                  </div>
                  <div className="h-2 rounded-full bg-gray-100">
                    <div className="h-2 rounded-full bg-brand-500" style={{ width: `${(m.importe / maximo) * 100}%` }} />
                  </div>
                </li>
              ))}
            </ul>
          </Tarjeta>

          <Tarjeta titulo="Ventas">
            <Tabla columnas={["Folio", "Fecha", "Cajero", "Estado", "Total"]} vacia="Sin ventas en estas fechas.">
              {datos.ventas.map((v) => (
                <Fila key={v.id}>
                  <Celda>
                    <Link className="text-brand-500 hover:underline" to={`/ventas/${v.id}`}>
                      {v.folio}
                    </Link>
                  </Celda>
                  <Celda>{fecha(v.fecha)}</Celda>
                  <Celda>{v.cajero_nombre}</Celda>
                  <Celda>{v.estado === "activa" ? <Insignia color="verde">Activa</Insignia> : <Insignia color="roja">Cancelada</Insignia>}</Celda>
                  <Celda>{pesos(v.total)}</Celda>
                </Fila>
              ))}
            </Tabla>
          </Tarjeta>
        </>
      )}
    </>
  );
}

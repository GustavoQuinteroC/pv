// Archivo: Turnos.tsx
// Qué hace: historial de cortes de caja (solo administración).

import { Link } from "react-router";
import { fecha, pesos, useApi } from "../lib";
import type { Turno } from "../tipos";
import { Aviso, Celda, Encabezado, Fila, Insignia, Tabla, Tarjeta } from "../ui";

export default function Turnos() {
  const { datos: turnos, error } = useApi<Turno[]>("/turnos");

  return (
    <>
      <Encabezado titulo="Cortes de caja" />
      <Aviso texto={error} />
      <Tarjeta>
        <Tabla columnas={["Abierto", "Cajero", "Estado", "Diferencia", ""]} vacia="Todavía no hay turnos.">
          {turnos?.map((t) => (
            <Fila key={t.id}>
              <Celda>{fecha(t.abierto)}</Celda>
              <Celda>{t.cajero_nombre}</Celda>
              <Celda>{t.estado === "abierto" ? <Insignia color="azul">Abierto</Insignia> : <Insignia>Cerrado</Insignia>}</Celda>
              <Celda>{t.estado === "cerrado" && pesos(t.contado - t.esperado)}</Celda>
              <Celda>
                <Link className="text-brand-500 hover:underline" to={`/turnos/${t.id}`}>
                  Ver
                </Link>
              </Celda>
            </Fila>
          ))}
        </Tabla>
      </Tarjeta>
    </>
  );
}

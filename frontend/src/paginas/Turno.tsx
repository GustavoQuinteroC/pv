// Archivo: Turno.tsx
// Qué hace: turno en curso. Registra entradas y retiros de efectivo y cierra el
// turno con el corte ciego (la persona cuenta el efectivo sin ver lo esperado).

import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";
import { aCentavos, api, fecha, pesos, useApi } from "../lib";
import type { Movimiento, Turno as TurnoTipo } from "../tipos";
import { Aviso, Boton, Campo, Celda, Encabezado, Etiqueta, Fila, Seleccion, Tabla, Tarjeta } from "../ui";

export default function Turno() {
  const navegar = useNavigate();
  const { datos, error, recargar } = useApi<{ turno: TurnoTipo; movimientos: Movimiento[] }>("/turno");
  const [fallo, setFallo] = useState("");

  // Sin turno abierto la API responde un aviso: se ofrece ir a abrirlo.
  if (!datos) {
    return (
      <>
        <Aviso texto={error} />
        {error && <Link className="text-brand-500 hover:underline" to="/caja">Ir a la caja</Link>}
      </>
    );
  }

  const { turno, movimientos } = datos;

  async function registrar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    const formulario = evento.currentTarget;
    const d = new FormData(formulario);
    try {
      await api("/turno/movimientos", { tipo: d.get("tipo"), monto: aCentavos(String(d.get("monto"))), motivo: d.get("motivo") });
      formulario.reset();
      setFallo("");
      await recargar();
    } catch (e) {
      setFallo((e as Error).message);
    }
  }

  async function cerrar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    if (!confirm("¿Cerrar el turno? No se podrá vender hasta abrir otro.")) return;
    const d = new FormData(evento.currentTarget);
    try {
      const { id } = await api<{ id: string }>("/turno/cerrar", { contado: aCentavos(String(d.get("contado"))) });
      navegar(`/turnos/${id}`);
    } catch (e) {
      setFallo((e as Error).message);
    }
  }

  return (
    <>
      <Encabezado titulo={`Turno de ${turno.cajero_nombre}`} />
      <p className="mb-6 text-sm text-gray-500">
        Abierto el {fecha(turno.abierto)} con un fondo de {pesos(turno.fondo)}.
      </p>
      <Aviso texto={fallo} />

      <Tarjeta titulo="Entradas y retiros de efectivo" className="mb-6">
        <form onSubmit={registrar} className="mb-4 grid gap-4 sm:grid-cols-4 sm:items-end">
          <Etiqueta texto="Tipo">
            <Seleccion name="tipo">
              <option value="entrada">Entrada</option>
              <option value="retiro">Retiro</option>
            </Seleccion>
          </Etiqueta>
          <Etiqueta texto="Monto">
            <Campo name="monto" inputMode="decimal" required />
          </Etiqueta>
          <Etiqueta texto="Motivo">
            <Campo name="motivo" required />
          </Etiqueta>
          <Boton tipo="primario">Registrar</Boton>
        </form>
        {movimientos.length > 0 && (
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
        )}
      </Tarjeta>

      <Tarjeta titulo="Cerrar turno">
        <p className="mb-4 text-sm text-gray-500">Cuenta el efectivo del cajón y escribe el total. El sistema no te dice cuánto debería haber.</p>
        <form onSubmit={cerrar} className="flex flex-wrap items-end gap-4">
          <Etiqueta texto="Efectivo contado">
            <Campo name="contado" inputMode="decimal" required />
          </Etiqueta>
          <Boton tipo="peligro">Cerrar turno</Boton>
        </form>
      </Tarjeta>
    </>
  );
}

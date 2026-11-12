// Archivo: Ajustes.tsx
// Qué hace: ajustes del negocio (solo administración): nombre y mensaje al final del ticket.

import { useState, type FormEvent } from "react";
import { api } from "../lib";
import { useSesion } from "../sesion";
import { Aviso, Boton, Campo, Encabezado, Etiqueta, Tarjeta } from "../ui";

export default function Ajustes() {
  const { negocio, recargar } = useSesion();
  const [aviso, setAviso] = useState<{ texto: string; tipo: "error" | "exito" }>();

  async function guardar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    const datos = new FormData(evento.currentTarget);
    try {
      await api("/ajustes", { nombre: datos.get("nombre"), mensaje: datos.get("mensaje") }, "PUT");
      await recargar();
      setAviso({ texto: "Guardado.", tipo: "exito" });
    } catch (e) {
      setAviso({ texto: (e as Error).message, tipo: "error" });
    }
  }

  return (
    <>
      <Encabezado titulo="Ajustes" />
      <Tarjeta className="max-w-xl">
        <Aviso {...aviso} />
        <form onSubmit={guardar} className="space-y-4">
          <Etiqueta texto="Nombre del negocio">
            <Campo name="nombre" defaultValue={negocio.nombre} required />
          </Etiqueta>
          <Etiqueta texto="Mensaje al final del ticket">
            <Campo name="mensaje" defaultValue={negocio.mensaje} />
          </Etiqueta>
          <Boton tipo="primario">Guardar</Boton>
        </form>
      </Tarjeta>
    </>
  );
}

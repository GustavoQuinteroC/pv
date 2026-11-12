// Archivo: Usuarios.tsx
// Qué hace: administración de usuarios (solo administración): alta de cuentas
// y activar o desactivar.

import { useState, type FormEvent } from "react";
import { api, useApi } from "../lib";
import { useSesion } from "../sesion";
import type { Usuario } from "../tipos";
import { Aviso, Boton, Campo, Celda, Encabezado, Etiqueta, Fila, Insignia, Seleccion, Tabla, Tarjeta } from "../ui";

export default function Usuarios() {
  const { usuario: yo } = useSesion();
  const { datos: usuarios, error, recargar } = useApi<Usuario[]>("/usuarios");
  const [fallo, setFallo] = useState("");

  // Ejecuta una acción, recarga la lista y muestra el error si lo hay.
  async function intentar(accion: () => Promise<void>) {
    try {
      await accion();
      setFallo("");
      await recargar();
    } catch (e) {
      setFallo((e as Error).message);
    }
  }

  function agregar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    const formulario = evento.currentTarget;
    const d = Object.fromEntries(new FormData(formulario));
    intentar(async () => {
      await api("/usuarios", d);
      formulario.reset();
    });
  }

  return (
    <>
      <Encabezado titulo="Usuarios" />
      <Aviso texto={error || fallo} />

      <Tarjeta titulo="Agregar usuario" className="mb-6">
        <form onSubmit={agregar} className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5 lg:items-end">
          <Etiqueta texto="Nombre">
            <Campo name="nombre" required />
          </Etiqueta>
          <Etiqueta texto="Correo">
            <Campo name="correo" type="email" required />
          </Etiqueta>
          <Etiqueta texto="Contraseña (mínimo 8)">
            <Campo name="clave" type="password" minLength={8} autoComplete="new-password" required />
          </Etiqueta>
          <Etiqueta texto="Rol">
            <Seleccion name="rol">
              <option value="cajero">Cajero</option>
              <option value="admin">Administrador</option>
            </Seleccion>
          </Etiqueta>
          <Boton tipo="primario">Agregar</Boton>
        </form>
      </Tarjeta>

      <Tarjeta>
        <Tabla columnas={["Nombre", "Correo", "Rol", "Estado", ""]} vacia="Sin usuarios.">
          {usuarios?.map((u) => (
            <Fila key={u.id}>
              <Celda>{u.nombre}</Celda>
              <Celda>{u.correo}</Celda>
              <Celda>{u.rol}</Celda>
              <Celda>{u.activo ? <Insignia color="verde">Activo</Insignia> : <Insignia>Inactivo</Insignia>}</Celda>
              <Celda>
                {u.id !== yo?.id && (
                  <Boton onClick={() => intentar(() => api(`/usuarios/${u.id}/activo`, { activo: !u.activo }))}>
                    {u.activo ? "Desactivar" : "Activar"}
                  </Boton>
                )}
              </Celda>
            </Fila>
          ))}
        </Tabla>
      </Tarjeta>
    </>
  );
}

// Archivo: Login.tsx
// Qué hace: pantalla de inicio de sesión (correo y contraseña).

import { useState, type FormEvent } from "react";
import { Navigate, useNavigate } from "react-router";
import { api } from "../lib";
import { useSesion } from "../sesion";
import { Aviso, Boton, Campo, Etiqueta } from "../ui";

export default function Login() {
  const { usuario, negocio, recargar } = useSesion();
  const navegar = useNavigate();
  const [error, setError] = useState("");

  // Si ya hay sesión, no tiene caso ver el login.
  if (usuario) return <Navigate to="/caja" replace />;

  async function entrar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    const datos = new FormData(evento.currentTarget);
    try {
      await api("/login", { correo: datos.get("correo"), clave: datos.get("clave") });
      await recargar();
      navegar("/caja");
    } catch (e) {
      setError((e as Error).message);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <form onSubmit={entrar} className="w-full max-w-sm space-y-4 rounded-2xl border border-gray-200 bg-white p-6 sm:p-8">
        <h1 className="text-2xl font-semibold text-gray-800">{negocio.nombre}</h1>
        <p className="text-sm text-gray-500">Entra con tu correo y contraseña.</p>
        <Aviso texto={error} />
        <Etiqueta texto="Correo">
          <Campo name="correo" type="email" autoComplete="username" required autoFocus />
        </Etiqueta>
        <Etiqueta texto="Contraseña">
          <Campo name="clave" type="password" autoComplete="current-password" required />
        </Etiqueta>
        <Boton tipo="primario" className="w-full">
          Entrar
        </Boton>
      </form>
    </div>
  );
}

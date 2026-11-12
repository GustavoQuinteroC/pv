// Archivo: sesion.tsx
// Qué hace: pregunta una vez a la API quién tiene la sesión y cuáles son los
// datos del negocio, y los comparte con todas las pantallas (useSesion).

import { createContext, useContext, type ReactNode } from "react";
import { useApi } from "./lib";
import type { Negocio, Usuario } from "./tipos";

interface Sesion {
  usuario: Usuario | null;
  negocio: Negocio;
  recargar: () => Promise<void>;
}

const Contexto = createContext<Sesion | null>(null);

export function ProveedorSesion({ children }: { children: ReactNode }) {
  const { datos, recargar } = useApi<Omit<Sesion, "recargar">>("/yo");
  // Mientras llega la respuesta no se pinta nada, para no parpadear entre login y sistema.
  if (!datos) return null;
  return <Contexto value={{ ...datos, recargar }}>{children}</Contexto>;
}

export const useSesion = () => useContext(Contexto)!;

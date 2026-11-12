// Archivo: lib.ts
// Qué hace: utilidades que usan todas las pantallas: hablar con la API de Go,
// cargar datos, guardar en el navegador y dar formato a dinero, cantidades y fechas.

import { useCallback, useEffect, useState } from "react";

// ---------- API ----------

// Llama a la API de Go. Sin cuerpo es GET; con cuerpo es POST (o el método indicado).
// Si el servidor responde un error, lanza un Error con su mensaje en español.
export async function api<T = void>(ruta: string, cuerpo?: unknown, metodo?: string): Promise<T> {
  const respuesta = await fetch(`/api${ruta}`, {
    method: metodo ?? (cuerpo === undefined ? "GET" : "POST"),
    headers: cuerpo === undefined ? undefined : { "Content-Type": "application/json" },
    body: cuerpo === undefined ? undefined : JSON.stringify(cuerpo),
  });
  // 204 = salió bien y no hay nada que devolver.
  if (respuesta.status === 204) return undefined as T;

  const datos = await respuesta.json().catch(() => ({}));
  if (!respuesta.ok) throw new Error(datos.error ?? "Algo salió mal. Intenta de nuevo.");
  return datos;
}

// Carga datos de la API al abrir la pantalla. `recargar` vuelve a pedirlos.
export function useApi<T>(ruta: string) {
  const [datos, setDatos] = useState<T>();
  const [error, setError] = useState("");

  const recargar = useCallback(
    () =>
      api<T>(ruta)
        .then((d) => {
          setDatos(d);
          setError("");
        })
        .catch((e: Error) => setError(e.message)),
    [ruta],
  );

  useEffect(() => {
    recargar();
  }, [recargar]);

  return { datos, error, recargar };
}

// Como useState, pero el valor se guarda en el navegador y sobrevive a recargar la página.
export function useGuardado<T>(clave: string, inicial: T) {
  const [valor, setValor] = useState<T>(() => {
    try {
      return JSON.parse(localStorage.getItem(clave) ?? "") ?? inicial;
    } catch {
      return inicial;
    }
  });

  useEffect(() => localStorage.setItem(clave, JSON.stringify(valor)), [clave, valor]);
  return [valor, setValor] as const;
}

// ---------- Formato ----------

// 12345 centavos → "$123.45"
export const pesos = (centavos: number) =>
  (centavos / 100).toLocaleString("es-MX", { style: "currency", currency: "MXN" });

// "12.50" → 1250 centavos. Texto inválido o vacío → 0.
export const aCentavos = (texto: string) => Math.round(parseFloat(texto.replaceAll(",", "")) * 100) || 0;

// "1.5" → 1500 milésimas. Texto inválido o vacío → 0.
export const aMilesimas = (texto: string) => Math.round(parseFloat(texto.replaceAll(",", "")) * 1000) || 0;

// 1250 milésimas → "1.25"
export const cantidad = (milesimas: number) => String(milesimas / 1000);

// "2026-11-12T15:30:00Z" → "12/11/26, 9:30"
export const fecha = (iso: string) =>
  new Date(iso).toLocaleString("es-MX", { dateStyle: "short", timeStyle: "short" });

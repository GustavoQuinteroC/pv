// Archivo: Layout.tsx
// Qué hace: el marco de todas las pantallas con sesión, al estilo TailAdmin:
// menú lateral (se oculta en celular), barra superior con la persona y botón de
// salir. Si no hay sesión, manda al login. Al imprimir se oculta todo el marco.

import { Clock, LogOut, Menu, Package, Receipt, Settings, ShoppingCart, TrendingUp, Users } from "lucide-react";
import { useState } from "react";
import { Navigate, NavLink, Outlet } from "react-router";
import { api } from "./lib";
import { useSesion } from "./sesion";

// Opciones del menú. `admin` = solo las ve la administración.
const opciones = [
  { a: "/caja", texto: "Caja", icono: ShoppingCart },
  { a: "/turno", texto: "Turno", icono: Clock },
  { a: "/productos", texto: "Productos", icono: Package, admin: true },
  { a: "/reportes", texto: "Reportes", icono: TrendingUp, admin: true },
  { a: "/turnos", texto: "Cortes", icono: Receipt, admin: true },
  { a: "/usuarios", texto: "Usuarios", icono: Users, admin: true },
  { a: "/ajustes", texto: "Ajustes", icono: Settings, admin: true },
];

export default function Layout() {
  const { usuario, negocio, recargar } = useSesion();
  const [menuAbierto, setMenuAbierto] = useState(false);

  if (!usuario) return <Navigate to="/login" replace />;

  async function salir() {
    await api("/salir", {});
    await recargar();
  }

  return (
    <div className="min-h-screen">
      {/* Menú lateral: fijo en pantallas grandes, se desliza en celular. */}
      <aside
        className={`fixed inset-y-0 left-0 z-50 w-64 border-r border-gray-200 bg-white px-4 py-6 transition-transform print:hidden lg:translate-x-0 ${menuAbierto ? "" : "-translate-x-full"}`}
      >
        <p className="mb-8 px-3 text-xl font-semibold text-gray-800">{negocio.nombre}</p>
        <nav className="space-y-1">
          {opciones
            .filter((o) => !o.admin || usuario.rol === "admin")
            .map(({ a, texto, icono: Icono }) => (
              <NavLink
                key={a}
                to={a}
                onClick={() => setMenuAbierto(false)}
                className={({ isActive }) =>
                  `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium ${isActive ? "bg-brand-50 text-brand-500" : "text-gray-700 hover:bg-gray-100"}`
                }
              >
                <Icono className="size-5" />
                {texto}
              </NavLink>
            ))}
        </nav>
      </aside>

      <div className="lg:pl-64 print:pl-0">
        {/* Barra superior. */}
        <header className="sticky top-0 z-40 flex items-center justify-between border-b border-gray-200 bg-white px-4 py-3 print:hidden sm:px-6">
          <button className="rounded-lg border border-gray-200 p-2 lg:hidden" onClick={() => setMenuAbierto(!menuAbierto)} aria-label="Menú">
            <Menu className="size-5" />
          </button>
          <span className="ml-auto mr-4 text-sm text-gray-600">{usuario.nombre}</span>
          <button className="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50" onClick={salir}>
            <LogOut className="size-4" /> Salir
          </button>
        </header>

        <main className="mx-auto max-w-7xl p-4 sm:p-6 print:p-0">
          <Outlet />
        </main>
      </div>
    </div>
  );
}

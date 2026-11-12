// Archivo: App.tsx
// Qué hace: lista qué pantalla (componente) se muestra en cada dirección.
// Todas, menos el login, van dentro del marco de Layout.tsx.

import { Navigate, Route, Routes } from "react-router";
import Layout from "./Layout";
import Ajustes from "./paginas/Ajustes";
import Caja from "./paginas/Caja";
import Login from "./paginas/Login";
import Productos from "./paginas/Productos";
import Reportes from "./paginas/Reportes";
import Ticket from "./paginas/Ticket";
import Turno from "./paginas/Turno";
import TurnoDetalle from "./paginas/TurnoDetalle";
import Turnos from "./paginas/Turnos";
import Usuarios from "./paginas/Usuarios";

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route element={<Layout />}>
        <Route path="/" element={<Navigate to="/caja" replace />} />
        <Route path="/caja" element={<Caja />} />
        <Route path="/turno" element={<Turno />} />
        <Route path="/turnos" element={<Turnos />} />
        <Route path="/turnos/:id" element={<TurnoDetalle />} />
        <Route path="/ventas/:id" element={<Ticket />} />
        <Route path="/productos" element={<Productos />} />
        <Route path="/reportes" element={<Reportes />} />
        <Route path="/usuarios" element={<Usuarios />} />
        <Route path="/ajustes" element={<Ajustes />} />
      </Route>
    </Routes>
  );
}

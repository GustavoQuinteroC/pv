// Archivo: main.tsx
// Qué hace: punto de entrada de React. Monta la aplicación dentro del <div id="raiz">
// de index.html, con el enrutador (direcciones) y la sesión disponibles para todos.

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router";
import App from "./App";
import "./index.css";
import { ProveedorSesion } from "./sesion";

createRoot(document.getElementById("raiz")!).render(
  <StrictMode>
    <BrowserRouter>
      <ProveedorSesion>
        <App />
      </ProveedorSesion>
    </BrowserRouter>
  </StrictMode>,
);

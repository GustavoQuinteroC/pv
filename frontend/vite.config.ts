// Archivo: vite.config.ts
// Qué hace: configura Vite, la herramienta que compila la interfaz React.
// - Al compilar, deja el resultado en ../internal/web/dist (Go lo incrusta en el programa).
// - En desarrollo (`npm run dev`) reenvía /api al servidor de Go, para no tener dos orígenes.

import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: { outDir: "../internal/web/dist", emptyOutDir: true },
  server: { proxy: { "/api": "http://localhost:8080" } },
});

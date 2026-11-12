// Archivo: Productos.tsx
// Qué hace: catálogo de productos (solo administración). Lista los productos y
// permite crear o editar uno en una ventana emergente.

import { Plus } from "lucide-react";
import { useState, type FormEvent } from "react";
import { api, aCentavos, pesos, useApi } from "../lib";
import type { Producto } from "../tipos";
import { Aviso, Boton, Campo, Casilla, Celda, Encabezado, Etiqueta, Fila, Insignia, Seleccion, Tabla, Tarjeta, Ventana } from "../ui";

// Un producto nuevo empieza por piezas y activo.
const nuevo = { id: "", nombre: "", codigo: "", precio: 0, unidad: "pieza", favorito: false, activo: true } as Producto;

export default function Productos() {
  const { datos: productos, error, recargar } = useApi<Producto[]>("/productos");
  // El producto que se está editando en la ventana (null = ventana cerrada).
  const [editando, setEditando] = useState<Producto | null>(null);
  const [fallo, setFallo] = useState("");

  async function guardar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    const d = new FormData(evento.currentTarget);
    const producto = {
      nombre: d.get("nombre"),
      codigo: d.get("codigo"),
      precio: aCentavos(String(d.get("precio"))),
      unidad: d.get("unidad"),
      favorito: d.get("favorito") === "on",
      activo: d.get("activo") === "on",
    };

    try {
      // Con id se edita (PUT); sin id se crea (POST).
      if (editando!.id) await api(`/productos/${editando!.id}`, producto, "PUT");
      else await api("/productos", producto);
      setEditando(null);
      setFallo("");
      await recargar();
    } catch (e) {
      setFallo((e as Error).message);
    }
  }

  return (
    <>
      <Encabezado titulo="Productos">
        <Boton tipo="primario" onClick={() => setEditando(nuevo)}>
          <Plus className="size-4" /> Nuevo producto
        </Boton>
      </Encabezado>
      <Aviso texto={error} />

      <Tarjeta>
        <Tabla columnas={["Nombre", "Código", "Precio", "Estado", ""]} vacia="Todavía no hay productos.">
          {productos?.map((p) => (
            <Fila key={p.id}>
              <Celda>
                {p.nombre} {p.favorito && "★"}
              </Celda>
              <Celda>{p.codigo}</Celda>
              <Celda>
                {pesos(p.precio)} / {p.unidad}
              </Celda>
              <Celda>{p.activo ? <Insignia color="verde">Activo</Insignia> : <Insignia>Inactivo</Insignia>}</Celda>
              <Celda>
                <Boton onClick={() => setEditando(p)}>Editar</Boton>
              </Celda>
            </Fila>
          ))}
        </Tabla>
      </Tarjeta>

      <Ventana abierta={editando !== null} alCerrar={() => setEditando(null)} titulo={editando?.id ? "Editar producto" : "Nuevo producto"}>
        <form onSubmit={guardar} className="space-y-4">
          <Aviso texto={fallo} />
          <Etiqueta texto="Nombre">
            <Campo name="nombre" defaultValue={editando?.nombre} required autoFocus />
          </Etiqueta>
          <Etiqueta texto="Código de barras (opcional)">
            <Campo name="codigo" defaultValue={editando?.codigo} inputMode="numeric" />
          </Etiqueta>
          <Etiqueta texto="Precio por unidad">
            <Campo name="precio" defaultValue={editando?.precio ? (editando.precio / 100).toFixed(2) : ""} inputMode="decimal" required />
          </Etiqueta>
          <Etiqueta texto="Se vende por">
            <Seleccion name="unidad" defaultValue={editando?.unidad}>
              <option value="pieza">pieza</option>
              <option value="kg">kg</option>
              <option value="litro">litro</option>
            </Seleccion>
          </Etiqueta>
          <Casilla name="favorito" texto="Mostrar entre los favoritos de la caja" defaultChecked={editando?.favorito} />
          <Casilla name="activo" texto="Activo (si se desmarca, ya no se puede vender)" defaultChecked={editando?.activo} />
          <div className="flex justify-end gap-3">
            <Boton type="button" onClick={() => setEditando(null)}>
              Cancelar
            </Boton>
            <Boton tipo="primario">Guardar</Boton>
          </div>
        </form>
      </Ventana>
    </>
  );
}

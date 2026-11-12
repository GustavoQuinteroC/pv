// Archivo: ui.tsx
// Qué hace: los componentes visuales reutilizables, con el estilo de TailAdmin:
// botón, campo de texto, tarjeta, aviso, etiqueta, tabla y ventana emergente.

import type { ComponentProps, ReactNode } from "react";
import { useEffect, useRef } from "react";

// Une nombres de clases CSS ignorando los vacíos.
const unir = (...clases: (string | false | undefined)[]) => clases.filter(Boolean).join(" ");

// ---------- Botón ----------

const estilosBoton = {
  primario: "bg-brand-500 text-white shadow-theme-xs hover:bg-brand-600 disabled:bg-brand-300",
  contorno: "border border-gray-300 bg-white text-gray-700 shadow-theme-xs hover:bg-gray-50",
  peligro: "bg-red-500 text-white shadow-theme-xs hover:bg-red-600 disabled:bg-red-300",
};

export function Boton({ tipo = "contorno", className, ...props }: ComponentProps<"button"> & { tipo?: keyof typeof estilosBoton }) {
  return (
    <button
      className={unir(
        "inline-flex h-10 items-center justify-center gap-2 rounded-[10px] px-4 text-sm font-medium transition disabled:cursor-not-allowed",
        estilosBoton[tipo],
        className,
      )}
      {...props}
    />
  );
}

// ---------- Campos ----------

const estiloCampo =
  "h-11 w-full rounded-[10px] border border-gray-300 bg-white px-4 text-sm text-gray-800 shadow-theme-xs outline-hidden placeholder:text-gray-400 focus:border-brand-300 focus:shadow-focus-ring";

// Etiqueta con su campo debajo. `children` es el <Campo> o <Seleccion>.
export function Etiqueta({ texto, children, className }: { texto: string; children: ReactNode; className?: string }) {
  return (
    <label className={unir("block", className)}>
      <span className="mb-1.5 block text-sm font-medium text-gray-700">{texto}</span>
      {children}
    </label>
  );
}

export const Campo = (props: ComponentProps<"input">) => <input className={estiloCampo} {...props} />;

export const Seleccion = (props: ComponentProps<"select">) => <select className={estiloCampo} {...props} />;

// Casilla de verificación con su texto.
export function Casilla({ texto, ...props }: ComponentProps<"input"> & { texto: string }) {
  return (
    <label className="flex items-center gap-3 text-sm text-gray-700">
      <input type="checkbox" className="size-5 rounded-md border-gray-300 accent-brand-500" {...props} />
      {texto}
    </label>
  );
}

// ---------- Contenedores ----------

export function Tarjeta({ titulo, children, className }: { titulo?: string; children: ReactNode; className?: string }) {
  return (
    <section className={unir("rounded-2xl border border-gray-200 bg-white p-5 sm:p-6", className)}>
      {titulo && <h2 className="mb-4 text-lg font-semibold text-gray-800">{titulo}</h2>}
      {children}
    </section>
  );
}

// Encabezado de pantalla: título a la izquierda y acciones a la derecha.
export function Encabezado({ titulo, children }: { titulo: string; children?: ReactNode }) {
  return (
    <div className="mb-6 flex flex-wrap items-center justify-between gap-3 print:hidden">
      <h1 className="text-title-sm text-xl font-semibold text-gray-800">{titulo}</h1>
      {children}
    </div>
  );
}

const estilosAviso = {
  error: "border-red-500 bg-red-50 text-red-700",
  exito: "border-green-500 bg-green-50 text-green-700",
  info: "border-sky-500 bg-sky-50 text-sky-700",
};

// Mensaje de aviso. Sin texto no muestra nada, así se puede usar directo con un estado.
export function Aviso({ texto, tipo = "error" }: { texto?: string; tipo?: keyof typeof estilosAviso }) {
  if (!texto) return null;
  return (
    <p role="alert" className={unir("mb-4 rounded-xl border px-4 py-3 text-sm", estilosAviso[tipo])}>
      {texto}
    </p>
  );
}

const estilosInsignia = {
  verde: "bg-green-50 text-green-600",
  roja: "bg-red-50 text-red-600",
  ambar: "bg-amber-50 text-amber-600",
  gris: "bg-gray-100 text-gray-600",
  azul: "bg-brand-50 text-brand-500",
};

export function Insignia({ color = "gris", children }: { color?: keyof typeof estilosInsignia; children: ReactNode }) {
  return <span className={unir("inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium", estilosInsignia[color])}>{children}</span>;
}

// ---------- Tabla ----------

// Tabla con encabezados. Cada fila la escribe quien la usa con <Fila> y <Celda>.
export function Tabla({ columnas, vacia, children }: { columnas: string[]; vacia: string; children: ReactNode[] | ReactNode }) {
  const hayFilas = Array.isArray(children) ? children.length > 0 : Boolean(children);
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-left text-sm">
        <thead>
          <tr className="border-b border-gray-100 text-xs text-gray-500">
            {columnas.map((c) => (
              <th key={c} className="px-3 py-3 font-medium">
                {c}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {children}
          {!hayFilas && (
            <tr>
              <td colSpan={columnas.length} className="px-3 py-6 text-center text-gray-500">
                {vacia}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

export const Fila = (props: ComponentProps<"tr">) => <tr {...props} />;
export const Celda = ({ className, ...props }: ComponentProps<"td">) => <td className={unir("px-3 py-3 text-gray-700", className)} {...props} />;

// ---------- Ventana emergente ----------

// Ventana nativa <dialog>: el navegador se encarga del fondo oscuro y de la tecla Esc.
export function Ventana({ abierta, alCerrar, titulo, children }: { abierta: boolean; alCerrar: () => void; titulo: string; children: ReactNode }) {
  const dialogo = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const d = dialogo.current!;
    if (abierta && !d.open) d.showModal();
    if (!abierta && d.open) d.close();
  }, [abierta]);

  return (
    <dialog ref={dialogo} onClose={alCerrar} className="m-auto w-[min(30rem,94vw)] rounded-2xl p-6 shadow-theme-lg backdrop:bg-gray-900/50">
      {abierta && (
        <>
          <h2 className="mb-4 text-lg font-semibold text-gray-800">{titulo}</h2>
          {children}
        </>
      )}
    </dialog>
  );
}

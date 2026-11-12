// Archivo: tipos.ts
// Qué hace: describe la forma de los datos que manda la API de Go (JSON).
// Dinero en centavos; cantidades en milésimas; fechas como texto ISO.

export interface Usuario {
  id: string;
  nombre: string;
  correo: string;
  rol: "admin" | "cajero";
  activo: boolean;
}

export interface Negocio {
  nombre: string;
  mensaje: string;
}

export interface Producto {
  id: string;
  nombre: string;
  codigo: string;
  precio: number;
  unidad: "pieza" | "kg" | "litro";
  favorito: boolean;
  activo: boolean;
}

export interface Renglon {
  producto: string;
  nombre: string;
  unidad: string;
  precio: number;
  cantidad: number;
  importe: number;
}

export interface Pago {
  forma: "efectivo" | "tarjeta" | "transferencia";
  monto: number;
}

export interface Venta {
  id: string;
  folio: number;
  cajero_nombre: string;
  renglones: Renglon[];
  descuento: number;
  total: number;
  pagos: Pago[];
  cambio: number;
  estado: "activa" | "cancelada";
  motivo_cancelacion: string;
  fecha: string;
}

export interface Turno {
  id: string;
  cajero: string;
  cajero_nombre: string;
  fondo: number;
  estado: "abierto" | "cerrado";
  abierto: string;
  cerrado: string;
  contado: number;
  esperado: number;
}

export interface Movimiento {
  id: string;
  tipo: "entrada" | "retiro";
  monto: number;
  motivo: string;
  fecha: string;
}

export interface Resumen {
  cantidad: number;
  total: number;
  porForma: Record<string, number>;
}

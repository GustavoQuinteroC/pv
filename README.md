# pv · Punto de venta sencillo

Punto de venta para tiendas, cremerías, carnicerías, tortillerías, dulcerías y negocios parecidos que no quieren pagar un sistema avanzado. Hecho con **Go** y **MongoDB**, sin frameworks.

Una caja, un negocio, sin inventario y sin facturación: lo necesario para vender, cobrar y cuadrar la caja.

**Demo:** https://pv.gustavoquintero.dev

## Qué hace

- **Venta rápida.** El carrito vive en el navegador: escanear o tocar un producto es instantáneo. El servidor recibe la venta completa solo al cobrar.
- **Productos por pieza o a granel** (kg y litro). Cantidades decimales y venta por monto: «$20 de tortilla» calcula los kilos.
- **Cobro** en efectivo, tarjeta, transferencia o mezclando formas de pago, con cambio.
- **Tickets en espera**, descuento y cancelación con motivo.
- **Ticket imprimible** por el diálogo del navegador (58 mm, 80 mm o carta).
- **Turno y corte ciego.** Se abre con fondo, admite entradas y retiros de efectivo, y al cerrar la persona cuenta sin ver cuánto debería haber. Después se muestra la diferencia.
- **Reportes** por rango de fechas, lo más vendido y descarga en CSV.
- Dos roles: **administrador** y **cajero**.

## Cómo está hecho

| Pieza | Elección |
|---|---|
| Servidor | Go (biblioteca estándar: `net/http`, `log/slog`) con una API JSON |
| Base de datos | MongoDB con el driver oficial v2 |
| Interfaz | React 19 + TypeScript + Tailwind CSS 4, con el diseño de [TailAdmin](https://github.com/TailAdmin/free-react-tailwind-admin-dashboard) (licencia MIT) |
| Contraseñas | `bcrypt`, con bloqueo tras 5 intentos fallidos |
| Despliegue | Una imagen Docker de un solo binario (`distroless`): la interfaz compilada va incrustada en el programa de Go |

### Decisiones de diseño

- **El dinero nunca es decimal.** Los precios van en centavos y las cantidades en milésimas (`1250` = 1.250 kg), ambos en `int64`. No hay errores de redondeo al sumar.
- **Una venta es un solo documento.** Sus renglones y sus pagos van dentro, así que guardarla es una operación atómica y no hace falta una transacción.
- **Las reglas las hacen cumplir los índices de MongoDB**, no solo el código:
  - índice único parcial sobre `turnos.estado = "abierto"`: nunca hay dos turnos abiertos;
  - índice único sobre `ventas.token`: un doble clic en «Cobrar» no cobra dos veces;
  - índice único sobre `ventas.folio` y sobre el correo de cada usuario.
- **El servidor no confía en el navegador.** La pantalla manda productos y cantidades; los precios, los totales y el cambio los calcula el servidor con el catálogo.
- **Cada renglón guarda el nombre y el precio que tenía al venderse.** Cambiar el catálogo no altera ventas pasadas.
- **Cancelar es una sola operación condicional** (`UpdateOne` con filtro por estado y turno), así dos cancelaciones simultáneas no se pisan.
- **El efectivo esperado se calcula** con el fondo, el efectivo cobrado sin el cambio, las entradas y los retiros. Las ventas con tarjeta, las transferencias y las canceladas no entran.
- **Seguridad:** sesiones en MongoDB que caducan solas, cookie `HttpOnly` y `SameSite`, protección CSRF de la biblioteca estándar (`http.CrossOriginProtection`), política de contenido que no permite scripts en línea.

## Estructura

```
cmd/pv/            arranque del servidor
internal/
  config/          variables de entorno
  bd/              conexión e índices de MongoDB
  aviso/           errores que sí se le muestran a la persona usuaria
  usuario/         cuentas, acceso y sesiones
  producto/        catálogo
  venta/           totales, cobro, cancelación y resumen (con pruebas)
  caja/            turnos, movimientos y corte (con pruebas)
  negocio/         nombre y mensaje del ticket
  semilla/         primer administrador y productos de ejemplo
  web/             API JSON y entrega de la interfaz compilada (dist/)
frontend/          interfaz React (src/paginas = una pantalla por archivo)
```

## Desarrollar la interfaz

Necesitas Go, Node 24 y una MongoDB. Se trabaja con dos procesos: el servidor de Go y el de Vite, que recarga la página al guardar y reenvía `/api` a Go.

```bash
go run ./cmd/pv                        # API en http://localhost:8080 (compila antes la interfaz una vez)
cd frontend && npm ci && npm run dev   # interfaz en http://localhost:5173
```

`npm run build` deja la interfaz en `internal/web/dist`, que Go incrusta al compilar.

## Probarlo en tu equipo

Necesitas Docker.

```bash
cp .env.example .env     # cambia las contraseñas
docker compose up --build
```

Abre http://localhost:8080 y entra con el correo y la contraseña de `ADMIN_CORREO` y `ADMIN_CLAVE`. Con `SEMBRAR_DEMO=true` se cargan productos de ejemplo la primera vez.

## Pruebas

Las pruebas de cálculo corren sin base de datos:

```bash
go test ./...
```

Las de integración usan una base de MongoDB desechable y se saltan solas si no la defines:

```bash
MONGO_URI_PRUEBAS=mongodb://localhost:27017 go test ./...
```

## Variables de entorno

| Variable | Para qué sirve | Por omisión |
|---|---|---|
| `MONGO_URI` | Conexión a MongoDB | `mongodb://localhost:27017` |
| `MONGO_BD` | Nombre de la base | `pv` |
| `PUERTO` | Puerto del servidor | `8080` |
| `ADMIN_CORREO`, `ADMIN_CLAVE` | Primer administrador (solo si no hay usuarios) | vacío |
| `SEMBRAR_DEMO` | `true` carga productos de ejemplo si el catálogo está vacío | `false` |

## Fuera de alcance

Facturación, inventario, varias cajas o sucursales, trabajo sin conexión y promociones. Es un MVP a propósito.

## Licencia

MIT.

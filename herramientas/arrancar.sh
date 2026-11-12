#!/bin/sh
# Archivo: arrancar.sh
# Qué hace: script de arranque para Coolify. La imagen es golang y la carpeta
# del proyecto está montada en /app; este script prepara las dependencias y
# ejecuta el programa.
# La interfaz React ya debe estar compilada en internal/web/dist (cd frontend && npm run build);
# esta imagen solo trae Go. go mod tidy genera go.sum si falta; la caché de módulos vive en un volumen.

# Detiene el script en cuanto un comando falle.
set -e
# Entra a la carpeta del proyecto.
cd /app
# Ordena y descarga las dependencias.
go mod tidy
# Reemplaza este proceso por el programa (así recibe bien las señales de apagado).
exec go run ./cmd/pv

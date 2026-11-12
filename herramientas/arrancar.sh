#!/bin/sh
# Arranque en Coolify: la imagen es golang y la carpeta del proyecto está montada en /app.
# go mod tidy genera go.sum si falta; la caché de módulos vive en un volumen.
set -e
cd /app
go mod tidy
exec go run ./cmd/pv

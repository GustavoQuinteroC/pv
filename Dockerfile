# Archivo: Dockerfile
# Qué hace: receta para construir la imagen de Docker de producción. Compila la
# interfaz React con Node, compila el programa con Go y copia solo el binario a una imagen final
# mínima, así la imagen resultante es pequeña y segura.

# ---- Etapa 1: compilar la interfaz React con Node. Genera internal/web/dist. ----
FROM node:24 AS interfaz
# La carpeta se llama igual que en el repo, porque Vite escribe en ../internal/web/dist.
WORKDIR /app/frontend
# Dependencias primero, para que Docker las guarde en caché.
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend ./
# Revisa los tipos con TypeScript y empaqueta con Vite.
RUN npx tsc && npx vite build

# ---- Etapa 2: construir el programa. Imagen con Go y herramientas de compilación. ----
FROM golang:1.25 AS construir
# Carpeta de trabajo dentro de la imagen.
WORKDIR /src

# Las dependencias van primero para que Docker las guarde en caché.
COPY go.mod go.sum ./
# Descarga las dependencias (solo se repite si cambian go.mod o go.sum).
RUN go mod download

# Copia el resto del código.
COPY . .
# Trae la interfaz ya compilada; Go la incrusta dentro del programa.
COPY --from=interfaz /app/internal/web/dist ./internal/web/dist
# Compila un binario estático (CGO desactivado), sin rutas locales (-trimpath) y reducido (-s -w).
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /pv ./cmd/pv

# ---- Etapa 3: imagen final. Solo trae el binario: sin shell, sin compilador, sin root. ----
FROM gcr.io/distroless/static-debian12:nonroot
# Copia únicamente el programa ya compilado de la etapa anterior.
COPY --from=construir /pv /pv

# Zona horaria de los tickets y reportes.
ENV TZ=America/Mexico_City
# El programa escucha en el puerto 8080.
EXPOSE 8080
# Comando que se ejecuta al iniciar el contenedor.
ENTRYPOINT ["/pv"]

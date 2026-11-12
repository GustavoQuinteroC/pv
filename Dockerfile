FROM golang:1.25 AS construir
WORKDIR /src

# Las dependencias van primero para que Docker las guarde en caché.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /pv ./cmd/pv

# La imagen final solo trae el binario: sin shell, sin compilador, sin root.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=construir /pv /pv

ENV TZ=America/Mexico_City
EXPOSE 8080
ENTRYPOINT ["/pv"]

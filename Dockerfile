# Build stage
FROM golang:1.23.2-alpine AS builder

# Instalar dependencias necesarias para SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copiar archivos de dependencias
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar el código fuente
COPY . .

# Compilar la aplicación
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .

# Runtime stage
FROM alpine:latest

# Instalar dependencias de runtime
RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /root/

# Copiar el binario compilado desde el builder
COPY --from=builder /app/main .

# Copiar las plantillas HTML
COPY --from=builder /app/templates ./templates

# Exponer el puerto 8081
EXPOSE 8081

# Configurar la variable de entorno PORT
ENV PORT=8081

# Ejecutar la aplicación
CMD ["./main"]

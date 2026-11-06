# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Dependencias (cacheadas mientras go.mod/go.sum no cambien)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# pgx es Go puro: no se necesita CGO
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

# Runtime stage
FROM alpine:latest

# ca-certificates: TLS hacia Supabase; tzdata: fechas con zona horaria local
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

ENV PORT=8081
EXPOSE 8081

CMD ["./main"]

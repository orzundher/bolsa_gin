# 🚀 Despliegue con Docker

La aplicación se despliega con Docker Compose. La base de datos es PostgreSQL (Supabase), por lo que el contenedor no guarda datos y no necesita volúmenes.

## Requisitos

- Docker y Docker Compose.
- Un archivo `.env` junto a `docker-compose.yml` (no se versiona):

```
DATABASE_URL=postgresql://user:password@host:port/database
AUTH_USER=admin
AUTH_PASSWORD=change-me
AUTH_SECRET=a-long-random-secret-for-cookie-signing
```

- El **servicio de precios** (fuera de este repositorio) corriendo en el host en el puerto `3010`. El contenedor lo alcanza vía `PRECIOS_BASE_URL=http://host.docker.internal:3010` (definido en `docker-compose.yml`). Sin él fallan la actualización de precios y `/api/fetch-price/*`.

## Desplegar / actualizar

En el servidor, tras actualizar el código:

```bash
./build.sh
```

Equivale a:

```bash
docker compose down
docker compose build --no-cache
docker compose up -d
```

La aplicación queda publicada en el puerto **80** del host (80 → 8081 del contenedor). Con `restart: unless-stopped` se reinicia sola tras un reinicio del servidor.

## Comandos útiles

```bash
docker compose logs -f bolsa_gin   # ver logs
docker compose ps                  # estado
docker compose restart bolsa_gin   # reiniciar
docker compose down                # detener y eliminar el contenedor
```

## Ejecución sin Compose (pruebas)

```bash
docker build -t bolsa-gin:latest .
docker run -d -p 8081:8081 --env-file .env \
  -e PRECIOS_BASE_URL=http://host.docker.internal:3010 \
  --add-host host.docker.internal:host-gateway \
  --name bolsa-gin-app bolsa-gin:latest
```

Disponible en **http://localhost:8081**.

## Notas

- Las migraciones de base de datos se ejecutan automáticamente al arrancar.
- El `Dockerfile` es multi-stage (Go 1.25 sobre Alpine); el binario se compila con `CGO_ENABLED=0`.

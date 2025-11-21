# 🚀 Instrucciones Docker

## Construir la imagen Docker

```bash
docker build -t bolsa-gin:latest .
```

## Ejecutar el contenedor

```bash
docker run -d -p 8081:8081 --name bolsa-gin-app bolsa-gin:latest
```

La aplicación estará disponible en: **http://localhost:8081**

## Comandos útiles

### Ver logs del contenedor
```bash
docker logs bolsa-gin-app
```

### Detener el contenedor
```bash
docker stop bolsa-gin-app
```

### Iniciar el contenedor detenido
```bash
docker start bolsa-gin-app
```

### Eliminar el contenedor
```bash
docker rm bolsa-gin-app
```

### Eliminar la imagen
```bash
docker rmi bolsa-gin:latest
```

## Persistencia de datos

Si deseas persistir la base de datos SQLite entre reinicios del contenedor, puedes montar un volumen:

```bash
docker run -d -p 8081:8081 -v $(pwd)/data:/root --name bolsa-gin-app bolsa-gin:latest
```

En Windows PowerShell:
```powershell
docker run -d -p 8081:8081 -v ${PWD}/data:/root --name bolsa-gin-app bolsa-gin:latest
```

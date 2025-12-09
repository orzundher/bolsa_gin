# API Documentation - Bolsa GIN

Este documento describe la especificación OpenAPI del backend de Bolsa GIN.

## 📄 Archivo de Especificación

El archivo `openapi.yaml` contiene la especificación completa de la API en formato OpenAPI 3.0.3.

## 🎯 Propósito

Esta especificación OpenAPI sirve como:

1. **Documentación oficial** de todos los endpoints del backend
2. **Contrato de API** para desarrollo futuro de clientes
3. **Base para generación automática** de código cliente/servidor
4. **Referencia para testing** y validación de la API

## 📚 Endpoints Documentados

### Vistas HTML (Renderizado de páginas)
- `GET /` - Página principal con dashboard
- `GET /resumen` - Resumen por ticker
- `GET /compras` - Historial de compras
- `GET /ventas` - Historial de ventas
- `GET /precios` - Lista de tickers y precios
- `GET /snapshots` - Lista de snapshots históricos
- `GET /ticker/{id}` - Detalle de un ticker
- `GET /edit/{id}` - Formulario de edición de compra

### Tickers
- `POST /add-ticker` - Crear nuevo ticker
- `POST /update-ticker/{id}` - Actualizar ticker
- `POST /delete-ticker` - Eliminar ticker

### Snapshots de Precios
- `POST /create-snapshot` - Crear snapshot de precios actuales
- `POST /delete-snapshot` - Eliminar snapshot

### Inversiones (Compras)
- `POST /add-investment` - Registrar nueva compra
- `POST /update/{id}` - Actualizar compra (form)
- `POST /delete-investment` - Eliminar compra
- `GET /api/investment/{id}` - Obtener datos de compra (JSON)
- `PUT /api/investment/{id}` - Actualizar compra (JSON)

### Ventas
- `POST /add-sale` - Registrar nueva venta
- `POST /update-sale/{id}` - Actualizar venta (form)
- `POST /delete-sale` - Eliminar venta
- `GET /api/sale/{id}` - Obtener datos de venta (JSON)
- `PUT /api/sale/{id}` - Actualizar venta (JSON)

### Análisis
- `GET /sale-calculation/{id}` - Detalles del cálculo de utilidad de una venta
- `GET /api/portfolio-utility-history` - Historial de utilidad de la cartera

## 🛠️ Herramientas Recomendadas

### Visualización de la API

Puedes visualizar y explorar la API usando:

#### 1. Swagger UI (Online)
Visita [Swagger Editor](https://editor.swagger.io/) y pega el contenido de `openapi.yaml`

#### 2. Swagger UI (Local)
```bash
# Usando Docker
docker run -p 8080:8080 -e SWAGGER_JSON=/openapi.yaml -v ${PWD}:/usr/share/nginx/html swaggerapi/swagger-ui

# Luego visita http://localhost:8080
```

#### 3. Redoc (Online)
Visita [Redoc Try](https://redocly.github.io/redoc/) y carga el archivo `openapi.yaml`

#### 4. VS Code Extension
Instala la extensión "OpenAPI (Swagger) Editor" para VS Code

### Generación de Código Cliente

Puedes generar clientes automáticamente usando OpenAPI Generator:

```bash
# Instalar OpenAPI Generator
npm install @openapitools/openapi-generator-cli -g

# Generar cliente JavaScript/TypeScript
openapi-generator-cli generate -i openapi.yaml -g typescript-axios -o ./client/typescript

# Generar cliente Python
openapi-generator-cli generate -i openapi.yaml -g python -o ./client/python

# Generar cliente Go
openapi-generator-cli generate -i openapi.yaml -g go -o ./client/go

# Ver todos los generadores disponibles
openapi-generator-cli list
```

### Testing de la API

#### Usando Postman
1. Importa el archivo `openapi.yaml` en Postman
2. Postman generará automáticamente una colección con todos los endpoints

#### Usando curl (ejemplos)

```bash
# Crear un nuevo ticker
curl -X POST http://localhost:8081/add-ticker \
  -d "name=AAPL&current_price=150.50"

# Obtener datos de una compra (JSON)
curl http://localhost:8081/api/investment/1

# Actualizar una compra (JSON)
curl -X PUT http://localhost:8081/api/investment/1 \
  -H "Content-Type: application/json" \
  -d '{
    "ticker_id": 1,
    "purchase_date": "2023-12-07T10:30",
    "shares": 10.5,
    "purchase_price": 150.50,
    "operation_cost": 5.0
  }'

# Crear snapshot
curl -X POST http://localhost:8081/create-snapshot

# Obtener historial de utilidad
curl http://localhost:8081/api/portfolio-utility-history
```

## 🔄 Migración a API REST Pura

Si en el futuro decides crear una API REST dedicada (sin vistas HTML), puedes:

1. **Separar los endpoints**: Los endpoints bajo `/api/*` ya devuelven JSON puro
2. **Convertir endpoints de formulario**: Cambiar los endpoints POST que usan `application/x-www-form-urlencoded` a JSON
3. **Eliminar redirecciones**: Reemplazar las respuestas `302` con respuestas JSON
4. **Agregar autenticación**: Implementar JWT u OAuth2
5. **Versionar la API**: Agregar prefijo `/v1/` a todos los endpoints

### Ejemplo de estructura futura:

```
/api/v1/tickers
  GET    /           - Listar todos los tickers
  POST   /           - Crear ticker
  GET    /{id}       - Obtener ticker
  PUT    /{id}       - Actualizar ticker
  DELETE /{id}       - Eliminar ticker

/api/v1/investments
  GET    /           - Listar inversiones
  POST   /           - Crear inversión
  GET    /{id}       - Obtener inversión
  PUT    /{id}       - Actualizar inversión
  DELETE /{id}       - Eliminar inversión

/api/v1/sales
  GET    /           - Listar ventas
  POST   /           - Crear venta
  GET    /{id}       - Obtener venta
  PUT    /{id}       - Actualizar venta
  DELETE /{id}       - Eliminar venta

/api/v1/snapshots
  GET    /           - Listar snapshots
  POST   /           - Crear snapshot
  DELETE /{id}       - Eliminar snapshot

/api/v1/analytics
  GET    /portfolio-utility-history
  GET    /sale-calculation/{id}
  GET    /ticker-summary
```

## 📝 Notas Importantes

### Formatos de Fecha
La API acepta múltiples formatos de fecha:
- ISO 8601: `2023-12-07T10:30`
- Solo fecha: `2023-12-07`
- Formato DD/MM/YYYY (solo para ventas): `07/12/2023`

### Números Decimales
Los campos numéricos aceptan tanto punto como coma como separador decimal:
- `150.50` ✅
- `150,50` ✅ (se convierte automáticamente)

### Soft Delete
Las compras y ventas usan "soft delete" (borrado lógico), por lo que no se eliminan físicamente de la base de datos.

### Cálculo de WAC (Weighted Average Cost)
El sistema calcula automáticamente el precio promedio ponderado para determinar la utilidad de las ventas, considerando todas las compras y ventas previas en orden cronológico.

## 🚀 Próximos Pasos

1. **Validar la especificación**: Usa herramientas como Swagger Validator
2. **Generar documentación estática**: Usa Redoc CLI para generar HTML
3. **Implementar tests automáticos**: Usa la especificación para generar tests
4. **Considerar GraphQL**: Si necesitas consultas más flexibles
5. **Agregar rate limiting**: Para proteger la API en producción
6. **Implementar CORS**: Si planeas consumir la API desde un frontend separado

## 📞 Soporte

Para más información sobre OpenAPI:
- [OpenAPI Specification](https://swagger.io/specification/)
- [OpenAPI Generator](https://openapi-generator.tech/)
- [Swagger Tools](https://swagger.io/tools/)

# 📊 Bolsa Gin - Trading Dashboard

Una aplicación web simple desarrollada con Go y el framework Gin para visualizar un portafolio de inversiones en acciones.

## Características

- **Dashboard interactivo**: Visualiza todas tus transacciones de compra de acciones
- **Cálculos automáticos**: Capital invertido, valor actual y utilidad/pérdida por acción
- **Base de datos SQLite**: Almacenamiento local persistente
- **Diseño responsive**: Interfaz moderna que se adapta a cualquier dispositivo
- **Precios simulados**: Sistema de precios de mercado simulados para demostración

## Tecnologías

- **Backend**: Go 1.x con Gin Web Framework
- **Base de datos**: SQLite3
- **Frontend**: HTML5 + CSS3 (sin dependencias JavaScript)

## Requisitos Previos

Para ejecutar este proyecto necesitas tener instalado:

1.  **Go**: Versión 1.23 o superior. [Descargar Go](https://go.dev/dl/)
2.  **Compilador C (GCC)**: Necesario para la base de datos SQLite (`go-sqlite3`).
    *   **Windows**: Se recomienda instalar [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) o [MinGW-w64](https://www.mingw-w64.org/).
    *   **Linux/macOS**: Generalmente ya incluyen GCC o se instala fácilmente (`sudo apt install build-essential` o `xcode-select --install`).
3.  **Git**: Para clonar el repositorio.

## Instalación

1. Clona el repositorio:
```bash
git clone https://github.com/orzundher/bolsa_gin.git
cd bolsa_gin
```

2. Instala las dependencias:
```bash
go mod download
```

3. Compila la aplicación:
```bash
go build -o bolsa_gin
```

4. Ejecuta la aplicación:
```bash
./bolsa_gin
```

5. Abre tu navegador en: http://localhost:8080

## Estructura del Proyecto

```
bolsa_gin/
├── main.go              # Aplicación principal con servidor Gin
├── templates/
│   └── index.html       # Plantilla HTML del dashboard
├── go.mod               # Dependencias del proyecto
├── go.sum               # Checksums de dependencias
└── README.md            # Este archivo
```

## Uso

Al iniciar la aplicación por primera vez, se creará automáticamente:
- Una base de datos SQLite (`portfolio.db`)
- Datos de ejemplo con 5 transacciones de acciones

### Datos de Ejemplo

La aplicación incluye las siguientes transacciones:
- **AAPL**: 10 acciones compradas a $150.00
- **GOOGL**: 5 acciones compradas a $2,800.00
- **MSFT**: 15 acciones compradas a $300.00
- **TSLA**: 8 acciones compradas a $700.00
- **AMZN**: 12 acciones compradas a $3,200.00

## Funcionalidades del Dashboard

El dashboard muestra para cada acción:
- ✅ Símbolo de la acción
- ✅ Cantidad de acciones
- ✅ Precio de compra
- ✅ Capital invertido (cantidad × precio compra)
- ✅ Precio actual (simulado)
- ✅ Valor actual (cantidad × precio actual)
- ✅ Utilidad/Pérdida (valor actual - capital invertido)
  - 🟢 Verde: Ganancia
  - 🔴 Rojo: Pérdida

También muestra:
- 💰 **Capital Total Invertido** en todo el portafolio

## Desarrollo

Para ejecutar en modo desarrollo con recarga automática:
```bash
go run main.go
```

## 📚 Documentación de API

Este proyecto incluye documentación completa de la API para facilitar el desarrollo de clientes y la migración futura a una arquitectura de API REST dedicada.

### Documentos Disponibles

- **[openapi.yaml](openapi.yaml)**: Especificación OpenAPI 3.0 completa de todos los endpoints
- **[API_README.md](API_README.md)**: Guía de uso de la especificación OpenAPI, herramientas recomendadas y próximos pasos
- **[API_EXAMPLES.md](API_EXAMPLES.md)**: Ejemplos prácticos de uso de cada endpoint en curl, JavaScript y Python
- **[MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)**: Guía completa para migrar a una API REST dedicada con frontend separado

### Visualizar la API

Puedes visualizar y explorar la API usando:

1. **Swagger Editor Online**: Visita [editor.swagger.io](https://editor.swagger.io/) y carga el archivo `openapi.yaml`
2. **Swagger UI Local**:
   ```bash
   docker run -p 8080:8080 -e SWAGGER_JSON=/openapi.yaml -v ${PWD}:/usr/share/nginx/html swaggerapi/swagger-ui
   ```
3. **VS Code**: Instala la extensión "OpenAPI (Swagger) Editor"

### Endpoints Principales

- **Vistas HTML**: `/`, `/resumen`, `/compras`, `/ventas`, `/precios`, `/snapshots`
- **Tickers**: `POST /add-ticker`, `POST /update-ticker/:id`, `POST /delete-ticker`
- **Inversiones**: `POST /add-investment`, `PUT /api/investment/:id`, `DELETE /delete-investment`
- **Ventas**: `POST /add-sale`, `PUT /api/sale/:id`, `DELETE /delete-sale`
- **Análisis**: `GET /sale-calculation/:id`, `GET /api/portfolio-utility-history`
- **Snapshots**: `POST /create-snapshot`, `POST /delete-snapshot`

Para más detalles, consulta la [documentación completa de la API](API_README.md).

## Licencia

Este proyecto está disponible como código abierto.

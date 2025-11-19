<<<<<<< HEAD
# Proyecto Portafolio de Inversiones (bolsa_gin)

## Descripción

Esta es una aplicación web simple desarrollada con Go y el framework Gin. Su propósito es mostrar un resumen de un portafolio de inversiones en acciones, leyendo los datos desde una base de datos SQLite local.

La página principal muestra una tabla con todas las transacciones de compra, calcula el capital invertido por acción, el valor actual basado en precios de mercado (simulados), y la utilidad o pérdida correspondiente. También calcula y muestra el capital total invertido en el portafolio.

## Tecnologías

*   **Backend**: Go
*   **Framework Web**: Gin Gonic
*   **Base de Datos**: SQLite 3
*   **Frontend**: HTML, Bootstrap 5 (cargado desde CDN)

## Estructura del Proyecto

*   `main.go`: Contiene toda la lógica de la aplicación, incluyendo la configuración del servidor, la conexión a la base de datos, y el manejo de las rutas.
*   `investments.db`: Archivo de la base de datos SQLite. Se crea automáticamente en la primera ejecución si no existe.
*   `templates/index.html`: Plantilla HTML que se renderiza para mostrar los datos del portafolio.
*   `go.mod`, `go.sum`: Archivos que gestionan las dependencias del proyecto de Go.
*   `gemini.md`: Este archivo, con la descripción del proyecto.

## Cómo Ejecutar la Aplicación

1.  **Instalar dependencias**:
    Si aún no lo has hecho, ejecuta este comando para descargar Gin y el driver de SQLite.
    ```bash
    go mod tidy
    ```

2.  **Ejecutar el servidor**:
    Usa el siguiente comando para iniciar la aplicación.
    ```bash
    go run main.go
    ```

3.  **Acceder a la aplicación**:
    Una vez que el servidor esté en funcionamiento, abre tu navegador y ve a la siguiente URL:
    [http://localhost:8080](http://localhost:8080)

## Notas Adicionales

*   **Precios de Acciones**: Para esta versión inicial, los "precios actuales" de las acciones no son en tiempo real. Están definidos en un mapa dentro del archivo `main.go` a modo de demostración. Un siguiente paso lógico sería integrar una API de datos de mercado para obtener precios actualizados.

## Actualizaciones Recientes

*   **Resumen del Portafolio**: La sección de "Capital Total Invertido" y "Utilidad Neta Actual" ahora se muestra de forma más prominente, justo debajo del título principal.
*   **Visualización de Utilidades**: La tabla "Resumen por Ticker" ahora resalta las filas con utilidad positiva en verde y las de utilidad negativa en rojo, facilitando la identificación rápida del rendimiento.
*   **Orden por Defecto**: La tabla "Resumen por Ticker" se ordena automáticamente por "Utilidad (+/-)" de mayor a menor al cargar la página.
*   **Actualización de Precios Mejorada**: La herramienta para "Actualizar Precios de Mercado" ha sido rediseñada como una tabla ordenable, lo que mejora la usabilidad y la gestión de los precios de las acciones.
=======
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

## Licencia

Este proyecto está disponible como código abierto.
>>>>>>> copilot/add-portfolio-summary-page

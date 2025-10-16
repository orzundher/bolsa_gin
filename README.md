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
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

# 📚 Índice de Documentación - Bolsa GIN

Bienvenido a la documentación completa del proyecto **Bolsa GIN**. Este índice te ayudará a navegar por todos los documentos disponibles.

---

## 🎯 Inicio Rápido

**¿Primera vez aquí?** Empieza por estos documentos en orden:

1. 📖 **[README.md](README.md)** - Introducción al proyecto y cómo ejecutarlo
2. 🌟 **[API_DOCUMENTATION_SUMMARY.md](API_DOCUMENTATION_SUMMARY.md)** - Resumen ejecutivo de la documentación de API
3. 📋 **[openapi.yaml](openapi.yaml)** - Especificación OpenAPI (ábrelo en [Swagger Editor](https://editor.swagger.io/))

---

## 📁 Documentación General

### [README.md](README.md)
**Documentación principal del proyecto**

- ✅ Descripción del proyecto
- ✅ Características principales
- ✅ Requisitos e instalación
- ✅ Estructura del proyecto
- ✅ Uso básico
- ✅ Referencias a documentación de API

**👉 Empieza aquí si es tu primera vez**

---

## 🔌 Documentación de API

### [openapi.yaml](openapi.yaml)
**Especificación OpenAPI 3.0.3 completa**

- ✅ Todos los endpoints documentados (38 endpoints)
- ✅ Schemas de request/response
- ✅ Ejemplos de uso
- ✅ Validado con swagger-cli
- ✅ Listo para generar código cliente

**Cómo usar:**
```bash
# Visualizar en Swagger Editor
# Visita https://editor.swagger.io/ y carga este archivo

# O localmente con Docker
docker run -p 8080:8080 -e SWAGGER_JSON=/openapi.yaml -v ${PWD}:/usr/share/nginx/html swaggerapi/swagger-ui
```

**👉 Referencia técnica completa de la API**

---

### [API_README.md](API_README.md)
**Guía de uso de la especificación OpenAPI**

Contenido:
- 🛠️ Herramientas recomendadas
- 📊 Cómo visualizar la API
- 🔧 Generación de código cliente
- 🧪 Testing de la API
- 🚀 Próximos pasos
- 📝 Notas importantes

**👉 Lee esto para entender cómo usar openapi.yaml**

---

### [API_EXAMPLES.md](API_EXAMPLES.md)
**Ejemplos prácticos de uso de cada endpoint**

Incluye ejemplos en:
- 🔧 curl (terminal)
- 🟨 JavaScript/TypeScript (Fetch API y Axios)
- 🐍 Python (requests)

Cubre:
- Todos los endpoints principales
- Casos de uso comunes
- Flujos completos
- Manejo de errores

**👉 Copia y pega estos ejemplos para empezar rápido**

---

### [API_DOCUMENTATION_SUMMARY.md](API_DOCUMENTATION_SUMMARY.md)
**Resumen ejecutivo de la documentación de API**

- 📊 Estadísticas de la documentación
- 🎯 Beneficios inmediatos
- 🚀 Cómo empezar
- 🛣️ Roadmap sugerido
- 💡 Casos de uso

**👉 Vista general de toda la documentación de API**

---

## 🔄 Migración y Arquitectura

### [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)
**Guía completa de migración a API REST dedicada**

Roadmap en 4 fases:
1. **Fase 1**: Preparación (Semanas 1-2)
2. **Fase 2**: Refactorización backend (Semanas 3-4)
3. **Fase 3**: Nuevos endpoints REST (Semanas 5-6)
4. **Fase 4**: Frontend separado (Semanas 7-10)

Incluye:
- 🏗️ Arquitectura propuesta
- 🔐 Autenticación JWT
- 🌐 CORS y seguridad
- ⚡ Rate limiting
- 📊 Versionado de API
- ✅ Checklist completo

**👉 Planifica la evolución del proyecto a largo plazo**

---

## 📊 Documentación Técnica

### [data_model.puml](data_model.puml)
**Diagrama del modelo de datos (PlantUML)**

- Estructura de la base de datos
- Relaciones entre entidades
- Campos y tipos de datos

**Cómo visualizar:**
```bash
# Online: http://www.plantuml.com/plantuml/
# O con extensión de VS Code: PlantUML
```

---

## 🗂️ Organización de Archivos

```
bolsa_gin/
├── 📖 README.md                          # Documentación principal
├── 📚 INDEX.md                           # Este archivo (índice)
│
├── 🔌 API Documentation/
│   ├── openapi.yaml                      # Especificación OpenAPI
│   ├── API_README.md                     # Guía de uso de OpenAPI
│   ├── API_EXAMPLES.md                   # Ejemplos prácticos
│   ├── API_DOCUMENTATION_SUMMARY.md      # Resumen ejecutivo
│   └── MIGRATION_GUIDE.md                # Guía de migración
│
├── 🗄️ Database/
│   ├── data_model.puml                   # Diagrama del modelo
│   └── investments.db                    # Base de datos SQLite
│
├── 💻 Source Code/
│   ├── main.go                           # Aplicación principal
│   ├── go.mod                            # Dependencias
│   └── go.sum                            # Checksums
│
├── 🎨 Frontend/
│   ├── templates/                        # Plantillas HTML
│   │   ├── index.html
│   │   ├── compras.html
│   │   ├── ventas.html
│   │   ├── precios.html
│   │   ├── resumen.html
│   │   ├── snapshots.html
│   │   ├── ticker_detail.html
│   │   ├── edit.html
│   │   └── header.html
│   └── static/                           # Archivos estáticos
│       └── styles.css
│
└── ⚙️ Configuration/
    ├── .env                              # Variables de entorno
    ├── .gitignore                        # Archivos ignorados
    └── .dockerignore                     # Docker ignore
```

---

## 🎓 Guías por Rol

### Para Desarrolladores Backend

1. **Entender el proyecto**: [README.md](README.md)
2. **Explorar la API**: [openapi.yaml](openapi.yaml) en Swagger Editor
3. **Ver ejemplos**: [API_EXAMPLES.md](API_EXAMPLES.md)
4. **Planificar migración**: [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)

### Para Desarrolladores Frontend

1. **Entender el proyecto**: [README.md](README.md)
2. **Ver endpoints disponibles**: [API_README.md](API_README.md)
3. **Copiar ejemplos JavaScript**: [API_EXAMPLES.md](API_EXAMPLES.md)
4. **Generar cliente TypeScript**:
   ```bash
   openapi-generator-cli generate -i openapi.yaml -g typescript-axios -o ./client
   ```

### Para QA/Testing

1. **Entender la API**: [openapi.yaml](openapi.yaml)
2. **Importar en Postman**: Importar `openapi.yaml`
3. **Ver ejemplos de testing**: [API_EXAMPLES.md](API_EXAMPLES.md)
4. **Automatizar tests**: Usar schemathesis o similar

### Para Product Managers

1. **Visión general**: [README.md](README.md)
2. **Resumen de API**: [API_DOCUMENTATION_SUMMARY.md](API_DOCUMENTATION_SUMMARY.md)
3. **Roadmap técnico**: [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)

### Para DevOps

1. **Configuración**: [README.md](README.md) - Sección de instalación
2. **Variables de entorno**: `.env`
3. **Arquitectura futura**: [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)

---

## 🔍 Búsqueda Rápida

### ¿Necesitas...?

| Necesidad | Documento |
|-----------|-----------|
| Instalar y ejecutar el proyecto | [README.md](README.md) |
| Ver todos los endpoints | [openapi.yaml](openapi.yaml) |
| Ejemplos de código | [API_EXAMPLES.md](API_EXAMPLES.md) |
| Generar cliente automáticamente | [API_README.md](API_README.md) |
| Planificar migración a API REST | [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md) |
| Entender el modelo de datos | [data_model.puml](data_model.puml) |
| Resumen ejecutivo | [API_DOCUMENTATION_SUMMARY.md](API_DOCUMENTATION_SUMMARY.md) |

---

## 📋 Endpoints por Categoría

### Vistas HTML
- `GET /` - Dashboard principal
- `GET /resumen` - Resumen por ticker
- `GET /compras` - Historial de compras
- `GET /ventas` - Historial de ventas
- `GET /precios` - Lista de tickers
- `GET /snapshots` - Snapshots históricos
- `GET /ticker/{id}` - Detalle de ticker
- `GET /edit/{id}` - Editar compra

### Tickers
- `POST /add-ticker` - Crear ticker
- `POST /update-ticker/{id}` - Actualizar ticker
- `POST /delete-ticker` - Eliminar ticker

### Inversiones (Compras)
- `POST /add-investment` - Crear compra
- `POST /update/{id}` - Actualizar compra (form)
- `POST /delete-investment` - Eliminar compra
- `GET /api/investment/{id}` - Obtener compra (JSON)
- `PUT /api/investment/{id}` - Actualizar compra (JSON)

### Ventas
- `POST /add-sale` - Crear venta
- `POST /update-sale/{id}` - Actualizar venta (form)
- `POST /delete-sale` - Eliminar venta
- `GET /api/sale/{id}` - Obtener venta (JSON)
- `PUT /api/sale/{id}` - Actualizar venta (JSON)

### Snapshots
- `POST /create-snapshot` - Crear snapshot
- `POST /delete-snapshot` - Eliminar snapshot

### Análisis
- `GET /sale-calculation/{id}` - Cálculo de venta
- `GET /api/portfolio-utility-history` - Historial de utilidad

**Detalles completos**: Ver [openapi.yaml](openapi.yaml)

---

## 🛠️ Herramientas Recomendadas

### Visualización de API
- **Swagger Editor**: https://editor.swagger.io/
- **Swagger UI**: Docker o npm
- **Redoc**: https://redocly.github.io/redoc/

### Generación de Código
- **OpenAPI Generator**: https://openapi-generator.tech/
- **Swagger Codegen**: https://swagger.io/tools/swagger-codegen/

### Testing
- **Postman**: https://www.postman.com/
- **Insomnia**: https://insomnia.rest/
- **Schemathesis**: https://schemathesis.readthedocs.io/

### Desarrollo
- **VS Code Extension**: OpenAPI (Swagger) Editor
- **PlantUML Extension**: Para visualizar data_model.puml

---

## 📊 Estadísticas del Proyecto

### Documentación
- **Total de archivos de documentación**: 6
- **Total de líneas de documentación**: ~3,500
- **Idiomas de ejemplos**: 3 (curl, JavaScript, Python)

### API
- **Total de endpoints**: 38
- **Endpoints HTML**: 8
- **Endpoints JSON**: 30
- **Schemas definidos**: 12
- **Tags/Categorías**: 6

### Código
- **Lenguaje principal**: Go
- **Framework**: Gin
- **Base de datos**: PostgreSQL (antes SQLite)
- **ORM**: GORM

---

## 🔄 Mantenimiento de la Documentación

### Actualizar cuando:
- ✅ Se agreguen nuevos endpoints
- ✅ Se modifiquen endpoints existentes
- ✅ Cambien los schemas de datos
- ✅ Se implementen nuevas funcionalidades

### Cómo actualizar:
1. Modificar `openapi.yaml`
2. Validar con swagger-cli: `npx @apidevtools/swagger-cli validate openapi.yaml`
3. Actualizar ejemplos en `API_EXAMPLES.md`
4. Actualizar guías si es necesario

---

## 🤝 Contribución

Para contribuir a la documentación:

1. **Fork** el repositorio
2. **Crea** una rama para tu cambio
3. **Actualiza** la documentación
4. **Valida** que openapi.yaml siga siendo válido
5. **Envía** un Pull Request

---

## 📞 Soporte

Para preguntas o problemas:
- 📧 Abre un issue en GitHub
- 💬 Consulta la documentación existente
- 🔍 Busca en los ejemplos

---

## ✨ Próximos Pasos Recomendados

1. ✅ **Explorar la API**: Abre `openapi.yaml` en Swagger Editor
2. ✅ **Probar endpoints**: Usa Postman o curl con los ejemplos
3. ✅ **Generar cliente**: Crea un cliente TypeScript/Python
4. ✅ **Planificar migración**: Lee `MIGRATION_GUIDE.md`
5. ✅ **Implementar tests**: Usa la especificación OpenAPI

---

## 📚 Recursos Adicionales

- **OpenAPI Specification**: https://swagger.io/specification/
- **Gin Framework**: https://gin-gonic.com/
- **GORM**: https://gorm.io/
- **Go Documentation**: https://go.dev/doc/

---

**Última actualización**: Diciembre 2023

**Versión de la documentación**: 1.0.0

---

¿Tienes preguntas? Empieza por el [README.md](README.md) o el [resumen ejecutivo](API_DOCUMENTATION_SUMMARY.md).

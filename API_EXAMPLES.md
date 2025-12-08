# Ejemplos de Uso de la API - Bolsa GIN

Este documento contiene ejemplos prácticos de cómo usar cada endpoint de la API.

## 📋 Tabla de Contenidos

- [Gestión de Tickers](#gestión-de-tickers)
- [Gestión de Snapshots](#gestión-de-snapshots)
- [Gestión de Inversiones](#gestión-de-inversiones)
- [Gestión de Ventas](#gestión-de-ventas)
- [Análisis y Reportes](#análisis-y-reportes)

---

## Gestión de Tickers

### Crear un nuevo ticker

```bash
curl -X POST http://localhost:8081/add-ticker \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "name=AAPL&current_price=150.50"
```

**Respuesta**: Redirección a `/precios`

### Actualizar un ticker existente

```bash
curl -X POST http://localhost:8081/update-ticker/1 \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "name=AAPL&current_price=155.75"
```

**Respuesta**: Redirección a `/precios`

### Eliminar un ticker

```bash
curl -X POST http://localhost:8081/delete-ticker \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "id=1"
```

**Respuesta**: Redirección a `/precios`

**Nota**: Solo se puede eliminar si no tiene inversiones o ventas asociadas.

---

## Gestión de Snapshots

### Crear un snapshot de precios

```bash
curl -X POST http://localhost:8081/create-snapshot
```

**Respuesta JSON**:
```json
{
  "success": true,
  "message": "Snapshot creado exitosamente con 5 precios",
  "snapshotID": "20231207-154530",
  "count": 5
}
```

### Eliminar un snapshot

```bash
curl -X POST http://localhost:8081/delete-snapshot \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "snapshot_id=20231207-154530"
```

**Respuesta**: Redirección a `/snapshots`

---

## Gestión de Inversiones

### Registrar una nueva compra (Formulario)

```bash
curl -X POST http://localhost:8081/add-investment \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "ticker_id=1" \
  -d "purchase_date=2023-12-07T10:30" \
  -d "shares=10.5" \
  -d "purchase_price=150.50" \
  -d "operation_cost=5.0" \
  -d "redirect_to=/compras"
```

**Respuesta**: Redirección a `/compras`

### Obtener datos de una compra (JSON)

```bash
curl http://localhost:8081/api/investment/1
```

**Respuesta JSON**:
```json
{
  "id": 1,
  "ticker_id": 1,
  "ticker": "AAPL",
  "purchase_date": "2023-12-07T10:30",
  "shares": 10.5,
  "purchase_price": 150.50,
  "operation_cost": 5.0
}
```

### Actualizar una compra (JSON)

```bash
curl -X PUT http://localhost:8081/api/investment/1 \
  -H "Content-Type: application/json" \
  -d '{
    "ticker_id": 1,
    "purchase_date": "2023-12-07T10:30",
    "shares": 12.0,
    "purchase_price": 148.75,
    "operation_cost": 6.0
  }'
```

**Respuesta JSON**:
```json
{
  "id": 1,
  "ticker_id": 1,
  "ticker": "AAPL",
  "purchase_date": "07 Dec 2023 10:30",
  "shares": 12.0,
  "purchase_price": 148.75,
  "operation_cost": 6.0,
  "invested_capital": 1785.0,
  "current_price": 155.75,
  "current_value": 1869.0,
  "profit_loss": 78.0
}
```

### Actualizar una compra (Formulario)

```bash
curl -X POST http://localhost:8081/update/1 \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "ticker_id=1" \
  -d "purchase_date=2023-12-07T10:30" \
  -d "shares=12.0" \
  -d "purchase_price=148.75" \
  -d "operation_cost=6.0"
```

**Respuesta**: Redirección a `/compras`

### Eliminar una compra

```bash
curl -X POST http://localhost:8081/delete-investment \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "id=1" \
  -d "redirect_to=/compras"
```

**Respuesta**: Redirección a `/compras`

**Nota**: Usa soft delete, el registro no se elimina físicamente.

---

## Gestión de Ventas

### Registrar una nueva venta (Formulario)

```bash
curl -X POST http://localhost:8081/add-sale \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "ticker_id=1" \
  -d "sale_date=2023-12-15T15:30" \
  -d "shares=5.0" \
  -d "sale_price=160.00" \
  -d "operation_cost=3.0" \
  -d "withheld_tax=10.0" \
  -d "redirect_to=/ventas"
```

**Respuesta**: Redirección a `/ventas`

### Obtener datos de una venta (JSON)

```bash
curl http://localhost:8081/api/sale/1
```

**Respuesta JSON**:
```json
{
  "id": 1,
  "ticker_id": 1,
  "ticker": "AAPL",
  "sale_date": "2023-12-15T15:30",
  "shares": 5.0,
  "sale_price": 160.00,
  "operation_cost": 3.0,
  "withheld_tax": 10.0
}
```

### Actualizar una venta (JSON)

```bash
curl -X PUT http://localhost:8081/api/sale/1 \
  -H "Content-Type: application/json" \
  -d '{
    "ticker_id": 1,
    "sale_date": "2023-12-15T15:30",
    "shares": 6.0,
    "sale_price": 162.00,
    "operation_cost": 3.5,
    "withheld_tax": 12.0
  }'
```

**Respuesta JSON**:
```json
{
  "id": 1,
  "ticker_id": 1,
  "ticker": "AAPL",
  "sale_date": "15 Dec 2023 15:30",
  "shares": 6.0,
  "sale_price": 162.00,
  "operation_cost": 3.5,
  "withheld_tax": 12.0,
  "total_sale_value": 972.0,
  "performance": 8.64,
  "profit": 68.70
}
```

### Actualizar una venta (Formulario)

```bash
curl -X POST http://localhost:8081/update-sale/1 \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "ticker_id=1" \
  -d "sale_date=2023-12-15T15:30" \
  -d "shares=6.0" \
  -d "sale_price=162.00" \
  -d "operation_cost=3.5" \
  -d "withheld_tax=12.0" \
  -d "redirect_to=/ventas"
```

**Respuesta**: Redirección a `/ventas`

### Eliminar una venta

```bash
curl -X POST http://localhost:8081/delete-sale \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "id=1" \
  -d "redirect_to=/ventas"
```

**Respuesta**: Redirección a `/ventas`

---

## Análisis y Reportes

### Obtener detalles del cálculo de utilidad de una venta

```bash
curl http://localhost:8081/sale-calculation/1
```

**Respuesta JSON**:
```json
{
  "ticker": "AAPL",
  "sale_date": "15 Dec 2023 15:30",
  "shares": 5.0,
  "sale_price": 160.00,
  "purchases": [
    {
      "date": "01 Dec 2023 10:30",
      "shares": 10.5,
      "price": 150.50,
      "total": 1580.25
    },
    {
      "date": "05 Dec 2023 14:00",
      "shares": 5.0,
      "price": 148.00,
      "total": 740.00
    }
  ],
  "total_capital": 2320.25,
  "total_shares": 15.5,
  "wac": 149.69,
  "profit": 51.55
}
```

**Explicación**:
- `wac` (Weighted Average Cost): Precio promedio ponderado de las compras
- `profit`: Utilidad calculada como `(sale_price - wac) * shares`
- `purchases`: Lista de todas las compras previas a la venta
- `total_capital`: Capital total invertido antes de la venta
- `total_shares`: Total de acciones antes de la venta

### Obtener historial de utilidad de la cartera

```bash
curl http://localhost:8081/api/portfolio-utility-history
```

**Respuesta JSON**:
```json
{
  "dates": [
    "01 Dec 2023 10:00",
    "02 Dec 2023 10:00",
    "03 Dec 2023 10:00",
    "04 Dec 2023 10:00"
  ],
  "utilities": [
    150.50,
    175.25,
    200.75,
    185.30
  ]
}
```

**Explicación**:
- `dates`: Fechas de los snapshots de precios
- `utilities`: Utilidad de la cartera en cada snapshot (diferencia entre valor actual y WAC)

Este endpoint es útil para:
- Crear gráficos de evolución de la cartera
- Analizar el rendimiento histórico
- Identificar tendencias de ganancia/pérdida

---

## 🔧 Ejemplos con JavaScript/TypeScript

### Usando Fetch API

```javascript
// Crear un nuevo ticker
async function createTicker(name, price) {
  const formData = new URLSearchParams();
  formData.append('name', name);
  formData.append('current_price', price);

  const response = await fetch('http://localhost:8081/add-ticker', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
    },
    body: formData,
    redirect: 'manual' // Para manejar la redirección manualmente
  });

  return response;
}

// Obtener datos de una compra
async function getInvestment(id) {
  const response = await fetch(`http://localhost:8081/api/investment/${id}`);
  const data = await response.json();
  return data;
}

// Actualizar una compra
async function updateInvestment(id, investmentData) {
  const response = await fetch(`http://localhost:8081/api/investment/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(investmentData)
  });
  
  const data = await response.json();
  return data;
}

// Crear snapshot
async function createSnapshot() {
  const response = await fetch('http://localhost:8081/create-snapshot', {
    method: 'POST'
  });
  
  const data = await response.json();
  return data;
}

// Obtener historial de utilidad
async function getPortfolioUtilityHistory() {
  const response = await fetch('http://localhost:8081/api/portfolio-utility-history');
  const data = await response.json();
  return data;
}

// Ejemplo de uso
async function example() {
  // Crear ticker
  await createTicker('AAPL', 150.50);
  
  // Obtener datos de inversión
  const investment = await getInvestment(1);
  console.log('Investment:', investment);
  
  // Actualizar inversión
  const updated = await updateInvestment(1, {
    ticker_id: 1,
    purchase_date: '2023-12-07T10:30',
    shares: 12.0,
    purchase_price: 148.75,
    operation_cost: 6.0
  });
  console.log('Updated:', updated);
  
  // Crear snapshot
  const snapshot = await createSnapshot();
  console.log('Snapshot:', snapshot);
  
  // Obtener historial
  const history = await getPortfolioUtilityHistory();
  console.log('History:', history);
}
```

### Usando Axios

```javascript
import axios from 'axios';

const API_BASE_URL = 'http://localhost:8081';

// Crear cliente axios
const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json'
  }
});

// Obtener inversión
const getInvestment = async (id) => {
  const { data } = await api.get(`/api/investment/${id}`);
  return data;
};

// Actualizar inversión
const updateInvestment = async (id, investmentData) => {
  const { data } = await api.put(`/api/investment/${id}`, investmentData);
  return data;
};

// Obtener venta
const getSale = async (id) => {
  const { data } = await api.get(`/api/sale/${id}`);
  return data;
};

// Actualizar venta
const updateSale = async (id, saleData) => {
  const { data } = await api.put(`/api/sale/${id}`, saleData);
  return data;
};

// Obtener cálculo de venta
const getSaleCalculation = async (id) => {
  const { data } = await api.get(`/sale-calculation/${id}`);
  return data;
};

// Obtener historial de utilidad
const getPortfolioUtilityHistory = async () => {
  const { data } = await api.get('/api/portfolio-utility-history');
  return data;
};

// Crear snapshot
const createSnapshot = async () => {
  const { data } = await api.post('/create-snapshot');
  return data;
};

export {
  getInvestment,
  updateInvestment,
  getSale,
  updateSale,
  getSaleCalculation,
  getPortfolioUtilityHistory,
  createSnapshot
};
```

---

## 🐍 Ejemplos con Python

### Usando requests

```python
import requests
from typing import Dict, Any

API_BASE_URL = "http://localhost:8081"

def create_ticker(name: str, price: float) -> requests.Response:
    """Crear un nuevo ticker"""
    data = {
        'name': name,
        'current_price': price
    }
    return requests.post(f"{API_BASE_URL}/add-ticker", data=data)

def get_investment(investment_id: int) -> Dict[str, Any]:
    """Obtener datos de una inversión"""
    response = requests.get(f"{API_BASE_URL}/api/investment/{investment_id}")
    return response.json()

def update_investment(investment_id: int, investment_data: Dict[str, Any]) -> Dict[str, Any]:
    """Actualizar una inversión"""
    response = requests.put(
        f"{API_BASE_URL}/api/investment/{investment_id}",
        json=investment_data
    )
    return response.json()

def get_sale(sale_id: int) -> Dict[str, Any]:
    """Obtener datos de una venta"""
    response = requests.get(f"{API_BASE_URL}/api/sale/{sale_id}")
    return response.json()

def update_sale(sale_id: int, sale_data: Dict[str, Any]) -> Dict[str, Any]:
    """Actualizar una venta"""
    response = requests.put(
        f"{API_BASE_URL}/api/sale/{sale_id}",
        json=sale_data
    )
    return response.json()

def get_sale_calculation(sale_id: int) -> Dict[str, Any]:
    """Obtener detalles del cálculo de utilidad de una venta"""
    response = requests.get(f"{API_BASE_URL}/sale-calculation/{sale_id}")
    return response.json()

def get_portfolio_utility_history() -> Dict[str, Any]:
    """Obtener historial de utilidad de la cartera"""
    response = requests.get(f"{API_BASE_URL}/api/portfolio-utility-history")
    return response.json()

def create_snapshot() -> Dict[str, Any]:
    """Crear un snapshot de precios"""
    response = requests.post(f"{API_BASE_URL}/create-snapshot")
    return response.json()

# Ejemplo de uso
if __name__ == "__main__":
    # Crear ticker
    create_ticker("AAPL", 150.50)
    
    # Obtener inversión
    investment = get_investment(1)
    print(f"Investment: {investment}")
    
    # Actualizar inversión
    updated = update_investment(1, {
        "ticker_id": 1,
        "purchase_date": "2023-12-07T10:30",
        "shares": 12.0,
        "purchase_price": 148.75,
        "operation_cost": 6.0
    })
    print(f"Updated: {updated}")
    
    # Crear snapshot
    snapshot = create_snapshot()
    print(f"Snapshot: {snapshot}")
    
    # Obtener historial
    history = get_portfolio_utility_history()
    print(f"History: {history}")
```

---

## 📊 Casos de Uso Comunes

### 1. Flujo completo de registro de inversión

```bash
# 1. Crear ticker si no existe
curl -X POST http://localhost:8081/add-ticker \
  -d "name=MSFT&current_price=380.50"

# 2. Registrar compra
curl -X POST http://localhost:8081/add-investment \
  -d "ticker_id=2" \
  -d "purchase_date=2023-12-07T10:30" \
  -d "shares=15" \
  -d "purchase_price=380.50" \
  -d "operation_cost=7.5"

# 3. Crear snapshot para registro histórico
curl -X POST http://localhost:8081/create-snapshot
```

### 2. Flujo completo de venta con análisis

```bash
# 1. Registrar venta
curl -X POST http://localhost:8081/add-sale \
  -d "ticker_id=2" \
  -d "sale_date=2023-12-20T14:00" \
  -d "shares=10" \
  -d "sale_price=395.00" \
  -d "operation_cost=5.0" \
  -d "withheld_tax=15.0"

# 2. Obtener detalles del cálculo
curl http://localhost:8081/sale-calculation/1

# 3. Crear snapshot post-venta
curl -X POST http://localhost:8081/create-snapshot
```

### 3. Análisis de rendimiento de cartera

```bash
# 1. Obtener historial de utilidad
curl http://localhost:8081/api/portfolio-utility-history

# 2. Ver detalle de un ticker específico
# (Abrir en navegador)
# http://localhost:8081/ticker/1
```

---

## 🔐 Notas de Seguridad

**IMPORTANTE**: Esta API actualmente NO tiene autenticación. Para uso en producción:

1. Implementar autenticación (JWT, OAuth2, etc.)
2. Agregar validación de CORS
3. Implementar rate limiting
4. Usar HTTPS
5. Validar y sanitizar todas las entradas
6. Implementar logging de auditoría

---

## 📝 Notas Adicionales

- Todos los endpoints que devuelven redirecciones (302) son para uso con formularios HTML
- Los endpoints bajo `/api/*` devuelven JSON puro
- Los números decimales pueden usar punto o coma como separador
- Las fechas aceptan múltiples formatos (ISO 8601, YYYY-MM-DD, DD/MM/YYYY)
- El sistema usa soft delete para inversiones y ventas

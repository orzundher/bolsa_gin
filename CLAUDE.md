# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run in development
go run main.go

# Build
go build -o bolsa_gin.exe

# Download dependencies
go mod download
```

The server starts on `http://localhost:8080` by default (override with `PORT` env var).

**Windows note:** Requires a C compiler (TDM-GCC or MinGW-w64) for the PostgreSQL driver.

## Architecture

**Bolsa Gin** is a Spanish-language stock portfolio dashboard. It tracks buy/sell transactions, calculates Weighted Average Cost (WAC), and shows portfolio performance metrics.

### Structure

- **`main.go`** (~2,900 lines) — single-file monolith: all GORM models, Gin routes, and business logic live here
- **`notes_model.go`** — `Note` and `TickerNote` models and their migrations
- **`templates/`** — 9 server-rendered Go HTML templates (header.html is a partial included by all pages)
- **`static/`** — CSS, vanilla JS, and images; Tailwind + Flowbite via CDN

### Key data models

| Model | Purpose |
|---|---|
| `Ticker` | Stock symbol, current price, group, Yahoo Finance ticker |
| `Investment` | Buy transaction (date, shares, price, operation cost) |
| `Sale` | Sell transaction (date, shares, price) |
| `PriceHistory` | Historical price snapshots grouped by `SnapshotID` |
| `TickerGroup` | Category labels for tickers |
| `Note` / `TickerNote` | User annotations |
| `Migration` | Tracks which schema migrations have run |

### Core calculation: `getInvestmentData()`

This is the central function. For each ticker it computes:
- **WAC** (Weighted Average Cost) — average cost per share across all buys
- **Performance %** — `((currentPrice - WAC) / WAC) * 100`
- **Profit/Loss** — `(currentPrice - WAC) * shares`

### External dependency: price service

Price updates call a **separate microservice** at `http://localhost:3010`:
- `GET /precio/{ticker}` — single price lookup
- `POST /precios` — batch update

This service is **not** part of this repo. Routes `/api/fetch-price/:yahoo_ticker` and the batch update on `/precios` will fail if it is not running.

## Environment

Requires a `.env` file (not committed):

```
DATABASE_URL=postgresql://user:password@host:port/database
PORT=8080
```

Database is PostgreSQL hosted on Supabase. Connection uses `prepare=false` and `PreferSimpleProtocol: true` for Supabase pooler compatibility.

## Database migrations

Migrations run automatically on startup via the `runMigrations()` function. They are tracked in a `migrations` table. To add a migration, append a new entry to the migrations slice with a unique name and raw SQL.

## API specification

`openapi.yaml` contains a complete OpenAPI 3.0.3 spec for all 38 endpoints. Use it as the source of truth for route contracts when adding or changing endpoints.

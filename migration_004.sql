-- Migración 004: Agregar campo yahoo_finance_ticker a la tabla tickers
ALTER TABLE tickers ADD COLUMN IF NOT EXISTS yahoo_finance_ticker TEXT;

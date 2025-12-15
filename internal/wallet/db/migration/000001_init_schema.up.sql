-- internal/wallet/db/migration/000001_init_schema.up.sql

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Wallets Table (The Snapshot)
CREATE TABLE wallets (
                         user_id VARCHAR(50) PRIMARY KEY,
                         balance DOUBLE PRECISION NOT NULL DEFAULT 0.0,
                         updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Transactions Table (The Ledger)
CREATE TABLE transactions (
                              id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                              wallet_id VARCHAR(50) NOT NULL REFERENCES wallets(user_id),
                              amount DOUBLE PRECISION NOT NULL, -- Negative for debit, Positive for credit
                              description TEXT NOT NULL,
                              reference_id VARCHAR(50), -- To link to Order ID or Payment ID
                              created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fast history lookup (e.g., "Show me my last 10 transactions")
CREATE INDEX idx_transactions_wallet ON transactions(wallet_id);
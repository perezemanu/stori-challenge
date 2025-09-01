-- Database initialization script for Stori Transaction Processor
-- This script sets up the database schema and initial configuration

-- Enable UUID extension for PostgreSQL
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create transactions table (GORM will handle this via AutoMigrate, but this is for reference)
-- CREATE TABLE IF NOT EXISTS transactions (
--     id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
--     account_id VARCHAR(255) NOT NULL,
--     transaction_date TIMESTAMP NOT NULL,
--     amount DECIMAL(15,2) NOT NULL,
--     type VARCHAR(10) NOT NULL CHECK (type IN ('credit', 'debit')),
--     description VARCHAR(255),
--     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
--     updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );

-- Create indexes for better performance
-- CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions(account_id);
-- CREATE INDEX IF NOT EXISTS idx_transactions_account_date ON transactions(account_id, transaction_date);
-- CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);
-- CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(transaction_date);

-- Insert initial configuration or reference data if needed
-- (Currently not required for this application)

COMMENT ON SCHEMA public IS 'Stori Transaction Processor Database';
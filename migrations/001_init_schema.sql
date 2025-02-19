-- Enable UUID extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    email VARCHAR(255) UNIQUE NOT NULL, -- Unique index
    password VARCHAR(255) NOT NULL,
    phone VARCHAR(20) UNIQUE, -- Unique index
    role VARCHAR(50) DEFAULT 'user',
    token_version INT DEFAULT 1
);

-- Wallets table
CREATE TABLE wallets (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    user_id INT NOT NULL REFERENCES users (id) ON DELETE CASCADE, -- Foreign key
    balance DECIMAL(15, 2) DEFAULT 0.00,
    currency VARCHAR(3) DEFAULT 'USD',
    qr_code_id TEXT NOT NULL DEFAULT 'default-qr-code-id' -- Provide a default value here
);

-- Add indexes for wallets
CREATE INDEX idx_wallets_user_id ON wallets (user_id);

CREATE INDEX idx_wallets_qr_code_id ON wallets (qr_code_id);

-- Credit Cards table
CREATE TABLE credit_cards (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    user_id INT NOT NULL REFERENCES users (id) ON DELETE CASCADE, -- Foreign key
    card_number VARCHAR(255) NOT NULL, -- Or TEXT, depending on your needs
    expiry_month VARCHAR(2) NOT NULL,
    expiry_year VARCHAR(4) NOT NULL,
    card_type VARCHAR(255) NOT NULL -- e.g., "Visa", "Mastercard"
);

-- Add index for credit_cards
CREATE INDEX idx_credit_cards_user_id ON credit_cards (user_id);

-- Transactions table
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    sender_id INT NOT NULL REFERENCES users (id) ON DELETE CASCADE, -- Foreign key
    receiver_id INT NOT NULL REFERENCES users (id) ON DELETE CASCADE, -- Foreign key
    amount DECIMAL(10, 2) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    qr_code_id VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add indexes for transactions
CREATE INDEX idx_transactions_sender_id ON transactions (sender_id);

CREATE INDEX idx_transactions_receiver_id ON transactions (receiver_id);

CREATE INDEX idx_transactions_qr_code_id ON transactions (qr_code_id);

ALTER TABLE transactions ADD COLUMN type VARCHAR(20);

-- Create merchants table
CREATE TABLE IF NOT EXISTS merchants (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id BIGINT NOT NULL,
    business_name VARCHAR(255) NOT NULL,
    business_type VARCHAR(100),
    business_address TEXT,
    business_id VARCHAR(100) UNIQUE,
    tax_id VARCHAR(100),
    website VARCHAR(255),
    merchant_category VARCHAR(100),
    legal_entity_type VARCHAR(100),
    registration_number VARCHAR(100),
    year_established INTEGER,
    monthly_volume DECIMAL(20,2) DEFAULT 0,
    processing_fee_rate DECIMAL(5,2) DEFAULT 2.5,
    verification_status VARCHAR(50) DEFAULT 'pending',
    documents_submitted BOOLEAN DEFAULT false,
    approved_at TIMESTAMP WITH TIME ZONE,
    api_key VARCHAR(255) UNIQUE,
    webhook_url TEXT,
    ip_whitelist TEXT[],
    operating_hours TEXT[],
    support_email VARCHAR(255),
    support_phone VARCHAR(50),
    settlement_cycle VARCHAR(50) DEFAULT 'daily',
    min_settlement_amount DECIMAL(20,2) DEFAULT 0,
    total_transactions BIGINT DEFAULT 0,
    total_volume DECIMAL(20,2) DEFAULT 0,
    rating INTEGER DEFAULT 0,
    risk_score INTEGER DEFAULT 50,
    compliance_level VARCHAR(50) DEFAULT 'medium_risk',
    dispute_rate DECIMAL(5,2) DEFAULT 0,
    CONSTRAINT fk_merchants_user FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Create merchant_limits table
CREATE TABLE IF NOT EXISTS merchant_limits (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    merchant_id BIGINT NOT NULL,
    daily_transaction_limit DECIMAL(20,2) NOT NULL DEFAULT 10000,
    monthly_transaction_limit DECIMAL(20,2) NOT NULL DEFAULT 100000,
    single_transaction_limit DECIMAL(20,2) NOT NULL DEFAULT 5000,
    min_transaction_amount DECIMAL(20,2) NOT NULL DEFAULT 1,
    max_transaction_amount DECIMAL(20,2) NOT NULL DEFAULT 5000,
    concurrent_transactions INT NOT NULL DEFAULT 10,
    allowed_currencies TEXT[] DEFAULT ARRAY['USD', 'EUR'],
    CONSTRAINT fk_merchant_limits_merchant
        FOREIGN KEY (merchant_id)
        REFERENCES merchants(id)
        ON DELETE CASCADE
);

-- Create indexes
CREATE INDEX idx_merchants_user_id ON merchants(user_id);
CREATE INDEX idx_merchant_limits_merchant_id ON merchant_limits(merchant_id);
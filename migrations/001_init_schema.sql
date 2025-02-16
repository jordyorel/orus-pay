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
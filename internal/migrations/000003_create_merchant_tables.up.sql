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

-- Create index on merchant_id
CREATE INDEX idx_merchant_limits_merchant_id ON merchant_limits(merchant_id); 
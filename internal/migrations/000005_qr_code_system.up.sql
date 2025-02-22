-- Create QR codes table
CREATE TABLE IF NOT EXISTS qr_codes (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    code VARCHAR(255) UNIQUE NOT NULL,
    user_id INTEGER NOT NULL REFERENCES users(id),
    user_type VARCHAR(50) NOT NULL, -- 'regular' or 'merchant'
    type VARCHAR(50) NOT NULL, -- 'static' or 'dynamic'
    amount DECIMAL(10,2),
    expires_at TIMESTAMP WITH TIME ZONE,
    max_uses INTEGER NOT NULL DEFAULT 1,
    usage_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    payment_purpose TEXT,
    
    -- Merchant-specific fields
    daily_limit DECIMAL(10,2),
    monthly_limit DECIMAL(10,2),
    allowed_customers INTEGER[],
    metadata JSONB,

    -- Add constraints
    CONSTRAINT valid_user_type CHECK (user_type IN ('regular', 'merchant')),
    CONSTRAINT valid_qr_type CHECK (type IN ('static', 'dynamic'))
);

-- Create QR transactions table
CREATE TABLE IF NOT EXISTS qr_transactions (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    qr_code_id INTEGER REFERENCES qr_codes(id),
    transaction_id INTEGER UNIQUE REFERENCES transactions(id),
    customer_id INTEGER REFERENCES users(id),
    amount DECIMAL(20,2) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    completed_at TIMESTAMP WITH TIME ZONE,
    failure_reason TEXT
);

-- Add indexes
CREATE INDEX idx_qr_codes_code ON qr_codes(code);
CREATE INDEX idx_qr_codes_user ON qr_codes(user_id, user_type);
CREATE INDEX idx_qr_transactions_qr_code_id ON qr_transactions(qr_code_id);
CREATE INDEX idx_qr_transactions_customer_id ON qr_transactions(customer_id); 
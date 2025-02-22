-- Add new fields to transactions table
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS transaction_id VARCHAR(255) UNIQUE,
    ADD COLUMN IF NOT EXISTS type VARCHAR(50),
    ADD COLUMN IF NOT EXISTS payment_method VARCHAR(50),
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS metadata JSONB,
    ADD COLUMN IF NOT EXISTS payment_type VARCHAR(50);

-- Update existing transaction_ids
UPDATE transactions 
SET transaction_id = CONCAT('TX-', id, '-', EXTRACT(EPOCH FROM created_at)::text)
WHERE transaction_id IS NULL;

-- Make transaction_id NOT NULL after filling existing records
ALTER TABLE transactions
    ALTER COLUMN transaction_id SET NOT NULL;

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_transactions_transaction_id ON transactions(transaction_id);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type); 
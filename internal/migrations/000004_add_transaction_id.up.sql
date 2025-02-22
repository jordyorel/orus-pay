-- First add the column as nullable
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS transaction_id text;

-- Update existing records with a generated transaction ID
UPDATE transactions 
SET transaction_id = CONCAT('TX-', id, '-', EXTRACT(EPOCH FROM created_at)::text)
WHERE transaction_id IS NULL;

-- Then make it NOT NULL and unique
ALTER TABLE transactions 
    ALTER COLUMN transaction_id SET NOT NULL,
    ADD CONSTRAINT transactions_transaction_id_unique UNIQUE (transaction_id); 
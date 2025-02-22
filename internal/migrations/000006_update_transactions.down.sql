ALTER TABLE transactions
    DROP COLUMN IF EXISTS transaction_id,
    DROP COLUMN IF EXISTS type,
    DROP COLUMN IF EXISTS payment_method,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS payment_type; 
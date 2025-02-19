-- First drop existing constraints if they exist
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_wallet;
ALTER TABLE wallets DROP CONSTRAINT IF EXISTS fk_wallets_user;

-- Modify users table
ALTER TABLE users ALTER COLUMN wallet_id DROP NOT NULL;
ALTER TABLE users ALTER COLUMN wallet_id SET DEFAULT NULL;

-- Modify wallets table
ALTER TABLE wallets ALTER COLUMN user_id SET NOT NULL;

-- Add constraints in correct order
ALTER TABLE wallets
    ADD CONSTRAINT fk_wallets_user
    FOREIGN KEY (user_id)
    REFERENCES users(id)
    ON DELETE CASCADE;

ALTER TABLE users
    ADD CONSTRAINT fk_users_wallet
    FOREIGN KEY (wallet_id)
    REFERENCES wallets(id)
    ON DELETE SET NULL; 
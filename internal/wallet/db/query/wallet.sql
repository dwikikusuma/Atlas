-- internal/wallet/db/query/wallet.sql

-- name: CreateWallet :one
INSERT INTO wallets (
    user_id, balance
) VALUES (
             $1, 0.0
         ) RETURNING *;

-- name: GetWallet :one
SELECT * FROM wallets
WHERE user_id = $1 LIMIT 1;

-- name: CreateTransaction :one
INSERT INTO transactions (
    wallet_id, amount, description, reference_id
) VALUES (
             $1, $2, $3, $4
         ) RETURNING *;

-- name: AddWalletBalance :one
UPDATE wallets
SET balance = balance + sqlc.arg(amount), -- SQLC will generate 'Amount' param
    updated_at = NOW()
WHERE user_id = sqlc.arg(user_id)
    RETURNING *;
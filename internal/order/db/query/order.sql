-- internal/order/db/query/order.sql

-- name: CreateOrder :one
INSERT INTO orders (
    id, passenger_id, driver_id,
    pickup_lat, pickup_long, dropoff_lat, dropoff_long,
    status, price
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9
         ) RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders
WHERE id = $1 LIMIT 1;

-- name: UpdateOrderStatus :exec
UPDATE orders
SET status = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpdateOrderDriver :exec
UPDATE orders
SET driver_id = $2, status = 'MATCHED', updated_at = NOW()
WHERE id = $1;
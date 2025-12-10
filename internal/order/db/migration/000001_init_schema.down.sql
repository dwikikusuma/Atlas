-- internal/order/db/migration/000001_init_schema.down.sql
-- Rollback for 000001_init_schema.up.sql

BEGIN;
DROP INDEX IF EXISTS idx_orders_driver;
DROP INDEX IF EXISTS idx_orders_passenger;
DROP TABLE IF EXISTS orders;
COMMIT;
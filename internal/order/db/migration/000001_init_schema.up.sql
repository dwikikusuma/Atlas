-- internal/order/db/migration/000001_init_schema.sql

CREATE TABLE orders
(
    id           VARCHAR(50) PRIMARY KEY,   -- We use string IDs like "RIDE-123"
    passenger_id VARCHAR(50)      NOT NULL,
    driver_id    VARCHAR(50)      NOT NULL,
    pickup_lat   DOUBLE PRECISION NOT NULL,
    pickup_long  DOUBLE PRECISION NOT NULL,
    dropoff_lat  DOUBLE PRECISION NOT NULL,
    dropoff_long DOUBLE PRECISION NOT NULL,
    status       VARCHAR(20)      NOT NULL, -- CREATED, MATCHED, STARTED, FINISHED
    price        DOUBLE PRECISION NOT NULL,
    created_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

-- Indexes for faster lookups
CREATE INDEX idx_orders_passenger ON orders (passenger_id);
CREATE INDEX idx_orders_driver ON orders (driver_id);
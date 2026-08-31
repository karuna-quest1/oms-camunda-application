-- Customer-facing dashboard read model.
-- Auto-applied on startup (spring.sql.init). Compatible with H2 and PostgreSQL.
CREATE TABLE IF NOT EXISTS orders_dashboard (
    order_id   VARCHAR(255) PRIMARY KEY,
    status     VARCHAR(50)  NOT NULL,
    updated_at TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

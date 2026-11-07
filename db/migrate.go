package db

import "github.com/jmoiron/sqlx"

func Migrate(database *sqlx.DB) error {
	_, err := database.Exec(schema)
	return err
}

const schema = `
CREATE TABLE IF NOT EXISTS stores (
	id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name          TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- theme_color が無い既存DBへの追加（冪等）
ALTER TABLE stores ADD COLUMN IF NOT EXISTS theme_color INT DEFAULT 1333408;

CREATE TABLE IF NOT EXISTS business_days (
	id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	store_id  UUID NOT NULL REFERENCES stores(id),
	opened_at TIMESTAMPTZ DEFAULT NOW(),
	closed_at TIMESTAMPTZ,
	is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS products (
	id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	store_id      UUID NOT NULL REFERENCES stores(id),
	name          TEXT NOT NULL,
	price         INT  NOT NULL,
	display_order INT  DEFAULT 0,
	is_active     BOOLEAN DEFAULT TRUE,
	created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
	id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	store_id              UUID NOT NULL REFERENCES stores(id),
	business_day_id       UUID REFERENCES business_days(id),
	order_number          TEXT NOT NULL,
	subtotal              INT  NOT NULL,
	discount_amount       INT  DEFAULT 0,
	manual_discount_amount INT DEFAULT 0,
	total_amount          INT  NOT NULL,
	status                TEXT NOT NULL DEFAULT 'pending',
	payment_method        TEXT,
	received_amount       INT,
	ordered_at            TIMESTAMPTZ DEFAULT NOW(),
	updated_at            TIMESTAMPTZ DEFAULT NOW()
);

-- manual_discount_amount が無い既存DBへの追加（冪等）
ALTER TABLE orders ADD COLUMN IF NOT EXISTS manual_discount_amount INT DEFAULT 0;

CREATE TABLE IF NOT EXISTS order_items (
	id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	order_id     UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	product_id   UUID REFERENCES products(id),
	product_name TEXT NOT NULL,
	unit_price   INT  NOT NULL,
	quantity     INT  NOT NULL
);

CREATE TABLE IF NOT EXISTS discount_rules (
	id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	store_id            UUID NOT NULL REFERENCES stores(id),
	name                TEXT NOT NULL,
	is_active           BOOLEAN DEFAULT TRUE,
	condition_type      TEXT NOT NULL,
	target_product_ids  TEXT[] DEFAULT '{}',
	min_quantity        INT DEFAULT 2,
	min_amount          INT DEFAULT 1000,
	discount_type       TEXT NOT NULL,
	discount_value      INT NOT NULL,
	created_at          TIMESTAMPTZ DEFAULT NOW()
);
`

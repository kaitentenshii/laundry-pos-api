CREATE TABLE laundry.orders (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 order_number TEXT GENERATED ALWAYS AS ('LD-' || id::text) STORED UNIQUE,
 customer_id BIGINT NOT NULL REFERENCES laundry.customers(id),
 customer_name TEXT NOT NULL,
 customer_whatsapp TEXT NOT NULL,
 customer_opt_in BOOLEAN NOT NULL,
 created_by BIGINT NOT NULL REFERENCES laundry.users(id),
 due_at TIMESTAMPTZ NOT NULL,
 notes TEXT NOT NULL DEFAULT '',
 status TEXT NOT NULL DEFAULT 'RECEIVED' CHECK (status IN ('RECEIVED','WASHING','COMPLETED','PICKED_UP')),
 payment_status TEXT NOT NULL DEFAULT 'UNPAID' CHECK (payment_status IN ('UNPAID','PAID')),
 total BIGINT NOT NULL CHECK (total BETWEEN 0 AND 9000000000000),
 request_key TEXT NOT NULL,
 request_hash TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE (created_by,request_key)
);
CREATE INDEX orders_customer_lookup ON laundry.orders(customer_id);
CREATE TABLE laundry.order_items (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 order_id BIGINT NOT NULL REFERENCES laundry.orders(id) ON DELETE CASCADE,
 service_id BIGINT NOT NULL REFERENCES laundry.services(id),
 price_version_id BIGINT NOT NULL REFERENCES laundry.service_price_versions(id),
 service_name TEXT NOT NULL,
 unit TEXT NOT NULL CHECK (unit IN ('KG','ITEM')),
 unit_price BIGINT NOT NULL CHECK (unit_price >= 0),
 quantity_milli BIGINT NOT NULL CHECK (quantity_milli BETWEEN 1 AND 1000000000 AND (unit='KG' OR quantity_milli % 1000 = 0)),
 subtotal BIGINT NOT NULL CHECK (subtotal BETWEEN 0 AND 9000000000000)
);
CREATE INDEX order_items_order_lookup ON laundry.order_items(order_id,id);

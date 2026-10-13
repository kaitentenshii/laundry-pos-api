CREATE TABLE laundry.payments (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 order_id BIGINT NOT NULL UNIQUE REFERENCES laundry.orders(id) ON DELETE CASCADE,
 method TEXT NOT NULL CHECK (method IN ('CASH','TRANSFER','QRIS')),
 amount_received BIGINT NOT NULL CHECK (amount_received BETWEEN 0 AND 9000000000000),
 amount_settled BIGINT NOT NULL CHECK (amount_settled BETWEEN 0 AND amount_received),
 change_amount BIGINT NOT NULL CHECK (change_amount >= 0 AND change_amount=CASE WHEN method='CASH' THEN amount_received-amount_settled ELSE 0 END),
 paid_at TIMESTAMPTZ NOT NULL,
 created_by BIGINT NOT NULL REFERENCES laundry.users(id),
 notes TEXT NOT NULL DEFAULT '',
 confirmed BOOLEAN NOT NULL CHECK (method='CASH' OR confirmed),
 request_key TEXT NOT NULL,
 request_hash TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

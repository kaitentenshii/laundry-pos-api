CREATE TABLE laundry.order_status_history (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 order_id BIGINT NOT NULL REFERENCES laundry.orders(id) ON DELETE CASCADE,
 from_status TEXT CHECK (from_status IN ('RECEIVED','WASHING','COMPLETED','PICKED_UP')),
 to_status TEXT NOT NULL CHECK (to_status IN ('RECEIVED','WASHING','COMPLETED','PICKED_UP')),
 changed_by BIGINT NOT NULL REFERENCES laundry.users(id),
 actor_name TEXT NOT NULL,
 changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 request_key TEXT NOT NULL,
 UNIQUE(order_id,request_key)
);
CREATE INDEX order_status_history_order ON laundry.order_status_history(order_id,id DESC);
INSERT INTO laundry.order_status_history(order_id,from_status,to_status,changed_by,actor_name,changed_at,request_key)
 SELECT o.id,NULL,'RECEIVED',o.created_by,u.name,o.created_at,'created:'||o.id::text
 FROM laundry.orders o JOIN laundry.users u ON u.id=o.created_by;

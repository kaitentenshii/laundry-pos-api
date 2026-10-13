CREATE UNIQUE INDEX order_status_history_once_per_state ON laundry.order_status_history(order_id,to_status);
ALTER TABLE laundry.order_status_history ADD CONSTRAINT status_history_valid_sequence CHECK (
 (from_status IS NULL AND to_status='RECEIVED') OR
 (from_status='RECEIVED' AND to_status='WASHING') OR
 (from_status='WASHING' AND to_status='COMPLETED') OR
 (from_status='COMPLETED' AND to_status='PICKED_UP')
);
CREATE FUNCTION laundry.reject_status_history_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'Order status history is immutable'; END $$;
CREATE TRIGGER status_history_immutable BEFORE UPDATE ON laundry.order_status_history
 FOR EACH ROW EXECUTE FUNCTION laundry.reject_status_history_update();

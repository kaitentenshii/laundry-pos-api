ALTER TABLE laundry.order_status_history ADD COLUMN handover_confirmed BOOLEAN NOT NULL DEFAULT FALSE;
-- Existing history stays factual; confirmation is mandatory for all new pickup records.
ALTER TABLE laundry.order_status_history ADD CONSTRAINT pickup_handover_confirmed
 CHECK (to_status <> 'PICKED_UP' OR handover_confirmed) NOT VALID;

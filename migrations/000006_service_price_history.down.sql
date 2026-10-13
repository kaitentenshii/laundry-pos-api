DROP TRIGGER service_price_history ON laundry.services;
DROP FUNCTION laundry.record_service_price_version();
ALTER TABLE laundry.services DROP COLUMN price_version_id;
DROP TABLE laundry.service_price_versions;
DROP FUNCTION laundry.reject_price_version_update();

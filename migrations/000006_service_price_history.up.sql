CREATE TABLE laundry.service_price_versions (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 service_id BIGINT NOT NULL REFERENCES laundry.services(id) ON DELETE CASCADE,
 name TEXT NOT NULL,
 unit TEXT NOT NULL CHECK (unit IN ('KG','ITEM')),
 price BIGINT NOT NULL CHECK (price BETWEEN 0 AND 999999999),
 estimated_hours INTEGER NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX service_price_versions_lookup ON laundry.service_price_versions(service_id,id DESC);
ALTER TABLE laundry.services ADD COLUMN price_version_id BIGINT REFERENCES laundry.service_price_versions(id) ON DELETE SET NULL;
INSERT INTO laundry.service_price_versions(service_id,name,unit,price,estimated_hours)
 SELECT id,name,unit,price,estimated_hours FROM laundry.services;
UPDATE laundry.services s SET price_version_id=v.id FROM laundry.service_price_versions v WHERE v.service_id=s.id;

CREATE FUNCTION laundry.record_service_price_version() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE version_id BIGINT;
BEGIN
 IF TG_OP='INSERT' THEN
  INSERT INTO laundry.service_price_versions(service_id,name,unit,price,estimated_hours)
   VALUES(NEW.id,NEW.name,NEW.unit,NEW.price,NEW.estimated_hours) RETURNING id INTO version_id;
  UPDATE laundry.services SET price_version_id=version_id WHERE id=NEW.id;
 ELSIF (OLD.name,OLD.unit,OLD.price,OLD.estimated_hours) IS DISTINCT FROM (NEW.name,NEW.unit,NEW.price,NEW.estimated_hours) THEN
  INSERT INTO laundry.service_price_versions(service_id,name,unit,price,estimated_hours)
   VALUES(NEW.id,NEW.name,NEW.unit,NEW.price,NEW.estimated_hours) RETURNING id INTO version_id;
  UPDATE laundry.services SET price_version_id=version_id WHERE id=NEW.id;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER service_price_history AFTER INSERT OR UPDATE ON laundry.services
 FOR EACH ROW EXECUTE FUNCTION laundry.record_service_price_version();
CREATE FUNCTION laundry.reject_price_version_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'Service price versions are immutable'; END $$;
CREATE TRIGGER price_version_immutable BEFORE UPDATE ON laundry.service_price_versions
 FOR EACH ROW EXECUTE FUNCTION laundry.reject_price_version_update();

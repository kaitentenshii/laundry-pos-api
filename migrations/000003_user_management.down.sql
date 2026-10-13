BEGIN;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM laundry.users WHERE email IS NULL) THEN
        RAISE EXCEPTION 'Cannot rollback user management while username-only accounts exist; assign email addresses first';
    END IF;
END $$;
ALTER TABLE laundry.users ALTER COLUMN email SET NOT NULL;
ALTER TABLE laundry.users DROP COLUMN last_active_at;
COMMIT;

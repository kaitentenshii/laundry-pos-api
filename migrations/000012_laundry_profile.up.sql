CREATE TABLE laundry.profile (
 id boolean PRIMARY KEY DEFAULT true CHECK(id),
 name text NOT NULL DEFAULT 'Laundry Mitra' CHECK(char_length(name) BETWEEN 1 AND 120),
 contact text NOT NULL DEFAULT '',
 operating_hours text NOT NULL DEFAULT '' CHECK(char_length(operating_hours)<=120),
 address text NOT NULL DEFAULT '' CHECK(char_length(address)<=1000),
 updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO laundry.profile(id) VALUES(true);

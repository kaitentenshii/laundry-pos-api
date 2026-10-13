CREATE TABLE laundry.customers (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
    whatsapp TEXT NOT NULL CHECK (whatsapp ~ '^\+[1-9][0-9]{7,14}$'),
    address TEXT NOT NULL DEFAULT '' CHECK (char_length(address) <= 1000),
    notes TEXT NOT NULL DEFAULT '' CHECK (char_length(notes) <= 2000),
    whatsapp_opt_in BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

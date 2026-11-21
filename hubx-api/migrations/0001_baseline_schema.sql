-- 0001 baseline schema: users, shipments and the ledger of updates, plus demo data.
-- Owner: A (foundation). Stories add their own migrations (NNNN_name.sql); never edit an applied one.

-- Participants. The email is stored normalized (trimmed, lower case) so sign-in can match it exactly.
CREATE TABLE users (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text        NOT NULL UNIQUE CHECK (email = lower(btrim(email)) AND email <> ''),
    name          text        NOT NULL CHECK (btrim(name) <> ''),
    role          text        NOT NULL CHECK (role IN ('operator', 'ground partner', 'customer')),
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- Shipments. stage and assigned_partner_id are a projection of the ledger kept for fast reads
-- (lists, counts, assigned shipments); the canonical history is the updates table.
CREATE TABLE shipments (
    id                  uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    code                text          NOT NULL UNIQUE,
    origin              text          NOT NULL CHECK (btrim(origin) <> ''),
    destination         text          NOT NULL CHECK (btrim(destination) <> ''),
    flight              text          NOT NULL CHECK (btrim(flight) <> ''),
    weight_kg           numeric(6, 1) NOT NULL CHECK (weight_kg BETWEEN 0.1 AND 5000),
    customer_id         uuid          NOT NULL REFERENCES users (id),
    stage               text          NOT NULL DEFAULT 'Created' CHECK (stage IN (
                            'Created', 'In flight', 'At destination airport',
                            'Assigned to partner', 'Out for delivery', 'Delivered')),
    assigned_partner_id uuid          REFERENCES users (id),
    created_by          uuid          NOT NULL REFERENCES users (id),
    created_at          timestamptz   NOT NULL DEFAULT now(),
    CHECK (lower(btrim(origin)) <> lower(btrim(destination)))
);

CREATE INDEX shipments_stage_created_at_idx ON shipments (stage, created_at DESC);
CREATE INDEX shipments_customer_idx ON shipments (customer_id);
CREATE INDEX shipments_partner_idx ON shipments (assigned_partner_id);

-- Ledger of updates (constitution Principle I). Each update stores the fingerprint of the previous one;
-- the first update of a shipment has previous_fingerprint NULL. The fingerprint formula lives in the
-- ledger contract (contracts/, item 7); this table only stores its result as lower-case hex SHA-256.
CREATE TABLE updates (
    id                   bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    shipment_id          uuid        NOT NULL REFERENCES shipments (id),
    sequence             integer     NOT NULL CHECK (sequence >= 1),
    stage                text        NOT NULL CHECK (stage IN (
                             'In flight', 'At destination airport',
                             'Assigned to partner', 'Out for delivery', 'Delivered')),
    responsible_party_id uuid        NOT NULL REFERENCES users (id),
    assigned_partner_id  uuid        REFERENCES users (id),
    recorded_at          timestamptz NOT NULL DEFAULT now(),
    previous_fingerprint char(64)    CHECK (previous_fingerprint ~ '^[0-9a-f]{64}$'),
    fingerprint          char(64)    NOT NULL UNIQUE CHECK (fingerprint ~ '^[0-9a-f]{64}$'),
    UNIQUE (shipment_id, sequence),
    CHECK ((sequence = 1) = (previous_fingerprint IS NULL)),
    CHECK ((stage = 'Assigned to partner') = (assigned_partner_id IS NOT NULL))
);

-- Immutability (cross-cutting rule 3): updates are never modified or deleted. This is defence in depth;
-- the application never issues UPDATE/DELETE on this table. The tamper test bypasses it on purpose with
-- SET session_replication_role = replica (superuser only).
CREATE FUNCTION updates_are_immutable() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'updates are immutable: % is not allowed', TG_OP;
END;
$$;

CREATE TRIGGER updates_immutable
    BEFORE UPDATE OR DELETE ON updates
    FOR EACH ROW EXECUTE FUNCTION updates_are_immutable();

-- Demo data. Every demo user's password is "Hubx-Demo-2026" (bcrypt, cost 12). Demo-only credentials:
-- they are public by design and must be rotated or removed outside the demo environment.
INSERT INTO users (id, email, name, role, password_hash) VALUES
    ('00000000-0000-4000-8000-000000000001', 'operator@hubx.co',    'Laura Operator',        'operator',       '$2a$12$D.Y8Bwuo1bnt3fmAWPujjuJu00X7sGR6bT5FDkfaXhQLjdvNf/PnK'),
    ('00000000-0000-4000-8000-000000000002', 'rapido@hubx.co',      'Transportes Rápido',    'ground partner', '$2a$12$D.Y8Bwuo1bnt3fmAWPujjuJu00X7sGR6bT5FDkfaXhQLjdvNf/PnK'),
    ('00000000-0000-4000-8000-000000000003', 'express@hubx.co',     'Logística Express',     'ground partner', '$2a$12$D.Y8Bwuo1bnt3fmAWPujjuJu00X7sGR6bT5FDkfaXhQLjdvNf/PnK'),
    ('00000000-0000-4000-8000-000000000004', 'ana@hubx.co',         'Ana Customer',          'customer',       '$2a$12$D.Y8Bwuo1bnt3fmAWPujjuJu00X7sGR6bT5FDkfaXhQLjdvNf/PnK'),
    ('00000000-0000-4000-8000-000000000005', 'andina@hubx.co',      'Distribuciones Andina', 'customer',       '$2a$12$D.Y8Bwuo1bnt3fmAWPujjuJu00X7sGR6bT5FDkfaXhQLjdvNf/PnK');

-- Shipments in the stage Created (no updates yet, per HU-02). Shipments in later stages need chained
-- updates and are seeded in 0002_seed_ledger.sql once the fingerprint formula is fixed (item 7, owner C).
INSERT INTO shipments (code, origin, destination, flight, weight_kg, customer_id, created_by, created_at) VALUES
    ('HBX-SEED-0001', 'Bogotá',   'Miami',     'AV-204', 120.0, '00000000-0000-4000-8000-000000000005', '00000000-0000-4000-8000-000000000001', now() - interval '3 days'),
    ('HBX-SEED-0002', 'Bogotá',   'Miami',     'AV-204',  35.5, '00000000-0000-4000-8000-000000000004', '00000000-0000-4000-8000-000000000001', now() - interval '2 days'),
    ('HBX-SEED-0003', 'Medellín', 'Madrid',    'IB-6740', 980.0, '00000000-0000-4000-8000-000000000005', '00000000-0000-4000-8000-000000000001', now() - interval '1 day'),
    ('HBX-SEED-0004', 'Cali',     'Panamá',    'CM-221',    0.1, '00000000-0000-4000-8000-000000000004', '00000000-0000-4000-8000-000000000001', now());

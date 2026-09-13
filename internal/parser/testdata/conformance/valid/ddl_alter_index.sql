ALTER TABLE imaginary_events
    ADD COLUMN archived boolean NOT NULL DEFAULT false;

CREATE INDEX imaginary_events_payload_idx
    ON imaginary_events USING gin (payload jsonb_path_ops)
    WHERE NOT archived;

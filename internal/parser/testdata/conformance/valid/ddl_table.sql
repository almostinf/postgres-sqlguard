CREATE TABLE imaginary_events (
    event_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id bigint NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    period tstzrange NOT NULL,
    EXCLUDE USING gist (tenant_id WITH =, period WITH &&)
);

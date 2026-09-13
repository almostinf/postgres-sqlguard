COPY imaginary_imports (account_id, display_name)
FROM '/tmp/sqlguard-synthetic-import.csv'
WITH (FORMAT csv, HEADER true, ON_ERROR ignore, LOG_VERBOSITY verbose);

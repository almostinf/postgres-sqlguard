INSERT INTO imaginary_accounts (account_id, display_name, enabled)
VALUES ($1, $2, TRUE), ($3, $4, FALSE)
ON CONFLICT (account_id) DO UPDATE
SET display_name = EXCLUDED.display_name
RETURNING account_id;

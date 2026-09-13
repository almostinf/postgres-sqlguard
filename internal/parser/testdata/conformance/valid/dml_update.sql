UPDATE imaginary_accounts
SET enabled = $1, updated_at = CURRENT_TIMESTAMP
WHERE account_id = $2
RETURNING account_id;

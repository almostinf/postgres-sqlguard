DELETE FROM imaginary_accounts
WHERE account_id = $1
RETURNING account_id;

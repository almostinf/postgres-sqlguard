SELECT account_id, display_name
FROM imaginary_accounts
WHERE tenant_id = $1
ORDER BY account_id
LIMIT 10;

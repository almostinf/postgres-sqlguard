SELECT 1 AS first_statement;
INSERT INTO imaginary_audit (event_id, message) VALUES (1, 'synthetic');
UPDATE imaginary_accounts SET enabled = false WHERE account_id = 7;
DELETE FROM imaginary_sessions WHERE expires_at < CURRENT_TIMESTAMP;

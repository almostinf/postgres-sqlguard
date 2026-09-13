WITH RECURSIVE
    first_declared AS (
        WITH
            first_nested AS (
                DELETE FROM imaginary_sessions
                WHERE account_id = 201
                RETURNING account_id
            ),
            second_nested AS (
                UPDATE imaginary_accounts
                SET enabled = false
                WHERE account_id = 202
                RETURNING account_id
            )
        UPDATE imaginary_accounts
        SET updated_at = CURRENT_TIMESTAMP
        WHERE account_id IN (SELECT account_id FROM first_nested)
        RETURNING account_id
    ),
    second_declared AS (
        DELETE FROM imaginary_sessions
        WHERE account_id = 203
        RETURNING account_id
    )
SELECT account_id FROM first_declared
UNION ALL
SELECT account_id FROM second_declared;

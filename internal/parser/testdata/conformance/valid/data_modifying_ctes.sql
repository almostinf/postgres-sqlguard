WITH
    inserted AS (
        INSERT INTO imaginary_accounts (account_id, display_name)
        VALUES (101, 'synthetic')
        RETURNING account_id
    ),
    updated_without_where AS (
        UPDATE imaginary_accounts
        SET enabled = true
        RETURNING account_id
    ),
    deleted_without_where AS (
        DELETE FROM imaginary_sessions
        RETURNING account_id
    )
SELECT account_id FROM inserted;

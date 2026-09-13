SELECT
    "select"."Mixed""Name",
    'DELETE FROM imaginary_accounts WHERE id = 1' AS ordinary_text,
    E'line one\nline two' AS escaped_text,
    U&'d\0061ta' AS unicode_text,
    $$UPDATE imaginary_accounts SET note = 'not syntax'$$ AS dollar_text,
    $body$DELETE FROM imaginary_accounts$body$ AS tagged_dollar_text
FROM "select" AS "select";

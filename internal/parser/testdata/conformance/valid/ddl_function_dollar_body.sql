CREATE FUNCTION imaginary_increment(value integer)
RETURNS integer
LANGUAGE sql
IMMUTABLE
RETURN value + 1;

CREATE FUNCTION imaginary_label(value text)
RETURNS text
LANGUAGE plpgsql
AS $function$
BEGIN
    RETURN 'label:' || value;
END
$function$;

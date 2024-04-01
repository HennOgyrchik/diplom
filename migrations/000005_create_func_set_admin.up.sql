CREATE OR REPLACE FUNCTION set_admin(tag_f character varying, old_id bigint, new_id bigint)
    RETURNS boolean
    LANGUAGE plpgsql
AS $function$

begin
    begin

        update members set admin = false where tag = tag_f and member_id = old_id;
        update members set admin = true where tag = tag_f and member_id = new_id;

        return true;

    exception WHEN OTHERS then
        return false;
    end;
end;
$function$
;

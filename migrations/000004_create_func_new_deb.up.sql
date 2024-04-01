CREATE OR REPLACE FUNCTION public.new_deb(tag_f character varying, summa double precision, comnt text, purp text, recpt text, create_d date, memb_id bigint)
    RETURNS boolean
    LANGUAGE plpgsql
AS $function$
declare
    cc_id integer;

begin
    begin
        insert into cash_collections (tag, sum, status, comment, create_date, close_date, purpose) values (tag_f, summa,'закрыт',comnt,create_d, create_d, purp) returning (id) into cc_id;
        insert into transactions (cash_collection_id, sum, type, status, receipt, member_id, date) values (cc_id,summa,'списание','подтвержден',recpt,memb_id,create_d);
        update funds set balance = balance - summa where tag = tag_f;
        return true;

    exception WHEN OTHERS then
        return false;
    end;
end;
$function$
;

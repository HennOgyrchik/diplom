CREATE OR REPLACE PROCEDURE check_debtors(IN cc_id integer)
    LANGUAGE plpgsql
AS $procedure$
declare
    debtors int;
begin
    debtors = (select count(*) from members where member_id not in (select member_id from transactions where cash_collection_id = cc_id and status = 'подтвержден'));

    if debtors=0
    then
        update cash_collections set status='закрыт', close_date=current_date where id=cc_id;
    else
        update cash_collections set status='открыт', close_date='0001-01-01' where id=cc_id;
    end if;

end;
$procedure$
;

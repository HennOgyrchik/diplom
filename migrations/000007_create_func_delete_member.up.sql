CREATE OR REPLACE PROCEDURE delete_member(IN tag_fund character varying, IN id_member bigint)
    LANGUAGE plpgsql
AS $procedure$
declare

    cc_id int4;
    cc_array int4[];

begin
    delete from members where tag = tag_fund and member_id = id_member;

    cc_array = array(select id from cash_collections where tag = tag_fund and status = 'открыт');

    delete from transactions where cash_collection_id = any(cc_array) and member_id = id_member and status !='подтвержден';

    FOREACH cc_id in array cc_array
        LOOP
            call check_debtors(cc_id);
        END LOOP;


end;
$procedure$
;

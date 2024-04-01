CREATE OR REPLACE PROCEDURE delete_fund(IN tag_fund character varying)
    LANGUAGE plpgsql
AS $procedure$
begin
    delete from funds f where tag = tag_fund;
    delete from members where tag = tag_fund;
    delete from transactions where cash_collection_id = (select id from cash_collections where tag = tag_fund);
    delete from cash_collections where tag = tag_fund;

end;
$procedure$
;

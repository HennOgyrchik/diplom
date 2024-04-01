
CREATE OR REPLACE FUNCTION update_sum_fund()
    RETURNS trigger
    LANGUAGE plpgsql
AS $function$
declare
    t varchar(15);
    s float8;

begin

    t = (select tag from cash_collections cc inner join transactions t on cc.id=t.cash_collection_id where t.id=new.id);
    s = (select balance from funds where tag=t);

    if old.type = 'пополнение' and new.type = 'пополнение'
    then
        if old.status = 'подтвержден' and new.status != 'подтвержден'
        then
            begin
                s=s-new.sum;
            end;
        elseif old.status != 'подтвержден' and new.status = 'подтвержден'
        then
            begin
                s=s+new.sum;
            end;
        end if;

        update funds set balance = s where tag = t;

    end if;

    return new;

end;
$function$
;

create trigger updt_transactions before
    update
    on
        transactions for each row execute function update_sum_fund();
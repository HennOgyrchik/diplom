package db

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
	"time"
)

const (
	StatusPaymentConfirmation = "подтвержден"
	StatusPaymentExpectation  = "ожидание"
	StatusPaymentRejection    = "отказ"
	StatusCashCollectionOpen  = "открыт"
	TypeTransactionDebiting   = "списание"
	NumberEntriesPerPage      = 3
)

type Repository struct {
	db      *pgxpool.Pool
	timeout time.Duration
}

func New(userDB *pgxpool.Pool, timeout time.Duration) *Repository {
	return &Repository{db: userDB, timeout: timeout}
}

// DoesTagExist возвращает true если тег существует
func (r *Repository) DoesTagExist(ctx context.Context, tag string) (ok bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	var count int

	err = r.db.QueryRow(ctx, "select count(*) from funds where tag=$1", tag).Scan(&count)
	if err == nil && count > 0 {
		ok = true
	}

	return
}

// CreateFund Создает новый фонд
func (r *Repository) CreateFund(ctx context.Context, tag string, balance float64) (err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	_, err = r.db.Exec(ctx, "insert into funds (tag,balance) values ($1,$2)", tag, balance)

	return
}

// GetAdminFund возвращает id адмнистратора фонда
func (r *Repository) GetAdminFund(ctx context.Context, tag string) (memberId int64, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	err = r.db.QueryRow(ctx, "select member_id from members where tag = $1 and admin = true", tag).Scan(&memberId)

	return
}

// ShowBalance возвращает баланс фонда
func (r *Repository) ShowBalance(ctx context.Context, tag string) (balance float64, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	err = r.db.QueryRow(ctx, "select balance from funds where tag=$1", tag).Scan(&balance)

	return

}

// DeleteFund удаляет фонд
func (r *Repository) DeleteFund(ctx context.Context, tag string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	_, err = r.db.Exec(ctx, "call delete_fund($1)", tag)

	return
}

// DeleteMember удаляет пользователя из фонда
func (r *Repository) DeleteMember(ctx context.Context, tag string, memberId int64) (err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	_, err = r.db.Exec(ctx, "call delete_member($1,$2)", tag, memberId)

	return
}

// GetTag возвращает тег фонда, в котором пользователь находится
func (r *Repository) GetTag(ctx context.Context, memberId int64) (tag string, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	err = r.db.QueryRow(ctx, "select tag from members where member_id=$1", memberId).Scan(&tag)

	return
}

// UpdateStatusCashCollection вызывает SQL функцию check_debtors
func (r *Repository) UpdateStatusCashCollection(ctx context.Context, idCashCollection int) (err error) {

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	_, err = r.db.Exec(ctx, "call check_debtors($1)", idCashCollection)

	return
}

// IsMember возвращает true если пользователь существует
func (r *Repository) IsMember(ctx context.Context, memberId int64) (ok bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	var count int
	err = r.db.QueryRow(ctx, "select count(*) from members where member_id=$1", memberId).Scan(&count)

	if err == nil && count != 0 {
		ok = true
	}

	return
}

// ChangeStatusTransaction обновляет статус транзакции
func (r *Repository) ChangeStatusTransaction(ctx context.Context, idTransaction int, status string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	_, err = r.db.Exec(ctx, "update transactions set status = $1 where id= $2", status, idTransaction)

	return
}

// SetAdmin меняет администратора
func (r *Repository) SetAdmin(ctx context.Context, tag string, old, new int64) (ok bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	err = r.db.QueryRow(ctx, "select * from set_admin($1, $2, $3)", tag, old, new).Scan(&ok)

	return
}

// GetDebtorsByCollection возвращает []Member, которые не оплатили cashCollectionId
func (r *Repository) GetDebtorsByCollection(ctx context.Context, cashCollectionId int) (debtors []Member, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	rows, err := r.db.Query(ctx, "select member_id from members where member_id not in (select member_id from transactions t where t.cash_collection_id = $1 and status = $2)", cashCollectionId, StatusPaymentConfirmation)
	if err != nil {
		return
	}

	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return
		}

		member, err := r.GetInfoAboutMember(ctx, id)
		if err != nil {
			return nil, err
		}
		debtors = append(debtors, member)
	}

	return
}

type Member struct {
	ID      int64
	Tag     string
	IsAdmin bool
	Login   string
	Name    string
}

// AddMember создает нового пользователя
func (r *Repository) AddMember(ctx context.Context, member Member) (err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	_, err = r.db.Exec(ctx, "insert into members (tag,member_id,admin,login,name) values ($1,$2,$3,$4,$5)", member.Tag, member.ID, member.IsAdmin, member.Login, member.Name)

	return
}

// GetMembers возвращает список пользователей фонда
func (r *Repository) GetMembers(ctx context.Context, tag string) (members []Member, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	rows, err := r.db.Query(ctx, "select member_id, tag, admin, login, name from members where tag =$1", tag)
	if err != nil {
		return
	}

	for rows.Next() {
		var member Member
		if err = rows.Scan(&member.ID, &member.Tag, &member.IsAdmin, &member.Login, &member.Name); err != nil {
			return
		}
		members = append(members, member)
	}
	return
}

// GetInfoAboutMember возвращает полную информацию о Member
func (r *Repository) GetInfoAboutMember(ctx context.Context, memberId int64) (member Member, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	member.ID = memberId

	err = r.db.QueryRow(ctx, "select tag,admin,login,name from members where member_id = $1", memberId).Scan(&member.Tag, &member.IsAdmin, &member.Login, &member.Name)

	return
}

type CashCollection struct {
	ID         int
	Tag        string
	Sum        float64
	Status     string
	Comment    string
	CreateDate time.Time
	CloseDate  time.Time
	Purpose    string
}

// CreateCashCollection создает новый CashCollection и возвращает его id
func (r *Repository) CreateCashCollection(ctx context.Context, cashCollection CashCollection) (id int, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	err = r.db.QueryRow(ctx, "insert into cash_collections (tag, sum, status, comment,purpose,create_date, close_date) values ($1,$2,$3,$4,$5,$6,$7) RETURNING id",
		cashCollection.Tag,
		cashCollection.Sum,
		cashCollection.Status,
		cashCollection.Comment,
		cashCollection.Purpose,
		cashCollection.CreateDate.Format(time.DateOnly),
		cashCollection.CloseDate.Format(time.DateOnly)).Scan(&id)

	return

}

// InfoAboutCashCollection возвращает полную информацию о CashCollection
func (r *Repository) InfoAboutCashCollection(ctx context.Context, idCashCollection int) (cc CashCollection, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	cc.ID = idCashCollection

	err = r.db.QueryRow(ctx, "select tag, sum, status, comment, create_date, close_date, purpose from cash_collections where id =$1", idCashCollection).Scan(
		&cc.Tag,
		&cc.Sum,
		&cc.Status,
		&cc.Comment,
		&cc.CreateDate,
		&cc.CloseDate,
		&cc.Purpose)

	return
}

// CreateDebitingFunds создает CashCollection типа "списание" и возвращает его id
func (r *Repository) CreateDebitingFunds(ctx context.Context, cashCollection CashCollection, memberID int64, receipt string) (ok bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	err = r.db.QueryRow(ctx, "select * from  new_deb($1, $2, $3,$4,$5,$6, $7)",
		cashCollection.Tag,
		cashCollection.Sum,
		cashCollection.Comment,
		cashCollection.Purpose,
		receipt,
		cashCollection.CreateDate.Format(time.DateOnly),
		memberID).Scan(&ok)

	return
}

// FindCashCollectionByStatus поиск всех CashCollection по тегу и статусу
func (r *Repository) FindCashCollectionByStatus(ctx context.Context, tag string, status string) (list []CashCollection, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	rows, err := r.db.Query(ctx, "select id, tag, sum, status, comment, create_date, close_date, purpose from cash_collections cc where cc.tag = $1 and cc.status =$2", tag, status)
	if err != nil {
		return
	}

	for rows.Next() {
		var cc CashCollection
		if err = rows.Scan(&cc.ID, &cc.Tag, &cc.Sum, &cc.Status, &cc.Comment, &cc.CreateDate, &cc.CloseDate, &cc.Purpose); err != nil {
			return
		}
		list = append(list, cc)
	}
	return
}

type HistoryData struct {
	Purpose string
	Sum     float64
	Date    time.Time
	Receipt string
}

// History возвращает список HistoryData порциями
func (r *Repository) History(ctx context.Context, tag string, page int) (list []HistoryData, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	rows, err := r.db.Query(ctx, "select cc.purpose, cc.sum, cc.create_date, t.receipt from cash_collections cc inner join transactions t on cc.id = t.cash_collection_id where t.type = $1 and cc.tag = $2 order by t.cash_collection_id desc limit $3 offset $4",
		TypeTransactionDebiting,
		tag,
		NumberEntriesPerPage,
		NumberEntriesPerPage*page)
	if err != nil {
		return
	}

	for rows.Next() {
		var hd HistoryData
		if err = rows.Scan(&hd.Purpose, &hd.Sum, &hd.Date, &hd.Receipt); err != nil {
			return
		}
		list = append(list, hd)
	}

	return
}

type Transaction struct {
	ID               int
	CashCollectionID int
	Sum              float64
	Type             string
	Status           string
	Receipt          string
	MemberID         int64
	Date             time.Time
}

// InfoAboutTransaction возвращает полную информацию о Transaction
func (r *Repository) InfoAboutTransaction(ctx context.Context, idTransaction int) (t Transaction, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	err = r.db.QueryRow(ctx, "select * from transactions where id = $1", idTransaction).Scan(&t.ID, &t.CashCollectionID, &t.Sum, &t.Type, &t.Status, &t.Receipt, &t.MemberID, &t.Date)

	return
}

// InsertInTransactions создает новую запись Transaction и возвращает его id
func (r *Repository) InsertInTransactions(ctx context.Context, transaction Transaction) (id int, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	err = r.db.QueryRow(ctx, "insert into transactions (cash_collection_id, sum, type, status,receipt, member_id, date) values ($1,$2,$3,$4,$5,$6,$7) RETURNING id",
		transaction.CashCollectionID,
		transaction.Sum,
		transaction.Type,
		transaction.Status,
		transaction.Receipt,
		transaction.MemberID,
		transaction.Date.Format(time.DateOnly)).Scan(&id)

	return
}

// Close закрывает подключение
func (r *Repository) Close() {
	r.db.Close()
}

type Payment struct {
	IDTransaction int
	Sum           float64
	Purpose       string
	Name          string
}

func (r *Repository) GetTransactionsByStatus(ctx context.Context, tag string, cashCollectionStatus string, transactionStatus string) (list []Payment, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	query := "select t.id, t.sum, cc.purpose, m.name " +
		"from cash_collections cc inner join transactions t on cc.id = t.cash_collection_id inner join members m on t.member_id = m.member_id" +
		" where cc.tag = $1 and cc.status = $2 and t.status = $3"

	rows, err := r.db.Query(ctx, query, tag, cashCollectionStatus, transactionStatus)
	if err != nil {
		return
	}

	for rows.Next() {
		var p Payment
		if err = rows.Scan(&p.IDTransaction, &p.Sum, &p.Purpose, &p.Name); err != nil {
			return
		}
		list = append(list, p)
	}
	return

}

func (r *Repository) GetPaymentByTransactionID(ctx context.Context, trancastionID int) (payment Payment, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	query := "select t.id, t.sum, cc.purpose, m.name from cash_collections cc inner join transactions t on cc.id = t.cash_collection_id inner join members m on t.member_id = m.member_id where t.id = $1"

	err = r.db.QueryRow(ctx, query, trancastionID).Scan(&payment)
	return
}

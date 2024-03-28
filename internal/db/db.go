package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	_ "regexp"
	"strings"
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

type ConnString string

func dbConnection(connStr ConnString) (*sql.DB, error) {
	db, err := sql.Open("postgres", string(connStr))

	if err != nil {
		_ = db.Close()
	}

	return db, err
}

func NewDBConnString(socket, dbName, user, password, sslMode string) (ConnString, error) {
	addr := strings.Split(socket, ":")
	if len(addr) != 2 {
		return "", InvalidAddressError
	}
	return ConnString(fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s host=%s port=%s", user, password, dbName, sslMode, addr[0], addr[1])), nil
}

func (connStr ConnString) DoesTagExist(tag string) (bool, error) {
	db, err := dbConnection(connStr)
	if err != nil {
		return false, err
	}
	defer db.Close()

	stmt, err := db.Prepare("select count(*) from funds where tag=$1")
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	var count int
	err = stmt.QueryRow(tag).Scan(&count)
	switch {
	case err != nil:
		return false, err
	case count > 0:
		return true, err
	default:
		return false, err
	}
}

func (connStr ConnString) CreateFund(tag string, balance float64) error {
	db, err := dbConnection(connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("insert into funds (tag,balance) values ($1,$2)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_ = stmt.QueryRow(tag, balance)
	return err
}

func (connStr ConnString) GetAdminFund(tag string) (int64, error) {
	db, err := dbConnection(connStr)
	if err != nil {
		return -1, err
	}
	defer db.Close()

	stmt, err := db.Prepare("select member_id from members where tag = $1 and admin = true")
	if err != nil {
		return -1, err
	}
	defer stmt.Close()

	var memberId int64

	err = stmt.QueryRow(tag).Scan(&memberId)
	return memberId, err
}

// ShowBalance возвращает баланс фонда
func (connStr ConnString) ShowBalance(tag string) (float64, error) {
	db, err := dbConnection(connStr)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	stmt, err := db.Prepare("select balance from funds where tag=$1")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var balance float64
	err = stmt.QueryRow(tag).Scan(&balance)
	return balance, err
}

// DeleteFund удаляет фонд
func (connStr ConnString) DeleteFund(tag string) error {
	db, err := dbConnection(connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("call delete_fund($1)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_ = stmt.QueryRow(tag)

	return err
}

func (connStr ConnString) DeleteMember(tag string, memberId int64) error {
	db, err := dbConnection(connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("call delete_member($1,$2)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_ = stmt.QueryRow(tag, memberId)

	return nil
}

// GetTag возвращает тег фонда, в котором пользователь находится
func (connStr ConnString) GetTag(memberId int64) (string, error) {
	db, err := dbConnection(connStr)
	if err != nil {
		return "", err
	}
	defer db.Close()

	stmt, err := db.Prepare("select tag from members where member_id=$1")
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	var tag string
	err = stmt.QueryRow(memberId).Scan(&tag)
	return tag, err
}

func (connStr ConnString) UpdateStatusCashCollection(idCashCollection int) error {
	db, err := dbConnection(connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("call check_debtors($1)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_ = stmt.QueryRow(idCashCollection)

	return nil
}

func (connStr ConnString) IsMember(memberId int64) (bool, error) {
	db, err := dbConnection(connStr)
	if err != nil {
		return false, err
	}
	defer db.Close()

	stmt, err := db.Prepare("select count(*) from members where member_id=$1")
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	var count int
	err = stmt.QueryRow(memberId).Scan(&count)

	if (err != nil) || (count == 0) {
		return false, err
	}

	return true, nil
}

func (connStr ConnString) ChangeStatusTransaction(idTransaction int, status string) error {
	db, err := dbConnection(connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("update transactions set status = $1 where id= $2")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_ = stmt.QueryRow(status, idTransaction)

	return nil
}

func (connStr ConnString) IsAdmin(memberId int64) (bool, error) {
	db, err := dbConnection(connStr)
	if err != nil {
		return false, err
	}
	defer db.Close()

	stmt, err := db.Prepare("select admin from members m  where member_id=$1")
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	var result bool
	err = stmt.QueryRow(memberId).Scan(&result)
	return result, err
}

func (connStr ConnString) SetAdmin(tag string, old, new int64) (ok bool, err error) {
	db, err := dbConnection(connStr)
	if err != nil {
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("select * from set_admin($1, $2, $3)")
	if err != nil {
		return
	}
	defer stmt.Close()

	_ = stmt.QueryRow(tag, old, new).Scan(&ok)
	return
}

func (connStr ConnString) GetDebtorsByCollection(cashCollectionId int) ([]int64, error) {
	var members []int64

	db, err := dbConnection(connStr)
	if err != nil {
		return members, err
	}
	defer db.Close()

	stmt, err := db.Prepare("select member_id from members where member_id not in (select member_id from transactions t where t.cash_collection_id = $1 and status = $2)")
	if err != nil {
		return members, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(cashCollectionId, StatusPaymentConfirmation)
	if err != nil {
		return members, err
	}

	for rows.Next() {
		var memberId int64
		if err = rows.Scan(&memberId); err != nil {
			return members, err
		}
		members = append(members, memberId)
	}

	return members, nil
}

type Member struct {
	ID      int64
	Tag     string
	IsAdmin bool
	Login   string
	Name    string
}

func (connStr ConnString) AddMember(member Member) error {
	db, err := dbConnection(connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("insert into members (tag,member_id,admin,login,name) values ($1,$2,$3,$4,$5)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_ = stmt.QueryRow(member.Tag, member.ID, member.IsAdmin, member.Login, member.Name)
	return err
}

// GetMembers возвращает список пользователей фонда
func (connStr ConnString) GetMembers(tag string) ([]Member, error) {
	var members []Member
	db, err := dbConnection(connStr)
	if err != nil {
		return members, err
	}
	defer db.Close()

	stmt, err := db.Prepare("select member_id, tag, admin, login, name from members where tag =$1")
	if err != nil {
		return members, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(tag)
	if err != nil {
		return members, err
	}
	defer rows.Close()

	for rows.Next() {
		var member Member
		if err = rows.Scan(&member.ID, &member.Tag, &member.IsAdmin, &member.Login, &member.Name); err != nil {
			return members, err
		}
		members = append(members, member)
	}
	return members, nil
}

// GetInfoAboutMember возвращает полную информацию о пользователе
func (connStr ConnString) GetInfoAboutMember(memberId int64) (Member, error) {
	member := Member{ID: memberId}

	db, err := dbConnection(connStr)
	if err != nil {
		return member, err
	}
	defer db.Close()

	stmt, err := db.Prepare("select tag,admin,login,name from members where member_id = $1")
	if err != nil {
		return member, err
	}
	defer stmt.Close()

	err = stmt.QueryRow(memberId).Scan(&member.Tag, &member.IsAdmin, &member.Login, &member.Name)
	return member, err
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

func (connStr ConnString) CreateCashCollection(cashCollection CashCollection) (int, error) {
	db, err := dbConnection(connStr)
	if err != nil {
		return -1, err
	}
	defer db.Close()

	var stmt *sql.Stmt
	var id int

	stmt, err = db.Prepare("insert into cash_collections (tag, sum, status, comment,purpose,create_date, close_date) values ($1,$2,$3,$4,$5,$6,$7) RETURNING id")
	if err != nil {
		return -1, err
	}
	err = stmt.QueryRow(cashCollection.Tag, cashCollection.Sum, cashCollection.Status, cashCollection.Comment, cashCollection.Purpose, cashCollection.CreateDate.Format(time.DateOnly), cashCollection.CloseDate.Format(time.DateOnly)).Scan(&id)

	_ = stmt.Close()
	return id, nil

}

func (connStr ConnString) InfoAboutCashCollection(idCashCollection int) (CashCollection, error) {
	cc := CashCollection{ID: idCashCollection}

	db, err := dbConnection(connStr)
	if err != nil {
		return cc, err
	}
	defer db.Close()

	stmt, err := db.Prepare("select tag, sum, status, comment, create_date, close_date, purpose from cash_collections where id =$1")
	if err != nil {

		return cc, err
	}
	defer stmt.Close()

	err = stmt.QueryRow(idCashCollection).Scan(&cc.Tag, &cc.Sum, &cc.Status, &cc.Comment, &cc.CreateDate, &cc.CloseDate, &cc.Purpose)
	return cc, err
}

func (connStr ConnString) CreateDebitingFunds(cashCollection CashCollection, memberID int64, receipt string) (ok bool, err error) {
	db, err := dbConnection(connStr)
	if err != nil {
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("select * from  new_deb($1, $2, $3,$4,$5,$6, $7)")
	if err != nil {
		return
	}

	err = stmt.QueryRow(cashCollection.Tag, cashCollection.Sum, cashCollection.Comment, cashCollection.Purpose, receipt, cashCollection.CreateDate.Format(time.DateOnly), memberID).Scan(&ok)
	return
}

func (connStr ConnString) FindCashCollectionByStatus(tag string, status string) ([]CashCollection, error) {
	var list []CashCollection

	db, err := dbConnection(connStr)
	if err != nil {
		return list, err
	}
	defer db.Close()

	stmt, err := db.Prepare("select id, tag, sum, status, comment, create_date, close_date, purpose from cash_collections cc where cc.tag = $1 and cc.status =$2")
	if err != nil {

		return list, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(tag, status)
	if err != nil {
		return list, err
	}
	defer rows.Close()

	for rows.Next() {
		var cc CashCollection
		if err = rows.Scan(&cc.ID, &cc.Tag, &cc.Sum, &cc.Status, &cc.Comment, &cc.CreateDate, &cc.CloseDate, &cc.Purpose); err != nil {
			return list, err
		}
		list = append(list, cc)
	}
	return list, nil
}

type HistoryData struct {
	Purpose string
	Sum     float64
	Date    time.Time
	Receipt string
}

func (connStr ConnString) History(tag string, page int) ([]HistoryData, error) {
	db, err := dbConnection(connStr)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	stmt, err := db.Prepare("select cc.purpose, cc.sum, cc.create_date, t.receipt from cash_collections cc inner join transactions t on cc.id =t.cash_collection_id where t.type = $1 and cc.tag = $2 order by t.cash_collection_id desc limit $3 offset $4")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(TypeTransactionDebiting, tag, NumberEntriesPerPage, NumberEntriesPerPage*page)
	if err != nil {
		return nil, err
	}

	var list []HistoryData

	for rows.Next() {
		var hd HistoryData
		if err = rows.Scan(&hd.Purpose, &hd.Sum, &hd.Date, &hd.Receipt); err != nil {
			return list, err
		}
		list = append(list, hd)
	}

	return list, nil
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

func (connStr ConnString) InfoAboutTransaction(idTransaction int) (Transaction, error) {
	db, err := dbConnection(connStr)
	if err != nil {
		return Transaction{}, err
	}
	defer db.Close()

	stmt, err := db.Prepare("select * from transactions where id = $1")
	if err != nil {
		return Transaction{}, err
	}
	defer stmt.Close()

	var t Transaction
	err = stmt.QueryRow(idTransaction).Scan(&t.ID, &t.CashCollectionID, &t.Sum, &t.Type, &t.Status, &t.Receipt, &t.MemberID, &t.Date)

	return t, err
}

func (connStr ConnString) InsertInTransactions(transaction Transaction) (int, error) {
	db, err := dbConnection(connStr)
	if err != nil {
		return -1, err
	}
	defer db.Close()

	stmt, err := db.Prepare("insert into transactions (cash_collection_id, sum, type, status,receipt, member_id, date) values ($1,$2,$3,$4,$5,$6,$7) RETURNING id")
	if err != nil {
		return -1, err
	}

	var id int
	_ = stmt.QueryRow(transaction.CashCollectionID, transaction.Sum, transaction.Type, transaction.Status, transaction.Receipt, transaction.MemberID, transaction.Date.Format(time.DateOnly)).Scan(&id)
	return id, nil
}

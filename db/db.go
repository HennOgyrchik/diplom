package db

import (
	"database/sql"
	_ "github.com/lib/pq"
	"regexp"
	_ "regexp"
)

func dbConnection() (*sql.DB, error) {
	connStr := "user=postgres password=111 dbname=postgres sslmode=disable host=192.168.0.116 port=5432" //как то убрать логин и пароль, заменить ip на имя контейнера
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		_ = db.Close()
	}

	return db, err
}

func IsMember(memberId int64) (bool, error) {
	db, err := dbConnection()
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

func IsAdmin(memberId int64) (bool, error) {
	db, err := dbConnection()
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

func DoesTagExist(tag string) (bool, error) {
	db, err := dbConnection()
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

func CreateFund(tag string, balance float64) error {
	db, err := dbConnection()
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

func AddMember(tag string, memberId int64, isAdmin bool, login string, name string) error {
	db, err := dbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("insert into members (tag,member_id,admin,login,name) values ($1,$2,$3,$4,$5)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_ = stmt.QueryRow(tag, memberId, isAdmin, login, name)
	return err
}

// DeleteFund удаляет фонд
func DeleteFund(tag string) error {
	db, err := dbConnection()
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

// GetTag возвращает тег фонда, в котором пользователь находится
func GetTag(memberId int64) (string, error) {
	db, err := dbConnection()
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

// ShowBalance возвращает баланс фонда
func ShowBalance(tag string) (float64, error) {
	db, err := dbConnection()
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

// GetMembers возвращает список id пользователей фонда
func GetMembers(tag string) ([]int64, error) {
	var members []int64
	db, err := dbConnection()
	if err != nil {
		return members, err
	}
	defer db.Close()

	stmt, err := db.Prepare("select member_id from members where tag =$1")
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
		var member int64
		if err = rows.Scan(&member); err != nil {
			return members, err
		}
		members = append(members, member)
	}
	return members, nil
}

// GetInfoAboutMember возвращает полную информацию о пользователе
func GetInfoAboutMember(memberId int64) (isAdmin bool, login string, name string, err error) {
	db, err := dbConnection()
	if err != nil {
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("select admin,login,name from members where member_id = $1")
	if err != nil {
		return
	}
	defer stmt.Close()

	err = stmt.QueryRow(memberId).Scan(&isAdmin, &login, &name)
	return
}

func CreateCashCollection(tag string, sum float64, status string, comment string, purpose string, closingDate string) (id int, err error) {
	db, err := dbConnection()
	if err != nil {
		return
	}
	defer db.Close()

	var datePat = regexp.MustCompile(`^\d-\d-\d.`)
	var stmt *sql.Stmt

	if datePat.MatchString(closingDate) {
		stmt, err = db.Prepare("insert into cash_collections (tag, sum, status, comment,purpose, close_date) values ($1,$2,$3,$4,$5,$6) RETURNING id")
		if err != nil {
			return
		}
		err = stmt.QueryRow(tag, sum, status, comment, purpose, closingDate).Scan(&id)
	} else {
		stmt, err = db.Prepare("insert into cash_collections (tag, sum, status, comment,purpose) values ($1,$2,$3,$4,$5) RETURNING id")
		if err != nil {
			return
		}
		err = stmt.QueryRow(tag, sum, status, comment, purpose).Scan(&id)
	}

	_ = stmt.Close()
	return

}

func InfoAboutCashCollection(idCollection int) (sum float64, purpose string, err error) {
	db, err := dbConnection()
	if err != nil {
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("select sum, purpose from cash_collections where id =$1")
	if err != nil {
		return
	}
	defer stmt.Close()

	err = stmt.QueryRow(idCollection).Scan(&sum, &purpose)
	return
}

//
//func ExistsFund(tag string) (result bool, err error) {
//	result = false
//
//	db, err := dbConnection()
//	if err != nil {
//		return
//	}
//	defer db.Close()
//
//	stmt, err := db.Prepare("select count(*) from funds where tag=$1")
//	if err != nil {
//		return
//	}
//
//	var count int
//	err = stmt.QueryRow(tag).Scan(&count)
//
//	if (err != nil) || (count == 0) {
//		return
//	}
//
//	result = true
//	return
//}

//func InsertInTransactions(cashCollectionId int, sum float64, typeOfTransaction string, status string, pathToReceipt string, memberId int64) (id int, err error) {
//	id = -1
//	db, err := dbConnection()
//	if err != nil {
//		return
//	}
//	defer db.Close()
//
//	stmt, err := db.Prepare("insert into transactions (cash_collection_id, sum, type, status,receipt, member_id) values ($1,$2,$3,$4,$5,$6) RETURNING id")
//	if err != nil {
//		return
//	}
//	_ = stmt.QueryRow(cashCollectionId, sum, typeOfTransaction, status, pathToReceipt, memberId).Scan(&id)
//	return
//}
//
//func GetAdminFund(tag string) (memberId int64, err error) {
//	db, err := dbConnection()
//	if err != nil {
//		return
//	}
//	defer db.Close()
//
//	stmt, err := db.Prepare("select member_id from members where tag_fund = $1 and admin = true")
//	if err != nil {
//		return
//	}
//
//	err = stmt.QueryRow(tag).Scan(&memberId)
//	return
//}
//
//func InfoAboutTransaction(idTransaction int) (status string, typeOfTransaction string, pathToReceipt string, memberId int64, sum float64, err error) {
//	db, err := dbConnection()
//	if err != nil {
//		return
//	}
//	defer db.Close()
//
//	stmt, err := db.Prepare("select status,type,receipt,member_id, sum from transactions where id = $1")
//	if err != nil {
//		return
//	}
//
//	err = stmt.QueryRow(idTransaction).Scan(&status, &typeOfTransaction, &pathToReceipt, &memberId, &sum)
//	return
//}

//
//func ChangeStatusTransaction(idTransaction int, status string) error {
//	db, err := dbConnection()
//	if err != nil {
//		return err
//	}
//	defer db.Close()
//
//	stmt, err := db.Prepare("update transactions set status = $1 where id= $2")
//	if err != nil {
//		return err
//	}
//
//	_ = stmt.QueryRow(status, idTransaction)
//
//	return nil
//}
//
//func CreateDebitingFunds(memberId int64, tag string, sum float64, comment string, purpose string, receipt string) (ok bool, err error) {
//	db, err := dbConnection()
//	if err != nil {
//		return
//	}
//	defer db.Close()
//
//	stmt, err := db.Prepare("select * from  new_deb($1, $2, $3,$4,$5, $6)")
//	if err != nil {
//		return
//	}
//
//	err = stmt.QueryRow(tag, sum, comment, purpose, receipt, memberId).Scan(&ok)
//	return
//}
//
//func GetDebtors(idCashCollection int) (members []int64, err error) {
//	db, err := dbConnection()
//	if err != nil {
//		return
//	}
//	defer db.Close()
//
//	stmt, err := db.Prepare("select member_id from members where member_id not in (select member_id  from transactions where cash_collection_id =$1 and status = 'подтвержден')")
//	if err != nil {
//		return
//	}
//
//	rows, err := stmt.Query(idCashCollection)
//	if err != nil {
//		return
//	}
//	defer rows.Close()
//
//	for rows.Next() {
//		var member int64
//		if err = rows.Scan(&member); err != nil {
//			return
//		}
//		members = append(members, member)
//	}
//	return
//}

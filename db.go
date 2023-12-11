package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

func dbConnection() (*sql.DB, error) {
	connStr := "user=postgres password=111 dbname=postgres sslmode=disable host=192.168.0.103 port=5432" //как то убрать логин и пароль, заменить ip на имя контейнера
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		fmt.Println(err)
		db.Close()
	}

	return db, err
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
	_ = stmt.QueryRow(tag, balance)
	return err
}

func DoesTagExist(tag string) (result bool, err error) {
	result = false

	db, err := dbConnection()
	if err != nil {
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("select count(*) from funds where tag=$1")
	if err != nil {
		return
	}

	var count int
	err = stmt.QueryRow(tag).Scan(&count)
	if (err != nil) || (count != 0) {
		return
	}

	result = true
	return
}

func AddMember(tag string, memberId int64, isAdmin bool) error {
	db, err := dbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("insert into members (tag_fund,member,admin) values ($1,$2,$3)")
	if err != nil {
		return err
	}
	_ = stmt.QueryRow(tag, memberId, isAdmin)
	return err
}

func IsMember(memberId int64) (result bool, err error) {
	result = false

	db, err := dbConnection()
	if err != nil {
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("select count(*) from members where member=$1")
	if err != nil {
		return
	}

	var count int
	err = stmt.QueryRow(memberId).Scan(&count)

	if (err != nil) || (count == 0) {
		return
	}

	result = true
	return
}

func ExistsFund(tag string) (result bool, err error) {
	result = false

	db, err := dbConnection()
	if err != nil {
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("select count(*) from funds where tag=$1")
	if err != nil {
		return
	}

	var count int
	err = stmt.QueryRow(tag).Scan(&count)

	if (err != nil) || (count == 0) {
		return
	}

	result = true
	return
}

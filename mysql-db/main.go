package main

import (
	"database/sql"
	"example.com/kendrick/protocol"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

var db *sql.DB

func getUsers() *sql.Rows {
	res, err := db.Query("SELECT * FROM users")
	if err != nil {
		log.Panicln(err)
	}
	return res
}

func connect() *sql.DB {
	database, err := sql.Open("mysql", "root:MonsterHunter!@@tcp(localhost:3306)/users_db")
	if err != nil {
		log.Panicln(err.Error())
	}
	log.Println("Connected to MySQL database")
	return database
}

func main() {
	db = connect()
	defer db.Close()

	rows := getUsers()
	for rows.Next() {
		var user protocol.User
		err := rows.Scan(&user.Nickname, &user.Username, &user.PwHash, &user.ProfilePic)
		if err != nil {
			log.Panicln(err)
		}
		log.Println(user)
	}
}

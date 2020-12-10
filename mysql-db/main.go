package database

import (
	"database/sql"
	"example.com/kendrick/protocol"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

var db *sql.DB

// Converts sql return statement to slice of User
func rowsToUser(rows *sql.Rows) []protocol.User {
	const DEFAULT_SIZE = 10
	ret := make([]protocol.User, DEFAULT_SIZE)
	for rows.Next() {
		var user protocol.User
		err := rows.Scan(&user.Nickname, &user.Username, &user.PwHash, &user.ProfilePic)
		if err != nil {
			log.Panicln(err)
		}
		ret = append(ret, user)
	}
	return ret
}

func GetUsers() []protocol.User {
	res, err := db.Query("SELECT * FROM users")
	if err != nil {
		log.Panicln(err)
	}
	return rowsToUser(res)
}

// Retrieves a user based on key (his unique username)
func GetUser(key string) []protocol.User {
	res, err := db.Query("SELECT * FROM users WHERE username = ?", key)
	if err != nil {
		log.Panicln(err)
	}
	return rowsToUser(res)
}

func connect() *sql.DB {
	database, err := sql.Open("mysql", "root:MonsterHunter!@@tcp(localhost:3306)/users_db")
	if err != nil {
		log.Panicln(err.Error())
	}
	log.Println("Connected to MySQL database")
	return database
}

func Connect() {
	db = connect()
}

func Disconnect() {
	_ = db.Close()
}

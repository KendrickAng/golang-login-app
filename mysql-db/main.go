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
	var ret []protocol.User
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
	ensureConnected(db)
	res, err := db.Query("SELECT * FROM users")
	if err != nil {
		log.Panicln(err)
	}
	return rowsToUser(res)
}

func EditUser(key string, nickname string, picPath string) int64 {
	ensureConnected(db)
	result, err := db.Exec("UPDATE users SET nickname=?, profile_pic=? WHERE username=?", nickname, picPath, key)
	if err != nil {
		log.Panicln(err)
	}
	log.Println("UPDATE: username: " + key + " | nickname: " + nickname + " | profile_pic: " + picPath)
	var rows int64
	rows, err = result.RowsAffected()
	return rows
}

// Retrieves a user based on key (his unique username)
func GetUser(key string) []protocol.User {
	ensureConnected(db)
	res, err := db.Query("SELECT * FROM users WHERE username = ?", key)
	if err != nil {
		log.Panicln(err)
	}
	return rowsToUser(res)
}

func connect() {
	var err error
	db, err = sql.Open("mysql", "root:MonsterHunter!@@tcp(localhost:3306)/users_db")
	if err != nil {
		log.Panicln(err.Error())
	}
	log.Println("Connected to MySQL database")
}

func disconnect() {
	_ = db.Close()
}

func ensureConnected(ptr *sql.DB) {
	if ptr == nil {
		connect()
	}
	if db == nil {
		panic("Not connected to MySQL database!")
	}
}

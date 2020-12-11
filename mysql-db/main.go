package database

import (
	"database/sql"
	"example.com/kendrick/protocol"
	"example.com/kendrick/tcp-server/fileio"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

var db *sql.DB
var pw string = ""

// https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html
const (
	DUP_PKEY = 1062
)

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

// TODO: Not sure whether this works
func CreateUser(username string, pwHash string, nickname string) int64 {
	ensureConnected(db)
	result, err := db.Exec("INSERT INTO users VALUES (?, ?, ?, ?)", username, nickname, pwHash, sql.NullString{})
	if err != nil {
		// duplicate username pkey
		if me, ok := err.(*mysql.MySQLError); ok && me.Number == DUP_PKEY {
			return 0
		}
		log.Panicln(err)
	}
	log.Println("INSERT: username: " + username + " | nickname: " + nickname + " | pwHash " + pwHash)
	var rows int64
	rows, err = result.RowsAffected()
	return rows
}

// Retrieves a user based on key (his unique username)
func GetUser(key string) []protocol.User {
	ensureConnected(db)
	res, err := db.Query("SELECT username, nickname, pw_hash, COALESCE(profile_pic, '') FROM users WHERE username = ?", key)
	if err != nil {
		log.Panicln(err)
	}
	return rowsToUser(res)
}

func connect() {
	var err error
	readPw()
	db, err = sql.Open("mysql", "root:"+pw+"@tcp(localhost:3306)/users_db")
	if err != nil {
		log.Panicln(err.Error())
	}
	log.Println("Connected to MySQL database")
}

func disconnect() {
	_ = db.Close()
}

func readPw() {
	if pw == "" {
		pw = fileio.ReadPw()
	}
}

func ensureConnected(ptr *sql.DB) {
	if ptr == nil {
		connect()
	}
	if db == nil {
		panic("Not connected to MySQL database!")
	}
}

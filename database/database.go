package database

import (
	"database/sql"
	"example.com/kendrick/common"
	"example.com/kendrick/fileio"
	"example.com/kendrick/protocol"
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
func rowsToUsers(rows *sql.Rows) []protocol.User {
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

func rowsToSessions(rows *sql.Rows) []protocol.Session {
	var ret []protocol.Session
	for rows.Next() {
		var sess protocol.Session
		err := rows.Scan(&sess.Uuid, &sess.Username)
		if err != nil {
			log.Panicln(err)
		}
		ret = append(ret, sess)
	}
	return ret
}

func UpdateUser(key string, nickname string, picPath string) int64 {
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

func InsertUser(username string, pwHash string, nickname string) int64 {
	ensureConnected(db)
	result, err := db.Exec("INSERT INTO users VALUES (?, ?, ?, ?)", username, nickname, pwHash, sql.NullString{})
	if err != nil {
		// duplicate username pkey
		if me, ok := err.(*mysql.MySQLError); ok && me.Number == DUP_PKEY {
			return 0
		}
		log.Panicln(err)
	}
	log.Println("INSERT users: username: " + username + " | nickname: " + nickname + " | pwHash " + pwHash)
	rows, err := result.RowsAffected()
	return rows
}

// Retrieves a user based on key (his unique username)
func GetUser(key string) []protocol.User {
	ensureConnected(db)
	res, err := db.Query("SELECT username, nickname, pw_hash, COALESCE(profile_pic, '') FROM users WHERE username = ?", key)
	if err != nil {
		log.Panicln(err)
	}
	ret := rowsToUsers(res)
	if len(ret) > 1 {
		log.Panicln("GET USER: too many rows for key " + key)
	}
	return ret
}

func InsertSession(uuid string, username string) int64 {
	ensureConnected(db)
	result, err := db.Exec("INSERT INTO sessions VALUES (?, ?)", uuid, username)
	if common.IsError(err) {
		return 0
	}
	log.Println("INSERT SESSION: uuid: " + uuid + " | username: " + username)
	rows, err := result.RowsAffected()
	if rows != 1 {
		common.Print("CREATE SESSION: one row not inserted!")
	}
	return rows
}

func DeleteSession(uuid string) int64 {
	ensureConnected(db)
	result, err := db.Exec("DELETE FROM sessions WHERE uuid = ?", uuid)
	if err != nil {
		log.Panicln(err)
	}
	log.Println("DELETE sessions: uuid: " + uuid)
	rows, err := result.RowsAffected()
	return rows
}

func GetSession(uuid string) []protocol.Session {
	ensureConnected(db)
	res, err := db.Query("SELECT uuid, username FROM sessions WHERE uuid=?", uuid)
	if err != nil {
		log.Panicln(err)
	}
	ret := rowsToSessions(res)
	if len(ret) > 1 {
		log.Panicln("GET SESSION: Too many sessions for " + uuid)
	}
	return ret
}

func DeleteSessions() int64 {
	ensureConnected(db)
	res, err := db.Exec("DELETE FROM sessions")
	if err != nil {
		log.Panicln(err)
	}
	rows, _ := res.RowsAffected()
	log.Println("DELETE SESSIONS: all sessions deleted")
	return rows
}

func Connect() {
	var err error
	// read password
	if pw == "" {
		pw = fileio.ReadPw()
	}
	// Connect to the database
	db, err = sql.Open("mysql", "root:"+pw+"@tcp(localhost:3306)/users_db")
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
		Connect()
	}
	if db == nil {
		panic("Not connected to MySQL database!")
	}
}

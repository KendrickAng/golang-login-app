package database

import (
	"database/sql"
	"example.com/kendrick/api"
	"example.com/kendrick/internal/utils"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"time"
)

var db *sql.DB
var pw string = ""

// https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html
const (
	DUP_PKEY = 1062
)

var (
	updateUser    *sql.Stmt
	insertUser    *sql.Stmt
	getUser       *sql.Stmt
	insertSession *sql.Stmt
	deleteSession *sql.Stmt
	getSession    *sql.Stmt
)

// Converts sql return statement to slice of User
func rowsToUsers(rows *sql.Rows) []api.User {
	var ret []api.User
	for rows.Next() {
		var user api.User
		err := rows.Scan(&user.Nickname, &user.Username, &user.PwHash, &user.ProfilePic)
		if err != nil {
			log.Panicln(err)
		}
		ret = append(ret, user)
	}
	return ret
}

func rowsToSessions(rows *sql.Rows) []api.Session {
	var ret []api.Session
	for rows.Next() {
		var sess api.Session
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
	result, err := updateUser.Exec(nickname, picPath, key)
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
	result, err := insertUser.Exec(username, nickname, pwHash, sql.NullString{})
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
func GetUser(key string) []api.User {
	ensureConnected(db)
	res, err := getUser.Query(key)
	if err != nil {
		log.Panicln(err)
	}
	defer res.Close()
	ret := rowsToUsers(res)
	if len(ret) > 1 {
		log.Panicln("GET USER: too many rows for key " + key)
	}
	return ret
}

func InsertSession(uuid string, username string) int64 {
	ensureConnected(db)
	result, err := insertSession.Exec(uuid, username)
	if utils.IsError(err) {
		return 0
	}
	//log.Println("INSERT SESSION: uuid: " + uuid + " | username: " + username)
	rows, err := result.RowsAffected()
	if rows != 1 {
		log.Println("CREATE SESSION: one row not inserted!")
	}
	return rows
}

func DeleteSession(uuid string) int64 {
	ensureConnected(db)
	result, err := deleteSession.Exec(uuid)
	if err != nil {
		log.Panicln(err)
	}
	log.Println("DELETE sessions: uuid: " + uuid)
	rows, err := result.RowsAffected()
	return rows
}

func GetSession(uuid string) []api.Session {
	ensureConnected(db)
	rows, err := getSession.Query(uuid)
	if err != nil {
		log.Panicln(err)
	}
	defer rows.Close()
	ret := rowsToSessions(rows)
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
		pw = utils.ReadPw()
	}
	// Connect to the database
	db, err = sql.Open("mysql", "root:"+pw+"@tcp(localhost:3306)/users_db")
	if err != nil {
		log.Panicln(err.Error())
	}
	// Prepare statements
	updateUser, err = db.Prepare("UPDATE users_test SET nickname=?, profile_pic=? WHERE username=?")
	if err != nil {
		log.Panicln(err)
	}
	insertUser, err = db.Prepare("INSERT INTO users_test VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Panicln(err)
	}
	getUser, err = db.Prepare("SELECT username, nickname, pw_hash, COALESCE(profile_pic, '') FROM users_test WHERE username = ?")
	if err != nil {
		log.Panicln(err)
	}
	insertSession, err = db.Prepare("INSERT INTO sessions VALUES (?, ?)")
	if err != nil {
		log.Panicln(err)
	}
	deleteSession, err = db.Prepare("DELETE FROM sessions WHERE uuid = ?")
	if err != nil {
		log.Panicln(err)
	}
	getSession, err = db.Prepare("SELECT uuid, username FROM sessions WHERE uuid=?")
	if err != nil {
		log.Panicln(err)
	}

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(150)
	db.SetConnMaxLifetime(time.Second * 60)
	log.Println("Connected to MySQL database")
}

func Disconnect() {
	_ = db.Close()
	log.Println("Disconnected from MySQL database")
}

func ensureConnected(ptr *sql.DB) {
	if ptr == nil {
		Connect()
	}
	if db == nil {
		panic("Not connected to MySQL database!")
	}
}

package database

import (
	"database/sql"
	"errors"
	"example.com/kendrick/api"
	"example.com/kendrick/internal/tcp_server/cache"
	"example.com/kendrick/internal/utils"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"time"
)

// https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html
const (
	GET_USER       = iota
	INSERT_USER    = iota
	UPDATE_USER    = iota
	GET_SESSION    = iota
	GET_SESSIONS   = iota
	INSERT_SESSION = iota
	DUP_PKEY       = 1062
)

var (
	ERR_USER_NOT_FOUND = errors.New("No such user found!")
)

type DB interface {
	Connect()
	Disconnect()
	GetUser(username string) (*api.User, error)
	InsertUser(username string, pwHash string, nickname string) int64
	UpdateUser(key string, nickname string, picPath string) int64
}

type DBStruct struct {
	sqlDB      *sql.DB
	statements map[int]*sql.Stmt
	userCache  cache.DBCache
}

func NewDB() (DB, error) {
	ret := DBStruct{
		sqlDB:      nil,
		statements: nil,
		userCache:  nil,
	}
	ret.Connect()
	return &ret, nil
}

// Converts sql return statement to slice of User
func rowsToUsers(rows *sql.Rows) []api.User {
	var ret []api.User
	for rows.Next() {
		var user api.User
		err := rows.Scan(&user.Username, &user.Nickname, &user.PwHash, &user.ProfilePic)
		if err != nil {
			log.Panicln(err)
		}
		ret = append(ret, user)
	}
	return ret
}

func (db *DBStruct) UpdateUser(key string, nickname string, picPath string) int64 {
	db.ensureConnected()
	result, err := db.statements[UPDATE_USER].Exec(nickname, picPath, key)
	if err != nil {
		log.Panicln(err)
	}
	log.Debug("UPDATE: username: " + key + " | nickname: " + nickname + " | profile_pic: " + picPath)
	var rows int64
	rows, err = result.RowsAffected()
	if rows == 1 {
		// update the redis cache
		res, err := db.statements[GET_USER].Query(key)
		if utils.IsError(err) {
			return rows
		}
		defer res.Close()
		newRows := rowsToUsers(res)
		err = db.userCache.SetUser(key, newRows)
		if utils.IsError(err) {
			return rows
		}
		log.Debug("UPDATE redis user cache ", newRows)
	}
	return rows
}

func (db *DBStruct) InsertUser(username string, pwHash string, nickname string) int64 {
	db.ensureConnected()
	result, err := db.statements[INSERT_USER].Exec(username, nickname, pwHash, sql.NullString{})
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
func (db *DBStruct) GetUser(key string) (*api.User, error) {
	db.ensureConnected()
	userRows, err := db.userCache.GetUser(key)
	if err != nil {
		log.Error(err)
		res, err := db.statements[GET_USER].Query(key)
		if utils.IsError(err) {
			return nil, err
		}
		defer res.Close()
		ret := rowsToUsers(res)
		err = db.userCache.SetUser(key, ret)
		if utils.IsError(err) {
			return nil, err
		}
		return &ret[0], nil
	}
	if len(userRows) < 1 {
		return nil, ERR_USER_NOT_FOUND
	}
	return &userRows[0], nil
}

func (db *DBStruct) Connect() {
	var err error
	// read password
	pw := utils.ReadPw()

	// Connect to the database
	db.sqlDB, err = sql.Open("mysql", "root:"+pw+"@tcp(localhost:3306)/users_db")
	if err != nil {
		log.Panicln(err.Error())
	}
	// Prepare statements
	statements := make(map[int]*sql.Stmt, 10)
	updateUser, err := db.sqlDB.Prepare("UPDATE users_test SET nickname=?, profile_pic=? WHERE username=?")
	if err != nil {
		log.Panicln(err)
	}
	insertUser, err := db.sqlDB.Prepare("INSERT INTO users_test VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Panicln(err)
	}
	getUser, err := db.sqlDB.Prepare("SELECT username, nickname, pw_hash, COALESCE(profile_pic, '') FROM users_test WHERE username = ?")
	if err != nil {
		log.Panicln(err)
	}

	statements[GET_USER] = getUser
	statements[INSERT_USER] = insertUser
	statements[UPDATE_USER] = updateUser
	db.statements = statements

	// user cache
	userCache := cache.NewRedisCache("localhost:6379", 0, time.Minute)
	db.userCache = userCache

	db.sqlDB.SetMaxOpenConns(100)
	db.sqlDB.SetMaxIdleConns(150)
	db.sqlDB.SetConnMaxLifetime(time.Second * 60)
	log.Println("Connected to MySQL database")
}

func (db *DBStruct) Disconnect() {
	_ = db.sqlDB.Close()
	log.Println("Disconnected from MySQL database")
}

func (db *DBStruct) ensureConnected() {
	if db.sqlDB == nil {
		db.Connect()
	}
	if db == nil {
		panic("Not connected to MySQL database!")
	}
}

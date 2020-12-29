package database

import (
	"database/sql"
	"example.com/kendrick/internal/utils"
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"
)

func BenchmarkPreparedStatements(b *testing.B) {
	// database setup
	pwBytes, err := ioutil.ReadFile(filepath.Join(utils.RootDir(), "../../configs/dbPw.txt"))
	if err != nil {
		log.Panicln(err)
	}
	pw := string(pwBytes)
	db, err := sql.Open("mysql", "root:"+pw+"@tcp(localhost:3306)/users_db")
	if err != nil {
		log.Panicln(err.Error())
	}
	// Prepare statement
	getUser, err := db.Prepare("SELECT username, nickname, pw_hash, COALESCE(profile_pic, '') FROM users_test WHERE username = ?")

	// Get users without prepared statements
	b.Run("GET USER (No prepared statements", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rows, err := db.Query("SELECT username, nickname, pw_hash, COALESCE(profile_pic, '') FROM users_test WHERE username = ?", "kendrick")
			if err != nil {
				log.Panicln(err)
			}
			rows.Close()
			rowsToUsers(rows)
		}
	})
	// Get users with prepared statements
	b.Run("GET USER (Prepared statements", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rows, err := getUser.Query("kendrick")
			if err != nil {
				log.Panicln(err)
			}
			rows.Close()
			rowsToUsers(rows)
		}
	})
}

//func BenchmarkInsertSession(b *testing.B) {
//	pwBytes, err := ioutil.ReadFile(filepath.Join(utils.RootDir(), "../../configs/dbPw.txt"))
//	if err != nil {
//		log.Fatalln(err)
//	}
//	pw := string(pwBytes)
//	db, err := sql.Open("mysql", "root:"+pw+"@tcp(localhost:3306)/users_db")
//	if err != nil {
//		log.Panicln(err.Error())
//	}
//	_, err = db.Exec("DELETE FROM sessions")
//	if err != nil {
//		log.Panicln(err)
//	}
//
//	// Main loop
//	for i := 0; i < b.N; i++ {
//		result, err := db.Exec("INSERT INTO sessions VALUES (?, ?)", strconv.Itoa(i), "username")
//		if err != nil {
//			log.Fatalln(err)
//		}
//		_, _ = result.RowsAffected()
//	}
//
//	_, err = db.Exec("DELETE FROM sessions")
//	if err != nil {
//		log.Panicln(err)
//	}
//	db.Close()
//}
//
//func BenchmarkInsertSessionPreparedStmt(b *testing.B) {
//	pwBytes, err := ioutil.ReadFile(filepath.Join(utils.RootDir(), "../../configs/dbPw.txt"))
//	if err != nil {
//		log.Fatalln(err)
//	}
//	pw := string(pwBytes)
//	db, err := sql.Open("mysql", "root:"+pw+"@tcp(localhost:3306)/users_db")
//	if err != nil {
//		log.Panicln(err.Error())
//	}
//	_, err = db.Exec("DELETE FROM sessions")
//	if err != nil {
//		log.Panicln(err)
//	}
//	stmt, err := db.Prepare("INSERT INTO sessions VALUES (?, ?)")
//	if err != nil {
//		log.Fatalln(err)
//	}
//
//	// Main loop
//	for i := 0; i < b.N; i++ {
//		result, err := stmt.Exec(strconv.Itoa(i), "username")
//		if err != nil {
//			log.Fatalln(err)
//		}
//		_, _ = result.RowsAffected()
//	}
//
//	_, err = db.Exec("DELETE FROM sessions")
//	if err != nil {
//		log.Panicln(err)
//	}
//	db.Close()
//}
//
//func BenchmarkGetSession(b *testing.B) {
//	pwBytes, err := ioutil.ReadFile(filepath.Join(utils.RootDir(), "../../configs/dbPw.txt"))
//	if err != nil {
//		log.Fatalln(err)
//	}
//	pw := string(pwBytes)
//	db, err := sql.Open("mysql", "root:"+pw+"@tcp(localhost:3306)/users_db")
//	if err != nil {
//		log.Panicln(err.Error())
//	}
//	_, err = db.Exec("DELETE FROM sessions")
//	if err != nil {
//		log.Panicln(err)
//	}
//	getSession, err := db.Prepare("SELECT uuid, username FROM sessions WHERE uuid=?")
//	if err != nil {
//		log.Panicln(err)
//	}
//
//	// main loop
//	for i := 0; i < b.N; i++ {
//		rows, err := getSession.Query("12345")
//		if err != nil {
//			log.Fatalln(err)
//		}
//		rows.Close()
//	}
//
//	_, err = db.Exec("DELETE FROM sessions")
//	if err != nil {
//		log.Panicln(err)
//	}
//	db.Close()
//}

//func BenchmarkGetSessionWithRedis(b *testing.B) {
//	Connect()
//
//	// main loop
//	for i := 0; i < b.N; i++ {
//		GetSession("12345")
//	}
//
//	DeleteSessions()
//	db.Close()
//}

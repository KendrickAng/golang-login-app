package database

import (
	"database/sql"
	"example.com/kendrick/internal/utils"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"testing"
)

func BenchmarkInsertSession(b *testing.B) {
	pwBytes, err := ioutil.ReadFile(filepath.Join(utils.RootDir(), "../../configs/dbPw.txt"))
	if err != nil {
		log.Fatalln(err)
	}
	pw := string(pwBytes)
	db, err := sql.Open("mysql", "root:"+pw+"@tcp(localhost:3306)/users_db")
	if err != nil {
		log.Panicln(err.Error())
	}
	_, err = db.Exec("DELETE FROM sessions")
	if err != nil {
		log.Panicln(err)
	}

	// Main loop
	for i := 0; i < b.N; i++ {
		result, err := db.Exec("INSERT INTO sessions VALUES (?, ?)", strconv.Itoa(i), "username")
		if err != nil {
			log.Fatalln(err)
		}
		_, _ = result.RowsAffected()
	}

	_, err = db.Exec("DELETE FROM sessions")
	if err != nil {
		log.Panicln(err)
	}
	db.Close()
}

func BenchmarkInsertSessionPreparedStmt(b *testing.B) {
	pwBytes, err := ioutil.ReadFile(filepath.Join(utils.RootDir(), "../../configs/dbPw.txt"))
	if err != nil {
		log.Fatalln(err)
	}
	pw := string(pwBytes)
	db, err := sql.Open("mysql", "root:"+pw+"@tcp(localhost:3306)/users_db")
	if err != nil {
		log.Panicln(err.Error())
	}
	_, err = db.Exec("DELETE FROM sessions")
	if err != nil {
		log.Panicln(err)
	}
	stmt, err := db.Prepare("INSERT INTO sessions VALUES (?, ?)")
	if err != nil {
		log.Fatalln(err)
	}

	// Main loop
	for i := 0; i < b.N; i++ {
		result, err := stmt.Exec(strconv.Itoa(i), "username")
		if err != nil {
			log.Fatalln(err)
		}
		_, _ = result.RowsAffected()
	}

	_, err = db.Exec("DELETE FROM sessions")
	if err != nil {
		log.Panicln(err)
	}
	db.Close()
}

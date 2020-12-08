package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

func main() {
	fmt.Println("Go MySQL")
	db, err := sql.Open("mysql", "myroot:mypassword@tcp(127.0.0.1:3306)/mydb")

	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	log.Println("Connected to MySQL database")
}

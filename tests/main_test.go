package go_ora_test

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
)

var DB *sql.DB

func TestMain(m *testing.M) {
	var err error
	connStr := os.Getenv("GOORA_TESTDB")
	if connStr == "" {
		log.Fatal(fmt.Errorf("Provide  oracle server url in environment variable GOORA_TESTDB"))
	}
	DB, err = sql.Open("oracle", connStr)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	rc := m.Run()

	DB.Close()
	os.Exit(rc)
}

//go:build integration
// +build integration

package database

import (
	"context"
	"fmt"
	"os"
	"testing"
)

var (
	testSQLite     *sqlite
	testPostgreSQL *postgresql
)

func TestMain(m *testing.M) {
	code, err := runDatabaseTests(m)
	if err != nil {
		fmt.Println(err)
		os.Exit(code)
	}

	os.Exit(code)
}

func runDatabaseTests(m *testing.M) (code int, err error) {
	// Create SQLite connection
	sqliteConn, err := NewSQLiteSessionImpl(context.TODO(), nil, "./test_db.sqlite")
	if err != nil {
		return -1, fmt.Errorf("could not create or connect to database: %w", err)
	}

	var ok bool
	testSQLite, ok = sqliteConn.(*sqlite)
	if !ok {
		return -1, fmt.Errorf("could not convert DB interface to *sqlite")
	}

	defer func() {
		testPostgresqlEnd()
	}()

	// Create PostgresSQL connection
	postgresqlConn, err := NewPostgresqlSessionImpl(context.TODO(), nil, "127.0.0.1:5532", "sessions", "postgres", "postgres")
	if err != nil {
		return -1, fmt.Errorf("could not create or connect to database: %w", err)
	}

	testPostgreSQL, ok = postgresqlConn.(*postgresql)
	if !ok {
		return -1, fmt.Errorf("could not convert DB interface to *sqlite")
	}

	defer func() {
		testSQLiteEnd()
	}()

	// run all the tests
	return m.Run(), nil
}

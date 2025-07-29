package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	_ "github.com/mattn/go-sqlite3"
)

// GetTestDatabasePath returns the path to the test SQLite database
func GetTestDatabasePath() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "test_database.sqlite")
}

// GetTestDB returns a connection to the test database
func GetTestDB() (*sql.DB, error) {
	dbPath := GetTestDatabasePath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, err
	}
	return sql.Open("sqlite3", dbPath)
}

// SetupTestEnvironment configures environment variables for testing
func SetupTestEnvironment() {
	os.Setenv("DATABASE_DRIVER", "sqlite3")
	os.Setenv("DATABASE_URL", GetTestDatabasePath())
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("JWT_SECRET", "test_jwt_secret_key_for_testing_only_minimum_32_characters_long")
}

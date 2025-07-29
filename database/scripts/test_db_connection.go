package main

import (
	"database/sql"
	"fmt"
	"log"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Open the test database
	db, err := sql.Open("sqlite3", "./test_database.sqlite")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test basic connectivity
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Database connection successful")

	// Check tables exist
	tables := []string{"tenants", "users", "projects", "teams", "roles", "team_members", "permission_requests", "team_invitations"}
	
	for _, table := range tables {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			fmt.Printf("❌ Error querying table %s: %v\n", table, err)
		} else {
			fmt.Printf("✅ Table %s: %d records\n", table, count)
		}
	}

	// Test a simple query to make sure the schema works
	var teamName string
	err = db.QueryRow("SELECT name FROM teams WHERE id = 1").Scan(&teamName)
	if err != nil {
		fmt.Printf("❌ Error querying team: %v\n", err)
	} else {
		fmt.Printf("✅ Team query successful: %s\n", teamName)
	}

	// Test roles table with new columns
	var roleName string
	var isSystem bool
	var createdBy int
	err = db.QueryRow("SELECT name, is_system, created_by FROM roles WHERE id = 1").Scan(&roleName, &isSystem, &createdBy)
	if err != nil {
		fmt.Printf("❌ Error querying role: %v\n", err)
	} else {
		fmt.Printf("✅ Role query successful: %s (system: %t, created_by: %d)\n", roleName, isSystem, createdBy)
	}

	fmt.Println("🎉 Database schema verification completed!")
}
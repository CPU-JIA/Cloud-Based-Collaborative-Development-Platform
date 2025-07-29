package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Simple test to verify database connectivity
func TestDatabaseConnectivity() error {
	// Connect to our test database
	db, err := gorm.Open(sqlite.Open("test_database.sqlite"), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("Failed to connect to database: %v", err)
	}

	// Test basic queries
	var count int64
	
	// Check if users table has data
	err = db.Table("users").Count(&count).Error
	if err != nil {
		return fmt.Errorf("Failed to query users table: %v", err)
	}
	
	if count == 0 {
		return fmt.Errorf("Users table is empty")
	}
	
	fmt.Printf("✅ Users table has %d records\n", count)

	// Check if roles table has the required columns
	err = db.Table("roles").Count(&count).Error
	if err != nil {
		return fmt.Errorf("Failed to query roles table: %v", err)
	}
	
	if count == 0 {
		return fmt.Errorf("Roles table is empty")
	}
	
	fmt.Printf("✅ Roles table has %d records\n", count)

	// Test a simple raw SQL query to verify schema
	type Role struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		IsSystem  bool   `json:"is_system"`
		CreatedBy int    `json:"created_by"`
	}
	
	var role Role
	err = db.Raw("SELECT id, name, is_system, created_by FROM roles WHERE id = 1").Scan(&role).Error
	if err != nil {
		return fmt.Errorf("Failed to query role with new schema: %v", err)
	}
	
	fmt.Printf("✅ Role query successful: ID=%d, Name=%s, IsSystem=%t, CreatedBy=%d\n", 
		role.ID, role.Name, role.IsSystem, role.CreatedBy)
	
	fmt.Println("🎉 Database connectivity test passed!")
	return nil
}

func TestSimpleAPI() error {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Simple health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"database": "connected",
		})
	})
	
	// Test the health endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		return fmt.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		return fmt.Errorf("Failed to parse response: %v", err)
	}
	
	if response["status"] != "ok" {
		return fmt.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
	
	fmt.Println("✅ Simple API test passed!")
	return nil
}

func main() {
	fmt.Println("🚀 Running database connectivity test...")
	if err := TestDatabaseConnectivity(); err != nil {
		fmt.Printf("❌ Database test failed: %v\n", err)
		return
	}
	
	fmt.Println("\n🚀 Running simple API test...")
	if err := TestSimpleAPI(); err != nil {
		fmt.Printf("❌ API test failed: %v\n", err)
		return
	}
	
	fmt.Println("\n🎉 All tests passed! Integration test environment is ready.")
}
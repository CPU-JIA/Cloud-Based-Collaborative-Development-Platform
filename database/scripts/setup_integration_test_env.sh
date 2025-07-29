#!/bin/bash

# Cloud-Based Collaborative Development Platform
# Integration Test Database Setup and Verification
# This script creates a complete SQLite test database and verifies the setup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Log functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Get project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

log_info "Setting up integration test environment..."
log_info "Project root: $PROJECT_ROOT"

cd "$PROJECT_ROOT"

# Clean up any existing test database
if [ -f "test_database.sqlite" ]; then
    rm test_database.sqlite
    log_info "Removed existing test database"
fi

# Create the test database
log_info "Creating SQLite test database..."
if go run database/scripts/create_test_db.go; then
    log_success "Test database created successfully"
else
    log_error "Failed to create test database"
    exit 1
fi

# Verify the database setup
log_info "Verifying database setup..."
if go run database/scripts/simple_integration_check.go; then
    log_success "Database verification completed"
else
    log_error "Database verification failed"
    exit 1
fi

# Create test configuration
log_info "Creating test configuration..."
cat > .env.test << EOF
# SQLite Test Environment Configuration
# Generated: $(date)

# Database Configuration
DATABASE_DRIVER=sqlite3
DATABASE_URL=$PROJECT_ROOT/test_database.sqlite
DATABASE_PATH=$PROJECT_ROOT/test_database.sqlite

# Environment
ENVIRONMENT=test

# JWT Configuration (test only)
JWT_SECRET=test_jwt_secret_key_for_testing_only_minimum_32_characters_long

# Server Configuration
SERVER_PORT=8080
SERVER_HOST=localhost

# Disable external services in tests
TWO_FACTOR_ENABLED=false
FEATURE_SSO_ENABLED=false

# Test database file
TEST_DATABASE_PATH=$PROJECT_ROOT/test_database.sqlite
EOF

log_success "Test environment configuration saved to .env.test"

# Create test helper Go file
log_info "Creating test helper configuration..."
cat > test_config.go << 'EOF'
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
EOF

log_success "Test helper configuration saved to test_config.go"

# Final verification
log_info "Running final connectivity test..."
if go run database/scripts/test_db_connection.go; then
    log_success "Final verification completed"
else
    log_error "Final verification failed"
    exit 1
fi

log_success "======================================"
log_success "Integration Test Environment Ready!"
log_success "======================================"
log_info ""
log_info "Database Details:"
log_info "  Type: SQLite"
log_info "  File: $PROJECT_ROOT/test_database.sqlite"
log_info "  Records: Users(3), Projects(2), Teams(2), Roles(4)"
log_info ""
log_info "Available Test Data:"
log_info "  - Default tenant: 'default'"
log_info "  - Test users: testuser1, testuser2, testuser3"
log_info "  - Test projects: 'Test Project 1', 'Test Project 2'"
log_info "  - Test teams: 'Test Team', 'Development Team'"
log_info "  - Default roles: admin, manager, viewer"
log_info ""
log_info "How to run tests:"
log_info "  1. Run all tests: go test ./tests/..."
log_info "  2. Run integration tests: go test ./tests/api_integration_test.go"
log_info "  3. Use test database in your tests with: gorm.Open(sqlite.Open(\"test_database.sqlite\"))"
log_info ""
log_success "Setup completed successfully!"
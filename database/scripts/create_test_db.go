package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// createTestDatabase creates an SQLite test database with all necessary tables
func createTestDatabase() error {
	// Get project root directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %v", err)
	}

	dbPath := filepath.Join(wd, "test_database.sqlite")
	
	// Remove existing database if it exists
	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Remove(dbPath); err != nil {
			return fmt.Errorf("failed to remove existing database: %v", err)
		}
	}

	// Create new database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %v", err)
	}

	// Create tables
	tables := []string{
		// Tenants table
		`CREATE TABLE tenants (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT UNIQUE NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		
		// Users table
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			tenant_id INTEGER NOT NULL,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		);`,
		
		// Roles table
		`CREATE TABLE roles (
			id INTEGER PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			project_id INTEGER,
			name TEXT NOT NULL,
			description TEXT,
			permissions TEXT,
			is_system BOOLEAN DEFAULT 0,
			created_by INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		
		// Projects table
		`CREATE TABLE projects (
			id INTEGER PRIMARY KEY,
			tenant_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			owner_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (tenant_id) REFERENCES tenants(id),
			FOREIGN KEY (owner_id) REFERENCES users(id)
		);`,
		
		// Teams table
		`CREATE TABLE teams (
			id INTEGER PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			project_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			avatar TEXT,
			settings TEXT,
			is_active BOOLEAN DEFAULT 1,
			created_by INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (project_id) REFERENCES projects(id),
			FOREIGN KEY (created_by) REFERENCES users(id)
		);`,
		
		// Team members table
		`CREATE TABLE team_members (
			id INTEGER PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			team_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			role_id INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			invited_by INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (team_id) REFERENCES teams(id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (role_id) REFERENCES roles(id),
			FOREIGN KEY (invited_by) REFERENCES users(id),
			UNIQUE(team_id, user_id)
		);`,
		
		// Tasks table
		`CREATE TABLE tasks (
			id INTEGER PRIMARY KEY,
			tenant_id INTEGER NOT NULL,
			project_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			priority TEXT NOT NULL DEFAULT 'medium',
			assignee_id INTEGER,
			created_by INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (tenant_id) REFERENCES tenants(id),
			FOREIGN KEY (project_id) REFERENCES projects(id),
			FOREIGN KEY (assignee_id) REFERENCES users(id),
			FOREIGN KEY (created_by) REFERENCES users(id)
		);`,
		
		// Repositories table
		`CREATE TABLE repositories (
			id INTEGER PRIMARY KEY,
			tenant_id INTEGER NOT NULL,
			project_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			url TEXT,
			branch TEXT DEFAULT 'main',
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (tenant_id) REFERENCES tenants(id),
			FOREIGN KEY (project_id) REFERENCES projects(id)
		);`,
		
		// Permission requests table
		`CREATE TABLE permission_requests (
			id INTEGER PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			project_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			request_type TEXT NOT NULL,
			target_id INTEGER,
			permission TEXT NOT NULL,
			reason TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			reviewed_by INTEGER,
			reviewed_at DATETIME,
			review_reason TEXT,
			expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (project_id) REFERENCES projects(id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (reviewed_by) REFERENCES users(id)
		);`,
		
		// Team invitations table
		`CREATE TABLE team_invitations (
			id INTEGER PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			team_id INTEGER NOT NULL,
			project_id INTEGER NOT NULL,
			email TEXT NOT NULL,
			role_id INTEGER NOT NULL,
			token TEXT UNIQUE NOT NULL,
			message TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			invited_by INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			accepted_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (team_id) REFERENCES teams(id),
			FOREIGN KEY (project_id) REFERENCES projects(id),
			FOREIGN KEY (role_id) REFERENCES roles(id),
			FOREIGN KEY (invited_by) REFERENCES users(id)
		);`,
		
		// Schema migrations table
		`CREATE TABLE schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	// Execute table creation
	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}

	// Insert test data
	testData := []string{
		// Insert default tenant
		`INSERT INTO tenants (id, name, slug, status) VALUES 
			(1, 'Default Tenant', 'default', 'active');`,
		
		// Insert test users
		`INSERT INTO users (id, tenant_id, username, email, password_hash, status) VALUES 
			(1, 1, 'testuser1', 'test1@example.com', '$2a$10$dummy.hash.for.testing', 'active'),
			(2, 1, 'testuser2', 'test2@example.com', '$2a$10$dummy.hash.for.testing', 'active'),
			(3, 1, 'testuser3', 'test3@example.com', '$2a$10$dummy.hash.for.testing', 'active');`,
		
		// Insert test projects
		`INSERT INTO projects (id, tenant_id, name, slug, description, status, owner_id) VALUES 
			(1, 1, 'Test Project 1', 'test-project-1', 'Test project for integration tests', 'active', 1),
			(2, 1, 'Test Project 2', 'test-project-2', 'Another test project', 'active', 2);`,
		
		// Insert default roles
		`INSERT INTO roles (id, tenant_id, project_id, name, description, permissions, is_system, created_by) VALUES 
			(1, 'default', NULL, 'admin', 'System Administrator', '["*"]', 1, 1),
			(2, 'default', NULL, 'manager', 'Project Manager', '["project.read", "project.write", "user.read"]', 1, 1),
			(3, 'default', NULL, 'viewer', 'Viewer', '["project.read"]', 1, 1),
			(4, 'default', 1, 'viewer', 'Project Viewer', '["project.read"]', 1, 1);`,
		
		// Insert test teams
		`INSERT INTO teams (id, tenant_id, project_id, name, description, avatar, settings, is_active, created_by) VALUES 
			(1, 'default', 1, 'Test Team', 'This is a test team', '', '', 1, 1),
			(2, 'default', 1, 'Development Team', 'Main development team', '', '', 1, 1);`,
		
		// Insert test tasks
		`INSERT INTO tasks (id, tenant_id, project_id, title, description, status, priority, assignee_id, created_by) VALUES 
			(1, 1, 1, 'Test Task 1', 'First test task', 'pending', 'medium', 1, 1),
			(2, 1, 1, 'Test Task 2', 'Second test task', 'in_progress', 'high', 2, 1);`,
		
		// Insert test repositories
		`INSERT INTO repositories (id, tenant_id, project_id, name, description, url, branch, status) VALUES 
			(1, 1, 1, 'test-repo-1', 'Test repository 1', 'https://github.com/test/repo1', 'main', 'active'),
			(2, 1, 2, 'test-repo-2', 'Test repository 2', 'https://github.com/test/repo2', 'main', 'active');`,
		
		// Record migration
		`INSERT INTO schema_migrations (version) VALUES ('sqlite_test_schema');`,
	}

	// Execute test data insertion
	for _, data := range testData {
		if _, err := db.Exec(data); err != nil {
			return fmt.Errorf("failed to insert test data: %v", err)
		}
	}

	fmt.Printf("✅ SQLite test database created successfully: %s\n", dbPath)
	
	// Verify tables
	tables_to_check := []string{"tenants", "users", "projects", "tasks", "repositories", "roles", "teams"}
	for _, table := range tables_to_check {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to verify table %s: %v", table, err)
		}
		fmt.Printf("✅ Table %s exists (records: %d)\n", table, count)
	}

	return nil
}

func main() {
	fmt.Println("🚀 Creating SQLite test database...")
	
	if err := createTestDatabase(); err != nil {
		log.Fatalf("❌ Failed to create test database: %v", err)
	}
	
	fmt.Println("🎉 Test database initialization completed!")
	fmt.Println("📊 You can now run integration tests with: go test ./tests/...")
}
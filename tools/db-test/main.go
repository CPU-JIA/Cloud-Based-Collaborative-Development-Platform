package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/cloud-platform/collaborative-dev/shared/config"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	_ "github.com/lib/pq" // PostgreSQL driver
)

func main() {
	fmt.Println("=== Cloud-Based Collaborative Development Platform ===")
	fmt.Println("数据库连接测试工具")
	fmt.Println()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	fmt.Printf("数据库配置:\n")
	fmt.Printf("  Host: %s\n", cfg.Database.Host)
	fmt.Printf("  Port: %d\n", cfg.Database.Port)
	fmt.Printf("  Name: %s\n", cfg.Database.Name)
	fmt.Printf("  User: %s\n", cfg.Database.User)
	fmt.Printf("  SSL Mode: %s\n", cfg.Database.SSLMode)
	fmt.Println()

	// 测试基本连接
	fmt.Println("1. 测试基本数据库连接...")
	dsn := cfg.GetDatabaseDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("连接字符串错误: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Printf("❌ 数据库连接失败: %v", err)
		fmt.Println("\n请检查:")
		fmt.Println("  - PostgreSQL 服务是否运行")
		fmt.Println("  - 数据库用户和密码是否正确")
		fmt.Println("  - 数据库是否存在")
		fmt.Println("  - 网络连接是否正常")
		os.Exit(1)
	}
	fmt.Println("✅ 基本数据库连接成功")

	// 测试GORM连接
	fmt.Println("\n2. 测试GORM数据库连接...")
	dbConfig := cfg.Database.ToDBConfig().(database.Config)
	gormDB, err := database.NewPostgresDB(dbConfig)
	if err != nil {
		log.Fatalf("❌ GORM连接失败: %v", err)
	}

	sqlDB, err := gormDB.DB.DB()
	if err != nil {
		log.Fatalf("❌ 获取底层数据库连接失败: %v", err)
	}
	defer sqlDB.Close()

	if err = sqlDB.Ping(); err != nil {
		log.Fatalf("❌ GORM数据库Ping失败: %v", err)
	}
	fmt.Println("✅ GORM数据库连接成功")

	// 检查数据库版本
	fmt.Println("\n3. 检查数据库版本信息...")
	var version string
	err = db.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		log.Printf("❌ 获取数据库版本失败: %v", err)
	} else {
		fmt.Printf("✅ PostgreSQL版本: %s\n", version[:100]+"...")
	}

	// 检查UUID扩展
	fmt.Println("\n4. 检查必要的数据库扩展...")
	var extExists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'uuid-ossp')").Scan(&extExists)
	if err != nil {
		log.Printf("❌ 检查UUID扩展失败: %v", err)
	} else if extExists {
		fmt.Println("✅ uuid-ossp 扩展已安装")
	} else {
		fmt.Println("⚠️  uuid-ossp 扩展未安装")
		fmt.Println("   请执行: CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
	}

	// 检查核心表是否存在
	fmt.Println("\n5. 检查核心数据表...")
	coreTables := []string{
		"subscription_plans",
		"tenants", 
		"users",
		"projects",
		"repositories",
		"pipelines",
	}

	for _, table := range coreTables {
		var tableExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = $1)", table).Scan(&tableExists)
		if err != nil {
			fmt.Printf("❌ 检查表 %s 失败: %v\n", table, err)
		} else if tableExists {
			fmt.Printf("✅ 表 %s 存在\n", table)
		} else {
			fmt.Printf("⚠️  表 %s 不存在 - 需要运行数据库迁移\n", table)
		}
	}

	// 检查数据库函数
	fmt.Println("\n6. 检查自定义函数...")
	var funcExists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_proc WHERE proname = 'uuid_generate_v7')").Scan(&funcExists)
	if err != nil {
		fmt.Printf("❌ 检查uuid_generate_v7函数失败: %v\n", err)
	} else if funcExists {
		fmt.Println("✅ uuid_generate_v7 函数存在")
	} else {
		fmt.Println("⚠️  uuid_generate_v7 函数不存在 - 需要运行数据库迁移")
	}

	// 连接池状态
	fmt.Println("\n7. 数据库连接池状态...")
	stats := sqlDB.Stats()
	fmt.Printf("✅ 连接池状态:\n")
	fmt.Printf("   打开连接: %d\n", stats.OpenConnections)
	fmt.Printf("   使用中连接: %d\n", stats.InUse)
	fmt.Printf("   空闲连接: %d\n", stats.Idle)

	fmt.Println("\n=== 数据库连接测试完成 ===")
	fmt.Println()
	
	if extExists && funcExists {
		fmt.Println("🎉 数据库已就绪，可以启动服务")
	} else {
		fmt.Println("🔧 需要先运行数据库迁移脚本")
		fmt.Println("   可运行: psql -d <database_name> -f database/migrations/001_initial_schema.sql")
	}
}
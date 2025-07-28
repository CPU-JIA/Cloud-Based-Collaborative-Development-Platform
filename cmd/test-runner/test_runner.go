package main

// +build testrunner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TestResult 存储单个测试的结果
type TestResult struct {
	File     string
	Success  bool
	Duration time.Duration
	Output   string
}

func main() {
	fmt.Println("🚀 Cloud-Based Collaborative Development Platform - 测试运行器")
	fmt.Println("============================================")

	// 获取所有测试文件
	testFiles := []string{
		"test/unit/project_validation_test.go",
		"test/unit/git_gateway_test.go",
		"test/unit/tenant_service_test.go",
		"test/unit/iam_service_test.go",
		"test/unit/notification_service_test.go",
		"test/unit/cicd_service_test.go",
		"test/unit/file_service_test.go",
		"test/unit/team_service_test.go",
		"test/unit/knowledge_base_service_test.go",
	}

	results := make([]TestResult, 0)
	totalTests := 0
	passedTests := 0

	// 运行每个测试文件
	for _, testFile := range testFiles {
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			fmt.Printf("❌ 文件不存在: %s\n", testFile)
			continue
		}

		fmt.Printf("\n📋 运行测试: %s\n", testFile)
		start := time.Now()

		// 创建临时目录
		tempDir, err := os.MkdirTemp("", "test_")
		if err != nil {
			fmt.Printf("❌ 创建临时目录失败: %v\n", err)
			continue
		}
		defer os.RemoveAll(tempDir)

		// 复制测试文件到临时目录
		tempFile := filepath.Join(tempDir, filepath.Base(testFile))
		content, err := os.ReadFile(testFile)
		if err != nil {
			fmt.Printf("❌ 读取文件失败: %v\n", err)
			continue
		}

		// 写入临时文件
		err = os.WriteFile(tempFile, content, 0644)
		if err != nil {
			fmt.Printf("❌ 写入临时文件失败: %v\n", err)
			continue
		}

		// 运行测试
		cmd := exec.Command("go", "test", "-v", tempFile)
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		result := TestResult{
			File:     testFile,
			Success:  err == nil,
			Duration: duration,
			Output:   string(output),
		}
		results = append(results, result)

		if result.Success {
			fmt.Printf("✅ 通过 (耗时: %.2fs)\n", duration.Seconds())
			passedTests++
		} else {
			fmt.Printf("❌ 失败 (耗时: %.2fs)\n", duration.Seconds())
			// 显示错误输出的最后几行
			lines := strings.Split(result.Output, "\n")
			if len(lines) > 5 {
				fmt.Println("错误输出:")
				for _, line := range lines[len(lines)-5:] {
					if strings.TrimSpace(line) != "" {
						fmt.Printf("    %s\n", line)
					}
				}
			}
		}
		totalTests++
	}

	// 运行集成测试
	fmt.Printf("\n📋 运行集成测试\n")
	start := time.Now()
	cmd := exec.Command("go", "test", "-v", "./test/integration/...")
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	// 打印输出用于调试
	fmt.Printf("集成测试输出:\n%s\n", string(output))

	if err == nil {
		fmt.Printf("✅ 集成测试通过 (耗时: %.2fs)\n", duration.Seconds())
		passedTests++
	} else {
		fmt.Printf("❌ 集成测试失败 (耗时: %.2fs)\n", duration.Seconds())
	}
	totalTests++

	// 输出总结
	fmt.Println("\n============================================")
	fmt.Println("📊 测试摘要")
	fmt.Println("============================================")
	fmt.Printf("总测试数: %d\n", totalTests)
	fmt.Printf("通过: %d\n", passedTests)
	fmt.Printf("失败: %d\n", totalTests-passedTests)
	if totalTests > 0 {
		fmt.Printf("成功率: %.1f%%\n", float64(passedTests)*100/float64(totalTests))
	}

	// 如果有失败的测试，返回非零退出码
	if passedTests < totalTests {
		os.Exit(1)
	}
}

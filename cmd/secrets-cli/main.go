package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cloud-platform/collaborative-dev/shared/config/secrets"
	vault "github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"
)

var (
	environment string
	provider    string
	configPath  string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "secrets-cli",
		Short: "密钥管理命令行工具",
		Long:  `用于管理 Cloud Platform 密钥的命令行工具，支持设置、获取、轮换密钥等操作`,
	}

	// 全局标志
	rootCmd.PersistentFlags().StringVarP(&environment, "env", "e", "development", "环境 (development, test, production)")
	rootCmd.PersistentFlags().StringVarP(&provider, "provider", "p", "file", "密钥提供者 (file, env, vault)")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "./configs", "配置路径")

	// 添加子命令
	rootCmd.AddCommand(
		newSetCommand(),
		newGetCommand(),
		newListCommand(),
		newRotateCommand(),
		newValidateCommand(),
		newInitCommand(),
		newExportCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// newSetCommand 创建设置密钥命令
func newSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "设置密钥值",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := createSecretManager()
			if err != nil {
				return err
			}

			key := args[0]
			value := args[1]

			// 如果值为 "-"，从标准输入读取
			if value == "-" {
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					value = scanner.Text()
				}
			}

			if err := manager.SetSecret(key, value); err != nil {
				return fmt.Errorf("failed to set secret: %w", err)
			}

			fmt.Printf("✅ Secret '%s' has been set successfully\n", key)
			return nil
		},
	}
}

// newGetCommand 创建获取密钥命令
func newGetCommand() *cobra.Command {
	var showValue bool

	cmd := &cobra.Command{
		Use:   "get [key]",
		Short: "获取密钥值",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := createSecretManager()
			if err != nil {
				return err
			}

			key := args[0]
			value, err := manager.GetSecret(key)
			if err != nil {
				return fmt.Errorf("failed to get secret: %w", err)
			}

			if showValue {
				fmt.Println(value)
			} else {
				// 默认只显示部分值
				masked := maskSecret(value)
				fmt.Printf("Secret '%s': %s\n", key, masked)
				fmt.Println("Use --show flag to display the full value")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&showValue, "show", false, "显示完整的密钥值")
	return cmd
}

// newListCommand 创建列出密钥命令
func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出所有密钥",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := createSecretManager()
			if err != nil {
				return err
			}

			secrets := manager.ListSecrets()
			if len(secrets) == 0 {
				fmt.Println("No secrets found")
				return nil
			}

			fmt.Println("Configured secrets:")
			for _, key := range secrets {
				fmt.Printf("  - %s\n", key)
			}

			return nil
		},
	}
}

// newRotateCommand 创建轮换密钥命令
func newRotateCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "rotate [key]",
		Short: "轮换密钥",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := createSecretManager()
			if err != nil {
				return err
			}

			key := args[0]

			// 确认操作
			if !force {
				fmt.Printf("⚠️  Are you sure you want to rotate secret '%s'? (y/N): ", key)
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" {
					fmt.Println("Operation cancelled")
					return nil
				}
			}

			newValue, err := manager.RotateSecret(key)
			if err != nil {
				return fmt.Errorf("failed to rotate secret: %w", err)
			}

			fmt.Printf("✅ Secret '%s' has been rotated successfully\n", key)
			fmt.Printf("New value: %s\n", maskSecret(newValue))

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "强制轮换，不需要确认")
	return cmd
}

// newValidateCommand 创建验证密钥命令
func newValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "验证所有必需的密钥",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := createSecretManager()
			if err != nil {
				return err
			}

			if err := manager.ValidateSecrets(); err != nil {
				fmt.Printf("❌ Validation failed: %v\n", err)
				return err
			}

			fmt.Println("✅ All required secrets are properly configured")
			return nil
		},
	}
}

// newInitCommand 创建初始化命令
func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "初始化密钥配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Initializing secrets for environment: %s\n", environment)

			// 创建必需的目录
			secretsDir := fmt.Sprintf("%s/secrets", configPath)
			if err := os.MkdirAll(secretsDir, 0700); err != nil {
				return fmt.Errorf("failed to create secrets directory: %w", err)
			}

			// 创建密钥管理器
			manager, err := createSecretManager()
			if err != nil {
				return err
			}

			// 初始化必需的密钥
			requiredSecrets := map[string]string{
				"database_password": "Database password",
				"jwt_secret":        "JWT secret key (min 32 chars)",
				"redis_password":    "Redis password (optional, press Enter to skip)",
				"encryption_key":    "Encryption key for secrets",
			}

			for key, description := range requiredSecrets {
				// 检查是否已存在
				if _, err := manager.GetSecret(key); err == nil {
					fmt.Printf("✓ %s already configured\n", key)
					continue
				}

				// 提示输入
				fmt.Printf("Enter %s: ", description)
				value := readPassword()

				if value != "" {
					if err := manager.SetSecret(key, value); err != nil {
						return fmt.Errorf("failed to set %s: %w", key, err)
					}
					fmt.Printf("✓ %s configured\n", key)
				}
			}

			// 验证配置
			if err := manager.ValidateSecrets(); err != nil {
				fmt.Printf("⚠️  Warning: %v\n", err)
			}

			fmt.Println("\n✅ Secrets initialization completed")
			fmt.Printf("Configuration saved to: %s\n", secretsDir)

			return nil
		},
	}
}

// newExportCommand 创建导出命令
func newExportCommand() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "导出密钥配置（用于环境变量）",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := createSecretManager()
			if err != nil {
				return err
			}

			secrets := manager.ListSecrets()

			switch format {
			case "env":
				fmt.Println("# Cloud Platform Secret Environment Variables")
				fmt.Printf("# Generated for environment: %s\n\n", environment)

				for _, key := range secrets {
					envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
					fmt.Printf("# export CLOUDPLATFORM_%s=<value>\n", envKey)
				}

			case "docker":
				fmt.Println("# Docker Compose environment variables")
				fmt.Println("environment:")

				for _, key := range secrets {
					envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
					fmt.Printf("  - CLOUDPLATFORM_%s=${CLOUDPLATFORM_%s}\n", envKey, envKey)
				}

			case "k8s":
				fmt.Println("apiVersion: v1")
				fmt.Println("kind: Secret")
				fmt.Println("metadata:")
				fmt.Printf("  name: cloudplatform-secrets-%s\n", environment)
				fmt.Println("type: Opaque")
				fmt.Println("stringData:")

				for _, key := range secrets {
					fmt.Printf("  %s: \"${%s}\"\n", key, strings.ToUpper(strings.ReplaceAll(key, ".", "_")))
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "env", "导出格式 (env, docker, k8s)")
	return cmd
}

// createSecretManager 创建密钥管理器
func createSecretManager() (secrets.SecretManager, error) {
	var secretProvider secrets.SecretProvider
	var err error

	switch provider {
	case "file":
		secretsPath := fmt.Sprintf("%s/secrets/%s.secrets.yaml", configPath, environment)
		secretProvider, err = secrets.NewFileProvider(secretsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file provider: %w", err)
		}

	case "env":
		secretProvider = secrets.NewEnvironmentProvider("CLOUDPLATFORM")

	case "vault":
		// 从环境变量获取Vault配置
		vaultConfig := secrets.VaultConfig{
			Address:    os.Getenv("VAULT_ADDR"),
			Token:      os.Getenv("VAULT_TOKEN"),
			Namespace:  os.Getenv("VAULT_NAMESPACE"),
			MountPath:  os.Getenv("VAULT_MOUNT_PATH"),
			SecretPath: fmt.Sprintf("%s/%s", environment, "secrets"),
		}

		// 如果没有设置地址，使用默认值
		if vaultConfig.Address == "" {
			vaultConfig.Address = "http://127.0.0.1:8200"
		}

		// 如果没有设置挂载路径，使用默认值
		if vaultConfig.MountPath == "" {
			vaultConfig.MountPath = "secret"
		}

		// TLS配置（如果需要）
		if os.Getenv("VAULT_CACERT") != "" || os.Getenv("VAULT_CLIENT_CERT") != "" {
			vaultConfig.TLSConfig = &vault.TLSConfig{
				CACert:     os.Getenv("VAULT_CACERT"),
				ClientCert: os.Getenv("VAULT_CLIENT_CERT"),
				ClientKey:  os.Getenv("VAULT_CLIENT_KEY"),
				Insecure:   os.Getenv("VAULT_SKIP_VERIFY") == "true",
			}
		}

		secretProvider, err = secrets.NewVaultProvider(vaultConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create vault provider: %w", err)
		}

	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	// 获取加密密钥
	encryptionKey := os.Getenv("SECRETS_ENCRYPTION_KEY")
	if encryptionKey == "" && environment != "development" {
		return nil, fmt.Errorf("SECRETS_ENCRYPTION_KEY is required for %s environment", environment)
	}

	if encryptionKey == "" {
		encryptionKey = "development_encryption_key_only"
		fmt.Println("⚠️  Warning: Using default encryption key for development")
	}

	return secrets.NewManager(secretProvider, encryptionKey)
}

// maskSecret 遮罩密钥值
func maskSecret(value string) string {
	if len(value) <= 8 {
		return "********"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

// readPassword 读取密码输入
func readPassword() string {
	// 在实际使用中，应该使用 golang.org/x/term 来隐藏输入
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

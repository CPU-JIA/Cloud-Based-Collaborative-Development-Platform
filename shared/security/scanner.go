package security

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cloud-platform/collaborative-dev/shared/logger"
)

// ScanType 扫描类型
type ScanType string

const (
	ScanTypeSAST       ScanType = "sast"       // 静态应用安全测试
	ScanTypeSCA        ScanType = "sca"        // 软件组成分析
	ScanTypeDependency ScanType = "dependency" // 依赖项安全检查
	ScanTypeContainer  ScanType = "container"  // 容器安全扫描
	ScanTypeLicense    ScanType = "license"    // 许可证合规性检查
)

// Severity 漏洞严重性
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Vulnerability 漏洞信息
type Vulnerability struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Severity    Severity               `json:"severity"`
	CVE         string                 `json:"cve,omitempty"`
	CWE         string                 `json:"cwe,omitempty"`
	CVSS        float64                `json:"cvss,omitempty"`
	Location    VulnLocation           `json:"location"`
	Fix         *VulnFix               `json:"fix,omitempty"`
	References  []string               `json:"references,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// VulnLocation 漏洞位置
type VulnLocation struct {
	File      string `json:"file"`
	Line      int    `json:"line,omitempty"`
	Column    int    `json:"column,omitempty"`
	Function  string `json:"function,omitempty"`
	Component string `json:"component,omitempty"`
	Version   string `json:"version,omitempty"`
}

// VulnFix 漏洞修复信息
type VulnFix struct {
	Suggestion     string `json:"suggestion"`
	FixedVersion   string `json:"fixed_version,omitempty"`
	AutoFixable    bool   `json:"auto_fixable"`
	BreakingChange bool   `json:"breaking_change"`
}

// ScanResult 扫描结果
type ScanResult struct {
	ID              uuid.UUID              `json:"id"`
	ScanType        ScanType               `json:"scan_type"`
	ProjectPath     string                 `json:"project_path"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         time.Time              `json:"end_time"`
	Duration        time.Duration          `json:"duration"`
	Status          string                 `json:"status"` // running, completed, failed
	Vulnerabilities []Vulnerability        `json:"vulnerabilities"`
	Summary         ScanSummary            `json:"summary"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Error           string                 `json:"error,omitempty"`
}

// ScanSummary 扫描摘要
type ScanSummary struct {
	TotalVulns   int              `json:"total_vulnerabilities"`
	BySeverity   map[Severity]int `json:"by_severity"`
	FilesScanned int              `json:"files_scanned"`
	LinesScanned int              `json:"lines_scanned"`
	Coverage     float64          `json:"coverage_percentage"`
}

// SecurityScanner 安全扫描器接口
type SecurityScanner interface {
	// 扫描操作
	ScanProject(ctx context.Context, projectPath string, scanTypes []ScanType) (*ScanResult, error)
	ScanRepository(ctx context.Context, repoURL string, scanTypes []ScanType) (*ScanResult, error)
	ScanContainer(ctx context.Context, imageName string) (*ScanResult, error)

	// 扫描结果管理
	GetScanResult(ctx context.Context, scanID uuid.UUID) (*ScanResult, error)
	ListScanResults(ctx context.Context, projectID uuid.UUID, limit int) ([]*ScanResult, error)
	DeleteScanResult(ctx context.Context, scanID uuid.UUID) error

	// 配置管理
	UpdateScanConfig(ctx context.Context, config *ScanConfig) error
	GetScanConfig(ctx context.Context) (*ScanConfig, error)
}

// ScanConfig 扫描配置
type ScanConfig struct {
	// 扫描规则配置
	EnabledScanTypes  []ScanType `json:"enabled_scan_types"`
	SeverityThreshold Severity   `json:"severity_threshold"`
	ExcludePatterns   []string   `json:"exclude_patterns"`
	IncludePatterns   []string   `json:"include_patterns"`

	// 工具配置
	ToolConfigs map[ScanType]ToolConfig `json:"tool_configs"`

	// 报告配置
	ReportFormat   []string `json:"report_formats"` // json, sarif, junit
	MaxResults     int      `json:"max_results"`
	IncludeFixInfo bool     `json:"include_fix_info"`

	// 调度配置
	ScheduleEnabled bool          `json:"schedule_enabled"`
	ScheduleCron    string        `json:"schedule_cron"`
	Timeout         time.Duration `json:"timeout"`
}

// ToolConfig 工具配置
type ToolConfig struct {
	Enabled bool                   `json:"enabled"`
	Version string                 `json:"version"`
	Args    []string               `json:"args"`
	Config  map[string]interface{} `json:"config"`
}

// securityScanner 安全扫描器实现
type securityScanner struct {
	logger logger.Logger
	config *ScanConfig
}

// NewSecurityScanner 创建安全扫描器实例
func NewSecurityScanner(logger logger.Logger, config *ScanConfig) SecurityScanner {
	if config == nil {
		config = getDefaultScanConfig()
	}

	return &securityScanner{
		logger: logger,
		config: config,
	}
}

// ScanProject 扫描项目
func (s *securityScanner) ScanProject(ctx context.Context, projectPath string, scanTypes []ScanType) (*ScanResult, error) {
	scanID := uuid.New()
	startTime := time.Now()

	result := &ScanResult{
		ID:          scanID,
		ProjectPath: projectPath,
		StartTime:   startTime,
		Status:      "running",
		Summary:     ScanSummary{BySeverity: make(map[Severity]int)},
	}

	s.logger.Info("开始安全扫描", "scan_id", scanID, "project_path", projectPath, "scan_types", scanTypes)

	var allVulns []Vulnerability

	for _, scanType := range scanTypes {
		if !s.isScanTypeEnabled(scanType) {
			s.logger.Debug("跳过禁用的扫描类型", "scan_type", scanType)
			continue
		}

		vulns, err := s.performScan(ctx, scanType, projectPath)
		if err != nil {
			s.logger.Error("扫描失败", "scan_type", scanType, "error", err)
			result.Error = fmt.Sprintf("扫描类型 %s 失败: %v", scanType, err)
			// 继续其他扫描
			continue
		}

		allVulns = append(allVulns, vulns...)
	}

	// 过滤和排序漏洞
	filteredVulns := s.filterVulnerabilities(allVulns)
	result.Vulnerabilities = filteredVulns

	// 生成摘要
	result.Summary = s.generateSummary(filteredVulns)

	// 完成扫描
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Status = "completed"

	s.logger.Info("安全扫描完成",
		"scan_id", scanID,
		"duration", result.Duration,
		"total_vulns", result.Summary.TotalVulns,
		"critical", result.Summary.BySeverity[SeverityCritical],
		"high", result.Summary.BySeverity[SeverityHigh],
	)

	return result, nil
}

// performScan 执行具体扫描
func (s *securityScanner) performScan(ctx context.Context, scanType ScanType, projectPath string) ([]Vulnerability, error) {
	switch scanType {
	case ScanTypeSAST:
		return s.performSASTScan(ctx, projectPath)
	case ScanTypeSCA:
		return s.performSCAScan(ctx, projectPath)
	case ScanTypeDependency:
		return s.performDependencyScan(ctx, projectPath)
	case ScanTypeContainer:
		return s.performContainerScan(ctx, projectPath)
	case ScanTypeLicense:
		return s.performLicenseScan(ctx, projectPath)
	default:
		return nil, fmt.Errorf("不支持的扫描类型: %s", scanType)
	}
}

// performSASTScan 执行SAST扫描（使用CodeQL或Semgrep）
func (s *securityScanner) performSASTScan(ctx context.Context, projectPath string) ([]Vulnerability, error) {
	// 这里使用Semgrep作为示例
	cmd := exec.CommandContext(ctx, "semgrep",
		"--config=auto",
		"--json",
		"--severity=ERROR",
		"--severity=WARNING",
		projectPath,
	)

	output, err := cmd.Output()
	if err != nil {
		// Semgrep在找到漏洞时会返回非零退出码，需要特殊处理
		if exitError, ok := err.(*exec.ExitError); ok && len(output) > 0 {
			// 有输出说明扫描成功，只是找到了漏洞
			s.logger.Debug("SAST扫描发现漏洞", "exit_code", exitError.ExitCode())
		} else {
			return nil, fmt.Errorf("SAST扫描失败: %w", err)
		}
	}

	return s.parseSemgrepOutput(output)
}

// performSCAScan 执行SCA扫描（软件组成分析）
func (s *securityScanner) performSCAScan(ctx context.Context, projectPath string) ([]Vulnerability, error) {
	// 使用Nancy（Go依赖扫描）或其他SCA工具
	cmd := exec.CommandContext(ctx, "nancy", "sleuth", "-f", "json", projectPath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("SCA扫描失败: %w", err)
	}

	return s.parseNancyOutput(output)
}

// performDependencyScan 执行依赖项安全检查
func (s *securityScanner) performDependencyScan(ctx context.Context, projectPath string) ([]Vulnerability, error) {
	// 使用go list -m -json all 获取依赖信息，然后查询漏洞数据库
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", "all")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("获取依赖列表失败: %w", err)
	}

	return s.analyzeDependencies(output)
}

// performContainerScan 执行容器安全扫描
func (s *securityScanner) performContainerScan(ctx context.Context, projectPath string) ([]Vulnerability, error) {
	// 扫描Dockerfile和容器镜像
	dockerfilePath := fmt.Sprintf("%s/Dockerfile", projectPath)

	// 使用Hadolint扫描Dockerfile
	cmd := exec.CommandContext(ctx, "hadolint", "--format", "json", dockerfilePath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("容器扫描失败: %w", err)
	}

	return s.parseHadolintOutput(output)
}

// performLicenseScan 执行许可证合规性检查
func (s *securityScanner) performLicenseScan(ctx context.Context, projectPath string) ([]Vulnerability, error) {
	// 扫描依赖的许可证合规性
	cmd := exec.CommandContext(ctx, "go-licenses", "check", projectPath)

	output, err := cmd.Output()
	if err != nil {
		// go-licenses在发现问题时返回非零退出码
		if len(output) > 0 {
			return s.parseLicenseOutput(output), nil
		}
		return nil, fmt.Errorf("许可证扫描失败: %w", err)
	}

	return []Vulnerability{}, nil // 没有发现许可证问题
}

// 解析器方法

func (s *securityScanner) parseSemgrepOutput(output []byte) ([]Vulnerability, error) {
	var result struct {
		Results []struct {
			CheckID string `json:"check_id"`
			Path    string `json:"path"`
			Start   struct {
				Line int `json:"line"`
				Col  int `json:"col"`
			} `json:"start"`
			Message  string `json:"message"`
			Severity string `json:"severity"`
			Metadata struct {
				CWE      []string `json:"cwe,omitempty"`
				OWASP    []string `json:"owasp,omitempty"`
				Category string   `json:"category,omitempty"`
			} `json:"metadata"`
		} `json:"results"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("解析Semgrep输出失败: %w", err)
	}

	var vulnerabilities []Vulnerability
	for _, r := range result.Results {
		vuln := Vulnerability{
			ID:          r.CheckID,
			Title:       r.CheckID,
			Description: r.Message,
			Severity:    s.mapSemgrepSeverity(r.Severity),
			Location: VulnLocation{
				File:   r.Path,
				Line:   r.Start.Line,
				Column: r.Start.Col,
			},
		}

		if len(r.Metadata.CWE) > 0 {
			vuln.CWE = r.Metadata.CWE[0]
		}

		vulnerabilities = append(vulnerabilities, vuln)
	}

	return vulnerabilities, nil
}

func (s *securityScanner) parseNancyOutput(output []byte) ([]Vulnerability, error) {
	// Nancy输出格式的解析逻辑
	// 这里简化处理
	return []Vulnerability{}, nil
}

func (s *securityScanner) analyzeDependencies(output []byte) ([]Vulnerability, error) {
	// 分析Go模块依赖的安全漏洞
	// 这里简化处理
	return []Vulnerability{}, nil
}

func (s *securityScanner) parseHadolintOutput(output []byte) ([]Vulnerability, error) {
	// 解析Hadolint输出
	// 这里简化处理
	return []Vulnerability{}, nil
}

func (s *securityScanner) parseLicenseOutput(output []byte) []Vulnerability {
	// 解析许可证问题
	// 这里简化处理
	return []Vulnerability{}
}

// 辅助方法

func (s *securityScanner) isScanTypeEnabled(scanType ScanType) bool {
	for _, enabled := range s.config.EnabledScanTypes {
		if enabled == scanType {
			return true
		}
	}
	return false
}

func (s *securityScanner) mapSemgrepSeverity(severity string) Severity {
	switch strings.ToLower(severity) {
	case "error":
		return SeverityHigh
	case "warning":
		return SeverityMedium
	case "info":
		return SeverityLow
	default:
		return SeverityInfo
	}
}

func (s *securityScanner) filterVulnerabilities(vulns []Vulnerability) []Vulnerability {
	var filtered []Vulnerability

	for _, vuln := range vulns {
		// 根据严重性阈值过滤
		if s.shouldIncludeVuln(vuln) {
			filtered = append(filtered, vuln)
		}
	}

	// 限制结果数量
	if len(filtered) > s.config.MaxResults {
		filtered = filtered[:s.config.MaxResults]
	}

	return filtered
}

func (s *securityScanner) shouldIncludeVuln(vuln Vulnerability) bool {
	thresholdOrder := map[Severity]int{
		SeverityInfo:     0,
		SeverityLow:      1,
		SeverityMedium:   2,
		SeverityHigh:     3,
		SeverityCritical: 4,
	}

	vulnLevel := thresholdOrder[vuln.Severity]
	thresholdLevel := thresholdOrder[s.config.SeverityThreshold]

	return vulnLevel >= thresholdLevel
}

func (s *securityScanner) generateSummary(vulns []Vulnerability) ScanSummary {
	summary := ScanSummary{
		TotalVulns: len(vulns),
		BySeverity: make(map[Severity]int),
	}

	for _, vuln := range vulns {
		summary.BySeverity[vuln.Severity]++
	}

	return summary
}

func getDefaultScanConfig() *ScanConfig {
	return &ScanConfig{
		EnabledScanTypes: []ScanType{
			ScanTypeSAST,
			ScanTypeSCA,
			ScanTypeDependency,
		},
		SeverityThreshold: SeverityMedium,
		ExcludePatterns: []string{
			"**/vendor/**",
			"**/node_modules/**",
			"**/.git/**",
			"**/test/**",
			"**/tests/**",
		},
		ReportFormat:   []string{"json"},
		MaxResults:     1000,
		IncludeFixInfo: true,
		Timeout:        30 * time.Minute,
	}
}

// 其他接口方法的实现...
func (s *securityScanner) ScanRepository(ctx context.Context, repoURL string, scanTypes []ScanType) (*ScanResult, error) {
	// TODO: 实现仓库扫描
	return nil, fmt.Errorf("仓库扫描功能尚未实现")
}

func (s *securityScanner) ScanContainer(ctx context.Context, imageName string) (*ScanResult, error) {
	// TODO: 实现容器镜像扫描
	return nil, fmt.Errorf("容器镜像扫描功能尚未实现")
}

func (s *securityScanner) GetScanResult(ctx context.Context, scanID uuid.UUID) (*ScanResult, error) {
	// TODO: 从存储中获取扫描结果
	return nil, fmt.Errorf("获取扫描结果功能尚未实现")
}

func (s *securityScanner) ListScanResults(ctx context.Context, projectID uuid.UUID, limit int) ([]*ScanResult, error) {
	// TODO: 列出扫描结果
	return nil, fmt.Errorf("列出扫描结果功能尚未实现")
}

func (s *securityScanner) DeleteScanResult(ctx context.Context, scanID uuid.UUID) error {
	// TODO: 删除扫描结果
	return fmt.Errorf("删除扫描结果功能尚未实现")
}

func (s *securityScanner) UpdateScanConfig(ctx context.Context, config *ScanConfig) error {
	s.config = config
	return nil
}

func (s *securityScanner) GetScanConfig(ctx context.Context) (*ScanConfig, error) {
	return s.config, nil
}

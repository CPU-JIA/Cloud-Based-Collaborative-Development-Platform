package security

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cloud-platform/collaborative-dev/shared/logger"
)

// SBOMFormat SBOM格式
type SBOMFormat string

const (
	SBOMFormatSPDX      SBOMFormat = "spdx"      // SPDX格式
	SBOMFormatCycloneDX SBOMFormat = "cyclonedx" // CycloneDX格式
	SBOMFormatSYFT      SBOMFormat = "syft"      // Syft native格式
)

// ComponentType 组件类型
type ComponentType string

const (
	ComponentTypeLibrary     ComponentType = "library"
	ComponentTypeFramework   ComponentType = "framework"
	ComponentTypeApplication ComponentType = "application"
	ComponentTypeContainer   ComponentType = "container"
	ComponentTypeOS          ComponentType = "operating-system"
	ComponentTypeDevice      ComponentType = "device"
	ComponentTypeFile        ComponentType = "file"
)

// Component SBOM组件
type Component struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Type         ComponentType          `json:"type"`
	Supplier     *Organization          `json:"supplier,omitempty"`
	Author       string                 `json:"author,omitempty"`
	Publisher    string                 `json:"publisher,omitempty"`
	Group        string                 `json:"group,omitempty"`
	Scope        string                 `json:"scope,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Licenses     []License              `json:"licenses,omitempty"`
	Copyright    string                 `json:"copyright,omitempty"`
	CPE          string                 `json:"cpe,omitempty"`
	PURL         string                 `json:"purl,omitempty"`
	Hashes       []Hash                 `json:"hashes,omitempty"`
	ExternalRefs []ExternalRef          `json:"external_refs,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
}

// License 许可证信息
type License struct {
	ID   string `json:"id,omitempty"`   // SPDX License ID
	Name string `json:"name,omitempty"` // License name
	Text string `json:"text,omitempty"` // License text
	URL  string `json:"url,omitempty"`  // License URL
}

// Hash 哈希值
type Hash struct {
	Algorithm string `json:"algorithm"` // sha1, sha256, sha512, md5
	Value     string `json:"value"`
}

// ExternalRef 外部引用
type ExternalRef struct {
	Type    string `json:"type"` // website, issue-tracker, vcs
	URL     string `json:"url"`
	Comment string `json:"comment,omitempty"`
}

// Organization 组织信息
type Organization struct {
	Name    string    `json:"name"`
	URL     []string  `json:"url,omitempty"`
	Contact []Contact `json:"contact,omitempty"`
}

// Contact 联系方式
type Contact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// SBOM 软件物料清单
type SBOM struct {
	ID           uuid.UUID     `json:"id"`
	Name         string        `json:"name"`
	Version      string        `json:"version"`
	Format       SBOMFormat    `json:"format"`
	SpecVersion  string        `json:"spec_version"`
	CreatedAt    time.Time     `json:"created_at"`
	Authors      []string      `json:"authors"`
	Supplier     *Organization `json:"supplier,omitempty"`
	Manufacturer *Organization `json:"manufacturer,omitempty"`

	// 主要组件信息
	Components   []Component  `json:"components"`
	Dependencies []Dependency `json:"dependencies,omitempty"`

	// 元数据
	Metadata SBOMMetadata `json:"metadata"`
	Tools    []Tool       `json:"tools"`

	// 漏洞信息（可选）
	Vulnerabilities []SBOMVulnerability `json:"vulnerabilities,omitempty"`
}

// Dependency 依赖关系
type Dependency struct {
	Ref          string   `json:"ref"`          // 组件引用
	Dependencies []string `json:"dependencies"` // 依赖的组件ID列表
}

// SBOMMetadata SBOM元数据
type SBOMMetadata struct {
	Timestamp  time.Time              `json:"timestamp"`
	Tools      []Tool                 `json:"tools"`
	Authors    []string               `json:"authors"`
	Component  Component              `json:"component"` // 主组件
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// Tool 工具信息
type Tool struct {
	Vendor  string `json:"vendor,omitempty"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// SBOMVulnerability SBOM中的漏洞信息
type SBOMVulnerability struct {
	ID          string     `json:"id"`
	Source      VulnSource `json:"source"`
	Ratings     []Rating   `json:"ratings,omitempty"`
	CWEs        []int      `json:"cwes,omitempty"`
	Description string     `json:"description,omitempty"`
	Affects     []Affect   `json:"affects"`
	Published   time.Time  `json:"published,omitempty"`
	Updated     time.Time  `json:"updated,omitempty"`
}

// VulnSource 漏洞来源
type VulnSource struct {
	Name string `json:"name"` // NVD, OSV, GitHub, etc.
	URL  string `json:"url,omitempty"`
}

// Rating 评分
type Rating struct {
	Source   VulnSource `json:"source"`
	Score    float64    `json:"score,omitempty"`
	Severity string     `json:"severity,omitempty"`
	Method   string     `json:"method,omitempty"` // CVSSv2, CVSSv3, etc.
	Vector   string     `json:"vector,omitempty"`
}

// Affect 影响范围
type Affect struct {
	Ref      string         `json:"ref"`      // 组件引用
	Versions []VersionRange `json:"versions"` // 受影响的版本范围
}

// VersionRange 版本范围
type VersionRange struct {
	Version string `json:"version,omitempty"`
	Range   string `json:"range,omitempty"`  // >=1.0.0 <2.0.0
	Status  string `json:"status,omitempty"` // affected, unaffected, unknown
}

// SBOMGenerator SBOM生成器接口
type SBOMGenerator interface {
	// SBOM生成
	GenerateSBOM(ctx context.Context, projectPath string, format SBOMFormat) (*SBOM, error)
	GenerateFromContainer(ctx context.Context, imageName string, format SBOMFormat) (*SBOM, error)
	GenerateFromRepository(ctx context.Context, repoURL string, format SBOMFormat) (*SBOM, error)

	// SBOM导出
	ExportSBOM(ctx context.Context, sbom *SBOM, format SBOMFormat) ([]byte, error)
	ExportToFile(ctx context.Context, sbom *SBOM, format SBOMFormat, filePath string) error

	// SBOM分析
	AnalyzeDependencies(ctx context.Context, projectPath string) ([]Component, error)
	DetectLicenses(ctx context.Context, components []Component) error
	EnrichWithVulnerabilities(ctx context.Context, sbom *SBOM) error

	// SBOM验证
	ValidateSBOM(ctx context.Context, sbom *SBOM) error
	CompareSBOMs(ctx context.Context, sbom1, sbom2 *SBOM) (*SBOMComparison, error)
}

// SBOMComparison SBOM比较结果
type SBOMComparison struct {
	Added    []Component       `json:"added"`
	Removed  []Component       `json:"removed"`
	Modified []ComponentDiff   `json:"modified"`
	Summary  ComparisonSummary `json:"summary"`
}

// ComponentDiff 组件差异
type ComponentDiff struct {
	Component Component                  `json:"component"`
	Changes   map[string]ComponentChange `json:"changes"`
}

// ComponentChange 组件变更
type ComponentChange struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

// ComparisonSummary 比较摘要
type ComparisonSummary struct {
	TotalAdded    int `json:"total_added"`
	TotalRemoved  int `json:"total_removed"`
	TotalModified int `json:"total_modified"`
	RiskScore     int `json:"risk_score"` // 0-100的风险评分
}

// sbomGenerator SBOM生成器实现
type sbomGenerator struct {
	logger logger.Logger
	config *SBOMConfig
}

// SBOMConfig SBOM配置
type SBOMConfig struct {
	DefaultFormat          SBOMFormat `json:"default_format"`
	IncludeLicenses        bool       `json:"include_licenses"`
	IncludeHashes          bool       `json:"include_hashes"`
	IncludeDependencies    bool       `json:"include_dependencies"`
	IncludeVulnerabilities bool       `json:"include_vulnerabilities"`
	ExcludeDevDeps         bool       `json:"exclude_dev_dependencies"`
	ExcludeTestFiles       bool       `json:"exclude_test_files"`
	Tools                  []string   `json:"tools"` // syft, cyclonedx-cli, spdx-tools
}

// NewSBOMGenerator 创建SBOM生成器实例
func NewSBOMGenerator(logger logger.Logger, config *SBOMConfig) SBOMGenerator {
	if config == nil {
		config = getDefaultSBOMConfig()
	}

	return &sbomGenerator{
		logger: logger,
		config: config,
	}
}

// GenerateSBOM 生成SBOM
func (g *sbomGenerator) GenerateSBOM(ctx context.Context, projectPath string, format SBOMFormat) (*SBOM, error) {
	g.logger.Info("开始生成SBOM", "project_path", projectPath, "format", format)

	startTime := time.Now()
	sbomID := uuid.New()

	// 分析项目依赖
	components, err := g.AnalyzeDependencies(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("分析依赖失败: %w", err)
	}

	// 检测许可证信息
	if g.config.IncludeLicenses {
		if err := g.DetectLicenses(ctx, components); err != nil {
			g.logger.Warn("检测许可证失败", "error", err)
		}
	}

	// 创建SBOM
	sbom := &SBOM{
		ID:          sbomID,
		Name:        filepath.Base(projectPath),
		Version:     "1.0.0", // 可以从git tag或版本文件读取
		Format:      format,
		SpecVersion: g.getSpecVersion(format),
		CreatedAt:   startTime,
		Authors:     []string{"SBOM Generator"},
		Components:  components,
		Metadata: SBOMMetadata{
			Timestamp: startTime,
			Tools: []Tool{
				{
					Name:    "sbom-generator",
					Version: "1.0.0",
					Vendor:  "Cloud Platform",
				},
			},
			Authors: []string{"SBOM Generator"},
			Component: Component{
				ID:   sbomID.String(),
				Name: filepath.Base(projectPath),
				Type: ComponentTypeApplication,
			},
		},
	}

	// 添加依赖关系
	if g.config.IncludeDependencies {
		dependencies, err := g.analyzeDependencyGraph(ctx, projectPath, components)
		if err != nil {
			g.logger.Warn("分析依赖关系失败", "error", err)
		} else {
			sbom.Dependencies = dependencies
		}
	}

	// 添加漏洞信息
	if g.config.IncludeVulnerabilities {
		if err := g.EnrichWithVulnerabilities(ctx, sbom); err != nil {
			g.logger.Warn("添加漏洞信息失败", "error", err)
		}
	}

	g.logger.Info("SBOM生成完成",
		"sbom_id", sbomID,
		"components", len(components),
		"duration", time.Since(startTime),
	)

	return sbom, nil
}

// AnalyzeDependencies 分析项目依赖
func (g *sbomGenerator) AnalyzeDependencies(ctx context.Context, projectPath string) ([]Component, error) {
	// 检测项目类型并使用相应的工具
	if g.isGoProject(projectPath) {
		return g.analyzeGoDependencies(ctx, projectPath)
	}

	// 使用Syft作为通用分析工具
	return g.analyzeDependenciesWithSyft(ctx, projectPath)
}

// isGoProject 检查是否为Go项目
func (g *sbomGenerator) isGoProject(projectPath string) bool {
	goModPath := filepath.Join(projectPath, "go.mod")
	if _, err := filepath.Glob(goModPath); err == nil {
		return true
	}
	return false
}

// analyzeGoDependencies 分析Go项目依赖
func (g *sbomGenerator) analyzeGoDependencies(ctx context.Context, projectPath string) ([]Component, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", "all")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行go list失败: %w", err)
	}

	return g.parseGoModules(output)
}

// parseGoModules 解析Go模块信息
func (g *sbomGenerator) parseGoModules(output []byte) ([]Component, error) {
	var components []Component

	// 按行分割JSON对象
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var module struct {
			Path    string `json:"Path"`
			Version string `json:"Version"`
			Main    bool   `json:"Main,omitempty"`
			Dir     string `json:"Dir,omitempty"`
		}

		if err := json.Unmarshal([]byte(line), &module); err != nil {
			continue // 跳过无效的JSON
		}

		component := Component{
			ID:      fmt.Sprintf("go-module:%s@%s", module.Path, module.Version),
			Name:    module.Path,
			Version: module.Version,
			Type:    ComponentTypeLibrary,
			PURL:    fmt.Sprintf("pkg:golang/%s@%s", module.Path, module.Version),
		}

		if module.Main {
			component.Type = ComponentTypeApplication
		}

		components = append(components, component)
	}

	return components, nil
}

// analyzeDependenciesWithSyft 使用Syft分析依赖
func (g *sbomGenerator) analyzeDependenciesWithSyft(ctx context.Context, projectPath string) ([]Component, error) {
	cmd := exec.CommandContext(ctx, "syft",
		"packages",
		projectPath,
		"-o", "json",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行syft失败: %w", err)
	}

	return g.parseSyftOutput(output)
}

// parseSyftOutput 解析Syft输出
func (g *sbomGenerator) parseSyftOutput(output []byte) ([]Component, error) {
	var syftResult struct {
		Artifacts []struct {
			Name      string `json:"name"`
			Version   string `json:"version"`
			Type      string `json:"type"`
			Language  string `json:"language"`
			Locations []struct {
				Path string `json:"path"`
			} `json:"locations"`
			Licenses []string `json:"licenses"`
			PURL     string   `json:"purl"`
		} `json:"artifacts"`
	}

	if err := json.Unmarshal(output, &syftResult); err != nil {
		return nil, fmt.Errorf("解析Syft输出失败: %w", err)
	}

	var components []Component
	for _, artifact := range syftResult.Artifacts {
		component := Component{
			ID:      fmt.Sprintf("%s:%s@%s", artifact.Type, artifact.Name, artifact.Version),
			Name:    artifact.Name,
			Version: artifact.Version,
			Type:    g.mapSyftTypeToComponentType(artifact.Type),
			PURL:    artifact.PURL,
		}

		// 添加许可证信息
		for _, license := range artifact.Licenses {
			component.Licenses = append(component.Licenses, License{
				ID: license,
			})
		}

		components = append(components, component)
	}

	return components, nil
}

// mapSyftTypeToComponentType 映射Syft类型到组件类型
func (g *sbomGenerator) mapSyftTypeToComponentType(syftType string) ComponentType {
	switch syftType {
	case "go-module":
		return ComponentTypeLibrary
	case "npm-package":
		return ComponentTypeLibrary
	case "python-package":
		return ComponentTypeLibrary
	default:
		return ComponentTypeLibrary
	}
}

// DetectLicenses 检测许可证信息
func (g *sbomGenerator) DetectLicenses(ctx context.Context, components []Component) error {
	// 这里可以使用go-licenses或其他许可证检测工具
	// 简化实现
	for i := range components {
		if len(components[i].Licenses) == 0 {
			// 尝试检测许可证
			components[i].Licenses = []License{
				{ID: "UNKNOWN"},
			}
		}
	}
	return nil
}

// EnrichWithVulnerabilities 添加漏洞信息
func (g *sbomGenerator) EnrichWithVulnerabilities(ctx context.Context, sbom *SBOM) error {
	// 这里可以集成OSV数据库或其他漏洞数据源
	// 简化实现
	return nil
}

// analyzeDependencyGraph 分析依赖关系图
func (g *sbomGenerator) analyzeDependencyGraph(ctx context.Context, projectPath string, components []Component) ([]Dependency, error) {
	// 简化实现
	return []Dependency{}, nil
}

// ExportSBOM 导出SBOM
func (g *sbomGenerator) ExportSBOM(ctx context.Context, sbom *SBOM, format SBOMFormat) ([]byte, error) {
	switch format {
	case SBOMFormatSPDX:
		return g.exportToSPDX(sbom)
	case SBOMFormatCycloneDX:
		return g.exportToCycloneDX(sbom)
	case SBOMFormatSYFT:
		return json.MarshalIndent(sbom, "", "  ")
	default:
		return nil, fmt.Errorf("不支持的SBOM格式: %s", format)
	}
}

// exportToSPDX 导出为SPDX格式
func (g *sbomGenerator) exportToSPDX(sbom *SBOM) ([]byte, error) {
	// SPDX格式转换
	spdxDoc := map[string]interface{}{
		"spdxVersion": "SPDX-2.3",
		"creationInfo": map[string]interface{}{
			"created":  sbom.CreatedAt.Format(time.RFC3339),
			"creators": sbom.Authors,
		},
		"name":              sbom.Name,
		"SPDXID":            "SPDXRef-DOCUMENT",
		"documentNamespace": fmt.Sprintf("https://example.com/spdx/%s", sbom.ID),
		"packages":          g.convertComponentsToSPDXPackages(sbom.Components),
	}

	return json.MarshalIndent(spdxDoc, "", "  ")
}

// exportToCycloneDX 导出为CycloneDX格式
func (g *sbomGenerator) exportToCycloneDX(sbom *SBOM) ([]byte, error) {
	// CycloneDX格式转换
	cycloneDXDoc := map[string]interface{}{
		"bomFormat":    "CycloneDX",
		"specVersion":  "1.4",
		"serialNumber": fmt.Sprintf("urn:uuid:%s", sbom.ID),
		"version":      1,
		"metadata": map[string]interface{}{
			"timestamp": sbom.CreatedAt.Format(time.RFC3339),
			"tools":     sbom.Tools,
			"authors":   sbom.Authors,
		},
		"components": g.convertComponentsToCycloneDX(sbom.Components),
	}

	return json.MarshalIndent(cycloneDXDoc, "", "  ")
}

// 辅助转换方法
func (g *sbomGenerator) convertComponentsToSPDXPackages(components []Component) []map[string]interface{} {
	var packages []map[string]interface{}

	for _, comp := range components {
		pkg := map[string]interface{}{
			"SPDXID":           fmt.Sprintf("SPDXRef-%s", comp.ID),
			"name":             comp.Name,
			"downloadLocation": "NOASSERTION",
			"filesAnalyzed":    false,
		}

		if comp.Version != "" {
			pkg["versionInfo"] = comp.Version
		}

		if len(comp.Licenses) > 0 {
			pkg["licenseConcluded"] = comp.Licenses[0].ID
		}

		packages = append(packages, pkg)
	}

	return packages
}

func (g *sbomGenerator) convertComponentsToCycloneDX(components []Component) []map[string]interface{} {
	var cdxComponents []map[string]interface{}

	for _, comp := range components {
		cdxComp := map[string]interface{}{
			"type":    string(comp.Type),
			"name":    comp.Name,
			"version": comp.Version,
		}

		if comp.PURL != "" {
			cdxComp["purl"] = comp.PURL
		}

		if len(comp.Licenses) > 0 {
			var licenses []map[string]interface{}
			for _, license := range comp.Licenses {
				licenses = append(licenses, map[string]interface{}{
					"license": map[string]interface{}{
						"id": license.ID,
					},
				})
			}
			cdxComp["licenses"] = licenses
		}

		cdxComponents = append(cdxComponents, cdxComp)
	}

	return cdxComponents
}

// getSpecVersion 获取规范版本
func (g *sbomGenerator) getSpecVersion(format SBOMFormat) string {
	switch format {
	case SBOMFormatSPDX:
		return "SPDX-2.3"
	case SBOMFormatCycloneDX:
		return "1.4"
	case SBOMFormatSYFT:
		return "1.0"
	default:
		return "1.0"
	}
}

// getDefaultSBOMConfig 获取默认配置
func getDefaultSBOMConfig() *SBOMConfig {
	return &SBOMConfig{
		DefaultFormat:          SBOMFormatCycloneDX,
		IncludeLicenses:        true,
		IncludeHashes:          false,
		IncludeDependencies:    true,
		IncludeVulnerabilities: false,
		ExcludeDevDeps:         true,
		ExcludeTestFiles:       true,
		Tools:                  []string{"syft"},
	}
}

// 其他接口方法的简化实现...
func (g *sbomGenerator) GenerateFromContainer(ctx context.Context, imageName string, format SBOMFormat) (*SBOM, error) {
	return nil, fmt.Errorf("容器SBOM生成功能尚未实现")
}

func (g *sbomGenerator) GenerateFromRepository(ctx context.Context, repoURL string, format SBOMFormat) (*SBOM, error) {
	return nil, fmt.Errorf("仓库SBOM生成功能尚未实现")
}

func (g *sbomGenerator) ExportToFile(ctx context.Context, sbom *SBOM, format SBOMFormat, filePath string) error {
	_, err := g.ExportSBOM(ctx, sbom, format)
	if err != nil {
		return err
	}

	// 写入文件的逻辑
	return fmt.Errorf("文件导出功能尚未实现")
}

func (g *sbomGenerator) ValidateSBOM(ctx context.Context, sbom *SBOM) error {
	return fmt.Errorf("SBOM验证功能尚未实现")
}

func (g *sbomGenerator) CompareSBOMs(ctx context.Context, sbom1, sbom2 *SBOM) (*SBOMComparison, error) {
	return nil, fmt.Errorf("SBOM比较功能尚未实现")
}

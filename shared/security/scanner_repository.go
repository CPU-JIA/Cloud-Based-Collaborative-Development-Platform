package security

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ScanResultModel 扫描结果数据库模型
type ScanResultModel struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID       uuid.UUID `gorm:"type:uuid;index"`
	ScanType        string    `gorm:"type:varchar(50);index"`
	ProjectPath     string    `gorm:"type:text"`
	StartTime       time.Time `gorm:"type:timestamp"`
	EndTime         time.Time `gorm:"type:timestamp"`
	Duration        int64     `gorm:"type:bigint"` // 存储为纳秒
	Status          string    `gorm:"type:varchar(20);index"`
	Vulnerabilities string    `gorm:"type:jsonb"` // 存储为JSON
	Summary         string    `gorm:"type:jsonb"` // 存储为JSON
	Metadata        string    `gorm:"type:jsonb"` // 存储为JSON
	Error           string    `gorm:"type:text"`
	CreatedAt       time.Time `gorm:"type:timestamp"`
	UpdatedAt       time.Time `gorm:"type:timestamp"`
}

// TableName 指定表名
func (ScanResultModel) TableName() string {
	return "security_scan_results"
}

// ScanResultRepository 扫描结果存储接口
type ScanResultRepository interface {
	// 创建扫描结果
	Create(ctx context.Context, result *ScanResult) error
	// 更新扫描结果
	Update(ctx context.Context, result *ScanResult) error
	// 获取扫描结果
	GetByID(ctx context.Context, scanID uuid.UUID) (*ScanResult, error)
	// 列出项目的扫描结果
	ListByProject(ctx context.Context, projectID uuid.UUID, limit int) ([]*ScanResult, error)
	// 删除扫描结果
	Delete(ctx context.Context, scanID uuid.UUID) error
	// 获取最新的扫描结果
	GetLatestByProject(ctx context.Context, projectID uuid.UUID, scanType ScanType) (*ScanResult, error)
	// 统计扫描结果
	GetStatsByProject(ctx context.Context, projectID uuid.UUID) (*ScanStats, error)
}

// ScanStats 扫描统计
type ScanStats struct {
	TotalScans      int              `json:"total_scans"`
	LastScanTime    *time.Time       `json:"last_scan_time"`
	VulnsBySeverity map[Severity]int `json:"vulns_by_severity"`
	VulnsByType     map[ScanType]int `json:"vulns_by_type"`
	TrendData       []ScanTrend      `json:"trend_data"`
}

// ScanTrend 扫描趋势数据
type ScanTrend struct {
	Date       time.Time        `json:"date"`
	TotalVulns int              `json:"total_vulns"`
	BySeverity map[Severity]int `json:"by_severity"`
}

// scanResultRepository 扫描结果存储实现
type scanResultRepository struct {
	db *gorm.DB
}

// NewScanResultRepository 创建扫描结果存储
func NewScanResultRepository(db *gorm.DB) ScanResultRepository {
	return &scanResultRepository{db: db}
}

// Create 创建扫描结果
func (r *scanResultRepository) Create(ctx context.Context, result *ScanResult) error {
	model := r.toModel(result)

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("failed to create scan result: %w", err)
	}

	result.ID = model.ID
	return nil
}

// Update 更新扫描结果
func (r *scanResultRepository) Update(ctx context.Context, result *ScanResult) error {
	model := r.toModel(result)

	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return fmt.Errorf("failed to update scan result: %w", err)
	}

	return nil
}

// GetByID 获取扫描结果
func (r *scanResultRepository) GetByID(ctx context.Context, scanID uuid.UUID) (*ScanResult, error) {
	var model ScanResultModel

	if err := r.db.WithContext(ctx).Where("id = ?", scanID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("scan result not found")
		}
		return nil, fmt.Errorf("failed to get scan result: %w", err)
	}

	return r.fromModel(&model)
}

// ListByProject 列出项目的扫描结果
func (r *scanResultRepository) ListByProject(ctx context.Context, projectID uuid.UUID, limit int) ([]*ScanResult, error) {
	var models []ScanResultModel

	query := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to list scan results: %w", err)
	}

	results := make([]*ScanResult, len(models))
	for i, model := range models {
		result, err := r.fromModel(&model)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}

	return results, nil
}

// Delete 删除扫描结果
func (r *scanResultRepository) Delete(ctx context.Context, scanID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", scanID).Delete(&ScanResultModel{}).Error; err != nil {
		return fmt.Errorf("failed to delete scan result: %w", err)
	}

	return nil
}

// GetLatestByProject 获取项目最新的扫描结果
func (r *scanResultRepository) GetLatestByProject(ctx context.Context, projectID uuid.UUID, scanType ScanType) (*ScanResult, error) {
	var model ScanResultModel

	query := r.db.WithContext(ctx).Where("project_id = ?", projectID)
	if scanType != "" {
		query = query.Where("scan_type = ?", scanType)
	}

	if err := query.Order("created_at DESC").First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest scan result: %w", err)
	}

	return r.fromModel(&model)
}

// GetStatsByProject 获取项目的扫描统计
func (r *scanResultRepository) GetStatsByProject(ctx context.Context, projectID uuid.UUID) (*ScanStats, error) {
	stats := &ScanStats{
		VulnsBySeverity: make(map[Severity]int),
		VulnsByType:     make(map[ScanType]int),
	}

	// 获取总扫描次数
	var count int64
	if err := r.db.WithContext(ctx).Model(&ScanResultModel{}).
		Where("project_id = ?", projectID).
		Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to count scans: %w", err)
	}
	stats.TotalScans = int(count)

	// 获取最后扫描时间
	var lastScan ScanResultModel
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).
		Order("created_at DESC").
		First(&lastScan).Error; err == nil {
		stats.LastScanTime = &lastScan.CreatedAt
	}

	// 获取最近30天的趋势数据
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	var models []ScanResultModel
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND created_at >= ?", projectID, thirtyDaysAgo).
		Order("created_at").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get trend data: %w", err)
	}

	// 处理趋势数据
	trendMap := make(map[string]*ScanTrend)
	for _, model := range models {
		result, err := r.fromModel(&model)
		if err != nil {
			continue
		}

		dateKey := model.CreatedAt.Format("2006-01-02")
		trend, exists := trendMap[dateKey]
		if !exists {
			trend = &ScanTrend{
				Date:       model.CreatedAt,
				BySeverity: make(map[Severity]int),
			}
			trendMap[dateKey] = trend
		}

		// 统计漏洞
		for _, vuln := range result.Vulnerabilities {
			trend.TotalVulns++
			trend.BySeverity[vuln.Severity]++
			stats.VulnsBySeverity[vuln.Severity]++
		}

		stats.VulnsByType[result.ScanType]++
	}

	// 转换趋势数据
	for _, trend := range trendMap {
		stats.TrendData = append(stats.TrendData, *trend)
	}

	return stats, nil
}

// toModel 转换为数据库模型
func (r *scanResultRepository) toModel(result *ScanResult) *ScanResultModel {
	vulnsJSON, _ := json.Marshal(result.Vulnerabilities)
	summaryJSON, _ := json.Marshal(result.Summary)
	metadataJSON, _ := json.Marshal(result.Metadata)

	// 从ProjectPath提取ProjectID（简化处理）
	projectID := uuid.New() // 实际应该从路径或其他方式获取

	return &ScanResultModel{
		ID:              result.ID,
		ProjectID:       projectID,
		ScanType:        string(result.ScanType),
		ProjectPath:     result.ProjectPath,
		StartTime:       result.StartTime,
		EndTime:         result.EndTime,
		Duration:        int64(result.Duration),
		Status:          result.Status,
		Vulnerabilities: string(vulnsJSON),
		Summary:         string(summaryJSON),
		Metadata:        string(metadataJSON),
		Error:           result.Error,
	}
}

// fromModel 从数据库模型转换
func (r *scanResultRepository) fromModel(model *ScanResultModel) (*ScanResult, error) {
	result := &ScanResult{
		ID:          model.ID,
		ScanType:    ScanType(model.ScanType),
		ProjectPath: model.ProjectPath,
		StartTime:   model.StartTime,
		EndTime:     model.EndTime,
		Duration:    time.Duration(model.Duration),
		Status:      model.Status,
		Error:       model.Error,
	}

	// 解析JSON字段
	if model.Vulnerabilities != "" {
		if err := json.Unmarshal([]byte(model.Vulnerabilities), &result.Vulnerabilities); err != nil {
			return nil, fmt.Errorf("failed to unmarshal vulnerabilities: %w", err)
		}
	}

	if model.Summary != "" {
		if err := json.Unmarshal([]byte(model.Summary), &result.Summary); err != nil {
			return nil, fmt.Errorf("failed to unmarshal summary: %w", err)
		}
	}

	if model.Metadata != "" {
		if err := json.Unmarshal([]byte(model.Metadata), &result.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return result, nil
}

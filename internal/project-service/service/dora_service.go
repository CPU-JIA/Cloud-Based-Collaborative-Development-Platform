package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DORAService DORA指标服务接口
type DORAService interface {
	// DORA指标计算
	CalculateDORAMetrics(ctx context.Context, projectID uuid.UUID, period time.Time, userID, tenantID uuid.UUID) (*models.DORAMetrics, error)
	BatchCalculateDORAMetrics(ctx context.Context, projectIDs []uuid.UUID, startPeriod, endPeriod time.Time, userID, tenantID uuid.UUID) error

	// 历史指标查询
	GetDORAMetrics(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time, userID, tenantID uuid.UUID) ([]models.DORAMetrics, error)
	GetDORAMetricsByID(ctx context.Context, metricsID uuid.UUID, userID, tenantID uuid.UUID) (*models.DORAMetrics, error)

	// 趋势分析
	GetDORATrends(ctx context.Context, projectID uuid.UUID, period string, limit int, userID, tenantID uuid.UUID) (*DORATrendData, error)
	GetDORAComparison(ctx context.Context, projectID uuid.UUID, compareWithProject *uuid.UUID, period string, userID, tenantID uuid.UUID) (*DORAComparisonData, error)

	// 基准测试
	GetDORABenchmark(ctx context.Context, projectID uuid.UUID, industry string, userID, tenantID uuid.UUID) (*DORABenchmarkData, error)

	// 数据收集
	CollectDeploymentData(ctx context.Context, projectID uuid.UUID, deploymentData *DeploymentEventData) error
	CollectChangeData(ctx context.Context, projectID uuid.UUID, changeData *ChangeEventData) error
	CollectIncidentData(ctx context.Context, projectID uuid.UUID, incidentData *IncidentEventData) error

	// 自动化收集
	EnableAutoCollection(ctx context.Context, projectID uuid.UUID, config *AutoCollectionConfig, userID, tenantID uuid.UUID) error
	DisableAutoCollection(ctx context.Context, projectID uuid.UUID, userID, tenantID uuid.UUID) error
}

// DORAServiceImpl DORA指标服务实现
type DORAServiceImpl struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewDORAService 创建DORA指标服务
func NewDORAService(db *gorm.DB, logger *zap.Logger) DORAService {
	return &DORAServiceImpl{
		db:     db,
		logger: logger,
	}
}

// DeploymentEventData 部署事件数据
type DeploymentEventData struct {
	DeploymentID   string    `json:"deployment_id"`
	Environment    string    `json:"environment"` // production, staging, development
	Status         string    `json:"status"`      // success, failure, rollback
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	CommitSHA      string    `json:"commit_sha"`
	Branch         string    `json:"branch"`
	ReleaseVersion string    `json:"release_version"`
	TriggeredBy    string    `json:"triggered_by"`    // user_id or automated
	DeploymentType string    `json:"deployment_type"` // blue-green, rolling, canary
}

// ChangeEventData 变更事件数据
type ChangeEventData struct {
	ChangeID        string     `json:"change_id"`
	Type            string     `json:"type"` // commit, pull_request, merge
	CommitSHA       string     `json:"commit_sha"`
	AuthorID        string     `json:"author_id"`
	CreatedTime     time.Time  `json:"created_time"`
	MergedTime      *time.Time `json:"merged_time,omitempty"`
	FirstCommitTime time.Time  `json:"first_commit_time"`
	LinesAdded      int        `json:"lines_added"`
	LinesDeleted    int        `json:"lines_deleted"`
	FilesChanged    int        `json:"files_changed"`
	ReviewTime      *float64   `json:"review_time_hours,omitempty"`
	TestsPassed     bool       `json:"tests_passed"`
}

// IncidentEventData 事件数据
type IncidentEventData struct {
	IncidentID       string     `json:"incident_id"`
	Severity         string     `json:"severity"` // critical, high, medium, low
	Status           string     `json:"status"`   // open, investigating, resolved, closed
	StartTime        time.Time  `json:"start_time"`
	ResolvedTime     *time.Time `json:"resolved_time,omitempty"`
	DetectionTime    time.Time  `json:"detection_time"`
	ImpactedServices []string   `json:"impacted_services"`
	RootCause        string     `json:"root_cause"`
	CausedByChange   *string    `json:"caused_by_change,omitempty"` // commit_sha
	ResponderIDs     []string   `json:"responder_ids"`
}

// AutoCollectionConfig 自动收集配置
type AutoCollectionConfig struct {
	EnableDeploymentTracking bool             `json:"enable_deployment_tracking"`
	EnableGitIntegration     bool             `json:"enable_git_integration"`
	EnableCICD               bool             `json:"enable_cicd"`
	EnableIncidentTracking   bool             `json:"enable_incident_tracking"`
	Webhooks                 []WebhookConfig  `json:"webhooks"`
	GitRepositories          []GitRepoConfig  `json:"git_repositories"`
	CICDPlatforms            []CICDConfig     `json:"cicd_platforms"`
	IncidentSources          []IncidentConfig `json:"incident_sources"`
}

// WebhookConfig Webhook配置
type WebhookConfig struct {
	URL     string            `json:"url"`
	Secret  string            `json:"secret"`
	Events  []string          `json:"events"`
	Headers map[string]string `json:"headers"`
}

// GitRepoConfig Git仓库配置
type GitRepoConfig struct {
	RepoURL     string `json:"repo_url"`
	Branch      string `json:"branch"`
	AccessToken string `json:"access_token"`
}

// CICDConfig CI/CD平台配置
type CICDConfig struct {
	Platform    string `json:"platform"` // jenkins, github_actions, gitlab_ci
	URL         string `json:"url"`
	AccessToken string `json:"access_token"`
	ProjectID   string `json:"project_id"`
}

// IncidentConfig 事件源配置
type IncidentConfig struct {
	Source      string `json:"source"` // pagerduty, opsgenie, custom
	URL         string `json:"url"`
	AccessToken string `json:"access_token"`
	ProjectID   string `json:"project_id"`
}

// DORATrendData DORA趋势数据
type DORATrendData struct {
	ProjectID  uuid.UUID        `json:"project_id"`
	Period     string           `json:"period"`
	DataPoints []DORATrendPoint `json:"data_points"`
	Summary    DORATrendSummary `json:"summary"`
}

// DORATrendPoint DORA趋势数据点
type DORATrendPoint struct {
	Date                time.Time `json:"date"`
	DeploymentFrequency float64   `json:"deployment_frequency"`
	LeadTimeHours       float64   `json:"lead_time_hours"`
	ChangeFailureRate   float64   `json:"change_failure_rate"`
	MTTRHours           float64   `json:"mttr_hours"`
	DORALevel           string    `json:"dora_level"`
	OverallScore        float64   `json:"overall_score"`
}

// DORATrendSummary DORA趋势摘要
type DORATrendSummary struct {
	AverageDeploymentFreq float64 `json:"average_deployment_freq"`
	AverageLeadTime       float64 `json:"average_lead_time"`
	AverageChangeFailure  float64 `json:"average_change_failure"`
	AverageMTTR           float64 `json:"average_mttr"`
	BestPerformingMetric  string  `json:"best_performing_metric"`
	WorstPerformingMetric string  `json:"worst_performing_metric"`
	TrendDirection        string  `json:"trend_direction"` // improving, declining, stable
}

// DORAComparisonData DORA对比数据
type DORAComparisonData struct {
	BaseProject        uuid.UUID             `json:"base_project"`
	CompareWithProject *uuid.UUID            `json:"compare_with_project,omitempty"`
	Period             string                `json:"period"`
	BaseMetrics        DORATrendSummary      `json:"base_metrics"`
	CompareMetrics     *DORATrendSummary     `json:"compare_metrics,omitempty"`
	IndustryBenchmark  *DORABenchmarkSummary `json:"industry_benchmark,omitempty"`
	Recommendations    []DORARecommendation  `json:"recommendations"`
}

// DORABenchmarkData DORA基准数据
type DORABenchmarkData struct {
	Industry    string                  `json:"industry"`
	ProjectID   uuid.UUID               `json:"project_id"`
	Period      string                  `json:"period"`
	Benchmark   DORABenchmarkSummary    `json:"benchmark"`
	ProjectData DORATrendSummary        `json:"project_data"`
	Comparison  DORABenchmarkComparison `json:"comparison"`
}

// DORABenchmarkSummary DORA基准摘要
type DORABenchmarkSummary struct {
	EliteThresholds  DORAThresholds `json:"elite_thresholds"`
	HighThresholds   DORAThresholds `json:"high_thresholds"`
	MediumThresholds DORAThresholds `json:"medium_thresholds"`
	LowThresholds    DORAThresholds `json:"low_thresholds"`
}

// DORAThresholds DORA阈值
type DORAThresholds struct {
	DeploymentFrequency float64 `json:"deployment_frequency"`
	LeadTimeHours       float64 `json:"lead_time_hours"`
	ChangeFailureRate   float64 `json:"change_failure_rate"`
	MTTRHours           float64 `json:"mttr_hours"`
}

// DORABenchmarkComparison DORA基准对比
type DORABenchmarkComparison struct {
	Level              string  `json:"level"`
	DeploymentFreqDiff float64 `json:"deployment_freq_diff"`
	LeadTimeDiff       float64 `json:"lead_time_diff"`
	ChangeFailureDiff  float64 `json:"change_failure_diff"`
	MTTRDiff           float64 `json:"mttr_diff"`
	OverallScoreDiff   float64 `json:"overall_score_diff"`
}

// Note: DORARecommendation is defined in dashboard_dto.go

// CalculateDORAMetrics 计算DORA指标
func (d *DORAServiceImpl) CalculateDORAMetrics(ctx context.Context, projectID uuid.UUID, period time.Time, userID, tenantID uuid.UUID) (*models.DORAMetrics, error) {
	// 计算指标周期（月度）
	startDate := time.Date(period.Year(), period.Month(), 1, 0, 0, 0, 0, period.Location())
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Second)

	d.logger.Info("开始计算DORA指标",
		zap.String("project_id", projectID.String()),
		zap.Time("start_date", startDate),
		zap.Time("end_date", endDate))

	// 1. 计算部署频率 (Deployment Frequency)
	deploymentFreq, deploymentCount, err := d.calculateDeploymentFrequency(ctx, projectID, startDate, endDate)
	if err != nil {
		d.logger.Error("计算部署频率失败", zap.Error(err))
		return nil, fmt.Errorf("failed to calculate deployment frequency: %w", err)
	}

	// 2. 计算变更前置时间 (Lead Time for Changes)
	leadTimeHours, leadTimeP50, leadTimeP90, err := d.calculateLeadTime(ctx, projectID, startDate, endDate)
	if err != nil {
		d.logger.Error("计算前置时间失败", zap.Error(err))
		return nil, fmt.Errorf("failed to calculate lead time: %w", err)
	}

	// 3. 计算变更失败率 (Change Failure Rate)
	changeFailureRate, totalChanges, failedChanges, err := d.calculateChangeFailureRate(ctx, projectID, startDate, endDate)
	if err != nil {
		d.logger.Error("计算变更失败率失败", zap.Error(err))
		return nil, fmt.Errorf("failed to calculate change failure rate: %w", err)
	}

	// 4. 计算恢复时间 (Mean Time to Recovery)
	mttr, recoveryTimeHours, incidentCount, err := d.calculateMTTR(ctx, projectID, startDate, endDate)
	if err != nil {
		d.logger.Error("计算恢复时间失败", zap.Error(err))
		return nil, fmt.Errorf("failed to calculate MTTR: %w", err)
	}

	// 构建DORA指标
	doraMetrics := &models.DORAMetrics{
		ProjectID:           projectID,
		MetricDate:          startDate,
		DeploymentCount:     deploymentCount,
		DeploymentFrequency: deploymentFreq,
		LeadTimeHours:       leadTimeHours,
		LeadTimeP50:         leadTimeP50,
		LeadTimeP90:         leadTimeP90,
		TotalChanges:        totalChanges,
		FailedChanges:       failedChanges,
		ChangeFailureRate:   changeFailureRate,
		IncidentCount:       incidentCount,
		RecoveryTimeHours:   recoveryTimeHours,
		MTTR:                mttr,
	}

	// 计算DORA等级和综合评分
	doraMetrics.DORALevel = doraMetrics.CalculateDORALevel()
	doraMetrics.OverallScore = doraMetrics.CalculateOverallScore()

	// 保存到数据库
	if err := d.db.WithContext(ctx).Create(doraMetrics).Error; err != nil {
		d.logger.Error("保存DORA指标失败", zap.Error(err))
		return nil, fmt.Errorf("failed to save DORA metrics: %w", err)
	}

	d.logger.Info("DORA指标计算完成",
		zap.String("project_id", projectID.String()),
		zap.String("dora_level", doraMetrics.DORALevel),
		zap.Float64("overall_score", doraMetrics.OverallScore))

	return doraMetrics, nil
}

// calculateDeploymentFrequency 计算部署频率
func (d *DORAServiceImpl) calculateDeploymentFrequency(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time) (float64, int, error) {
	// 查询部署记录（这里需要根据实际的部署数据表结构调整）
	var deploymentCount int64

	// 暂时使用项目的Sprint完成作为部署的代理指标
	err := d.db.WithContext(ctx).
		Model(&models.Sprint{}).
		Where("project_id = ? AND status = ? AND end_date BETWEEN ? AND ?",
			projectID, models.SprintStatusClosed, startDate, endDate).
		Count(&deploymentCount).Error

	if err != nil {
		return 0, 0, err
	}

	// 计算天数
	days := endDate.Sub(startDate).Hours() / 24
	if days <= 0 {
		days = 1
	}

	// 部署频率：每天的部署次数
	frequency := float64(deploymentCount) / days

	return frequency, int(deploymentCount), nil
}

// calculateLeadTime 计算变更前置时间
func (d *DORAServiceImpl) calculateLeadTime(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time) (float64, float64, float64, error) {
	// 查询已完成的任务，计算从创建到完成的时间
	var tasks []struct {
		CreatedAt   time.Time
		CompletedAt *time.Time
		LeadTime    float64
	}

	query := `
		SELECT 
			created_at,
			updated_at as completed_at,
			EXTRACT(EPOCH FROM (updated_at - created_at))/3600.0 as lead_time
		FROM agile_tasks 
		WHERE project_id = ? 
			AND status = 'done' 
			AND updated_at BETWEEN ? AND ?
			AND deleted_at IS NULL
	`

	if err := d.db.WithContext(ctx).Raw(query, projectID, startDate, endDate).Scan(&tasks).Error; err != nil {
		return 0, 0, 0, err
	}

	if len(tasks) == 0 {
		return 0, 0, 0, nil
	}

	// 提取前置时间数据
	leadTimes := make([]float64, 0, len(tasks))
	totalLeadTime := 0.0

	for _, task := range tasks {
		if task.CompletedAt != nil && task.LeadTime > 0 {
			leadTimes = append(leadTimes, task.LeadTime)
			totalLeadTime += task.LeadTime
		}
	}

	if len(leadTimes) == 0 {
		return 0, 0, 0, nil
	}

	// 计算平均值
	avgLeadTime := totalLeadTime / float64(len(leadTimes))

	// 计算中位数(P50)和90分位数(P90)
	p50, p90 := calculatePercentiles(leadTimes, []float64{0.5, 0.9})

	return avgLeadTime, p50, p90, nil
}

// calculateChangeFailureRate 计算变更失败率
func (d *DORAServiceImpl) calculateChangeFailureRate(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time) (float64, int, int, error) {
	// 统计总变更数（已完成的任务）
	var totalChanges int64
	err := d.db.WithContext(ctx).
		Model(&models.AgileTask{}).
		Where("project_id = ? AND status IN ('done', 'cancelled') AND updated_at BETWEEN ? AND ?",
			projectID, startDate, endDate).
		Count(&totalChanges).Error

	if err != nil {
		return 0, 0, 0, err
	}

	// 统计失败的变更（被取消的任务或者有bug类型的任务）
	var failedChanges int64
	err = d.db.WithContext(ctx).
		Model(&models.AgileTask{}).
		Where("project_id = ? AND (status = 'cancelled' OR type = 'bug') AND updated_at BETWEEN ? AND ?",
			projectID, startDate, endDate).
		Count(&failedChanges).Error

	if err != nil {
		return 0, 0, 0, err
	}

	// 计算失败率
	var failureRate float64
	if totalChanges > 0 {
		failureRate = float64(failedChanges) / float64(totalChanges) * 100.0
	}

	return failureRate, int(totalChanges), int(failedChanges), nil
}

// calculateMTTR 计算平均故障恢复时间
func (d *DORAServiceImpl) calculateMTTR(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time) (float64, float64, int, error) {
	// 查询bug类型的任务，计算从创建到解决的时间
	var incidents []struct {
		CreatedAt    time.Time
		ResolvedAt   *time.Time
		RecoveryTime float64
	}

	query := `
		SELECT 
			created_at,
			updated_at as resolved_at,
			EXTRACT(EPOCH FROM (updated_at - created_at))/3600.0 as recovery_time
		FROM agile_tasks 
		WHERE project_id = ? 
			AND type = 'bug' 
			AND status = 'done'
			AND updated_at BETWEEN ? AND ?
			AND deleted_at IS NULL
	`

	if err := d.db.WithContext(ctx).Raw(query, projectID, startDate, endDate).Scan(&incidents).Error; err != nil {
		return 0, 0, 0, err
	}

	if len(incidents) == 0 {
		return 0, 0, 0, nil
	}

	// 计算恢复时间
	totalRecoveryTime := 0.0
	validIncidents := 0

	for _, incident := range incidents {
		if incident.ResolvedAt != nil && incident.RecoveryTime > 0 {
			totalRecoveryTime += incident.RecoveryTime
			validIncidents++
		}
	}

	var avgRecoveryTime float64
	if validIncidents > 0 {
		avgRecoveryTime = totalRecoveryTime / float64(validIncidents)
	}

	return avgRecoveryTime, totalRecoveryTime, len(incidents), nil
}

// BatchCalculateDORAMetrics 批量计算DORA指标
func (d *DORAServiceImpl) BatchCalculateDORAMetrics(ctx context.Context, projectIDs []uuid.UUID, startPeriod, endPeriod time.Time, userID, tenantID uuid.UUID) error {
	d.logger.Info("开始批量计算DORA指标",
		zap.Int("project_count", len(projectIDs)),
		zap.Time("start_period", startPeriod),
		zap.Time("end_period", endPeriod))

	var errors []error
	successCount := 0

	for _, projectID := range projectIDs {
		// 按月份计算
		currentPeriod := startPeriod
		for currentPeriod.Before(endPeriod) {
			_, err := d.CalculateDORAMetrics(ctx, projectID, currentPeriod, userID, tenantID)
			if err != nil {
				d.logger.Error("项目DORA指标计算失败",
					zap.String("project_id", projectID.String()),
					zap.Time("period", currentPeriod),
					zap.Error(err))
				errors = append(errors, fmt.Errorf("project %s period %v: %w", projectID, currentPeriod, err))
			} else {
				successCount++
			}

			// 移动到下个月
			currentPeriod = currentPeriod.AddDate(0, 1, 0)
		}
	}

	d.logger.Info("批量计算DORA指标完成",
		zap.Int("success_count", successCount),
		zap.Int("error_count", len(errors)))

	if len(errors) > 0 {
		return fmt.Errorf("batch calculation completed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// GetDORAMetrics 获取历史DORA指标
func (d *DORAServiceImpl) GetDORAMetrics(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time, userID, tenantID uuid.UUID) ([]models.DORAMetrics, error) {
	var metrics []models.DORAMetrics

	err := d.db.WithContext(ctx).
		Where("project_id = ? AND metric_date BETWEEN ? AND ?", projectID, startDate, endDate).
		Order("metric_date DESC").
		Find(&metrics).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get DORA metrics: %w", err)
	}

	return metrics, nil
}

// GetDORAMetricsByID 根据ID获取DORA指标
func (d *DORAServiceImpl) GetDORAMetricsByID(ctx context.Context, metricsID uuid.UUID, userID, tenantID uuid.UUID) (*models.DORAMetrics, error) {
	var metrics models.DORAMetrics

	err := d.db.WithContext(ctx).
		Preload("Project").
		First(&metrics, "id = ?", metricsID).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get DORA metrics by ID: %w", err)
	}

	return &metrics, nil
}

// 辅助函数：计算百分位数
func calculatePercentiles(values []float64, percentiles []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}

	// 简单排序实现
	sorted := make([]float64, len(values))
	copy(sorted, values)

	// 冒泡排序
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	// 计算P50和P90
	p50Index := int(float64(len(sorted)) * 0.5)
	p90Index := int(float64(len(sorted)) * 0.9)

	if p50Index >= len(sorted) {
		p50Index = len(sorted) - 1
	}
	if p90Index >= len(sorted) {
		p90Index = len(sorted) - 1
	}

	return sorted[p50Index], sorted[p90Index]
}

// CollectDeploymentData 收集部署数据
func (d *DORAServiceImpl) CollectDeploymentData(ctx context.Context, projectID uuid.UUID, deploymentData *DeploymentEventData) error {
	d.logger.Info("收集部署数据",
		zap.String("project_id", projectID.String()),
		zap.String("deployment_id", deploymentData.DeploymentID),
		zap.String("status", deploymentData.Status))

	// TODO: 实现部署数据的存储
	// 这里应该将部署数据存储到专门的部署事件表中

	return nil
}

// CollectChangeData 收集变更数据
func (d *DORAServiceImpl) CollectChangeData(ctx context.Context, projectID uuid.UUID, changeData *ChangeEventData) error {
	d.logger.Info("收集变更数据",
		zap.String("project_id", projectID.String()),
		zap.String("change_id", changeData.ChangeID),
		zap.String("type", changeData.Type))

	// TODO: 实现变更数据的存储

	return nil
}

// CollectIncidentData 收集事件数据
func (d *DORAServiceImpl) CollectIncidentData(ctx context.Context, projectID uuid.UUID, incidentData *IncidentEventData) error {
	d.logger.Info("收集事件数据",
		zap.String("project_id", projectID.String()),
		zap.String("incident_id", incidentData.IncidentID),
		zap.String("severity", incidentData.Severity))

	// TODO: 实现事件数据的存储

	return nil
}

// EnableAutoCollection 启用自动收集
func (d *DORAServiceImpl) EnableAutoCollection(ctx context.Context, projectID uuid.UUID, config *AutoCollectionConfig, userID, tenantID uuid.UUID) error {
	d.logger.Info("启用DORA指标自动收集",
		zap.String("project_id", projectID.String()))

	// TODO: 实现自动收集配置的存储和启动

	return nil
}

// DisableAutoCollection 禁用自动收集
func (d *DORAServiceImpl) DisableAutoCollection(ctx context.Context, projectID uuid.UUID, userID, tenantID uuid.UUID) error {
	d.logger.Info("禁用DORA指标自动收集",
		zap.String("project_id", projectID.String()))

	// TODO: 实现自动收集的停止

	return nil
}

// GetDORATrends 获取DORA趋势数据（暂时简化实现）
func (d *DORAServiceImpl) GetDORATrends(ctx context.Context, projectID uuid.UUID, period string, limit int, userID, tenantID uuid.UUID) (*DORATrendData, error) {
	// 获取历史指标数据
	var metrics []models.DORAMetrics
	err := d.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("metric_date DESC").
		Limit(limit).
		Find(&metrics).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get DORA trend data: %w", err)
	}

	// 转换为趋势数据
	trendData := &DORATrendData{
		ProjectID:  projectID,
		Period:     period,
		DataPoints: make([]DORATrendPoint, 0, len(metrics)),
	}

	for _, metric := range metrics {
		point := DORATrendPoint{
			Date:                metric.MetricDate,
			DeploymentFrequency: metric.DeploymentFrequency,
			LeadTimeHours:       metric.LeadTimeHours,
			ChangeFailureRate:   metric.ChangeFailureRate,
			MTTRHours:           metric.MTTR,
			DORALevel:           metric.DORALevel,
			OverallScore:        metric.OverallScore,
		}
		trendData.DataPoints = append(trendData.DataPoints, point)
	}

	return trendData, nil
}

// GetDORAComparison 获取DORA对比数据（简化实现）
func (d *DORAServiceImpl) GetDORAComparison(ctx context.Context, projectID uuid.UUID, compareWithProject *uuid.UUID, period string, userID, tenantID uuid.UUID) (*DORAComparisonData, error) {
	// TODO: 实现完整的对比逻辑
	return &DORAComparisonData{
		BaseProject: projectID,
		Period:      period,
	}, nil
}

// GetDORABenchmark 获取DORA基准数据（简化实现）
func (d *DORAServiceImpl) GetDORABenchmark(ctx context.Context, projectID uuid.UUID, industry string, userID, tenantID uuid.UUID) (*DORABenchmarkData, error) {
	// TODO: 实现完整的基准对比逻辑
	return &DORABenchmarkData{
		Industry:  industry,
		ProjectID: projectID,
	}, nil
}

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProjectDashboard 项目仪表板模型
type ProjectDashboard struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID   uuid.UUID `json:"project_id" gorm:"type:uuid;not null;index;unique"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Description *string   `json:"description" gorm:"type:text"`

	// Dashboard配置
	Layout      string   `json:"layout" gorm:"type:jsonb;default:'{}'"` // 布局配置JSON
	Widgets     []string `json:"widgets" gorm:"type:jsonb"`             // 启用的组件列表
	RefreshRate int      `json:"refresh_rate" gorm:"default:300"`       // 刷新间隔(秒)

	// 审计字段
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`
	CreatedBy *uuid.UUID `json:"created_by" gorm:"type:uuid"`

	// 关联关系
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Creator *User    `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

// DORAMetrics DORA四大指标模型
type DORAMetrics struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID  uuid.UUID `json:"project_id" gorm:"type:uuid;not null;index"`
	MetricDate time.Time `json:"metric_date" gorm:"not null;index"`

	// 部署频率 (Deployment Frequency)
	DeploymentCount     int     `json:"deployment_count" gorm:"default:0"`
	DeploymentFrequency float64 `json:"deployment_frequency"` // 部署/天

	// 变更前置时间 (Lead Time for Changes)
	LeadTimeHours float64 `json:"lead_time_hours"` // 平均前置时间(小时)
	LeadTimeP50   float64 `json:"lead_time_p50"`   // 中位数
	LeadTimeP90   float64 `json:"lead_time_p90"`   // 90分位数

	// 变更失败率 (Change Failure Rate)
	TotalChanges      int     `json:"total_changes" gorm:"default:0"`
	FailedChanges     int     `json:"failed_changes" gorm:"default:0"`
	ChangeFailureRate float64 `json:"change_failure_rate"` // 失败率百分比

	// 恢复时间 (Mean Time to Recovery)
	IncidentCount     int     `json:"incident_count" gorm:"default:0"`
	RecoveryTimeHours float64 `json:"recovery_time_hours"` // 平均恢复时间(小时)
	MTTR              float64 `json:"mttr"`                // 平均故障恢复时间

	// 综合评分
	DORALevel    string  `json:"dora_level" gorm:"size:20"` // Elite/High/Medium/Low
	OverallScore float64 `json:"overall_score"`             // 0-100综合评分

	// 审计字段
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// 关联关系
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
}

// ProjectHealthMetrics 项目健康度指标
type ProjectHealthMetrics struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID  uuid.UUID `json:"project_id" gorm:"type:uuid;not null;index"`
	MetricDate time.Time `json:"metric_date" gorm:"not null;index"`

	// 代码质量指标
	CodeCoverage         float64 `json:"code_coverage"`         // 代码覆盖率
	TechnicalDebt        float64 `json:"technical_debt"`        // 技术债务(小时)
	CodeDuplication      float64 `json:"code_duplication"`      // 代码重复率
	CyclomaticComplexity float64 `json:"cyclomatic_complexity"` // 圈复杂度

	// 团队协作指标
	ActiveDevelopers     int     `json:"active_developers"`    // 活跃开发者数
	CommitFrequency      float64 `json:"commit_frequency"`     // 提交频率
	CodeReviewCoverage   float64 `json:"code_review_coverage"` // 代码审查覆盖率
	PullRequestMergeTime float64 `json:"pr_merge_time_hours"`  // PR平均合并时间

	// 交付效率指标
	VelocityPoints     int     `json:"velocity_points"`      // 团队速度(故事点)
	SprintGoalSuccess  float64 `json:"sprint_goal_success"`  // Sprint目标达成率
	TaskCompletionRate float64 `json:"task_completion_rate"` // 任务完成率
	BugRate            float64 `json:"bug_rate"`             // 缺陷率

	// 质量指标
	TestPassRate            float64 `json:"test_pass_rate"`           // 测试通过率
	BuildSuccessRate        float64 `json:"build_success_rate"`       // 构建成功率
	SecurityVulnerabilities int     `json:"security_vulnerabilities"` // 安全漏洞数

	// 综合健康评分
	HealthScore float64 `json:"health_score"`              // 0-100健康评分
	RiskLevel   string  `json:"risk_level" gorm:"size:20"` // Low/Medium/High/Critical

	// 审计字段
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// 关联关系
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
}

// MetricTrend 指标趋势数据
type MetricTrend struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID   uuid.UUID `json:"project_id" gorm:"type:uuid;not null;index"`
	MetricType  string    `json:"metric_type" gorm:"size:100;not null;index"` // dora, health, velocity等
	MetricName  string    `json:"metric_name" gorm:"size:100;not null"`       // 具体指标名称
	MetricValue float64   `json:"metric_value" gorm:"not null"`
	Period      string    `json:"period" gorm:"size:20;not null"` // daily, weekly, monthly
	PeriodStart time.Time `json:"period_start" gorm:"not null;index"`
	PeriodEnd   time.Time `json:"period_end" gorm:"not null;index"`

	// 趋势分析
	TrendDirection string  `json:"trend_direction" gorm:"size:20"` // up, down, stable
	ChangePercent  float64 `json:"change_percent"`                 // 变化百分比

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// AlertRule 告警规则模型
type AlertRule struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID   uuid.UUID `json:"project_id" gorm:"type:uuid;not null;index"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Description *string   `json:"description" gorm:"type:text"`

	// 规则配置
	MetricType string  `json:"metric_type" gorm:"size:100;not null"` // dora, health等
	MetricName string  `json:"metric_name" gorm:"size:100;not null"` // 具体指标
	Operator   string  `json:"operator" gorm:"size:20;not null"`     // >, <, >=, <=, ==
	Threshold  float64 `json:"threshold" gorm:"not null"`
	Severity   string  `json:"severity" gorm:"size:20;not null"` // low, medium, high, critical

	// 状态管理
	IsEnabled     bool       `json:"is_enabled" gorm:"default:true"`
	LastTriggered *time.Time `json:"last_triggered"`
	TriggerCount  int        `json:"trigger_count" gorm:"default:0"`

	// 通知配置
	NotificationChannels []string `json:"notification_channels" gorm:"type:jsonb"` // email, slack, webhook

	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`
	CreatedBy *uuid.UUID `json:"created_by" gorm:"type:uuid"`
}

// DashboardWidget 仪表板组件模型
type DashboardWidget struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	DashboardID uuid.UUID `json:"dashboard_id" gorm:"type:uuid;not null;index"`
	WidgetType  string    `json:"widget_type" gorm:"size:50;not null"` // chart, metric, table等
	Title       string    `json:"title" gorm:"size:255;not null"`

	// 位置和大小
	PositionX int `json:"position_x" gorm:"not null"`
	PositionY int `json:"position_y" gorm:"not null"`
	Width     int `json:"width" gorm:"not null;default:4"`
	Height    int `json:"height" gorm:"not null;default:3"`

	// 配置
	Configuration string `json:"configuration" gorm:"type:jsonb"` // 组件配置JSON
	DataSource    string `json:"data_source" gorm:"size:100"`     // 数据源
	RefreshRate   int    `json:"refresh_rate" gorm:"default:300"` // 刷新间隔

	// 状态
	IsVisible bool `json:"is_visible" gorm:"default:true"`
	Order     int  `json:"order" gorm:"default:0"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// 关联关系
	Dashboard *ProjectDashboard `json:"dashboard,omitempty" gorm:"foreignKey:DashboardID;constraint:OnDelete:CASCADE"`
}

// 枚举常量定义

// DORA等级常量
const (
	DORALevelElite  = "elite"
	DORALevelHigh   = "high"
	DORALevelMedium = "medium"
	DORALevelLow    = "low"
)

// 风险等级常量
const (
	RiskLevelLow      = "low"
	RiskLevelMedium   = "medium"
	RiskLevelHigh     = "high"
	RiskLevelCritical = "critical"
)

// 告警级别常量
const (
	AlertSeverityLow      = "low"
	AlertSeverityMedium   = "medium"
	AlertSeverityHigh     = "high"
	AlertSeverityCritical = "critical"
)

// 组件类型常量
const (
	WidgetTypeChart    = "chart"
	WidgetTypeMetric   = "metric"
	WidgetTypeTable    = "table"
	WidgetTypeProgress = "progress"
	WidgetTypeGauge    = "gauge"
	WidgetTypeHeatmap  = "heatmap"
)

// GORM钩子函数
func (pd *ProjectDashboard) BeforeCreate(tx *gorm.DB) error {
	if pd.ID == uuid.Nil {
		pd.ID = uuid.New()
	}
	return nil
}

func (dm *DORAMetrics) BeforeCreate(tx *gorm.DB) error {
	if dm.ID == uuid.Nil {
		dm.ID = uuid.New()
	}
	return nil
}

func (phm *ProjectHealthMetrics) BeforeCreate(tx *gorm.DB) error {
	if phm.ID == uuid.Nil {
		phm.ID = uuid.New()
	}
	return nil
}

func (mt *MetricTrend) BeforeCreate(tx *gorm.DB) error {
	if mt.ID == uuid.Nil {
		mt.ID = uuid.New()
	}
	return nil
}

func (ar *AlertRule) BeforeCreate(tx *gorm.DB) error {
	if ar.ID == uuid.Nil {
		ar.ID = uuid.New()
	}
	return nil
}

func (dw *DashboardWidget) BeforeCreate(tx *gorm.DB) error {
	if dw.ID == uuid.Nil {
		dw.ID = uuid.New()
	}
	return nil
}

// 表名设置
func (ProjectDashboard) TableName() string {
	return "project_dashboards"
}

func (DORAMetrics) TableName() string {
	return "dora_metrics"
}

func (ProjectHealthMetrics) TableName() string {
	return "project_health_metrics"
}

func (MetricTrend) TableName() string {
	return "metric_trends"
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

func (DashboardWidget) TableName() string {
	return "dashboard_widgets"
}

// 业务方法

// CalculateDORALevel 根据四项指标计算DORA成熟度等级
func (dm *DORAMetrics) CalculateDORALevel() string {
	// Elite级别标准
	if dm.DeploymentFrequency >= 1.0 && // 每天多次部署
		dm.LeadTimeHours <= 24 && // 前置时间小于1天
		dm.ChangeFailureRate <= 0.15 && // 失败率小于15%
		dm.MTTR <= 1 { // 恢复时间小于1小时
		return DORALevelElite
	}

	// High级别标准
	if dm.DeploymentFrequency >= 0.14 && // 每周1-2次部署
		dm.LeadTimeHours <= 168 && // 前置时间小于1周
		dm.ChangeFailureRate <= 0.2 && // 失败率小于20%
		dm.MTTR <= 24 { // 恢复时间小于1天
		return DORALevelHigh
	}

	// Medium级别标准
	if dm.DeploymentFrequency >= 0.033 && // 每月1-2次部署
		dm.LeadTimeHours <= 720 && // 前置时间小于1个月
		dm.ChangeFailureRate <= 0.3 && // 失败率小于30%
		dm.MTTR <= 168 { // 恢复时间小于1周
		return DORALevelMedium
	}

	return DORALevelLow
}

// CalculateOverallScore 计算DORA综合评分
func (dm *DORAMetrics) CalculateOverallScore() float64 {
	var score float64 = 0

	// 部署频率评分 (0-25分)
	if dm.DeploymentFrequency >= 1.0 {
		score += 25
	} else if dm.DeploymentFrequency >= 0.14 {
		score += 20
	} else if dm.DeploymentFrequency >= 0.033 {
		score += 15
	} else {
		score += 10
	}

	// 前置时间评分 (0-25分)
	if dm.LeadTimeHours <= 24 {
		score += 25
	} else if dm.LeadTimeHours <= 168 {
		score += 20
	} else if dm.LeadTimeHours <= 720 {
		score += 15
	} else {
		score += 10
	}

	// 失败率评分 (0-25分)
	if dm.ChangeFailureRate <= 0.15 {
		score += 25
	} else if dm.ChangeFailureRate <= 0.2 {
		score += 20
	} else if dm.ChangeFailureRate <= 0.3 {
		score += 15
	} else {
		score += 10
	}

	// 恢复时间评分 (0-25分)
	if dm.MTTR <= 1 {
		score += 25
	} else if dm.MTTR <= 24 {
		score += 20
	} else if dm.MTTR <= 168 {
		score += 15
	} else {
		score += 10
	}

	return score
}

// CalculateHealthScore 计算项目健康综合评分
func (phm *ProjectHealthMetrics) CalculateHealthScore() float64 {
	var score float64 = 0

	// 代码质量 (30%)
	qualityScore := (phm.CodeCoverage +
		(100 - phm.CodeDuplication) +
		phm.TestPassRate +
		phm.BuildSuccessRate) / 4
	score += qualityScore * 0.3

	// 团队协作 (25%)
	collaborationScore := (phm.CodeReviewCoverage +
		phm.SprintGoalSuccess +
		phm.TaskCompletionRate) / 3
	score += collaborationScore * 0.25

	// 交付效率 (25%)
	efficiencyScore := (phm.SprintGoalSuccess +
		phm.TaskCompletionRate +
		(100 - phm.BugRate)) / 3
	score += efficiencyScore * 0.25

	// 技术债务和安全 (20%)
	debtSecurityScore := 100 - (phm.TechnicalDebt/10 + float64(phm.SecurityVulnerabilities)*5)
	if debtSecurityScore < 0 {
		debtSecurityScore = 0
	}
	score += debtSecurityScore * 0.2

	if score > 100 {
		score = 100
	}

	return score
}

// GetRiskLevel 根据健康评分获取风险等级
func (phm *ProjectHealthMetrics) GetRiskLevel() string {
	if phm.HealthScore >= 80 {
		return RiskLevelLow
	} else if phm.HealthScore >= 60 {
		return RiskLevelMedium
	} else if phm.HealthScore >= 40 {
		return RiskLevelHigh
	}
	return RiskLevelCritical
}

// IsThresholdExceeded 检查是否超过告警阈值
func (ar *AlertRule) IsThresholdExceeded(value float64) bool {
	switch ar.Operator {
	case ">":
		return value > ar.Threshold
	case ">=":
		return value >= ar.Threshold
	case "<":
		return value < ar.Threshold
	case "<=":
		return value <= ar.Threshold
	case "==":
		return value == ar.Threshold
	default:
		return false
	}
}

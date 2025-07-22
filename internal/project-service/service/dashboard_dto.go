package service

import (
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/google/uuid"
)

// Dashboard相关DTO

// CreateDashboardRequest 创建仪表板请求
type CreateDashboardRequest struct {
	ProjectID    uuid.UUID `json:"project_id" binding:"required"`
	Name         string    `json:"name" binding:"required,min=1,max=255"`
	Description  *string   `json:"description,omitempty"`
	Layout       string    `json:"layout,omitempty"`
	Widgets      []string  `json:"widgets,omitempty"`
	RefreshRate  int       `json:"refresh_rate,omitempty" binding:"omitempty,min=30,max=3600"`
}

// UpdateDashboardRequest 更新仪表板请求
type UpdateDashboardRequest struct {
	Name        *string  `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description *string  `json:"description,omitempty"`
	Layout      *string  `json:"layout,omitempty"`
	Widgets     []string `json:"widgets,omitempty"`
	RefreshRate *int     `json:"refresh_rate,omitempty" binding:"omitempty,min=30,max=3600"`
}

// DashboardResponse 仪表板响应
type DashboardResponse struct {
	models.ProjectDashboard
	WidgetCount int                    `json:"widget_count"`
	LastUpdate  time.Time              `json:"last_update"`
	Widgets     []DashboardWidgetInfo  `json:"widgets,omitempty"`
}

// DashboardWidgetInfo 组件信息
type DashboardWidgetInfo struct {
	ID            uuid.UUID   `json:"id"`
	WidgetType    string      `json:"widget_type"`
	Title         string      `json:"title"`
	PositionX     int         `json:"position_x"`
	PositionY     int         `json:"position_y"`
	Width         int         `json:"width"`
	Height        int         `json:"height"`
	Configuration interface{} `json:"configuration"`
	DataSource    string      `json:"data_source"`
	IsVisible     bool        `json:"is_visible"`
	Order         int         `json:"order"`
}

// CreateWidgetRequest 创建组件请求
type CreateWidgetRequest struct {
	DashboardID   uuid.UUID   `json:"dashboard_id" binding:"required"`
	WidgetType    string      `json:"widget_type" binding:"required"`
	Title         string      `json:"title" binding:"required,min=1,max=255"`
	PositionX     int         `json:"position_x" binding:"min=0"`
	PositionY     int         `json:"position_y" binding:"min=0"`
	Width         int         `json:"width" binding:"min=1,max=12"`
	Height        int         `json:"height" binding:"min=1,max=12"`
	Configuration interface{} `json:"configuration,omitempty"`
	DataSource    string      `json:"data_source,omitempty"`
	RefreshRate   int         `json:"refresh_rate,omitempty" binding:"omitempty,min=30"`
}

// UpdateWidgetRequest 更新组件请求
type UpdateWidgetRequest struct {
	Title         *string     `json:"title,omitempty" binding:"omitempty,min=1,max=255"`
	PositionX     *int        `json:"position_x,omitempty" binding:"omitempty,min=0"`
	PositionY     *int        `json:"position_y,omitempty" binding:"omitempty,min=0"`
	Width         *int        `json:"width,omitempty" binding:"omitempty,min=1,max=12"`
	Height        *int        `json:"height,omitempty" binding:"omitempty,min=1,max=12"`
	Configuration interface{} `json:"configuration,omitempty"`
	DataSource    *string     `json:"data_source,omitempty"`
	RefreshRate   *int        `json:"refresh_rate,omitempty" binding:"omitempty,min=30"`
	IsVisible     *bool       `json:"is_visible,omitempty"`
	Order         *int        `json:"order,omitempty"`
}

// DORA指标相关DTO

// DORAMetricsRequest DORA指标请求
type DORAMetricsRequest struct {
	ProjectID uuid.UUID `json:"project_id" binding:"required"`
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
	Period    string    `json:"period,omitempty" binding:"omitempty,oneof=daily weekly monthly"`
}

// DORAMetricsResponse DORA指标响应
type DORAMetricsResponse struct {
	ProjectID   uuid.UUID               `json:"project_id"`
	ProjectName string                  `json:"project_name"`
	Period      string                  `json:"period"`
	StartDate   time.Time               `json:"start_date"`
	EndDate     time.Time               `json:"end_date"`
	Metrics     []DORAMetricsDataPoint  `json:"metrics"`
	Summary     DORAMetricsSummary      `json:"summary"`
	Trends      DORAMetricsTrends       `json:"trends"`
}

// DORAMetricsDataPoint DORA指标数据点
type DORAMetricsDataPoint struct {
	Date                    time.Time `json:"date"`
	DeploymentFrequency     float64   `json:"deployment_frequency"`
	LeadTimeHours           float64   `json:"lead_time_hours"`
	ChangeFailureRate       float64   `json:"change_failure_rate"`
	MTTR                    float64   `json:"mttr"`
	DORALevel               string    `json:"dora_level"`
	OverallScore            float64   `json:"overall_score"`
}

// DORAMetricsSummary DORA指标汇总
type DORAMetricsSummary struct {
	CurrentLevel            string                  `json:"current_level"`
	CurrentScore            float64                 `json:"current_score"`
	AvgDeploymentFrequency  float64                 `json:"avg_deployment_frequency"`
	AvgLeadTimeHours        float64                 `json:"avg_lead_time_hours"`
	AvgChangeFailureRate    float64                 `json:"avg_change_failure_rate"`
	AvgMTTR                 float64                 `json:"avg_mttr"`
	Recommendations         []DORARecommendation    `json:"recommendations"`
}

// DORAMetricsTrends DORA指标趋势
type DORAMetricsTrends struct {
	ScoreTrend              string  `json:"score_trend"`              // up/down/stable
	ScoreChange             float64 `json:"score_change"`             // 变化值
	ScoreChangePercent      float64 `json:"score_change_percent"`     // 变化百分比
	DeploymentFrequencyTrend string `json:"deployment_frequency_trend"`
	LeadTimeTrend           string  `json:"lead_time_trend"`
	ChangeFailureRateTrend  string  `json:"change_failure_rate_trend"`
	MTTRTrend               string  `json:"mttr_trend"`
}

// DORARecommendation DORA改进建议
type DORARecommendation struct {
	Category    string `json:"category"`    // deployment, lead_time, failure_rate, recovery
	Priority    string `json:"priority"`    // high, medium, low
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

// ProjectHealth相关DTO

// ProjectHealthRequest 项目健康度请求
type ProjectHealthRequest struct {
	ProjectID uuid.UUID `json:"project_id" binding:"required"`
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
	Period    string    `json:"period,omitempty" binding:"omitempty,oneof=daily weekly monthly"`
}

// ProjectHealthResponse 项目健康度响应
type ProjectHealthResponse struct {
	ProjectID   uuid.UUID                  `json:"project_id"`
	ProjectName string                     `json:"project_name"`
	Period      string                     `json:"period"`
	StartDate   time.Time                  `json:"start_date"`
	EndDate     time.Time                  `json:"end_date"`
	Metrics     []ProjectHealthDataPoint   `json:"metrics"`
	Summary     ProjectHealthSummary       `json:"summary"`
	Categories  ProjectHealthCategories    `json:"categories"`
	Alerts      []HealthAlert              `json:"alerts"`
}

// ProjectHealthDataPoint 项目健康度数据点
type ProjectHealthDataPoint struct {
	Date                    time.Time `json:"date"`
	HealthScore             float64   `json:"health_score"`
	RiskLevel               string    `json:"risk_level"`
	CodeCoverage            float64   `json:"code_coverage"`
	TestPassRate            float64   `json:"test_pass_rate"`
	BuildSuccessRate        float64   `json:"build_success_rate"`
	TaskCompletionRate      float64   `json:"task_completion_rate"`
	BugRate                 float64   `json:"bug_rate"`
	TechnicalDebt           float64   `json:"technical_debt"`
	SecurityVulnerabilities int       `json:"security_vulnerabilities"`
}

// ProjectHealthSummary 项目健康度汇总
type ProjectHealthSummary struct {
	CurrentScore    float64               `json:"current_score"`
	CurrentRisk     string                `json:"current_risk"`
	ScoreTrend      string                `json:"score_trend"`
	ScoreChange     float64               `json:"score_change"`
	KeyInsights     []string              `json:"key_insights"`
	ActionItems     []HealthActionItem    `json:"action_items"`
}

// ProjectHealthCategories 项目健康度分类指标
type ProjectHealthCategories struct {
	CodeQuality      CategoryScore `json:"code_quality"`
	TeamCollaboration CategoryScore `json:"team_collaboration"`
	DeliveryEfficiency CategoryScore `json:"delivery_efficiency"`
	TechnicalDebt    CategoryScore `json:"technical_debt"`
}

// CategoryScore 分类评分
type CategoryScore struct {
	Score   float64 `json:"score"`
	Level   string  `json:"level"`   // excellent, good, average, poor
	Trend   string  `json:"trend"`   // improving, stable, declining
	Details map[string]float64 `json:"details"`
}

// HealthAlert 健康度告警
type HealthAlert struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`        // metric_threshold, trend_alert, anomaly
	Severity    string    `json:"severity"`    // low, medium, high, critical
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	MetricName  string    `json:"metric_name"`
	Threshold   float64   `json:"threshold"`
	CurrentValue float64  `json:"current_value"`
	TriggeredAt time.Time `json:"triggered_at"`
	IsActive    bool      `json:"is_active"`
}

// HealthActionItem 健康度行动项
type HealthActionItem struct {
	Priority    string `json:"priority"`    // high, medium, low
	Category    string `json:"category"`    // code_quality, collaboration, delivery, security
	Title       string `json:"title"`
	Description string `json:"description"`
	EstimatedEffort string `json:"estimated_effort"` // hours, days, weeks
	Impact      string `json:"impact"`      // high, medium, low
}

// 告警规则相关DTO

// CreateAlertRuleRequest 创建告警规则请求
type CreateAlertRuleRequest struct {
	ProjectID             uuid.UUID `json:"project_id" binding:"required"`
	Name                  string    `json:"name" binding:"required,min=1,max=255"`
	Description           *string   `json:"description,omitempty"`
	MetricType            string    `json:"metric_type" binding:"required"`
	MetricName            string    `json:"metric_name" binding:"required"`
	Operator              string    `json:"operator" binding:"required,oneof=> < >= <= == !="`
	Threshold             float64   `json:"threshold" binding:"required"`
	Severity              string    `json:"severity" binding:"required,oneof=low medium high critical"`
	NotificationChannels  []string  `json:"notification_channels,omitempty"`
}

// UpdateAlertRuleRequest 更新告警规则请求
type UpdateAlertRuleRequest struct {
	Name                  *string  `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description           *string  `json:"description,omitempty"`
	Operator              *string  `json:"operator,omitempty" binding:"omitempty,oneof=> < >= <= == !="`
	Threshold             *float64 `json:"threshold,omitempty"`
	Severity              *string  `json:"severity,omitempty" binding:"omitempty,oneof=low medium high critical"`
	IsEnabled             *bool    `json:"is_enabled,omitempty"`
	NotificationChannels  []string `json:"notification_channels,omitempty"`
}

// AlertRuleResponse 告警规则响应
type AlertRuleResponse struct {
	models.AlertRule
	IsTriggered     bool      `json:"is_triggered"`
	LastValue       *float64  `json:"last_value,omitempty"`
	LastEvaluated   *time.Time `json:"last_evaluated,omitempty"`
	NextEvaluation  *time.Time `json:"next_evaluation,omitempty"`
}

// 趋势分析相关DTO

// MetricTrendRequest 指标趋势请求
type MetricTrendRequest struct {
	ProjectID  uuid.UUID `json:"project_id" binding:"required"`
	MetricType string    `json:"metric_type" binding:"required"`
	MetricName *string   `json:"metric_name,omitempty"`
	Period     string    `json:"period" binding:"required,oneof=daily weekly monthly quarterly yearly"`
	StartDate  time.Time `json:"start_date" binding:"required"`
	EndDate    time.Time `json:"end_date" binding:"required"`
}

// MetricTrendResponse 指标趋势响应
type MetricTrendResponse struct {
	ProjectID   uuid.UUID           `json:"project_id"`
	MetricType  string              `json:"metric_type"`
	MetricName  string              `json:"metric_name"`
	Period      string              `json:"period"`
	StartDate   time.Time           `json:"start_date"`
	EndDate     time.Time           `json:"end_date"`
	DataPoints  []TrendDataPoint    `json:"data_points"`
	Statistics  TrendStatistics     `json:"statistics"`
	Forecast    *TrendForecast      `json:"forecast,omitempty"`
}

// TrendDataPoint 趋势数据点
type TrendDataPoint struct {
	PeriodStart   time.Time `json:"period_start"`
	PeriodEnd     time.Time `json:"period_end"`
	Value         float64   `json:"value"`
	Direction     string    `json:"direction"`     // up, down, stable
	ChangePercent float64   `json:"change_percent"`
}

// TrendStatistics 趋势统计
type TrendStatistics struct {
	Average       float64 `json:"average"`
	Minimum       float64 `json:"minimum"`
	Maximum       float64 `json:"maximum"`
	StandardDev   float64 `json:"standard_deviation"`
	TotalChange   float64 `json:"total_change"`
	TotalChangePercent float64 `json:"total_change_percent"`
	Volatility    string  `json:"volatility"`    // low, medium, high
	OverallTrend  string  `json:"overall_trend"` // improving, stable, declining
}

// TrendForecast 趋势预测
type TrendForecast struct {
	NextPeriodValue    float64   `json:"next_period_value"`
	Confidence         float64   `json:"confidence"`        // 0-100
	PredictionDate     time.Time `json:"prediction_date"`
	TrendContinuation  string    `json:"trend_continuation"` // likely, uncertain, unlikely
}

// 批量操作DTO

// BatchMetricsUpdateRequest 批量更新指标请求
type BatchMetricsUpdateRequest struct {
	ProjectID     uuid.UUID           `json:"project_id" binding:"required"`
	MetricDate    time.Time           `json:"metric_date" binding:"required"`
	DORAMetrics   *DORAMetricsUpdate  `json:"dora_metrics,omitempty"`
	HealthMetrics *HealthMetricsUpdate `json:"health_metrics,omitempty"`
}

// DORAMetricsUpdate DORA指标更新
type DORAMetricsUpdate struct {
	DeploymentCount     *int     `json:"deployment_count,omitempty"`
	LeadTimeHours       *float64 `json:"lead_time_hours,omitempty"`
	TotalChanges        *int     `json:"total_changes,omitempty"`
	FailedChanges       *int     `json:"failed_changes,omitempty"`
	IncidentCount       *int     `json:"incident_count,omitempty"`
	RecoveryTimeHours   *float64 `json:"recovery_time_hours,omitempty"`
}

// HealthMetricsUpdate 健康度指标更新
type HealthMetricsUpdate struct {
	CodeCoverage            *float64 `json:"code_coverage,omitempty"`
	TechnicalDebt           *float64 `json:"technical_debt,omitempty"`
	CodeDuplication         *float64 `json:"code_duplication,omitempty"`
	ActiveDevelopers        *int     `json:"active_developers,omitempty"`
	CommitFrequency         *float64 `json:"commit_frequency,omitempty"`
	CodeReviewCoverage      *float64 `json:"code_review_coverage,omitempty"`
	VelocityPoints          *int     `json:"velocity_points,omitempty"`
	BugRate                 *float64 `json:"bug_rate,omitempty"`
	TestPassRate            *float64 `json:"test_pass_rate,omitempty"`
	BuildSuccessRate        *float64 `json:"build_success_rate,omitempty"`
	SecurityVulnerabilities *int     `json:"security_vulnerabilities,omitempty"`
}

// 组件配置相关DTO

// ChartWidgetConfig 图表组件配置
type ChartWidgetConfig struct {
	ChartType   string                 `json:"chart_type"`   // line, bar, pie, area, gauge
	DataSeries  []ChartDataSeries      `json:"data_series"`
	XAxis       ChartAxisConfig        `json:"x_axis"`
	YAxis       ChartAxisConfig        `json:"y_axis"`
	Colors      []string               `json:"colors,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// ChartDataSeries 图表数据系列
type ChartDataSeries struct {
	Name         string `json:"name"`
	MetricType   string `json:"metric_type"`
	MetricName   string `json:"metric_name"`
	Aggregation  string `json:"aggregation"`  // sum, avg, max, min, count
	Color        string `json:"color,omitempty"`
}

// ChartAxisConfig 图表坐标轴配置
type ChartAxisConfig struct {
	Label      string  `json:"label"`
	Min        *float64 `json:"min,omitempty"`
	Max        *float64 `json:"max,omitempty"`
	Unit       string  `json:"unit,omitempty"`
	Format     string  `json:"format,omitempty"`
}

// MetricWidgetConfig 指标组件配置
type MetricWidgetConfig struct {
	MetricType     string  `json:"metric_type"`
	MetricName     string  `json:"metric_name"`
	DisplayFormat  string  `json:"display_format"` // number, percentage, currency, duration
	Unit           string  `json:"unit,omitempty"`
	Precision      int     `json:"precision,omitempty"`
	ShowTrend      bool    `json:"show_trend,omitempty"`
	ThresholdRules []ThresholdRule `json:"threshold_rules,omitempty"`
}

// ThresholdRule 阈值规则
type ThresholdRule struct {
	Operator  string  `json:"operator"`  // >, <, >=, <=
	Value     float64 `json:"value"`
	Color     string  `json:"color"`     // green, yellow, orange, red
	Label     string  `json:"label,omitempty"`
}

// TableWidgetConfig 表格组件配置
type TableWidgetConfig struct {
	DataSource string              `json:"data_source"`
	Columns    []TableColumnConfig `json:"columns"`
	Pagination bool                `json:"pagination,omitempty"`
	Sorting    bool                `json:"sorting,omitempty"`
	Filtering  bool                `json:"filtering,omitempty"`
}

// TableColumnConfig 表格列配置
type TableColumnConfig struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Type        string `json:"type"`        // text, number, date, percentage
	Width       int    `json:"width,omitempty"`
	Sortable    bool   `json:"sortable,omitempty"`
	Filterable  bool   `json:"filterable,omitempty"`
	Format      string `json:"format,omitempty"`
}

// 缺失的类型定义

// ProjectDashboardData 项目仪表盘数据
type ProjectDashboardData struct {
	ProjectID      uuid.UUID                   `json:"project_id"`
	ProjectName    string                      `json:"project_name"`
	Overview       ProjectOverview             `json:"overview"`
	DORAMetrics    models.DORAMetrics          `json:"dora_metrics"`
	HealthMetrics  models.ProjectHealthMetrics `json:"health_metrics"`
	Sprint         *CurrentSprintInfo          `json:"sprint,omitempty"`
	TaskMetrics    TaskMetrics                 `json:"task_metrics"`
	TeamMetrics    TeamMetrics                 `json:"team_metrics"`
	Charts         *DashboardCharts            `json:"charts,omitempty"`
	RecentActivity []ActivityItem              `json:"recent_activity"`
	Widgets        []models.DashboardWidget    `json:"widgets"`
	LastUpdated    time.Time                   `json:"last_updated"`
}

// DashboardConfig 仪表盘配置
type DashboardConfig struct {
	ID          uuid.UUID           `json:"id"`
	ProjectID   uuid.UUID          `json:"project_id"`
	Layout      DashboardLayout    `json:"layout"`
	Widgets     []models.DashboardWidget  `json:"widgets"`
	Settings    map[string]interface{} `json:"settings"`
	CreatedBy   uuid.UUID          `json:"created_by"`
	UpdatedBy   uuid.UUID          `json:"updated_by"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// DashboardLayout 仪表盘布局
type DashboardLayout struct {
	Type        string                 `json:"type"`         // grid, flex
	Columns     int                    `json:"columns"`      // 网格列数
	Gap         int                    `json:"gap"`          // 间距
	Properties  map[string]interface{} `json:"properties"`   // 其他布局属性
}

// DateRange 日期范围
type DateRange struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Preset    string    `json:"preset,omitempty"` // last_7_days, last_30_days, last_3_months
}

// ProjectOverview 项目概览
type ProjectOverview struct {
	ProjectID        uuid.UUID `json:"project_id"`
	ProjectName      string    `json:"project_name"`
	TotalTasks       int64     `json:"total_tasks"`
	CompletedTasks   int64     `json:"completed_tasks"`
	InProgressTasks  int64     `json:"in_progress_tasks"`
	ActiveSprints    int64     `json:"active_sprints"`
	TeamMembers      int64     `json:"team_members"`
	TotalMembers     int64     `json:"total_members"`
	CompletionRate   float64   `json:"completion_rate"`
	BurndownRate     float64   `json:"burndown_rate"`
	VelocityAverage  float64   `json:"velocity_average"`
	VelocityTrend    string    `json:"velocity_trend"`
	TotalStoryPoints int64     `json:"total_story_points"`
	HealthScore      float64   `json:"health_score"`
	LastActivityAt   time.Time `json:"last_activity_at"`
}

// TeamMetrics 团队指标
type TeamMetrics struct {
	ProjectID       uuid.UUID          `json:"project_id"`
	DateRange       DateRange         `json:"date_range"`
	MemberMetrics   []MemberMetrics   `json:"member_metrics"`
	TotalMembers    int64             `json:"total_members"`
	ActiveMembers   int64             `json:"active_members"`
	TeamVelocity    float64           `json:"team_velocity"`
	TeamEfficiency  float64           `json:"team_efficiency"`
	CollaborationScore float64        `json:"collaboration_score"`
	WorkloadBalance    float64        `json:"workload_balance"`
	Velocity        float64           `json:"velocity"`
	Workload        map[string]MemberWorkload `json:"workload"`
	Productivity    []ProductivityMetric `json:"productivity"`
	CalculatedAt    time.Time         `json:"calculated_at"`
}

// MemberMetrics 成员指标
type MemberMetrics struct {
	UserID             uuid.UUID `json:"user_id"`
	UserName           string    `json:"user_name"`
	TasksCompleted     int64     `json:"tasks_completed"`
	StoryPointsDelivered int64   `json:"story_points_delivered"`
	AverageLeadTime    float64   `json:"average_lead_time"`
	CodeReviews        int64     `json:"code_reviews"`
	PullRequestsCreated int64    `json:"pull_requests_created"`
	BugsFound          int64     `json:"bugs_found"`
	ProductivityScore  float64   `json:"productivity_score"`
}

// ActivityItem 活动项目
type ActivityItem struct {
	ID          uuid.UUID   `json:"id"`
	Type        string      `json:"type"`         // task_completed, sprint_started, etc.
	Title       string      `json:"title"`
	Description string      `json:"description"`
	UserID      uuid.UUID   `json:"user_id"`
	UserName    string      `json:"user_name"`
	ProjectID   uuid.UUID   `json:"project_id"`
	EntityID    *uuid.UUID  `json:"entity_id,omitempty"` // 关联实体ID
	Timestamp   time.Time   `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// 额外的缺失类型定义

// UpdateDORAMetricsRequest 更新DORA指标请求
type UpdateDORAMetricsRequest struct {
	ProjectID             uuid.UUID `json:"project_id" binding:"required"`
	Period                time.Time `json:"period" binding:"required"`
	DeploymentFrequency   *float64  `json:"deployment_frequency,omitempty"`
	LeadTimeForChanges    *float64  `json:"lead_time_for_changes,omitempty"`
	TimeToRestoreService  *float64  `json:"time_to_restore_service,omitempty"`
	ChangeFailureRate     *float64  `json:"change_failure_rate,omitempty"`
	DeploymentCount       *int      `json:"deployment_count,omitempty"`
	IncidentCount         *int      `json:"incident_count,omitempty"`
	FailedDeploymentCount *int      `json:"failed_deployment_count,omitempty"`
}

// CumulativeFlowData 累积流图数据
type CumulativeFlowData struct {
	ProjectID   uuid.UUID              `json:"project_id"`
	StartDate   time.Time              `json:"start_date"`
	EndDate     time.Time              `json:"end_date"`
	DataPoints  []CumulativeFlowPoint  `json:"data_points"`
	Categories  []string               `json:"categories"`
}

// CumulativeFlowPoint 累积流图数据点
type CumulativeFlowPoint struct {
	Date   time.Time         `json:"date"`
	Values map[string]int64  `json:"values"` // status -> count
}

// CycleTimeChartData 周期时间图数据
type CycleTimeChartData struct {
	ProjectID   uuid.UUID            `json:"project_id"`
	StartDate   time.Time            `json:"start_date"`
	EndDate     time.Time            `json:"end_date"`
	DataPoints  []CycleTimePoint     `json:"data_points"`
	Statistics  CycleTimeStatistics  `json:"statistics"`
}

// CycleTimePoint 周期时间数据点
type CycleTimePoint struct {
	TaskID        uuid.UUID `json:"task_id"`
	TaskTitle     string    `json:"task_title"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	CycleTime     float64   `json:"cycle_time"`     // 小时数
	StageBreakdown map[string]float64 `json:"stage_breakdown"` // stage -> hours
}

// CycleTimeStatistics 周期时间统计
type CycleTimeStatistics struct {
	AverageCycleTime  float64 `json:"average_cycle_time"`
	MedianCycleTime   float64 `json:"median_cycle_time"`
	P50CycleTime      float64 `json:"p50_cycle_time"`
	P95CycleTime      float64 `json:"p95_cycle_time"`
	StandardDeviation float64 `json:"standard_deviation"`
	MinCycleTime      float64 `json:"min_cycle_time"`
	MaxCycleTime      float64 `json:"max_cycle_time"`
}

// 额外的缺失类型定义

// CurrentSprintInfo 当前冲刺信息
type CurrentSprintInfo struct {
	SprintID     *uuid.UUID `json:"sprint_id,omitempty"`
	SprintName   string     `json:"sprint_name"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	TotalTasks   int64      `json:"total_tasks"`
	CompletedTasks int64    `json:"completed_tasks"`
	RemainingDays  int      `json:"remaining_days"`
	CompletionRate float64  `json:"completion_rate"`
	BurndownRate   float64  `json:"burndown_rate"`
}

// TaskMetrics 任务指标
type TaskMetrics struct {
	TotalTasks       int64                    `json:"total_tasks"`
	CompletedTasks   int64                    `json:"completed_tasks"`
	InProgressTasks  int64                    `json:"in_progress_tasks"`
	PendingTasks     int64                    `json:"pending_tasks"`
	OverdueTasks     int64                    `json:"overdue_tasks"`
	TasksByStatus    map[string]int64         `json:"tasks_by_status"`
	TasksByPriority  map[string]int64         `json:"tasks_by_priority"`
	TasksByType      map[string]int64         `json:"tasks_by_type"`
	CompletionRate   float64                  `json:"completion_rate"`
	AverageCycleTime float64                  `json:"average_cycle_time"`
}

// DashboardCharts 仪表盘图表数据
type DashboardCharts struct {
	BurndownChart      *BurndownChartData      `json:"burndown_chart,omitempty"`
	VelocityChart      *VelocityChartData      `json:"velocity_chart,omitempty"`
	CumulativeFlowChart *CumulativeFlowData    `json:"cumulative_flow_chart,omitempty"`
	CycleTimeChart     *CycleTimeChartData     `json:"cycle_time_chart,omitempty"`
	TaskDistribution   *TaskDistributionChart  `json:"task_distribution,omitempty"`
	CodeQualityTrend   *CodeQualityData        `json:"code_quality_trend,omitempty"`
}

// CodeQualityData 代码质量数据
type CodeQualityData struct {
	TestCoverage   []QualityDataPoint `json:"test_coverage"`
	CodeReviewRate []QualityDataPoint `json:"code_review_rate"`
	DefectDensity  []QualityDataPoint `json:"defect_density"`
	TechnicalDebt  []QualityDataPoint `json:"technical_debt"`
}

// QualityDataPoint 质量数据点
type QualityDataPoint struct {
	Date  time.Time `json:"date"`
	Value float64   `json:"value"`
	Trend string    `json:"trend"` // up, down, stable
}

// BurndownChartData 燃尽图数据
type BurndownChartData struct {
	SprintID     uuid.UUID              `json:"sprint_id"`
	SprintName   string                 `json:"sprint_name"`
	StartDate    time.Time              `json:"start_date"`
	EndDate      time.Time              `json:"end_date"`
	TotalPoints  int64                  `json:"total_points"`
	DataPoints   []BurndownDataPoint    `json:"data_points"`
	IdealLine    []BurndownDataPoint    `json:"ideal_line"`
	IsCompleted  bool                   `json:"is_completed"`
}

// VelocityChartData 速度图数据
type VelocityChartData struct {
	ProjectID       uuid.UUID            `json:"project_id"`
	RecentSprints   []VelocityDataPoint  `json:"recent_sprints"`
	AverageVelocity float64              `json:"average_velocity"`
	Trend           string               `json:"trend"` // increasing, decreasing, stable
	PredictedNext   float64              `json:"predicted_next"`
}

// TaskDistributionChart 任务分布图数据
type TaskDistributionChart struct {
	ByStatus    []DistributionItem `json:"by_status"`
	ByPriority  []DistributionItem `json:"by_priority"`
	ByType      []DistributionItem `json:"by_type"`
	ByAssignee  []DistributionItem `json:"by_assignee"`
}

// DistributionItem 分布项目
type DistributionItem struct {
	Label string  `json:"label"`
	Value int64   `json:"value"`
	Color string  `json:"color,omitempty"`
	Percentage float64 `json:"percentage"`
}

// LeadTimeMetric 交付周期指标
type LeadTimeMetric struct {
	AverageLeadTime  float64              `json:"average_lead_time"`  // 小时
	MedianLeadTime   float64              `json:"median_lead_time"`   // 小时
	P95LeadTime      float64              `json:"p95_lead_time"`      // 小时
	LeadTimeRange    LeadTimeRange        `json:"lead_time_range"`
	RecentTrend      string               `json:"recent_trend"`       // improving, stable, declining
	DataPoints       []LeadTimeDataPoint  `json:"data_points"`
}

// LeadTimeRange 交付周期范围
type LeadTimeRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// LeadTimeDataPoint 交付周期数据点
type LeadTimeDataPoint struct {
	Date     time.Time `json:"date"`
	TaskID   uuid.UUID `json:"task_id"`
	TaskType string    `json:"task_type"`
	LeadTime float64   `json:"lead_time"` // 小时
}

// MemberWorkload 成员工作负载
type MemberWorkload struct {
	UserID             uuid.UUID `json:"user_id"`
	UserName           string    `json:"user_name"`
	TotalTasks         int64     `json:"total_tasks"`
	InProgressTasks    int64     `json:"in_progress_tasks"`
	CompletedTasks     int64     `json:"completed_tasks"`
	StoryPoints        int64     `json:"story_points"`
	EstimatedHours     float64   `json:"estimated_hours"`
	LoggedHours        float64   `json:"logged_hours"`
	UtilizationRate    float64   `json:"utilization_rate"`   // 利用率百分比
	OverloadStatus     string    `json:"overload_status"`    // normal, high, overloaded
}

// ProductivityMetric 生产力指标
type ProductivityMetric struct {
	UserID              uuid.UUID `json:"user_id"`
	UserName            string    `json:"user_name"`
	TasksCompleted      int64     `json:"tasks_completed"`
	StoryPointsDelivered int64    `json:"story_points_delivered"`
	AverageTaskTime     float64   `json:"average_task_time"`    // 小时
	QualityScore        float64   `json:"quality_score"`        // 0-100
	CollaborationScore  float64   `json:"collaboration_score"`  // 0-100
	OverallScore        float64   `json:"overall_score"`        // 0-100
}

// SprintTaskBreakdown Sprint任务分解
type SprintTaskBreakdown struct {
	Status       string `json:"status"`
	TaskCount    int64  `json:"task_count"`
	StoryPoints  int64  `json:"story_points"`
	Percentage   float64 `json:"percentage"`
}
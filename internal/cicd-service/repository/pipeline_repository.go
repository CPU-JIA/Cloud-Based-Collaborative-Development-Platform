package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PipelineRepository 流水线仓库接口
type PipelineRepository interface {
	// 流水线管理
	CreatePipeline(ctx context.Context, pipeline *models.Pipeline) error
	GetPipelineByID(ctx context.Context, id uuid.UUID) (*models.Pipeline, error)
	GetPipelinesByRepository(ctx context.Context, repositoryID uuid.UUID, page, pageSize int) ([]models.Pipeline, int64, error)
	UpdatePipeline(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	DeletePipeline(ctx context.Context, id uuid.UUID) error
	ListPipelines(ctx context.Context, page, pageSize int) ([]models.Pipeline, int64, error)

	// 流水线运行管理
	CreatePipelineRun(ctx context.Context, run *models.PipelineRun) error
	GetPipelineRunByID(ctx context.Context, id uuid.UUID) (*models.PipelineRun, error)
	GetPipelineRunsByPipeline(ctx context.Context, pipelineID uuid.UUID, page, pageSize int) ([]models.PipelineRun, int64, error)
	UpdatePipelineRun(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	CancelPipelineRun(ctx context.Context, id uuid.UUID) error
	GetRunningPipelineRuns(ctx context.Context) ([]models.PipelineRun, error)

	// 作业管理
	CreateJob(ctx context.Context, job *models.Job) error
	GetJobByID(ctx context.Context, id uuid.UUID) (*models.Job, error)
	GetJobsByPipelineRun(ctx context.Context, pipelineRunID uuid.UUID) ([]models.Job, error)
	UpdateJob(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	GetPendingJobs(ctx context.Context, runnerTags []string) ([]models.Job, error)

	// 执行器管理
	RegisterRunner(ctx context.Context, runner *models.Runner) error
	GetRunnerByID(ctx context.Context, id uuid.UUID) (*models.Runner, error)
	UpdateRunner(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	UnregisterRunner(ctx context.Context, id uuid.UUID) error
	GetAvailableRunners(ctx context.Context, tags []string) ([]models.Runner, error)
	UpdateRunnerStatus(ctx context.Context, id uuid.UUID, status models.RunnerStatus) error
	ListRunners(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]models.Runner, int64, error)

	// 统计查询
	GetPipelineStats(ctx context.Context, pipelineID uuid.UUID, days int) (*models.PipelineStats, error)
	GetRunnerStats(ctx context.Context, runnerID uuid.UUID) (*models.RunnerStats, error)
}

// pipelineRepository 流水线仓库实现
type pipelineRepository struct {
	db *gorm.DB
}

// NewPipelineRepository 创建流水线仓库实例
func NewPipelineRepository(db *gorm.DB) PipelineRepository {
	return &pipelineRepository{
		db: db,
	}
}

// 流水线管理实现

// CreatePipeline 创建流水线
func (r *pipelineRepository) CreatePipeline(ctx context.Context, pipeline *models.Pipeline) error {
	return r.db.WithContext(ctx).Create(pipeline).Error
}

// GetPipelineByID 根据ID获取流水线
func (r *pipelineRepository) GetPipelineByID(ctx context.Context, id uuid.UUID) (*models.Pipeline, error) {
	var pipeline models.Pipeline
	err := r.db.WithContext(ctx).
		Preload("Repository").
		First(&pipeline, "id = ? AND deleted_at IS NULL", id).Error
	if err != nil {
		return nil, err
	}
	return &pipeline, nil
}

// GetPipelinesByRepository 根据仓库ID获取流水线列表
func (r *pipelineRepository) GetPipelinesByRepository(ctx context.Context, repositoryID uuid.UUID, page, pageSize int) ([]models.Pipeline, int64, error) {
	var pipelines []models.Pipeline
	var total int64

	db := r.db.WithContext(ctx).
		Where("repository_id = ? AND deleted_at IS NULL", repositoryID)

	// 获取总数
	if err := db.Model(&models.Pipeline{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := db.Preload("Repository").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&pipelines).Error

	return pipelines, total, err
}

// UpdatePipeline 更新流水线
func (r *pipelineRepository) UpdatePipeline(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&models.Pipeline{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates).Error
}

// DeletePipeline 软删除流水线
func (r *pipelineRepository) DeletePipeline(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.Pipeline{}).
		Where("id = ?", id).
		Update("deleted_at", time.Now().UTC()).Error
}

// ListPipelines 获取流水线列表
func (r *pipelineRepository) ListPipelines(ctx context.Context, page, pageSize int) ([]models.Pipeline, int64, error) {
	var pipelines []models.Pipeline
	var total int64

	db := r.db.WithContext(ctx).
		Where("deleted_at IS NULL")

	// 获取总数
	if err := db.Model(&models.Pipeline{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := db.Preload("Repository").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&pipelines).Error

	return pipelines, total, err
}

// 流水线运行管理实现

// CreatePipelineRun 创建流水线运行
func (r *pipelineRepository) CreatePipelineRun(ctx context.Context, run *models.PipelineRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

// GetPipelineRunByID 根据ID获取流水线运行
func (r *pipelineRepository) GetPipelineRunByID(ctx context.Context, id uuid.UUID) (*models.PipelineRun, error) {
	var run models.PipelineRun
	err := r.db.WithContext(ctx).
		Preload("Pipeline").
		Preload("TriggerUser").
		Preload("Jobs").
		First(&run, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// GetPipelineRunsByPipeline 根据流水线ID获取运行列表
func (r *pipelineRepository) GetPipelineRunsByPipeline(ctx context.Context, pipelineID uuid.UUID, page, pageSize int) ([]models.PipelineRun, int64, error) {
	var runs []models.PipelineRun
	var total int64

	db := r.db.WithContext(ctx).
		Where("pipeline_id = ?", pipelineID)

	// 获取总数
	if err := db.Model(&models.PipelineRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := db.Preload("Pipeline").
		Preload("TriggerUser").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&runs).Error

	return runs, total, err
}

// UpdatePipelineRun 更新流水线运行
func (r *pipelineRepository) UpdatePipelineRun(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&models.PipelineRun{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// CancelPipelineRun 取消流水线运行
func (r *pipelineRepository) CancelPipelineRun(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":      models.PipelineStatusCancelled,
		"finished_at": now,
	}
	return r.UpdatePipelineRun(ctx, id, updates)
}

// GetRunningPipelineRuns 获取正在运行的流水线
func (r *pipelineRepository) GetRunningPipelineRuns(ctx context.Context) ([]models.PipelineRun, error) {
	var runs []models.PipelineRun
	err := r.db.WithContext(ctx).
		Where("status IN (?)", []string{string(models.PipelineStatusPending), string(models.PipelineStatusRunning)}).
		Preload("Pipeline").
		Find(&runs).Error
	return runs, err
}

// 作业管理实现

// CreateJob 创建作业
func (r *pipelineRepository) CreateJob(ctx context.Context, job *models.Job) error {
	return r.db.WithContext(ctx).Create(job).Error
}

// GetJobByID 根据ID获取作业
func (r *pipelineRepository) GetJobByID(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	var job models.Job
	err := r.db.WithContext(ctx).
		Preload("PipelineRun").
		Preload("Runner").
		First(&job, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// GetJobsByPipelineRun 根据流水线运行ID获取作业列表
func (r *pipelineRepository) GetJobsByPipelineRun(ctx context.Context, pipelineRunID uuid.UUID) ([]models.Job, error) {
	var jobs []models.Job
	err := r.db.WithContext(ctx).
		Where("pipeline_run_id = ?", pipelineRunID).
		Preload("Runner").
		Order("created_at ASC").
		Find(&jobs).Error
	return jobs, err
}

// UpdateJob 更新作业
func (r *pipelineRepository) UpdateJob(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&models.Job{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// GetPendingJobs 获取待执行的作业
func (r *pipelineRepository) GetPendingJobs(ctx context.Context, runnerTags []string) ([]models.Job, error) {
	var jobs []models.Job
	query := r.db.WithContext(ctx).
		Where("status = ?", models.JobStatusPending).
		Preload("PipelineRun").
		Order("created_at ASC")

	// 如果指定了标签，则进行标签过滤
	if len(runnerTags) > 0 {
		// 这里简化处理，实际应该根据作业需求的标签进行匹配
		query = query.Limit(10)
	}

	err := query.Find(&jobs).Error
	return jobs, err
}

// 执行器管理实现

// RegisterRunner 注册执行器
func (r *pipelineRepository) RegisterRunner(ctx context.Context, runner *models.Runner) error {
	return r.db.WithContext(ctx).Create(runner).Error
}

// GetRunnerByID 根据ID获取执行器
func (r *pipelineRepository) GetRunnerByID(ctx context.Context, id uuid.UUID) (*models.Runner, error) {
	var runner models.Runner
	err := r.db.WithContext(ctx).First(&runner, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &runner, nil
}

// UpdateRunner 更新执行器
func (r *pipelineRepository) UpdateRunner(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now().UTC()
	updates["last_contact_at"] = time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&models.Runner{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UnregisterRunner 注销执行器
func (r *pipelineRepository) UnregisterRunner(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Runner{}, "id = ?", id).Error
}

// GetAvailableRunners 获取可用的执行器
func (r *pipelineRepository) GetAvailableRunners(ctx context.Context, tags []string) ([]models.Runner, error) {
	var runners []models.Runner
	query := r.db.WithContext(ctx).
		Where("status IN (?)", []string{string(models.RunnerStatusOnline), string(models.RunnerStatusIdle)})

	// 标签匹配逻辑（简化版本）
	if len(tags) > 0 {
		// 使用 PostgreSQL 的 jsonb 操作符进行标签匹配
		for _, tag := range tags {
			query = query.Where("tags::jsonb ? ?", tag)
		}
	}

	err := query.Find(&runners).Error
	return runners, err
}

// UpdateRunnerStatus 更新执行器状态
func (r *pipelineRepository) UpdateRunnerStatus(ctx context.Context, id uuid.UUID, status models.RunnerStatus) error {
	updates := map[string]interface{}{
		"status":           status,
		"last_contact_at":  time.Now().UTC(),
		"updated_at":       time.Now().UTC(),
	}
	return r.UpdateRunner(ctx, id, updates)
}

// ListRunners 获取执行器列表
func (r *pipelineRepository) ListRunners(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]models.Runner, int64, error) {
	var runners []models.Runner
	var total int64

	db := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID)

	// 获取总数
	if err := db.Model(&models.Runner{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := db.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&runners).Error

	return runners, total, err
}

// 统计查询实现

// GetPipelineStats 获取流水线统计信息
func (r *pipelineRepository) GetPipelineStats(ctx context.Context, pipelineID uuid.UUID, days int) (*models.PipelineStats, error) {
	var stats models.PipelineStats
	
	// 统计最近N天的数据
	startDate := time.Now().UTC().AddDate(0, 0, -days)
	
	// 总运行次数
	err := r.db.WithContext(ctx).
		Model(&models.PipelineRun{}).
		Where("pipeline_id = ? AND created_at >= ?", pipelineID, startDate).
		Count(&stats.TotalRuns).Error
	if err != nil {
		return nil, err
	}

	// 成功运行次数
	err = r.db.WithContext(ctx).
		Model(&models.PipelineRun{}).
		Where("pipeline_id = ? AND status = ? AND created_at >= ?", pipelineID, models.PipelineStatusSuccess, startDate).
		Count(&stats.SuccessfulRuns).Error
	if err != nil {
		return nil, err
	}

	// 失败运行次数
	err = r.db.WithContext(ctx).
		Model(&models.PipelineRun{}).
		Where("pipeline_id = ? AND status = ? AND created_at >= ?", pipelineID, models.PipelineStatusFailed, startDate).
		Count(&stats.FailedRuns).Error
	if err != nil {
		return nil, err
	}

	// 平均执行时间
	var avgDuration sql.NullFloat64
	err = r.db.WithContext(ctx).
		Model(&models.PipelineRun{}).
		Where("pipeline_id = ? AND status = ? AND duration IS NOT NULL AND created_at >= ?", pipelineID, models.PipelineStatusSuccess, startDate).
		Select("AVG(duration)").
		Scan(&avgDuration).Error
	if err == nil && avgDuration.Valid {
		stats.AverageDuration = int64(avgDuration.Float64)
	}

	// 计算成功率
	if stats.TotalRuns > 0 {
		stats.SuccessRate = float64(stats.SuccessfulRuns) / float64(stats.TotalRuns) * 100
	}

	return &stats, nil
}

// GetRunnerStats 获取执行器统计信息
func (r *pipelineRepository) GetRunnerStats(ctx context.Context, runnerID uuid.UUID) (*models.RunnerStats, error) {
	var stats models.RunnerStats
	
	// 执行的作业总数
	err := r.db.WithContext(ctx).
		Model(&models.Job{}).
		Where("runner_id = ?", runnerID).
		Count(&stats.TotalJobs).Error
	if err != nil {
		return nil, err
	}

	// 成功的作业数
	err = r.db.WithContext(ctx).
		Model(&models.Job{}).
		Where("runner_id = ? AND status = ?", runnerID, models.JobStatusSuccess).
		Count(&stats.SuccessfulJobs).Error
	if err != nil {
		return nil, err
	}

	// 失败的作业数
	err = r.db.WithContext(ctx).
		Model(&models.Job{}).
		Where("runner_id = ? AND status = ?", runnerID, models.JobStatusFailed).
		Count(&stats.FailedJobs).Error
	if err != nil {
		return nil, err
	}

	// 平均执行时间
	var avgDuration sql.NullFloat64
	err = r.db.WithContext(ctx).
		Model(&models.Job{}).
		Where("runner_id = ? AND status = ? AND duration IS NOT NULL", runnerID, models.JobStatusSuccess).
		Select("AVG(duration)").
		Scan(&avgDuration).Error
	if err == nil && avgDuration.Valid {
		stats.AverageJobDuration = int64(avgDuration.Float64)
	}

	// 计算成功率
	if stats.TotalJobs > 0 {
		stats.SuccessRate = float64(stats.SuccessfulJobs) / float64(stats.TotalJobs) * 100
	}

	return &stats, nil
}
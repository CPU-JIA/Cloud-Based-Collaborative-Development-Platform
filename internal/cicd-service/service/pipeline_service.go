package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/engine"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/storage"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PipelineService 流水线服务接口
type PipelineService interface {
	// 流水线管理
	CreatePipeline(ctx context.Context, req *models.CreatePipelineRequest, userID uuid.UUID) (*models.Pipeline, error)
	GetPipeline(ctx context.Context, id uuid.UUID) (*models.Pipeline, error)
	GetPipelinesByRepository(ctx context.Context, repositoryID uuid.UUID, page, pageSize int) (*models.PipelineListResponse, error)
	UpdatePipeline(ctx context.Context, id uuid.UUID, req *models.UpdatePipelineRequest, userID uuid.UUID) (*models.Pipeline, error)
	DeletePipeline(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ListPipelines(ctx context.Context, page, pageSize int) (*models.PipelineListResponse, error)

	// 流水线运行管理
	TriggerPipeline(ctx context.Context, pipelineID uuid.UUID, req *models.TriggerPipelineRequest, userID uuid.UUID) (*models.PipelineRun, error)
	GetPipelineRun(ctx context.Context, id uuid.UUID) (*models.PipelineRun, error)
	GetPipelineRunsByPipeline(ctx context.Context, pipelineID uuid.UUID, page, pageSize int) (*models.PipelineRunListResponse, error)
	CancelPipelineRun(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	RetryPipelineRun(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.PipelineRun, error)

	// 作业管理
	GetJob(ctx context.Context, id uuid.UUID) (*models.Job, error)
	GetJobsByPipelineRun(ctx context.Context, pipelineRunID uuid.UUID) ([]models.Job, error)
	UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status models.JobStatus, runnerID uuid.UUID) error
	UpdateJobOutput(ctx context.Context, jobID uuid.UUID, output string, exitCode *int) error

	// 执行器管理
	RegisterRunner(ctx context.Context, req *models.RegisterRunnerRequest, tenantID uuid.UUID) (*models.Runner, error)
	GetRunner(ctx context.Context, id uuid.UUID) (*models.Runner, error)
	UpdateRunner(ctx context.Context, id uuid.UUID, req *models.UpdateRunnerRequest) (*models.Runner, error)
	UnregisterRunner(ctx context.Context, id uuid.UUID) error
	HeartbeatRunner(ctx context.Context, runnerID uuid.UUID, status models.RunnerStatus) error
	GetAvailableRunners(ctx context.Context, tags []string) ([]models.Runner, error)
	ListRunners(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]models.Runner, int64, error)

	// 统计查询
	GetPipelineStats(ctx context.Context, pipelineID uuid.UUID, days int) (*models.PipelineStats, error)
	GetRunnerStats(ctx context.Context, runnerID uuid.UUID) (*models.RunnerStats, error)

	// 后台任务
	ProcessPendingJobs(ctx context.Context) error
	CleanupOldRuns(ctx context.Context, retentionDays int) error
}

// pipelineService 流水线服务实现
type pipelineService struct {
	repo    repository.PipelineRepository
	storage storage.StorageManager
	engine  engine.PipelineEngine
	logger  *zap.Logger
}

// NewPipelineService 创建流水线服务实例
func NewPipelineService(repo repository.PipelineRepository, storage storage.StorageManager, logger *zap.Logger) PipelineService {
	pipelineEngine := engine.NewPipelineEngine(repo, storage, logger)
	
	return &pipelineService{
		repo:    repo,
		storage: storage,
		engine:  pipelineEngine,
		logger:  logger,
	}
}

// 流水线管理实现

// CreatePipeline 创建流水线
func (s *pipelineService) CreatePipeline(ctx context.Context, req *models.CreatePipelineRequest, userID uuid.UUID) (*models.Pipeline, error) {
	// 验证请求参数
	if err := s.validateCreatePipelineRequest(req); err != nil {
		return nil, fmt.Errorf("请求参数验证失败: %w", err)
	}

	repositoryID, err := uuid.Parse(req.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("无效的仓库ID: %w", err)
	}

	// 创建流水线
	pipeline := &models.Pipeline{
		RepositoryID:       repositoryID,
		Name:               req.Name,
		DefinitionFilePath: req.DefinitionFilePath,
		Description:        req.Description,
		IsActive:           true,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	if err := s.repo.CreatePipeline(ctx, pipeline); err != nil {
		s.logger.Error("创建流水线失败", zap.Error(err), zap.String("name", req.Name))
		return nil, fmt.Errorf("创建流水线失败: %w", err)
	}

	s.logger.Info("流水线创建成功", zap.String("pipeline_id", pipeline.ID.String()), zap.String("name", pipeline.Name))
	return pipeline, nil
}

// GetPipeline 获取流水线详情
func (s *pipelineService) GetPipeline(ctx context.Context, id uuid.UUID) (*models.Pipeline, error) {
	pipeline, err := s.repo.GetPipelineByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrPipelineNotFound) {
			return nil, ErrPipelineNotFound
		}
		s.logger.Error("获取流水线失败", zap.Error(err), zap.String("pipeline_id", id.String()))
		return nil, fmt.Errorf("获取流水线失败: %w", err)
	}
	return pipeline, nil
}

// GetPipelinesByRepository 根据仓库获取流水线列表
func (s *pipelineService) GetPipelinesByRepository(ctx context.Context, repositoryID uuid.UUID, page, pageSize int) (*models.PipelineListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	pipelines, total, err := s.repo.GetPipelinesByRepository(ctx, repositoryID, page, pageSize)
	if err != nil {
		s.logger.Error("获取流水线列表失败", zap.Error(err), zap.String("repository_id", repositoryID.String()))
		return nil, fmt.Errorf("获取流水线列表失败: %w", err)
	}

	return &models.PipelineListResponse{
		Pipelines: pipelines,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
	}, nil
}

// UpdatePipeline 更新流水线
func (s *pipelineService) UpdatePipeline(ctx context.Context, id uuid.UUID, req *models.UpdatePipelineRequest, userID uuid.UUID) (*models.Pipeline, error) {
	// 验证流水线是否存在
	existingPipeline, err := s.repo.GetPipelineByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("流水线不存在: %w", err)
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		if err := s.validatePipelineName(*req.Name); err != nil {
			return nil, err
		}
		updates["name"] = *req.Name
	}
	if req.DefinitionFilePath != nil {
		if err := s.validateDefinitionFilePath(*req.DefinitionFilePath); err != nil {
			return nil, err
		}
		updates["definition_file_path"] = *req.DefinitionFilePath
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) == 0 {
		return existingPipeline, nil
	}

	// 执行更新
	if err := s.repo.UpdatePipeline(ctx, id, updates); err != nil {
		s.logger.Error("更新流水线失败", zap.Error(err), zap.String("pipeline_id", id.String()))
		return nil, fmt.Errorf("更新流水线失败: %w", err)
	}

	// 返回更新后的流水线
	updatedPipeline, err := s.repo.GetPipelineByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取更新后的流水线失败: %w", err)
	}

	s.logger.Info("流水线更新成功", zap.String("pipeline_id", id.String()))
	return updatedPipeline, nil
}

// DeletePipeline 删除流水线
func (s *pipelineService) DeletePipeline(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// 验证流水线是否存在
	_, err := s.repo.GetPipelineByID(ctx, id)
	if err != nil {
		return fmt.Errorf("流水线不存在: %w", err)
	}

	// 检查是否有正在运行的流水线
	runningRuns, err := s.repo.GetRunningPipelineRuns(ctx)
	if err != nil {
		return fmt.Errorf("检查运行状态失败: %w", err)
	}

	for _, run := range runningRuns {
		if run.PipelineID == id {
			return errors.New("无法删除有正在运行任务的流水线")
		}
	}

	// 执行软删除
	if err := s.repo.DeletePipeline(ctx, id); err != nil {
		s.logger.Error("删除流水线失败", zap.Error(err), zap.String("pipeline_id", id.String()))
		return fmt.Errorf("删除流水线失败: %w", err)
	}

	s.logger.Info("流水线删除成功", zap.String("pipeline_id", id.String()))
	return nil
}

// ListPipelines 获取流水线列表
func (s *pipelineService) ListPipelines(ctx context.Context, page, pageSize int) (*models.PipelineListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	pipelines, total, err := s.repo.ListPipelines(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("获取流水线列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取流水线列表失败: %w", err)
	}

	return &models.PipelineListResponse{
		Pipelines: pipelines,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
	}, nil
}

// 流水线运行管理实现

// TriggerPipeline 触发流水线
func (s *pipelineService) TriggerPipeline(ctx context.Context, pipelineID uuid.UUID, req *models.TriggerPipelineRequest, userID uuid.UUID) (*models.PipelineRun, error) {
	// 验证流水线是否存在且激活
	pipeline, err := s.repo.GetPipelineByID(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("流水线不存在: %w", err)
	}

	if !pipeline.IsActive {
		return nil, errors.New("流水线已禁用")
	}

	// 验证提交SHA
	if err := s.validateCommitSHA(req.CommitSHA); err != nil {
		return nil, err
	}

	// 创建流水线运行
	run := &models.PipelineRun{
		PipelineID:  pipelineID,
		TriggerType: models.TriggerTypeManual,
		TriggerBy:   &userID,
		CommitSHA:   req.CommitSHA,
		Branch:      req.Branch,
		Status:      models.PipelineStatusPending,
		Variables:   req.Variables,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.repo.CreatePipelineRun(ctx, run); err != nil {
		s.logger.Error("创建流水线运行失败", zap.Error(err), zap.String("pipeline_id", pipelineID.String()))
		return nil, fmt.Errorf("创建流水线运行失败: %w", err)
	}

	// 使用执行引擎开始执行流水线
	if err := s.engine.ExecutePipeline(ctx, run); err != nil {
		s.logger.Error("启动流水线执行失败", zap.Error(err), zap.String("run_id", run.ID.String()))
		// 更新运行状态为失败
		s.repo.UpdatePipelineRun(ctx, run.ID, map[string]interface{}{
			"status": models.PipelineStatusFailed,
		})
		return nil, fmt.Errorf("启动流水线执行失败: %w", err)
	}

	s.logger.Info("流水线触发成功", 
		zap.String("pipeline_id", pipelineID.String()),
		zap.String("run_id", run.ID.String()),
		zap.String("commit_sha", req.CommitSHA))

	return run, nil
}

// GetPipelineRun 获取流水线运行详情
func (s *pipelineService) GetPipelineRun(ctx context.Context, id uuid.UUID) (*models.PipelineRun, error) {
	run, err := s.repo.GetPipelineRunByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrPipelineRunNotFound) {
			return nil, ErrPipelineRunNotFound
		}
		s.logger.Error("获取流水线运行失败", zap.Error(err), zap.String("run_id", id.String()))
		return nil, fmt.Errorf("获取流水线运行失败: %w", err)
	}
	return run, nil
}

// GetPipelineRunsByPipeline 根据流水线获取运行列表
func (s *pipelineService) GetPipelineRunsByPipeline(ctx context.Context, pipelineID uuid.UUID, page, pageSize int) (*models.PipelineRunListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	runs, total, err := s.repo.GetPipelineRunsByPipeline(ctx, pipelineID, page, pageSize)
	if err != nil {
		s.logger.Error("获取流水线运行列表失败", zap.Error(err), zap.String("pipeline_id", pipelineID.String()))
		return nil, fmt.Errorf("获取流水线运行列表失败: %w", err)
	}

	return &models.PipelineRunListResponse{
		PipelineRuns: runs,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

// CancelPipelineRun 取消流水线运行
func (s *pipelineService) CancelPipelineRun(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// 获取流水线运行
	run, err := s.repo.GetPipelineRunByID(ctx, id)
	if err != nil {
		return fmt.Errorf("流水线运行不存在: %w", err)
	}

	// 检查是否可以取消
	if !run.CanCancel() {
		return errors.New("流水线运行无法取消")
	}

	// 使用执行引擎取消流水线
	if err := s.engine.CancelPipeline(ctx, id); err != nil {
		s.logger.Error("取消流水线运行失败", zap.Error(err), zap.String("run_id", id.String()))
		return fmt.Errorf("取消流水线运行失败: %w", err)
	}

	s.logger.Info("流水线运行已取消", zap.String("run_id", id.String()))
	return nil
}

// RetryPipelineRun 重试流水线运行
func (s *pipelineService) RetryPipelineRun(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.PipelineRun, error) {
	// 获取原始运行
	originalRun, err := s.repo.GetPipelineRunByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("流水线运行不存在: %w", err)
	}

	// 只有失败或取消的运行可以重试
	if originalRun.Status != models.PipelineStatusFailed && originalRun.Status != models.PipelineStatusCancelled {
		return nil, errors.New("只能重试失败或取消的流水线运行")
	}

	// 创建新的运行
	newRun := &models.PipelineRun{
		PipelineID:  originalRun.PipelineID,
		TriggerType: models.TriggerTypeManual,
		TriggerBy:   &userID,
		CommitSHA:   originalRun.CommitSHA,
		Branch:      originalRun.Branch,
		Status:      models.PipelineStatusPending,
		Variables:   originalRun.Variables,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.repo.CreatePipelineRun(ctx, newRun); err != nil {
		s.logger.Error("创建重试运行失败", zap.Error(err), zap.String("original_run_id", id.String()))
		return nil, fmt.Errorf("创建重试运行失败: %w", err)
	}

	s.logger.Info("流水线运行重试成功", 
		zap.String("original_run_id", id.String()),
		zap.String("new_run_id", newRun.ID.String()))

	return newRun, nil
}

// 验证函数

// validateCreatePipelineRequest 验证创建流水线请求
func (s *pipelineService) validateCreatePipelineRequest(req *models.CreatePipelineRequest) error {
	if err := s.validatePipelineName(req.Name); err != nil {
		return err
	}
	
	if err := s.validateDefinitionFilePath(req.DefinitionFilePath); err != nil {
		return err
	}

	if req.Description != nil && len(*req.Description) > 2000 {
		return errors.New("描述不能超过2000个字符")
	}

	return nil
}

// validatePipelineName 验证流水线名称
func (s *pipelineService) validatePipelineName(name string) error {
	if name == "" {
		return errors.New("流水线名称不能为空")
	}
	
	if len(name) > 255 {
		return errors.New("流水线名称不能超过255个字符")
	}

	// 名称只能包含字母、数字、连字符和下划线
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-_]+$`, name)
	if !matched {
		return errors.New("流水线名称只能包含字母、数字、连字符和下划线")
	}

	return nil
}

// validateDefinitionFilePath 验证定义文件路径
func (s *pipelineService) validateDefinitionFilePath(path string) error {
	if path == "" {
		return errors.New("定义文件路径不能为空")
	}
	
	if len(path) > 512 {
		return errors.New("定义文件路径不能超过512个字符")
	}

	// 简单验证路径格式
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-_/.]+$`, path)
	if !matched {
		return errors.New("定义文件路径格式无效")
	}

	return nil
}

// validateCommitSHA 验证提交SHA
func (s *pipelineService) validateCommitSHA(sha string) error {
	if len(sha) != 40 {
		return errors.New("提交SHA必须是40位字符")
	}

	matched, _ := regexp.MatchString(`^[a-fA-F0-9]+$`, sha)
	if !matched {
		return errors.New("提交SHA格式无效")
	}

	return nil
}

// 作业管理实现

// GetJob 获取作业详情
func (s *pipelineService) GetJob(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	job, err := s.repo.GetJobByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			return nil, ErrJobNotFound
		}
		s.logger.Error("获取作业失败", zap.Error(err), zap.String("job_id", id.String()))
		return nil, fmt.Errorf("获取作业失败: %w", err)
	}
	return job, nil
}

// GetJobsByPipelineRun 根据流水线运行获取作业列表
func (s *pipelineService) GetJobsByPipelineRun(ctx context.Context, pipelineRunID uuid.UUID) ([]models.Job, error) {
	jobs, err := s.repo.GetJobsByPipelineRun(ctx, pipelineRunID)
	if err != nil {
		s.logger.Error("获取作业列表失败", zap.Error(err), zap.String("pipeline_run_id", pipelineRunID.String()))
		return nil, fmt.Errorf("获取作业列表失败: %w", err)
	}
	return jobs, nil
}

// UpdateJobStatus 更新作业状态
func (s *pipelineService) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status models.JobStatus, runnerID uuid.UUID) error {
	updates := map[string]interface{}{
		"status": status,
	}

	// 根据状态设置时间戳
	now := time.Now().UTC()
	switch status {
	case models.JobStatusRunning:
		updates["started_at"] = now
		updates["runner_id"] = runnerID
	case models.JobStatusSuccess, models.JobStatusFailed, models.JobStatusCancelled:
		updates["finished_at"] = now
		// 计算持续时间
		job, err := s.repo.GetJobByID(ctx, jobID)
		if err == nil && job.StartedAt != nil {
			duration := now.Sub(*job.StartedAt)
			durationSeconds := int64(duration.Seconds())
			updates["duration"] = durationSeconds
		}
	}

	if err := s.repo.UpdateJob(ctx, jobID, updates); err != nil {
		s.logger.Error("更新作业状态失败", zap.Error(err), zap.String("job_id", jobID.String()))
		return fmt.Errorf("更新作业状态失败: %w", err)
	}

	s.logger.Info("作业状态更新成功", 
		zap.String("job_id", jobID.String()),
		zap.String("status", string(status)))

	return nil
}

// UpdateJobOutput 更新作业输出
func (s *pipelineService) UpdateJobOutput(ctx context.Context, jobID uuid.UUID, output string, exitCode *int) error {
	updates := map[string]interface{}{
		"log_output": output,
	}

	if exitCode != nil {
		updates["exit_code"] = *exitCode
	}

	if err := s.repo.UpdateJob(ctx, jobID, updates); err != nil {
		s.logger.Error("更新作业输出失败", zap.Error(err), zap.String("job_id", jobID.String()))
		return fmt.Errorf("更新作业输出失败: %w", err)
	}

	return nil
}

// 执行器管理实现

// RegisterRunner 注册执行器
func (s *pipelineService) RegisterRunner(ctx context.Context, req *models.RegisterRunnerRequest, tenantID uuid.UUID) (*models.Runner, error) {
	// 验证请求参数
	if err := s.validateRegisterRunnerRequest(req); err != nil {
		return nil, fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 创建执行器
	runner := &models.Runner{
		TenantID:      tenantID,
		Name:          req.Name,
		Description:   req.Description,
		Tags:          req.Tags,
		Status:        models.RunnerStatusOnline,
		Version:       req.Version,
		OS:            req.OS,
		Architecture:  req.Architecture,
		LastContactAt: &time.Time{},
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := s.repo.RegisterRunner(ctx, runner); err != nil {
		s.logger.Error("注册执行器失败", zap.Error(err), zap.String("name", req.Name))
		return nil, fmt.Errorf("注册执行器失败: %w", err)
	}

	s.logger.Info("执行器注册成功", zap.String("runner_id", runner.ID.String()), zap.String("name", runner.Name))
	return runner, nil
}

// GetRunner 获取执行器详情
func (s *pipelineService) GetRunner(ctx context.Context, id uuid.UUID) (*models.Runner, error) {
	runner, err := s.repo.GetRunnerByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrRunnerNotFound) {
			return nil, ErrRunnerNotFound
		}
		s.logger.Error("获取执行器失败", zap.Error(err), zap.String("runner_id", id.String()))
		return nil, fmt.Errorf("获取执行器失败: %w", err)
	}
	return runner, nil
}

// UpdateRunner 更新执行器
func (s *pipelineService) UpdateRunner(ctx context.Context, id uuid.UUID, req *models.UpdateRunnerRequest) (*models.Runner, error) {
	// 验证执行器是否存在
	existingRunner, err := s.repo.GetRunnerByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("执行器不存在: %w", err)
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		if err := s.validateRunnerName(*req.Name); err != nil {
			return nil, err
		}
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}
	if req.Status != nil {
		status := models.RunnerStatus(*req.Status)
		if err := s.validateRunnerStatus(status); err != nil {
			return nil, err
		}
		updates["status"] = status
	}
	if req.Version != nil {
		updates["version"] = *req.Version
	}
	if req.OS != nil {
		updates["os"] = *req.OS
	}
	if req.Architecture != nil {
		updates["architecture"] = *req.Architecture
	}

	if len(updates) == 0 {
		return existingRunner, nil
	}

	// 执行更新
	if err := s.repo.UpdateRunner(ctx, id, updates); err != nil {
		s.logger.Error("更新执行器失败", zap.Error(err), zap.String("runner_id", id.String()))
		return nil, fmt.Errorf("更新执行器失败: %w", err)
	}

	// 返回更新后的执行器
	updatedRunner, err := s.repo.GetRunnerByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取更新后的执行器失败: %w", err)
	}

	s.logger.Info("执行器更新成功", zap.String("runner_id", id.String()))
	return updatedRunner, nil
}

// UnregisterRunner 注销执行器
func (s *pipelineService) UnregisterRunner(ctx context.Context, id uuid.UUID) error {
	// 验证执行器是否存在
	_, err := s.repo.GetRunnerByID(ctx, id)
	if err != nil {
		return fmt.Errorf("执行器不存在: %w", err)
	}

	// 执行注销
	if err := s.repo.UnregisterRunner(ctx, id); err != nil {
		s.logger.Error("注销执行器失败", zap.Error(err), zap.String("runner_id", id.String()))
		return fmt.Errorf("注销执行器失败: %w", err)
	}

	s.logger.Info("执行器注销成功", zap.String("runner_id", id.String()))
	return nil
}

// HeartbeatRunner 执行器心跳
func (s *pipelineService) HeartbeatRunner(ctx context.Context, runnerID uuid.UUID, status models.RunnerStatus) error {
	if err := s.validateRunnerStatus(status); err != nil {
		return err
	}

	if err := s.repo.UpdateRunnerStatus(ctx, runnerID, status); err != nil {
		s.logger.Error("更新执行器心跳失败", zap.Error(err), zap.String("runner_id", runnerID.String()))
		return fmt.Errorf("更新执行器心跳失败: %w", err)
	}

	return nil
}

// GetAvailableRunners 获取可用执行器
func (s *pipelineService) GetAvailableRunners(ctx context.Context, tags []string) ([]models.Runner, error) {
	runners, err := s.repo.GetAvailableRunners(ctx, tags)
	if err != nil {
		s.logger.Error("获取可用执行器失败", zap.Error(err))
		return nil, fmt.Errorf("获取可用执行器失败: %w", err)
	}
	return runners, nil
}

// ListRunners 获取执行器列表
func (s *pipelineService) ListRunners(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]models.Runner, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	runners, total, err := s.repo.ListRunners(ctx, tenantID, page, pageSize)
	if err != nil {
		s.logger.Error("获取执行器列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取执行器列表失败: %w", err)
	}

	return runners, total, nil
}

// 统计查询实现

// GetPipelineStats 获取流水线统计信息
func (s *pipelineService) GetPipelineStats(ctx context.Context, pipelineID uuid.UUID, days int) (*models.PipelineStats, error) {
	if days <= 0 || days > 365 {
		days = 30 // 默认30天
	}

	stats, err := s.repo.GetPipelineStats(ctx, pipelineID, days)
	if err != nil {
		s.logger.Error("获取流水线统计失败", zap.Error(err), zap.String("pipeline_id", pipelineID.String()))
		return nil, fmt.Errorf("获取流水线统计失败: %w", err)
	}

	return stats, nil
}

// GetRunnerStats 获取执行器统计信息
func (s *pipelineService) GetRunnerStats(ctx context.Context, runnerID uuid.UUID) (*models.RunnerStats, error) {
	stats, err := s.repo.GetRunnerStats(ctx, runnerID)
	if err != nil {
		s.logger.Error("获取执行器统计失败", zap.Error(err), zap.String("runner_id", runnerID.String()))
		return nil, fmt.Errorf("获取执行器统计失败: %w", err)
	}

	return stats, nil
}

// 后台任务实现

// ProcessPendingJobs 处理待执行作业
func (s *pipelineService) ProcessPendingJobs(ctx context.Context) error {
	// 获取待执行作业
	jobs, err := s.repo.GetPendingJobs(ctx, nil)
	if err != nil {
		return fmt.Errorf("获取待执行作业失败: %w", err)
	}

	for _, job := range jobs {
		// 为每个作业查找合适的执行器
		// 这里简化处理，实际应该根据作业需求匹配执行器
		runners, err := s.repo.GetAvailableRunners(ctx, nil)
		if err != nil {
			s.logger.Error("获取可用执行器失败", zap.Error(err))
			continue
		}

		if len(runners) > 0 {
			// 分配给第一个可用执行器
			runner := runners[0]
			updates := map[string]interface{}{
				"status":    models.JobStatusRunning,
				"runner_id": runner.ID,
				"started_at": time.Now().UTC(),
			}

			if err := s.repo.UpdateJob(ctx, job.ID, updates); err != nil {
				s.logger.Error("分配作业失败", zap.Error(err), zap.String("job_id", job.ID.String()))
				continue
			}

			// 更新执行器状态为忙碌
			if err := s.repo.UpdateRunnerStatus(ctx, runner.ID, models.RunnerStatusBusy); err != nil {
				s.logger.Error("更新执行器状态失败", zap.Error(err), zap.String("runner_id", runner.ID.String()))
			}

			s.logger.Info("作业分配成功", 
				zap.String("job_id", job.ID.String()),
				zap.String("runner_id", runner.ID.String()))
		}
	}

	return nil
}

// CleanupOldRuns 清理旧的运行记录
func (s *pipelineService) CleanupOldRuns(ctx context.Context, retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = 30 // 默认保留30天
	}

	cutoffDate := time.Now().UTC().AddDate(0, 0, -retentionDays)
	
	// 这里应该实现清理逻辑，删除旧的运行记录
	// 由于涉及多个表的关联删除，需要谨慎处理
	s.logger.Info("开始清理旧运行记录", zap.Time("cutoff_date", cutoffDate))
	
	// TODO: 实现具体的清理逻辑
	
	return nil
}

// 验证函数

// validateRegisterRunnerRequest 验证注册执行器请求
func (s *pipelineService) validateRegisterRunnerRequest(req *models.RegisterRunnerRequest) error {
	if err := s.validateRunnerName(req.Name); err != nil {
		return err
	}

	if req.Description != nil && len(*req.Description) > 2000 {
		return errors.New("描述不能超过2000个字符")
	}

	if req.Version == "" {
		return errors.New("版本不能为空")
	}

	if req.OS == "" {
		return errors.New("操作系统不能为空")
	}

	if req.Architecture == "" {
		return errors.New("架构不能为空")
	}

	return nil
}

// validateRunnerName 验证执行器名称
func (s *pipelineService) validateRunnerName(name string) error {
	if name == "" {
		return errors.New("执行器名称不能为空")
	}
	
	if len(name) > 255 {
		return errors.New("执行器名称不能超过255个字符")
	}

	// 名称只能包含字母、数字、连字符和下划线
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-_]+$`, name)
	if !matched {
		return errors.New("执行器名称只能包含字母、数字、连字符和下划线")
	}

	return nil
}

// validateRunnerStatus 验证执行器状态
func (s *pipelineService) validateRunnerStatus(status models.RunnerStatus) error {
	switch status {
	case models.RunnerStatusOnline, models.RunnerStatusOffline, models.RunnerStatusIdle, models.RunnerStatusBusy:
		return nil
	default:
		return fmt.Errorf("无效的执行器状态: %s", status)
	}
}

// 错误定义
var (
	ErrPipelineNotFound    = errors.New("流水线不存在")
	ErrPipelineRunNotFound = errors.New("流水线运行不存在")
	ErrJobNotFound         = errors.New("作业不存在")
	ErrRunnerNotFound      = errors.New("执行器不存在")
)
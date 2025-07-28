package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/storage"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PipelineEngine 流水线执行引擎接口
type PipelineEngine interface {
	// 执行流水线
	ExecutePipeline(ctx context.Context, run *models.PipelineRun) error

	// 取消流水线执行
	CancelPipeline(ctx context.Context, runID uuid.UUID) error

	// 获取执行状态
	GetExecutionStatus(ctx context.Context, runID uuid.UUID) (*ExecutionStatus, error)

	// 处理作业结果
	HandleJobResult(ctx context.Context, jobID uuid.UUID, result *JobResult) error
}

// ExecutionStatus 执行状态
type ExecutionStatus struct {
	RunID      uuid.UUID             `json:"run_id"`
	Status     models.PipelineStatus `json:"status"`
	StartedAt  *time.Time            `json:"started_at"`
	FinishedAt *time.Time            `json:"finished_at"`
	Duration   *int64                `json:"duration"`
	Jobs       []JobExecutionStatus  `json:"jobs"`
	Logs       []string              `json:"logs"`
}

// JobExecutionStatus 作业执行状态
type JobExecutionStatus struct {
	JobID      uuid.UUID        `json:"job_id"`
	Name       string           `json:"name"`
	Status     models.JobStatus `json:"status"`
	RunnerID   *uuid.UUID       `json:"runner_id"`
	StartedAt  *time.Time       `json:"started_at"`
	FinishedAt *time.Time       `json:"finished_at"`
	Duration   *int64           `json:"duration"`
	ExitCode   *int             `json:"exit_code"`
	Output     string           `json:"output"`
}

// JobResult 作业执行结果
type JobResult struct {
	JobID      uuid.UUID        `json:"job_id"`
	RunnerID   uuid.UUID        `json:"runner_id"`
	Status     models.JobStatus `json:"status"`
	ExitCode   *int             `json:"exit_code"`
	Output     string           `json:"output"`
	StartedAt  time.Time        `json:"started_at"`
	FinishedAt time.Time        `json:"finished_at"`
	Artifacts  []string         `json:"artifacts"`
}

// PipelineDefinition 流水线定义
type PipelineDefinition struct {
	Name        string               `yaml:"name"`
	Description string               `yaml:"description"`
	Trigger     TriggerConfig        `yaml:"trigger"`
	Variables   map[string]string    `yaml:"variables"`
	Jobs        map[string]JobConfig `yaml:"jobs"`
}

// TriggerConfig 触发器配置
type TriggerConfig struct {
	Push        *PushTrigger     `yaml:"push"`
	PullRequest *PRTrigger       `yaml:"pull_request"`
	Schedule    *ScheduleTrigger `yaml:"schedule"`
}

// PushTrigger Push触发器
type PushTrigger struct {
	Branches []string `yaml:"branches"`
	Tags     []string `yaml:"tags"`
	Paths    []string `yaml:"paths"`
}

// PRTrigger Pull Request触发器
type PRTrigger struct {
	Branches []string `yaml:"branches"`
	Types    []string `yaml:"types"`
}

// ScheduleTrigger 定时触发器
type ScheduleTrigger struct {
	Cron string `yaml:"cron"`
}

// JobConfig 作业配置
type JobConfig struct {
	Name       string            `yaml:"name"`
	RunsOn     string            `yaml:"runs-on"`
	DependsOn  []string          `yaml:"depends-on"`
	If         string            `yaml:"if"`
	TimeoutMin int               `yaml:"timeout-minutes"`
	Variables  map[string]string `yaml:"variables"`
	Steps      []StepConfig      `yaml:"steps"`
}

// StepConfig 步骤配置
type StepConfig struct {
	Name            string            `yaml:"name"`
	Uses            string            `yaml:"uses"`
	Run             string            `yaml:"run"`
	With            map[string]string `yaml:"with"`
	Env             map[string]string `yaml:"env"`
	If              string            `yaml:"if"`
	ContinueOnError bool              `yaml:"continue-on-error"`
}

// pipelineEngine 流水线执行引擎实现
type pipelineEngine struct {
	repo    repository.PipelineRepository
	storage storage.StorageManager
	logger  *zap.Logger

	// 执行中的流水线
	runningPipelines map[uuid.UUID]*pipelineExecution
}

// pipelineExecution 流水线执行状态
type pipelineExecution struct {
	RunID      uuid.UUID
	Definition *PipelineDefinition
	Context    context.Context
	Cancel     context.CancelFunc
	Status     models.PipelineStatus
	StartedAt  time.Time
	Jobs       map[string]*jobExecution
	Logger     *zap.Logger
}

// jobExecution 作业执行状态
type jobExecution struct {
	JobID      uuid.UUID
	Config     *JobConfig
	Status     models.JobStatus
	RunnerID   *uuid.UUID
	StartedAt  *time.Time
	FinishedAt *time.Time
	Output     strings.Builder
	ExitCode   *int
}

// NewPipelineEngine 创建流水线执行引擎
func NewPipelineEngine(repo repository.PipelineRepository, storage storage.StorageManager, logger *zap.Logger) PipelineEngine {
	return &pipelineEngine{
		repo:             repo,
		storage:          storage,
		logger:           logger,
		runningPipelines: make(map[uuid.UUID]*pipelineExecution),
	}
}

// ExecutePipeline 执行流水线
func (e *pipelineEngine) ExecutePipeline(ctx context.Context, run *models.PipelineRun) error {
	logger := e.logger.With(
		zap.String("run_id", run.ID.String()),
		zap.String("pipeline_id", run.PipelineID.String()),
	)

	logger.Info("开始执行流水线")

	// 更新运行状态为运行中
	runUpdates := map[string]interface{}{
		"status":     models.PipelineStatusRunning,
		"started_at": time.Now().UTC(),
	}
	if err := e.repo.UpdatePipelineRun(ctx, run.ID, runUpdates); err != nil {
		logger.Error("更新流水线运行状态失败", zap.Error(err))
		return err
	}

	// 获取流水线定义
	pipeline, err := e.repo.GetPipelineByID(ctx, run.PipelineID)
	if err != nil {
		logger.Error("获取流水线定义失败", zap.Error(err))
		return err
	}

	// 解析流水线定义文件
	definition, err := e.parsePipelineDefinition(ctx, pipeline.DefinitionFilePath)
	if err != nil {
		logger.Error("解析流水线定义失败", zap.Error(err))
		e.updateRunStatus(ctx, run.ID, models.PipelineStatusFailed)
		return err
	}

	// 创建执行上下文
	execCtx, cancel := context.WithCancel(ctx)
	execution := &pipelineExecution{
		RunID:      run.ID,
		Definition: definition,
		Context:    execCtx,
		Cancel:     cancel,
		Status:     models.PipelineStatusRunning,
		StartedAt:  time.Now().UTC(),
		Jobs:       make(map[string]*jobExecution),
		Logger:     logger,
	}

	// 注册执行中的流水线
	e.runningPipelines[run.ID] = execution

	// 异步执行流水线
	go func() {
		defer func() {
			// 清理执行状态
			delete(e.runningPipelines, run.ID)
			cancel()
		}()

		if err := e.executePipelineJobs(execution, run); err != nil {
			logger.Error("流水线执行失败", zap.Error(err))
			e.updateRunStatus(ctx, run.ID, models.PipelineStatusFailed)
			return
		}

		logger.Info("流水线执行完成")
		e.updateRunStatus(ctx, run.ID, models.PipelineStatusSuccess)
	}()

	return nil
}

// CancelPipeline 取消流水线执行
func (e *pipelineEngine) CancelPipeline(ctx context.Context, runID uuid.UUID) error {
	execution, exists := e.runningPipelines[runID]
	if !exists {
		return fmt.Errorf("流水线运行不存在或已完成: %s", runID)
	}

	execution.Logger.Info("取消流水线执行")

	// 取消执行上下文
	execution.Cancel()

	// 更新状态
	execution.Status = models.PipelineStatusCancelled

	// 取消所有正在运行的作业
	for _, job := range execution.Jobs {
		if job.Status == models.JobStatusRunning || job.Status == models.JobStatusPending {
			job.Status = models.JobStatusCancelled
			if job.JobID != uuid.Nil {
				updates := map[string]interface{}{
					"status": models.JobStatusCancelled,
				}
				e.repo.UpdateJob(ctx, job.JobID, updates)
			}
		}
	}

	// 更新流水线运行状态
	return e.updateRunStatus(ctx, runID, models.PipelineStatusCancelled)
}

// GetExecutionStatus 获取执行状态
func (e *pipelineEngine) GetExecutionStatus(ctx context.Context, runID uuid.UUID) (*ExecutionStatus, error) {
	execution, exists := e.runningPipelines[runID]
	if !exists {
		// 从数据库获取历史执行状态
		return e.getHistoricalStatus(ctx, runID)
	}

	status := &ExecutionStatus{
		RunID:     runID,
		Status:    execution.Status,
		StartedAt: &execution.StartedAt,
		Jobs:      make([]JobExecutionStatus, 0),
	}

	// 添加作业状态
	for _, job := range execution.Jobs {
		jobStatus := JobExecutionStatus{
			JobID:      job.JobID,
			Name:       job.Config.Name,
			Status:     job.Status,
			RunnerID:   job.RunnerID,
			StartedAt:  job.StartedAt,
			FinishedAt: job.FinishedAt,
			ExitCode:   job.ExitCode,
			Output:     job.Output.String(),
		}

		if job.StartedAt != nil && job.FinishedAt != nil {
			duration := job.FinishedAt.Sub(*job.StartedAt)
			durationSeconds := int64(duration.Seconds())
			jobStatus.Duration = &durationSeconds
		}

		status.Jobs = append(status.Jobs, jobStatus)
	}

	// 计算总持续时间
	if execution.Status == models.PipelineStatusSuccess ||
		execution.Status == models.PipelineStatusFailed ||
		execution.Status == models.PipelineStatusCancelled {
		now := time.Now().UTC()
		duration := now.Sub(execution.StartedAt)
		durationSeconds := int64(duration.Seconds())
		status.Duration = &durationSeconds
		status.FinishedAt = &now
	}

	return status, nil
}

// HandleJobResult 处理作业结果
func (e *pipelineEngine) HandleJobResult(ctx context.Context, jobID uuid.UUID, result *JobResult) error {
	logger := e.logger.With(zap.String("job_id", jobID.String()))

	// 更新数据库中的作业状态
	updates := map[string]interface{}{
		"status":      result.Status,
		"exit_code":   result.ExitCode,
		"log_output":  result.Output,
		"started_at":  result.StartedAt,
		"finished_at": result.FinishedAt,
	}

	duration := result.FinishedAt.Sub(result.StartedAt)
	durationSeconds := int64(duration.Seconds())
	updates["duration"] = durationSeconds

	if err := e.repo.UpdateJob(ctx, jobID, updates); err != nil {
		logger.Error("更新作业状态失败", zap.Error(err))
		return err
	}

	// 存储日志
	if result.Output != "" {
		if err := e.storage.WriteLog(ctx, jobID, "execution", []byte(result.Output)); err != nil {
			logger.Warn("存储作业日志失败", zap.Error(err))
		}
	}

	// 处理构建产物
	for _, artifactPath := range result.Artifacts {
		logger.Info("处理构建产物", zap.String("artifact", artifactPath))
		// TODO: 实现构建产物处理逻辑
	}

	// 更新内存中的执行状态
	e.updateJobExecutionStatus(jobID, result)

	logger.Info("作业结果处理完成",
		zap.String("status", string(result.Status)),
		zap.Int64("duration_seconds", durationSeconds))

	return nil
}

// 私有方法

// parsePipelineDefinition 解析流水线定义
func (e *pipelineEngine) parsePipelineDefinition(ctx context.Context, definitionPath string) (*PipelineDefinition, error) {
	// 这里应该从代码仓库读取定义文件
	// 暂时返回一个示例定义
	return &PipelineDefinition{
		Name:        "示例流水线",
		Description: "这是一个示例流水线",
		Variables: map[string]string{
			"NODE_VERSION": "18",
		},
		Jobs: map[string]JobConfig{
			"build": {
				Name:       "构建作业",
				RunsOn:     "ubuntu-latest",
				TimeoutMin: 30,
				Steps: []StepConfig{
					{
						Name: "检出代码",
						Uses: "actions/checkout@v4",
					},
					{
						Name: "设置Node.js",
						Uses: "actions/setup-node@v4",
						With: map[string]string{
							"node-version": "${{ vars.NODE_VERSION }}",
						},
					},
					{
						Name: "安装依赖",
						Run:  "npm ci",
					},
					{
						Name: "构建应用",
						Run:  "npm run build",
					},
					{
						Name: "运行测试",
						Run:  "npm test",
					},
				},
			},
		},
	}, nil
}

// executePipelineJobs 执行流水线作业
func (e *pipelineEngine) executePipelineJobs(execution *pipelineExecution, run *models.PipelineRun) error {
	// 解析作业依赖关系
	jobGraph, err := e.buildJobDependencyGraph(execution.Definition.Jobs)
	if err != nil {
		return fmt.Errorf("构建作业依赖图失败: %w", err)
	}

	// 按照依赖关系执行作业
	return e.executeJobsByDependency(execution, run, jobGraph)
}

// buildJobDependencyGraph 构建作业依赖图
func (e *pipelineEngine) buildJobDependencyGraph(jobs map[string]JobConfig) (map[string][]string, error) {
	graph := make(map[string][]string)

	for jobName, job := range jobs {
		graph[jobName] = job.DependsOn
	}

	// 检测循环依赖
	if e.hasCyclicDependency(graph) {
		return nil, fmt.Errorf("检测到循环依赖")
	}

	return graph, nil
}

// hasCyclicDependency 检测循环依赖
func (e *pipelineEngine) hasCyclicDependency(graph map[string][]string) bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for node := range graph {
		if !visited[node] {
			if e.hasCyclicDependencyUtil(node, graph, visited, recStack) {
				return true
			}
		}
	}

	return false
}

// hasCyclicDependencyUtil 循环依赖检测辅助函数
func (e *pipelineEngine) hasCyclicDependencyUtil(node string, graph map[string][]string, visited, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, neighbor := range graph[node] {
		if !visited[neighbor] {
			if e.hasCyclicDependencyUtil(neighbor, graph, visited, recStack) {
				return true
			}
		} else if recStack[neighbor] {
			return true
		}
	}

	recStack[node] = false
	return false
}

// executeJobsByDependency 按依赖关系执行作业
func (e *pipelineEngine) executeJobsByDependency(execution *pipelineExecution, run *models.PipelineRun, jobGraph map[string][]string) error {
	completed := make(map[string]bool)
	failed := make(map[string]bool)

	for len(completed)+len(failed) < len(jobGraph) {
		// 检查是否被取消
		select {
		case <-execution.Context.Done():
			return execution.Context.Err()
		default:
		}

		// 找到可以执行的作业（依赖已完成且未执行）
		readyJobs := e.findReadyJobs(jobGraph, completed, failed)
		if len(readyJobs) == 0 {
			// 没有可执行的作业，可能是因为某些作业失败导致依赖无法满足
			break
		}

		// 并行执行就绪的作业
		jobResults := make(chan jobExecutionResult, len(readyJobs))

		for _, jobName := range readyJobs {
			go func(name string) {
				err := e.executeJob(execution, run, name)
				jobResults <- jobExecutionResult{
					JobName: name,
					Error:   err,
				}
			}(jobName)
		}

		// 等待作业完成
		for i := 0; i < len(readyJobs); i++ {
			result := <-jobResults
			if result.Error != nil {
				execution.Logger.Error("作业执行失败",
					zap.String("job", result.JobName),
					zap.Error(result.Error))
				failed[result.JobName] = true
			} else {
				completed[result.JobName] = true
			}
		}
	}

	// 检查是否所有作业都成功完成
	if len(completed) != len(jobGraph) {
		return fmt.Errorf("流水线执行失败，成功: %d, 失败: %d, 总数: %d",
			len(completed), len(failed), len(jobGraph))
	}

	return nil
}

// jobExecutionResult 作业执行结果
type jobExecutionResult struct {
	JobName string
	Error   error
}

// findReadyJobs 找到准备就绪的作业
func (e *pipelineEngine) findReadyJobs(jobGraph map[string][]string, completed, failed map[string]bool) []string {
	var readyJobs []string

	for jobName, dependencies := range jobGraph {
		// 跳过已完成或失败的作业
		if completed[jobName] || failed[jobName] {
			continue
		}

		// 检查依赖是否都已完成
		canExecute := true
		for _, dep := range dependencies {
			if !completed[dep] {
				canExecute = false
				break
			}
		}

		if canExecute {
			readyJobs = append(readyJobs, jobName)
		}
	}

	return readyJobs
}

// executeJob 执行单个作业
func (e *pipelineEngine) executeJob(execution *pipelineExecution, run *models.PipelineRun, jobName string) error {
	logger := execution.Logger.With(zap.String("job", jobName))
	logger.Info("开始执行作业")

	jobConfig := execution.Definition.Jobs[jobName]

	// 创建作业记录
	job := &models.Job{
		PipelineRunID: run.ID,
		Name:          jobConfig.Name,
		Type:          models.JobTypeBuild, // 使用正确的类型
		Status:        models.JobStatusPending,
	}

	if err := e.repo.CreateJob(execution.Context, job); err != nil {
		return fmt.Errorf("创建作业记录失败: %w", err)
	}

	// 更新执行状态
	jobExec := &jobExecution{
		JobID:  job.ID,
		Config: &jobConfig,
		Status: models.JobStatusPending,
	}
	execution.Jobs[jobName] = jobExec

	// 查找可用的执行器
	runners, err := e.repo.GetAvailableRunners(execution.Context, nil)
	if err != nil {
		return fmt.Errorf("获取可用执行器失败: %w", err)
	}

	if len(runners) == 0 {
		return fmt.Errorf("没有可用的执行器")
	}

	// 分配给第一个可用的执行器
	runner := runners[0]

	// 更新作业状态为运行中
	now := time.Now().UTC()
	jobExec.Status = models.JobStatusRunning
	jobExec.RunnerID = &runner.ID
	jobExec.StartedAt = &now

	updates := map[string]interface{}{
		"status":     models.JobStatusRunning,
		"runner_id":  runner.ID,
		"started_at": now,
	}

	if err := e.repo.UpdateJob(execution.Context, job.ID, updates); err != nil {
		return fmt.Errorf("更新作业状态失败: %w", err)
	}

	// 更新执行器状态为忙碌
	if err := e.repo.UpdateRunnerStatus(execution.Context, runner.ID, models.RunnerStatusBusy); err != nil {
		logger.Warn("更新执行器状态失败", zap.Error(err))
	}

	logger.Info("作业已分配给执行器", zap.String("runner_id", runner.ID.String()))

	// 这里应该通过消息队列或其他方式通知执行器开始执行作业
	// 暂时模拟作业执行成功
	time.Sleep(2 * time.Second) // 模拟执行时间

	// 模拟作业完成
	finishedAt := time.Now().UTC()
	jobExec.Status = models.JobStatusSuccess
	jobExec.FinishedAt = &finishedAt
	jobExec.ExitCode = new(int) // 0 表示成功

	duration := finishedAt.Sub(now)
	durationSeconds := int64(duration.Seconds())

	finalUpdates := map[string]interface{}{
		"status":      models.JobStatusSuccess,
		"finished_at": finishedAt,
		"duration":    durationSeconds,
		"exit_code":   0,
		"log_output":  "作业执行成功",
	}

	if err := e.repo.UpdateJob(execution.Context, job.ID, finalUpdates); err != nil {
		return fmt.Errorf("更新作业完成状态失败: %w", err)
	}

	// 更新执行器状态为空闲
	if err := e.repo.UpdateRunnerStatus(execution.Context, runner.ID, models.RunnerStatusIdle); err != nil {
		logger.Warn("更新执行器状态失败", zap.Error(err))
	}

	logger.Info("作业执行完成", zap.Int64("duration_seconds", durationSeconds))
	return nil
}

// updateRunStatus 更新流水线运行状态
func (e *pipelineEngine) updateRunStatus(ctx context.Context, runID uuid.UUID, status models.PipelineStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == models.PipelineStatusSuccess ||
		status == models.PipelineStatusFailed ||
		status == models.PipelineStatusCancelled {
		updates["finished_at"] = time.Now().UTC()
	}

	return e.repo.UpdatePipelineRun(ctx, runID, updates)
}

// updateJobExecutionStatus 更新内存中的作业执行状态
func (e *pipelineEngine) updateJobExecutionStatus(jobID uuid.UUID, result *JobResult) {
	for _, execution := range e.runningPipelines {
		for _, job := range execution.Jobs {
			if job.JobID == jobID {
				job.Status = result.Status
				job.FinishedAt = &result.FinishedAt
				job.ExitCode = result.ExitCode
				job.Output.WriteString(result.Output)
				break
			}
		}
	}
}

// getHistoricalStatus 获取历史执行状态
func (e *pipelineEngine) getHistoricalStatus(ctx context.Context, runID uuid.UUID) (*ExecutionStatus, error) {
	// 从数据库获取流水线运行信息
	run, err := e.repo.GetPipelineRunByID(ctx, runID)
	if err != nil {
		return nil, err
	}

	// 获取作业列表
	jobs, err := e.repo.GetJobsByPipelineRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	status := &ExecutionStatus{
		RunID:      runID,
		Status:     run.Status,
		StartedAt:  run.StartedAt,
		FinishedAt: run.FinishedAt,
		Jobs:       make([]JobExecutionStatus, len(jobs)),
	}

	if run.StartedAt != nil && run.FinishedAt != nil {
		duration := run.FinishedAt.Sub(*run.StartedAt)
		durationSeconds := int64(duration.Seconds())
		status.Duration = &durationSeconds
	}

	// 转换作业状态
	for i, job := range jobs {
		// LogOutput字段不存在，使用ErrorMessage或LogPath
		var output string
		if job.ErrorMessage != "" {
			output = job.ErrorMessage
		}

		status.Jobs[i] = JobExecutionStatus{
			JobID:      job.ID,
			Name:       job.Name,
			Status:     job.Status,
			RunnerID:   job.RunnerID,
			StartedAt:  job.StartedAt,
			FinishedAt: job.FinishedAt,
			Duration:   convertDurationToInt64(job.Duration),
			ExitCode:   job.ExitCode,
			Output:     output,
		}
	}

	return status, nil
}

// convertDurationToInt64 将time.Duration转换为int64秒数
func convertDurationToInt64(d *time.Duration) *int64 {
	if d == nil {
		return nil
	}
	seconds := int64(d.Seconds())
	return &seconds
}

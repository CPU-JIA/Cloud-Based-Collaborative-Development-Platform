package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/docker"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/scheduler"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/storage"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ExecutionService 执行服务接口
type ExecutionService interface {
	// 启动服务
	Start(ctx context.Context) error
	
	// 停止服务
	Stop() error
	
	// 提交作业执行
	SubmitJobExecution(ctx context.Context, job *models.Job) error
	
	// 获取执行状态
	GetExecutionStatus(jobID uuid.UUID) (*JobExecutionStatus, error)
	
	// 取消作业执行
	CancelJobExecution(ctx context.Context, jobID uuid.UUID) error
	
	// 获取执行统计
	GetExecutionStats() (*ExecutionStats, error)
	
	// 健康检查
	HealthCheck(ctx context.Context) error
}

// ExecutionStats 执行统计
type ExecutionStats struct {
	TotalJobs         int                      `json:"total_jobs"`
	RunningJobs       int                      `json:"running_jobs"`
	PendingJobs       int                      `json:"pending_jobs"`
	CompletedJobs     int                      `json:"completed_jobs"`
	FailedJobs        int                      `json:"failed_jobs"`
	CancelledJobs     int                      `json:"cancelled_jobs"`
	AverageExecTime   time.Duration            `json:"average_exec_time"`
	ResourceUsage     *AggregateResourceUsage  `json:"resource_usage"`
	ExecutorHealth    bool                     `json:"executor_health"`
	DockerHealth      bool                     `json:"docker_health"`
	LastUpdated       time.Time                `json:"last_updated"`
}

// AggregateResourceUsage 聚合资源使用情况
type AggregateResourceUsage struct {
	TotalCPUUsage    float64 `json:"total_cpu_usage"`
	TotalMemoryUsage int64   `json:"total_memory_usage"`
	TotalNetworkRx   int64   `json:"total_network_rx"`
	TotalNetworkTx   int64   `json:"total_network_tx"`
	TotalDiskRead    int64   `json:"total_disk_read"`
	TotalDiskWrite   int64   `json:"total_disk_write"`
	ActiveContainers int     `json:"active_containers"`
}

// ExecutionServiceConfig 执行服务配置
type ExecutionServiceConfig struct {
	// 执行器配置
	ExecutorConfig *ExecutorConfig `json:"executor_config"`
	
	// 调度器集成配置
	JobPollingInterval  time.Duration `json:"job_polling_interval"`
	StatusUpdateInterval time.Duration `json:"status_update_interval"`
	
	// 重试配置
	MaxRetryAttempts int           `json:"max_retry_attempts"`
	RetryDelay       time.Duration `json:"retry_delay"`
	
	// 监控配置
	EnableMetrics        bool          `json:"enable_metrics"`
	MetricsUpdateInterval time.Duration `json:"metrics_update_interval"`
}

// executionService 执行服务实现
type executionService struct {
	config           *ExecutionServiceConfig
	executor         JobExecutor
	dockerManager    docker.DockerManager
	storageManager   storage.StorageManager
	pipelineRepo     repository.PipelineRepository
	scheduler        scheduler.JobScheduler
	logger           *zap.Logger
	
	// 状态管理
	running          bool
	stopCh           chan struct{}
	wg               sync.WaitGroup
	
	// 统计信息
	stats            *ExecutionStats
	statsMu          sync.RWMutex
}

// NewExecutionService 创建执行服务
func NewExecutionService(
	config *ExecutionServiceConfig,
	dockerManager docker.DockerManager,
	storageManager storage.StorageManager,
	pipelineRepo repository.PipelineRepository,
	jobScheduler scheduler.JobScheduler,
	logger *zap.Logger,
) (ExecutionService, error) {
	if config == nil {
		config = DefaultExecutionServiceConfig()
	}
	
	// 创建作业执行器
	executor := NewJobExecutor(
		config.ExecutorConfig,
		dockerManager,
		storageManager,
		logger,
	)
	
	service := &executionService{
		config:         config,
		executor:       executor,
		dockerManager:  dockerManager,
		storageManager: storageManager,
		pipelineRepo:   pipelineRepo,
		scheduler:      jobScheduler,
		logger:         logger.With(zap.String("component", "execution_service")),
		stopCh:         make(chan struct{}),
		stats: &ExecutionStats{
			LastUpdated: time.Now(),
		},
	}
	
	return service, nil
}

// Start 启动服务
func (es *executionService) Start(ctx context.Context) error {
	if es.running {
		return fmt.Errorf("执行服务已经在运行")
	}
	
	es.logger.Info("启动执行服务")
	
	// 启动作业处理循环
	es.wg.Add(1)
	go es.jobProcessingLoop()
	
	// 启动状态更新循环
	es.wg.Add(1)
	go es.statusUpdateLoop()
	
	// 启动指标收集循环
	if es.config.EnableMetrics {
		es.wg.Add(1)
		go es.metricsCollectionLoop()
	}
	
	// 启动健康检查循环
	es.wg.Add(1)
	go es.healthCheckLoop()
	
	es.running = true
	es.logger.Info("执行服务启动成功")
	
	return nil
}

// Stop 停止服务
func (es *executionService) Stop() error {
	if !es.running {
		return nil
	}
	
	es.logger.Info("停止执行服务")
	
	// 发送停止信号
	close(es.stopCh)
	es.running = false
	
	// 等待所有goroutine结束
	es.wg.Wait()
	
	es.logger.Info("执行服务已停止")
	return nil
}

// jobProcessingLoop 作业处理循环
func (es *executionService) jobProcessingLoop() {
	defer es.wg.Done()
	
	ticker := time.NewTicker(es.config.JobPollingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-es.stopCh:
			es.logger.Info("作业处理循环退出")
			return
			
		case <-ticker.C:
			es.processScheduledJobs()
		}
	}
}

// processScheduledJobs 处理调度的作业
func (es *executionService) processScheduledJobs() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 从调度器获取待执行的作业
	jobs, err := es.getScheduledJobs(ctx)
	if err != nil {
		es.logger.Error("获取调度作业失败", zap.Error(err))
		return
	}
	
	for _, job := range jobs {
		// 异步执行作业
		go es.executeJobAsync(job)
	}
}

// getScheduledJobs 从调度器获取待执行作业
func (es *executionService) getScheduledJobs(ctx context.Context) ([]*models.Job, error) {
	// 使用repository接口的方法获取待处理作业
	jobs, err := es.pipelineRepo.GetPendingJobs(ctx, []string{}) // 空标签表示获取所有作业
	if err != nil {
		return nil, fmt.Errorf("获取待处理作业失败: %v", err)
	}
	
	// 转换为指针数组
	jobPtrs := make([]*models.Job, len(jobs))
	for i, job := range jobs {
		jobCopy := job // 避免循环变量引用问题
		jobPtrs[i] = &jobCopy
	}
	
	return jobPtrs, nil
}

// executeJobAsync 异步执行作业
func (es *executionService) executeJobAsync(job *models.Job) {
	es.logger.Info("开始异步执行作业",
		zap.String("job_id", job.ID.String()),
		zap.String("job_name", job.Name))
	
	ctx, cancel := context.WithTimeout(context.Background(), es.config.ExecutorConfig.DefaultTimeout)
	defer cancel()
	
	// 更新作业状态为运行中
	if err := es.updateJobStatus(ctx, job.ID, models.JobStatusRunning, "", nil); err != nil {
		es.logger.Error("更新作业状态失败", zap.Error(err))
		return
	}
	
	// 执行作业
	err := es.executor.ExecuteJob(ctx, job)
	
	// 获取执行状态
	status, statusErr := es.executor.GetJobStatus(job.ID)
	if statusErr != nil {
		es.logger.Error("获取作业执行状态失败", zap.Error(statusErr))
	}
	
	// 更新数据库中的作业状态
	var finalStatus models.JobStatus
	var errorMessage string
	var exitCode *int
	
	if err != nil {
		finalStatus = models.JobStatusFailed
		errorMessage = err.Error()
		es.logger.Error("作业执行失败",
			zap.String("job_id", job.ID.String()),
			zap.Error(err))
	} else if status != nil {
		finalStatus = status.Status
		errorMessage = status.ErrorMessage
		exitCode = status.ExitCode
	} else {
		finalStatus = models.JobStatusFailed
		errorMessage = "无法获取执行状态"
	}
	
	// 更新作业最终状态
	if err := es.updateJobStatus(ctx, job.ID, finalStatus, errorMessage, exitCode); err != nil {
		es.logger.Error("更新作业最终状态失败", zap.Error(err))
	}
	
	// 更新统计信息
	es.updateStatsAfterJobCompletion(finalStatus)
}

// updateJobStatus 更新作业状态
func (es *executionService) updateJobStatus(ctx context.Context, jobID uuid.UUID, status models.JobStatus, errorMessage string, exitCode *int) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	
	if status == models.JobStatusRunning {
		updates["started_at"] = time.Now()
	}
	
	if status == models.JobStatusSuccess || status == models.JobStatusFailed || status == models.JobStatusCancelled {
		updates["finished_at"] = time.Now()
	}
	
	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}
	
	if exitCode != nil {
		updates["exit_code"] = *exitCode
	}
	
	err := es.pipelineRepo.UpdateJob(ctx, jobID, updates)
	if err != nil {
		return fmt.Errorf("更新作业状态失败: %v", err)
	}
	
	es.logger.Info("作业状态已更新",
		zap.String("job_id", jobID.String()),
		zap.String("status", string(status)))
	
	return nil
}

// statusUpdateLoop 状态更新循环
func (es *executionService) statusUpdateLoop() {
	defer es.wg.Done()
	
	ticker := time.NewTicker(es.config.StatusUpdateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-es.stopCh:
			es.logger.Info("状态更新循环退出")
			return
			
		case <-ticker.C:
			es.updateRunningJobsStatus()
		}
	}
}

// updateRunningJobsStatus 更新运行中作业的状态
func (es *executionService) updateRunningJobsStatus() {
	// TODO: 需要在repository接口中添加GetRunningJobs方法
	// 现在临时跳过这个功能
	es.logger.Debug("跳过运行中作业状态更新 - 需要实现GetRunningJobs方法")
}

// metricsCollectionLoop 指标收集循环
func (es *executionService) metricsCollectionLoop() {
	defer es.wg.Done()
	
	ticker := time.NewTicker(es.config.MetricsUpdateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-es.stopCh:
			es.logger.Info("指标收集循环退出")
			return
			
		case <-ticker.C:
			es.collectMetrics()
		}
	}
}

// collectMetrics 收集指标
func (es *executionService) collectMetrics() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	es.statsMu.Lock()
	defer es.statsMu.Unlock()
	
	// 收集作业统计
	if err := es.collectJobStats(ctx); err != nil {
		es.logger.Error("收集作业统计失败", zap.Error(err))
	}
	
	// 收集资源使用统计
	if err := es.collectResourceStats(); err != nil {
		es.logger.Error("收集资源统计失败", zap.Error(err))
	}
	
	// 更新健康状态
	es.stats.ExecutorHealth = es.checkExecutorHealth()
	es.stats.DockerHealth = es.checkDockerHealth(ctx)
	es.stats.LastUpdated = time.Now()
	
	es.logger.Debug("指标收集完成",
		zap.Int("running_jobs", es.stats.RunningJobs),
		zap.Int("pending_jobs", es.stats.PendingJobs),
		zap.Bool("executor_health", es.stats.ExecutorHealth),
		zap.Bool("docker_health", es.stats.DockerHealth))
}

// collectJobStats 收集作业统计
func (es *executionService) collectJobStats(ctx context.Context) error {
	// TODO: 简化统计实现，等repository接口完善后再实现复杂统计查询
	// 现在使用模拟数据
	es.stats.TotalJobs = 0
	es.stats.RunningJobs = 0
	es.stats.PendingJobs = 0
	es.stats.CompletedJobs = 0
	es.stats.FailedJobs = 0
	es.stats.CancelledJobs = 0
	es.stats.AverageExecTime = 0
	
	es.logger.Debug("使用模拟统计数据 - 等待repository接口完善")
	return nil
}

// collectResourceStats 收集资源统计
func (es *executionService) collectResourceStats() error {
	// 这里简化实现，实际应该从执行器获取真实的资源数据
	if es.stats.ResourceUsage == nil {
		es.stats.ResourceUsage = &AggregateResourceUsage{}
	}
	
	es.stats.ResourceUsage.ActiveContainers = es.stats.RunningJobs
	
	return nil
}

// healthCheckLoop 健康检查循环
func (es *executionService) healthCheckLoop() {
	defer es.wg.Done()
	
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-es.stopCh:
			es.logger.Info("健康检查循环退出")
			return
			
		case <-ticker.C:
			es.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (es *executionService) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 检查执行器健康状态
	executorHealth := es.executor.HealthCheck(ctx) == nil
	
	// 检查Docker健康状态
	dockerHealth := es.dockerManager.HealthCheck(ctx) == nil
	
	es.statsMu.Lock()
	es.stats.ExecutorHealth = executorHealth
	es.stats.DockerHealth = dockerHealth
	es.statsMu.Unlock()
	
	if !executorHealth {
		es.logger.Warn("执行器健康检查失败")
	}
	
	if !dockerHealth {
		es.logger.Warn("Docker健康检查失败")
	}
}

// SubmitJobExecution 提交作业执行
func (es *executionService) SubmitJobExecution(ctx context.Context, job *models.Job) error {
	if !es.running {
		return fmt.Errorf("执行服务未运行")
	}
	
	es.logger.Info("提交作业执行",
		zap.String("job_id", job.ID.String()),
		zap.String("job_name", job.Name))
	
	// 异步执行作业
	go es.executeJobAsync(job)
	
	return nil
}

// GetExecutionStatus 获取执行状态
func (es *executionService) GetExecutionStatus(jobID uuid.UUID) (*JobExecutionStatus, error) {
	return es.executor.GetJobStatus(jobID)
}

// CancelJobExecution 取消作业执行
func (es *executionService) CancelJobExecution(ctx context.Context, jobID uuid.UUID) error {
	es.logger.Info("取消作业执行", zap.String("job_id", jobID.String()))
	
	// 停止执行器中的作业
	if err := es.executor.StopJob(ctx, jobID); err != nil {
		es.logger.Error("停止作业失败", zap.Error(err))
	}
	
	// 更新数据库状态
	return es.updateJobStatus(ctx, jobID, models.JobStatusCancelled, "作业被取消", nil)
}

// GetExecutionStats 获取执行统计
func (es *executionService) GetExecutionStats() (*ExecutionStats, error) {
	es.statsMu.RLock()
	defer es.statsMu.RUnlock()
	
	// 返回统计信息副本
	statsCopy := *es.stats
	if es.stats.ResourceUsage != nil {
		resourceCopy := *es.stats.ResourceUsage
		statsCopy.ResourceUsage = &resourceCopy
	}
	
	return &statsCopy, nil
}

// HealthCheck 健康检查
func (es *executionService) HealthCheck(ctx context.Context) error {
	if !es.running {
		return fmt.Errorf("执行服务未运行")
	}
	
	// 检查执行器
	if err := es.executor.HealthCheck(ctx); err != nil {
		return fmt.Errorf("执行器健康检查失败: %v", err)
	}
	
	// 检查Docker管理器
	if err := es.dockerManager.HealthCheck(ctx); err != nil {
		return fmt.Errorf("Docker管理器健康检查失败: %v", err)
	}
	
	return nil
}

// 辅助方法

// checkExecutorHealth 检查执行器健康状态
func (es *executionService) checkExecutorHealth() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	return es.executor.HealthCheck(ctx) == nil
}

// checkDockerHealth 检查Docker健康状态
func (es *executionService) checkDockerHealth(ctx context.Context) bool {
	return es.dockerManager.HealthCheck(ctx) == nil
}

// updateStatsAfterJobCompletion 作业完成后更新统计信息
func (es *executionService) updateStatsAfterJobCompletion(status models.JobStatus) {
	es.statsMu.Lock()
	defer es.statsMu.Unlock()
	
	switch status {
	case models.JobStatusSuccess:
		if es.stats.CompletedJobs > 0 {
			es.stats.CompletedJobs++
		}
	case models.JobStatusFailed:
		if es.stats.FailedJobs > 0 {
			es.stats.FailedJobs++
		}
	case models.JobStatusCancelled:
		if es.stats.CancelledJobs > 0 {
			es.stats.CancelledJobs++
		}
	}
	
	if es.stats.RunningJobs > 0 {
		es.stats.RunningJobs--
	}
}

// DefaultExecutionServiceConfig 默认执行服务配置
func DefaultExecutionServiceConfig() *ExecutionServiceConfig {
	return &ExecutionServiceConfig{
		ExecutorConfig:        DefaultExecutorConfig(),
		JobPollingInterval:    5 * time.Second,
		StatusUpdateInterval:  10 * time.Second,
		MaxRetryAttempts:      3,
		RetryDelay:           30 * time.Second,
		EnableMetrics:        true,
		MetricsUpdateInterval: 30 * time.Second,
	}
}
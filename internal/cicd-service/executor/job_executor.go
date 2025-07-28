package executor

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/docker"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/storage"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// JobExecutor 作业执行器接口
type JobExecutor interface {
	// 执行作业
	ExecuteJob(ctx context.Context, job *models.Job) error

	// 停止作业执行
	StopJob(ctx context.Context, jobID uuid.UUID) error

	// 获取作业状态
	GetJobStatus(jobID uuid.UUID) (*JobExecutionStatus, error)

	// 获取作业日志
	GetJobLogs(ctx context.Context, jobID uuid.UUID) (io.ReadCloser, error)

	// 清理作业资源
	CleanupJob(ctx context.Context, jobID uuid.UUID) error

	// 健康检查
	HealthCheck(ctx context.Context) error
}

// JobExecutionStatus 作业执行状态
type JobExecutionStatus struct {
	JobID        uuid.UUID        `json:"job_id"`
	Status       models.JobStatus `json:"status"`
	ContainerID  string           `json:"container_id,omitempty"`
	StartTime    *time.Time       `json:"start_time,omitempty"`
	EndTime      *time.Time       `json:"end_time,omitempty"`
	ExitCode     *int             `json:"exit_code,omitempty"`
	ErrorMessage string           `json:"error_message,omitempty"`
	Resources    *ResourceUsage   `json:"resources,omitempty"`
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage int64   `json:"memory_usage"`
	MemoryLimit int64   `json:"memory_limit"`
	NetworkRx   int64   `json:"network_rx"`
	NetworkTx   int64   `json:"network_tx"`
	DiskRead    int64   `json:"disk_read"`
	DiskWrite   int64   `json:"disk_write"`
}

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	// Docker配置
	DockerConfig *docker.ManagerConfig `json:"docker_config"`

	// 执行配置
	MaxConcurrentJobs int           `json:"max_concurrent_jobs"`
	DefaultTimeout    time.Duration `json:"default_timeout"`

	// 资源限制
	DefaultCPULimit    float64 `json:"default_cpu_limit"`
	DefaultMemoryLimit int64   `json:"default_memory_limit"`
	DefaultDiskLimit   int64   `json:"default_disk_limit"`

	// 清理配置
	EnableAutoCleanup bool          `json:"enable_auto_cleanup"`
	CleanupTimeout    time.Duration `json:"cleanup_timeout"`

	// 日志配置
	LogRetentionDays int  `json:"log_retention_days"`
	StreamLogs       bool `json:"stream_logs"`
}

// jobExecutor 作业执行器实现
type jobExecutor struct {
	config         *ExecutorConfig
	dockerManager  docker.DockerManager
	storageManager storage.StorageManager
	logger         *zap.Logger

	// 执行状态管理
	executions   map[uuid.UUID]*JobExecutionStatus
	executionsMu sync.RWMutex

	// 并发控制
	semaphore chan struct{}

	// 停止信号
	stopCh  chan struct{}
	stopped bool
}

// NewJobExecutor 创建作业执行器
func NewJobExecutor(
	config *ExecutorConfig,
	dockerManager docker.DockerManager,
	storageManager storage.StorageManager,
	logger *zap.Logger,
) JobExecutor {
	if config == nil {
		config = DefaultExecutorConfig()
	}

	executor := &jobExecutor{
		config:         config,
		dockerManager:  dockerManager,
		storageManager: storageManager,
		logger:         logger.With(zap.String("component", "job_executor")),
		executions:     make(map[uuid.UUID]*JobExecutionStatus),
		semaphore:      make(chan struct{}, config.MaxConcurrentJobs),
		stopCh:         make(chan struct{}),
	}

	// 启动资源监控
	go executor.startResourceMonitoring()

	// 启动自动清理
	if config.EnableAutoCleanup {
		go executor.startAutoCleanup()
	}

	return executor
}

// ExecuteJob 执行作业
func (je *jobExecutor) ExecuteJob(ctx context.Context, job *models.Job) error {
	if je.stopped {
		return fmt.Errorf("执行器已停止")
	}

	// 获取并发许可
	select {
	case je.semaphore <- struct{}{}:
		defer func() { <-je.semaphore }()
	case <-ctx.Done():
		return ctx.Err()
	}

	je.logger.Info("开始执行作业",
		zap.String("job_id", job.ID.String()),
		zap.String("job_name", job.Name))

	// 初始化执行状态
	status := &JobExecutionStatus{
		JobID:  job.ID,
		Status: models.JobStatusRunning,
	}

	je.executionsMu.Lock()
	je.executions[job.ID] = status
	je.executionsMu.Unlock()

	// 创建执行上下文
	execCtx, cancel := je.createExecutionContext(ctx, job)
	defer cancel()

	// 执行作业
	return je.executeJobInternal(execCtx, job, status)
}

// executeJobInternal 内部作业执行逻辑
func (je *jobExecutor) executeJobInternal(ctx context.Context, job *models.Job, status *JobExecutionStatus) error {
	startTime := time.Now()
	status.StartTime = &startTime

	defer func() {
		endTime := time.Now()
		status.EndTime = &endTime
	}()

	// 1. 准备执行环境
	containerConfig, err := je.prepareExecutionEnvironment(job)
	if err != nil {
		return je.handleExecutionError(status, fmt.Errorf("准备执行环境失败: %v", err))
	}

	// 2. 创建容器
	container, err := je.dockerManager.CreateContainer(ctx, containerConfig)
	if err != nil {
		return je.handleExecutionError(status, fmt.Errorf("创建容器失败: %v", err))
	}

	status.ContainerID = container.ID
	je.logger.Info("容器创建成功",
		zap.String("job_id", job.ID.String()),
		zap.String("container_id", container.ID))

	// 3. 启动容器
	if err := je.dockerManager.StartContainer(ctx, container.ID); err != nil {
		// 清理容器
		_ = je.dockerManager.RemoveContainer(context.Background(), container.ID, true)
		return je.handleExecutionError(status, fmt.Errorf("启动容器失败: %v", err))
	}

	// 4. 监控容器执行
	return je.monitorContainerExecution(ctx, job, status, container.ID)
}

// prepareExecutionEnvironment 准备执行环境
func (je *jobExecutor) prepareExecutionEnvironment(job *models.Job) (*docker.ContainerConfig, error) {
	// 解析作业配置
	image := "ubuntu:20.04" // 默认镜像
	if job.Config != nil {
		if img, ok := job.Config["image"].(string); ok && img != "" {
			image = img
		}
	}

	// 构建执行命令
	commands := je.buildExecutionCommands(job)

	// 设置环境变量
	env := je.buildEnvironmentVariables(job)

	// 设置卷挂载
	volumes := je.buildVolumeBindings(job)

	// 设置资源限制
	cpuLimit := je.config.DefaultCPULimit
	memoryLimit := je.config.DefaultMemoryLimit

	if job.Config != nil {
		if cpu, ok := job.Config["cpu_limit"].(float64); ok && cpu > 0 {
			cpuLimit = cpu
		}
		if mem, ok := job.Config["memory_limit"].(float64); ok && mem > 0 {
			memoryLimit = int64(mem)
		}
	}

	// 构建容器配置
	config := &docker.ContainerConfig{
		Name:       fmt.Sprintf("job-%s", job.ID.String()),
		Image:      strings.Split(image, ":")[0],
		Tag:        je.extractImageTag(image),
		Cmd:        commands,
		Env:        env,
		WorkingDir: "/workspace",
		Volumes:    volumes,
		Labels: map[string]string{
			"job_id":      job.ID.String(),
			"pipeline_id": job.PipelineRunID.String(),
			"executor":    "cicd-service",
		},

		// 资源限制
		CPULimit:    cpuLimit,
		MemoryLimit: memoryLimit,
		DiskLimit:   je.config.DefaultDiskLimit,

		// 安全配置
		User:         "1000:1000", // 非root用户
		Privileged:   false,
		ReadOnly:     false,
		SecurityOpts: []string{"no-new-privileges"},

		// 运行时配置
		AutoRemove:    false, // 我们手动管理清理
		Timeout:       je.config.DefaultTimeout,
		RestartPolicy: "no",

		// 健康检查
		HealthCheck: &docker.HealthCheckConfig{
			Test:        []string{"CMD-SHELL", "echo 'healthy'"},
			Interval:    30 * time.Second,
			Timeout:     5 * time.Second,
			Retries:     3,
			StartPeriod: 10 * time.Second,
		},
	}

	return config, nil
}

// buildExecutionCommands 构建执行命令
func (je *jobExecutor) buildExecutionCommands(job *models.Job) []string {
	var commands []string

	// 基础设置命令
	commands = append(commands, "/bin/bash", "-c")

	// 构建脚本内容
	scriptParts := []string{
		"set -e", // 遇到错误立即退出
		"echo '=== 开始执行作业: " + job.Name + " ==='",
		"echo '作业ID: " + job.ID.String() + "'",
		"echo '开始时间: $(date)'",
		"echo '当前工作目录: $(pwd)'",
		"echo '====================================='",
	}

	// 添加作业步骤
	if job.Steps != nil && len(job.Steps) > 0 {
		for i, step := range job.Steps {
			scriptParts = append(scriptParts,
				fmt.Sprintf("echo '--- 步骤 %d: %s ---'", i+1, step.Name),
				step.Commands,
				fmt.Sprintf("echo '步骤 %d 完成'", i+1),
			)
		}
	} else {
		// 默认步骤
		scriptParts = append(scriptParts,
			"echo '执行默认构建步骤'",
			"ls -la",
			"echo '构建完成'",
		)
	}

	// 结束脚本
	scriptParts = append(scriptParts,
		"echo '====================================='",
		"echo '作业执行完成'",
		"echo '结束时间: $(date)'",
	)

	script := strings.Join(scriptParts, "\n")
	commands = append(commands, script)

	return commands
}

// buildEnvironmentVariables 构建环境变量
func (je *jobExecutor) buildEnvironmentVariables(job *models.Job) []string {
	env := []string{
		"HOME=/workspace",
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"DEBIAN_FRONTEND=noninteractive",
		fmt.Sprintf("JOB_ID=%s", job.ID.String()),
		fmt.Sprintf("JOB_NAME=%s", job.Name),
		fmt.Sprintf("PIPELINE_RUN_ID=%s", job.PipelineRunID.String()),
	}

	// 添加自定义环境变量
	if job.Config != nil {
		if envVars, ok := job.Config["environment"].(map[string]interface{}); ok {
			for key, value := range envVars {
				env = append(env, fmt.Sprintf("%s=%v", key, value))
			}
		}
	}

	return env
}

// buildVolumeBindings 构建卷绑定
func (je *jobExecutor) buildVolumeBindings(job *models.Job) map[string]string {
	volumes := make(map[string]string)

	// 工作空间卷
	workspaceDir := fmt.Sprintf("/tmp/cicd-workspaces/job-%s", job.ID.String())
	volumes[workspaceDir] = "/workspace"

	// 缓存卷
	cacheDir := fmt.Sprintf("/tmp/cicd-cache/job-%s", job.ID.String())
	volumes[cacheDir] = "/cache"

	// 添加自定义卷绑定
	if job.Config != nil {
		if volumeBindings, ok := job.Config["volumes"].(map[string]interface{}); ok {
			for hostPath, containerPath := range volumeBindings {
				if cp, ok := containerPath.(string); ok {
					volumes[hostPath] = cp
				}
			}
		}
	}

	return volumes
}

// monitorContainerExecution 监控容器执行
func (je *jobExecutor) monitorContainerExecution(ctx context.Context, job *models.Job, status *JobExecutionStatus, containerID string) error {
	// 创建监控ticker
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			je.logger.Info("作业执行被取消", zap.String("job_id", job.ID.String()))
			return je.handleExecutionCancellation(status, containerID)

		case <-ticker.C:
			// 检查容器状态
			container, err := je.dockerManager.GetContainer(ctx, containerID)
			if err != nil {
				je.logger.Error("获取容器状态失败",
					zap.String("job_id", job.ID.String()),
					zap.String("container_id", containerID),
					zap.Error(err))
				continue
			}

			// 更新资源使用情况
			if err := je.updateResourceUsage(ctx, status, containerID); err != nil {
				je.logger.Error("更新资源使用情况失败", zap.Error(err))
			}

			// 检查是否已完成
			if container.Status == "exited" {
				return je.handleContainerCompletion(status, container)
			}
		}
	}
}

// handleContainerCompletion 处理容器完成
func (je *jobExecutor) handleContainerCompletion(status *JobExecutionStatus, container *docker.Container) error {
	if container.ExitCode != nil {
		status.ExitCode = container.ExitCode

		if *container.ExitCode == 0 {
			status.Status = models.JobStatusSuccess
			je.logger.Info("作业执行成功",
				zap.String("job_id", status.JobID.String()),
				zap.String("container_id", container.ID))
		} else {
			status.Status = models.JobStatusFailed
			status.ErrorMessage = fmt.Sprintf("容器退出码: %d", *container.ExitCode)
			je.logger.Error("作业执行失败",
				zap.String("job_id", status.JobID.String()),
				zap.String("container_id", container.ID),
				zap.Int("exit_code", *container.ExitCode))
		}
	} else {
		status.Status = models.JobStatusFailed
		status.ErrorMessage = "容器异常退出，无退出码"
	}

	return nil
}

// handleExecutionError 处理执行错误
func (je *jobExecutor) handleExecutionError(status *JobExecutionStatus, err error) error {
	status.Status = models.JobStatusFailed
	status.ErrorMessage = err.Error()

	je.logger.Error("作业执行错误",
		zap.String("job_id", status.JobID.String()),
		zap.Error(err))

	return err
}

// handleExecutionCancellation 处理执行取消
func (je *jobExecutor) handleExecutionCancellation(status *JobExecutionStatus, containerID string) error {
	status.Status = models.JobStatusCancelled

	// 停止容器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := je.dockerManager.StopContainer(ctx, containerID, 10*time.Second); err != nil {
		je.logger.Error("停止容器失败", zap.Error(err))
	}

	return nil
}

// updateResourceUsage 更新资源使用情况
func (je *jobExecutor) updateResourceUsage(ctx context.Context, status *JobExecutionStatus, containerID string) error {
	stats, err := je.dockerManager.GetContainerStats(ctx, containerID)
	if err != nil {
		return err
	}

	status.Resources = &ResourceUsage{
		CPUUsage:    stats.CPUUsage,
		MemoryUsage: stats.MemoryUsage,
		MemoryLimit: stats.MemoryLimit,
		NetworkRx:   stats.NetworkRx,
		NetworkTx:   stats.NetworkTx,
		DiskRead:    stats.DiskRead,
		DiskWrite:   stats.DiskWrite,
	}

	return nil
}

// StopJob 停止作业执行
func (je *jobExecutor) StopJob(ctx context.Context, jobID uuid.UUID) error {
	je.executionsMu.RLock()
	status, exists := je.executions[jobID]
	je.executionsMu.RUnlock()

	if !exists {
		return fmt.Errorf("作业 %s 不存在", jobID.String())
	}

	if status.Status != models.JobStatusRunning {
		return fmt.Errorf("作业 %s 当前状态为 %s，无法停止", jobID.String(), status.Status)
	}

	if status.ContainerID != "" {
		return je.dockerManager.StopContainer(ctx, status.ContainerID, 10*time.Second)
	}

	return nil
}

// GetJobStatus 获取作业状态
func (je *jobExecutor) GetJobStatus(jobID uuid.UUID) (*JobExecutionStatus, error) {
	je.executionsMu.RLock()
	defer je.executionsMu.RUnlock()

	status, exists := je.executions[jobID]
	if !exists {
		return nil, fmt.Errorf("作业 %s 不存在", jobID.String())
	}

	// 返回状态副本
	statusCopy := *status
	if status.Resources != nil {
		resourcesCopy := *status.Resources
		statusCopy.Resources = &resourcesCopy
	}

	return &statusCopy, nil
}

// GetJobLogs 获取作业日志
func (je *jobExecutor) GetJobLogs(ctx context.Context, jobID uuid.UUID) (io.ReadCloser, error) {
	je.executionsMu.RLock()
	status, exists := je.executions[jobID]
	je.executionsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("作业 %s 不存在", jobID.String())
	}

	if status.ContainerID == "" {
		return nil, fmt.Errorf("作业 %s 没有关联的容器", jobID.String())
	}

	logOptions := &docker.LogOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
	}

	return je.dockerManager.GetContainerLogs(ctx, status.ContainerID, logOptions)
}

// CleanupJob 清理作业资源
func (je *jobExecutor) CleanupJob(ctx context.Context, jobID uuid.UUID) error {
	je.executionsMu.Lock()
	status, exists := je.executions[jobID]
	if exists {
		delete(je.executions, jobID)
	}
	je.executionsMu.Unlock()

	if !exists {
		return nil // 已经清理或不存在
	}

	je.logger.Info("清理作业资源", zap.String("job_id", jobID.String()))

	// 清理容器
	if status.ContainerID != "" {
		if err := je.dockerManager.RemoveContainer(ctx, status.ContainerID, true); err != nil {
			je.logger.Error("清理容器失败", zap.Error(err))
		}
	}

	// 清理工作空间
	workspaceDir := fmt.Sprintf("/tmp/cicd-workspaces/job-%s", jobID.String())
	// 这里应该清理文件系统，但为了安全，我们记录日志
	je.logger.Info("需要清理工作空间目录", zap.String("workspace", workspaceDir))

	return nil
}

// HealthCheck 健康检查
func (je *jobExecutor) HealthCheck(ctx context.Context) error {
	// 检查Docker管理器
	if err := je.dockerManager.HealthCheck(ctx); err != nil {
		return fmt.Errorf("Docker管理器健康检查失败: %v", err)
	}

	// 检查执行器状态
	if je.stopped {
		return fmt.Errorf("执行器已停止")
	}

	// 检查并发执行数量
	je.executionsMu.RLock()
	runningCount := 0
	for _, status := range je.executions {
		if status.Status == models.JobStatusRunning {
			runningCount++
		}
	}
	je.executionsMu.RUnlock()

	je.logger.Debug("健康检查通过",
		zap.Int("running_jobs", runningCount),
		zap.Int("max_concurrent", je.config.MaxConcurrentJobs))

	return nil
}

// 辅助方法

// createExecutionContext 创建执行上下文
func (je *jobExecutor) createExecutionContext(parentCtx context.Context, job *models.Job) (context.Context, context.CancelFunc) {
	timeout := je.config.DefaultTimeout

	if job.Config != nil {
		if t, ok := job.Config["timeout"].(float64); ok && t > 0 {
			timeout = time.Duration(t) * time.Second
		}
	}

	return context.WithTimeout(parentCtx, timeout)
}

// extractImageTag 提取镜像标签
func (je *jobExecutor) extractImageTag(image string) string {
	parts := strings.Split(image, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return "latest"
}

// startResourceMonitoring 启动资源监控
func (je *jobExecutor) startResourceMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-je.stopCh:
			return
		case <-ticker.C:
			je.logResourceUsage()
		}
	}
}

// logResourceUsage 记录资源使用情况
func (je *jobExecutor) logResourceUsage() {
	je.executionsMu.RLock()
	runningJobs := 0
	totalCPU := 0.0
	totalMemory := int64(0)

	for _, status := range je.executions {
		if status.Status == models.JobStatusRunning && status.Resources != nil {
			runningJobs++
			totalCPU += status.Resources.CPUUsage
			totalMemory += status.Resources.MemoryUsage
		}
	}
	je.executionsMu.RUnlock()

	if runningJobs > 0 {
		je.logger.Info("资源使用情况",
			zap.Int("running_jobs", runningJobs),
			zap.Float64("total_cpu_usage", totalCPU),
			zap.Int64("total_memory_usage", totalMemory))
	}
}

// startAutoCleanup 启动自动清理
func (je *jobExecutor) startAutoCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-je.stopCh:
			return
		case <-ticker.C:
			je.performAutoCleanup()
		}
	}
}

// performAutoCleanup 执行自动清理
func (je *jobExecutor) performAutoCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), je.config.CleanupTimeout)
	defer cancel()

	je.executionsMu.RLock()
	jobsToClean := make([]uuid.UUID, 0)

	for jobID, status := range je.executions {
		// 清理已完成超过1小时的作业
		if status.Status != models.JobStatusRunning &&
			status.EndTime != nil &&
			time.Since(*status.EndTime) > time.Hour {
			jobsToClean = append(jobsToClean, jobID)
		}
	}
	je.executionsMu.RUnlock()

	// 执行清理
	for _, jobID := range jobsToClean {
		if err := je.CleanupJob(ctx, jobID); err != nil {
			je.logger.Error("自动清理作业失败",
				zap.String("job_id", jobID.String()),
				zap.Error(err))
		}
	}

	if len(jobsToClean) > 0 {
		je.logger.Info("自动清理完成", zap.Int("cleaned_jobs", len(jobsToClean)))
	}
}

// DefaultExecutorConfig 默认执行器配置
func DefaultExecutorConfig() *ExecutorConfig {
	return &ExecutorConfig{
		DockerConfig:       docker.DefaultManagerConfig(),
		MaxConcurrentJobs:  10,
		DefaultTimeout:     30 * time.Minute,
		DefaultCPULimit:    1.0,
		DefaultMemoryLimit: 512 * 1024 * 1024,  // 512MB
		DefaultDiskLimit:   1024 * 1024 * 1024, // 1GB
		EnableAutoCleanup:  true,
		CleanupTimeout:     5 * time.Minute,
		LogRetentionDays:   7,
		StreamLogs:         true,
	}
}

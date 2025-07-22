package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/engine"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// JobScheduler 作业调度器接口
type JobScheduler interface {
	// 启动调度器
	Start(ctx context.Context) error
	
	// 停止调度器
	Stop() error
	
	// 提交作业到队列
	SubmitJob(job *ScheduleJob) error
	
	// 批量提交作业
	SubmitJobs(jobs []*ScheduleJob) error
	
	// 取消作业
	CancelJob(jobID uuid.UUID) error
	
	// 暂停/恢复调度器
	Pause() error
	Resume() error
	
	// 获取调度器状态
	GetStatus() *SchedulerStatus
	
	// 获取作业队列状态
	GetQueueStatus() *QueueStatus
	
	// 获取调度器指标
	GetMetrics() *SchedulerMetrics
	
	// 重新平衡作业负载
	RebalanceLoad() error
	
	// 设置调度策略
	SetSchedulingStrategy(strategy SchedulingStrategy) error
	
	// 获取作业历史
	GetJobHistory(limit int) ([]*JobExecution, error)
}

// ScheduleJob 调度作业
type ScheduleJob struct {
	JobID           uuid.UUID              `json:"job_id"`
	PipelineRunID   uuid.UUID              `json:"pipeline_run_id"`
	Name            string                 `json:"name"`
	Stage           string                 `json:"stage"`
	Priority        int                    `json:"priority"`
	RequiredTags    []string               `json:"required_tags"`
	CreatedAt       time.Time              `json:"created_at"`
	Config          map[string]interface{} `json:"config"`
	Dependencies    []uuid.UUID            `json:"dependencies"`    // 依赖作业ID列表
	MaxRetries      int                    `json:"max_retries"`     // 最大重试次数
	RetryCount      int                    `json:"retry_count"`     // 当前重试次数
	EstimatedDuration time.Duration        `json:"estimated_duration"` // 预估执行时间
	ResourceRequests *ResourceRequests     `json:"resource_requests"`  // 资源需求
}

// ResourceRequests 资源需求
type ResourceRequests struct {
	CPU    float64 `json:"cpu"`     // CPU核心数
	Memory int64   `json:"memory"`  // 内存字节数
	Disk   int64   `json:"disk"`    // 磁盘空间字节数
}

// SchedulingStrategy 调度策略
type SchedulingStrategy string

const (
	StrategyFIFO        SchedulingStrategy = "fifo"        // 先进先出
	StrategyPriority    SchedulingStrategy = "priority"    // 优先级调度
	StrategyLoadBalance SchedulingStrategy = "load_balance" // 负载均衡
	StrategyShortestJob SchedulingStrategy = "shortest_job" // 最短作业优先
	StrategyDeadline    SchedulingStrategy = "deadline"     // 截止时间调度
)

// SchedulerStatus 调度器状态
type SchedulerStatus struct {
	IsRunning       bool               `json:"is_running"`
	IsPaused        bool               `json:"is_paused"`
	StartedAt       time.Time          `json:"started_at"`
	ProcessedJobs   int64              `json:"processed_jobs"`
	FailedJobs      int64              `json:"failed_jobs"`
	ActiveWorkers   int                `json:"active_workers"`
	QueuedJobs      int                `json:"queued_jobs"`
	LastProcessedAt time.Time          `json:"last_processed_at"`
	Strategy        SchedulingStrategy `json:"strategy"`
	Uptime          time.Duration      `json:"uptime"`
}

// QueueStatus 队列状态
type QueueStatus struct {
	PendingJobs    int                    `json:"pending_jobs"`
	RunningJobs    int                    `json:"running_jobs"`
	HighPriority   int                    `json:"high_priority"`
	MediumPriority int                    `json:"medium_priority"`
	LowPriority    int                    `json:"low_priority"`
	QueuesByStage  map[string]int         `json:"queues_by_stage"`
	OldestJob      *time.Time             `json:"oldest_job"`
	AverageWaitTime time.Duration         `json:"average_wait_time"`
	TotalCapacity   int                    `json:"total_capacity"`
	UsedCapacity    int                    `json:"used_capacity"`
}

// SchedulerMetrics 调度器指标
type SchedulerMetrics struct {
	JobsPerSecond    float64           `json:"jobs_per_second"`
	AverageJobTime   time.Duration     `json:"average_job_time"`
	SuccessRate      float64           `json:"success_rate"`
	ThroughputByHour map[int]int64     `json:"throughput_by_hour"`
	ErrorsByType     map[string]int64  `json:"errors_by_type"`
	RunnerUtilization map[uuid.UUID]float64 `json:"runner_utilization"`
	QueueDepthHistory []QueueDepthPoint `json:"queue_depth_history"`
}

// QueueDepthPoint 队列深度历史点
type QueueDepthPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Depth     int       `json:"depth"`
}

// JobExecution 作业执行历史
type JobExecution struct {
	JobID         uuid.UUID     `json:"job_id"`
	Name          string        `json:"name"`
	Status        string        `json:"status"`
	StartedAt     *time.Time    `json:"started_at"`
	FinishedAt    *time.Time    `json:"finished_at"`
	Duration      time.Duration `json:"duration"`
	RunnerID      uuid.UUID     `json:"runner_id"`
	ExitCode      *int          `json:"exit_code"`
	RetryCount    int           `json:"retry_count"`
}

// JobAssignment 作业分配结果
type JobAssignment struct {
	JobID    uuid.UUID `json:"job_id"`
	RunnerID uuid.UUID `json:"runner_id"`
	AssignedAt time.Time `json:"assigned_at"`
}

// RunnerCommunicator Runner通信接口（避免循环依赖）
type RunnerCommunicator interface {
	SendJobToRunner(runnerID uuid.UUID, job *JobMessage) error
	GetOnlineRunners() []uuid.UUID
}

// JobMessage 作业消息
type JobMessage struct {
	Type      string                 `json:"type"`
	JobID     uuid.UUID              `json:"job_id"`
	Name      string                 `json:"name"`
	Commands  []string               `json:"commands"`
	Env       map[string]string      `json:"env"`
	Timeout   int                    `json:"timeout"`
	Workspace string                 `json:"workspace"`
	Config    map[string]interface{} `json:"config"`
}

// jobScheduler 作业调度器实现
type jobScheduler struct {
	repo              repository.PipelineRepository
	engine            engine.PipelineEngine
	runnerComm        RunnerCommunicator
	logger            *zap.Logger
	
	// 调度配置
	config   SchedulerConfig
	strategy SchedulingStrategy
	
	// 状态管理
	mu        sync.RWMutex
	isRunning bool
	isPaused  bool
	startedAt time.Time
	
	// 多级队列和优先级处理
	priorityQueues  map[int]chan *ScheduleJob  // 优先级队列
	dependencyGraph map[uuid.UUID][]uuid.UUID // 依赖关系图
	readyJobs       chan *ScheduleJob          // 准备就绪的作业
	workers         []*worker
	assignments     chan *JobAssignment
	
	// 统计和指标数据
	processedJobs     int64
	failedJobs        int64
	cancelledJobs     int64
	lastProcessedAt   time.Time
	jobExecutions     []*JobExecution  // 作业执行历史
	queueDepthHistory []QueueDepthPoint // 队列深度历史
	metricsHistory    map[time.Time]*SchedulerMetrics // 指标历史
	
	// 负载均衡和容量管理
	runnerCapacity    map[uuid.UUID]*ResourceRequests // Runner容量
	runnerUtilization map[uuid.UUID]float64           // Runner利用率
	
	// 停止信号和控制
	stopCh   chan struct{}
	doneCh   chan struct{}
	pauseCh  chan struct{}
	resumeCh chan struct{}
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	WorkerCount         int                `json:"worker_count"`
	QueueSize           int                `json:"queue_size"`
	PollInterval        time.Duration      `json:"poll_interval"`
	JobTimeout          time.Duration      `json:"job_timeout"`
	MaxRetries          int                `json:"max_retries"`
	EnablePriority      bool               `json:"enable_priority"`
	EnableLoadBalance   bool               `json:"enable_load_balance"`
	EnableDependency    bool               `json:"enable_dependency"`    // 启用依赖管理
	MaxHistorySize      int                `json:"max_history_size"`     // 历史记录最大数量
	MetricsInterval     time.Duration      `json:"metrics_interval"`     // 指标收集间隔
	QueueDepthInterval  time.Duration      `json:"queue_depth_interval"` // 队列深度收集间隔
	DefaultStrategy     SchedulingStrategy `json:"default_strategy"`     // 默认调度策略
	MaxConcurrentJobs   int                `json:"max_concurrent_jobs"`  // 最大并发作业数
	HealthCheckInterval time.Duration      `json:"health_check_interval"` // 健康检查间隔
}

// worker 工作器
type worker struct {
	id        int
	scheduler *jobScheduler
	jobCh     chan *ScheduleJob
	stopCh    chan struct{}
	logger    *zap.Logger
}

// DefaultSchedulerConfig 默认调度器配置
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		WorkerCount:         5,
		QueueSize:           1000,
		PollInterval:        10 * time.Second,
		JobTimeout:          30 * time.Minute,
		MaxRetries:          3,
		EnablePriority:      true,
		EnableLoadBalance:   true,
		EnableDependency:    true,
		MaxHistorySize:      1000,
		MetricsInterval:     1 * time.Minute,
		QueueDepthInterval:  10 * time.Second,
		DefaultStrategy:     StrategyPriority,
		MaxConcurrentJobs:   50,
		HealthCheckInterval: 30 * time.Second,
	}
}

// NewJobScheduler 创建作业调度器
func NewJobScheduler(repo repository.PipelineRepository, engine engine.PipelineEngine, config SchedulerConfig, logger *zap.Logger) JobScheduler {
	scheduler := &jobScheduler{
		repo:     repo,
		engine:   engine,
		config:   config,
		strategy: config.DefaultStrategy,
		logger:   logger,
		
		// 初始化多级优先级队列
		priorityQueues:    make(map[int]chan *ScheduleJob),
		dependencyGraph:   make(map[uuid.UUID][]uuid.UUID),
		readyJobs:         make(chan *ScheduleJob, config.QueueSize),
		assignments:       make(chan *JobAssignment, config.QueueSize),
		jobExecutions:     make([]*JobExecution, 0, config.MaxHistorySize),
		queueDepthHistory: make([]QueueDepthPoint, 0, config.MaxHistorySize),
		metricsHistory:    make(map[time.Time]*SchedulerMetrics),
		runnerCapacity:    make(map[uuid.UUID]*ResourceRequests),
		runnerUtilization: make(map[uuid.UUID]float64),
		
		// 控制通道
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		pauseCh:  make(chan struct{}),
		resumeCh: make(chan struct{}),
	}
	
	// 初始化优先级队列（1-10级）
	for priority := 1; priority <= 10; priority++ {
		scheduler.priorityQueues[priority] = make(chan *ScheduleJob, config.QueueSize/10)
	}
	
	return scheduler
}

// SetRunnerCommunicator 设置Runner通信器（避免循环依赖）
func (s *jobScheduler) SetRunnerCommunicator(comm RunnerCommunicator) {
	s.runnerComm = comm
}

// Start 启动调度器
func (s *jobScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.isRunning {
		return fmt.Errorf("调度器已在运行")
	}
	
	s.logger.Info("启动作业调度器", 
		zap.Int("worker_count", s.config.WorkerCount),
		zap.Int("queue_size", s.config.QueueSize))
	
	s.isRunning = true
	s.startedAt = time.Now()
	
	// 创建工作器
	s.workers = make([]*worker, s.config.WorkerCount)
	for i := 0; i < s.config.WorkerCount; i++ {
		s.workers[i] = &worker{
			id:        i,
			scheduler: s,
			jobCh:     make(chan *ScheduleJob, 1),
			stopCh:    make(chan struct{}),
			logger:    s.logger.With(zap.Int("worker_id", i)),
		}
		
		go s.workers[i].run()
	}
	
	// 启动主调度循环
	go s.scheduleLoop(ctx)
	
	// 启动作业发现循环
	go s.jobDiscoveryLoop(ctx)
	
	// 启动统计更新循环
	go s.statisticsLoop(ctx)
	
	// 启动指标收集循环
	go s.metricsCollectionLoop(ctx)
	
	// 启动队列深度监控循环
	go s.queueDepthMonitorLoop(ctx)
	
	// 启动依赖解析循环
	if s.config.EnableDependency {
		go s.dependencyResolutionLoop(ctx)
	}
	
	s.logger.Info("作业调度器启动成功")
	return nil
}

// Stop 停止调度器
func (s *jobScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isRunning {
		return fmt.Errorf("调度器未运行")
	}
	
	s.logger.Info("停止作业调度器")
	
	// 发送停止信号
	close(s.stopCh)
	
	// 停止所有工作器
	for _, worker := range s.workers {
		close(worker.stopCh)
	}
	
	// 等待调度器完全停止
	<-s.doneCh
	
	s.isRunning = false
	s.logger.Info("作业调度器已停止")
	
	return nil
}

// SubmitJobs 批量提交作业到队列
func (s *jobScheduler) SubmitJobs(jobs []*ScheduleJob) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if !s.isRunning {
		return fmt.Errorf("调度器未运行")
	}
	
	successCount := 0
	for _, job := range jobs {
		if err := s.submitJobInternal(job); err != nil {
			s.logger.Error("批量提交作业失败", 
				zap.String("job_id", job.JobID.String()),
				zap.Error(err))
		} else {
			successCount++
		}
	}
	
	s.logger.Info("批量提交作业完成", 
		zap.Int("total", len(jobs)),
		zap.Int("success", successCount))
	
	if successCount == 0 {
		return fmt.Errorf("所有作业提交失败")
	}
	
	return nil
}

// CancelJob 取消作业
func (s *jobScheduler) CancelJob(jobID uuid.UUID) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if !s.isRunning {
		return fmt.Errorf("调度器未运行")
	}
	
	ctx := context.Background()
	
	// 更新数据库中的作业状态
	updates := map[string]interface{}{
		"status":      models.JobStatusCancelled,
		"finished_at": time.Now().UTC(),
		"log_output":  "作业被用户取消",
	}
	
	if err := s.repo.UpdateJob(ctx, jobID, updates); err != nil {
		return fmt.Errorf("取消作业失败: %v", err)
	}
	
	s.cancelledJobs++
	s.logger.Info("作业已取消", zap.String("job_id", jobID.String()))
	
	return nil
}

// Pause 暂停调度器
func (s *jobScheduler) Pause() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isRunning {
		return fmt.Errorf("调度器未运行")
	}
	
	if s.isPaused {
		return fmt.Errorf("调度器已暂停")
	}
	
	s.isPaused = true
	s.logger.Info("调度器已暂停")
	
	return nil
}

// Resume 恢复调度器
func (s *jobScheduler) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isRunning {
		return fmt.Errorf("调度器未运行")
	}
	
	if !s.isPaused {
		return fmt.Errorf("调度器未暂停")
	}
	
	s.isPaused = false
	s.logger.Info("调度器已恢复")
	
	return nil
}

// GetMetrics 获取调度器指标
func (s *jobScheduler) GetMetrics() *SchedulerMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 计算作业处理率
	uptime := time.Since(s.startedAt)
	jobsPerSecond := float64(s.processedJobs) / uptime.Seconds()
	
	// 计算成功率
	totalJobs := s.processedJobs + s.failedJobs + s.cancelledJobs
	successRate := float64(s.processedJobs) / float64(totalJobs)
	if totalJobs == 0 {
		successRate = 0
	}
	
	// 计算平均作业时间
	var totalDuration time.Duration
	var jobCount int
	for _, execution := range s.jobExecutions {
		if execution.Duration > 0 {
			totalDuration += execution.Duration
			jobCount++
		}
	}
	
	var averageJobTime time.Duration
	if jobCount > 0 {
		averageJobTime = totalDuration / time.Duration(jobCount)
	}
	
	return &SchedulerMetrics{
		JobsPerSecond:     jobsPerSecond,
		AverageJobTime:    averageJobTime,
		SuccessRate:       successRate,
		ThroughputByHour:  make(map[int]int64), // TODO: 实现小时吞吐量统计
		ErrorsByType:      make(map[string]int64), // TODO: 实现错误类型统计
		RunnerUtilization: s.runnerUtilization,
		QueueDepthHistory: s.queueDepthHistory,
	}
}

// RebalanceLoad 重新平衡作业负载
func (s *jobScheduler) RebalanceLoad() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isRunning {
		return fmt.Errorf("调度器未运行")
	}
	
	s.logger.Info("开始重新平衡作业负载")
	
	// TODO: 实现负载重平衡逻辑
	// 1. 分析当前Runner负载情况
	// 2. 重新分配排队中的作业
	// 3. 优化作业分配策略
	
	s.logger.Info("作业负载重平衡完成")
	
	return nil
}

// SetSchedulingStrategy 设置调度策略
func (s *jobScheduler) SetSchedulingStrategy(strategy SchedulingStrategy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.strategy = strategy
	s.logger.Info("调度策略已更新", zap.String("strategy", string(strategy)))
	
	return nil
}

// GetJobHistory 获取作业历史
func (s *jobScheduler) GetJobHistory(limit int) ([]*JobExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if limit <= 0 || limit > len(s.jobExecutions) {
		limit = len(s.jobExecutions)
	}
	
	// 返回最新的limit条记录
	start := len(s.jobExecutions) - limit
	if start < 0 {
		start = 0
	}
	
	result := make([]*JobExecution, limit)
	copy(result, s.jobExecutions[start:])
	
	return result, nil
}

// SubmitJob 提交作业到队列
func (s *jobScheduler) SubmitJob(job *ScheduleJob) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if !s.isRunning {
		return fmt.Errorf("调度器未运行")
	}
	
	if s.isPaused {
		return fmt.Errorf("调度器已暂停")
	}
	
	return s.submitJobInternal(job)
}

// GetStatus 获取调度器状态
func (s *jobScheduler) GetStatus() *SchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 计算总的排队作业数
	totalQueuedJobs := len(s.readyJobs)
	for _, queue := range s.priorityQueues {
		totalQueuedJobs += len(queue)
	}
	
	uptime := time.Duration(0)
	if s.isRunning {
		uptime = time.Since(s.startedAt)
	}
	
	return &SchedulerStatus{
		IsRunning:       s.isRunning,
		IsPaused:        s.isPaused,
		StartedAt:       s.startedAt,
		ProcessedJobs:   s.processedJobs,
		FailedJobs:      s.failedJobs,
		ActiveWorkers:   len(s.workers),
		QueuedJobs:      totalQueuedJobs,
		LastProcessedAt: s.lastProcessedAt,
		Strategy:        s.strategy,
		Uptime:          uptime,
	}
}

// GetQueueStatus 获取作业队列状态
func (s *jobScheduler) GetQueueStatus() *QueueStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 计算不同优先级的作业数量
	highPriority := 0
	mediumPriority := 0
	lowPriority := 0
	
	for priority, queue := range s.priorityQueues {
		switch {
		case priority >= 8:
			highPriority += len(queue)
		case priority >= 5:
			mediumPriority += len(queue)
		default:
			lowPriority += len(queue)
		}
	}
	
	// 计算不同阶段的作业分布
	queuesByStage := map[string]int{
		"ready": len(s.readyJobs),
	}
	
	// 计算总容量和已用容量
	totalCapacity := s.config.MaxConcurrentJobs
	usedCapacity := len(s.workers) // 简化计算
	
	// 计算平均等待时间
	var averageWaitTime time.Duration
	// TODO: 实现基于历史数据的平均等待时间计算
	
	// 查找最早的作业时间
	var oldestJob *time.Time
	// TODO: 实现查找队列中最早作业的逻辑
	
	totalPending := highPriority + mediumPriority + lowPriority + len(s.readyJobs)
	
	return &QueueStatus{
		PendingJobs:     totalPending,
		RunningJobs:     usedCapacity,
		HighPriority:    highPriority,
		MediumPriority:  mediumPriority,
		LowPriority:     lowPriority,
		QueuesByStage:   queuesByStage,
		OldestJob:       oldestJob,
		AverageWaitTime: averageWaitTime,
		TotalCapacity:   totalCapacity,
		UsedCapacity:    usedCapacity,
	}
}

// 私有方法

// scheduleLoop 主调度循环
func (s *jobScheduler) scheduleLoop(ctx context.Context) {
	defer close(s.doneCh)
	
	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case job := <-s.readyJobs:
			// 检查是否暂停
			if !s.isPaused {
				s.scheduleJob(ctx, job)
			} else {
				// 暂停状态下，将作业重新放回队列
				time.AfterFunc(time.Second, func() {
					s.readyJobs <- job
				})
			}
		case <-ticker.C:
			// 定期检查优先级队列并移动作业到就绪队列
			s.processPriorityQueues(ctx)
			// 定期检查是否有新的作业需要调度
			s.checkPendingJobs(ctx)
		}
	}
}

// jobDiscoveryLoop 作业发现循环
func (s *jobScheduler) jobDiscoveryLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.discoverPendingJobs(ctx)
		}
	}
}

// statisticsLoop 统计更新循环
func (s *jobScheduler) statisticsLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.updateStatistics()
		}
	}
}

// scheduleJob 调度单个作业
func (s *jobScheduler) scheduleJob(ctx context.Context, job *ScheduleJob) {
	logger := s.logger.With(
		zap.String("job_id", job.JobID.String()),
		zap.String("name", job.Name))
	
	logger.Debug("开始调度作业")
	
	// 查找合适的执行器
	runners, err := s.repo.GetAvailableRunners(ctx, job.RequiredTags)
	if err != nil {
		logger.Error("获取可用执行器失败", zap.Error(err))
		s.failedJobs++
		return
	}
	
	if len(runners) == 0 {
		logger.Debug("暂无可用执行器，稍后重试")
		// 重新放入队列
		time.AfterFunc(time.Second*30, func() {
			s.SubmitJob(job)
		})
		return
	}
	
	// 选择最佳执行器
	runner := s.selectBestRunner(runners, job)
	
	// 分配作业给工作器
	s.assignJobToWorker(job, &runner)
	
	logger.Info("作业已分配", zap.String("runner_id", runner.ID.String()))
	s.processedJobs++
	s.lastProcessedAt = time.Now()
}

// selectBestRunner 选择最佳执行器
func (s *jobScheduler) selectBestRunner(runners []models.Runner, job *ScheduleJob) models.Runner {
	if !s.config.EnableLoadBalance {
		return runners[0]
	}
	
	// 简单的负载均衡算法：选择空闲时间最长的执行器
	var bestRunner models.Runner
	var oldestContact time.Time
	
	for _, runner := range runners {
		if runner.LastContactAt == nil {
			return runner // 返回从未使用过的执行器
		}
		
		if bestRunner.ID == uuid.Nil || runner.LastContactAt.Before(oldestContact) {
			bestRunner = runner
			oldestContact = *runner.LastContactAt
		}
	}
	
	return bestRunner
}

// assignJobToWorker 分配作业给工作器
func (s *jobScheduler) assignJobToWorker(job *ScheduleJob, runner *models.Runner) {
	// 找到空闲的工作器
	for _, worker := range s.workers {
		select {
		case worker.jobCh <- job:
			// 记录分配
			assignment := &JobAssignment{
				JobID:      job.JobID,
				RunnerID:   runner.ID,
				AssignedAt: time.Now(),
			}
			
			select {
			case s.assignments <- assignment:
			default:
				// 分配通道满了，忽略
			}
			return
		default:
			// 工作器忙碌，尝试下一个
			continue
		}
	}
	
	// 所有工作器都忙碌，重新放入队列
	time.AfterFunc(time.Second*10, func() {
		s.SubmitJob(job)
	})
}

// checkPendingJobs 检查待处理作业
func (s *jobScheduler) checkPendingJobs(ctx context.Context) {
	jobs, err := s.repo.GetPendingJobs(ctx, nil)
	if err != nil {
		s.logger.Error("获取待处理作业失败", zap.Error(err))
		return
	}
	
	for _, job := range jobs {
		scheduleJob := &ScheduleJob{
			JobID:         job.ID,
			PipelineRunID: job.PipelineRunID,
			Name:          job.Name,
			Stage:         string(job.Type), // 使用Job类型作为Stage
			Priority:      1, // 默认优先级
			RequiredTags:  nil,
			CreatedAt:     job.CreatedAt,
			Config:        make(map[string]interface{}),
		}
		
		// 提交到队列
		select {
		case s.readyJobs <- scheduleJob:
		default:
			// 队列满了，下次再处理
			break
		}
	}
}

// discoverPendingJobs 发现待处理作业
func (s *jobScheduler) discoverPendingJobs(ctx context.Context) {
	// 从数据库发现新的待处理作业
	jobs, err := s.repo.GetPendingJobs(ctx, nil)
	if err != nil {
		s.logger.Error("发现待处理作业失败", zap.Error(err))
		return
	}
	
	for _, job := range jobs {
		scheduleJob := &ScheduleJob{
			JobID:         job.ID,
			PipelineRunID: job.PipelineRunID,
			Name:          job.Name,
			Stage:         string(job.Type), // 使用Job类型作为Stage
			Priority:      1,
			RequiredTags:  nil,
			CreatedAt:     job.CreatedAt,
			Config:        make(map[string]interface{}),
		}
		
		// 尝试提交到队列
		if err := s.SubmitJob(scheduleJob); err != nil {
			// 队列满了，下次再试
			break
		}
	}
}

// updateStatistics 更新统计信息
func (s *jobScheduler) updateStatistics() {
	// 这里可以添加更复杂的统计逻辑
	s.logger.Debug("更新调度器统计信息",
		zap.Int64("processed_jobs", s.processedJobs),
		zap.Int64("failed_jobs", s.failedJobs),
		zap.Int("queued_jobs", len(s.readyJobs)))
}

// worker运行方法

// run 工作器运行方法
func (w *worker) run() {
	w.logger.Debug("启动工作器")
	defer w.logger.Debug("工作器已停止")
	
	for {
		select {
		case <-w.stopCh:
			return
		case job := <-w.jobCh:
			w.processJob(job)
		}
	}
}

// processJob 处理作业
func (w *worker) processJob(job *ScheduleJob) {
	logger := w.logger.With(
		zap.String("job_id", job.JobID.String()),
		zap.String("name", job.Name))
	
	logger.Debug("开始处理作业")
	
	ctx, cancel := context.WithTimeout(context.Background(), w.scheduler.config.JobTimeout)
	defer cancel()
	
	// 更新作业状态为运行中
	updates := map[string]interface{}{
		"status":     models.JobStatusRunning,
		"started_at": time.Now().UTC(),
	}
	
	if err := w.scheduler.repo.UpdateJob(ctx, job.JobID, updates); err != nil {
		logger.Error("更新作业状态失败", zap.Error(err))
		w.scheduler.failedJobs++
		return
	}
	
	// 获取作业详情（暂时未使用详细信息）
	_, err := w.scheduler.repo.GetJobByID(ctx, job.JobID)
	if err != nil {
		logger.Error("获取作业详情失败", zap.Error(err))
		w.scheduler.failedJobs++
		return
	}
	
	// 通过Runner通信管理器发送作业
	if w.scheduler.runnerComm != nil {
		// 查找分配的Runner
		runners, err := w.scheduler.repo.GetAvailableRunners(ctx, job.RequiredTags)
		if err != nil || len(runners) == 0 {
			logger.Error("无可用Runner", zap.Error(err))
			w.scheduler.failedJobs++
			return
		}
		
		runner := runners[0] // 选择第一个可用Runner
		
		// 创建作业消息
		jobMessage := &JobMessage{
			Type:      "job_start",
			JobID:     job.JobID,
			Name:      job.Name,
			Commands:  []string{"echo 'Job started'", "sleep 2", "echo 'Job completed'"}, // 默认命令
			Env:       map[string]string{"CI": "true", "RUNNER_ID": runner.ID.String()},  // 默认环境变量
			Timeout:   int(w.scheduler.config.JobTimeout.Seconds()),
			Workspace: "/workspace",
			Config:    job.Config,
		}
		
		// 发送作业到Runner
		if err := w.scheduler.runnerComm.SendJobToRunner(runner.ID, jobMessage); err != nil {
			logger.Error("发送作业到Runner失败", zap.Error(err))
			w.scheduler.failedJobs++
			
			// 更新作业状态为失败
			failUpdates := map[string]interface{}{
				"status":      models.JobStatusFailed,
				"finished_at": time.Now().UTC(),
				"log_output":  fmt.Sprintf("发送作业到Runner失败: %v", err),
			}
			w.scheduler.repo.UpdateJob(ctx, job.JobID, failUpdates)
			return
		}
		
		logger.Info("作业已发送到Runner", zap.String("runner_id", runner.ID.String()))
		// 注意：作业完成状态将由Runner通信管理器处理
		return
	}
	
	// 降级处理：如果没有Runner通信管理器，模拟执行
	time.Sleep(2 * time.Second)
	
	// 模拟作业完成
	finishedAt := time.Now().UTC()
	finalUpdates := map[string]interface{}{
		"status":      models.JobStatusSuccess,
		"finished_at": finishedAt,
		"exit_code":   0,
		"log_output":  "作业执行成功（模拟）",
	}
	
	if err := w.scheduler.repo.UpdateJob(ctx, job.JobID, finalUpdates); err != nil {
		logger.Error("更新作业完成状态失败", zap.Error(err))
		w.scheduler.failedJobs++
		return
	}
	
	logger.Info("作业处理完成")
}

// 新增的辅助方法

// submitJobInternal 内部作业提交方法
func (s *jobScheduler) submitJobInternal(job *ScheduleJob) error {
	// 根据调度策略选择合适的队列
	switch s.strategy {
	case StrategyPriority:
		return s.submitToPriorityQueue(job)
	case StrategyFIFO:
		return s.submitToReadyQueue(job)
	case StrategyLoadBalance:
		return s.submitWithLoadBalance(job)
	default:
		return s.submitToReadyQueue(job)
	}
}

// submitToPriorityQueue 提交到优先级队列
func (s *jobScheduler) submitToPriorityQueue(job *ScheduleJob) error {
	priority := job.Priority
	if priority < 1 {
		priority = 1
	} else if priority > 10 {
		priority = 10
	}
	
	queue, exists := s.priorityQueues[priority]
	if !exists {
		return fmt.Errorf("优先级队列不存在: %d", priority)
	}
	
	select {
	case queue <- job:
		s.logger.Debug("作业已提交到优先级队列", 
			zap.String("job_id", job.JobID.String()),
			zap.Int("priority", priority))
		return nil
	default:
		return fmt.Errorf("优先级队列已满: %d", priority)
	}
}

// submitToReadyQueue 提交到就绪队列
func (s *jobScheduler) submitToReadyQueue(job *ScheduleJob) error {
	select {
	case s.readyJobs <- job:
		s.logger.Debug("作业已提交到就绪队列", 
			zap.String("job_id", job.JobID.String()))
		return nil
	default:
		return fmt.Errorf("就绪队列已满")
	}
}

// submitWithLoadBalance 负载均衡提交
func (s *jobScheduler) submitWithLoadBalance(job *ScheduleJob) error {
	// 选择负载最轻的队列
	minLoad := int(^uint(0) >> 1) // 最大int值
	bestPriority := 5 // 默认中等优先级
	
	for priority, queue := range s.priorityQueues {
		if len(queue) < minLoad {
			minLoad = len(queue)
			bestPriority = priority
		}
	}
	
	// 更新作业优先级并提交
	job.Priority = bestPriority
	return s.submitToPriorityQueue(job)
}

// metricsCollectionLoop 指标收集循环
func (s *jobScheduler) metricsCollectionLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.collectMetrics()
		}
	}
}

// collectMetrics 收集指标
func (s *jobScheduler) collectMetrics() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	metrics := s.GetMetrics()
	s.metricsHistory[time.Now()] = metrics
	
	// 清理过期的指标历史
	cutoff := time.Now().Add(-24 * time.Hour) // 保留24小时历史
	for timestamp := range s.metricsHistory {
		if timestamp.Before(cutoff) {
			delete(s.metricsHistory, timestamp)
		}
	}
	
	s.logger.Debug("指标收集完成",
		zap.Float64("jobs_per_second", metrics.JobsPerSecond),
		zap.Float64("success_rate", metrics.SuccessRate))
}

// queueDepthMonitorLoop 队列深度监控循环
func (s *jobScheduler) queueDepthMonitorLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.QueueDepthInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.recordQueueDepth()
		}
	}
}

// recordQueueDepth 记录队列深度
func (s *jobScheduler) recordQueueDepth() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 计算总队列深度
	totalDepth := len(s.readyJobs)
	for _, queue := range s.priorityQueues {
		totalDepth += len(queue)
	}
	
	// 记录深度点
	point := QueueDepthPoint{
		Timestamp: time.Now(),
		Depth:     totalDepth,
	}
	
	s.queueDepthHistory = append(s.queueDepthHistory, point)
	
	// 保持历史记录大小限制
	if len(s.queueDepthHistory) > s.config.MaxHistorySize {
		s.queueDepthHistory = s.queueDepthHistory[1:]
	}
}

// dependencyResolutionLoop 依赖解析循环
func (s *jobScheduler) dependencyResolutionLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second) // 每5秒检查一次依赖
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.resolveDependencies()
		}
	}
}

// resolveDependencies 解析依赖关系
func (s *jobScheduler) resolveDependencies() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// TODO: 实现依赖解析逻辑
	// 1. 检查作业依赖是否已完成
	// 2. 将满足依赖条件的作业移至就绪队列
	// 3. 处理循环依赖检测
	
	s.logger.Debug("依赖解析检查完成")
}

// addJobExecution 添加作业执行历史
func (s *jobScheduler) addJobExecution(execution *JobExecution) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.jobExecutions = append(s.jobExecutions, execution)
	
	// 保持历史记录大小限制
	if len(s.jobExecutions) > s.config.MaxHistorySize {
		s.jobExecutions = s.jobExecutions[1:]
	}
}

// processPriorityQueues 处理优先级队列
func (s *jobScheduler) processPriorityQueues(ctx context.Context) {
	// 按优先级从高到低处理队列（10到1）
	for priority := 10; priority >= 1; priority-- {
		queue, exists := s.priorityQueues[priority]
		if !exists {
			continue
		}
		
		// 尝试从优先级队列移动作业到就绪队列
		select {
		case job := <-queue:
			// 检查作业是否满足依赖条件（如果启用依赖管理）
			if s.config.EnableDependency && !s.areDependenciesSatisfied(job) {
				// 依赖未满足，重新放回队列
				time.AfterFunc(time.Second*5, func() {
					queue <- job
				})
				continue
			}
			
			// 尝试提交到就绪队列
			select {
			case s.readyJobs <- job:
				s.logger.Debug("作业从优先级队列移动到就绪队列",
					zap.String("job_id", job.JobID.String()),
					zap.Int("priority", priority))
			default:
				// 就绪队列满了，重新放回优先级队列
				queue <- job
			}
		default:
			// 该优先级队列为空，检查下一个
			continue
		}
		
		// 每次只处理一个作业，确保公平性
		break
	}
}

// areDependenciesSatisfied 检查作业依赖是否满足
func (s *jobScheduler) areDependenciesSatisfied(job *ScheduleJob) bool {
	if len(job.Dependencies) == 0 {
		return true
	}
	
	ctx := context.Background()
	for _, depJobID := range job.Dependencies {
		depJob, err := s.repo.GetJobByID(ctx, depJobID)
		if err != nil {
			s.logger.Error("检查依赖作业失败", 
				zap.String("job_id", job.JobID.String()),
				zap.String("dep_job_id", depJobID.String()),
				zap.Error(err))
			return false
		}
		
		// 依赖作业必须成功完成
		if depJob.Status != models.JobStatusSuccess {
			return false
		}
	}
	
	return true
}
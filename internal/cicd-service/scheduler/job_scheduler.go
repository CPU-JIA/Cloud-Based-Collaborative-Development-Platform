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
	
	// 获取调度器状态
	GetStatus() *SchedulerStatus
	
	// 获取作业队列状态
	GetQueueStatus() *QueueStatus
}

// ScheduleJob 调度作业
type ScheduleJob struct {
	JobID         uuid.UUID              `json:"job_id"`
	PipelineRunID uuid.UUID              `json:"pipeline_run_id"`
	Name          string                 `json:"name"`
	Stage         string                 `json:"stage"`
	Priority      int                    `json:"priority"`
	RequiredTags  []string               `json:"required_tags"`
	CreatedAt     time.Time              `json:"created_at"`
	Config        map[string]interface{} `json:"config"`
}

// SchedulerStatus 调度器状态
type SchedulerStatus struct {
	IsRunning       bool      `json:"is_running"`
	StartedAt       time.Time `json:"started_at"`
	ProcessedJobs   int64     `json:"processed_jobs"`
	FailedJobs      int64     `json:"failed_jobs"`
	ActiveWorkers   int       `json:"active_workers"`
	QueuedJobs      int       `json:"queued_jobs"`
	LastProcessedAt time.Time `json:"last_processed_at"`
}

// QueueStatus 队列状态
type QueueStatus struct {
	PendingJobs   int                    `json:"pending_jobs"`
	RunningJobs   int                    `json:"running_jobs"`
	HighPriority  int                    `json:"high_priority"`
	MediumPriority int                   `json:"medium_priority"`
	LowPriority   int                    `json:"low_priority"`
	QueuesByStage map[string]int         `json:"queues_by_stage"`
	OldestJob     *time.Time             `json:"oldest_job"`
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
	config SchedulerConfig
	
	// 状态管理
	mu         sync.RWMutex
	isRunning  bool
	startedAt  time.Time
	
	// 队列和统计
	jobQueue        chan *ScheduleJob
	workers         []*worker
	assignments     chan *JobAssignment
	
	// 统计数据
	processedJobs   int64
	failedJobs      int64
	lastProcessedAt time.Time
	
	// 停止信号
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	WorkerCount        int           `json:"worker_count"`
	QueueSize          int           `json:"queue_size"`
	PollInterval       time.Duration `json:"poll_interval"`
	JobTimeout         time.Duration `json:"job_timeout"`
	MaxRetries         int           `json:"max_retries"`
	EnablePriority     bool          `json:"enable_priority"`
	EnableLoadBalance  bool          `json:"enable_load_balance"`
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
		WorkerCount:       5,
		QueueSize:         1000,
		PollInterval:      10 * time.Second,
		JobTimeout:        30 * time.Minute,
		MaxRetries:        3,
		EnablePriority:    true,
		EnableLoadBalance: true,
	}
}

// NewJobScheduler 创建作业调度器
func NewJobScheduler(repo repository.PipelineRepository, engine engine.PipelineEngine, config SchedulerConfig, logger *zap.Logger) JobScheduler {
	return &jobScheduler{
		repo:        repo,
		engine:      engine,
		config:      config,
		logger:      logger,
		jobQueue:    make(chan *ScheduleJob, config.QueueSize),
		assignments: make(chan *JobAssignment, config.QueueSize),
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}
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

// SubmitJob 提交作业到队列
func (s *jobScheduler) SubmitJob(job *ScheduleJob) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if !s.isRunning {
		return fmt.Errorf("调度器未运行")
	}
	
	select {
	case s.jobQueue <- job:
		s.logger.Debug("作业已提交到队列", 
			zap.String("job_id", job.JobID.String()),
			zap.String("name", job.Name))
		return nil
	default:
		return fmt.Errorf("作业队列已满")
	}
}

// GetStatus 获取调度器状态
func (s *jobScheduler) GetStatus() *SchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return &SchedulerStatus{
		IsRunning:       s.isRunning,
		StartedAt:       s.startedAt,
		ProcessedJobs:   s.processedJobs,
		FailedJobs:      s.failedJobs,
		ActiveWorkers:   len(s.workers),
		QueuedJobs:      len(s.jobQueue),
		LastProcessedAt: s.lastProcessedAt,
	}
}

// GetQueueStatus 获取作业队列状态
func (s *jobScheduler) GetQueueStatus() *QueueStatus {
	// 这里应该查询数据库获取实际的队列状态
	// 暂时返回模拟数据
	return &QueueStatus{
		PendingJobs:    len(s.jobQueue),
		RunningJobs:    0,
		HighPriority:   0,
		MediumPriority: 0,
		LowPriority:    len(s.jobQueue),
		QueuesByStage:  map[string]int{"build": len(s.jobQueue)},
		OldestJob:      nil,
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
		case job := <-s.jobQueue:
			s.scheduleJob(ctx, job)
		case <-ticker.C:
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
			Stage:         job.Stage,
			Priority:      1, // 默认优先级
			RequiredTags:  nil,
			CreatedAt:     job.CreatedAt,
			Config:        make(map[string]interface{}),
		}
		
		// 提交到队列
		select {
		case s.jobQueue <- scheduleJob:
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
			Stage:         job.Stage,
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
		zap.Int("queued_jobs", len(s.jobQueue)))
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
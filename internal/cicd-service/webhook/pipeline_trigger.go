package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/scheduler"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PipelineTrigger 流水线触发器
type PipelineTrigger struct {
	repo      repository.PipelineRepository
	scheduler scheduler.JobScheduler
	logger    *zap.Logger
	config    PipelineTriggerConfig
}

// PipelineTriggerConfig 流水线触发器配置
type PipelineTriggerConfig struct {
	DefaultTimeout    time.Duration `yaml:"default_timeout"`
	MaxConcurrentJobs int           `yaml:"max_concurrent_jobs"`
	RetryAttempts     int           `yaml:"retry_attempts"`
	RetryInterval     time.Duration `yaml:"retry_interval"`
}

// NewPipelineTrigger 创建流水线触发器
func NewPipelineTrigger(repo repository.PipelineRepository, scheduler scheduler.JobScheduler, config PipelineTriggerConfig, logger *zap.Logger) *PipelineTrigger {
	return &PipelineTrigger{
		repo:      repo,
		scheduler: scheduler,
		config:    config,
		logger:    logger,
	}
}

// TriggerPipelineFromWebhook 从Webhook事件触发流水线
func (pt *PipelineTrigger) TriggerPipelineFromWebhook(ctx context.Context, repositoryID, pipelineID uuid.UUID, variables map[string]interface{}) error {
	pt.logger.Info("从Webhook触发流水线",
		zap.String("repository_id", repositoryID.String()),
		zap.String("pipeline_id", pipelineID.String()))

	// 获取流水线定义
	pipeline, err := pt.repo.GetPipelineByID(ctx, pipelineID)
	if err != nil {
		return fmt.Errorf("获取流水线定义失败: %w", err)
	}

	// 验证流水线是否启用
	if !pipeline.IsActive {
		return fmt.Errorf("流水线已禁用")
	}

	// 创建流水线运行
	run, err := pt.createPipelineRun(ctx, pipeline, repositoryID, variables)
	if err != nil {
		return fmt.Errorf("创建流水线运行失败: %w", err)
	}

	// 创建并提交作业
	jobs, err := pt.createJobsFromPipeline(pipeline, run, variables)
	if err != nil {
		return fmt.Errorf("创建作业失败: %w", err)
	}

	// 批量提交作业到调度器
	if err := pt.scheduler.SubmitJobs(jobs); err != nil {
		// 如果提交失败，更新运行状态
		pt.updateRunStatus(ctx, run.ID, models.PipelineStatusFailed, fmt.Sprintf("提交作业失败: %v", err))
		return fmt.Errorf("提交作业到调度器失败: %w", err)
	}

	pt.logger.Info("流水线触发成功",
		zap.String("pipeline_run_id", run.ID.String()),
		zap.Int("job_count", len(jobs)))

	return nil
}

// createPipelineRun 创建流水线运行
func (pt *PipelineTrigger) createPipelineRun(ctx context.Context, pipeline *models.Pipeline, repositoryID uuid.UUID, variables map[string]interface{}) (*models.PipelineRun, error) {
	// 构建运行变量
	runVariables := make(map[string]interface{})

	// 添加系统变量
	runVariables["PIPELINE_ID"] = pipeline.ID.String()
	runVariables["PIPELINE_NAME"] = pipeline.Name
	runVariables["REPOSITORY_ID"] = repositoryID.String()
	runVariables["RUN_ID"] = uuid.New().String()
	runVariables["TRIGGERED_BY"] = "webhook"
	runVariables["RUN_TIMESTAMP"] = time.Now().Unix()

	// 注意：Pipeline模型中没有Variables字段，暂时跳过
	// for k, v := range pipeline.Variables {
	//	runVariables[k] = v
	// }

	// 合并Webhook传入的变量（优先级最高）
	for k, v := range variables {
		runVariables[k] = v
	}

	// 转换变量为正确的类型
	stringVariables := make(map[string]string)
	for k, v := range runVariables {
		if str, ok := v.(string); ok {
			stringVariables[k] = str
		} else {
			stringVariables[k] = fmt.Sprintf("%v", v)
		}
	}

	// 创建运行记录
	run := &models.PipelineRun{
		PipelineID:  pipeline.ID,
		Status:      models.PipelineStatusPending,
		Variables:   stringVariables,
		TriggerType: models.TriggerTypeWebhook,
		CreatedAt:   time.Now().UTC(),
	}

	// 保存到数据库
	if err := pt.repo.CreatePipelineRun(ctx, run); err != nil {
		return nil, fmt.Errorf("保存流水线运行失败: %w", err)
	}

	return run, nil
}

// createJobsFromPipeline 从流水线定义创建作业
func (pt *PipelineTrigger) createJobsFromPipeline(pipeline *models.Pipeline, run *models.PipelineRun, variables map[string]interface{}) ([]*scheduler.ScheduleJob, error) {
	var jobs []*scheduler.ScheduleJob

	// 创建默认的构建作业
	buildJob := &scheduler.ScheduleJob{
		JobID:             uuid.New(),
		PipelineRunID:     run.ID,
		Name:              fmt.Sprintf("%s-build", pipeline.Name),
		Stage:             "build",
		Priority:          5, // 普通优先级
		RequiredTags:      []string{},
		CreatedAt:         time.Now(),
		Config:            make(map[string]interface{}),
		Dependencies:      []uuid.UUID{},
		MaxRetries:        3,
		RetryCount:        0,
		EstimatedDuration: 300 * time.Second, // 5分钟默认估计
		ResourceRequests: &scheduler.ResourceRequests{
			CPU:    1.0,
			Memory: 512 * 1024 * 1024,  // 512MB
			Disk:   1024 * 1024 * 1024, // 1GB
		},
	}

	jobs = append(jobs, buildJob)

	// 创建测试作业
	testJob := &scheduler.ScheduleJob{
		JobID:             uuid.New(),
		PipelineRunID:     run.ID,
		Name:              fmt.Sprintf("%s-test", pipeline.Name),
		Stage:             "test",
		Priority:          5,
		RequiredTags:      []string{},
		CreatedAt:         time.Now(),
		Config:            make(map[string]interface{}),
		Dependencies:      []uuid.UUID{buildJob.JobID}, // 依赖构建作业
		MaxRetries:        2,
		RetryCount:        0,
		EstimatedDuration: 600 * time.Second, // 10分钟默认估计
		ResourceRequests: &scheduler.ResourceRequests{
			CPU:    1.0,
			Memory: 1024 * 1024 * 1024, // 1GB
			Disk:   1024 * 1024 * 1024, // 1GB
		},
	}

	jobs = append(jobs, testJob)

	return jobs, nil
}

// parsePipelineConfig 解析流水线配置
func (pt *PipelineTrigger) parsePipelineConfig(config map[string]interface{}) (*PipelineConfig, error) {
	// 简化的流水线配置解析
	pipelineConfig := &PipelineConfig{
		Stages: []StageConfig{},
	}

	stages, ok := config["stages"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("流水线配置中缺少stages字段")
	}

	for _, stageData := range stages {
		stageMap, ok := stageData.(map[string]interface{})
		if !ok {
			continue
		}

		stage := StageConfig{
			Name: getString(stageMap, "name", "unknown"),
			Jobs: []JobConfig{},
		}

		jobs, ok := stageMap["jobs"].([]interface{})
		if ok {
			for _, jobData := range jobs {
				jobMap, ok := jobData.(map[string]interface{})
				if !ok {
					continue
				}

				job := JobConfig{
					Name:         getString(jobMap, "name", "unknown"),
					Image:        getString(jobMap, "image", ""),
					Commands:     getStringArray(jobMap, "commands"),
					Environment:  getStringMap(jobMap, "environment"),
					Timeout:      getInt(jobMap, "timeout", int(pt.config.DefaultTimeout.Seconds())),
					Resources:    getMap(jobMap, "resources"),
					Dependencies: getStringArray(jobMap, "dependencies"),
					Conditions:   getMap(jobMap, "conditions"),
				}

				stage.Jobs = append(stage.Jobs, job)
			}
		}

		pipelineConfig.Stages = append(pipelineConfig.Stages, stage)
	}

	return pipelineConfig, nil
}

// generateJobName 生成作业名称
func (pt *PipelineTrigger) generateJobName(pipelineName, stageName, jobName string, stageIndex, jobIndex int) string {
	return fmt.Sprintf("%s-%s-%s-%d-%d", pipelineName, stageName, jobName, stageIndex, jobIndex)
}

// calculateJobPriority 计算作业优先级
func (pt *PipelineTrigger) calculateJobPriority(stage StageConfig, job JobConfig) int {
	// 默认优先级为5
	priority := 5

	// 根据阶段调整优先级
	switch stage.Name {
	case "build":
		priority = 7
	case "test":
		priority = 6
	case "deploy":
		priority = 8
	case "cleanup":
		priority = 3
	}

	// 根据作业配置调整优先级
	if p, ok := job.Conditions["priority"].(int); ok {
		priority = p
	}

	// 确保优先级在1-10范围内
	if priority < 1 {
		priority = 1
	} else if priority > 10 {
		priority = 10
	}

	return priority
}

// extractRequiredTags 提取必需的标签
func (pt *PipelineTrigger) extractRequiredTags(job JobConfig) []string {
	tags := []string{}

	// 从作业配置中提取标签
	if tagsInterface, ok := job.Conditions["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	// 根据作业镜像添加默认标签
	if job.Image != "" {
		if contains(job.Image, "node") {
			tags = append(tags, "node")
		} else if contains(job.Image, "python") {
			tags = append(tags, "python")
		} else if contains(job.Image, "go") || contains(job.Image, "golang") {
			tags = append(tags, "golang")
		} else if contains(job.Image, "docker") {
			tags = append(tags, "docker")
		}
	}

	return tags
}

// mergeJobConfig 合并作业配置
func (pt *PipelineTrigger) mergeJobConfig(job JobConfig, variables map[string]interface{}) map[string]interface{} {
	config := map[string]interface{}{
		"name":        job.Name,
		"image":       job.Image,
		"commands":    job.Commands,
		"environment": job.Environment,
		"timeout":     job.Timeout,
		"resources":   job.Resources,
	}

	// 合并变量到环境变量
	if config["environment"] == nil {
		config["environment"] = make(map[string]string)
	}

	env := config["environment"].(map[string]string)
	for k, v := range variables {
		if str, ok := v.(string); ok {
			env[k] = str
		} else {
			env[k] = fmt.Sprintf("%v", v)
		}
	}

	return config
}

// resolveDependencies 解析作业依赖
func (pt *PipelineTrigger) resolveDependencies(existingJobs []*scheduler.ScheduleJob, stage StageConfig, job JobConfig) []uuid.UUID {
	var dependencies []uuid.UUID

	// 同一阶段内的依赖
	for _, depName := range job.Dependencies {
		for _, existingJob := range existingJobs {
			if existingJob.Stage == stage.Name && contains(existingJob.Name, depName) {
				dependencies = append(dependencies, existingJob.JobID)
				break
			}
		}
	}

	return dependencies
}

// getMaxRetries 获取最大重试次数
func (pt *PipelineTrigger) getMaxRetries(job JobConfig) int {
	if retries, ok := job.Conditions["max_retries"].(int); ok {
		return retries
	}
	return pt.config.RetryAttempts
}

// estimateJobDuration 估算作业执行时间
func (pt *PipelineTrigger) estimateJobDuration(job JobConfig) time.Duration {
	// 根据作业类型估算执行时间
	if duration, ok := job.Conditions["estimated_duration"].(int); ok {
		return time.Duration(duration) * time.Second
	}

	// 根据命令数量估算
	baseTime := 2 * time.Minute
	commandTime := time.Duration(len(job.Commands)) * 30 * time.Second

	return baseTime + commandTime
}

// parseResourceRequests 解析资源需求
func (pt *PipelineTrigger) parseResourceRequests(job JobConfig) *scheduler.ResourceRequests {
	requests := &scheduler.ResourceRequests{
		CPU:    1.0,                // 默认1核
		Memory: 512 * 1024 * 1024,  // 默认512MB
		Disk:   1024 * 1024 * 1024, // 默认1GB
	}

	if resources, ok := job.Resources["requests"].(map[string]interface{}); ok {
		if cpu, ok := resources["cpu"].(float64); ok {
			requests.CPU = cpu
		}
		if memory, ok := resources["memory"].(int64); ok {
			requests.Memory = memory
		}
		if disk, ok := resources["disk"].(int64); ok {
			requests.Disk = disk
		}
	}

	return requests
}

// updateRunStatus 更新运行状态
func (pt *PipelineTrigger) updateRunStatus(ctx context.Context, runID uuid.UUID, status models.PipelineStatus, message string) {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}

	if message != "" {
		updates["error_message"] = message
	}

	if err := pt.repo.UpdatePipelineRun(ctx, runID, updates); err != nil {
		pt.logger.Error("更新流水线运行状态失败",
			zap.String("run_id", runID.String()),
			zap.Error(err))
	}
}

// 配置结构体

type PipelineConfig struct {
	Stages []StageConfig `json:"stages"`
}

type StageConfig struct {
	Name string      `json:"name"`
	Jobs []JobConfig `json:"jobs"`
}

type JobConfig struct {
	Name         string                 `json:"name"`
	Image        string                 `json:"image"`
	Commands     []string               `json:"commands"`
	Environment  map[string]string      `json:"environment"`
	Timeout      int                    `json:"timeout"`
	Resources    map[string]interface{} `json:"resources"`
	Dependencies []string               `json:"dependencies"`
	Conditions   map[string]interface{} `json:"conditions"`
}

// 辅助函数

func getString(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string, defaultValue int) int {
	if v, ok := m[key].(int); ok {
		return v
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return defaultValue
}

func getStringArray(m map[string]interface{}, key string) []string {
	if v, ok := m[key].([]interface{}); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return []string{}
}

func getStringMap(m map[string]interface{}, key string) map[string]string {
	if v, ok := m[key].(map[string]interface{}); ok {
		result := make(map[string]string)
		for k, val := range v {
			if s, ok := val.(string); ok {
				result[k] = s
			}
		}
		return result
	}
	return make(map[string]string)
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return make(map[string]interface{})
}

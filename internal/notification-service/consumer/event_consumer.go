package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/notification-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/notification-service/services"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/segmentio/kafka-go"
)

// EventConsumer Kafka事件消费者
type EventConsumer struct {
	reader             *kafka.Reader
	notificationService *services.NotificationService
	logger             logger.Logger
	ctx                context.Context
	cancel             context.CancelFunc
}

// NewEventConsumer 创建新的事件消费者
func NewEventConsumer(kafkaConfig KafkaConfig, notificationService *services.NotificationService, appLogger logger.Logger) *EventConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        kafkaConfig.Brokers,
		GroupID:        kafkaConfig.GroupID,
		Topic:          kafkaConfig.Topic,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	ctx, cancel := context.WithCancel(context.Background())

	return &EventConsumer{
		reader:              reader,
		notificationService: notificationService,
		logger:              appLogger,
		ctx:                 ctx,
		cancel:              cancel,
	}
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
	GroupID string   `yaml:"group_id"`
	Topic   string   `yaml:"topic"`
}

// Start 启动事件消费
func (ec *EventConsumer) Start() error {
	ec.logger.Info("Starting Kafka event consumer...")

	go func() {
		for {
			select {
			case <-ec.ctx.Done():
				ec.logger.Info("Event consumer stopped")
				return
			default:
				message, err := ec.reader.ReadMessage(ec.ctx)
				if err != nil {
					ec.logger.Error("Error reading message from Kafka:", err)
					continue
				}

				ec.logger.Info(fmt.Sprintf("Received message: %s", string(message.Value)))

				// 处理消息
				if err := ec.processMessage(message); err != nil {
					ec.logger.Error("Error processing message:", err)
					// 记录失败消息，可以考虑发送到死信队列
					continue
				}
			}
		}
	}()

	return nil
}

// Stop 停止事件消费
func (ec *EventConsumer) Stop() error {
	ec.logger.Info("Stopping Kafka event consumer...")
	ec.cancel()

	if err := ec.reader.Close(); err != nil {
		return fmt.Errorf("failed to close Kafka reader: %w", err)
	}

	return nil
}

// processMessage 处理单个消息
func (ec *EventConsumer) processMessage(message kafka.Message) error {
	// 解析事件
	var event models.Event
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// 记录事件接收日志
	ec.logger.Info(fmt.Sprintf("Processing event: type=%s, source=%s, tenant_id=%s", 
		event.Type, event.Source, event.TenantID))

	// 根据事件类型进行处理
	switch event.Type {
	// 项目事件
	case models.EventTypeTaskAssigned:
		return ec.handleTaskAssignedEvent(event)
	case models.EventTypeTaskCompleted:
		return ec.handleTaskCompletedEvent(event)
	case models.EventTypeTaskOverdue:
		return ec.handleTaskOverdueEvent(event)
	case models.EventTypeSprintStarted:
		return ec.handleSprintStartedEvent(event)
	case models.EventTypeSprintCompleted:
		return ec.handleSprintCompletedEvent(event)
	case models.EventTypeMilestoneReached:
		return ec.handleMilestoneReachedEvent(event)

	// CI/CD事件
	case models.EventTypeBuildStarted:
		return ec.handleBuildStartedEvent(event)
	case models.EventTypeBuildCompleted:
		return ec.handleBuildCompletedEvent(event)
	case models.EventTypeBuildFailed:
		return ec.handleBuildFailedEvent(event)
	case models.EventTypeDeploymentStarted:
		return ec.handleDeploymentStartedEvent(event)
	case models.EventTypeDeploymentCompleted:
		return ec.handleDeploymentCompletedEvent(event)
	case models.EventTypeDeploymentFailed:
		return ec.handleDeploymentFailedEvent(event)

	// Git事件
	case models.EventTypePullRequestOpened:
		return ec.handlePullRequestOpenedEvent(event)
	case models.EventTypePullRequestMerged:
		return ec.handlePullRequestMergedEvent(event)
	case models.EventTypeCodeReviewRequested:
		return ec.handleCodeReviewRequestedEvent(event)
	case models.EventTypeCodeReviewCompleted:
		return ec.handleCodeReviewCompletedEvent(event)

	// 系统事件
	case models.EventTypeSystemAlert:
		return ec.handleSystemAlertEvent(event)
	case models.EventTypeSecurityAlert:
		return ec.handleSecurityAlertEvent(event)
	case models.EventTypeUserCreated:
		return ec.handleUserCreatedEvent(event)
	case models.EventTypePermissionChanged:
		return ec.handlePermissionChangedEvent(event)

	default:
		ec.logger.Warn(fmt.Sprintf("Unknown event type: %s", event.Type))
		return nil
	}
}

// 任务相关事件处理器

func (ec *EventConsumer) handleTaskAssignedEvent(event models.Event) error {
	var eventData models.TaskAssignedEventData
	if err := json.Unmarshal(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to unmarshal task assigned event data: %w", err)
	}

	// 创建通知请求
	request := &services.CreateNotificationRequest{
		UserID:      &eventData.AssigneeID,
		TenantID:    event.TenantID,
		ProjectID:   &eventData.ProjectID,
		Type:        models.EventTypeTaskAssigned,
		Category:    models.CategoryProject,
		Priority:    ec.getPriorityFromTaskPriority(eventData.Priority),
		EventData:   event.Data,
		TemplateID:  nil, // 将由服务自动选择模板
		Channels: &models.Channels{
			Email: &models.EmailChannel{
				Enabled: true,
				To:      []string{}, // 将由服务查询用户邮箱
				Subject: fmt.Sprintf("新任务分配：%s", eventData.TaskTitle),
			},
			InApp: &models.InAppChannel{
				Enabled: true,
				Badge:   true,
				Popup:   true,
			},
		},
		CorrelationID: event.CorrelationID,
	}

	return ec.notificationService.CreateNotification(ec.ctx, request)
}

func (ec *EventConsumer) handleTaskCompletedEvent(event models.Event) error {
	var eventData models.TaskCompletedEventData
	if err := json.Unmarshal(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to unmarshal task completed event data: %w", err)
	}

	// 通知项目相关人员任务完成
	request := &services.CreateNotificationRequest{
		UserID:    &eventData.CompletedByID,
		TenantID:  event.TenantID,
		ProjectID: &eventData.ProjectID,
		Type:      models.EventTypeTaskCompleted,
		Category:  models.CategoryProject,
		Priority:  models.PriorityMedium,
		EventData: event.Data,
		Channels: &models.Channels{
			InApp: &models.InAppChannel{
				Enabled: true,
				Badge:   true,
			},
		},
		CorrelationID: event.CorrelationID,
	}

	return ec.notificationService.CreateNotification(ec.ctx, request)
}

func (ec *EventConsumer) handleTaskOverdueEvent(event models.Event) error {
	// TODO: 实现任务超期事件处理
	ec.logger.Info("Task overdue event received - handler to be implemented")
	return nil
}

// Sprint相关事件处理器

func (ec *EventConsumer) handleSprintStartedEvent(event models.Event) error {
	var eventData models.SprintStartedEventData
	if err := json.Unmarshal(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to unmarshal sprint started event data: %w", err)
	}

	// 通知所有团队成员Sprint开始
	for _, member := range eventData.TeamMembers {
		request := &services.CreateNotificationRequest{
			UserID:    &member.UserID,
			TenantID:  event.TenantID,
			ProjectID: &eventData.ProjectID,
			Type:      models.EventTypeSprintStarted,
			Category:  models.CategoryProject,
			Priority:  models.PriorityHigh,
			EventData: event.Data,
			Channels: &models.Channels{
				Email: &models.EmailChannel{
					Enabled: true,
					To:      []string{member.Email},
					Subject: fmt.Sprintf("Sprint开始：%s - %s", eventData.SprintName, eventData.ProjectName),
				},
				InApp: &models.InAppChannel{
					Enabled: true,
					Badge:   true,
					Popup:   true,
				},
			},
			CorrelationID: event.CorrelationID,
		}

		if err := ec.notificationService.CreateNotification(ec.ctx, request); err != nil {
			ec.logger.Error(fmt.Sprintf("Failed to create notification for user %s: %v", member.UserID, err))
		}
	}

	return nil
}

func (ec *EventConsumer) handleSprintCompletedEvent(event models.Event) error {
	// TODO: 实现Sprint完成事件处理
	ec.logger.Info("Sprint completed event received - handler to be implemented")
	return nil
}

func (ec *EventConsumer) handleMilestoneReachedEvent(event models.Event) error {
	// TODO: 实现里程碑达成事件处理
	ec.logger.Info("Milestone reached event received - handler to be implemented")
	return nil
}

// CI/CD相关事件处理器

func (ec *EventConsumer) handleBuildStartedEvent(event models.Event) error {
	// TODO: 实现构建开始事件处理
	ec.logger.Info("Build started event received - handler to be implemented")
	return nil
}

func (ec *EventConsumer) handleBuildCompletedEvent(event models.Event) error {
	var eventData models.BuildCompletedEventData
	if err := json.Unmarshal(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to unmarshal build completed event data: %w", err)
	}

	// 根据构建结果确定优先级
	priority := models.PriorityMedium
	if eventData.Status == "failed" {
		priority = models.PriorityHigh
	}

	request := &services.CreateNotificationRequest{
		TenantID:  event.TenantID,
		ProjectID: &eventData.ProjectID,
		Type:      models.EventTypeBuildCompleted,
		Category:  models.CategoryCICD,
		Priority:  priority,
		EventData: event.Data,
		Channels: &models.Channels{
			InApp: &models.InAppChannel{
				Enabled: true,
				Badge:   true,
			},
		},
		CorrelationID: event.CorrelationID,
	}

	return ec.notificationService.CreateNotification(ec.ctx, request)
}

func (ec *EventConsumer) handleBuildFailedEvent(event models.Event) error {
	// TODO: 实现构建失败事件处理
	ec.logger.Info("Build failed event received - handler to be implemented")
	return nil
}

func (ec *EventConsumer) handleDeploymentStartedEvent(event models.Event) error {
	// TODO: 实现部署开始事件处理
	ec.logger.Info("Deployment started event received - handler to be implemented")
	return nil
}

func (ec *EventConsumer) handleDeploymentCompletedEvent(event models.Event) error {
	// TODO: 实现部署完成事件处理
	ec.logger.Info("Deployment completed event received - handler to be implemented")
	return nil
}

func (ec *EventConsumer) handleDeploymentFailedEvent(event models.Event) error {
	// TODO: 实现部署失败事件处理
	ec.logger.Info("Deployment failed event received - handler to be implemented")
	return nil
}

// Git相关事件处理器

func (ec *EventConsumer) handlePullRequestOpenedEvent(event models.Event) error {
	// TODO: 实现Pull Request创建事件处理
	ec.logger.Info("Pull request opened event received - handler to be implemented")
	return nil
}

func (ec *EventConsumer) handlePullRequestMergedEvent(event models.Event) error {
	// TODO: 实现Pull Request合并事件处理
	ec.logger.Info("Pull request merged event received - handler to be implemented")
	return nil
}

func (ec *EventConsumer) handleCodeReviewRequestedEvent(event models.Event) error {
	// TODO: 实现代码审查请求事件处理
	ec.logger.Info("Code review requested event received - handler to be implemented")
	return nil
}

func (ec *EventConsumer) handleCodeReviewCompletedEvent(event models.Event) error {
	// TODO: 实现代码审查完成事件处理
	ec.logger.Info("Code review completed event received - handler to be implemented")
	return nil
}

// 系统相关事件处理器

func (ec *EventConsumer) handleSystemAlertEvent(event models.Event) error {
	var eventData models.SystemAlertEventData
	if err := json.Unmarshal(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to unmarshal system alert event data: %w", err)
	}

	// 系统告警通常需要立即通知管理员
	request := &services.CreateNotificationRequest{
		TenantID: event.TenantID,
		Type:     models.EventTypeSystemAlert,
		Category: models.CategorySystem,
		Priority: ec.getPriorityFromSeverity(eventData.Severity),
		EventData: event.Data,
		Channels: &models.Channels{
			Email: &models.EmailChannel{
				Enabled: true,
				Subject: fmt.Sprintf("[%s] 系统告警：%s", eventData.Severity, eventData.Title),
			},
			InApp: &models.InAppChannel{
				Enabled: true,
				Badge:   true,
				Popup:   true,
			},
		},
		CorrelationID: event.CorrelationID,
	}

	return ec.notificationService.CreateNotification(ec.ctx, request)
}

func (ec *EventConsumer) handleSecurityAlertEvent(event models.Event) error {
	var eventData models.SecurityEventData
	if err := json.Unmarshal(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to unmarshal security alert event data: %w", err)
	}

	// 安全告警优先级高，需要立即通知
	request := &services.CreateNotificationRequest{
		UserID:   eventData.UserID,
		TenantID: event.TenantID,
		Type:     models.EventTypeSecurityAlert,
		Category: models.CategorySecurity,
		Priority: models.PriorityHigh,
		EventData: event.Data,
		Channels: &models.Channels{
			Email: &models.EmailChannel{
				Enabled: true,
				Subject: fmt.Sprintf("[安全告警] %s", eventData.Description),
			},
			InApp: &models.InAppChannel{
				Enabled: true,
				Badge:   true,
				Popup:   true,
				Sound:   true,
			},
		},
		CorrelationID: event.CorrelationID,
	}

	return ec.notificationService.CreateNotification(ec.ctx, request)
}

func (ec *EventConsumer) handleUserCreatedEvent(event models.Event) error {
	// TODO: 实现用户创建事件处理
	ec.logger.Info("User created event received - handler to be implemented")
	return nil
}

func (ec *EventConsumer) handlePermissionChangedEvent(event models.Event) error {
	// TODO: 实现权限变更事件处理
	ec.logger.Info("Permission changed event received - handler to be implemented")
	return nil
}

// 辅助方法

func (ec *EventConsumer) getPriorityFromTaskPriority(taskPriority string) string {
	switch taskPriority {
	case "低", "low":
		return models.PriorityLow
	case "中", "medium", "normal":
		return models.PriorityMedium
	case "高", "high":
		return models.PriorityHigh
	case "紧急", "urgent", "critical":
		return models.PriorityCritical
	default:
		return models.PriorityMedium
	}
}

func (ec *EventConsumer) getPriorityFromSeverity(severity string) string {
	switch severity {
	case "low":
		return models.PriorityLow
	case "medium":
		return models.PriorityMedium
	case "high":
		return models.PriorityHigh
	case "critical":
		return models.PriorityCritical
	default:
		return models.PriorityMedium
	}
}
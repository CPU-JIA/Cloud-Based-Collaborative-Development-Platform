package repository

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

// WebhookRepository Webhook数据访问接口
type WebhookRepository interface {
	// Webhook事件管理
	CreateWebhookEvent(ctx context.Context, event *models.WebhookEvent) error
	GetWebhookEvent(ctx context.Context, eventID uuid.UUID) (*models.WebhookEvent, error)
	ListWebhookEvents(ctx context.Context, filter *models.WebhookEventFilter, page, pageSize int) (*models.WebhookEventListResponse, error)
	UpdateWebhookEvent(ctx context.Context, eventID uuid.UUID, updates map[string]interface{}) error
	DeleteWebhookEvent(ctx context.Context, eventID uuid.UUID) error

	// 触发器管理
	CreateWebhookTrigger(ctx context.Context, trigger *models.WebhookTrigger) error
	GetWebhookTrigger(ctx context.Context, triggerID uuid.UUID) (*models.WebhookTrigger, error)
	ListWebhookTriggers(ctx context.Context, repositoryID *uuid.UUID, page, pageSize int) (*models.WebhookTriggerListResponse, error)
	UpdateWebhookTrigger(ctx context.Context, triggerID uuid.UUID, updates map[string]interface{}) error
	DeleteWebhookTrigger(ctx context.Context, triggerID uuid.UUID) error

	// 投递管理
	CreateWebhookDelivery(ctx context.Context, delivery *models.WebhookDelivery) error
	GetWebhookDelivery(ctx context.Context, deliveryID uuid.UUID) (*models.WebhookDelivery, error)
	ListWebhookDeliveries(ctx context.Context, webhookID *uuid.UUID, page, pageSize int) (*models.WebhookDeliveryListResponse, error)
	UpdateWebhookDelivery(ctx context.Context, deliveryID uuid.UUID, updates map[string]interface{}) error

	// 统计信息
	GetWebhookStatistics(ctx context.Context, repositoryID *uuid.UUID) (*models.WebhookStatistics, error)
}

// webhookRepository Webhook数据访问实现
type webhookRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewWebhookRepository 创建Webhook数据访问实例
func NewWebhookRepository(db *sqlx.DB, logger *zap.Logger) WebhookRepository {
	return &webhookRepository{
		db:     db,
		logger: logger,
	}
}

// JSONB 类型支持
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONB", value)
	}
	return json.Unmarshal(bytes, j)
}

// StringArray 类型支持
type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	return pq.Array(s).Value()
}

func (s *StringArray) Scan(value interface{}) error {
	return pq.Array(s).Scan(value)
}

// CreateWebhookEvent 创建Webhook事件
func (r *webhookRepository) CreateWebhookEvent(ctx context.Context, event *models.WebhookEvent) error {
	query := `
		INSERT INTO webhook_events (
			id, repository_id, event_type, event_data, source, 
			signature, processed, processed_at, error_message, 
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	eventDataJSON, err := json.Marshal(event.EventData)
	if err != nil {
		return fmt.Errorf("序列化事件数据失败: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		event.ID, event.RepositoryID, event.EventType, eventDataJSON,
		event.Source, event.Signature, event.Processed, event.ProcessedAt,
		event.ErrorMessage, event.CreatedAt, event.UpdatedAt)

	if err != nil {
		return fmt.Errorf("插入Webhook事件失败: %w", err)
	}

	return nil
}

// GetWebhookEvent 获取Webhook事件
func (r *webhookRepository) GetWebhookEvent(ctx context.Context, eventID uuid.UUID) (*models.WebhookEvent, error) {
	query := `
		SELECT id, repository_id, event_type, event_data, source,
			   signature, processed, processed_at, error_message,
			   created_at, updated_at
		FROM webhook_events
		WHERE id = $1`

	var event models.WebhookEvent
	var eventDataJSON []byte

	err := r.db.QueryRowContext(ctx, query, eventID).Scan(
		&event.ID, &event.RepositoryID, &event.EventType, &eventDataJSON,
		&event.Source, &event.Signature, &event.Processed, &event.ProcessedAt,
		&event.ErrorMessage, &event.CreatedAt, &event.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("查询Webhook事件失败: %w", err)
	}

	// 解析事件数据
	if len(eventDataJSON) > 0 {
		if err := json.Unmarshal(eventDataJSON, &event.EventData); err != nil {
			return nil, fmt.Errorf("解析事件数据失败: %w", err)
		}
	}

	return &event, nil
}

// ListWebhookEvents 列出Webhook事件
func (r *webhookRepository) ListWebhookEvents(ctx context.Context, filter *models.WebhookEventFilter, page, pageSize int) (*models.WebhookEventListResponse, error) {
	// 构建WHERE条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter != nil {
		if filter.RepositoryID != nil {
			whereClause += fmt.Sprintf(" AND repository_id = $%d", argIndex)
			args = append(args, *filter.RepositoryID)
			argIndex++
		}

		if len(filter.EventTypes) > 0 {
			whereClause += fmt.Sprintf(" AND event_type = ANY($%d)", argIndex)
			args = append(args, pq.Array(filter.EventTypes))
			argIndex++
		}

		if filter.Source != "" {
			whereClause += fmt.Sprintf(" AND source = $%d", argIndex)
			args = append(args, filter.Source)
			argIndex++
		}

		if filter.Processed != nil {
			whereClause += fmt.Sprintf(" AND processed = $%d", argIndex)
			args = append(args, *filter.Processed)
			argIndex++
		}

		if filter.StartTime != nil {
			whereClause += fmt.Sprintf(" AND created_at >= $%d", argIndex)
			args = append(args, *filter.StartTime)
			argIndex++
		}

		if filter.EndTime != nil {
			whereClause += fmt.Sprintf(" AND created_at <= $%d", argIndex)
			args = append(args, *filter.EndTime)
			argIndex++
		}
	}

	// 计算总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM webhook_events %s", whereClause)
	var total int64
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("计算事件总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	listQuery := fmt.Sprintf(`
		SELECT id, repository_id, event_type, event_data, source,
			   signature, processed, processed_at, error_message,
			   created_at, updated_at
		FROM webhook_events 
		%s 
		ORDER BY created_at DESC 
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("查询事件列表失败: %w", err)
	}
	defer rows.Close()

	var events []models.WebhookEvent
	for rows.Next() {
		var event models.WebhookEvent
		var eventDataJSON []byte

		err := rows.Scan(
			&event.ID, &event.RepositoryID, &event.EventType, &eventDataJSON,
			&event.Source, &event.Signature, &event.Processed, &event.ProcessedAt,
			&event.ErrorMessage, &event.CreatedAt, &event.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("扫描事件数据失败: %w", err)
		}

		// 解析事件数据
		if len(eventDataJSON) > 0 {
			if err := json.Unmarshal(eventDataJSON, &event.EventData); err != nil {
				r.logger.Warn("解析事件数据失败", zap.Error(err))
				event.EventData = make(map[string]interface{})
			}
		}

		events = append(events, event)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &models.WebhookEventListResponse{
		Events:     events,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateWebhookEvent 更新Webhook事件
func (r *webhookRepository) UpdateWebhookEvent(ctx context.Context, eventID uuid.UUID, updates map[string]interface{}) error {
	// 定义允许更新的列白名单
	allowedColumns := map[string]bool{
		"processed":      true,
		"processed_at":   true,
		"error_message":  true,
		"updated_at":     true,
		"event_data":     true,
		"signature":      true,
	}

	// 构建SET子句
	setClause := ""
	args := []interface{}{}
	argIndex := 1

	for column, value := range updates {
		// 验证列名是否在白名单中
		if !allowedColumns[column] {
			return fmt.Errorf("不允许更新的列: %s", column)
		}

		if argIndex > 1 {
			setClause += ", "
		}
		setClause += fmt.Sprintf("%s = $%d", column, argIndex)
		args = append(args, value)
		argIndex++
	}

	if setClause == "" {
		return fmt.Errorf("没有要更新的字段")
	}

	// 添加WHERE条件
	query := fmt.Sprintf("UPDATE webhook_events SET %s WHERE id = $%d", setClause, argIndex)
	args = append(args, eventID)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("更新Webhook事件失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("事件不存在")
	}

	return nil
}

// DeleteWebhookEvent 删除Webhook事件
func (r *webhookRepository) DeleteWebhookEvent(ctx context.Context, eventID uuid.UUID) error {
	query := "DELETE FROM webhook_events WHERE id = $1"

	result, err := r.db.ExecContext(ctx, query, eventID)
	if err != nil {
		return fmt.Errorf("删除Webhook事件失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("事件不存在")
	}

	return nil
}

// CreateWebhookTrigger 创建钩子触发器
func (r *webhookRepository) CreateWebhookTrigger(ctx context.Context, trigger *models.WebhookTrigger) error {
	query := `
		INSERT INTO webhook_triggers (
			id, repository_id, name, event_types, conditions, 
			actions, enabled, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	eventTypesJSON, err := json.Marshal(trigger.EventTypes)
	if err != nil {
		return fmt.Errorf("序列化事件类型失败: %w", err)
	}

	conditionsJSON, err := json.Marshal(trigger.Conditions)
	if err != nil {
		return fmt.Errorf("序列化触发条件失败: %w", err)
	}

	actionsJSON, err := json.Marshal(trigger.Actions)
	if err != nil {
		return fmt.Errorf("序列化触发动作失败: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		trigger.ID, trigger.RepositoryID, trigger.Name, eventTypesJSON,
		conditionsJSON, actionsJSON, trigger.Enabled, trigger.CreatedAt, trigger.UpdatedAt)

	if err != nil {
		return fmt.Errorf("插入钩子触发器失败: %w", err)
	}

	return nil
}

// GetWebhookTrigger 获取钩子触发器
func (r *webhookRepository) GetWebhookTrigger(ctx context.Context, triggerID uuid.UUID) (*models.WebhookTrigger, error) {
	query := `
		SELECT id, repository_id, name, event_types, conditions,
			   actions, enabled, created_at, updated_at
		FROM webhook_triggers
		WHERE id = $1`

	var trigger models.WebhookTrigger
	var eventTypesJSON, conditionsJSON, actionsJSON []byte

	err := r.db.QueryRowContext(ctx, query, triggerID).Scan(
		&trigger.ID, &trigger.RepositoryID, &trigger.Name, &eventTypesJSON,
		&conditionsJSON, &actionsJSON, &trigger.Enabled,
		&trigger.CreatedAt, &trigger.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("查询钩子触发器失败: %w", err)
	}

	// 解析JSON字段
	if err := json.Unmarshal(eventTypesJSON, &trigger.EventTypes); err != nil {
		return nil, fmt.Errorf("解析事件类型失败: %w", err)
	}

	if err := json.Unmarshal(conditionsJSON, &trigger.Conditions); err != nil {
		return nil, fmt.Errorf("解析触发条件失败: %w", err)
	}

	if err := json.Unmarshal(actionsJSON, &trigger.Actions); err != nil {
		return nil, fmt.Errorf("解析触发动作失败: %w", err)
	}

	return &trigger, nil
}

// ListWebhookTriggers 列出钩子触发器
func (r *webhookRepository) ListWebhookTriggers(ctx context.Context, repositoryID *uuid.UUID, page, pageSize int) (*models.WebhookTriggerListResponse, error) {
	// 构建WHERE条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if repositoryID != nil {
		whereClause += fmt.Sprintf(" AND repository_id = $%d", argIndex)
		args = append(args, *repositoryID)
		argIndex++
	}

	// 计算总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM webhook_triggers %s", whereClause)
	var total int64
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("计算触发器总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	listQuery := fmt.Sprintf(`
		SELECT id, repository_id, name, event_types, conditions,
			   actions, enabled, created_at, updated_at
		FROM webhook_triggers 
		%s 
		ORDER BY created_at DESC 
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("查询触发器列表失败: %w", err)
	}
	defer rows.Close()

	var triggers []models.WebhookTrigger
	for rows.Next() {
		var trigger models.WebhookTrigger
		var eventTypesJSON, conditionsJSON, actionsJSON []byte

		err := rows.Scan(
			&trigger.ID, &trigger.RepositoryID, &trigger.Name, &eventTypesJSON,
			&conditionsJSON, &actionsJSON, &trigger.Enabled,
			&trigger.CreatedAt, &trigger.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("扫描触发器数据失败: %w", err)
		}

		// 解析JSON字段
		if err := json.Unmarshal(eventTypesJSON, &trigger.EventTypes); err != nil {
			r.logger.Warn("解析事件类型失败", zap.Error(err))
			trigger.EventTypes = []models.WebhookEventType{}
		}

		if err := json.Unmarshal(conditionsJSON, &trigger.Conditions); err != nil {
			r.logger.Warn("解析触发条件失败", zap.Error(err))
		}

		if err := json.Unmarshal(actionsJSON, &trigger.Actions); err != nil {
			r.logger.Warn("解析触发动作失败", zap.Error(err))
		}

		triggers = append(triggers, trigger)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &models.WebhookTriggerListResponse{
		Triggers:   triggers,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateWebhookTrigger 更新钩子触发器
func (r *webhookRepository) UpdateWebhookTrigger(ctx context.Context, triggerID uuid.UUID, updates map[string]interface{}) error {
	// 定义允许更新的列白名单
	allowedColumns := map[string]bool{
		"name":         true,
		"event_types":  true,
		"conditions":   true,
		"actions":      true,
		"enabled":      true,
		"updated_at":   true,
	}

	// 构建SET子句
	setClause := ""
	args := []interface{}{}
	argIndex := 1

	for column, value := range updates {
		// 验证列名是否在白名单中
		if !allowedColumns[column] {
			return fmt.Errorf("不允许更新的列: %s", column)
		}

		if argIndex > 1 {
			setClause += ", "
		}
		setClause += fmt.Sprintf("%s = $%d", column, argIndex)
		args = append(args, value)
		argIndex++
	}

	if setClause == "" {
		return fmt.Errorf("没有要更新的字段")
	}

	// 添加WHERE条件
	query := fmt.Sprintf("UPDATE webhook_triggers SET %s WHERE id = $%d", setClause, argIndex)
	args = append(args, triggerID)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("更新钩子触发器失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("触发器不存在")
	}

	return nil
}

// DeleteWebhookTrigger 删除钩子触发器
func (r *webhookRepository) DeleteWebhookTrigger(ctx context.Context, triggerID uuid.UUID) error {
	query := "DELETE FROM webhook_triggers WHERE id = $1"

	result, err := r.db.ExecContext(ctx, query, triggerID)
	if err != nil {
		return fmt.Errorf("删除钩子触发器失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("触发器不存在")
	}

	return nil
}

// 其他方法的实现占位符
func (r *webhookRepository) CreateWebhookDelivery(ctx context.Context, delivery *models.WebhookDelivery) error {
	// TODO: 实现投递记录创建
	return fmt.Errorf("未实现")
}

func (r *webhookRepository) GetWebhookDelivery(ctx context.Context, deliveryID uuid.UUID) (*models.WebhookDelivery, error) {
	// TODO: 实现投递记录查询
	return nil, fmt.Errorf("未实现")
}

func (r *webhookRepository) ListWebhookDeliveries(ctx context.Context, webhookID *uuid.UUID, page, pageSize int) (*models.WebhookDeliveryListResponse, error) {
	// TODO: 实现投递记录列表
	return &models.WebhookDeliveryListResponse{}, nil
}

func (r *webhookRepository) UpdateWebhookDelivery(ctx context.Context, deliveryID uuid.UUID, updates map[string]interface{}) error {
	// TODO: 实现投递记录更新
	return fmt.Errorf("未实现")
}

func (r *webhookRepository) GetWebhookStatistics(ctx context.Context, repositoryID *uuid.UUID) (*models.WebhookStatistics, error) {
	// TODO: 实现统计信息查询
	return &models.WebhookStatistics{}, nil
}

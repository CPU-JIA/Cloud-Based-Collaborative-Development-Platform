package repository

import (
	"context"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/notification-service/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationRepository 通知存储库
type NotificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository 创建新的通知存储库
func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{
		db: db,
	}
}

// Create 创建通知
func (nr *NotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	return nr.db.WithContext(ctx).Create(notification).Error
}

// GetByID 根据ID获取通知
func (nr *NotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	var notification models.Notification
	err := nr.db.WithContext(ctx).
		Preload("Template").
		Preload("DeliveryLogs").
		First(&notification, id).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// GetNotificationsOptions 获取通知选项
type GetNotificationsOptions struct {
	ProjectID *uuid.UUID `json:"project_id,omitempty"`
	Category  string     `json:"category,omitempty"`
	Status    string     `json:"status,omitempty"`
	Priority  string     `json:"priority,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
	SortBy    string     `json:"sort_by,omitempty"` // created_at, priority
	SortOrder string     `json:"sort_order,omitempty"` // asc, desc
}

// GetByUser 获取用户的通知列表
func (nr *NotificationRepository) GetByUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID, options *GetNotificationsOptions) ([]*models.Notification, error) {
	query := nr.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID)

	// 应用过滤条件
	if options != nil {
		if options.ProjectID != nil {
			query = query.Where("project_id = ?", *options.ProjectID)
		}
		if options.Category != "" {
			query = query.Where("category = ?", options.Category)
		}
		if options.Status != "" {
			query = query.Where("status = ?", options.Status)
		}
		if options.Priority != "" {
			query = query.Where("priority = ?", options.Priority)
		}

		// 排序
		if options.SortBy != "" {
			sortOrder := "DESC"
			if options.SortOrder == "asc" {
				sortOrder = "ASC"
			}
			query = query.Order(options.SortBy + " " + sortOrder)
		} else {
			query = query.Order("created_at DESC")
		}

		// 分页
		if options.Limit > 0 {
			query = query.Limit(options.Limit)
		}
		if options.Offset > 0 {
			query = query.Offset(options.Offset)
		}
	} else {
		query = query.Order("created_at DESC").Limit(20)
	}

	var notifications []*models.Notification
	err := query.
		Preload("Template").
		Find(&notifications).Error

	return notifications, err
}

// GetUnreadCount 获取未读通知数量
func (nr *NotificationRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (int, error) {
	var count int64
	err := nr.db.WithContext(ctx).
		Model(&models.Notification{}).
		Where("user_id = ? AND tenant_id = ? AND status NOT IN ?", userID, tenantID, []string{"read", "deleted"}).
		Count(&count).Error
	
	return int(count), err
}

// UpdateStatus 更新通知状态
func (nr *NotificationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nr.db.WithContext(ctx).
		Model(&models.Notification{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// Update 更新通知
func (nr *NotificationRepository) Update(ctx context.Context, notification *models.Notification) error {
	return nr.db.WithContext(ctx).Save(notification).Error
}

// SoftDelete 软删除通知
func (nr *NotificationRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return nr.db.WithContext(ctx).Delete(&models.Notification{}, id).Error
}

// GetByCorrelationID 根据关联ID获取通知
func (nr *NotificationRepository) GetByCorrelationID(ctx context.Context, correlationID string, tenantID uuid.UUID) ([]*models.Notification, error) {
	var notifications []*models.Notification
	err := nr.db.WithContext(ctx).
		Where("correlation_id = ? AND tenant_id = ?", correlationID, tenantID).
		Preload("Template").
		Order("created_at DESC").
		Find(&notifications).Error

	return notifications, err
}

// CountNotificationsByRule 统计规则相关的通知数量
func (nr *NotificationRepository) CountNotificationsByRule(ctx context.Context, ruleID uuid.UUID, since time.Time) (int, error) {
	var count int64
	
	// 这里需要根据实际的数据库设计来调整查询
	// 假设我们有一个字段来追踪是哪个规则创建的通知
	err := nr.db.WithContext(ctx).
		Model(&models.Notification{}).
		Where("created_at >= ?", since).
		// TODO: 添加rule_id关联查询
		Count(&count).Error
	
	return int(count), err
}

// GetFailedNotifications 获取失败的通知
func (nr *NotificationRepository) GetFailedNotifications(ctx context.Context, limit int) ([]*models.Notification, error) {
	var notifications []*models.Notification
	err := nr.db.WithContext(ctx).
		Where("status = ? AND retry_count < max_retries", models.StatusFailed).
		Order("created_at ASC").
		Limit(limit).
		Find(&notifications).Error

	return notifications, err
}

// GetNotificationsByDateRange 根据日期范围获取通知
func (nr *NotificationRepository) GetNotificationsByDateRange(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) ([]*models.Notification, error) {
	var notifications []*models.Notification
	err := nr.db.WithContext(ctx).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, startDate, endDate).
		Order("created_at DESC").
		Find(&notifications).Error

	return notifications, err
}

// GetNotificationStats 获取通知统计信息
func (nr *NotificationRepository) GetNotificationStats(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*NotificationStats, error) {
	var stats NotificationStats
	
	// 总通知数量
	err := nr.db.WithContext(ctx).
		Model(&models.Notification{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, startDate, endDate).
		Count(&stats.Total).Error
	if err != nil {
		return nil, err
	}
	
	// 按状态统计
	statusStats := make(map[string]int64)
	rows, err := nr.db.WithContext(ctx).
		Model(&models.Notification{}).
		Select("status, COUNT(*) as count").
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, startDate, endDate).
		Group("status").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		statusStats[status] = count
	}
	
	stats.Sent = statusStats[models.StatusSent]
	stats.Failed = statusStats[models.StatusFailed]
	stats.Pending = statusStats[models.StatusPending]
	
	// 按类别统计
	categoryStats := make(map[string]int64)
	rows, err = nr.db.WithContext(ctx).
		Model(&models.Notification{}).
		Select("category, COUNT(*) as count").
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, startDate, endDate).
		Group("category").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var category string
		var count int64
		if err := rows.Scan(&category, &count); err != nil {
			return nil, err
		}
		categoryStats[category] = count
	}
	
	stats.CategoryStats = categoryStats
	
	return &stats, nil
}

// NotificationStats 通知统计信息
type NotificationStats struct {
	Total         int64            `json:"total"`
	Sent          int64            `json:"sent"`
	Failed        int64            `json:"failed"`
	Pending       int64            `json:"pending"`
	CategoryStats map[string]int64 `json:"category_stats"`
}
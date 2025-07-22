package repository

import (
	"context"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/notification-service/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeliveryLogRepository 发送日志存储库
type DeliveryLogRepository struct {
	db *gorm.DB
}

// NewDeliveryLogRepository 创建新的发送日志存储库
func NewDeliveryLogRepository(db *gorm.DB) *DeliveryLogRepository {
	return &DeliveryLogRepository{
		db: db,
	}
}

// Create 创建发送日志
func (dlr *DeliveryLogRepository) Create(ctx context.Context, log *models.DeliveryLog) error {
	return dlr.db.WithContext(ctx).Create(log).Error
}

// GetByID 根据ID获取发送日志
func (dlr *DeliveryLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DeliveryLog, error) {
	var log models.DeliveryLog
	err := dlr.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetByNotificationID 根据通知ID获取发送日志
func (dlr *DeliveryLogRepository) GetByNotificationID(ctx context.Context, notificationID uuid.UUID) ([]*models.DeliveryLog, error) {
	var logs []*models.DeliveryLog
	err := dlr.db.WithContext(ctx).
		Where("notification_id = ?", notificationID).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, err
}

// Update 更新发送日志
func (dlr *DeliveryLogRepository) Update(ctx context.Context, log *models.DeliveryLog) error {
	return dlr.db.WithContext(ctx).Save(log).Error
}

// GetFailedLogs 获取失败的发送日志
func (dlr *DeliveryLogRepository) GetFailedLogs(ctx context.Context, limit int) ([]*models.DeliveryLog, error) {
	var logs []*models.DeliveryLog
	err := dlr.db.WithContext(ctx).
		Where("status = ? AND attempt_count < ?", models.DeliveryStatusFailed, 3).
		Order("created_at ASC").
		Limit(limit).
		Find(&logs).Error

	return logs, err
}

// GetDeliveryStats 获取发送统计信息
func (dlr *DeliveryLogRepository) GetDeliveryStats(ctx context.Context, startDate, endDate time.Time) (*DeliveryStats, error) {
	var stats DeliveryStats
	
	// 按渠道统计
	channelStats := make(map[string]*ChannelStats)
	
	// 查询统计数据
	rows, err := dlr.db.WithContext(ctx).
		Model(&models.DeliveryLog{}).
		Select("channel, status, COUNT(*) as count, AVG(EXTRACT(EPOCH FROM (COALESCE(sent_at, failed_at) - created_at))) as avg_duration").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("channel, status").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var channel, status string
		var count int64
		var avgDuration float64
		
		if err := rows.Scan(&channel, &status, &count, &avgDuration); err != nil {
			return nil, err
		}
		
		if _, exists := channelStats[channel]; !exists {
			channelStats[channel] = &ChannelStats{
				Channel: channel,
			}
		}
		
		switch status {
		case models.DeliveryStatusSent:
			channelStats[channel].Sent = count
		case models.DeliveryStatusFailed:
			channelStats[channel].Failed = count
		case models.DeliveryStatusSending:
			channelStats[channel].Pending = count
		}
		
		channelStats[channel].AvgDuration = avgDuration
		channelStats[channel].Total = channelStats[channel].Sent + channelStats[channel].Failed + channelStats[channel].Pending
	}
	
	// 转换为切片
	for _, channelStat := range channelStats {
		stats.ChannelStats = append(stats.ChannelStats, *channelStat)
		stats.TotalSent += channelStat.Sent
		stats.TotalFailed += channelStat.Failed
		stats.TotalPending += channelStat.Pending
	}
	
	stats.Total = stats.TotalSent + stats.TotalFailed + stats.TotalPending
	
	if stats.Total > 0 {
		stats.SuccessRate = float64(stats.TotalSent) / float64(stats.Total) * 100
	}
	
	return &stats, nil
}

// GetLogsByChannel 根据渠道获取发送日志
func (dlr *DeliveryLogRepository) GetLogsByChannel(ctx context.Context, channel string, limit int) ([]*models.DeliveryLog, error) {
	var logs []*models.DeliveryLog
	err := dlr.db.WithContext(ctx).
		Where("channel = ?", channel).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error

	return logs, err
}

// DeleteOldLogs 删除旧的发送日志
func (dlr *DeliveryLogRepository) DeleteOldLogs(ctx context.Context, before time.Time) error {
	return dlr.db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&models.DeliveryLog{}).Error
}

// DeliveryStats 发送统计信息
type DeliveryStats struct {
	Total         int64          `json:"total"`
	TotalSent     int64          `json:"total_sent"`
	TotalFailed   int64          `json:"total_failed"`
	TotalPending  int64          `json:"total_pending"`
	SuccessRate   float64        `json:"success_rate"`
	ChannelStats  []ChannelStats `json:"channel_stats"`
}

// ChannelStats 渠道统计信息
type ChannelStats struct {
	Channel     string  `json:"channel"`
	Total       int64   `json:"total"`
	Sent        int64   `json:"sent"`
	Failed      int64   `json:"failed"`
	Pending     int64   `json:"pending"`
	AvgDuration float64 `json:"avg_duration"` // 平均处理时间（秒）
}
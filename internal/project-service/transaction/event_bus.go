package transaction

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// InMemoryEventBus 内存事件总线实现
type InMemoryEventBus struct {
	handlers map[EventType][]EventHandler
	mutex    sync.RWMutex
	logger   *zap.Logger
}

// NewInMemoryEventBus 创建内存事件总线
func NewInMemoryEventBus(logger *zap.Logger) *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[EventType][]EventHandler),
		logger:   logger,
	}
}

// Publish 发布事件
func (bus *InMemoryEventBus) Publish(ctx context.Context, event *DomainEvent) error {
	bus.mutex.RLock()
	handlers, exists := bus.handlers[event.Type]
	bus.mutex.RUnlock()

	if !exists || len(handlers) == 0 {
		bus.logger.Debug("没有找到事件处理器",
			zap.String("event_type", string(event.Type)),
			zap.String("event_id", event.ID.String()))
		return nil
	}

	bus.logger.Info("发布事件",
		zap.String("event_type", string(event.Type)),
		zap.String("event_id", event.ID.String()),
		zap.Int("handler_count", len(handlers)))

	// 异步执行所有处理器
	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					bus.logger.Error("事件处理器执行panic",
						zap.String("event_type", string(event.Type)),
						zap.String("event_id", event.ID.String()),
						zap.Any("panic", r))
				}
			}()

			if err := h(ctx, event); err != nil {
				bus.logger.Error("事件处理器执行失败",
					zap.String("event_type", string(event.Type)),
					zap.String("event_id", event.ID.String()),
					zap.Error(err))
			}
		}(handler)
	}

	return nil
}

// Subscribe 订阅事件
func (bus *InMemoryEventBus) Subscribe(eventType EventType, handler EventHandler) error {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	bus.handlers[eventType] = append(bus.handlers[eventType], handler)

	bus.logger.Info("注册事件处理器",
		zap.String("event_type", string(eventType)),
		zap.Int("total_handlers", len(bus.handlers[eventType])))

	return nil
}

// AsyncEventBus 异步事件总线（持久化到数据库）
type AsyncEventBus struct {
	db     EventStore
	logger *zap.Logger
}

// EventStore 事件存储接口
type EventStore interface {
	SaveEvent(ctx context.Context, event *DomainEvent) error
	GetPendingEvents(ctx context.Context, limit int) ([]*DomainEvent, error)
	MarkEventProcessed(ctx context.Context, eventID string) error
	MarkEventFailed(ctx context.Context, eventID string, error string, retryCount int) error
}

// NewAsyncEventBus 创建异步事件总线
func NewAsyncEventBus(db EventStore, logger *zap.Logger) *AsyncEventBus {
	return &AsyncEventBus{
		db:     db,
		logger: logger,
	}
}

// Publish 发布事件（异步持久化）
func (bus *AsyncEventBus) Publish(ctx context.Context, event *DomainEvent) error {
	// 设置事件时间戳
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// 持久化事件
	if err := bus.db.SaveEvent(ctx, event); err != nil {
		bus.logger.Error("保存事件失败",
			zap.String("event_type", string(event.Type)),
			zap.String("event_id", event.ID.String()),
			zap.Error(err))
		return fmt.Errorf("保存事件失败: %w", err)
	}

	bus.logger.Info("事件已发布并持久化",
		zap.String("event_type", string(event.Type)),
		zap.String("event_id", event.ID.String()),
		zap.String("aggregate_id", event.AggregateID.String()))

	return nil
}

// Subscribe 订阅事件（异步事件总线不需要立即订阅）
func (bus *AsyncEventBus) Subscribe(eventType EventType, handler EventHandler) error {
	bus.logger.Info("异步事件总线订阅",
		zap.String("event_type", string(eventType)))

	// 在异步模式下，处理器通过ProcessPendingEvents方法调用
	return nil
}

// DatabaseEventStore 数据库事件存储实现
type DatabaseEventStore struct {
	// 这里应该注入GORM DB实例
	// 简化实现，实际使用时需要完整实现
}

// SaveEvent 保存事件
func (store *DatabaseEventStore) SaveEvent(ctx context.Context, event *DomainEvent) error {
	// 实现数据库保存逻辑
	// db.Create(event)
	return nil
}

// GetPendingEvents 获取待处理事件
func (store *DatabaseEventStore) GetPendingEvents(ctx context.Context, limit int) ([]*DomainEvent, error) {
	// 实现获取待处理事件逻辑
	return nil, nil
}

// MarkEventProcessed 标记事件已处理
func (store *DatabaseEventStore) MarkEventProcessed(ctx context.Context, eventID string) error {
	// 实现标记逻辑
	return nil
}

// MarkEventFailed 标记事件处理失败
func (store *DatabaseEventStore) MarkEventFailed(ctx context.Context, eventID string, error string, retryCount int) error {
	// 实现失败标记逻辑
	return nil
}

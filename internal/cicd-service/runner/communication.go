package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/engine"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/repository"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// RunnerCommunicationManager Runner通信管理器接口
type RunnerCommunicationManager interface {
	// 启动通信管理器
	Start(ctx context.Context) error

	// 停止通信管理器
	Stop() error

	// 处理Runner连接
	HandleRunnerConnection(w http.ResponseWriter, r *http.Request)

	// 向Runner发送作业
	SendJobToRunner(runnerID uuid.UUID, job *JobMessage) error

	// 取消Runner作业
	CancelRunnerJob(runnerID uuid.UUID, jobID uuid.UUID) error

	// 获取在线Runner列表
	GetOnlineRunners() []uuid.UUID

	// 获取Runner状态
	GetRunnerStatus(runnerID uuid.UUID) (*RunnerConnectionStatus, error)
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

// JobResult 作业结果
type JobResult struct {
	JobID      uuid.UUID `json:"job_id"`
	Status     string    `json:"status"`
	ExitCode   int       `json:"exit_code"`
	Output     string    `json:"output"`
	Error      string    `json:"error"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Artifacts  []string  `json:"artifacts"`
}

// RunnerMessage Runner消息
type RunnerMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	MessageID string      `json:"message_id"`
}

// RunnerConnectionStatus Runner连接状态
type RunnerConnectionStatus struct {
	RunnerID      uuid.UUID  `json:"runner_id"`
	IsConnected   bool       `json:"is_connected"`
	ConnectedAt   time.Time  `json:"connected_at"`
	LastPingAt    time.Time  `json:"last_ping_at"`
	CurrentJobID  *uuid.UUID `json:"current_job_id"`
	WorkerVersion string     `json:"worker_version"`
}

// runnerConnection Runner连接
type runnerConnection struct {
	runnerID    uuid.UUID
	conn        *websocket.Conn
	send        chan []byte
	manager     *runnerCommunicationManager
	logger      *zap.Logger
	connectedAt time.Time
	lastPingAt  time.Time
	currentJob  *uuid.UUID
	mu          sync.RWMutex
}

// runnerCommunicationManager Runner通信管理器实现
type runnerCommunicationManager struct {
	repo   repository.PipelineRepository
	engine engine.PipelineEngine
	logger *zap.Logger

	// WebSocket升级器
	upgrader websocket.Upgrader

	// 连接管理
	connections map[uuid.UUID]*runnerConnection
	connMu      sync.RWMutex

	// 消息队列
	broadcast  chan []byte
	register   chan *runnerConnection
	unregister chan *runnerConnection

	// 状态管理
	isRunning bool
	stopCh    chan struct{}
	doneCh    chan struct{}
}

// NewRunnerCommunicationManager 创建Runner通信管理器
func NewRunnerCommunicationManager(repo repository.PipelineRepository, engine engine.PipelineEngine, logger *zap.Logger) RunnerCommunicationManager {
	return &runnerCommunicationManager{
		repo:   repo,
		engine: engine,
		logger: logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// 在生产环境中应该进行适当的Origin检查
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		connections: make(map[uuid.UUID]*runnerConnection),
		broadcast:   make(chan []byte, 256),
		register:    make(chan *runnerConnection),
		unregister:  make(chan *runnerConnection),
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}
}

// Start 启动通信管理器
func (m *runnerCommunicationManager) Start(ctx context.Context) error {
	if m.isRunning {
		return fmt.Errorf("通信管理器已在运行")
	}

	m.logger.Info("启动Runner通信管理器")
	m.isRunning = true

	// 启动主消息处理循环
	go m.messageLoop()

	// 启动心跳检查循环
	go m.heartbeatLoop()

	m.logger.Info("Runner通信管理器启动成功")
	return nil
}

// Stop 停止通信管理器
func (m *runnerCommunicationManager) Stop() error {
	if !m.isRunning {
		return fmt.Errorf("通信管理器未运行")
	}

	m.logger.Info("停止Runner通信管理器")

	// 关闭所有连接
	m.connMu.Lock()
	for _, conn := range m.connections {
		close(conn.send)
		conn.conn.Close()
	}
	m.connMu.Unlock()

	// 发送停止信号
	close(m.stopCh)

	// 等待完全停止
	<-m.doneCh

	m.isRunning = false
	m.logger.Info("Runner通信管理器已停止")
	return nil
}

// HandleRunnerConnection 处理Runner连接
func (m *runnerCommunicationManager) HandleRunnerConnection(w http.ResponseWriter, r *http.Request) {
	// 获取Runner ID
	runnerIDStr := r.URL.Query().Get("runner_id")
	if runnerIDStr == "" {
		http.Error(w, "Missing runner_id parameter", http.StatusBadRequest)
		return
	}

	runnerID, err := uuid.Parse(runnerIDStr)
	if err != nil {
		http.Error(w, "Invalid runner_id format", http.StatusBadRequest)
		return
	}

	// 验证Runner是否存在
	runner, err := m.repo.GetRunnerByID(r.Context(), runnerID)
	if err != nil {
		http.Error(w, "Runner not found", http.StatusNotFound)
		return
	}

	// 升级WebSocket连接
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.logger.Error("WebSocket升级失败", zap.Error(err))
		return
	}

	// 创建Runner连接
	runnerConn := &runnerConnection{
		runnerID:    runnerID,
		conn:        conn,
		send:        make(chan []byte, 256),
		manager:     m,
		logger:      m.logger.With(zap.String("runner_id", runnerID.String())),
		connectedAt: time.Now(),
		lastPingAt:  time.Now(),
	}

	// 注册连接
	m.register <- runnerConn

	// 更新Runner状态为在线
	m.repo.UpdateRunnerStatus(r.Context(), runnerID, models.RunnerStatusOnline)

	runnerConn.logger.Info("Runner连接已建立", zap.String("runner_name", runner.Name))

	// 启动读写协程
	go runnerConn.writePump()
	go runnerConn.readPump()
}

// SendJobToRunner 向Runner发送作业
func (m *runnerCommunicationManager) SendJobToRunner(runnerID uuid.UUID, job *JobMessage) error {
	m.connMu.RLock()
	conn, exists := m.connections[runnerID]
	m.connMu.RUnlock()

	if !exists {
		return fmt.Errorf("Runner %s 未连接", runnerID)
	}

	message := RunnerMessage{
		Type:      "job_start",
		Data:      job,
		Timestamp: time.Now(),
		MessageID: uuid.New().String(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化作业消息失败: %w", err)
	}

	select {
	case conn.send <- data:
		conn.mu.Lock()
		conn.currentJob = &job.JobID
		conn.mu.Unlock()
		return nil
	default:
		return fmt.Errorf("Runner %s 发送队列已满", runnerID)
	}
}

// CancelRunnerJob 取消Runner作业
func (m *runnerCommunicationManager) CancelRunnerJob(runnerID uuid.UUID, jobID uuid.UUID) error {
	m.connMu.RLock()
	conn, exists := m.connections[runnerID]
	m.connMu.RUnlock()

	if !exists {
		return fmt.Errorf("Runner %s 未连接", runnerID)
	}

	message := RunnerMessage{
		Type: "job_cancel",
		Data: map[string]interface{}{
			"job_id": jobID,
		},
		Timestamp: time.Now(),
		MessageID: uuid.New().String(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化取消消息失败: %w", err)
	}

	select {
	case conn.send <- data:
		return nil
	default:
		return fmt.Errorf("Runner %s 发送队列已满", runnerID)
	}
}

// GetOnlineRunners 获取在线Runner列表
func (m *runnerCommunicationManager) GetOnlineRunners() []uuid.UUID {
	m.connMu.RLock()
	defer m.connMu.RUnlock()

	runners := make([]uuid.UUID, 0, len(m.connections))
	for runnerID := range m.connections {
		runners = append(runners, runnerID)
	}

	return runners
}

// GetRunnerStatus 获取Runner状态
func (m *runnerCommunicationManager) GetRunnerStatus(runnerID uuid.UUID) (*RunnerConnectionStatus, error) {
	m.connMu.RLock()
	conn, exists := m.connections[runnerID]
	m.connMu.RUnlock()

	if !exists {
		return &RunnerConnectionStatus{
			RunnerID:    runnerID,
			IsConnected: false,
		}, nil
	}

	conn.mu.RLock()
	status := &RunnerConnectionStatus{
		RunnerID:     runnerID,
		IsConnected:  true,
		ConnectedAt:  conn.connectedAt,
		LastPingAt:   conn.lastPingAt,
		CurrentJobID: conn.currentJob,
	}
	conn.mu.RUnlock()

	return status, nil
}

// 私有方法

// messageLoop 主消息处理循环
func (m *runnerCommunicationManager) messageLoop() {
	defer close(m.doneCh)

	for {
		select {
		case <-m.stopCh:
			return

		case conn := <-m.register:
			m.connMu.Lock()
			m.connections[conn.runnerID] = conn
			m.connMu.Unlock()
			conn.logger.Info("Runner连接已注册")

		case conn := <-m.unregister:
			m.connMu.Lock()
			if _, exists := m.connections[conn.runnerID]; exists {
				delete(m.connections, conn.runnerID)
				close(conn.send)
			}
			m.connMu.Unlock()

			// 更新Runner状态为离线
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			m.repo.UpdateRunnerStatus(ctx, conn.runnerID, models.RunnerStatusOffline)
			cancel()

			conn.logger.Info("Runner连接已注销")

		case message := <-m.broadcast:
			// 广播消息到所有连接的Runner
			m.connMu.RLock()
			for _, conn := range m.connections {
				select {
				case conn.send <- message:
				default:
					// 发送失败，关闭连接
					close(conn.send)
					delete(m.connections, conn.runnerID)
				}
			}
			m.connMu.RUnlock()
		}
	}
}

// heartbeatLoop 心跳检查循环
func (m *runnerCommunicationManager) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkHeartbeats()
		}
	}
}

// checkHeartbeats 检查心跳
func (m *runnerCommunicationManager) checkHeartbeats() {
	now := time.Now()
	timeout := 2 * time.Minute

	m.connMu.Lock()
	defer m.connMu.Unlock()

	for runnerID, conn := range m.connections {
		conn.mu.RLock()
		lastPing := conn.lastPingAt
		conn.mu.RUnlock()

		if now.Sub(lastPing) > timeout {
			conn.logger.Warn("Runner心跳超时，断开连接")
			conn.conn.Close()
			delete(m.connections, runnerID)
			close(conn.send)
		}
	}
}

// Runner连接方法

// readPump 读取消息
func (c *runnerConnection) readPump() {
	defer func() {
		c.manager.unregister <- c
		c.conn.Close()
	}()

	// 设置读取超时
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.mu.Lock()
		c.lastPingAt = time.Now()
		c.mu.Unlock()
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageData, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket连接异常关闭", zap.Error(err))
			}
			break
		}

		// 处理接收到的消息
		c.handleMessage(messageData)
	}
}

// writePump 发送消息
func (c *runnerConnection) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.logger.Error("发送消息失败", zap.Error(err))
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 处理接收到的消息
func (c *runnerConnection) handleMessage(data []byte) {
	var message RunnerMessage
	if err := json.Unmarshal(data, &message); err != nil {
		c.logger.Error("解析消息失败", zap.Error(err))
		return
	}

	c.logger.Debug("收到Runner消息", zap.String("type", message.Type))

	switch message.Type {
	case "job_result":
		c.handleJobResult(message.Data)
	case "job_progress":
		c.handleJobProgress(message.Data)
	case "heartbeat":
		c.handleHeartbeat()
	case "log":
		c.handleLog(message.Data)
	default:
		c.logger.Warn("未知消息类型", zap.String("type", message.Type))
	}
}

// handleJobResult 处理作业结果
func (c *runnerConnection) handleJobResult(data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		c.logger.Error("序列化作业结果失败", zap.Error(err))
		return
	}

	var result JobResult
	if err := json.Unmarshal(dataBytes, &result); err != nil {
		c.logger.Error("解析作业结果失败", zap.Error(err))
		return
	}

	c.logger.Info("收到作业结果",
		zap.String("job_id", result.JobID.String()),
		zap.String("status", result.Status))

	// 转换为引擎的JobResult格式
	engineResult := &engine.JobResult{
		JobID:      result.JobID,
		RunnerID:   c.runnerID,
		Status:     models.JobStatus(result.Status),
		ExitCode:   &result.ExitCode,
		Output:     result.Output,
		StartedAt:  result.StartedAt,
		FinishedAt: result.FinishedAt,
		Artifacts:  result.Artifacts,
	}

	// 通知执行引擎
	if err := c.manager.engine.HandleJobResult(context.Background(), result.JobID, engineResult); err != nil {
		c.logger.Error("处理作业结果失败", zap.Error(err))
	}

	// 清除当前作业
	c.mu.Lock()
	c.currentJob = nil
	c.mu.Unlock()
}

// handleJobProgress 处理作业进度
func (c *runnerConnection) handleJobProgress(data interface{}) {
	c.logger.Debug("收到作业进度更新")
	// TODO: 实现作业进度处理
}

// handleHeartbeat 处理心跳
func (c *runnerConnection) handleHeartbeat() {
	c.mu.Lock()
	c.lastPingAt = time.Now()
	c.mu.Unlock()

	// 更新数据库中的最后联系时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	updates := map[string]interface{}{
		"last_contact_at": now,
		"status":          models.RunnerStatusOnline,
	}

	if err := c.manager.repo.UpdateRunner(ctx, c.runnerID, updates); err != nil {
		c.logger.Error("更新Runner心跳失败", zap.Error(err))
	}
}

// handleLog 处理日志
func (c *runnerConnection) handleLog(data interface{}) {
	c.logger.Debug("收到Runner日志")
	// TODO: 实现日志处理
}

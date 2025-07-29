package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// WebSocketServiceIntegrationTestSuite WebSocket服务集成测试套件
type WebSocketServiceIntegrationTestSuite struct {
	suite.Suite
	server   *httptest.Server
	hub      *Hub
	upgrader websocket.Upgrader
}

// MessageType 消息类型定义
type MessageType string

const (
	MessageTypeTaskUpdate    MessageType = "task_update"
	MessageTypeTaskCreate    MessageType = "task_create"
	MessageTypeTaskDelete    MessageType = "task_delete"
	MessageTypeUserJoin      MessageType = "user_join"
	MessageTypeUserLeave     MessageType = "user_leave"
	MessageTypeUserStatus    MessageType = "user_status"
	MessageTypeProjectUpdate MessageType = "project_update"
	MessageTypeChatMessage   MessageType = "chat_message"
	MessageTypeTyping        MessageType = "typing"
	MessageTypeHeartbeat     MessageType = "heartbeat"
)

// WSMessage WebSocket消息结构
type WSMessage struct {
	Type      MessageType `json:"type"`
	ProjectID int         `json:"project_id,omitempty"`
	UserID    int         `json:"user_id"`
	Username  string      `json:"username"`
	Avatar    string      `json:"avatar"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// TaskUpdateData 任务更新数据
type TaskUpdateData struct {
	TaskID      int    `json:"task_id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	StatusID    string `json:"status_id,omitempty"`
	Priority    string `json:"priority,omitempty"`
	AssigneeID  string `json:"assignee_id,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
}

// ChatMessageData 聊天消息数据
type ChatMessageData struct {
	Message   string `json:"message"`
	MessageID string `json:"message_id"`
}

// UserStatusData 用户状态数据
type UserStatusData struct {
	Status     string    `json:"status"` // online, away, busy, offline
	LastActive time.Time `json:"last_active"`
}

// TypingData 打字状态数据
type TypingData struct {
	IsTyping bool `json:"is_typing"`
	TaskID   int  `json:"task_id,omitempty"`
}

// Client 客户端连接管理
type Client struct {
	ID        string
	Conn      *websocket.Conn
	Send      chan WSMessage
	UserID    int
	Username  string
	Avatar    string
	ProjectID int
	Status    string
	LastSeen  time.Time
	mutex     sync.RWMutex
}

// ProjectRoom 项目房间管理
type ProjectRoom struct {
	ID      int
	Clients map[string]*Client
	mutex   sync.RWMutex
}

// Hub WebSocket中心管理器
type Hub struct {
	// 项目房间映射
	rooms map[int]*ProjectRoom

	// 注册客户端
	register chan *Client

	// 注销客户端
	unregister chan *Client

	// 广播消息
	broadcast chan WSMessage

	// 私聊消息
	direct chan WSMessage

	// 全局锁
	mutex sync.RWMutex

	// 停止通道
	stopCh chan struct{}
}

// TestClient 测试客户端
type TestClient struct {
	conn      *websocket.Conn
	userID    int
	username  string
	projectID int
	messages  []WSMessage
	mutex     sync.RWMutex
	doneCh    chan struct{}
}

// SetupSuite 设置测试套件
func (suite *WebSocketServiceIntegrationTestSuite) SetupSuite() {
	// 创建Hub
	suite.hub = &Hub{
		rooms:      make(map[int]*ProjectRoom),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan WSMessage),
		direct:     make(chan WSMessage),
		stopCh:     make(chan struct{}),
	}

	// 启动Hub
	go suite.hub.run()

	// 创建WebSocket升级器
	suite.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// 设置测试服务器
	suite.setupTestServer()
}

// TearDownSuite 清理测试套件
func (suite *WebSocketServiceIntegrationTestSuite) TearDownSuite() {
	if suite.hub != nil {
		close(suite.hub.stopCh)
	}
	if suite.server != nil {
		suite.server.Close()
	}
}

// setupTestServer 设置测试服务器
func (suite *WebSocketServiceIntegrationTestSuite) setupTestServer() {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// CORS配置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Upgrade", "Connection", "Sec-WebSocket-Key", "Sec-WebSocket-Version", "Sec-WebSocket-Extensions"},
		AllowCredentials: true,
	}))

	// WebSocket路由
	r.GET("/ws", suite.handleWebSocket)

	// REST API路由
	api := r.Group("/api/v1")
	{
		api.GET("/health", suite.healthCheck)
		api.GET("/rooms/:projectId/users", suite.getRoomUsers)
		api.POST("/rooms/:projectId/system-message", suite.sendSystemMessage)
	}

	suite.server = httptest.NewServer(r)
}

// handleWebSocket WebSocket连接处理器
func (suite *WebSocketServiceIntegrationTestSuite) handleWebSocket(c *gin.Context) {
	userIDStr := c.Query("user_id")
	username := c.Query("username")
	avatar := c.Query("avatar")
	projectIDStr := c.Query("project_id")

	if userIDStr == "" || username == "" || projectIDStr == "" {
		c.JSON(400, gin.H{"error": "缺少必要参数: user_id, username, project_id"})
		return
	}

	userID := parseIntDefault(userIDStr, 0)
	projectID := parseIntDefault(projectIDStr, 0)

	if userID == 0 || projectID == 0 {
		c.JSON(400, gin.H{"error": "参数格式错误"})
		return
	}

	// 升级HTTP连接为WebSocket
	conn, err := suite.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		suite.T().Logf("WebSocket升级失败: %v", err)
		return
	}

	// 创建新客户端
	client := suite.newClient(conn, userID, username, avatar, projectID)

	// 注册客户端
	suite.hub.register <- client

	// 启动读写协程
	go client.writePump()
	go client.readPump(suite.hub)
}

// healthCheck 健康检查
func (suite *WebSocketServiceIntegrationTestSuite) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"service": "WebSocket协作服务",
		"version": "1.0.0",
		"status":  "healthy",
		"rooms":   len(suite.hub.rooms),
	})
}

// getRoomUsers 获取房间在线用户
func (suite *WebSocketServiceIntegrationTestSuite) getRoomUsers(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID := parseIntDefault(projectIDStr, 0)

	if projectID == 0 {
		c.JSON(400, gin.H{"error": "项目ID格式错误"})
		return
	}

	room := suite.hub.getOrCreateRoom(projectID)
	room.mutex.RLock()
	defer room.mutex.RUnlock()

	var users []map[string]interface{}
	for _, client := range room.Clients {
		users = append(users, map[string]interface{}{
			"user_id":   client.UserID,
			"username":  client.Username,
			"avatar":    client.Avatar,
			"status":    client.Status,
			"last_seen": client.LastSeen,
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"project_id":   projectID,
			"online_count": len(users),
			"users":        users,
		},
	})
}

// sendSystemMessage 发送系统消息
func (suite *WebSocketServiceIntegrationTestSuite) sendSystemMessage(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID := parseIntDefault(projectIDStr, 0)

	if projectID == 0 {
		c.JSON(400, gin.H{"error": "项目ID格式错误"})
		return
	}

	var req struct {
		Type    MessageType `json:"type"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求数据格式错误"})
		return
	}

	message := WSMessage{
		Type:      req.Type,
		ProjectID: projectID,
		UserID:    0, // 系统用户
		Username:  "system",
		Avatar:    "",
		Data:      req.Data,
		Timestamp: time.Now(),
	}

	suite.hub.broadcast <- message

	c.JSON(200, gin.H{
		"success": true,
		"message": "系统消息发送成功",
	})
}

// createTestClient 创建测试客户端
func (suite *WebSocketServiceIntegrationTestSuite) createTestClient(userID int, username string, projectID int) (*TestClient, error) {
	// 构建WebSocket URL
	u, err := url.Parse(suite.server.URL)
	if err != nil {
		return nil, err
	}
	u.Scheme = "ws"
	u.Path = "/ws"
	u.RawQuery = fmt.Sprintf("user_id=%d&username=%s&project_id=%d&avatar=test.jpg", userID, username, projectID)

	// 连接WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	testClient := &TestClient{
		conn:      conn,
		userID:    userID,
		username:  username,
		projectID: projectID,
		messages:  []WSMessage{},
		doneCh:    make(chan struct{}),
	}

	// 启动消息接收协程
	go testClient.readMessages()

	return testClient, nil
}

// readMessages 读取消息
func (tc *TestClient) readMessages() {
	defer close(tc.doneCh)
	for {
		var message WSMessage
		err := tc.conn.ReadJSON(&message)
		if err != nil {
			return
		}

		tc.mutex.Lock()
		tc.messages = append(tc.messages, message)
		tc.mutex.Unlock()
	}
}

// sendMessage 发送消息
func (tc *TestClient) sendMessage(msgType MessageType, data interface{}) error {
	message := WSMessage{
		Type:      msgType,
		ProjectID: tc.projectID,
		UserID:    tc.userID,
		Username:  tc.username,
		Data:      data,
		Timestamp: time.Now(),
	}
	return tc.conn.WriteJSON(message)
}

// getMessages 获取接收到的消息
func (tc *TestClient) getMessages() []WSMessage {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	return append([]WSMessage{}, tc.messages...)
}

// close 关闭连接
func (tc *TestClient) close() {
	tc.conn.Close()
	<-tc.doneCh
}

// TestHealthCheck 测试健康检查
func (suite *WebSocketServiceIntegrationTestSuite) TestHealthCheck() {
	resp, err := http.Get(suite.server.URL + "/api/v1/health")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), result["success"].(bool))
	assert.Equal(suite.T(), "WebSocket协作服务", result["service"])
	assert.Equal(suite.T(), "healthy", result["status"])
}

// TestWebSocketConnection 测试WebSocket连接
func (suite *WebSocketServiceIntegrationTestSuite) TestWebSocketConnection() {
	client, err := suite.createTestClient(1, "testuser", 1)
	require.NoError(suite.T(), err)
	defer client.close()

	// 等待连接建立
	time.Sleep(100 * time.Millisecond)

	// 验证连接成功
	assert.NotNil(suite.T(), client.conn)
}

// TestUserJoinAndLeave 测试用户加入和离开
func (suite *WebSocketServiceIntegrationTestSuite) TestUserJoinAndLeave() {
	// 创建第一个客户端
	client1, err := suite.createTestClient(1, "user1", 1)
	require.NoError(suite.T(), err)
	defer client1.close()

	// 等待连接建立
	time.Sleep(100 * time.Millisecond)

	// 创建第二个客户端
	client2, err := suite.createTestClient(2, "user2", 1)
	require.NoError(suite.T(), err)
	defer client2.close()

	// 等待消息传播
	time.Sleep(200 * time.Millisecond)

	// 验证第一个客户端收到用户加入消息
	messages1 := client1.getMessages()
	assert.Greater(suite.T(), len(messages1), 0)

	// 查找用户加入消息
	var joinMessage *WSMessage
	for _, msg := range messages1 {
		if msg.Type == MessageTypeUserJoin && msg.UserID == 2 {
			joinMessage = &msg
			break
		}
	}
	assert.NotNil(suite.T(), joinMessage)
	assert.Equal(suite.T(), "user2", joinMessage.Username)
}

// TestChatMessage 测试聊天消息
func (suite *WebSocketServiceIntegrationTestSuite) TestChatMessage() {
	// 创建两个客户端
	client1, err := suite.createTestClient(1, "user1", 1)
	require.NoError(suite.T(), err)
	defer client1.close()

	client2, err := suite.createTestClient(2, "user2", 1)
	require.NoError(suite.T(), err)
	defer client2.close()

	// 等待连接建立
	time.Sleep(100 * time.Millisecond)

	// 发送聊天消息
	chatData := ChatMessageData{
		Message:   "Hello, World!",
		MessageID: "msg_001",
	}
	err = client1.sendMessage(MessageTypeChatMessage, chatData)
	require.NoError(suite.T(), err)

	// 等待消息传播
	time.Sleep(200 * time.Millisecond)

	// 验证第二个客户端收到聊天消息
	messages2 := client2.getMessages()
	var chatMessage *WSMessage
	for _, msg := range messages2 {
		if msg.Type == MessageTypeChatMessage && msg.UserID == 1 {
			chatMessage = &msg
			break
		}
	}
	assert.NotNil(suite.T(), chatMessage)
	assert.Equal(suite.T(), "user1", chatMessage.Username)

	// 验证消息内容
	if data, ok := chatMessage.Data.(map[string]interface{}); ok {
		assert.Equal(suite.T(), "Hello, World!", data["message"])
		assert.Equal(suite.T(), "msg_001", data["message_id"])
	}
}

// TestTaskUpdate 测试任务更新消息
func (suite *WebSocketServiceIntegrationTestSuite) TestTaskUpdate() {
	// 创建两个客户端
	client1, err := suite.createTestClient(1, "user1", 1)
	require.NoError(suite.T(), err)
	defer client1.close()

	client2, err := suite.createTestClient(2, "user2", 1)
	require.NoError(suite.T(), err)
	defer client2.close()

	// 等待连接建立
	time.Sleep(100 * time.Millisecond)

	// 发送任务更新消息
	taskData := TaskUpdateData{
		TaskID:      123,
		Title:       "Updated Task",
		Description: "Task description updated",
		StatusID:    "in_progress",
		Priority:    "high",
		AssigneeID:  "user2",
	}
	err = client1.sendMessage(MessageTypeTaskUpdate, taskData)
	require.NoError(suite.T(), err)

	// 等待消息传播
	time.Sleep(200 * time.Millisecond)

	// 验证第二个客户端收到任务更新消息
	messages2 := client2.getMessages()
	var taskMessage *WSMessage
	for _, msg := range messages2 {
		if msg.Type == MessageTypeTaskUpdate && msg.UserID == 1 {
			taskMessage = &msg
			break
		}
	}
	assert.NotNil(suite.T(), taskMessage)

	// 验证任务数据
	if data, ok := taskMessage.Data.(map[string]interface{}); ok {
		assert.Equal(suite.T(), float64(123), data["task_id"])
		assert.Equal(suite.T(), "Updated Task", data["title"])
		assert.Equal(suite.T(), "in_progress", data["status_id"])
		assert.Equal(suite.T(), "high", data["priority"])
	}
}

// TestTypingIndicator 测试打字状态
func (suite *WebSocketServiceIntegrationTestSuite) TestTypingIndicator() {
	// 创建两个客户端
	client1, err := suite.createTestClient(1, "user1", 1)
	require.NoError(suite.T(), err)
	defer client1.close()

	client2, err := suite.createTestClient(2, "user2", 1)
	require.NoError(suite.T(), err)
	defer client2.close()

	// 等待连接建立
	time.Sleep(100 * time.Millisecond)

	// 发送打字状态
	typingData := TypingData{
		IsTyping: true,
		TaskID:   456,
	}
	err = client1.sendMessage(MessageTypeTyping, typingData)
	require.NoError(suite.T(), err)

	// 等待消息传播
	time.Sleep(200 * time.Millisecond)

	// 验证第二个客户端收到打字状态消息
	messages2 := client2.getMessages()
	var typingMessage *WSMessage
	for _, msg := range messages2 {
		if msg.Type == MessageTypeTyping && msg.UserID == 1 {
			typingMessage = &msg
			break
		}
	}
	assert.NotNil(suite.T(), typingMessage)

	// 验证打字状态数据
	if data, ok := typingMessage.Data.(map[string]interface{}); ok {
		assert.Equal(suite.T(), true, data["is_typing"])
		assert.Equal(suite.T(), float64(456), data["task_id"])
	}
}

// TestUserStatusUpdate 测试用户状态更新
func (suite *WebSocketServiceIntegrationTestSuite) TestUserStatusUpdate() {
	// 创建两个客户端
	client1, err := suite.createTestClient(1, "user1", 1)
	require.NoError(suite.T(), err)
	defer client1.close()

	client2, err := suite.createTestClient(2, "user2", 1)
	require.NoError(suite.T(), err)
	defer client2.close()

	// 等待连接建立
	time.Sleep(100 * time.Millisecond)

	// 发送用户状态更新
	statusData := UserStatusData{
		Status:     "away",
		LastActive: time.Now(),
	}
	err = client1.sendMessage(MessageTypeUserStatus, statusData)
	require.NoError(suite.T(), err)

	// 等待消息传播
	time.Sleep(200 * time.Millisecond)

	// 验证第二个客户端收到状态更新消息
	messages2 := client2.getMessages()
	var statusMessage *WSMessage
	for _, msg := range messages2 {
		if msg.Type == MessageTypeUserStatus && msg.UserID == 1 {
			statusMessage = &msg
			break
		}
	}
	assert.NotNil(suite.T(), statusMessage)

	// 验证状态数据
	if data, ok := statusMessage.Data.(map[string]interface{}); ok {
		assert.Equal(suite.T(), "away", data["status"])
	}
}

// TestRoomIsolation 测试房间隔离
func (suite *WebSocketServiceIntegrationTestSuite) TestRoomIsolation() {
	// 创建不同项目的客户端
	client1, err := suite.createTestClient(1, "user1", 1) // 项目1
	require.NoError(suite.T(), err)
	defer client1.close()

	client2, err := suite.createTestClient(2, "user2", 2) // 项目2
	require.NoError(suite.T(), err)
	defer client2.close()

	// 等待连接建立
	time.Sleep(100 * time.Millisecond)

	// 在项目1发送消息
	chatData := ChatMessageData{
		Message:   "Project 1 message",
		MessageID: "msg_project1",
	}
	err = client1.sendMessage(MessageTypeChatMessage, chatData)
	require.NoError(suite.T(), err)

	// 等待消息传播
	time.Sleep(200 * time.Millisecond)

	// 验证项目2的客户端不应该收到项目1的消息
	messages2 := client2.getMessages()
	for _, msg := range messages2 {
		if msg.Type == MessageTypeChatMessage && msg.UserID == 1 {
			suite.T().Error("项目2的客户端不应该收到项目1的消息")
		}
	}
}

// TestRoomUsersAPI 测试房间用户API
func (suite *WebSocketServiceIntegrationTestSuite) TestRoomUsersAPI() {
	// 创建几个客户端
	client1, err := suite.createTestClient(1, "user1", 1)
	require.NoError(suite.T(), err)
	defer client1.close()

	client2, err := suite.createTestClient(2, "user2", 1)
	require.NoError(suite.T(), err)
	defer client2.close()

	// 等待连接建立
	time.Sleep(100 * time.Millisecond)

	// 调用房间用户API
	resp, err := http.Get(suite.server.URL + "/api/v1/rooms/1/users")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), result["success"].(bool))

	data := result["data"].(map[string]interface{})
	assert.Equal(suite.T(), float64(1), data["project_id"])
	assert.Equal(suite.T(), float64(2), data["online_count"])

	users := data["users"].([]interface{})
	assert.Equal(suite.T(), 2, len(users))
}

// TestSystemMessage 测试系统消息
func (suite *WebSocketServiceIntegrationTestSuite) TestSystemMessage() {
	// 创建客户端
	client, err := suite.createTestClient(1, "user1", 1)
	require.NoError(suite.T(), err)
	defer client.close()

	// 等待连接建立
	time.Sleep(100 * time.Millisecond)

	// 发送系统消息
	systemMsg := map[string]interface{}{
		"type":    "task_create",
		"message": "New task created",
		"data": map[string]interface{}{
			"task_id": 789,
			"title":   "System Created Task",
		},
	}

	jsonData, _ := json.Marshal(systemMsg)
	resp, err := http.Post(
		suite.server.URL+"/api/v1/rooms/1/system-message",
		"application/json",
		strings.NewReader(string(jsonData)),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// 等待消息传播
	time.Sleep(200 * time.Millisecond)

	// 验证客户端收到系统消息
	messages := client.getMessages()
	var systemMessage *WSMessage
	for _, msg := range messages {
		if msg.Type == MessageTypeTaskCreate && msg.UserID == 0 {
			systemMessage = &msg
			break
		}
	}
	assert.NotNil(suite.T(), systemMessage)
	assert.Equal(suite.T(), "system", systemMessage.Username)
}

// TestConcurrentConnections 测试并发连接
func (suite *WebSocketServiceIntegrationTestSuite) TestConcurrentConnections() {
	const numClients = 10
	clients := make([]*TestClient, numClients)
	var wg sync.WaitGroup

	// 并发创建多个客户端
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client, err := suite.createTestClient(id+1, fmt.Sprintf("user%d", id+1), 1)
			require.NoError(suite.T(), err)
			clients[id] = client
		}(i)
	}

	wg.Wait()

	// 等待所有连接建立
	time.Sleep(200 * time.Millisecond)

	// 验证所有客户端都连接成功
	for i, client := range clients {
		assert.NotNil(suite.T(), client, "Client %d should not be nil", i+1)
		if client != nil {
			defer client.close()
		}
	}

	// 发送消息测试
	if clients[0] != nil {
		chatData := ChatMessageData{
			Message:   "Broadcast test",
			MessageID: "broadcast_001",
		}
		err := clients[0].sendMessage(MessageTypeChatMessage, chatData)
		require.NoError(suite.T(), err)
	}

	// 等待消息传播
	time.Sleep(300 * time.Millisecond)

	// 验证其他客户端收到消息
	receivedCount := 0
	for i := 1; i < numClients; i++ {
		if clients[i] != nil {
			messages := clients[i].getMessages()
			for _, msg := range messages {
				if msg.Type == MessageTypeChatMessage && msg.UserID == 1 {
					receivedCount++
					break
				}
			}
		}
	}

	assert.Greater(suite.T(), receivedCount, 0, "At least some clients should receive the broadcast message")
}

// TestHeartbeat 测试心跳机制
func (suite *WebSocketServiceIntegrationTestSuite) TestHeartbeat() {
	// 创建客户端
	client, err := suite.createTestClient(1, "user1", 1)
	require.NoError(suite.T(), err)
	defer client.close()

	// 等待连接建立
	time.Sleep(100 * time.Millisecond)

	// 发送心跳消息
	err = client.sendMessage(MessageTypeHeartbeat, nil)
	require.NoError(suite.T(), err)

	// 心跳消息不应该被广播，所以不会收到回复
	time.Sleep(100 * time.Millisecond)

	messages := client.getMessages()
	for _, msg := range messages {
		assert.NotEqual(suite.T(), MessageTypeHeartbeat, msg.Type, "Heartbeat messages should not be broadcast")
	}
}

// TestErrorHandling 测试错误处理
func (suite *WebSocketServiceIntegrationTestSuite) TestErrorHandling() {
	// 测试缺少参数的WebSocket连接
	u, _ := url.Parse(suite.server.URL)
	u.Scheme = "ws"
	u.Path = "/ws"
	u.RawQuery = "user_id=1" // 缺少username和project_id

	_, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

	// 测试无效的房间用户API请求
	resp2, err := http.Get(suite.server.URL + "/api/v1/rooms/invalid/users")
	require.NoError(suite.T(), err)
	defer resp2.Body.Close()
	assert.Equal(suite.T(), http.StatusBadRequest, resp2.StatusCode)
}

// ========== 辅助方法 ==========

// Hub方法实现
func (h *Hub) run() {
	for {
		select {
		case <-h.stopCh:
			return

		case client := <-h.register:
			room := h.getOrCreateRoom(client.ProjectID)
			room.mutex.Lock()
			room.Clients[client.ID] = client
			room.mutex.Unlock()

			// 通知其他用户有新用户加入
			joinMessage := WSMessage{
				Type:      MessageTypeUserJoin,
				ProjectID: client.ProjectID,
				UserID:    client.UserID,
				Username:  client.Username,
				Avatar:    client.Avatar,
				Data: UserStatusData{
					Status:     client.Status,
					LastActive: client.LastSeen,
				},
				Timestamp: time.Now(),
			}
			h.broadcastToRoom(client.ProjectID, joinMessage, client.ID)

		case client := <-h.unregister:
			room := h.getOrCreateRoom(client.ProjectID)
			room.mutex.Lock()
			if _, ok := room.Clients[client.ID]; ok {
				delete(room.Clients, client.ID)
				close(client.Send)

				// 通知其他用户有用户离开
				leaveMessage := WSMessage{
					Type:      MessageTypeUserLeave,
					ProjectID: client.ProjectID,
					UserID:    client.UserID,
					Username:  client.Username,
					Avatar:    client.Avatar,
					Timestamp: time.Now(),
				}
				h.broadcastToRoom(client.ProjectID, leaveMessage, client.ID)
			}
			room.mutex.Unlock()

		case message := <-h.broadcast:
			h.broadcastToRoom(message.ProjectID, message, "")
		}
	}
}

func (h *Hub) getOrCreateRoom(projectID int) *ProjectRoom {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	room, exists := h.rooms[projectID]
	if !exists {
		room = &ProjectRoom{
			ID:      projectID,
			Clients: make(map[string]*Client),
		}
		h.rooms[projectID] = room
	}

	return room
}

func (h *Hub) broadcastToRoom(projectID int, message WSMessage, excludeClientID string) {
	room := h.getOrCreateRoom(projectID)
	room.mutex.RLock()
	defer room.mutex.RUnlock()

	for clientID, client := range room.Clients {
		if clientID != excludeClientID {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(room.Clients, clientID)
			}
		}
	}
}

// newClient 创建新客户端
func (suite *WebSocketServiceIntegrationTestSuite) newClient(conn *websocket.Conn, userID int, username, avatar string, projectID int) *Client {
	return &Client{
		ID:        fmt.Sprintf("%d_%d_%d", userID, projectID, time.Now().Unix()),
		Conn:      conn,
		Send:      make(chan WSMessage, 256),
		UserID:    userID,
		Username:  username,
		Avatar:    avatar,
		ProjectID: projectID,
		Status:    "online",
		LastSeen:  time.Now(),
	}
}

// Client方法实现
func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var message WSMessage
		err := c.Conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error
			}
			break
		}

		c.mutex.Lock()
		c.LastSeen = time.Now()
		c.mutex.Unlock()

		message.UserID = c.UserID
		message.Username = c.Username
		message.Avatar = c.Avatar
		message.ProjectID = c.ProjectID
		message.Timestamp = time.Now()

		switch message.Type {
		case MessageTypeHeartbeat:
			continue
		case MessageTypeUserStatus:
			if data, ok := message.Data.(map[string]interface{}); ok {
				if status, exists := data["status"].(string); exists {
					c.mutex.Lock()
					c.Status = status
					c.mutex.Unlock()
				}
			}
		}

		hub.broadcast <- message
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// parseIntDefault 解析整数，失败时返回默认值
func parseIntDefault(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	
	// 简单的整数解析
	result := 0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			result = result*10 + int(ch-'0')
		} else {
			return defaultValue
		}
	}
	return result
}

// TestWebSocketServiceIntegration 运行WebSocket服务集成测试
func TestWebSocketServiceIntegration(t *testing.T) {
	suite.Run(t, new(WebSocketServiceIntegrationTestSuite))
}

// ========== 性能测试 ==========

func BenchmarkWebSocketMessageBroadcast(b *testing.B) {
	// 设置测试环境
	gin.SetMode(gin.TestMode)

	hub := &Hub{
		rooms:      make(map[int]*ProjectRoom),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan WSMessage),
		direct:     make(chan WSMessage),
		stopCh:     make(chan struct{}),
	}

	// 启动Hub
	go hub.run()
	defer close(hub.stopCh)

	// 创建测试房间和客户端
	projectID := 1
	room := hub.getOrCreateRoom(projectID)

	// 模拟多个客户端
	for i := 0; i < 100; i++ {
		client := &Client{
			ID:        fmt.Sprintf("bench_client_%d", i),
			Send:      make(chan WSMessage, 256),
			UserID:    i,
			Username:  fmt.Sprintf("user%d", i),
			ProjectID: projectID,
			Status:    "online",
			LastSeen:  time.Now(),
		}

		room.mutex.Lock()
		room.Clients[client.ID] = client
		room.mutex.Unlock()

		// 启动客户端接收协程
		go func(c *Client) {
			for range c.Send {
				// 消费消息
			}
		}(client)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		message := WSMessage{
			Type:      MessageTypeChatMessage,
			ProjectID: projectID,
			UserID:    1,
			Username:  "benchuser",
			Data: ChatMessageData{
				Message:   fmt.Sprintf("Benchmark message %d", i),
				MessageID: fmt.Sprintf("bench_%d", i),
			},
			Timestamp: time.Now(),
		}

		hub.broadcast <- message
	}
}

func BenchmarkConcurrentConnections(b *testing.B) {
	// 测试并发连接建立的性能
	gin.SetMode(gin.TestMode)

	hub := &Hub{
		rooms:      make(map[int]*ProjectRoom),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan WSMessage),
		direct:     make(chan WSMessage),
		stopCh:     make(chan struct{}),
	}

	go hub.run()
	defer close(hub.stopCh)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client := &Client{
			ID:        fmt.Sprintf("bench_concurrent_%d", i),
			Send:      make(chan WSMessage, 256),
			UserID:    i,
			Username:  fmt.Sprintf("user%d", i),
			ProjectID: 1,
			Status:    "online",
			LastSeen:  time.Now(),
		}

		hub.register <- client
	}
}
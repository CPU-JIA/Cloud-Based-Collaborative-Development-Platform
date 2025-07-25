package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket升级器配置
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有来源的连接（开发环境）
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// 消息类型定义
type MessageType string

const (
	MessageTypeTaskUpdate     MessageType = "task_update"
	MessageTypeTaskCreate     MessageType = "task_create"
	MessageTypeTaskDelete     MessageType = "task_delete"
	MessageTypeUserJoin       MessageType = "user_join"
	MessageTypeUserLeave      MessageType = "user_leave"
	MessageTypeUserStatus     MessageType = "user_status"
	MessageTypeProjectUpdate  MessageType = "project_update"
	MessageTypeChatMessage    MessageType = "chat_message"
	MessageTypeTyping         MessageType = "typing"
	MessageTypeHeartbeat      MessageType = "heartbeat"
)

// WebSocket消息结构
type WSMessage struct {
	Type      MessageType `json:"type"`
	ProjectID int         `json:"project_id,omitempty"`
	UserID    int         `json:"user_id"`
	Username  string      `json:"username"`
	Avatar    string      `json:"avatar"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// 任务更新数据
type TaskUpdateData struct {
	TaskID      int    `json:"task_id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	StatusID    string `json:"status_id,omitempty"`
	Priority    string `json:"priority,omitempty"`
	AssigneeID  string `json:"assignee_id,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
}

// 聊天消息数据
type ChatMessageData struct {
	Message   string `json:"message"`
	MessageID string `json:"message_id"`
}

// 用户状态数据
type UserStatusData struct {
	Status     string `json:"status"` // online, away, busy, offline
	LastActive time.Time `json:"last_active"`
}

// 打字状态数据
type TypingData struct {
	IsTyping bool `json:"is_typing"`
	TaskID   int  `json:"task_id,omitempty"`
}

// 客户端连接管理
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

// 项目房间管理
type ProjectRoom struct {
	ID      int
	Clients map[string]*Client
	mutex   sync.RWMutex
}

// WebSocket中心管理器
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
}

// 全局Hub实例
var hub = &Hub{
	rooms:      make(map[int]*ProjectRoom),
	register:   make(chan *Client),
	unregister: make(chan *Client),
	broadcast:  make(chan WSMessage),
	direct:     make(chan WSMessage),
}

// 创建新客户端
func newClient(conn *websocket.Conn, userID int, username, avatar string, projectID int) *Client {
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

// 获取或创建项目房间
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
		fmt.Printf("🏠 创建项目房间: %d\n", projectID)
	}
	
	return room
}

// 中心消息处理循环
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			room := h.getOrCreateRoom(client.ProjectID)
			room.mutex.Lock()
			room.Clients[client.ID] = client
			room.mutex.Unlock()
			
			fmt.Printf("👤 用户 %s 加入项目 %d (连接ID: %s)\n", 
				client.Username, client.ProjectID, client.ID)
			
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
			
			// 发送当前在线用户列表给新用户
			h.sendOnlineUsers(client)
			
		case client := <-h.unregister:
			room := h.getOrCreateRoom(client.ProjectID)
			room.mutex.Lock()
			if _, ok := room.Clients[client.ID]; ok {
				delete(room.Clients, client.ID)
				close(client.Send)
				
				fmt.Printf("👋 用户 %s 离开项目 %d (连接ID: %s)\n", 
					client.Username, client.ProjectID, client.ID)
				
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
			
		case message := <-h.direct:
			h.sendDirectMessage(message)
		}
	}
}

// 向房间广播消息
func (h *Hub) broadcastToRoom(projectID int, message WSMessage, excludeClientID string) {
	room := h.getOrCreateRoom(projectID)
	room.mutex.RLock()
	defer room.mutex.RUnlock()
	
	for clientID, client := range room.Clients {
		if clientID != excludeClientID {
			select {
			case client.Send <- message:
			default:
				// 发送失败，关闭连接
				close(client.Send)
				delete(room.Clients, clientID)
			}
		}
	}
}

// 发送直接消息
func (h *Hub) sendDirectMessage(message WSMessage) {
	// 实现私聊逻辑
	// 目前暂时不实现，后续可扩展
}

// 发送在线用户列表
func (h *Hub) sendOnlineUsers(client *Client) {
	room := h.getOrCreateRoom(client.ProjectID)
	room.mutex.RLock()
	defer room.mutex.RUnlock()
	
	var onlineUsers []map[string]interface{}
	for _, c := range room.Clients {
		if c.ID != client.ID {
			onlineUsers = append(onlineUsers, map[string]interface{}{
				"user_id":  c.UserID,
				"username": c.Username,
				"avatar":   c.Avatar,
				"status":   c.Status,
				"last_seen": c.LastSeen,
			})
		}
	}
	
	message := WSMessage{
		Type:      MessageTypeUserStatus,
		ProjectID: client.ProjectID,
		UserID:    0, // 系统消息
		Username:  "system",
		Data:      map[string]interface{}{"online_users": onlineUsers},
		Timestamp: time.Now(),
	}
	
	select {
	case client.Send <- message:
	default:
		close(client.Send)
	}
}

// 客户端读取消息
func (c *Client) readPump() {
	defer func() {
		hub.unregister <- c
		c.Conn.Close()
	}()
	
	// 设置读取超时
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
				log.Printf("WebSocket错误: %v", err)
			}
			break
		}
		
		// 更新用户最后活跃时间
		c.mutex.Lock()
		c.LastSeen = time.Now()
		c.mutex.Unlock()
		
		// 设置消息发送者信息
		message.UserID = c.UserID
		message.Username = c.Username
		message.Avatar = c.Avatar
		message.ProjectID = c.ProjectID
		message.Timestamp = time.Now()
		
		// 处理不同类型的消息
		switch message.Type {
		case MessageTypeHeartbeat:
			// 心跳消息，不广播
			continue
		case MessageTypeUserStatus:
			if data, ok := message.Data.(map[string]interface{}); ok {
				if status, exists := data["status"].(string); exists {
					c.mutex.Lock()
					c.Status = status
					c.mutex.Unlock()
				}
			}
		case MessageTypeTyping:
			// 打字状态只发送给房间内其他用户，不存储
		default:
			// 其他消息类型正常处理
		}
		
		// 广播消息到房间
		hub.broadcast <- message
		
		fmt.Printf("📨 收到消息: %s 从用户 %s (项目: %d)\n", 
			message.Type, c.Username, c.ProjectID)
	}
}

// 客户端写入消息
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
				log.Printf("WebSocket写入错误: %v", err)
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

// WebSocket连接处理器
func handleWebSocket(c *gin.Context) {
	// 获取查询参数
	userIDStr := c.Query("user_id")
	username := c.Query("username")
	avatar := c.Query("avatar")
	projectIDStr := c.Query("project_id")
	
	if userIDStr == "" || username == "" || projectIDStr == "" {
		c.JSON(400, gin.H{"error": "缺少必要参数: user_id, username, project_id"})
		return
	}
	
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "用户ID格式错误"})
		return
	}
	
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "项目ID格式错误"})
		return
	}
	
	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}
	
	// 创建新客户端
	client := newClient(conn, userID, username, avatar, projectID)
	
	// 注册客户端
	hub.register <- client
	
	// 启动读写协程
	go client.writePump()
	go client.readPump()
}

// REST API：获取房间在线用户
func getRoomUsers(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "项目ID格式错误"})
		return
	}
	
	room := hub.getOrCreateRoom(projectID)
	room.mutex.RLock()
	defer room.mutex.RUnlock()
	
	var users []map[string]interface{}
	for _, client := range room.Clients {
		users = append(users, map[string]interface{}{
			"user_id":    client.UserID,
			"username":   client.Username,
			"avatar":     client.Avatar,
			"status":     client.Status,
			"last_seen":  client.LastSeen,
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

// REST API：向房间发送系统消息
func sendSystemMessage(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
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
	
	hub.broadcast <- message
	
	c.JSON(200, gin.H{
		"success": true,
		"message": "系统消息发送成功",
	})
}

// 健康检查
func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"service": "WebSocket协作服务",
		"version": "1.0.0",
		"status":  "healthy",
		"rooms":   len(hub.rooms),
		"uptime":  time.Since(startTime).String(),
	})
}

var startTime = time.Now()

func main() {
	// 启动Hub
	go hub.run()
	
	// 创建Gin路由
	r := gin.Default()
	
	// CORS配置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3001", "http://localhost:3002", "http://localhost:3003", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Upgrade", "Connection", "Sec-WebSocket-Key", "Sec-WebSocket-Version", "Sec-WebSocket-Extensions"},
		AllowCredentials: true,
	}))
	
	// WebSocket路由
	r.GET("/ws", handleWebSocket)
	
	// REST API路由
	api := r.Group("/api/v1")
	{
		api.GET("/health", healthCheck)
		api.GET("/rooms/:projectId/users", getRoomUsers)
		api.POST("/rooms/:projectId/system-message", sendSystemMessage)
	}
	
	// 启动信息
	fmt.Println("🚀 WebSocket协作服务启动成功！")
	fmt.Println("📡 WebSocket地址: ws://localhost:8084/ws")
	fmt.Println("🌐 HTTP API地址: http://localhost:8084/api/v1")
	fmt.Println("🔍 健康检查: http://localhost:8084/api/v1/health")
	fmt.Println("")
	fmt.Println("📚 WebSocket连接参数:")
	fmt.Println("   ?user_id=1&username=demo&avatar=url&project_id=1")
	fmt.Println("")
	fmt.Println("📝 支持的消息类型:")
	fmt.Println("   - task_update: 任务状态更新")
	fmt.Println("   - task_create: 创建新任务")
	fmt.Println("   - task_delete: 删除任务")
	fmt.Println("   - chat_message: 聊天消息")
	fmt.Println("   - typing: 打字状态")
	fmt.Println("   - user_status: 用户状态变更")
	fmt.Println("")
	
	// 启动服务
	log.Fatal(r.Run(":8084"))
}
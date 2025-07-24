package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

// JWT密钥
var jwtSecretKey = []byte("your-256-bit-secret-key-change-in-production-2025")

// 文档结构
type Document struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	AuthorID    int       `json:"author_id"`
	AuthorEmail string    `json:"author_email"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int       `json:"version"`
	IsPublic    bool      `json:"is_public"`
	Tags        []string  `json:"tags"`
	Category    string    `json:"category"`
}

// 文档版本历史
type DocumentVersion struct {
	ID         int       `json:"id"`
	DocumentID int       `json:"document_id"`
	Version    int       `json:"version"`
	Content    string    `json:"content"`
	AuthorID   int       `json:"author_id"`
	CreatedAt  time.Time `json:"created_at"`
	ChangeLog  string    `json:"change_log"`
}

// 协作会话
type CollaborationSession struct {
	DocumentID    int                        `json:"document_id"`
	Participants  map[int]*websocket.Conn    `json:"-"`
	Cursors       map[int]CursorPosition     `json:"cursors"`
	LastActivity  time.Time                  `json:"last_activity"`
	mutex         sync.RWMutex
}

// 光标位置
type CursorPosition struct {
	UserID int `json:"user_id"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

// 实时编辑操作
type EditOperation struct {
	Type       string    `json:"type"` // insert, delete, replace
	Position   int       `json:"position"`
	Content    string    `json:"content"`
	Length     int       `json:"length"`
	UserID     int       `json:"user_id"`
	Timestamp  time.Time `json:"timestamp"`
	DocumentID int       `json:"document_id"`
}

// JWT Claims
type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// 内存存储
type DocumentStore struct {
	documents map[int]*Document
	versions  map[int][]*DocumentVersion
	sessions  map[int]*CollaborationSession
	nextID    int
	mutex     sync.RWMutex
}

func NewDocumentStore() *DocumentStore {
	store := &DocumentStore{
		documents: make(map[int]*Document),
		versions:  make(map[int][]*DocumentVersion),
		sessions:  make(map[int]*CollaborationSession),
		nextID:    1,
	}
	
	// 创建示例文档
	store.createSampleDocuments()
	return store
}

func (ds *DocumentStore) createSampleDocuments() {
	// 示例文档1：项目概述
	doc1 := &Document{
		ID:          1,
		Title:       "企业协作开发平台 - 项目概述",
		Content: `# 企业协作开发平台

## 🎯 项目愿景
打造现代化的企业级协作开发平台，支持敏捷开发流程、实时协作和知识管理。

## 🏗️ 核心功能

### 1. 敏捷项目管理
- Scrum看板管理
- 任务拖拽操作
- 进度可视化
- 团队协作

### 2. 实时通信
- WebSocket实时通知
- 多人协作编辑
- 即时消息推送

### 3. 知识库管理
- Markdown文档编辑
- 版本控制
- 协作编辑
- 标签分类

## 📊 技术架构

### 后端技术栈
- **语言**: Go 1.21+
- **框架**: Gin Web Framework
- **认证**: JWT Token
- **实时通信**: WebSocket
- **存储**: PostgreSQL (生产) / 内存存储 (演示)

### 前端技术栈
- **核心**: HTML5 + CSS3 + JavaScript ES6+
- **编辑器**: Monaco Editor (VS Code内核)
- **实时协作**: WebSocket + Operational Transform
- **UI框架**: 自定义响应式设计

## 🚀 部署架构
- **认证服务**: :8083
- **知识库服务**: :8084
- **前端服务**: :3001

## 📈 开发进度
- [x] 基础架构搭建
- [x] 用户认证系统
- [x] Scrum看板功能
- [x] WebSocket实时通知
- [ ] 知识库功能 (当前开发中)
- [ ] 产品演示完善

---
*最后更新: 2025-07-23*
*作者: Claude AI Assistant*`,
		AuthorID:    1,
		AuthorEmail: "jia@example.com",
		CreatedAt:   time.Now().Add(-2 * time.Hour),
		UpdatedAt:   time.Now().Add(-30 * time.Minute),
		Version:     3,
		IsPublic:    true,
		Tags:        []string{"项目管理", "技术文档", "架构设计"},
		Category:    "项目文档",
	}
	
	// 示例文档2：API文档
	doc2 := &Document{
		ID:          2,
		Title:       "知识库 API 文档",
		Content: `# 知识库 API 文档

## 认证
所有API请求需要在Header中包含JWT token：
` + "```" + `
Authorization: Bearer <your_jwt_token>
` + "```" + `

## 文档管理

### 获取文档列表
` + "```http" + `
GET /api/v1/documents
` + "```" + `

响应示例：
` + "```json" + `
{
  "data": [
    {
      "id": 1,
      "title": "项目概述",
      "author_email": "jia@example.com",
      "created_at": "2025-07-23T10:00:00Z",
      "updated_at": "2025-07-23T12:00:00Z",
      "version": 3,
      "tags": ["项目管理", "文档"]
    }
  ],
  "total": 1,
  "page": 1
}
` + "```" + `

### 创建文档
` + "```http" + `
POST /api/v1/documents
Content-Type: application/json

{
  "title": "新文档标题",
  "content": "# 文档内容\n\n这是一个新文档。",
  "is_public": true,
  "tags": ["标签1", "标签2"],
  "category": "分类名称"
}
` + "```" + `

### 获取文档详情
` + "```http" + `
GET /api/v1/documents/{id}
` + "```" + `

### 更新文档
` + "```http" + `
PUT /api/v1/documents/{id}
Content-Type: application/json

{
  "title": "更新的标题",
  "content": "更新的内容",
  "change_log": "更新说明"
}
` + "```" + `

## 实时协作

### WebSocket连接
` + "```" + `
ws://localhost:8084/ws/documents/{document_id}
` + "```" + `

### 消息格式
` + "```json" + `
{
  "type": "edit_operation",
  "data": {
    "operation": "insert",
    "position": 10,
    "content": "新增内容",
    "user_id": 1
  }
}
` + "```" + `

## 版本控制

### 获取版本历史
` + "```http" + `
GET /api/v1/documents/{id}/versions
` + "```" + `

### 回滚到指定版本
` + "```http" + `
POST /api/v1/documents/{id}/rollback
Content-Type: application/json

{
  "version": 2,
  "reason": "回滚原因"
}
` + "```" + ``,
		AuthorID:    1,
		AuthorEmail: "jia@example.com",
		CreatedAt:   time.Now().Add(-1 * time.Hour),
		UpdatedAt:   time.Now().Add(-15 * time.Minute),
		Version:     2,
		IsPublic:    true,
		Tags:        []string{"API", "文档", "开发指南"},
		Category:    "技术文档",
	}
	
	ds.documents[1] = doc1
	ds.documents[2] = doc2
	ds.nextID = 3
	
	// 创建版本历史
	ds.versions[1] = []*DocumentVersion{
		{ID: 1, DocumentID: 1, Version: 1, AuthorID: 1, CreatedAt: doc1.CreatedAt, ChangeLog: "初始版本"},
		{ID: 2, DocumentID: 1, Version: 2, AuthorID: 1, CreatedAt: doc1.CreatedAt.Add(30 * time.Minute), ChangeLog: "添加技术架构部分"},
		{ID: 3, DocumentID: 1, Version: 3, AuthorID: 1, CreatedAt: doc1.UpdatedAt, ChangeLog: "更新开发进度"},
	}
	
	ds.versions[2] = []*DocumentVersion{
		{ID: 4, DocumentID: 2, Version: 1, AuthorID: 1, CreatedAt: doc2.CreatedAt, ChangeLog: "初始版本"},
		{ID: 5, DocumentID: 2, Version: 2, AuthorID: 1, CreatedAt: doc2.UpdatedAt, ChangeLog: "添加WebSocket和版本控制API"},
	}
}

// 获取所有文档
func (ds *DocumentStore) GetDocuments(userID int, page, pageSize int) ([]*Document, int) {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	
	var docs []*Document
	for _, doc := range ds.documents {
		// 只返回公开文档或用户自己的文档
		if doc.IsPublic || doc.AuthorID == userID {
			docs = append(docs, doc)
		}
	}
	
	// 简单分页
	total := len(docs)
	start := (page - 1) * pageSize
	end := start + pageSize
	
	if start > total {
		return []*Document{}, total
	}
	if end > total {
		end = total
	}
	
	return docs[start:end], total
}

// 获取文档详情
func (ds *DocumentStore) GetDocument(id, userID int) (*Document, error) {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	
	doc, exists := ds.documents[id]
	if !exists {
		return nil, fmt.Errorf("文档不存在")
	}
	
	// 权限检查
	if !doc.IsPublic && doc.AuthorID != userID {
		return nil, fmt.Errorf("无权限访问此文档")
	}
	
	return doc, nil
}

// 创建文档
func (ds *DocumentStore) CreateDocument(title, content, category string, tags []string, isPublic bool, authorID int, authorEmail string) (*Document, error) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	
	doc := &Document{
		ID:          ds.nextID,
		Title:       title,
		Content:     content,
		AuthorID:    authorID,
		AuthorEmail: authorEmail,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
		IsPublic:    isPublic,
		Tags:        tags,
		Category:    category,
	}
	
	ds.documents[ds.nextID] = doc
	
	// 创建初始版本
	version := &DocumentVersion{
		ID:         len(ds.versions)*10 + 1,
		DocumentID: ds.nextID,
		Version:    1,
		Content:    content,
		AuthorID:   authorID,
		CreatedAt:  time.Now(),
		ChangeLog:  "初始版本",
	}
	ds.versions[ds.nextID] = []*DocumentVersion{version}
	
	ds.nextID++
	return doc, nil
}

// 更新文档
func (ds *DocumentStore) UpdateDocument(id int, title, content, changeLog string, userID int) (*Document, error) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	
	doc, exists := ds.documents[id]
	if !exists {
		return nil, fmt.Errorf("文档不存在")
	}
	
	// 权限检查
	if doc.AuthorID != userID {
		return nil, fmt.Errorf("无权限修改此文档")
	}
	
	// 创建新版本
	doc.Version++
	doc.Title = title
	doc.Content = content
	doc.UpdatedAt = time.Now()
	
	// 保存版本历史
	version := &DocumentVersion{
		ID:         len(ds.versions)*10 + doc.Version,
		DocumentID: id,
		Version:    doc.Version,
		Content:    content,
		AuthorID:   userID,
		CreatedAt:  time.Now(),
		ChangeLog:  changeLog,
	}
	ds.versions[id] = append(ds.versions[id], version)
	
	return doc, nil
}

// 获取文档版本历史
func (ds *DocumentStore) GetDocumentVersions(docID, userID int) ([]*DocumentVersion, error) {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	
	doc, exists := ds.documents[docID]
	if !exists {
		return nil, fmt.Errorf("文档不存在")
	}
	
	// 权限检查
	if !doc.IsPublic && doc.AuthorID != userID {
		return nil, fmt.Errorf("无权限访问此文档")
	}
	
	versions, exists := ds.versions[docID]
	if !exists {
		return []*DocumentVersion{}, nil
	}
	
	return versions, nil
}

// 协作会话管理
func (ds *DocumentStore) StartCollaboration(docID, userID int, conn *websocket.Conn) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	
	// 检查文档是否存在
	doc, exists := ds.documents[docID]
	if !exists {
		return fmt.Errorf("文档不存在")
	}
	
	// 权限检查
	if !doc.IsPublic && doc.AuthorID != userID {
		return fmt.Errorf("无权限协作此文档")
	}
	
	// 获取或创建协作会话
	session, exists := ds.sessions[docID]
	if !exists {
		session = &CollaborationSession{
			DocumentID:   docID,
			Participants: make(map[int]*websocket.Conn),
			Cursors:      make(map[int]CursorPosition),
			LastActivity: time.Now(),
		}
		ds.sessions[docID] = session
	}
	
	session.mutex.Lock()
	session.Participants[userID] = conn
	session.LastActivity = time.Now()
	session.mutex.Unlock()
	
	return nil
}

// WebSocket升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 生产环境应该检查origin
	},
}

// JWT中间件
func jwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少Authorization header"})
			c.Abort()
			return
		}
		
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token格式"})
			c.Abort()
			return
		}
		
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey, nil
		})
		
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			c.Abort()
			return
		}
		
		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token claims"})
			c.Abort()
			return
		}
		
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

func main() {
	// 设置Gin模式
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	
	// 创建Gin路由
	r := gin.Default()
	
	// CORS中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})
	
	// 初始化文档存储
	docStore := NewDocumentStore()
	
	// 健康检查
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":   "knowledge-base-service",
			"status":    "healthy",
			"version":   "v1.0.0-kb",
			"timestamp": time.Now(),
			"features":  []string{"markdown-editing", "real-time-collaboration", "version-control", "document-management"},
		})
	})
	
	// API路由组 (需要认证)
	api := r.Group("/api/v1")
	api.Use(jwtMiddleware())
	
	// 文档管理API
	{
		// 获取文档列表
		api.GET("/documents", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			
			// 分页参数
			page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
			pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
			category := c.Query("category")
			tag := c.Query("tag")
			
			if page < 1 {
				page = 1
			}
			if pageSize < 1 || pageSize > 100 {
				pageSize = 10
			}
			
			docs, total := docStore.GetDocuments(userID, page, pageSize)
			
			// 简单过滤 (生产环境应该在数据库层面过滤)
			var filteredDocs []*Document
			for _, doc := range docs {
				if category != "" && doc.Category != category {
					continue
				}
				if tag != "" {
					hasTag := false
					for _, t := range doc.Tags {
						if t == tag {
							hasTag = true
							break
						}
					}
					if !hasTag {
						continue
					}
				}
				filteredDocs = append(filteredDocs, doc)
			}
			
			c.JSON(http.StatusOK, gin.H{
				"data":  filteredDocs,
				"total": total,
				"page":  page,
			})
		})
		
		// 获取文档详情
		api.GET("/documents/:id", func(c *gin.Context) {
			id, err := strconv.Atoi(c.Param("id"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文档ID"})
				return
			}
			
			userID := c.GetInt("user_id")
			doc, err := docStore.GetDocument(id, userID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			
			c.JSON(http.StatusOK, gin.H{"data": doc})
		})
		
		// 创建文档
		api.POST("/documents", func(c *gin.Context) {
			var req struct {
				Title    string   `json:"title" binding:"required"`
				Content  string   `json:"content" binding:"required"`
				Category string   `json:"category"`
				Tags     []string `json:"tags"`
				IsPublic bool     `json:"is_public"`
			}
			
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
				return
			}
			
			userID := c.GetInt("user_id")
			userEmail := c.GetString("user_email")
			
			doc, err := docStore.CreateDocument(req.Title, req.Content, req.Category, req.Tags, req.IsPublic, userID, userEmail)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			
			c.JSON(http.StatusCreated, gin.H{
				"message": "文档创建成功",
				"data":    doc,
			})
		})
		
		// 更新文档
		api.PUT("/documents/:id", func(c *gin.Context) {
			id, err := strconv.Atoi(c.Param("id"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文档ID"})
				return
			}
			
			var req struct {
				Title     string `json:"title" binding:"required"`
				Content   string `json:"content" binding:"required"`
				ChangeLog string `json:"change_log"`
			}
			
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
				return
			}
			
			userID := c.GetInt("user_id")
			doc, err := docStore.UpdateDocument(id, req.Title, req.Content, req.ChangeLog, userID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			
			c.JSON(http.StatusOK, gin.H{
				"message": "文档更新成功",
				"data":    doc,
			})
		})
		
		// 获取文档版本历史
		api.GET("/documents/:id/versions", func(c *gin.Context) {
			id, err := strconv.Atoi(c.Param("id"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文档ID"})
				return
			}
			
			userID := c.GetInt("user_id")
			versions, err := docStore.GetDocumentVersions(id, userID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			
			c.JSON(http.StatusOK, gin.H{"data": versions})
		})
		
		// 删除文档
		api.DELETE("/documents/:id", func(c *gin.Context) {
			id, err := strconv.Atoi(c.Param("id"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文档ID"})
				return
			}
			
			userID := c.GetInt("user_id")
			
			// 简单的删除逻辑
			docStore.mutex.Lock()
			doc, exists := docStore.documents[id]
			if !exists {
				docStore.mutex.Unlock()
				c.JSON(http.StatusNotFound, gin.H{"error": "文档不存在"})
				return
			}
			
			if doc.AuthorID != userID {
				docStore.mutex.Unlock()
				c.JSON(http.StatusForbidden, gin.H{"error": "无权限删除此文档"})
				return
			}
			
			delete(docStore.documents, id)
			delete(docStore.versions, id)
			delete(docStore.sessions, id)
			docStore.mutex.Unlock()
			
			c.JSON(http.StatusOK, gin.H{"message": "文档删除成功"})
		})
	}
	
	// WebSocket协作端点
	r.GET("/ws/documents/:id", func(c *gin.Context) {
		// 从查询参数获取token (WebSocket不支持Authorization header)
		tokenString := c.Query("token")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少token参数"})
			return
		}
		
		// 验证token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey, nil
		})
		
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			return
		}
		
		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token claims"})
			return
		}
		
		// 获取文档ID
		docID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文档ID"})
			return
		}
		
		// 升级WebSocket连接
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket升级失败: %v", err)
			return
		}
		defer conn.Close()
		
		// 开始协作会话
		err = docStore.StartCollaboration(docID, claims.UserID, conn)
		if err != nil {
			conn.WriteJSON(gin.H{"error": err.Error()})
			return
		}
		
		// 发送欢迎消息
		conn.WriteJSON(gin.H{
			"type": "connected",
			"message": "协作会话已建立",
			"user_id": claims.UserID,
			"document_id": docID,
		})
		
		// 处理WebSocket消息
		for {
			var message gin.H
			err := conn.ReadJSON(&message)
			if err != nil {
				log.Printf("WebSocket读取消息失败: %v", err)
				break
			}
			
			// 处理不同类型的消息
			msgType, ok := message["type"].(string)
			if !ok {
				conn.WriteJSON(gin.H{"error": "消息类型无效"})
				continue
			}
			
			switch msgType {
			case "edit_operation":
				// 处理编辑操作
				handleEditOperation(docStore, docID, claims.UserID, message, conn)
			case "cursor_position":
				// 处理光标位置更新
				handleCursorUpdate(docStore, docID, claims.UserID, message)
			case "ping":
				// 心跳检测
				conn.WriteJSON(gin.H{"type": "pong"})
			default:
				conn.WriteJSON(gin.H{"error": "未知的消息类型: " + msgType})
			}
		}
		
		// 清理连接
		cleanupCollaboration(docStore, docID, claims.UserID)
	})
	
	// 启动服务
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}
	
	log.Printf("🚀 知识库服务已启动在端口 %s", port)
	log.Printf("📚 Markdown编辑功能已启用")
	log.Printf("⚡ 实时协作功能已启用")
	log.Printf("📝 版本控制功能已启用")
	
	log.Fatal(r.Run(":" + port))
}

// 处理编辑操作
func handleEditOperation(docStore *DocumentStore, docID, userID int, message gin.H, conn *websocket.Conn) {
	data, ok := message["data"].(map[string]interface{})
	if !ok {
		conn.WriteJSON(gin.H{"error": "编辑操作数据格式无效"})
		return
	}
	
	operation := EditOperation{
		Type:       data["operation"].(string),
		Position:   int(data["position"].(float64)),
		Content:    data["content"].(string),
		UserID:     userID,
		Timestamp:  time.Now(),
		DocumentID: docID,
	}
	
	// 广播编辑操作给所有协作者
	broadcastToCollaborators(docStore, docID, userID, gin.H{
		"type": "edit_operation",
		"data": operation,
	})
}

// 处理光标位置更新
func handleCursorUpdate(docStore *DocumentStore, docID, userID int, message gin.H) {
	data, ok := message["data"].(map[string]interface{})
	if !ok {
		return
	}
	
	cursor := CursorPosition{
		UserID: userID,
		Line:   int(data["line"].(float64)),
		Column: int(data["column"].(float64)),
	}
	
	// 更新光标位置
	docStore.mutex.Lock()
	if session, exists := docStore.sessions[docID]; exists {
		session.mutex.Lock()
		session.Cursors[userID] = cursor
		session.mutex.Unlock()
	}
	docStore.mutex.Unlock()
	
	// 广播光标位置给其他协作者
	broadcastToCollaborators(docStore, docID, userID, gin.H{
		"type": "cursor_update",
		"data": cursor,
	})
}

// 广播消息给协作者
func broadcastToCollaborators(docStore *DocumentStore, docID, excludeUserID int, message gin.H) {
	docStore.mutex.RLock()
	session, exists := docStore.sessions[docID]
	docStore.mutex.RUnlock()
	
	if !exists {
		return
	}
	
	session.mutex.RLock()
	defer session.mutex.RUnlock()
	
	for userID, conn := range session.Participants {
		if userID != excludeUserID {
			err := conn.WriteJSON(message)
			if err != nil {
				log.Printf("广播消息失败 (用户 %d): %v", userID, err)
				// 连接断开，清理
				delete(session.Participants, userID)
				delete(session.Cursors, userID)
			}
		}
	}
}

// 清理协作会话
func cleanupCollaboration(docStore *DocumentStore, docID, userID int) {
	docStore.mutex.Lock()
	defer docStore.mutex.Unlock()
	
	session, exists := docStore.sessions[docID]
	if !exists {
		return
	}
	
	session.mutex.Lock()
	defer session.mutex.Unlock()
	
	delete(session.Participants, userID)
	delete(session.Cursors, userID)
	
	// 如果没有协作者了，清理会话
	if len(session.Participants) == 0 {
		delete(docStore.sessions, docID)
	}
}
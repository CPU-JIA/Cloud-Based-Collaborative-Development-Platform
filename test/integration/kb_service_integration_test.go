package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloud-platform/collaborative-dev/shared/auth"
	"github.com/cloud-platform/collaborative-dev/shared/config"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// ================================
// Knowledge Base Models and DTOs
// ================================

// Document 知识库文档模型
type KBDocument struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Title       string    `json:"title" gorm:"not null;size:200"`
	Content     string    `json:"content" gorm:"not null;type:text"`
	Category    string    `json:"category" gorm:"size:50"`
	Tags        []string  `json:"tags" gorm:"type:text[]"`
	AuthorID    uuid.UUID `json:"author_id" gorm:"type:uuid;not null"`
	AuthorEmail string    `json:"author_email" gorm:"not null;size:255"`
	IsPublic    bool      `json:"is_public" gorm:"default:true"`
	Version     int       `json:"version" gorm:"default:1"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DocumentVersion 文档版本历史
type KBDocumentVersion struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	DocumentID uuid.UUID `json:"document_id" gorm:"type:uuid;not null"`
	Version    int       `json:"version" gorm:"not null"`
	Content    string    `json:"content" gorm:"not null;type:text"`
	AuthorID   uuid.UUID `json:"author_id" gorm:"type:uuid;not null"`
	ChangeLog  string    `json:"change_log" gorm:"size:500"`
	CreatedAt  time.Time `json:"created_at"`
}

// KnowledgeBase 知识库目录结构
type KnowledgeBase struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string    `json:"name" gorm:"not null;size:100"`
	Description string    `json:"description" gorm:"size:500"`
	ParentID    *uuid.UUID `json:"parent_id" gorm:"type:uuid"`
	TenantID    uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null"`
	IsPublic    bool      `json:"is_public" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Documents   []KBDocument `json:"documents"`
	Total       int64        `json:"total"`
	Page        int          `json:"page"`
	PageSize    int          `json:"page_size"`
	Query       string       `json:"query"`
	Category    string       `json:"category"`
	Tags        []string     `json:"tags"`
	SearchTime  time.Duration `json:"search_time"`
}

// DTO Models
type CreateDocumentRequest struct {
	Title       string   `json:"title" binding:"required,max=200"`
	Content     string   `json:"content" binding:"required"`
	Category    string   `json:"category" binding:"max=50"`
	Tags        []string `json:"tags"`
	IsPublic    bool     `json:"is_public"`
}

type UpdateDocumentRequest struct {
	Title     string `json:"title" binding:"required,max=200"`
	Content   string `json:"content" binding:"required"`
	ChangeLog string `json:"change_log" binding:"max=500"`
}

type CreateKnowledgeBaseRequest struct {
	Name        string     `json:"name" binding:"required,max=100"`
	Description string     `json:"description" binding:"max=500"`
	ParentID    *uuid.UUID `json:"parent_id"`
	IsPublic    bool       `json:"is_public"`
}

type SearchDocumentsRequest struct {
	Query       string   `json:"query"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Page        int      `json:"page"`
	PageSize    int      `json:"page_size"`
	AuthorID    string   `json:"author_id"`
	IsPublic    *bool    `json:"is_public"`
}

// ================================
// Mock Knowledge Base Service
// ================================

// KBService 知识库服务接口
type KBService interface {
	CreateDocument(req *CreateDocumentRequest, authorID uuid.UUID, authorEmail string) (*KBDocument, error)
	GetDocument(id uuid.UUID, userID uuid.UUID) (*KBDocument, error)
	UpdateDocument(id uuid.UUID, req *UpdateDocumentRequest, userID uuid.UUID) (*KBDocument, error)
	DeleteDocument(id uuid.UUID, userID uuid.UUID) error
	GetDocuments(req *SearchDocumentsRequest, userID uuid.UUID) (*SearchResult, error)
	GetDocumentVersions(docID uuid.UUID, userID uuid.UUID) ([]KBDocumentVersion, error)
	CreateKnowledgeBase(req *CreateKnowledgeBaseRequest, tenantID uuid.UUID) (*KnowledgeBase, error)
	GetKnowledgeBases(tenantID uuid.UUID, userID uuid.UUID) ([]KnowledgeBase, error)
	SearchDocuments(query string, userID uuid.UUID, tenantID uuid.UUID) (*SearchResult, error)
}

// MockKBService 模拟知识库服务
type MockKBService struct {
	documents      map[uuid.UUID]*KBDocument
	versions       map[uuid.UUID][]KBDocumentVersion
	knowledgeBases map[uuid.UUID]*KnowledgeBase
}

// NewMockKBService 创建模拟知识库服务
func NewMockKBService() *MockKBService {
	return &MockKBService{
		documents:      make(map[uuid.UUID]*KBDocument),
		versions:       make(map[uuid.UUID][]KBDocumentVersion),
		knowledgeBases: make(map[uuid.UUID]*KnowledgeBase),
	}
}

func (s *MockKBService) CreateDocument(req *CreateDocumentRequest, authorID uuid.UUID, authorEmail string) (*KBDocument, error) {
	doc := &KBDocument{
		ID:          uuid.New(),
		Title:       req.Title,
		Content:     req.Content,
		Category:    req.Category,
		Tags:        req.Tags,
		AuthorID:    authorID,
		AuthorEmail: authorEmail,
		IsPublic:    req.IsPublic,
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.documents[doc.ID] = doc

	// 创建初始版本
	version := KBDocumentVersion{
		ID:         uuid.New(),
		DocumentID: doc.ID,
		Version:    1,
		Content:    req.Content,
		AuthorID:   authorID,
		ChangeLog:  "初始版本",
		CreatedAt:  time.Now(),
	}
	s.versions[doc.ID] = []KBDocumentVersion{version}

	return doc, nil
}

func (s *MockKBService) GetDocument(id uuid.UUID, userID uuid.UUID) (*KBDocument, error) {
	doc, exists := s.documents[id]
	if !exists {
		return nil, fmt.Errorf("文档不存在")
	}

	if !doc.IsPublic && doc.AuthorID != userID {
		return nil, fmt.Errorf("无权限访问此文档")
	}

	return doc, nil
}

func (s *MockKBService) UpdateDocument(id uuid.UUID, req *UpdateDocumentRequest, userID uuid.UUID) (*KBDocument, error) {
	doc, exists := s.documents[id]
	if !exists {
		return nil, fmt.Errorf("文档不存在")
	}

	if doc.AuthorID != userID {
		return nil, fmt.Errorf("无权限修改此文档")
	}

	doc.Version++
	doc.Title = req.Title
	doc.Content = req.Content
	doc.UpdatedAt = time.Now()

	// 创建新版本
	version := KBDocumentVersion{
		ID:         uuid.New(),
		DocumentID: id,
		Version:    doc.Version,
		Content:    req.Content,
		AuthorID:   userID,
		ChangeLog:  req.ChangeLog,
		CreatedAt:  time.Now(),
	}
	s.versions[id] = append(s.versions[id], version)

	return doc, nil
}

func (s *MockKBService) DeleteDocument(id uuid.UUID, userID uuid.UUID) error {
	doc, exists := s.documents[id]
	if !exists {
		return fmt.Errorf("文档不存在")
	}

	if doc.AuthorID != userID {
		return fmt.Errorf("无权限删除此文档")
	}

	delete(s.documents, id)
	delete(s.versions, id)
	return nil
}

func (s *MockKBService) GetDocuments(req *SearchDocumentsRequest, userID uuid.UUID) (*SearchResult, error) {
	var docs []KBDocument
	
	for _, doc := range s.documents {
		// 权限检查
		if !doc.IsPublic && doc.AuthorID != userID {
			continue
		}

		// 分类过滤
		if req.Category != "" && doc.Category != req.Category {
			continue
		}

		// 标签过滤
		if len(req.Tags) > 0 {
			hasMatchingTag := false
			for _, reqTag := range req.Tags {
				for _, docTag := range doc.Tags {
					if docTag == reqTag {
						hasMatchingTag = true
						break
					}
				}
				if hasMatchingTag {
					break
				}
			}
			if !hasMatchingTag {
				continue
			}
		}

		// 作者过滤
		if req.AuthorID != "" {
			authorUUID, err := uuid.Parse(req.AuthorID)
			if err == nil && doc.AuthorID != authorUUID {
				continue
			}
		}

		docs = append(docs, *doc)
	}

	// 分页
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 10
	}

	total := int64(len(docs))
	start := (page - 1) * pageSize
	end := start + pageSize

	if start > len(docs) {
		docs = []KBDocument{}
	} else if end > len(docs) {
		docs = docs[start:]
	} else {
		docs = docs[start:end]
	}

	return &SearchResult{
		Documents: docs,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		Query:     req.Query,
		Category:  req.Category,
		Tags:      req.Tags,
	}, nil
}

func (s *MockKBService) GetDocumentVersions(docID uuid.UUID, userID uuid.UUID) ([]KBDocumentVersion, error) {
	doc, exists := s.documents[docID]
	if !exists {
		return nil, fmt.Errorf("文档不存在")
	}

	if !doc.IsPublic && doc.AuthorID != userID {
		return nil, fmt.Errorf("无权限访问此文档")
	}

	versions, exists := s.versions[docID]
	if !exists {
		return []KBDocumentVersion{}, nil
	}

	return versions, nil
}

func (s *MockKBService) CreateKnowledgeBase(req *CreateKnowledgeBaseRequest, tenantID uuid.UUID) (*KnowledgeBase, error) {
	kb := &KnowledgeBase{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
		TenantID:    tenantID,
		IsPublic:    req.IsPublic,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.knowledgeBases[kb.ID] = kb
	return kb, nil
}

func (s *MockKBService) GetKnowledgeBases(tenantID uuid.UUID, userID uuid.UUID) ([]KnowledgeBase, error) {
	var kbs []KnowledgeBase
	
	for _, kb := range s.knowledgeBases {
		if kb.TenantID == tenantID {
			kbs = append(kbs, *kb)
		}
	}

	return kbs, nil
}

func (s *MockKBService) SearchDocuments(query string, userID uuid.UUID, tenantID uuid.UUID) (*SearchResult, error) {
	var docs []KBDocument
	
	for _, doc := range s.documents {
		// 权限检查
		if !doc.IsPublic && doc.AuthorID != userID {
			continue
		}

		// 搜索匹配
		if query != "" {
			queryLower := strings.ToLower(query)
			titleMatch := strings.Contains(strings.ToLower(doc.Title), queryLower)
			contentMatch := strings.Contains(strings.ToLower(doc.Content), queryLower)
			
			if !titleMatch && !contentMatch {
				continue
			}
		}

		docs = append(docs, *doc)
	}

	return &SearchResult{
		Documents:  docs,
		Total:      int64(len(docs)),
		Query:      query,
		SearchTime: time.Millisecond * 10, // 模拟搜索时间
	}, nil
}

// ================================
// Mock Knowledge Base Handler
// ================================

type MockKBHandler struct {
	service KBService
}

func NewMockKBHandler(service KBService) *MockKBHandler {
	return &MockKBHandler{service: service}
}

func (h *MockKBHandler) SetupRoutes(r *gin.RouterGroup) {
	kb := r.Group("/kb")
	{
		// 文档管理
		docs := kb.Group("/documents")
		{
			docs.POST("", h.CreateDocument)
			docs.GET("", h.GetDocuments)
			docs.GET("/:id", h.GetDocument)
			docs.PUT("/:id", h.UpdateDocument)
			docs.DELETE("/:id", h.DeleteDocument)
			docs.GET("/:id/versions", h.GetDocumentVersions)
		}

		// 知识库管理
		bases := kb.Group("/bases")
		{
			bases.POST("", h.CreateKnowledgeBase)
			bases.GET("", h.GetKnowledgeBases)
		}

		// 搜索功能
		kb.POST("/search", h.SearchDocuments)
	}
}

func (h *MockKBHandler) CreateDocument(c *gin.Context) {
	var req CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从context获取用户信息
	userID := uuid.New() // 模拟用户ID
	userEmail := "test@example.com"

	doc, err := h.service.CreateDocument(&req, userID, userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": doc})
}

func (h *MockKBHandler) GetDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文档ID"})
		return
	}

	userID := uuid.New() // 模拟用户ID

	doc, err := h.service.GetDocument(id, userID)
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": doc})
}

func (h *MockKBHandler) UpdateDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文档ID"})
		return
	}

	var req UpdateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从现有文档获取作者ID来模拟权限
	doc, getErr := h.service.GetDocument(id, uuid.New())
	if getErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文档不存在"})
		return
	}

	updatedDoc, err := h.service.UpdateDocument(id, &req, doc.AuthorID)
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updatedDoc})
}

func (h *MockKBHandler) DeleteDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文档ID"})
		return
	}

	// 从现有文档获取作者ID来模拟权限
	doc, getErr := h.service.GetDocument(id, uuid.New())
	if getErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文档不存在"})
		return
	}

	err = h.service.DeleteDocument(id, doc.AuthorID)
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "文档删除成功"})
}

func (h *MockKBHandler) GetDocuments(c *gin.Context) {
	var req SearchDocumentsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := uuid.New() // 模拟用户ID

	result, err := h.service.GetDocuments(&req, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *MockKBHandler) GetDocumentVersions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文档ID"})
		return
	}

	userID := uuid.New() // 模拟用户ID

	versions, err := h.service.GetDocumentVersions(id, userID)
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": versions})
}

func (h *MockKBHandler) CreateKnowledgeBase(c *gin.Context) {
	var req CreateKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := uuid.New() // 模拟租户ID

	kb, err := h.service.CreateKnowledgeBase(&req, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": kb})
}

func (h *MockKBHandler) GetKnowledgeBases(c *gin.Context) {
	tenantID := uuid.New() // 模拟租户ID
	userID := uuid.New()   // 模拟用户ID

	kbs, err := h.service.GetKnowledgeBases(tenantID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": kbs})
}

func (h *MockKBHandler) SearchDocuments(c *gin.Context) {
	query := c.Query("q")
	userID := uuid.New()   // 模拟用户ID
	tenantID := uuid.New() // 模拟租户ID

	result, err := h.service.SearchDocuments(query, userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// ================================
// Knowledge Base Integration Test Suite
// ================================

type KBServiceIntegrationTestSuite struct {
	suite.Suite
	db        *gorm.DB
	logger    logger.Logger
	router    *gin.Engine
	server    *httptest.Server
	kbService KBService
	kbHandler *MockKBHandler
	jwtService *auth.JWTService
	
	testTenantID uuid.UUID
	testUserID   uuid.UUID
	testDocs     []uuid.UUID
}

func (suite *KBServiceIntegrationTestSuite) SetupSuite() {
	// 初始化测试配置
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test",
			Password: "test",
			Name:     "test_kb_service",
		},
		Auth: config.AuthConfig{
			JWTSecret: "test-secret-key-for-kb-service",
		},
	}

	// 初始化logger
	suite.logger, _ = logger.NewZapLogger(logger.Config{
		Level:  "info",
		Format: "json",
	})

	// 初始化JWT服务
	suite.jwtService = auth.NewJWTService(cfg.Auth.JWTSecret, time.Hour*24, time.Hour*24*7)

	// 初始化知识库服务
	suite.kbService = NewMockKBService()
	suite.kbHandler = NewMockKBHandler(suite.kbService)

	// 设置测试用户和租户
	suite.testTenantID = uuid.New()
	suite.testUserID = uuid.New()

	// 设置Gin路由
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// 设置中间件
	suite.router.Use(gin.Logger())
	suite.router.Use(gin.Recovery())

	// 设置路由
	v1 := suite.router.Group("/api/v1")
	suite.kbHandler.SetupRoutes(v1)

	// 启动测试服务器
	suite.server = httptest.NewServer(suite.router)
}

func (suite *KBServiceIntegrationTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
}

func (suite *KBServiceIntegrationTestSuite) SetupTest() {
	// 清理测试数据
	suite.testDocs = []uuid.UUID{}
}

func (suite *KBServiceIntegrationTestSuite) TearDownTest() {
	// 清理创建的测试文档
	for _, docID := range suite.testDocs {
		suite.kbService.DeleteDocument(docID, suite.testUserID)
	}
}

// 辅助函数：发送HTTP请求
func (suite *KBServiceIntegrationTestSuite) makeRequest(method, path string, body interface{}, token string) (*http.Response, []byte) {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req, err := http.NewRequest(method, suite.server.URL+path, bytes.NewBuffer(reqBody))
	require.NoError(suite.T(), err)

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(suite.T(), err)

	var respBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	resp.Body.Close()
	
	respBytes, _ := json.Marshal(respBody)
	return resp, respBytes
}

// ================================
// 文档管理测试
// ================================

func (suite *KBServiceIntegrationTestSuite) TestDocumentManagement() {
	suite.Run("创建文档", func() {
		createReq := CreateDocumentRequest{
			Title:    "API设计指南",
			Content:  "# API设计指南\n\n## RESTful设计原则\n\n1. 使用名词表示资源\n2. 使用HTTP动词表示操作",
			Category: "技术文档",
			Tags:     []string{"API", "设计", "RESTful"},
			IsPublic: true,
		}

		resp, body := suite.makeRequest("POST", "/api/v1/kb/documents", createReq, "")
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		data := response["data"].(map[string]interface{})
		assert.Equal(suite.T(), createReq.Title, data["title"])
		assert.Equal(suite.T(), createReq.Category, data["category"])
		assert.True(suite.T(), data["is_public"].(bool))

		// 保存文档ID用于后续测试
		docID, err := uuid.Parse(data["id"].(string))
		require.NoError(suite.T(), err)
		suite.testDocs = append(suite.testDocs, docID)
	})

	suite.Run("获取文档详情", func() {
		// 先创建一个文档
		createReq := CreateDocumentRequest{
			Title:    "测试文档",
			Content:  "这是测试内容",
			Category: "测试",
			Tags:     []string{"测试"},
			IsPublic: true,
		}

		doc, err := suite.kbService.CreateDocument(&createReq, suite.testUserID, "test@example.com")
		require.NoError(suite.T(), err)
		suite.testDocs = append(suite.testDocs, doc.ID)

		// 获取文档
		resp, body := suite.makeRequest("GET", "/api/v1/kb/documents/"+doc.ID.String(), nil, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		data := response["data"].(map[string]interface{})
		assert.Equal(suite.T(), doc.Title, data["title"])
		assert.Equal(suite.T(), doc.Content, data["content"])
	})

	suite.Run("更新文档", func() {
		// 先创建一个文档
		createReq := CreateDocumentRequest{
			Title:    "原始文档",
			Content:  "原始内容",
			Category: "测试",
			Tags:     []string{"原始"},
			IsPublic: true,
		}

		doc, err := suite.kbService.CreateDocument(&createReq, suite.testUserID, "test@example.com")
		require.NoError(suite.T(), err)
		suite.testDocs = append(suite.testDocs, doc.ID)

		// 更新文档
		updateReq := UpdateDocumentRequest{
			Title:     "更新后的文档",
			Content:   "更新后的内容",
			ChangeLog: "更新了标题和内容",
		}

		resp, body := suite.makeRequest("PUT", "/api/v1/kb/documents/"+doc.ID.String(), updateReq, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		data := response["data"].(map[string]interface{})
		assert.Equal(suite.T(), updateReq.Title, data["title"])
		assert.Equal(suite.T(), updateReq.Content, data["content"])
		assert.Equal(suite.T(), float64(2), data["version"]) // 版本应该增加到2
	})

	suite.Run("删除文档", func() {
		// 先创建一个文档
		createReq := CreateDocumentRequest{
			Title:    "待删除文档",
			Content:  "待删除内容",
			Category: "测试",
			Tags:     []string{"删除"},
			IsPublic: true,
		}

		doc, err := suite.kbService.CreateDocument(&createReq, suite.testUserID, "test@example.com")
		require.NoError(suite.T(), err)

		// 删除文档
		resp, body := suite.makeRequest("DELETE", "/api/v1/kb/documents/"+doc.ID.String(), nil, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), "文档删除成功", response["message"])

		// 验证文档已删除
		resp, _ = suite.makeRequest("GET", "/api/v1/kb/documents/"+doc.ID.String(), nil, "")
		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
	})
}

// ================================
// 文档版本管理测试
// ================================

func (suite *KBServiceIntegrationTestSuite) TestDocumentVersions() {
	suite.Run("获取文档版本历史", func() {
		// 创建文档
		createReq := CreateDocumentRequest{
			Title:    "版本测试文档",
			Content:  "初始版本内容",
			Category: "测试",
			Tags:     []string{"版本"},
			IsPublic: true,
		}

		doc, err := suite.kbService.CreateDocument(&createReq, suite.testUserID, "test@example.com")
		require.NoError(suite.T(), err)
		suite.testDocs = append(suite.testDocs, doc.ID)

		// 更新文档几次
		for i := 2; i <= 3; i++ {
			updateReq := UpdateDocumentRequest{
				Title:     fmt.Sprintf("版本测试文档 v%d", i),
				Content:   fmt.Sprintf("第%d版本内容", i),
				ChangeLog: fmt.Sprintf("更新到版本%d", i),
			}
			_, err := suite.kbService.UpdateDocument(doc.ID, &updateReq, suite.testUserID)
			require.NoError(suite.T(), err)
		}

		// 获取版本历史
		resp, body := suite.makeRequest("GET", "/api/v1/kb/documents/"+doc.ID.String()+"/versions", nil, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		data := response["data"].([]interface{})
		assert.Len(suite.T(), data, 3) // 应该有3个版本

		// 验证版本顺序
		version1 := data[0].(map[string]interface{})
		assert.Equal(suite.T(), "初始版本", version1["change_log"])
		assert.Equal(suite.T(), float64(1), version1["version"])

		version3 := data[2].(map[string]interface{})
		assert.Equal(suite.T(), "更新到版本3", version3["change_log"])
		assert.Equal(suite.T(), float64(3), version3["version"])
	})
}

// ================================
// 文档搜索和列表测试
// ================================

func (suite *KBServiceIntegrationTestSuite) TestDocumentSearch() {
	// 创建多个测试文档
	testDocs := []CreateDocumentRequest{
		{
			Title:    "Go语言开发指南",
			Content:  "Go是Google开发的编程语言，适合构建高性能的网络服务",
			Category: "编程语言",
			Tags:     []string{"Go", "编程", "后端"},
			IsPublic: true,
		},
		{
			Title:    "React前端开发",
			Content:  "React是Facebook开发的前端框架，用于构建用户界面",
			Category: "前端框架",
			Tags:     []string{"React", "前端", "JavaScript"},
			IsPublic: true,
		},
		{
			Title:    "数据库设计原则",
			Content:  "数据库设计需要遵循范式理论，确保数据一致性和完整性",
			Category: "数据库",
			Tags:     []string{"数据库", "设计", "SQL"},
			IsPublic: false,
		},
	}

	// 创建测试文档
	for _, docReq := range testDocs {
		doc, err := suite.kbService.CreateDocument(&docReq, suite.testUserID, "test@example.com")
		require.NoError(suite.T(), err)
		suite.testDocs = append(suite.testDocs, doc.ID)
	}

	suite.Run("获取文档列表", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/kb/documents", nil, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		data := response["data"].(map[string]interface{})
		documents := data["documents"].([]interface{})
		
		// 应该能看到所有公开文档和自己的私有文档
		assert.GreaterOrEqual(suite.T(), len(documents), 2)
		assert.GreaterOrEqual(suite.T(), int(data["total"].(float64)), 2)
	})

	suite.Run("按分类过滤文档", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/kb/documents?category=编程语言", nil, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		data := response["data"].(map[string]interface{})
		documents := data["documents"].([]interface{})
		
		assert.Len(suite.T(), documents, 1)
		doc := documents[0].(map[string]interface{})
		assert.Equal(suite.T(), "Go语言开发指南", doc["title"])
	})

	suite.Run("按标签过滤文档", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/kb/documents?tags=前端", nil, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		data := response["data"].(map[string]interface{})
		documents := data["documents"].([]interface{})
		
		assert.Len(suite.T(), documents, 1)
		doc := documents[0].(map[string]interface{})
		assert.Equal(suite.T(), "React前端开发", doc["title"])
	})

	suite.Run("全文搜索", func() {
		resp, body := suite.makeRequest("POST", "/api/v1/kb/search?q=Go", nil, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		data := response["data"].(map[string]interface{})
		documents := data["documents"].([]interface{})
		
		assert.GreaterOrEqual(suite.T(), len(documents), 1)
		
		// 验证搜索结果包含Go相关文档
		foundGoDocs := false
		for _, docInterface := range documents {
			doc := docInterface.(map[string]interface{})
			title := strings.ToLower(doc["title"].(string))
			content := strings.ToLower(doc["content"].(string))
			if strings.Contains(title, "go") || strings.Contains(content, "go") {
				foundGoDocs = true
				break
			}
		}
		assert.True(suite.T(), foundGoDocs)
	})
}

// ================================
// 知识库目录管理测试
// ================================

func (suite *KBServiceIntegrationTestSuite) TestKnowledgeBaseManagement() {
	suite.Run("创建知识库", func() {
		createReq := CreateKnowledgeBaseRequest{
			Name:        "技术文档库",
			Description: "存放技术相关文档的知识库",
			IsPublic:    true,
		}

		resp, body := suite.makeRequest("POST", "/api/v1/kb/bases", createReq, "")
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		data := response["data"].(map[string]interface{})
		assert.Equal(suite.T(), createReq.Name, data["name"])
		assert.Equal(suite.T(), createReq.Description, data["description"])
		assert.True(suite.T(), data["is_public"].(bool))
	})

	suite.Run("获取知识库列表", func() {
		// 先创建几个知识库
		kbNames := []string{"开发指南", "API文档", "用户手册"}
		for _, name := range kbNames {
			createReq := CreateKnowledgeBaseRequest{
				Name:        name,
				Description: fmt.Sprintf("%s相关文档", name),
				IsPublic:    true,
			}
			_, err := suite.kbService.CreateKnowledgeBase(&createReq, suite.testTenantID)
			require.NoError(suite.T(), err)
		}

		resp, body := suite.makeRequest("GET", "/api/v1/kb/bases", nil, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		data := response["data"].([]interface{})
		assert.GreaterOrEqual(suite.T(), len(data), 3)

		// 验证知识库信息
		kbFound := make(map[string]bool)
		for _, kbInterface := range data {
			kb := kbInterface.(map[string]interface{})
			kbFound[kb["name"].(string)] = true
		}

		for _, name := range kbNames {
			assert.True(suite.T(), kbFound[name], fmt.Sprintf("知识库 %s 未找到", name))
		}
	})
}

// ================================
// 权限控制测试
// ================================

func (suite *KBServiceIntegrationTestSuite) TestPermissionControl() {
	suite.Run("访问权限控制", func() {
		// 创建私有文档
		createReq := CreateDocumentRequest{
			Title:    "私有文档",
			Content:  "这是私有内容，只有作者可以访问",
			Category: "私人",
			Tags:     []string{"私有"},
			IsPublic: false,
		}

		doc, err := suite.kbService.CreateDocument(&createReq, suite.testUserID, "author@example.com")
		require.NoError(suite.T(), err)
		suite.testDocs = append(suite.testDocs, doc.ID)

		// 作者可以访问
		retrievedDoc, err := suite.kbService.GetDocument(doc.ID, suite.testUserID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), doc.Title, retrievedDoc.Title)

		// 其他用户无法访问
		otherUserID := uuid.New()
		_, err = suite.kbService.GetDocument(doc.ID, otherUserID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "无权限访问")
	})

	suite.Run("编辑权限控制", func() {
		// 创建文档
		createReq := CreateDocumentRequest{
			Title:    "编辑权限测试",
			Content:  "原始内容",
			Category: "测试",
			Tags:     []string{"权限"},
			IsPublic: true,
		}

		doc, err := suite.kbService.CreateDocument(&createReq, suite.testUserID, "author@example.com")
		require.NoError(suite.T(), err)
		suite.testDocs = append(suite.testDocs, doc.ID)

		// 作者可以编辑
		updateReq := UpdateDocumentRequest{
			Title:     "编辑后的标题",
			Content:   "编辑后的内容",
			ChangeLog: "作者编辑",
		}

		_, err = suite.kbService.UpdateDocument(doc.ID, &updateReq, suite.testUserID)
		require.NoError(suite.T(), err)

		// 其他用户无法编辑
		otherUserID := uuid.New()
		_, err = suite.kbService.UpdateDocument(doc.ID, &updateReq, otherUserID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "无权限修改")
	})
}

// ================================
// 错误处理测试
// ================================

func (suite *KBServiceIntegrationTestSuite) TestErrorHandling() {
	suite.Run("无效文档ID", func() {
		resp, body := suite.makeRequest("GET", "/api/v1/kb/documents/invalid-uuid", nil, "")
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		assert.Contains(suite.T(), response["error"].(string), "无效的文档ID")
	})

	suite.Run("文档不存在", func() {
		nonExistentID := uuid.New()
		resp, body := suite.makeRequest("GET", "/api/v1/kb/documents/"+nonExistentID.String(), nil, "")
		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		assert.Contains(suite.T(), response["error"].(string), "文档不存在")
	})

	suite.Run("创建文档验证错误", func() {
		// 空标题
		createReq := CreateDocumentRequest{
			Title:    "",
			Content:  "内容",
			Category: "测试",
			Tags:     []string{},
			IsPublic: true,
		}

		resp, body := suite.makeRequest("POST", "/api/v1/kb/documents", createReq, "")
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err := json.Unmarshal(body, &response)
		require.NoError(suite.T(), err)

		assert.Contains(suite.T(), response["error"].(string), "required")
	})
}

// ================================
// 性能测试
// ================================

func (suite *KBServiceIntegrationTestSuite) TestPerformance() {
	suite.Run("批量创建文档性能", func() {
		start := time.Now()
		
		for i := 0; i < 50; i++ {
			createReq := CreateDocumentRequest{
				Title:    fmt.Sprintf("性能测试文档_%d", i),
				Content:  fmt.Sprintf("这是第%d个性能测试文档的内容", i),
				Category: "性能测试",
				Tags:     []string{"性能", "测试"},
				IsPublic: true,
			}

			doc, err := suite.kbService.CreateDocument(&createReq, suite.testUserID, "perf@example.com")
			require.NoError(suite.T(), err)
			suite.testDocs = append(suite.testDocs, doc.ID)
		}

		duration := time.Since(start)
		
		// 创建50个文档应该在100ms内完成
		assert.Less(suite.T(), duration, 100*time.Millisecond, "批量创建文档性能不达标")
		suite.T().Logf("创建50个文档耗时: %v", duration)
	})

	suite.Run("搜索性能测试", func() {
		// 先创建一些文档用于搜索
		for i := 0; i < 20; i++ {
			createReq := CreateDocumentRequest{
				Title:    fmt.Sprintf("搜索测试文档_%d", i),
				Content:  fmt.Sprintf("搜索关键词测试内容_%d", i),
				Category: "搜索测试",
				Tags:     []string{"搜索", "性能"},
				IsPublic: true,
			}

			doc, err := suite.kbService.CreateDocument(&createReq, suite.testUserID, "search@example.com")
			require.NoError(suite.T(), err)
			suite.testDocs = append(suite.testDocs, doc.ID)
		}

		start := time.Now()
		
		// 执行多次搜索
		for i := 0; i < 20; i++ {
			_, err := suite.kbService.SearchDocuments("搜索", suite.testUserID, suite.testTenantID)
			require.NoError(suite.T(), err)
		}

		duration := time.Since(start)
		
		// 20次搜索应该在50ms内完成
		assert.Less(suite.T(), duration, 50*time.Millisecond, "搜索性能不达标")
		suite.T().Logf("执行20次搜索耗时: %v", duration)
	})
}

// ================================
// 集成工作流程测试
// ================================

func (suite *KBServiceIntegrationTestSuite) TestCompleteWorkflow() {
	suite.Run("完整知识库管理工作流程", func() {
		// 1. 创建知识库
		kbReq := CreateKnowledgeBaseRequest{
			Name:        "项目文档库",
			Description: "项目相关的所有文档",
			IsPublic:    true,
		}

		kb, err := suite.kbService.CreateKnowledgeBase(&kbReq, suite.testTenantID)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), kb)

		// 2. 创建多个分类的文档
		docCategories := []struct {
			title    string
			content  string
			category string
			tags     []string
		}{
			{
				title:    "项目架构设计",
				content:  "# 项目架构\n\n本项目采用微服务架构...",
				category: "架构设计",
				tags:     []string{"架构", "设计", "微服务"},
			},
			{
				title:    "API接口文档",
				content:  "# API文档\n\n## 用户接口\n\n### 登录接口",
				category: "API文档",
				tags:     []string{"API", "接口", "文档"},
			},
			{
				title:    "部署指南",
				content:  "# 部署指南\n\n## 环境准备\n\n1. Docker环境",
				category: "运维文档",
				tags:     []string{"部署", "运维", "Docker"},
			},
		}

		var createdDocs []uuid.UUID
		for _, docInfo := range docCategories {
			createReq := CreateDocumentRequest{
				Title:    docInfo.title,
				Content:  docInfo.content,
				Category: docInfo.category,
				Tags:     docInfo.tags,
				IsPublic: true,
			}

			doc, err := suite.kbService.CreateDocument(&createReq, suite.testUserID, "author@company.com")
			require.NoError(suite.T(), err)
			createdDocs = append(createdDocs, doc.ID)
			suite.testDocs = append(suite.testDocs, doc.ID)
		}

		// 3. 更新文档内容
		updateReq := UpdateDocumentRequest{
			Title:     "项目架构设计v2.0",
			Content:   "# 项目架构v2.0\n\n本项目采用微服务架构，新增了服务网格...",
			ChangeLog: "添加了服务网格相关内容",
		}

		updatedDoc, err := suite.kbService.UpdateDocument(createdDocs[0], &updateReq, suite.testUserID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 2, updatedDoc.Version)

		// 4. 查看版本历史
		versions, err := suite.kbService.GetDocumentVersions(createdDocs[0], suite.testUserID)
		require.NoError(suite.T(), err)
		assert.Len(suite.T(), versions, 2)
		assert.Equal(suite.T(), "初始版本", versions[0].ChangeLog)
		assert.Equal(suite.T(), "添加了服务网格相关内容", versions[1].ChangeLog)

		// 5. 按分类获取文档
		searchReq := &SearchDocumentsRequest{
			Category: "API文档",
			Page:     1,
			PageSize: 10,
		}

		result, err := suite.kbService.GetDocuments(searchReq, suite.testUserID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), int64(1), result.Total)
		assert.Equal(suite.T(), "API接口文档", result.Documents[0].Title)

		// 6. 全文搜索
		searchResult, err := suite.kbService.SearchDocuments("Docker", suite.testUserID, suite.testTenantID)
		require.NoError(suite.T(), err)
		assert.Greater(suite.T(), searchResult.Total, int64(0))

		// 验证搜索结果
		foundDockerDoc := false
		for _, doc := range searchResult.Documents {
			if strings.Contains(strings.ToLower(doc.Content), "docker") {
				foundDockerDoc = true
				break
			}
		}
		assert.True(suite.T(), foundDockerDoc, "应该找到包含Docker的文档")

		// 7. 验证文档数量和质量
		allDocsReq := &SearchDocumentsRequest{
			Page:     1,
			PageSize: 100,
		}

		allResult, err := suite.kbService.GetDocuments(allDocsReq, suite.testUserID)
		require.NoError(suite.T(), err)
		assert.GreaterOrEqual(suite.T(), allResult.Total, int64(3), "应该至少有3个文档")

		// 8. 清理测试数据
		for _, docID := range createdDocs {
			err := suite.kbService.DeleteDocument(docID, suite.testUserID)
			require.NoError(suite.T(), err)
		}

		// 验证文档已删除
		for _, docID := range createdDocs {
			_, err := suite.kbService.GetDocument(docID, suite.testUserID)
			assert.Error(suite.T(), err, "文档应该已被删除")
		}

		suite.T().Log("✅ 完整知识库管理工作流程测试通过")
	})
}

// ================================
// 测试套件运行器
// ================================

func TestKBServiceIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过KB服务集成测试")
	}

	suite.Run(t, new(KBServiceIntegrationTestSuite))
}

// 基准测试
func BenchmarkKBServiceOperations(b *testing.B) {
	service := NewMockKBService()
	userID := uuid.New()

	b.Run("CreateDocument", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := &CreateDocumentRequest{
				Title:    fmt.Sprintf("基准测试文档_%d", i),
				Content:  "这是基准测试的内容",
				Category: "基准测试",
				Tags:     []string{"基准", "测试"},
				IsPublic: true,
			}
			service.CreateDocument(req, userID, "bench@example.com")
		}
	})

	// 先创建一些文档用于搜索基准测试
	for i := 0; i < 100; i++ {
		req := &CreateDocumentRequest{
			Title:    fmt.Sprintf("搜索基准测试_%d", i),
			Content:  fmt.Sprintf("搜索基准测试内容_%d", i),
			Category: "基准测试",
			Tags:     []string{"搜索", "基准"},
			IsPublic: true,
		}
		service.CreateDocument(req, userID, "bench@example.com")
	}

	b.Run("SearchDocuments", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			service.SearchDocuments("搜索", userID, uuid.New())
		}
	})

	b.Run("GetDocuments", func(b *testing.B) {
		req := &SearchDocumentsRequest{
			Page:     1,
			PageSize: 10,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			service.GetDocuments(req, userID)
		}
	})
}
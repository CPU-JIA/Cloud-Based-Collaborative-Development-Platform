package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/handlers"
	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/models"
	"github.com/cloud-platform/collaborative-dev/shared/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// GitGatewayPerformanceTestSuite Git网关性能测试套件
type GitGatewayPerformanceTestSuite struct {
	suite.Suite
	logger             *zap.Logger
	router             *gin.Engine
	server             *httptest.Server
	mockGitService     *MockGitPerformanceService
	gitHandler         *handlers.GitHandler
	testTenantID       uuid.UUID
	testUserID         uuid.UUID
	testProjectID      uuid.UUID
	authToken          string
	performanceMetrics *GitGatewayPerformanceMetrics
}

// GitGatewayPerformanceMetrics Git网关性能指标
type GitGatewayPerformanceMetrics struct {
	mu                  sync.RWMutex
	TotalRequests       int64                             `json:"total_requests"`
	TotalResponseTime   time.Duration                     `json:"total_response_time"`
	AverageResponseTime time.Duration                     `json:"average_response_time"`
	MinResponseTime     time.Duration                     `json:"min_response_time"`
	MaxResponseTime     time.Duration                     `json:"max_response_time"`
	ErrorCount          int64                             `json:"error_count"`
	SuccessRate         float64                           `json:"success_rate"`
	EndpointMetrics     map[string]*GitGatewayEndpointMetric `json:"endpoint_metrics"`
	ConcurrencyMetrics  map[int]*ConcurrencyMetric        `json:"concurrency_metrics"`
	OperationMetrics    map[string]*OperationMetric       `json:"operation_metrics"`
}

// GitGatewayEndpointMetric 端点性能指标
type GitGatewayEndpointMetric struct {
	RequestCount      int64         `json:"request_count"`
	TotalResponseTime time.Duration `json:"total_response_time"`
	AverageTime       time.Duration `json:"average_time"`
	MinTime           time.Duration `json:"min_time"`
	MaxTime           time.Duration `json:"max_time"`
	ErrorCount        int64         `json:"error_count"`
	SuccessRate       float64       `json:"success_rate"`
}

// ConcurrencyMetric 并发性能指标
type ConcurrencyMetric struct {
	ConcurrentUsers   int           `json:"concurrent_users"`
	TotalRequests     int64         `json:"total_requests"`
	TotalTime         time.Duration `json:"total_time"`
	RequestsPerSecond float64       `json:"requests_per_second"`
	ErrorRate         float64       `json:"error_rate"`
}

// OperationMetric 操作性能指标
type OperationMetric struct {
	OperationType     string        `json:"operation_type"`
	TotalOperations   int64         `json:"total_operations"`
	TotalTime         time.Duration `json:"total_time"`
	AverageTime       time.Duration `json:"average_time"`
	OperationsPerSec  float64       `json:"operations_per_second"`
	ErrorCount        int64         `json:"error_count"`
}

// MockGitPerformanceService Git服务性能测试模拟
type MockGitPerformanceService struct {
	repositories map[uuid.UUID]*models.Repository
	branches     map[uuid.UUID][]*models.Branch
	commits      map[uuid.UUID][]*models.Commit
	tags         map[uuid.UUID][]*models.Tag
	logger       *zap.Logger
	requestDelay time.Duration // 模拟网络延迟
	errorRate    float64       // 模拟错误率
	mu           sync.RWMutex
}

func NewMockGitPerformanceService(logger *zap.Logger) *MockGitPerformanceService {
	return &MockGitPerformanceService{
		repositories: make(map[uuid.UUID]*models.Repository),
		branches:     make(map[uuid.UUID][]*models.Branch),
		commits:      make(map[uuid.UUID][]*models.Commit),
		tags:         make(map[uuid.UUID][]*models.Tag),
		logger:       logger,
		requestDelay: time.Millisecond * 10, // 默认10ms延迟
		errorRate:    0.0,                   // 默认无错误
	}
}

// 设置性能测试参数
func (m *MockGitPerformanceService) SetRequestDelay(delay time.Duration) {
	m.requestDelay = delay
}

func (m *MockGitPerformanceService) SetErrorRate(rate float64) {
	m.errorRate = rate
}

// 模拟网络延迟和错误
func (m *MockGitPerformanceService) simulateNetworkConditions() error {
	if m.requestDelay > 0 {
		time.Sleep(m.requestDelay)
	}

	if m.errorRate > 0 {
		if time.Now().UnixNano()%100 < int64(m.errorRate*100) {
			return fmt.Errorf("simulated network error")
		}
	}

	return nil
}

// 实现Git服务接口（简化版）
func (m *MockGitPerformanceService) CreateRepository(ctx context.Context, req *models.CreateRepositoryRequest, userID uuid.UUID) (*models.Repository, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	projectID, _ := uuid.Parse(req.ProjectID)
	repository := &models.Repository{
		ID:          uuid.New(),
		ProjectID:   projectID,
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
		Status:      models.RepositoryStatusActive,
		GitPath:     fmt.Sprintf("/git/repos/%s", req.Name),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.repositories[repository.ID] = repository
	return repository, nil
}

func (m *MockGitPerformanceService) GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	repository, exists := m.repositories[id]
	if !exists {
		return nil, fmt.Errorf("repository not found")
	}

	return repository, nil
}

func (m *MockGitPerformanceService) ListRepositories(ctx context.Context, projectID *uuid.UUID, page, pageSize int) (*models.RepositoryListResponse, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var repositories []models.Repository
	for _, repo := range m.repositories {
		if projectID == nil || repo.ProjectID == *projectID {
			repositories = append(repositories, *repo)
		}
	}

	total := int64(len(repositories))
	start := (page - 1) * pageSize
	end := start + pageSize

	if start > len(repositories) {
		repositories = []models.Repository{}
	} else if end > len(repositories) {
		repositories = repositories[start:]
	} else {
		repositories = repositories[start:end]
	}

	return &models.RepositoryListResponse{
		Repositories: repositories,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

func (m *MockGitPerformanceService) UpdateRepository(ctx context.Context, id uuid.UUID, req *models.UpdateRepositoryRequest) (*models.Repository, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	repository, exists := m.repositories[id]
	if !exists {
		return nil, fmt.Errorf("repository not found")
	}

	if req.Name != nil {
		repository.Name = *req.Name
	}
	repository.UpdatedAt = time.Now()

	return repository, nil
}

func (m *MockGitPerformanceService) DeleteRepository(ctx context.Context, id uuid.UUID) error {
	if err := m.simulateNetworkConditions(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.repositories[id]; !exists {
		return fmt.Errorf("repository not found")
	}

	delete(m.repositories, id)
	return nil
}

func (m *MockGitPerformanceService) GetRepositoryStats(ctx context.Context, id uuid.UUID) (*models.RepositoryStats, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.repositories[id]; !exists {
		return nil, fmt.Errorf("repository not found")
	}

	return &models.RepositoryStats{
		Size:        1024 * 1024, // 1MB
		CommitCount: int64(len(m.commits[id])),
		BranchCount: int64(len(m.branches[id])),
		TagCount:    int64(len(m.tags[id])),
	}, nil
}

func (m *MockGitPerformanceService) SearchRepositories(ctx context.Context, query string, projectID *uuid.UUID, page, pageSize int) (*models.RepositoryListResponse, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	// 简化搜索逻辑
	return m.ListRepositories(ctx, projectID, page, pageSize)
}

// 分支管理接口
func (m *MockGitPerformanceService) CreateBranch(ctx context.Context, repositoryID uuid.UUID, req *models.CreateBranchRequest) (*models.Branch, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	branch := &models.Branch{
		ID:           uuid.New(),
		RepositoryID: repositoryID,
		Name:         req.Name,
		CommitSHA:    req.FromSHA,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if m.branches[repositoryID] == nil {
		m.branches[repositoryID] = []*models.Branch{}
	}
	m.branches[repositoryID] = append(m.branches[repositoryID], branch)

	return branch, nil
}

func (m *MockGitPerformanceService) ListBranches(ctx context.Context, repositoryID uuid.UUID) ([]*models.Branch, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	branches := m.branches[repositoryID]
	if branches == nil {
		return []*models.Branch{}, nil
	}

	return branches, nil
}

func (m *MockGitPerformanceService) GetBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) (*models.Branch, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	branches := m.branches[repositoryID]
	for _, branch := range branches {
		if branch.Name == branchName {
			return branch, nil
		}
	}

	return nil, fmt.Errorf("branch not found")
}

func (m *MockGitPerformanceService) DeleteBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	if err := m.simulateNetworkConditions(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	branches := m.branches[repositoryID]
	for i, branch := range branches {
		if branch.Name == branchName {
			m.branches[repositoryID] = append(branches[:i], branches[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("branch not found")
}

func (m *MockGitPerformanceService) SetDefaultBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	if err := m.simulateNetworkConditions(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	repository, exists := m.repositories[repositoryID]
	if !exists {
		return fmt.Errorf("repository not found")
	}

	repository.DefaultBranch = branchName
	repository.UpdatedAt = time.Now()

	return nil
}

func (m *MockGitPerformanceService) MergeBranch(ctx context.Context, repositoryID uuid.UUID, targetBranch, sourceBranch string) error {
	if err := m.simulateNetworkConditions(); err != nil {
		return err
	}

	// 简化合并逻辑，只做延迟模拟
	return nil
}

// 提交管理接口
func (m *MockGitPerformanceService) CreateCommit(ctx context.Context, repositoryID uuid.UUID, req *models.CreateCommitRequest) (*models.Commit, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	commit := &models.Commit{
		ID:             uuid.New(),
		RepositoryID:   repositoryID,
		SHA:            fmt.Sprintf("commit%d", time.Now().Unix()),
		Message:        req.Message,
		Author:         req.Author.Name,
		AuthorEmail:    req.Author.Email,
		ChangedFiles:   int32(len(req.Files)),
		CommittedAt:    time.Now(),
		CreatedAt:      time.Now(),
	}

	if m.commits[repositoryID] == nil {
		m.commits[repositoryID] = []*models.Commit{}
	}
	m.commits[repositoryID] = append(m.commits[repositoryID], commit)

	return commit, nil
}

func (m *MockGitPerformanceService) ListCommits(ctx context.Context, repositoryID uuid.UUID, branch string, page, pageSize int) (*models.CommitListResponse, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	commits := m.commits[repositoryID]
	if commits == nil {
		commits = []*models.Commit{}
	}

	commitList := make([]models.Commit, len(commits))
	for i, commit := range commits {
		commitList[i] = *commit
	}

	total := int64(len(commitList))
	start := (page - 1) * pageSize
	end := start + pageSize

	if start > len(commitList) {
		commitList = []models.Commit{}
	} else if end > len(commitList) {
		commitList = commitList[start:]
	} else {
		commitList = commitList[start:end]
	}

	return &models.CommitListResponse{
		Commits:  commitList,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (m *MockGitPerformanceService) GetCommit(ctx context.Context, repositoryID uuid.UUID, sha string) (*models.Commit, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	commits := m.commits[repositoryID]
	for _, commit := range commits {
		if commit.SHA == sha {
			return commit, nil
		}
	}

	return nil, fmt.Errorf("commit not found")
}

func (m *MockGitPerformanceService) GetCommitDiff(ctx context.Context, repositoryID uuid.UUID, sha string) (*models.GitDiff, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	return &models.GitDiff{
		FromSHA:      "previous_sha",
		ToSHA:        sha,
		Files:        []models.DiffFile{},
		TotalAdded:   10,
		TotalDeleted: 5,
	}, nil
}

func (m *MockGitPerformanceService) CompareBranches(ctx context.Context, repositoryID uuid.UUID, base, head string) (*models.GitDiff, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	return &models.GitDiff{
		FromSHA:      "base_sha",
		ToSHA:        "head_sha",
		Files:        []models.DiffFile{},
		TotalAdded:   15,
		TotalDeleted: 8,
	}, nil
}

// 标签管理接口
func (m *MockGitPerformanceService) CreateTag(ctx context.Context, repositoryID uuid.UUID, req *models.CreateTagRequest) (*models.Tag, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tag := &models.Tag{
		ID:           uuid.New(),
		RepositoryID: repositoryID,
		Name:         req.Name,
		CommitSHA:    req.CommitSHA,
		Message:      req.Message,
		CreatedAt:    time.Now(),
	}

	if m.tags[repositoryID] == nil {
		m.tags[repositoryID] = []*models.Tag{}
	}
	m.tags[repositoryID] = append(m.tags[repositoryID], tag)

	return tag, nil
}

func (m *MockGitPerformanceService) ListTags(ctx context.Context, repositoryID uuid.UUID) ([]*models.Tag, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	tags := m.tags[repositoryID]
	if tags == nil {
		return []*models.Tag{}, nil
	}

	return tags, nil
}

func (m *MockGitPerformanceService) GetTag(ctx context.Context, repositoryID uuid.UUID, tagName string) (*models.Tag, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	tags := m.tags[repositoryID]
	for _, tag := range tags {
		if tag.Name == tagName {
			return tag, nil
		}
	}

	return nil, fmt.Errorf("tag not found")
}

func (m *MockGitPerformanceService) DeleteTag(ctx context.Context, repositoryID uuid.UUID, tagName string) error {
	if err := m.simulateNetworkConditions(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tags := m.tags[repositoryID]
	for i, tag := range tags {
		if tag.Name == tagName {
			m.tags[repositoryID] = append(tags[:i], tags[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("tag not found")
}

// 文件操作接口
func (m *MockGitPerformanceService) GetFileContent(ctx context.Context, repositoryID uuid.UUID, branch, filePath string) ([]byte, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	content := fmt.Sprintf("Mock file content for %s", filePath)
	return []byte(content), nil
}

func (m *MockGitPerformanceService) GetDirectoryContent(ctx context.Context, repositoryID uuid.UUID, branch, dirPath string) ([]*models.FileInfo, error) {
	if err := m.simulateNetworkConditions(); err != nil {
		return nil, err
	}

	files := []*models.FileInfo{
		{
			Name: "README.md",
			Type: "file",
			Size: 1024,
		},
		{
			Name: "src",
			Type: "directory",
			Size: 0,
		},
	}

	return files, nil
}

// SetupSuite 设置测试套件
func (suite *GitGatewayPerformanceTestSuite) SetupSuite() {
	// 初始化日志
	logger, _ := zap.NewDevelopment()
	suite.logger = logger

	// 初始化测试数据
	suite.testTenantID = uuid.New()
	suite.testUserID = uuid.New()
	suite.testProjectID = uuid.New()
	suite.authToken = "performance-test-auth-token"

	// 初始化Mock Git服务
	suite.mockGitService = NewMockGitPerformanceService(suite.logger)

	// 初始化Git处理器
	suite.gitHandler = handlers.NewGitHandler(suite.mockGitService, suite.logger)

	// 初始化性能指标
	suite.performanceMetrics = &GitGatewayPerformanceMetrics{
		MinResponseTime:    time.Hour,
		EndpointMetrics:    make(map[string]*GitGatewayEndpointMetric),
		ConcurrencyMetrics: make(map[int]*ConcurrencyMetric),
		OperationMetrics:   make(map[string]*OperationMetric),
	}

	// 设置路由
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.setupRoutes()

	// 创建测试服务器
	suite.server = httptest.NewServer(suite.router)

	suite.logger.Info("Git网关性能测试套件初始化完成",
		zap.String("server_url", suite.server.URL))
}

// TearDownSuite 清理测试套件
func (suite *GitGatewayPerformanceTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}

	// 生成性能测试报告
	suite.generatePerformanceReport()

	suite.logger.Info("Git网关性能测试套件清理完成")
}

// setupRoutes 设置路由
func (suite *GitGatewayPerformanceTestSuite) setupRoutes() {
	// 添加认证中间件模拟器
	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", suite.testUserID.String())
		c.Set("tenant_id", suite.testTenantID.String())
		c.Next()
	})

	v1 := suite.router.Group("/api/v1")
	
	// 仓库管理路由
	repositories := v1.Group("/repositories")
	{
		repositories.POST("", suite.gitHandler.CreateRepository)
		repositories.GET("", suite.gitHandler.ListRepositories)
		repositories.GET("/:id", suite.gitHandler.GetRepository)
		repositories.PUT("/:id", suite.gitHandler.UpdateRepository)
		repositories.DELETE("/:id", suite.gitHandler.DeleteRepository)
		repositories.GET("/:id/stats", suite.gitHandler.GetRepositoryStats)
		repositories.GET("/search", suite.gitHandler.SearchRepositories)
	}

	// 分支管理路由
	branches := v1.Group("/repositories/:id/branches")
	{
		branches.POST("", suite.gitHandler.CreateBranch)
		branches.GET("", suite.gitHandler.ListBranches)
		branches.GET("/:branch", suite.gitHandler.GetBranch)
		branches.DELETE("/:branch", suite.gitHandler.DeleteBranch)
		branches.PUT("/default", suite.gitHandler.SetDefaultBranch)
		branches.POST("/merge", suite.gitHandler.MergeBranch)
	}

	// 提交管理路由
	commits := v1.Group("/repositories/:id/commits")
	{
		commits.POST("", suite.gitHandler.CreateCommit)
		commits.GET("", suite.gitHandler.ListCommits)
		commits.GET("/:sha", suite.gitHandler.GetCommit)
		commits.GET("/:sha/diff", suite.gitHandler.GetCommitDiff)
		commits.GET("/compare", suite.gitHandler.CompareBranches)
	}

	// 标签管理路由
	tags := v1.Group("/repositories/:id/tags")
	{
		tags.POST("", suite.gitHandler.CreateTag)
		tags.GET("", suite.gitHandler.ListTags)
		tags.GET("/:tag", suite.gitHandler.GetTag)
		tags.DELETE("/:tag", suite.gitHandler.DeleteTag)
	}

	// 文件操作路由
	files := v1.Group("/repositories/:id/files")
	{
		files.GET("/content", suite.gitHandler.GetFileContent)
		files.GET("/directory", suite.gitHandler.GetDirectoryContent)
	}
}

// makePerformanceRequest 发送HTTP请求并记录性能指标
func (suite *GitGatewayPerformanceTestSuite) makePerformanceRequest(method, endpoint string, body interface{}, authToken string) (*http.Response, []byte, time.Duration) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(bodyBytes)
	}

	url := suite.server.URL + endpoint
	req, _ := http.NewRequest(method, url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	startTime := time.Now()
	resp, err := http.DefaultClient.Do(req)
	responseTime := time.Since(startTime)

	var respBody []byte
	if err == nil && resp != nil {
		respBody, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
	}

	// 记录性能指标
	suite.recordPerformanceMetrics(endpoint, resp.StatusCode, responseTime, err)

	return resp, respBody, responseTime
}

// recordPerformanceMetrics 记录性能指标
func (suite *GitGatewayPerformanceTestSuite) recordPerformanceMetrics(endpoint string, statusCode int, responseTime time.Duration, err error) {
	suite.performanceMetrics.mu.Lock()
	defer suite.performanceMetrics.mu.Unlock()

	// 全局指标
	suite.performanceMetrics.TotalRequests++
	suite.performanceMetrics.TotalResponseTime += responseTime

	if responseTime < suite.performanceMetrics.MinResponseTime {
		suite.performanceMetrics.MinResponseTime = responseTime
	}
	if responseTime > suite.performanceMetrics.MaxResponseTime {
		suite.performanceMetrics.MaxResponseTime = responseTime
	}

	if err != nil || statusCode >= 400 {
		suite.performanceMetrics.ErrorCount++
	}

	// 端点指标
	if suite.performanceMetrics.EndpointMetrics[endpoint] == nil {
		suite.performanceMetrics.EndpointMetrics[endpoint] = &GitGatewayEndpointMetric{
			MinTime: time.Hour,
		}
	}

	endpointMetric := suite.performanceMetrics.EndpointMetrics[endpoint]
	endpointMetric.RequestCount++
	endpointMetric.TotalResponseTime += responseTime

	if responseTime < endpointMetric.MinTime {
		endpointMetric.MinTime = responseTime
	}
	if responseTime > endpointMetric.MaxTime {
		endpointMetric.MaxTime = responseTime
	}

	if err != nil || statusCode >= 400 {
		endpointMetric.ErrorCount++
	}
}

// TestRepositoryOperationsPerformance 测试仓库操作性能
func (suite *GitGatewayPerformanceTestSuite) TestRepositoryOperationsPerformance() {
	suite.Run("仓库操作性能测试", func() {
		const numOperations = 100
		repositories := make([]uuid.UUID, 0, numOperations)

		suite.logger.Info("开始仓库操作性能测试", zap.Int("operations", numOperations))

		startTime := time.Now()

		// 创建仓库
		for i := 0; i < numOperations; i++ {
			createReq := models.CreateRepositoryRequest{
				ProjectID:     suite.testProjectID.String(),
				Name:          fmt.Sprintf("perf-test-repo-%d", i),
				Description:   stringPtr(fmt.Sprintf("性能测试仓库%d", i)),
				Visibility:    models.RepositoryVisibilityPrivate,
				DefaultBranch: stringPtr("main"),
				InitReadme:    false,
			}

			resp, body, responseTime := suite.makePerformanceRequest("POST", "/api/v1/repositories", createReq, suite.authToken)
			
			suite.logger.Debug("创建仓库请求",
				zap.Int("index", i),
				zap.Int("status_code", resp.StatusCode),
				zap.Duration("response_time", responseTime))
			
			assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

			var createResponse response.Response
			err := json.Unmarshal(body, &createResponse)
			assert.NoError(suite.T(), err)

			if createResponse.Data != nil {
				repoData, _ := json.Marshal(createResponse.Data)
				var repository models.Repository
				err = json.Unmarshal(repoData, &repository)
				if err == nil {
					repositories = append(repositories, repository.ID)
				}
			}
		}

		// 读取仓库
		for i, repoID := range repositories {
			resp, _, responseTime := suite.makePerformanceRequest("GET", fmt.Sprintf("/api/v1/repositories/%s", repoID), nil, suite.authToken)
			
			suite.logger.Debug("读取仓库请求",
				zap.Int("index", i),
				zap.String("repo_id", repoID.String()),
				zap.Int("status_code", resp.StatusCode),
				zap.Duration("response_time", responseTime))
			
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		}

		// 更新仓库
		for i, repoID := range repositories {
			updateReq := models.UpdateRepositoryRequest{
				Name: stringPtr(fmt.Sprintf("updated-perf-test-repo-%d", i)),
			}

			resp, _, responseTime := suite.makePerformanceRequest("PUT", fmt.Sprintf("/api/v1/repositories/%s", repoID), updateReq, suite.authToken)
			
			suite.logger.Debug("更新仓库请求",
				zap.Int("index", i),
				zap.String("repo_id", repoID.String()),
				zap.Int("status_code", resp.StatusCode),
				zap.Duration("response_time", responseTime))
			
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		}

		// 删除仓库
		for i, repoID := range repositories {
			resp, _, responseTime := suite.makePerformanceRequest("DELETE", fmt.Sprintf("/api/v1/repositories/%s", repoID), nil, suite.authToken)
			
			suite.logger.Debug("删除仓库请求",
				zap.Int("index", i),
				zap.String("repo_id", repoID.String()),
				zap.Int("status_code", resp.StatusCode),
				zap.Duration("response_time", responseTime))
			
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		}

		totalTime := time.Since(startTime)
		totalOperations := numOperations * 4 // 每个仓库4个操作

		// 记录操作性能指标
		suite.recordOperationMetrics("repository_crud", int64(totalOperations), totalTime, 0)

		suite.logger.Info("仓库操作性能测试完成",
			zap.Int("operations", totalOperations),
			zap.Duration("total_time", totalTime),
			zap.Float64("ops_per_second", float64(totalOperations)/totalTime.Seconds()))
	})
}

// TestConcurrentRepositoryAccess 测试并发仓库访问
func (suite *GitGatewayPerformanceTestSuite) TestConcurrentRepositoryAccess() {
	suite.Run("并发仓库访问测试", func() {
		concurrencyLevels := []int{1, 5, 10, 20, 50}
		
		// 先创建一个测试仓库
		createReq := models.CreateRepositoryRequest{
			ProjectID:     suite.testProjectID.String(),
			Name:          "concurrent-test-repo",
			Description:   stringPtr("并发测试仓库"),
			Visibility:    models.RepositoryVisibilityPrivate,
			DefaultBranch: stringPtr("main"),
			InitReadme:    false,
		}

		resp, body, _ := suite.makePerformanceRequest("POST", "/api/v1/repositories", createReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		var createResponse response.Response
		err := json.Unmarshal(body, &createResponse)
		assert.NoError(suite.T(), err)

		repoData, _ := json.Marshal(createResponse.Data)
		var repository models.Repository
		err = json.Unmarshal(repoData, &repository)
		assert.NoError(suite.T(), err)

		// 测试不同并发级别
		for _, concurrency := range concurrencyLevels {
			suite.testConcurrentAccess(repository.ID, concurrency)
		}

		// 清理测试仓库
		suite.makePerformanceRequest("DELETE", fmt.Sprintf("/api/v1/repositories/%s", repository.ID), nil, suite.authToken)
	})
}

// testConcurrentAccess 测试指定并发级别的访问
func (suite *GitGatewayPerformanceTestSuite) testConcurrentAccess(repositoryID uuid.UUID, concurrency int) {
	suite.Run(fmt.Sprintf("并发级别%d", concurrency), func() {
		const operationsPerUser = 20
		var wg sync.WaitGroup
		startTime := time.Now()
		
		suite.logger.Info("开始并发访问测试",
			zap.Int("concurrency", concurrency),
			zap.Int("operations_per_user", operationsPerUser))
		
		for user := 0; user < concurrency; user++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()
				
				for op := 0; op < operationsPerUser; op++ {
					// 随机选择操作类型
					switch op % 4 {
					case 0:
						// 获取仓库信息
						suite.makePerformanceRequest("GET", fmt.Sprintf("/api/v1/repositories/%s", repositoryID), nil, suite.authToken)
					case 1:
						// 获取仓库统计
						suite.makePerformanceRequest("GET", fmt.Sprintf("/api/v1/repositories/%s/stats", repositoryID), nil, suite.authToken)
					case 2:
						// 列出分支
						suite.makePerformanceRequest("GET", fmt.Sprintf("/api/v1/repositories/%s/branches", repositoryID), nil, suite.authToken)
					case 3:
						// 列out提交
						suite.makePerformanceRequest("GET", fmt.Sprintf("/api/v1/repositories/%s/commits", repositoryID), nil, suite.authToken)
					}
				}
			}(user)
		}
		
		wg.Wait()
		totalTime := time.Since(startTime)
		totalRequests := int64(concurrency * operationsPerUser)
		
		// 记录并发性能指标
		suite.performanceMetrics.mu.Lock()
		suite.performanceMetrics.ConcurrencyMetrics[concurrency] = &ConcurrencyMetric{
			ConcurrentUsers:   concurrency,
			TotalRequests:     totalRequests,
			TotalTime:         totalTime,
			RequestsPerSecond: float64(totalRequests) / totalTime.Seconds(),
			ErrorRate:         0.0, // 简化错误率计算
		}
		suite.performanceMetrics.mu.Unlock()
		
		suite.logger.Info("并发访问测试完成",
			zap.Int("concurrency", concurrency),
			zap.Int64("total_requests", totalRequests),
			zap.Duration("total_time", totalTime),
			zap.Float64("requests_per_second", float64(totalRequests)/totalTime.Seconds()))
	})
}

// TestNetworkConditionsImpact 测试网络条件对性能的影响
func (suite *GitGatewayPerformanceTestSuite) TestNetworkConditionsImpact() {
	suite.Run("网络条件影响测试", func() {
		// 测试不同网络延迟
		delays := []time.Duration{
			0,                    // 无延迟
			time.Millisecond * 10, // 10ms延迟
			time.Millisecond * 50, // 50ms延迟
			time.Millisecond * 100, // 100ms延迟
			time.Millisecond * 200, // 200ms延迟
		}

		for _, delay := range delays {
			suite.testWithNetworkDelay(delay)
		}

		// 测试不同错误率
		errorRates := []float64{0.0, 0.01, 0.05, 0.1} // 0%, 1%, 5%, 10%

		for _, errorRate := range errorRates {
			suite.testWithErrorRate(errorRate)
		}
	})
}

// testWithNetworkDelay 测试指定网络延迟下的性能
func (suite *GitGatewayPerformanceTestSuite) testWithNetworkDelay(delay time.Duration) {
	suite.Run(fmt.Sprintf("网络延迟%v", delay), func() {
		// 设置网络延迟
		suite.mockGitService.SetRequestDelay(delay)
		defer suite.mockGitService.SetRequestDelay(0)

		const numRequests = 50
		startTime := time.Now()

		for i := 0; i < numRequests; i++ {
			resp, _, _ := suite.makePerformanceRequest("GET", "/api/v1/repositories", nil, suite.authToken)
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		}

		totalTime := time.Since(startTime)
		avgTime := totalTime / numRequests

		suite.logger.Info("网络延迟测试完成",
			zap.Duration("network_delay", delay),
			zap.Int("requests", numRequests),
			zap.Duration("total_time", totalTime),
			zap.Duration("avg_time", avgTime))
	})
}

// testWithErrorRate 测试指定错误率下的性能
func (suite *GitGatewayPerformanceTestSuite) testWithErrorRate(errorRate float64) {
	suite.Run(fmt.Sprintf("错误率%.1f%%", errorRate*100), func() {
		// 设置错误率
		suite.mockGitService.SetErrorRate(errorRate)
		defer suite.mockGitService.SetErrorRate(0.0)

		const numRequests = 100
		var successCount, errorCount int

		for i := 0; i < numRequests; i++ {
			resp, _, _ := suite.makePerformanceRequest("GET", "/api/v1/repositories", nil, suite.authToken)
			if resp.StatusCode < 400 {
				successCount++
			} else {
				errorCount++
			}
		}

		actualErrorRate := float64(errorCount) / float64(numRequests)

		suite.logger.Info("错误率测试完成",
			zap.Float64("expected_error_rate", errorRate),
			zap.Float64("actual_error_rate", actualErrorRate),
			zap.Int("success_count", successCount),
			zap.Int("error_count", errorCount))
	})
}

// TestGitOperationsPerformance 测试Git操作性能
func (suite *GitGatewayPerformanceTestSuite) TestGitOperationsPerformance() {
	suite.Run("Git操作性能测试", func() {
		// 先创建测试仓库
		createReq := models.CreateRepositoryRequest{
			ProjectID:     suite.testProjectID.String(),
			Name:          "git-ops-test-repo",
			Description:   stringPtr("Git操作测试仓库"),
			Visibility:    models.RepositoryVisibilityPrivate,
			DefaultBranch: stringPtr("main"),
			InitReadme:    true,
		}

		resp, body, _ := suite.makePerformanceRequest("POST", "/api/v1/repositories", createReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		var createResponse response.Response
		err := json.Unmarshal(body, &createResponse)
		assert.NoError(suite.T(), err)

		repoData, _ := json.Marshal(createResponse.Data)
		var repository models.Repository
		err = json.Unmarshal(repoData, &repository)
		assert.NoError(suite.T(), err)

		// 测试分支操作性能
		suite.testBranchOperationsPerformance(repository.ID)

		// 测试提交操作性能
		suite.testCommitOperationsPerformance(repository.ID)

		// 测试标签操作性能
		suite.testTagOperationsPerformance(repository.ID)

		// 清理测试仓库
		suite.makePerformanceRequest("DELETE", fmt.Sprintf("/api/v1/repositories/%s", repository.ID), nil, suite.authToken)
	})
}

// testBranchOperationsPerformance 测试分支操作性能
func (suite *GitGatewayPerformanceTestSuite) testBranchOperationsPerformance(repositoryID uuid.UUID) {
	suite.Run("分支操作性能", func() {
		const numBranches = 50
		branches := make([]string, 0, numBranches)

		startTime := time.Now()

		// 创建分支
		for i := 0; i < numBranches; i++ {
			createReq := models.CreateBranchRequest{
				Name:    fmt.Sprintf("feature/perf-test-%d", i),
				FromSHA: "initial_commit_sha",
			}

			resp, body, _ := suite.makePerformanceRequest("POST", fmt.Sprintf("/api/v1/repositories/%s/branches", repositoryID), createReq, suite.authToken)
			assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

			var createResponse response.Response
			err := json.Unmarshal(body, &createResponse)
			assert.NoError(suite.T(), err)

			branchData, _ := json.Marshal(createResponse.Data)
			var branch models.Branch
			err = json.Unmarshal(branchData, &branch)
			if err == nil {
				branches = append(branches, branch.Name)
			}
		}

		// 列出分支
		resp, _, _ := suite.makePerformanceRequest("GET", fmt.Sprintf("/api/v1/repositories/%s/branches", repositoryID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		// 删除分支
		for _, branchName := range branches {
			resp, _, _ := suite.makePerformanceRequest("DELETE", fmt.Sprintf("/api/v1/repositories/%s/branches/%s", repositoryID, branchName), nil, suite.authToken)
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		}

		totalTime := time.Since(startTime)
		totalOperations := int64(numBranches*2 + 1) // 创建+删除+列出

		suite.recordOperationMetrics("branch_operations", totalOperations, totalTime, 0)

		suite.logger.Info("分支操作性能测试完成",
			zap.Int("branches", numBranches),
			zap.Int64("operations", totalOperations),
			zap.Duration("total_time", totalTime))
	})
}

// testCommitOperationsPerformance 测试提交操作性能
func (suite *GitGatewayPerformanceTestSuite) testCommitOperationsPerformance(repositoryID uuid.UUID) {
	suite.Run("提交操作性能", func() {
		const numCommits = 30
		commits := make([]string, 0, numCommits)

		startTime := time.Now()

		// 创建提交
		for i := 0; i < numCommits; i++ {
			createReq := models.CreateCommitRequest{
				Branch:  "main",
				Message: fmt.Sprintf("性能测试提交 %d", i),
				Author: models.CommitAuthor{
					Name:  "Performance Test User",
					Email: "perf@test.com",
				},
				Files: []models.CreateCommitFile{
					{
						Path:    fmt.Sprintf("test%d.txt", i),
						Content: fmt.Sprintf("测试文件内容 %d", i),
						Mode:    "100644",
					},
				},
			}

			resp, body, _ := suite.makePerformanceRequest("POST", fmt.Sprintf("/api/v1/repositories/%s/commits", repositoryID), createReq, suite.authToken)
			assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

			var createResponse response.Response
			err := json.Unmarshal(body, &createResponse)
			assert.NoError(suite.T(), err)

			commitData, _ := json.Marshal(createResponse.Data)
			var commit models.Commit
			err = json.Unmarshal(commitData, &commit)
			if err == nil {
				commits = append(commits, commit.SHA)
			}
		}

		// 列出提交
		resp, _, _ := suite.makePerformanceRequest("GET", fmt.Sprintf("/api/v1/repositories/%s/commits", repositoryID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		// 获取提交详情
		for _, commitSHA := range commits {
			resp, _, _ := suite.makePerformanceRequest("GET", fmt.Sprintf("/api/v1/repositories/%s/commits/%s", repositoryID, commitSHA), nil, suite.authToken)
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		}

		totalTime := time.Since(startTime)
		totalOperations := int64(numCommits*2 + 1) // 创建+获取+列出

		suite.recordOperationMetrics("commit_operations", totalOperations, totalTime, 0)

		suite.logger.Info("提交操作性能测试完成",
			zap.Int("commits", numCommits),
			zap.Int64("operations", totalOperations),
			zap.Duration("total_time", totalTime))
	})
}

// testTagOperationsPerformance 测试标签操作性能
func (suite *GitGatewayPerformanceTestSuite) testTagOperationsPerformance(repositoryID uuid.UUID) {
	suite.Run("标签操作性能", func() {
		const numTags = 20
		tags := make([]string, 0, numTags)

		startTime := time.Now()

		// 创建标签
		for i := 0; i < numTags; i++ {
			createReq := models.CreateTagRequest{
				Name:      fmt.Sprintf("v1.%d.0", i),
				CommitSHA: "initial_commit_sha",
				Message:   stringPtr(fmt.Sprintf("版本1.%d.0", i)),
				Tagger: models.CommitAuthor{
					Name:  "Performance Test User",
					Email: "perf@test.com",
				},
			}

			resp, body, _ := suite.makePerformanceRequest("POST", fmt.Sprintf("/api/v1/repositories/%s/tags", repositoryID), createReq, suite.authToken)
			assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

			var createResponse response.Response
			err := json.Unmarshal(body, &createResponse)
			assert.NoError(suite.T(), err)

			tagData, _ := json.Marshal(createResponse.Data)
			var tag models.Tag
			err = json.Unmarshal(tagData, &tag)
			if err == nil {
				tags = append(tags, tag.Name)
			}
		}

		// 列出标签
		resp, _, _ := suite.makePerformanceRequest("GET", fmt.Sprintf("/api/v1/repositories/%s/tags", repositoryID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		// 删除标签
		for _, tagName := range tags {
			resp, _, _ := suite.makePerformanceRequest("DELETE", fmt.Sprintf("/api/v1/repositories/%s/tags/%s", repositoryID, tagName), nil, suite.authToken)
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		}

		totalTime := time.Since(startTime)
		totalOperations := int64(numTags*2 + 1) // 创建+删除+列出

		suite.recordOperationMetrics("tag_operations", totalOperations, totalTime, 0)

		suite.logger.Info("标签操作性能测试完成",
			zap.Int("tags", numTags),
			zap.Int64("operations", totalOperations),
			zap.Duration("total_time", totalTime))
	})
}

// recordOperationMetrics 记录操作性能指标
func (suite *GitGatewayPerformanceTestSuite) recordOperationMetrics(operationType string, totalOps int64, totalTime time.Duration, errorCount int64) {
	suite.performanceMetrics.mu.Lock()
	defer suite.performanceMetrics.mu.Unlock()

	suite.performanceMetrics.OperationMetrics[operationType] = &OperationMetric{
		OperationType:    operationType,
		TotalOperations:  totalOps,
		TotalTime:        totalTime,
		AverageTime:      totalTime / time.Duration(totalOps),
		OperationsPerSec: float64(totalOps) / totalTime.Seconds(),
		ErrorCount:       errorCount,
	}
}

// generatePerformanceReport 生成性能测试报告
func (suite *GitGatewayPerformanceTestSuite) generatePerformanceReport() {
	suite.performanceMetrics.mu.Lock()
	defer suite.performanceMetrics.mu.Unlock()

	if suite.performanceMetrics.TotalRequests == 0 {
		suite.logger.Info("无性能数据，跳过报告生成")
		return
	}

	// 计算全局指标
	suite.performanceMetrics.AverageResponseTime = suite.performanceMetrics.TotalResponseTime / time.Duration(suite.performanceMetrics.TotalRequests)
	suite.performanceMetrics.SuccessRate = float64(suite.performanceMetrics.TotalRequests-suite.performanceMetrics.ErrorCount) / float64(suite.performanceMetrics.TotalRequests) * 100

	// 计算端点指标
	for _, metric := range suite.performanceMetrics.EndpointMetrics {
		if metric.RequestCount > 0 {
			metric.AverageTime = metric.TotalResponseTime / time.Duration(metric.RequestCount)
			metric.SuccessRate = float64(metric.RequestCount-metric.ErrorCount) / float64(metric.RequestCount) * 100
		}
	}

	// 生成JSON报告
	reportData, err := json.MarshalIndent(suite.performanceMetrics, "", "  ")
	if err != nil {
		suite.logger.Error("生成性能报告失败", zap.Error(err))
		return
	}

	// 写入报告文件
	reportFile := "test-report/git_gateway_performance_report.json"
	err = os.MkdirAll("test-report", 0755)
	if err != nil {
		suite.logger.Error("创建报告目录失败", zap.Error(err))
		return
	}

	err = os.WriteFile(reportFile, reportData, 0644)
	if err != nil {
		suite.logger.Error("写入性能报告失败", zap.Error(err))
	} else {
		suite.logger.Info("Git网关性能测试报告已生成", zap.String("file", reportFile))
	}

	// 输出汇总信息
	suite.logger.Info("Git网关性能测试汇总报告",
		zap.Int64("total_requests", suite.performanceMetrics.TotalRequests),
		zap.Int64("total_errors", suite.performanceMetrics.ErrorCount),
		zap.Float64("success_rate", suite.performanceMetrics.SuccessRate),
		zap.Duration("avg_response_time", suite.performanceMetrics.AverageResponseTime),
		zap.Duration("min_response_time", suite.performanceMetrics.MinResponseTime),
		zap.Duration("max_response_time", suite.performanceMetrics.MaxResponseTime),
		zap.Int("endpoints_tested", len(suite.performanceMetrics.EndpointMetrics)),
		zap.Int("concurrency_levels", len(suite.performanceMetrics.ConcurrencyMetrics)),
		zap.Int("operation_types", len(suite.performanceMetrics.OperationMetrics)))
}

// 辅助函数
func stringPtr(s string) *string {
	return &s
}

// TestGitGatewayPerformance 运行Git网关性能测试
func TestGitGatewayPerformance(t *testing.T) {
	suite.Run(t, new(GitGatewayPerformanceTestSuite))
}
package unit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
)

// GitGatewayClientTestSuite Git网关客户端测试套件
type GitGatewayClientTestSuite struct {
	suite.Suite
	client       client.GitGatewayClient
	mockServer   *httptest.Server
	logger       *zap.Logger
	testRepo     *client.Repository
	testBranch   *client.Branch
	testCommit   *client.Commit
	testTag      *client.Tag
	testRepoID   uuid.UUID
	testProjectID uuid.UUID
}

func (suite *GitGatewayClientTestSuite) SetupSuite() {
	suite.logger = zaptest.NewLogger(suite.T())
	suite.testRepoID = uuid.New()
	suite.testProjectID = uuid.New()
	
	// 创建测试数据
	suite.testRepo = &client.Repository{
		ID:            suite.testRepoID,
		ProjectID:     suite.testProjectID,
		Name:          "test-repository",
		Description:   &[]string{"Test repository description"}[0],
		Visibility:    client.RepositoryVisibilityPrivate,
		Status:        client.RepositoryStatusActive,
		DefaultBranch: "main",
		GitPath:       "/git/test-repository.git",
		CloneURL:      "https://git.example.com/test-repository.git",
		SSHURL:        "git@git.example.com:test-repository.git",
		Size:          1024000,
		CommitCount:   42,
		BranchCount:   5,
		TagCount:      3,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	suite.testBranch = &client.Branch{
		ID:           uuid.New(),
		RepositoryID: suite.testRepoID,
		Name:         "feature/test-branch",
		CommitSHA:    "abc123def456",
		IsDefault:    false,
		IsProtected:  false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	suite.testCommit = &client.Commit{
		ID:             uuid.New(),
		RepositoryID:   suite.testRepoID,
		SHA:            "abc123def456",
		Message:        "Test commit message",
		Author:         "John Doe",
		AuthorEmail:    "john@example.com",
		Committer:      "John Doe",
		CommitterEmail: "john@example.com",
		ParentSHAs:     []string{"parent123"},
		TreeSHA:        "tree456",
		AddedLines:     10,
		DeletedLines:   2,
		ChangedFiles:   3,
		CommittedAt:    time.Now(),
		CreatedAt:      time.Now(),
	}

	suite.testTag = &client.Tag{
		ID:           uuid.New(),
		RepositoryID: suite.testRepoID,
		Name:         "v1.0.0",
		CommitSHA:    "abc123def456",
		Message:      &[]string{"Release version 1.0.0"}[0],
		Tagger:       "John Doe",
		TaggerEmail:  "john@example.com",
		TaggedAt:     time.Now(),
		CreatedAt:    time.Now(),
	}
}

func (suite *GitGatewayClientTestSuite) SetupTest() {
	// 为每个测试创建新的mock服务器
	suite.mockServer = httptest.NewServer(http.HandlerFunc(suite.handleRequest))
	
	// 创建客户端
	config := &client.GitGatewayClientConfig{
		BaseURL: suite.mockServer.URL,
		Timeout: 10 * time.Second,
		APIKey:  "test-api-key",
		Logger:  suite.logger,
	}
	suite.client = client.NewGitGatewayClient(config)
}

func (suite *GitGatewayClientTestSuite) TearDownTest() {
	if suite.mockServer != nil {
		suite.mockServer.Close()
	}
}

// handleRequest 处理模拟服务器的HTTP请求
func (suite *GitGatewayClientTestSuite) handleRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// 检查认证头
	if r.Header.Get("Authorization") != "Bearer test-api-key" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	path := r.URL.Path
	method := r.Method
	
	switch {
	// 仓库管理路由
	case method == "POST" && path == "/api/v1/repositories":
		suite.handleCreateRepository(w, r)
	case method == "GET" && strings.HasPrefix(path, "/api/v1/repositories/") && !strings.Contains(path[len("/api/v1/repositories/"):], "/"):
		suite.handleGetRepository(w, r)
	case method == "PUT" && strings.HasPrefix(path, "/api/v1/repositories/") && !strings.Contains(path[len("/api/v1/repositories/"):], "/"):
		suite.handleUpdateRepository(w, r)
	case method == "DELETE" && strings.HasPrefix(path, "/api/v1/repositories/") && !strings.Contains(path[len("/api/v1/repositories/"):], "/"):
		suite.handleDeleteRepository(w, r)
	case method == "GET" && path == "/api/v1/repositories":
		suite.handleListRepositories(w, r)
		
	// 分支管理路由
	case method == "POST" && strings.Contains(path, "/branches") && !strings.Contains(path, "/branches/"):
		suite.handleCreateBranch(w, r)
	case method == "GET" && strings.Contains(path, "/branches/") && strings.Count(path, "/") == 5:
		suite.handleGetBranch(w, r)
	case method == "GET" && strings.Contains(path, "/branches") && !strings.Contains(path, "/branches/"):
		suite.handleListBranches(w, r)
	case method == "DELETE" && strings.Contains(path, "/branches/"):
		suite.handleDeleteBranch(w, r)
	case method == "PUT" && strings.Contains(path, "/default-branch"):
		suite.handleSetDefaultBranch(w, r)
	case method == "POST" && strings.Contains(path, "/merge"):
		suite.handleMergeBranch(w, r)
		
	// 提交管理路由
	case method == "POST" && strings.Contains(path, "/commits") && !strings.Contains(path, "/commits/"):
		suite.handleCreateCommit(w, r)
	case method == "GET" && strings.Contains(path, "/commits/") && !strings.Contains(path, "/diff"):
		suite.handleGetCommit(w, r)
	case method == "GET" && strings.Contains(path, "/commits") && !strings.Contains(path, "/commits/"):
		suite.handleListCommits(w, r)
	case method == "GET" && strings.Contains(path, "/commits/") && strings.Contains(path, "/diff"):
		suite.handleGetCommitDiff(w, r)
	case method == "GET" && strings.Contains(path, "/compare"):
		suite.handleCompareBranches(w, r)
		
	// 标签管理路由  
	case method == "POST" && strings.Contains(path, "/tags") && !strings.Contains(path, "/tags/"):
		suite.handleCreateTag(w, r)
	case method == "GET" && strings.Contains(path, "/tags/"):
		suite.handleGetTag(w, r)
	case method == "GET" && strings.Contains(path, "/tags") && !strings.Contains(path, "/tags/"):
		suite.handleListTags(w, r)
	case method == "DELETE" && strings.Contains(path, "/tags/"):
		suite.handleDeleteTag(w, r)
		
	// 文件操作路由
	case method == "GET" && strings.Contains(path, "/files"):
		suite.handleGetFileContent(w, r)
	case method == "GET" && strings.Contains(path, "/tree"):
		suite.handleGetDirectoryContent(w, r)
		
	// 统计和搜索路由
	case method == "GET" && strings.Contains(path, "/stats"):
		suite.handleGetRepositoryStats(w, r)
	case method == "GET" && strings.Contains(path, "/search/repositories"):
		suite.handleSearchRepositories(w, r)
		
	default:
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

// 仓库管理处理器

func (suite *GitGatewayClientTestSuite) handleCreateRepository(w http.ResponseWriter, r *http.Request) {
	var req client.CreateRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	
	repo := *suite.testRepo
	repo.Name = req.Name
	repo.Description = req.Description
	repo.Visibility = req.Visibility
	if req.DefaultBranch != nil {
		repo.DefaultBranch = *req.DefaultBranch
	}
	
	response := client.APIResponse{
		Success: true,
		Message: "Repository created successfully",
		Data:    repo,
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleGetRepository(w http.ResponseWriter, r *http.Request) {
	response := client.APIResponse{
		Success: true,
		Message: "Repository retrieved successfully",
		Data:    *suite.testRepo,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleUpdateRepository(w http.ResponseWriter, r *http.Request) {
	var req client.UpdateRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	
	repo := *suite.testRepo
	if req.Name != nil {
		repo.Name = *req.Name
	}
	if req.Description != nil {
		repo.Description = req.Description
	}
	if req.Visibility != nil {
		repo.Visibility = *req.Visibility
	}
	if req.DefaultBranch != nil {
		repo.DefaultBranch = *req.DefaultBranch
	}
	
	response := client.APIResponse{
		Success: true,
		Message: "Repository updated successfully",
		Data:    repo,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleDeleteRepository(w http.ResponseWriter, r *http.Request) {
	response := client.APIResponse{
		Success: true,
		Message: "Repository deleted successfully",
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleListRepositories(w http.ResponseWriter, r *http.Request) {
	repositories := []client.Repository{*suite.testRepo}
	
	listResponse := client.RepositoryListResponse{
		Repositories: repositories,
		Total:        1,
		Page:         1,
		PageSize:     20,
	}
	
	response := client.APIResponse{
		Success: true,
		Message: "Repositories listed successfully",
		Data:    listResponse,
	}
	json.NewEncoder(w).Encode(response)
}

// 分支管理处理器

func (suite *GitGatewayClientTestSuite) handleCreateBranch(w http.ResponseWriter, r *http.Request) {
	var req client.CreateBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	
	branch := *suite.testBranch
	branch.Name = req.Name
	branch.CommitSHA = req.FromSHA
	if req.Protected != nil {
		branch.IsProtected = *req.Protected
	}
	
	response := client.APIResponse{
		Success: true,
		Message: "Branch created successfully",
		Data:    branch,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleGetBranch(w http.ResponseWriter, r *http.Request) {
	response := client.APIResponse{
		Success: true,
		Message: "Branch retrieved successfully",
		Data:    *suite.testBranch,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleListBranches(w http.ResponseWriter, r *http.Request) {
	branches := []client.Branch{*suite.testBranch}
	
	result := struct {
		Branches []client.Branch `json:"branches"`
	}{
		Branches: branches,
	}
	
	response := client.APIResponse{
		Success: true,
		Message: "Branches listed successfully",
		Data:    result,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleDeleteBranch(w http.ResponseWriter, r *http.Request) {
	response := client.APIResponse{
		Success: true,
		Message: "Branch deleted successfully",
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleSetDefaultBranch(w http.ResponseWriter, r *http.Request) {
	response := client.APIResponse{
		Success: true,
		Message: "Default branch set successfully",
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleMergeBranch(w http.ResponseWriter, r *http.Request) {
	response := client.APIResponse{
		Success: true,
		Message: "Branch merged successfully",
	}
	json.NewEncoder(w).Encode(response)
}

// 提交管理处理器

func (suite *GitGatewayClientTestSuite) handleCreateCommit(w http.ResponseWriter, r *http.Request) {
	var req client.CreateCommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	
	commit := *suite.testCommit
	commit.Message = req.Message
	commit.Author = req.Author.Name
	commit.AuthorEmail = req.Author.Email
	
	response := client.APIResponse{
		Success: true,
		Message: "Commit created successfully",
		Data:    commit,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleGetCommit(w http.ResponseWriter, r *http.Request) {
	response := client.APIResponse{
		Success: true,
		Message: "Commit retrieved successfully",
		Data:    *suite.testCommit,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleListCommits(w http.ResponseWriter, r *http.Request) {
	commits := []client.Commit{*suite.testCommit}
	
	listResponse := client.CommitListResponse{
		Commits:  commits,
		Total:    1,
		Page:     1,
		PageSize: 20,
	}
	
	response := client.APIResponse{
		Success: true,
		Message: "Commits listed successfully",
		Data:    listResponse,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleGetCommitDiff(w http.ResponseWriter, r *http.Request) {
	diff := client.GitDiff{
		FromSHA:      "parent123",
		ToSHA:        "abc123def456",
		TotalAdded:   10,
		TotalDeleted: 2,
		Files: []client.DiffFile{
			{
				Path:         "test.go",
				Status:       "modified",
				AddedLines:   5,
				DeletedLines: 1,
				Patch:        "@@ -1,3 +1,7 @@\n+added line\n existing line\n-removed line",
			},
		},
	}
	
	response := client.APIResponse{
		Success: true,
		Message: "Commit diff retrieved successfully",
		Data:    diff,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleCompareBranches(w http.ResponseWriter, r *http.Request) {
	diff := client.GitDiff{
		FromSHA:      "branch1-sha",
		ToSHA:        "branch2-sha",
		TotalAdded:   15,
		TotalDeleted: 5,
		Files: []client.DiffFile{
			{
				Path:         "compare.go",
				Status:       "modified",
				AddedLines:   15,
				DeletedLines: 5,
				Patch:        "@@ -1,10 +1,20 @@\n+new feature\n existing code\n-old code",
			},
		},
	}
	
	response := client.APIResponse{
		Success: true,
		Message: "Branch comparison completed successfully",
		Data:    diff,
	}
	json.NewEncoder(w).Encode(response)
}

// 标签管理处理器

func (suite *GitGatewayClientTestSuite) handleCreateTag(w http.ResponseWriter, r *http.Request) {
	var req client.CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	
	tag := *suite.testTag
	tag.Name = req.Name
	tag.CommitSHA = req.CommitSHA
	tag.Message = req.Message
	tag.Tagger = req.Tagger.Name
	tag.TaggerEmail = req.Tagger.Email
	
	response := client.APIResponse{
		Success: true,
		Message: "Tag created successfully",
		Data:    tag,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleGetTag(w http.ResponseWriter, r *http.Request) {
	response := client.APIResponse{
		Success: true,
		Message: "Tag retrieved successfully",
		Data:    *suite.testTag,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleListTags(w http.ResponseWriter, r *http.Request) {
	tags := []client.Tag{*suite.testTag}
	
	result := struct {
		Tags []client.Tag `json:"tags"`
	}{
		Tags: tags,
	}
	
	response := client.APIResponse{
		Success: true,
		Message: "Tags listed successfully",
		Data:    result,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
	response := client.APIResponse{
		Success: true,
		Message: "Tag deleted successfully",
	}
	json.NewEncoder(w).Encode(response)
}

// 文件操作处理器

func (suite *GitGatewayClientTestSuite) handleGetFileContent(w http.ResponseWriter, r *http.Request) {
	result := struct {
		Content string `json:"content"`
	}{
		Content: "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}",
	}
	
	response := client.APIResponse{
		Success: true,
		Message: "File content retrieved successfully",
		Data:    result,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleGetDirectoryContent(w http.ResponseWriter, r *http.Request) {
	files := []client.FileInfo{
		{
			Name: "main.go",
			Path: "main.go",
			Type: "file",
			Size: 1024,
			Mode: "100644",
			SHA:  "file123",
		},
		{
			Name: "src",
			Path: "src",
			Type: "directory",
			Size: 0,
			Mode: "040000",
			SHA:  "dir456",
		},
	}
	
	result := struct {
		Files []client.FileInfo `json:"files"`
	}{
		Files: files,
	}
	
	response := client.APIResponse{
		Success: true,
		Message: "Directory content retrieved successfully",
		Data:    result,
	}
	json.NewEncoder(w).Encode(response)
}

// 统计和搜索处理器

func (suite *GitGatewayClientTestSuite) handleGetRepositoryStats(w http.ResponseWriter, r *http.Request) {
	stats := client.RepositoryStats{
		Size:         1024000,
		CommitCount:  42,
		BranchCount:  5,
		TagCount:     3,
		LastPushedAt: &suite.testRepo.CreatedAt,
	}
	
	response := client.APIResponse{
		Success: true,
		Message: "Repository stats retrieved successfully",
		Data:    stats,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *GitGatewayClientTestSuite) handleSearchRepositories(w http.ResponseWriter, r *http.Request) {
	repositories := []client.Repository{*suite.testRepo}
	
	listResponse := client.RepositoryListResponse{
		Repositories: repositories,
		Total:        1,
		Page:         1,
		PageSize:     20,
	}
	
	response := client.APIResponse{
		Success: true,
		Message: "Repository search completed successfully",
		Data:    listResponse,
	}
	json.NewEncoder(w).Encode(response)
}

// 测试用例

// TestRepositoryOperations 测试仓库操作
func (suite *GitGatewayClientTestSuite) TestRepositoryOperations() {
	ctx := context.Background()
	
	suite.Run("创建仓库", func() {
		req := &client.CreateRepositoryRequest{
			ProjectID:     suite.testProjectID.String(),
			Name:          "test-repo",
			Description:   &[]string{"Test repository"}[0],
			Visibility:    client.RepositoryVisibilityPrivate,
			DefaultBranch: &[]string{"main"}[0],
			InitReadme:    true,
		}
		
		result, err := suite.client.CreateRepository(ctx, req)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), "test-repo", result.Name)
		assert.Equal(suite.T(), client.RepositoryVisibilityPrivate, result.Visibility)
		assert.Equal(suite.T(), "main", result.DefaultBranch)
	})
	
	suite.Run("获取仓库详情", func() {
		result, err := suite.client.GetRepository(ctx, suite.testRepoID)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), suite.testRepoID, result.ID)
		assert.Equal(suite.T(), suite.testRepo.Name, result.Name)
	})
	
	suite.Run("更新仓库", func() {
		req := &client.UpdateRepositoryRequest{
			Name:        &[]string{"updated-repo"}[0],
			Description: &[]string{"Updated description"}[0],
		}
		
		result, err := suite.client.UpdateRepository(ctx, suite.testRepoID, req)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), "updated-repo", result.Name)
	})
	
	suite.Run("删除仓库", func() {
		err := suite.client.DeleteRepository(ctx, suite.testRepoID)
		assert.NoError(suite.T(), err)
	})
	
	suite.Run("获取仓库列表", func() {
		result, err := suite.client.ListRepositories(ctx, &suite.testProjectID, 1, 20)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Len(suite.T(), result.Repositories, 1)
		assert.Equal(suite.T(), int64(1), result.Total)
		assert.Equal(suite.T(), 1, result.Page)
		assert.Equal(suite.T(), 20, result.PageSize)
	})
}

// TestBranchOperations 测试分支操作
func (suite *GitGatewayClientTestSuite) TestBranchOperations() {
	ctx := context.Background()
	
	suite.Run("创建分支", func() {
		req := &client.CreateBranchRequest{
			Name:      "feature/new-feature",
			FromSHA:   "abc123",
			Protected: &[]bool{false}[0],
		}
		
		result, err := suite.client.CreateBranch(ctx, suite.testRepoID, req)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), "feature/new-feature", result.Name)
		assert.Equal(suite.T(), "abc123", result.CommitSHA)
		assert.False(suite.T(), result.IsProtected)
	})
	
	suite.Run("获取分支详情", func() {
		result, err := suite.client.GetBranch(ctx, suite.testRepoID, "main")
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), suite.testBranch.Name, result.Name)
	})
	
	suite.Run("获取分支列表", func() {
		result, err := suite.client.ListBranches(ctx, suite.testRepoID)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Len(suite.T(), result, 1)
		assert.Equal(suite.T(), suite.testBranch.Name, result[0].Name)
	})
	
	suite.Run("删除分支", func() {
		err := suite.client.DeleteBranch(ctx, suite.testRepoID, "feature/old-feature")
		assert.NoError(suite.T(), err)
	})
	
	suite.Run("设置默认分支", func() {
		err := suite.client.SetDefaultBranch(ctx, suite.testRepoID, "develop")
		assert.NoError(suite.T(), err)
	})
	
	suite.Run("合并分支", func() {
		err := suite.client.MergeBranch(ctx, suite.testRepoID, "main", "feature/new-feature")
		assert.NoError(suite.T(), err)
	})
}

// TestCommitOperations 测试提交操作
func (suite *GitGatewayClientTestSuite) TestCommitOperations() {
	ctx := context.Background()
	
	suite.Run("创建提交", func() {
		req := &client.CreateCommitRequest{
			Branch:  "main",
			Message: "Add new feature",
			Author: client.CommitAuthor{
				Name:  "John Doe",
				Email: "john@example.com",
			},
			Files: []client.CreateCommitFile{
				{
					Path:    "main.go",
					Content: "package main\n\nfunc main() { println(\"Hello\") }",
					Mode:    "100644",
				},
			},
		}
		
		result, err := suite.client.CreateCommit(ctx, suite.testRepoID, req)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), "Add new feature", result.Message)
		assert.Equal(suite.T(), "John Doe", result.Author)
		assert.Equal(suite.T(), "john@example.com", result.AuthorEmail)
	})
	
	suite.Run("获取提交详情", func() {
		result, err := suite.client.GetCommit(ctx, suite.testRepoID, "abc123def456")
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), "abc123def456", result.SHA)
		assert.Equal(suite.T(), suite.testCommit.Message, result.Message)
	})
	
	suite.Run("获取提交列表", func() {
		result, err := suite.client.ListCommits(ctx, suite.testRepoID, "main", 1, 20)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Len(suite.T(), result.Commits, 1)
		assert.Equal(suite.T(), int64(1), result.Total)
		assert.Equal(suite.T(), 1, result.Page)
		assert.Equal(suite.T(), 20, result.PageSize)
	})
	
	suite.Run("获取提交差异", func() {
		result, err := suite.client.GetCommitDiff(ctx, suite.testRepoID, "abc123def456")
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), "parent123", result.FromSHA)
		assert.Equal(suite.T(), "abc123def456", result.ToSHA)
		assert.Equal(suite.T(), int32(10), result.TotalAdded)
		assert.Equal(suite.T(), int32(2), result.TotalDeleted)
		assert.Len(suite.T(), result.Files, 1)
	})
	
	suite.Run("比较分支", func() {
		result, err := suite.client.CompareBranches(ctx, suite.testRepoID, "main", "develop")
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), "branch1-sha", result.FromSHA)
		assert.Equal(suite.T(), "branch2-sha", result.ToSHA)
		assert.Equal(suite.T(), int32(15), result.TotalAdded)
		assert.Equal(suite.T(), int32(5), result.TotalDeleted)
		assert.Len(suite.T(), result.Files, 1)
	})
}

// TestTagOperations 测试标签操作
func (suite *GitGatewayClientTestSuite) TestTagOperations() {
	ctx := context.Background()
	
	suite.Run("创建标签", func() {
		req := &client.CreateTagRequest{
			Name:      "v2.0.0",
			CommitSHA: "def456abc789",
			Message:   &[]string{"Release version 2.0.0"}[0],
			Tagger: client.CommitAuthor{
				Name:  "Jane Doe",
				Email: "jane@example.com",
			},
		}
		
		result, err := suite.client.CreateTag(ctx, suite.testRepoID, req)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), "v2.0.0", result.Name)
		assert.Equal(suite.T(), "def456abc789", result.CommitSHA)
		assert.Equal(suite.T(), "Jane Doe", result.Tagger)
		assert.Equal(suite.T(), "jane@example.com", result.TaggerEmail)
	})
	
	suite.Run("获取标签详情", func() {
		result, err := suite.client.GetTag(ctx, suite.testRepoID, "v1.0.0")
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), "v1.0.0", result.Name)
		assert.Equal(suite.T(), suite.testTag.CommitSHA, result.CommitSHA)
	})
	
	suite.Run("获取标签列表", func() {
		result, err := suite.client.ListTags(ctx, suite.testRepoID)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Len(suite.T(), result, 1)
		assert.Equal(suite.T(), suite.testTag.Name, result[0].Name)
	})
	
	suite.Run("删除标签", func() {
		err := suite.client.DeleteTag(ctx, suite.testRepoID, "v0.1.0")
		assert.NoError(suite.T(), err)
	})
}

// TestFileOperations 测试文件操作
func (suite *GitGatewayClientTestSuite) TestFileOperations() {
	ctx := context.Background()
	
	suite.Run("获取文件内容", func() {
		result, err := suite.client.GetFileContent(ctx, suite.testRepoID, "main", "main.go")
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		expectedContent := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}"
		assert.Equal(suite.T(), expectedContent, string(result))
	})
	
	suite.Run("获取目录内容", func() {
		result, err := suite.client.GetDirectoryContent(ctx, suite.testRepoID, "main", ".")
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Len(suite.T(), result, 2)
		
		// 检查文件
		fileFound := false
		dirFound := false
		for _, item := range result {
			if item.Name == "main.go" && item.Type == "file" {
				fileFound = true
				assert.Equal(suite.T(), int64(1024), item.Size)
				assert.Equal(suite.T(), "100644", item.Mode)
			}
			if item.Name == "src" && item.Type == "directory" {
				dirFound = true
				assert.Equal(suite.T(), "040000", item.Mode)
			}
		}
		assert.True(suite.T(), fileFound, "应该找到main.go文件")
		assert.True(suite.T(), dirFound, "应该找到src目录")
	})
}

// TestStatsAndSearch 测试统计和搜索
func (suite *GitGatewayClientTestSuite) TestStatsAndSearch() {
	ctx := context.Background()
	
	suite.Run("获取仓库统计", func() {
		result, err := suite.client.GetRepositoryStats(ctx, suite.testRepoID)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), int64(1024000), result.Size)
		assert.Equal(suite.T(), int64(42), result.CommitCount)
		assert.Equal(suite.T(), int64(5), result.BranchCount)
		assert.Equal(suite.T(), int64(3), result.TagCount)
		assert.NotNil(suite.T(), result.LastPushedAt)
	})
	
	suite.Run("搜索仓库", func() {
		result, err := suite.client.SearchRepositories(ctx, "test", &suite.testProjectID, 1, 20)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Len(suite.T(), result.Repositories, 1)
		assert.Equal(suite.T(), int64(1), result.Total)
		assert.Equal(suite.T(), 1, result.Page)
		assert.Equal(suite.T(), 20, result.PageSize)
		assert.Equal(suite.T(), suite.testRepo.Name, result.Repositories[0].Name)
	})
}

// TestErrorHandling 测试错误处理
func (suite *GitGatewayClientTestSuite) TestErrorHandling() {
	ctx := context.Background()
	
	suite.Run("未授权访问", func() {
		// 创建一个没有API密钥的客户端
		config := &client.GitGatewayClientConfig{
			BaseURL: suite.mockServer.URL,
			Timeout: 10 * time.Second,
			APIKey:  "", // 空的API密钥
			Logger:  suite.logger,
		}
		unauthorizedClient := client.NewGitGatewayClient(config)
		
		_, err := unauthorizedClient.GetRepository(ctx, suite.testRepoID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "401")
	})
	
	suite.Run("请求超时", func() {
		// 创建一个超时很短的客户端
		config := &client.GitGatewayClientConfig{
			BaseURL: suite.mockServer.URL,
			Timeout: 1 * time.Nanosecond, // 极短的超时时间
			APIKey:  "test-api-key",
			Logger:  suite.logger,
		}
		timeoutClient := client.NewGitGatewayClient(config)
		
		_, err := timeoutClient.GetRepository(ctx, suite.testRepoID)
		assert.Error(suite.T(), err)
	})
	
	suite.Run("不存在的接口", func() {
		// 模拟调用不存在的接口，通过修改mockServer的处理逻辑
		originalHandler := suite.mockServer.Config.Handler
		suite.mockServer.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Not Found", http.StatusNotFound)
		})
		
		_, err := suite.client.GetRepository(ctx, suite.testRepoID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "404")
		
		// 恢复原始处理器
		suite.mockServer.Config.Handler = originalHandler
	})
}

// TestClientUtilityMethods 测试客户端工具方法
func (suite *GitGatewayClientTestSuite) TestClientUtilityMethods() {
	suite.Run("仓库状态检查", func() {
		// 测试Repository模型的工具方法
		repo := suite.testRepo
		assert.True(suite.T(), repo.IsActive())
		assert.False(suite.T(), repo.IsPublic())
		assert.True(suite.T(), repo.IsPrivate())
	})
	
	suite.Run("分支状态检查", func() {
		// 测试Branch模型的工具方法
		branch := suite.testBranch
		assert.False(suite.T(), branch.IsDefaultBranch())
		assert.False(suite.T(), branch.IsProtectedBranch())
		
		// 测试默认分支
		defaultBranch := *branch
		defaultBranch.IsDefault = true
		defaultBranch.IsProtected = true
		assert.True(suite.T(), defaultBranch.IsDefaultBranch())
		assert.True(suite.T(), defaultBranch.IsProtectedBranch())
	})
	
	suite.Run("提交信息处理", func() {
		// 测试Commit模型的工具方法
		commit := suite.testCommit
		assert.Equal(suite.T(), "abc123d", commit.GetShortSHA())
		assert.Equal(suite.T(), int32(12), commit.GetTotalChanges()) // 10 + 2
	})
	
	suite.Run("标签类型检查", func() {
		// 测试Tag模型的工具方法
		tag := suite.testTag
		assert.True(suite.T(), tag.IsAnnotated()) // 有消息的标签是注释标签
		
		// 测试轻量标签
		lightTag := *tag
		lightTag.Message = nil
		assert.False(suite.T(), lightTag.IsAnnotated())
		
		emptyTag := *tag
		emptyTag.Message = &[]string{""}[0]
		assert.False(suite.T(), emptyTag.IsAnnotated())
	})
}

// TestConcurrentRequests 测试并发请求
func (suite *GitGatewayClientTestSuite) TestConcurrentRequests() {
	ctx := context.Background()
	
	suite.Run("并发获取仓库详情", func() {
		numGoroutines := 10
		results := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, err := suite.client.GetRepository(ctx, suite.testRepoID)
				results <- err
			}()
		}
		
		// 收集所有结果
		for i := 0; i < numGoroutines; i++ {
			select {
			case err := <-results:
				assert.NoError(suite.T(), err, "并发请求不应该失败")
			case <-time.After(5 * time.Second):
				suite.T().Fatal("并发测试超时")
			}
		}
	})
	
	suite.Run("并发创建分支", func() {
		numGoroutines := 5
		results := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				req := &client.CreateBranchRequest{
					Name:    fmt.Sprintf("feature/concurrent-test-%d", index),
					FromSHA: "abc123",
				}
				_, err := suite.client.CreateBranch(ctx, suite.testRepoID, req)
				results <- err
			}(i)
		}
		
		// 收集所有结果
		for i := 0; i < numGoroutines; i++ {
			select {
			case err := <-results:
				assert.NoError(suite.T(), err, "并发创建分支不应该失败")
			case <-time.After(5 * time.Second):
				suite.T().Fatal("并发测试超时")
			}
		}
	})
}

// 运行测试套件
func TestGitGatewayClientSuite(t *testing.T) {
	suite.Run(t, new(GitGatewayClientTestSuite))
}

// TestModelConstants 测试模型常量
func TestModelConstants(t *testing.T) {
	// 测试仓库状态常量
	assert.Equal(t, "active", string(client.RepositoryStatusActive))
	assert.Equal(t, "archived", string(client.RepositoryStatusArchived))
	assert.Equal(t, "deleted", string(client.RepositoryStatusDeleted))
	
	// 测试仓库可见性常量
	assert.Equal(t, "public", string(client.RepositoryVisibilityPublic))
	assert.Equal(t, "private", string(client.RepositoryVisibilityPrivate))
	assert.Equal(t, "internal", string(client.RepositoryVisibilityInternal))
}

// TestClientConfiguration 测试客户端配置
func TestClientConfiguration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	t.Run("默认配置", func(t *testing.T) {
		config := &client.GitGatewayClientConfig{
			BaseURL: "https://git.example.com",
			APIKey:  "test-key",
		}
		
		gitClient := client.NewGitGatewayClient(config)
		assert.NotNil(t, gitClient)
	})
	
	t.Run("完整配置", func(t *testing.T) {
		config := &client.GitGatewayClientConfig{
			BaseURL: "https://git.example.com",
			Timeout: 30 * time.Second,
			APIKey:  "test-key",
			Logger:  logger,
		}
		
		gitClient := client.NewGitGatewayClient(config)
		assert.NotNil(t, gitClient)
	})
	
	t.Run("无日志配置", func(t *testing.T) {
		config := &client.GitGatewayClientConfig{
			BaseURL: "https://git.example.com",
			APIKey:  "test-key",
			Logger:  nil, // 测试nil logger的处理
		}
		
		gitClient := client.NewGitGatewayClient(config)
		assert.NotNil(t, gitClient)
	})
}
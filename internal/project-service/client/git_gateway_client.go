package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GitGatewayClient Git网关客户端接口
type GitGatewayClient interface {
	// 仓库管理
	CreateRepository(ctx context.Context, req *CreateRepositoryRequest) (*Repository, error)
	GetRepository(ctx context.Context, repositoryID uuid.UUID) (*Repository, error)
	UpdateRepository(ctx context.Context, repositoryID uuid.UUID, req *UpdateRepositoryRequest) (*Repository, error)
	DeleteRepository(ctx context.Context, repositoryID uuid.UUID) error
	ListRepositories(ctx context.Context, projectID *uuid.UUID, page, pageSize int) (*RepositoryListResponse, error)

	// 分支管理
	CreateBranch(ctx context.Context, repositoryID uuid.UUID, req *CreateBranchRequest) (*Branch, error)
	GetBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) (*Branch, error)
	ListBranches(ctx context.Context, repositoryID uuid.UUID) ([]Branch, error)
	DeleteBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error
	SetDefaultBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error
	MergeBranch(ctx context.Context, repositoryID uuid.UUID, targetBranch, sourceBranch string) error

	// 提交管理
	CreateCommit(ctx context.Context, repositoryID uuid.UUID, req *CreateCommitRequest) (*Commit, error)
	GetCommit(ctx context.Context, repositoryID uuid.UUID, sha string) (*Commit, error)
	ListCommits(ctx context.Context, repositoryID uuid.UUID, branch string, page, pageSize int) (*CommitListResponse, error)
	GetCommitDiff(ctx context.Context, repositoryID uuid.UUID, sha string) (*GitDiff, error)
	CompareBranches(ctx context.Context, repositoryID uuid.UUID, base, head string) (*GitDiff, error)

	// 标签管理
	CreateTag(ctx context.Context, repositoryID uuid.UUID, req *CreateTagRequest) (*Tag, error)
	GetTag(ctx context.Context, repositoryID uuid.UUID, tagName string) (*Tag, error)
	ListTags(ctx context.Context, repositoryID uuid.UUID) ([]Tag, error)
	DeleteTag(ctx context.Context, repositoryID uuid.UUID, tagName string) error

	// 文件操作
	GetFileContent(ctx context.Context, repositoryID uuid.UUID, branch, filePath string) ([]byte, error)
	GetDirectoryContent(ctx context.Context, repositoryID uuid.UUID, branch, dirPath string) ([]FileInfo, error)

	// 统计和搜索
	GetRepositoryStats(ctx context.Context, repositoryID uuid.UUID) (*RepositoryStats, error)
	SearchRepositories(ctx context.Context, query string, projectID *uuid.UUID, page, pageSize int) (*RepositoryListResponse, error)
}

// gitGatewayClient Git网关客户端实现
type gitGatewayClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	logger     *zap.Logger
}

// GitGatewayClientConfig 客户端配置
type GitGatewayClientConfig struct {
	BaseURL string
	Timeout time.Duration
	APIKey  string
	Logger  *zap.Logger
}

// NewGitGatewayClient 创建Git网关客户端
func NewGitGatewayClient(config *GitGatewayClientConfig) GitGatewayClient {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &gitGatewayClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiKey: config.APIKey,
		logger: logger,
	}
}

// 响应结构
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   interface{} `json:"error,omitempty"`
}

// 通用HTTP请求方法
func (c *gitGatewayClient) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader

	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// 记录请求日志
	c.logger.Debug("Making Git Gateway request",
		zap.String("method", method),
		zap.String("url", url),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Git Gateway request failed",
			zap.String("method", method),
			zap.String("url", url),
			zap.Error(err),
		)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// 记录响应日志
	c.logger.Debug("Git Gateway response received",
		zap.String("method", method),
		zap.String("url", url),
		zap.Int("status", resp.StatusCode),
		zap.Int("body_size", len(respBody)),
	)

	// 检查HTTP状态码
	if resp.StatusCode >= 400 {
		var apiResp APIResponse
		if err := json.Unmarshal(respBody, &apiResp); err == nil {
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, apiResp.Message)
		}
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// 解析响应
	if result != nil {
		var apiResp APIResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("API request failed: %s", apiResp.Message)
		}

		// 将data字段解析到result
		dataBytes, err := json.Marshal(apiResp.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal API data: %w", err)
		}

		if err := json.Unmarshal(dataBytes, result); err != nil {
			return fmt.Errorf("failed to unmarshal API data: %w", err)
		}
	}

	return nil
}

// 仓库管理方法实现

// CreateRepository 创建仓库
func (c *gitGatewayClient) CreateRepository(ctx context.Context, req *CreateRepositoryRequest) (*Repository, error) {
	var result Repository
	err := c.doRequest(ctx, "POST", "/api/v1/repositories", req, &result)
	if err != nil {
		c.logger.Error("Failed to create repository",
			zap.String("name", req.Name),
			zap.Error(err),
		)
		return nil, err
	}

	c.logger.Info("Repository created successfully",
		zap.String("repository_id", result.ID.String()),
		zap.String("name", result.Name),
	)
	return &result, nil
}

// GetRepository 获取仓库详情
func (c *gitGatewayClient) GetRepository(ctx context.Context, repositoryID uuid.UUID) (*Repository, error) {
	var result Repository
	path := fmt.Sprintf("/api/v1/repositories/%s", repositoryID.String())
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to get repository",
			zap.String("repository_id", repositoryID.String()),
			zap.Error(err),
		)
		return nil, err
	}
	return &result, nil
}

// UpdateRepository 更新仓库
func (c *gitGatewayClient) UpdateRepository(ctx context.Context, repositoryID uuid.UUID, req *UpdateRepositoryRequest) (*Repository, error) {
	var result Repository
	path := fmt.Sprintf("/api/v1/repositories/%s", repositoryID.String())
	err := c.doRequest(ctx, "PUT", path, req, &result)
	if err != nil {
		c.logger.Error("Failed to update repository",
			zap.String("repository_id", repositoryID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	c.logger.Info("Repository updated successfully",
		zap.String("repository_id", repositoryID.String()),
	)
	return &result, nil
}

// DeleteRepository 删除仓库
func (c *gitGatewayClient) DeleteRepository(ctx context.Context, repositoryID uuid.UUID) error {
	path := fmt.Sprintf("/api/v1/repositories/%s", repositoryID.String())
	err := c.doRequest(ctx, "DELETE", path, nil, nil)
	if err != nil {
		c.logger.Error("Failed to delete repository",
			zap.String("repository_id", repositoryID.String()),
			zap.Error(err),
		)
		return err
	}

	c.logger.Info("Repository deleted successfully",
		zap.String("repository_id", repositoryID.String()),
	)
	return nil
}

// ListRepositories 获取仓库列表
func (c *gitGatewayClient) ListRepositories(ctx context.Context, projectID *uuid.UUID, page, pageSize int) (*RepositoryListResponse, error) {
	path := fmt.Sprintf("/api/v1/repositories?page=%d&page_size=%d", page, pageSize)
	if projectID != nil {
		path += "&project_id=" + projectID.String()
	}

	var result RepositoryListResponse
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to list repositories",
			zap.Int("page", page),
			zap.Int("page_size", pageSize),
			zap.Error(err),
		)
		return nil, err
	}
	return &result, nil
}

// 分支管理方法实现

// CreateBranch 创建分支
func (c *gitGatewayClient) CreateBranch(ctx context.Context, repositoryID uuid.UUID, req *CreateBranchRequest) (*Branch, error) {
	var result Branch
	path := fmt.Sprintf("/api/v1/repositories/%s/branches", repositoryID.String())
	err := c.doRequest(ctx, "POST", path, req, &result)
	if err != nil {
		c.logger.Error("Failed to create branch",
			zap.String("repository_id", repositoryID.String()),
			zap.String("branch_name", req.Name),
			zap.Error(err),
		)
		return nil, err
	}

	c.logger.Info("Branch created successfully",
		zap.String("repository_id", repositoryID.String()),
		zap.String("branch_name", result.Name),
	)
	return &result, nil
}

// GetBranch 获取分支详情
func (c *gitGatewayClient) GetBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) (*Branch, error) {
	var result Branch
	path := fmt.Sprintf("/api/v1/repositories/%s/branches/%s", repositoryID.String(), branchName)
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to get branch",
			zap.String("repository_id", repositoryID.String()),
			zap.String("branch_name", branchName),
			zap.Error(err),
		)
		return nil, err
	}
	return &result, nil
}

// ListBranches 获取分支列表
func (c *gitGatewayClient) ListBranches(ctx context.Context, repositoryID uuid.UUID) ([]Branch, error) {
	var result struct {
		Branches []Branch `json:"branches"`
	}
	path := fmt.Sprintf("/api/v1/repositories/%s/branches", repositoryID.String())
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to list branches",
			zap.String("repository_id", repositoryID.String()),
			zap.Error(err),
		)
		return nil, err
	}
	return result.Branches, nil
}

// DeleteBranch 删除分支
func (c *gitGatewayClient) DeleteBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	path := fmt.Sprintf("/api/v1/repositories/%s/branches/%s", repositoryID.String(), branchName)
	err := c.doRequest(ctx, "DELETE", path, nil, nil)
	if err != nil {
		c.logger.Error("Failed to delete branch",
			zap.String("repository_id", repositoryID.String()),
			zap.String("branch_name", branchName),
			zap.Error(err),
		)
		return err
	}

	c.logger.Info("Branch deleted successfully",
		zap.String("repository_id", repositoryID.String()),
		zap.String("branch_name", branchName),
	)
	return nil
}

// SetDefaultBranch 设置默认分支
func (c *gitGatewayClient) SetDefaultBranch(ctx context.Context, repositoryID uuid.UUID, branchName string) error {
	req := struct {
		BranchName string `json:"branch_name"`
	}{
		BranchName: branchName,
	}

	path := fmt.Sprintf("/api/v1/repositories/%s/default-branch", repositoryID.String())
	err := c.doRequest(ctx, "PUT", path, req, nil)
	if err != nil {
		c.logger.Error("Failed to set default branch",
			zap.String("repository_id", repositoryID.String()),
			zap.String("branch_name", branchName),
			zap.Error(err),
		)
		return err
	}

	c.logger.Info("Default branch set successfully",
		zap.String("repository_id", repositoryID.String()),
		zap.String("branch_name", branchName),
	)
	return nil
}

// MergeBranch 合并分支
func (c *gitGatewayClient) MergeBranch(ctx context.Context, repositoryID uuid.UUID, targetBranch, sourceBranch string) error {
	req := struct {
		TargetBranch string `json:"target_branch"`
		SourceBranch string `json:"source_branch"`
	}{
		TargetBranch: targetBranch,
		SourceBranch: sourceBranch,
	}

	path := fmt.Sprintf("/api/v1/repositories/%s/merge", repositoryID.String())
	err := c.doRequest(ctx, "POST", path, req, nil)
	if err != nil {
		c.logger.Error("Failed to merge branch",
			zap.String("repository_id", repositoryID.String()),
			zap.String("target_branch", targetBranch),
			zap.String("source_branch", sourceBranch),
			zap.Error(err),
		)
		return err
	}

	c.logger.Info("Branch merged successfully",
		zap.String("repository_id", repositoryID.String()),
		zap.String("target_branch", targetBranch),
		zap.String("source_branch", sourceBranch),
	)
	return nil
}

// 提交管理方法实现

// CreateCommit 创建提交
func (c *gitGatewayClient) CreateCommit(ctx context.Context, repositoryID uuid.UUID, req *CreateCommitRequest) (*Commit, error) {
	var result Commit
	path := fmt.Sprintf("/api/v1/repositories/%s/commits", repositoryID.String())
	err := c.doRequest(ctx, "POST", path, req, &result)
	if err != nil {
		c.logger.Error("Failed to create commit",
			zap.String("repository_id", repositoryID.String()),
			zap.String("message", req.Message),
			zap.Error(err),
		)
		return nil, err
	}

	c.logger.Info("Commit created successfully",
		zap.String("repository_id", repositoryID.String()),
		zap.String("commit_sha", result.SHA),
	)
	return &result, nil
}

// GetCommit 获取提交详情
func (c *gitGatewayClient) GetCommit(ctx context.Context, repositoryID uuid.UUID, sha string) (*Commit, error) {
	var result Commit
	path := fmt.Sprintf("/api/v1/repositories/%s/commits/%s", repositoryID.String(), sha)
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to get commit",
			zap.String("repository_id", repositoryID.String()),
			zap.String("sha", sha),
			zap.Error(err),
		)
		return nil, err
	}
	return &result, nil
}

// ListCommits 获取提交列表
func (c *gitGatewayClient) ListCommits(ctx context.Context, repositoryID uuid.UUID, branch string, page, pageSize int) (*CommitListResponse, error) {
	path := fmt.Sprintf("/api/v1/repositories/%s/commits?page=%d&page_size=%d", repositoryID.String(), page, pageSize)
	if branch != "" {
		path += "&branch=" + branch
	}

	var result CommitListResponse
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to list commits",
			zap.String("repository_id", repositoryID.String()),
			zap.String("branch", branch),
			zap.Error(err),
		)
		return nil, err
	}
	return &result, nil
}

// GetCommitDiff 获取提交差异
func (c *gitGatewayClient) GetCommitDiff(ctx context.Context, repositoryID uuid.UUID, sha string) (*GitDiff, error) {
	var result GitDiff
	path := fmt.Sprintf("/api/v1/repositories/%s/commits/%s/diff", repositoryID.String(), sha)
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to get commit diff",
			zap.String("repository_id", repositoryID.String()),
			zap.String("sha", sha),
			zap.Error(err),
		)
		return nil, err
	}
	return &result, nil
}

// CompareBranches 比较分支
func (c *gitGatewayClient) CompareBranches(ctx context.Context, repositoryID uuid.UUID, base, head string) (*GitDiff, error) {
	var result GitDiff
	path := fmt.Sprintf("/api/v1/repositories/%s/compare?base=%s&head=%s", repositoryID.String(), base, head)
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to compare branches",
			zap.String("repository_id", repositoryID.String()),
			zap.String("base", base),
			zap.String("head", head),
			zap.Error(err),
		)
		return nil, err
	}
	return &result, nil
}

// 标签管理方法实现

// CreateTag 创建标签
func (c *gitGatewayClient) CreateTag(ctx context.Context, repositoryID uuid.UUID, req *CreateTagRequest) (*Tag, error) {
	var result Tag
	path := fmt.Sprintf("/api/v1/repositories/%s/tags", repositoryID.String())
	err := c.doRequest(ctx, "POST", path, req, &result)
	if err != nil {
		c.logger.Error("Failed to create tag",
			zap.String("repository_id", repositoryID.String()),
			zap.String("tag_name", req.Name),
			zap.Error(err),
		)
		return nil, err
	}

	c.logger.Info("Tag created successfully",
		zap.String("repository_id", repositoryID.String()),
		zap.String("tag_name", result.Name),
	)
	return &result, nil
}

// GetTag 获取标签详情
func (c *gitGatewayClient) GetTag(ctx context.Context, repositoryID uuid.UUID, tagName string) (*Tag, error) {
	var result Tag
	path := fmt.Sprintf("/api/v1/repositories/%s/tags/%s", repositoryID.String(), tagName)
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to get tag",
			zap.String("repository_id", repositoryID.String()),
			zap.String("tag_name", tagName),
			zap.Error(err),
		)
		return nil, err
	}
	return &result, nil
}

// ListTags 获取标签列表
func (c *gitGatewayClient) ListTags(ctx context.Context, repositoryID uuid.UUID) ([]Tag, error) {
	var result struct {
		Tags []Tag `json:"tags"`
	}
	path := fmt.Sprintf("/api/v1/repositories/%s/tags", repositoryID.String())
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to list tags",
			zap.String("repository_id", repositoryID.String()),
			zap.Error(err),
		)
		return nil, err
	}
	return result.Tags, nil
}

// DeleteTag 删除标签
func (c *gitGatewayClient) DeleteTag(ctx context.Context, repositoryID uuid.UUID, tagName string) error {
	path := fmt.Sprintf("/api/v1/repositories/%s/tags/%s", repositoryID.String(), tagName)
	err := c.doRequest(ctx, "DELETE", path, nil, nil)
	if err != nil {
		c.logger.Error("Failed to delete tag",
			zap.String("repository_id", repositoryID.String()),
			zap.String("tag_name", tagName),
			zap.Error(err),
		)
		return err
	}

	c.logger.Info("Tag deleted successfully",
		zap.String("repository_id", repositoryID.String()),
		zap.String("tag_name", tagName),
	)
	return nil
}

// 文件操作方法实现

// GetFileContent 获取文件内容
func (c *gitGatewayClient) GetFileContent(ctx context.Context, repositoryID uuid.UUID, branch, filePath string) ([]byte, error) {
	var result struct {
		Content string `json:"content"`
	}
	path := fmt.Sprintf("/api/v1/repositories/%s/files?branch=%s&path=%s", repositoryID.String(), branch, filePath)
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to get file content",
			zap.String("repository_id", repositoryID.String()),
			zap.String("branch", branch),
			zap.String("file_path", filePath),
			zap.Error(err),
		)
		return nil, err
	}
	return []byte(result.Content), nil
}

// GetDirectoryContent 获取目录内容
func (c *gitGatewayClient) GetDirectoryContent(ctx context.Context, repositoryID uuid.UUID, branch, dirPath string) ([]FileInfo, error) {
	var result struct {
		Files []FileInfo `json:"files"`
	}
	path := fmt.Sprintf("/api/v1/repositories/%s/tree?branch=%s&path=%s", repositoryID.String(), branch, dirPath)
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to get directory content",
			zap.String("repository_id", repositoryID.String()),
			zap.String("branch", branch),
			zap.String("dir_path", dirPath),
			zap.Error(err),
		)
		return nil, err
	}
	return result.Files, nil
}

// 统计和搜索方法实现

// GetRepositoryStats 获取仓库统计信息
func (c *gitGatewayClient) GetRepositoryStats(ctx context.Context, repositoryID uuid.UUID) (*RepositoryStats, error) {
	var result RepositoryStats
	path := fmt.Sprintf("/api/v1/repositories/%s/stats", repositoryID.String())
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to get repository stats",
			zap.String("repository_id", repositoryID.String()),
			zap.Error(err),
		)
		return nil, err
	}
	return &result, nil
}

// SearchRepositories 搜索仓库
func (c *gitGatewayClient) SearchRepositories(ctx context.Context, query string, projectID *uuid.UUID, page, pageSize int) (*RepositoryListResponse, error) {
	path := fmt.Sprintf("/api/v1/search/repositories?q=%s&page=%d&page_size=%d", query, page, pageSize)
	if projectID != nil {
		path += "&project_id=" + projectID.String()
	}

	var result RepositoryListResponse
	err := c.doRequest(ctx, "GET", path, nil, &result)
	if err != nil {
		c.logger.Error("Failed to search repositories",
			zap.String("query", query),
			zap.Int("page", page),
			zap.Int("page_size", pageSize),
			zap.Error(err),
		)
		return nil, err
	}
	return &result, nil
}

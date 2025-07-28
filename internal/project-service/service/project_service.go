package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/compensation"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/transaction"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ProjectService 项目服务接口
type ProjectService interface {
	// 项目管理
	CreateProject(ctx context.Context, req *models.CreateProjectRequest, userID, tenantID uuid.UUID) (*models.Project, error)
	GetProject(ctx context.Context, id uuid.UUID, userID, tenantID uuid.UUID) (*models.Project, error)
	GetProjectByKey(ctx context.Context, key string, userID, tenantID uuid.UUID) (*models.Project, error)
	UpdateProject(ctx context.Context, id uuid.UUID, req *models.UpdateProjectRequest, userID, tenantID uuid.UUID) (*models.Project, error)
	DeleteProject(ctx context.Context, id uuid.UUID, userID, tenantID uuid.UUID) error
	ListProjects(ctx context.Context, page, pageSize int, filters map[string]interface{}, userID, tenantID uuid.UUID) (*models.ProjectListResponse, error)

	// 成员管理
	AddMember(ctx context.Context, projectID uuid.UUID, req *models.AddMemberRequest, userID, tenantID uuid.UUID) error
	RemoveMember(ctx context.Context, projectID, memberUserID, userID, tenantID uuid.UUID) error
	GetMembers(ctx context.Context, projectID, userID, tenantID uuid.UUID) ([]models.ProjectMember, error)

	// Git仓库管理
	CreateRepository(ctx context.Context, projectID uuid.UUID, req *CreateRepositoryRequest, userID, tenantID uuid.UUID) (*models.Repository, error)
	GetRepository(ctx context.Context, repositoryID uuid.UUID, userID, tenantID uuid.UUID) (*models.Repository, error)
	ListRepositories(ctx context.Context, projectID uuid.UUID, page, pageSize int, userID, tenantID uuid.UUID) (*RepositoryListResponse, error)
	UpdateRepository(ctx context.Context, repositoryID uuid.UUID, req *UpdateRepositoryRequest, userID, tenantID uuid.UUID) (*models.Repository, error)
	DeleteRepository(ctx context.Context, repositoryID uuid.UUID, userID, tenantID uuid.UUID) error

	// Git高级操作
	CreateBranch(ctx context.Context, repositoryID uuid.UUID, req *CreateBranchRequest, userID, tenantID uuid.UUID) (*models.Branch, error)
	ListBranches(ctx context.Context, repositoryID uuid.UUID, page, pageSize int, userID, tenantID uuid.UUID) (*BranchListResponse, error)
	DeleteBranch(ctx context.Context, repositoryID uuid.UUID, branchName string, userID, tenantID uuid.UUID) error

	CreatePullRequest(ctx context.Context, repositoryID uuid.UUID, req *CreatePullRequestRequest, userID, tenantID uuid.UUID) (*models.PullRequest, error)
	GetPullRequest(ctx context.Context, repositoryID, pullRequestID uuid.UUID, userID, tenantID uuid.UUID) (*models.PullRequest, error)

	// 权限检查
	CheckProjectAccess(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
	GetUserProjects(ctx context.Context, userID, tenantID uuid.UUID) ([]models.Project, error)
}

// projectService 项目服务实现
type projectService struct {
	repo            repository.ProjectRepository
	gitClient       client.GitGatewayClient
	compensationMgr *compensation.CompensationManager
	transactionMgr  *transaction.DistributedTransactionManager
	logger          *zap.Logger
}

// NewProjectService 创建项目服务实例
func NewProjectService(repo repository.ProjectRepository, gitClient client.GitGatewayClient, logger *zap.Logger) ProjectService {
	// 创建补偿管理器
	compensationMgr := compensation.NewCompensationManager(gitClient, logger)

	// 创建分布式事务管理器
	transactionMgr := transaction.NewDistributedTransactionManager(repo, gitClient, compensationMgr, logger)

	return &projectService{
		repo:            repo,
		gitClient:       gitClient,
		compensationMgr: compensationMgr,
		transactionMgr:  transactionMgr,
		logger:          logger,
	}
}

// NewProjectServiceWithTransaction 创建带有指定分布式事务管理器的项目服务实例
func NewProjectServiceWithTransaction(
	repo repository.ProjectRepository,
	gitClient client.GitGatewayClient,
	transactionMgr *transaction.DistributedTransactionManager,
	logger *zap.Logger,
) ProjectService {
	// 使用现有的事务管理器获取补偿管理器
	compensationMgr := compensation.NewCompensationManager(gitClient, logger)

	return &projectService{
		repo:            repo,
		gitClient:       gitClient,
		compensationMgr: compensationMgr,
		transactionMgr:  transactionMgr,
		logger:          logger,
	}
}

// validateProjectKey 验证项目key格式
func (s *projectService) validateProjectKey(key string) error {
	if len(key) < 2 || len(key) > 20 {
		return errors.New("项目key长度必须在2-20个字符之间")
	}

	// 只允许字母、数字和连字符，且必须以字母开头
	matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9\-]*$`, key)
	if !matched {
		return errors.New("项目key只能包含字母、数字和连字符，且必须以字母开头")
	}

	return nil
}

// CreateProject 创建项目
func (s *projectService) CreateProject(ctx context.Context, req *models.CreateProjectRequest, userID, tenantID uuid.UUID) (*models.Project, error) {
	// 验证项目key格式
	if err := s.validateProjectKey(req.Key); err != nil {
		return nil, err
	}

	// 规范化项目key（转为小写）
	req.Key = strings.ToLower(req.Key)

	// 创建项目实例
	project := &models.Project{
		TenantID:    tenantID,
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
	}

	// 如果指定了管理员，验证并设置
	if req.ManagerID != nil && *req.ManagerID != "" {
		managerUUID, err := uuid.Parse(*req.ManagerID)
		if err != nil {
			return nil, errors.New("无效的管理员ID格式")
		}
		project.ManagerID = &managerUUID
	} else {
		// 默认创建者为管理员
		project.ManagerID = &userID
	}

	// 创建项目
	if err := s.repo.Create(ctx, project); err != nil {
		s.logger.Error("创建项目失败",
			zap.Error(err),
			zap.String("project_key", req.Key),
			zap.String("tenant_id", tenantID.String()),
			zap.String("user_id", userID.String()))
		return nil, err
	}

	// 添加创建者为项目成员（如果不是管理员的话）
	if project.ManagerID == nil || *project.ManagerID != userID {
		// TODO: 这里需要获取默认的项目成员角色ID
		// 暂时跳过，等角色系统完善后再实现
	}

	s.logger.Info("项目创建成功",
		zap.String("project_id", project.ID.String()),
		zap.String("project_key", project.Key),
		zap.String("created_by", userID.String()))

	return project, nil
}

// GetProject 获取项目详情
func (s *projectService) GetProject(ctx context.Context, id uuid.UUID, userID, tenantID uuid.UUID) (*models.Project, error) {
	// 获取项目
	project, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// 检查访问权限
	hasAccess, err := s.repo.CheckUserAccess(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("检查访问权限失败: %w", err)
	}
	if !hasAccess {
		return nil, errors.New("无权限访问此项目")
	}

	return project, nil
}

// GetProjectByKey 根据key获取项目
func (s *projectService) GetProjectByKey(ctx context.Context, key string, userID, tenantID uuid.UUID) (*models.Project, error) {
	// 获取项目
	project, err := s.repo.GetByKey(ctx, strings.ToLower(key), tenantID)
	if err != nil {
		return nil, err
	}

	// 检查访问权限
	hasAccess, err := s.repo.CheckUserAccess(ctx, project.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("检查访问权限失败: %w", err)
	}
	if !hasAccess {
		return nil, errors.New("无权限访问此项目")
	}

	return project, nil
}

// UpdateProject 更新项目
func (s *projectService) UpdateProject(ctx context.Context, id uuid.UUID, req *models.UpdateProjectRequest, userID, tenantID uuid.UUID) (*models.Project, error) {
	// 获取现有项目
	project, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// 检查权限：只有项目管理员或平台管理员可以更新项目
	if project.ManagerID == nil || *project.ManagerID != userID {
		// TODO: 检查是否为平台管理员
		return nil, errors.New("无权限更新此项目")
	}

	// 更新字段
	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Description != nil {
		project.Description = req.Description
	}
	if req.ManagerID != nil {
		if *req.ManagerID == "" {
			project.ManagerID = nil
		} else {
			managerUUID, err := uuid.Parse(*req.ManagerID)
			if err != nil {
				return nil, errors.New("无效的管理员ID格式")
			}
			project.ManagerID = &managerUUID
		}
	}
	if req.Status != nil {
		project.Status = *req.Status
	}

	// 保存更新
	if err := s.repo.Update(ctx, project); err != nil {
		s.logger.Error("更新项目失败",
			zap.Error(err),
			zap.String("project_id", id.String()),
			zap.String("user_id", userID.String()))
		return nil, err
	}

	s.logger.Info("项目更新成功",
		zap.String("project_id", project.ID.String()),
		zap.String("updated_by", userID.String()))

	return project, nil
}

// DeleteProject 删除项目
func (s *projectService) DeleteProject(ctx context.Context, id uuid.UUID, userID, tenantID uuid.UUID) error {
	// 获取项目
	project, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	// 检查权限：只有项目管理员或平台管理员可以删除项目
	if project.ManagerID == nil || *project.ManagerID != userID {
		// TODO: 检查是否为平台管理员
		return errors.New("无权限删除此项目")
	}

	// 删除项目
	if err := s.repo.Delete(ctx, id, tenantID); err != nil {
		s.logger.Error("删除项目失败",
			zap.Error(err),
			zap.String("project_id", id.String()),
			zap.String("user_id", userID.String()))
		return err
	}

	s.logger.Info("项目删除成功",
		zap.String("project_id", id.String()),
		zap.String("deleted_by", userID.String()))

	return nil
}

// ListProjects 获取项目列表
func (s *projectService) ListProjects(ctx context.Context, page, pageSize int, filters map[string]interface{}, userID, tenantID uuid.UUID) (*models.ProjectListResponse, error) {
	// 验证分页参数
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取用户能访问的项目列表
	projects, _, err := s.repo.List(ctx, tenantID, page, pageSize, filters)
	if err != nil {
		return nil, err
	}

	// 过滤用户有权限访问的项目
	accessibleProjects := make([]models.Project, 0)
	for _, project := range projects {
		hasAccess, err := s.repo.CheckUserAccess(ctx, project.ID, userID)
		if err != nil {
			s.logger.Warn("检查项目访问权限失败",
				zap.Error(err),
				zap.String("project_id", project.ID.String()),
				zap.String("user_id", userID.String()))
			continue
		}
		if hasAccess {
			accessibleProjects = append(accessibleProjects, project)
		}
	}

	return &models.ProjectListResponse{
		Projects: accessibleProjects,
		Total:    int64(len(accessibleProjects)), // 注意：这里应该是过滤后的总数
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// AddMember 添加项目成员
func (s *projectService) AddMember(ctx context.Context, projectID uuid.UUID, req *models.AddMemberRequest, userID, tenantID uuid.UUID) error {
	// 获取项目
	project, err := s.repo.GetByID(ctx, projectID, tenantID)
	if err != nil {
		return err
	}

	// 检查权限：只有项目管理员可以添加成员
	if project.ManagerID == nil || *project.ManagerID != userID {
		return errors.New("无权限添加项目成员")
	}

	// 解析用户ID
	memberUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		return errors.New("无效的用户ID格式")
	}

	// 解析角色ID
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		return errors.New("无效的角色ID格式")
	}

	// 创建成员关系
	member := &models.ProjectMember{
		ProjectID: projectID,
		UserID:    memberUserID,
		RoleID:    roleID,
		AddedBy:   &userID,
	}

	// 添加成员
	if err := s.repo.AddMember(ctx, member); err != nil {
		s.logger.Error("添加项目成员失败",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("user_id", req.UserID),
			zap.String("added_by", userID.String()))
		return err
	}

	s.logger.Info("项目成员添加成功",
		zap.String("project_id", projectID.String()),
		zap.String("user_id", req.UserID),
		zap.String("added_by", userID.String()))

	return nil
}

// RemoveMember 移除项目成员
func (s *projectService) RemoveMember(ctx context.Context, projectID, memberUserID, userID, tenantID uuid.UUID) error {
	// 获取项目
	project, err := s.repo.GetByID(ctx, projectID, tenantID)
	if err != nil {
		return err
	}

	// 检查权限：只有项目管理员可以移除成员
	if project.ManagerID == nil || *project.ManagerID != userID {
		return errors.New("无权限移除项目成员")
	}

	// 不能移除自己
	if memberUserID == userID {
		return errors.New("不能移除自己")
	}

	// 移除成员
	if err := s.repo.RemoveMember(ctx, projectID, memberUserID); err != nil {
		s.logger.Error("移除项目成员失败",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("member_user_id", memberUserID.String()),
			zap.String("removed_by", userID.String()))
		return err
	}

	s.logger.Info("项目成员移除成功",
		zap.String("project_id", projectID.String()),
		zap.String("member_user_id", memberUserID.String()),
		zap.String("removed_by", userID.String()))

	return nil
}

// GetMembers 获取项目成员列表
func (s *projectService) GetMembers(ctx context.Context, projectID, userID, tenantID uuid.UUID) ([]models.ProjectMember, error) {
	// 检查用户是否有访问权限
	hasAccess, err := s.repo.CheckUserAccess(ctx, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("检查访问权限失败: %w", err)
	}
	if !hasAccess {
		return nil, errors.New("无权限访问此项目")
	}

	// 获取成员列表
	members, err := s.repo.GetMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return members, nil
}

// CheckProjectAccess 检查项目访问权限
func (s *projectService) CheckProjectAccess(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	return s.repo.CheckUserAccess(ctx, projectID, userID)
}

// GetUserProjects 获取用户参与的项目列表
func (s *projectService) GetUserProjects(ctx context.Context, userID, tenantID uuid.UUID) ([]models.Project, error) {
	return s.repo.GetUserProjects(ctx, userID, tenantID)
}

// Git仓库管理方法

// CreateRepositoryRequest 创建仓库请求（项目服务层）
type CreateRepositoryRequest struct {
	Name          string  `json:"name" binding:"required,min=1,max=255"`
	Description   *string `json:"description,omitempty"`
	Visibility    string  `json:"visibility" binding:"required,oneof=public private internal"`
	DefaultBranch *string `json:"default_branch,omitempty"`
	InitReadme    bool    `json:"init_readme"`
}

// UpdateRepositoryRequest 更新仓库请求（项目服务层）
type UpdateRepositoryRequest struct {
	Name          *string `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
	Visibility    *string `json:"visibility,omitempty"`
	DefaultBranch *string `json:"default_branch,omitempty"`
}

// RepositoryListResponse 仓库列表响应（项目服务层）
type RepositoryListResponse struct {
	Repositories []models.Repository `json:"repositories"`
	Total        int64               `json:"total"`
	Page         int                 `json:"page"`
	PageSize     int                 `json:"page_size"`
}

// CreateRepository 创建Git仓库（使用分布式事务）
func (s *projectService) CreateRepository(ctx context.Context, projectID uuid.UUID, req *CreateRepositoryRequest, userID, tenantID uuid.UUID) (*models.Repository, error) {
	s.logger.Info("开始创建仓库（使用分布式事务）",
		zap.String("project_id", projectID.String()),
		zap.String("repository_name", req.Name),
		zap.String("user_id", userID.String()))

	// 转换可见性枚举
	var visibility client.RepositoryVisibility
	switch req.Visibility {
	case "public":
		visibility = client.RepositoryVisibilityPublic
	case "private":
		visibility = client.RepositoryVisibilityPrivate
	case "internal":
		visibility = client.RepositoryVisibilityInternal
	default:
		return nil, errors.New("无效的仓库可见性设置")
	}

	// 构建Git网关请求
	gitReq := &client.CreateRepositoryRequest{
		ProjectID:     projectID.String(),
		Name:          req.Name,
		Description:   req.Description,
		Visibility:    visibility,
		DefaultBranch: req.DefaultBranch,
		InitReadme:    req.InitReadme,
	}

	// 使用分布式事务管理器创建仓库
	repository, err := s.transactionMgr.CreateRepositoryTransaction(
		ctx,
		projectID,
		userID,
		tenantID,
		gitReq,
	)

	if err != nil {
		s.logger.Error("分布式事务创建仓库失败",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("repository_name", req.Name),
			zap.String("user_id", userID.String()))
		return nil, fmt.Errorf("创建仓库失败: %w", err)
	}

	s.logger.Info("仓库创建成功（分布式事务）",
		zap.String("project_id", projectID.String()),
		zap.String("repository_id", repository.ID.String()),
		zap.String("repository_name", repository.Name),
		zap.String("created_by", userID.String()))

	return repository, nil
}

// GetRepository 获取仓库详情
func (s *projectService) GetRepository(ctx context.Context, repositoryID uuid.UUID, userID, tenantID uuid.UUID) (*models.Repository, error) {
	// 从Git网关获取仓库信息
	gitRepo, err := s.gitClient.GetRepository(ctx, repositoryID)
	if err != nil {
		s.logger.Error("Git网关获取仓库失败",
			zap.Error(err),
			zap.String("repository_id", repositoryID.String()),
			zap.String("user_id", userID.String()))
		return nil, fmt.Errorf("获取仓库信息失败: %w", err)
	}

	// 检查项目访问权限
	hasAccess, err := s.repo.CheckUserAccess(ctx, gitRepo.ProjectID, userID)
	if err != nil {
		return nil, fmt.Errorf("检查项目访问权限失败: %w", err)
	}
	if !hasAccess {
		return nil, errors.New("无权限访问此仓库")
	}

	// 转换为项目服务的仓库模型
	repository := &models.Repository{
		ID:            gitRepo.ID,
		ProjectID:     gitRepo.ProjectID,
		Name:          gitRepo.Name,
		Description:   gitRepo.Description,
		Visibility:    string(gitRepo.Visibility),
		DefaultBranch: gitRepo.DefaultBranch,
	}

	return repository, nil
}

// ListRepositories 获取项目仓库列表
func (s *projectService) ListRepositories(ctx context.Context, projectID uuid.UUID, page, pageSize int, userID, tenantID uuid.UUID) (*RepositoryListResponse, error) {
	// 检查项目访问权限
	hasAccess, err := s.repo.CheckUserAccess(ctx, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("检查项目访问权限失败: %w", err)
	}
	if !hasAccess {
		return nil, errors.New("无权限访问此项目的仓库")
	}

	// 验证分页参数
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 从Git网关获取仓库列表
	gitRepoList, err := s.gitClient.ListRepositories(ctx, &projectID, page, pageSize)
	if err != nil {
		s.logger.Error("Git网关获取仓库列表失败",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("user_id", userID.String()))
		return nil, fmt.Errorf("获取仓库列表失败: %w", err)
	}

	// 转换为项目服务的仓库模型
	repositories := make([]models.Repository, len(gitRepoList.Repositories))
	for i, gitRepo := range gitRepoList.Repositories {
		repositories[i] = models.Repository{
			ID:            gitRepo.ID,
			ProjectID:     gitRepo.ProjectID,
			Name:          gitRepo.Name,
			Description:   gitRepo.Description,
			Visibility:    string(gitRepo.Visibility),
			DefaultBranch: gitRepo.DefaultBranch,
		}
	}

	return &RepositoryListResponse{
		Repositories: repositories,
		Total:        gitRepoList.Total,
		Page:         gitRepoList.Page,
		PageSize:     gitRepoList.PageSize,
	}, nil
}

// UpdateRepository 更新仓库
func (s *projectService) UpdateRepository(ctx context.Context, repositoryID uuid.UUID, req *UpdateRepositoryRequest, userID, tenantID uuid.UUID) (*models.Repository, error) {
	// 先获取仓库信息以检查项目访问权限
	gitRepo, err := s.gitClient.GetRepository(ctx, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("获取仓库信息失败: %w", err)
	}

	// 检查项目访问权限
	hasAccess, err := s.repo.CheckUserAccess(ctx, gitRepo.ProjectID, userID)
	if err != nil {
		return nil, fmt.Errorf("检查项目访问权限失败: %w", err)
	}
	if !hasAccess {
		return nil, errors.New("无权限更新此仓库")
	}

	// 构建Git网关更新请求
	gitReq := &client.UpdateRepositoryRequest{
		Name:          req.Name,
		Description:   req.Description,
		DefaultBranch: req.DefaultBranch,
	}

	// 转换可见性枚举
	if req.Visibility != nil {
		var visibility client.RepositoryVisibility
		switch *req.Visibility {
		case "public":
			visibility = client.RepositoryVisibilityPublic
			gitReq.Visibility = &visibility
		case "private":
			visibility = client.RepositoryVisibilityPrivate
			gitReq.Visibility = &visibility
		case "internal":
			visibility = client.RepositoryVisibilityInternal
			gitReq.Visibility = &visibility
		default:
			return nil, errors.New("无效的仓库可见性设置")
		}
	}

	// 调用Git网关更新仓库
	updatedGitRepo, err := s.gitClient.UpdateRepository(ctx, repositoryID, gitReq)
	if err != nil {
		s.logger.Error("Git网关更新仓库失败",
			zap.Error(err),
			zap.String("repository_id", repositoryID.String()),
			zap.String("user_id", userID.String()))
		return nil, fmt.Errorf("更新仓库失败: %w", err)
	}

	// 转换为项目服务的仓库模型
	repository := &models.Repository{
		ID:            updatedGitRepo.ID,
		ProjectID:     updatedGitRepo.ProjectID,
		Name:          updatedGitRepo.Name,
		Description:   updatedGitRepo.Description,
		Visibility:    string(updatedGitRepo.Visibility),
		DefaultBranch: updatedGitRepo.DefaultBranch,
	}

	s.logger.Info("仓库更新成功",
		zap.String("repository_id", repositoryID.String()),
		zap.String("updated_by", userID.String()))

	return repository, nil
}

// DeleteRepository 删除仓库
func (s *projectService) DeleteRepository(ctx context.Context, repositoryID uuid.UUID, userID, tenantID uuid.UUID) error {
	// 先获取仓库信息以检查项目访问权限
	gitRepo, err := s.gitClient.GetRepository(ctx, repositoryID)
	if err != nil {
		return fmt.Errorf("获取仓库信息失败: %w", err)
	}

	// 检查项目访问权限
	hasAccess, err := s.repo.CheckUserAccess(ctx, gitRepo.ProjectID, userID)
	if err != nil {
		return fmt.Errorf("检查项目访问权限失败: %w", err)
	}
	if !hasAccess {
		return errors.New("无权限删除此仓库")
	}

	// 调用Git网关删除仓库
	err = s.gitClient.DeleteRepository(ctx, repositoryID)
	if err != nil {
		s.logger.Error("Git网关删除仓库失败",
			zap.Error(err),
			zap.String("repository_id", repositoryID.String()),
			zap.String("user_id", userID.String()))
		return fmt.Errorf("删除仓库失败: %w", err)
	}

	s.logger.Info("仓库删除成功",
		zap.String("repository_id", repositoryID.String()),
		zap.String("deleted_by", userID.String()))

	return nil
}

// Git高级操作方法

// CreateBranchRequest 创建分支请求
type CreateBranchRequest struct {
	Name       string `json:"name" binding:"required,min=1,max=100"`
	SourceRef  string `json:"source_ref" binding:"required"`
	SourceType string `json:"source_type" binding:"required,oneof=branch commit tag"`
}

// BranchListResponse 分支列表响应
type BranchListResponse struct {
	Branches []models.Branch `json:"branches"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// CreatePullRequestRequest 创建合并请求
type CreatePullRequestRequest struct {
	Title        string      `json:"title" binding:"required,min=1,max=255"`
	Description  *string     `json:"description,omitempty"`
	SourceBranch string      `json:"source_branch" binding:"required"`
	TargetBranch string      `json:"target_branch" binding:"required"`
	AssigneeIDs  []uuid.UUID `json:"assignee_ids,omitempty"`
	ReviewerIDs  []uuid.UUID `json:"reviewer_ids,omitempty"`
}

// CreateBranch 创建分支
func (s *projectService) CreateBranch(ctx context.Context, repositoryID uuid.UUID, req *CreateBranchRequest, userID, tenantID uuid.UUID) (*models.Branch, error) {
	// 检查仓库访问权限
	gitRepo, err := s.gitClient.GetRepository(ctx, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("获取仓库信息失败: %w", err)
	}

	hasAccess, err := s.repo.CheckUserAccess(ctx, gitRepo.ProjectID, userID)
	if err != nil {
		return nil, fmt.Errorf("检查项目访问权限失败: %w", err)
	}
	if !hasAccess {
		return nil, errors.New("无权限在此仓库中创建分支")
	}

	// 构建Git网关请求
	gitReq := &client.CreateBranchRequest{
		Name:    req.Name,
		FromSHA: req.SourceRef,
	}

	// 调用Git网关创建分支
	gitBranch, err := s.gitClient.CreateBranch(ctx, repositoryID, gitReq)
	if err != nil {
		s.logger.Error("Git网关创建分支失败",
			zap.Error(err),
			zap.String("repository_id", repositoryID.String()),
			zap.String("branch_name", req.Name),
			zap.String("user_id", userID.String()))
		return nil, fmt.Errorf("创建分支失败: %w", err)
	}

	// 转换为项目服务的分支模型
	branch := &models.Branch{
		ID:           gitBranch.ID,
		RepositoryID: gitBranch.RepositoryID,
		Name:         gitBranch.Name,
		CommitSHA:    gitBranch.CommitSHA,
		IsDefault:    gitBranch.IsDefault,
		IsProtected:  gitBranch.IsProtected,
		CreatedBy:    userID,
		CreatedAt:    gitBranch.CreatedAt,
		UpdatedAt:    gitBranch.UpdatedAt,
	}

	s.logger.Info("分支创建成功",
		zap.String("repository_id", repositoryID.String()),
		zap.String("branch_name", branch.Name),
		zap.String("created_by", userID.String()))

	return branch, nil
}

// DeleteBranch 删除分支
func (s *projectService) DeleteBranch(ctx context.Context, repositoryID uuid.UUID, branchName string, userID, tenantID uuid.UUID) error {
	// 检查仓库访问权限
	gitRepo, err := s.gitClient.GetRepository(ctx, repositoryID)
	if err != nil {
		return fmt.Errorf("获取仓库信息失败: %w", err)
	}

	hasAccess, err := s.repo.CheckUserAccess(ctx, gitRepo.ProjectID, userID)
	if err != nil {
		return fmt.Errorf("检查项目访问权限失败: %w", err)
	}
	if !hasAccess {
		return errors.New("无权限删除此分支")
	}

	// 调用Git网关删除分支
	err = s.gitClient.DeleteBranch(ctx, repositoryID, branchName)
	if err != nil {
		s.logger.Error("Git网关删除分支失败",
			zap.Error(err),
			zap.String("repository_id", repositoryID.String()),
			zap.String("branch_name", branchName),
			zap.String("user_id", userID.String()))
		return fmt.Errorf("删除分支失败: %w", err)
	}

	s.logger.Info("分支删除成功",
		zap.String("repository_id", repositoryID.String()),
		zap.String("branch_name", branchName),
		zap.String("deleted_by", userID.String()))

	return nil
}

// ListBranches 获取分支列表
func (s *projectService) ListBranches(ctx context.Context, repositoryID uuid.UUID, page, pageSize int, userID, tenantID uuid.UUID) (*BranchListResponse, error) {
	// 检查仓库访问权限
	gitRepo, err := s.gitClient.GetRepository(ctx, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("获取仓库信息失败: %w", err)
	}

	hasAccess, err := s.repo.CheckUserAccess(ctx, gitRepo.ProjectID, userID)
	if err != nil {
		return nil, fmt.Errorf("检查项目访问权限失败: %w", err)
	}
	if !hasAccess {
		return nil, errors.New("无权限访问此仓库的分支")
	}

	// 验证分页参数
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 从Git网关获取分支列表
	gitBranchList, err := s.gitClient.ListBranches(ctx, repositoryID)
	if err != nil {
		s.logger.Error("Git网关获取分支列表失败",
			zap.Error(err),
			zap.String("repository_id", repositoryID.String()),
			zap.String("user_id", userID.String()))
		return nil, fmt.Errorf("获取分支列表失败: %w", err)
	}

	// 应用分页（简化实现，实际应在Git网关层实现）
	startIndex := (page - 1) * pageSize
	endIndex := startIndex + pageSize
	if startIndex >= len(gitBranchList) {
		startIndex = 0
		endIndex = 0
	} else if endIndex > len(gitBranchList) {
		endIndex = len(gitBranchList)
	}

	pagedBranches := gitBranchList[startIndex:endIndex]

	// 转换为项目服务的分支模型
	branches := make([]models.Branch, len(pagedBranches))
	for i, gitBranch := range pagedBranches {
		branches[i] = models.Branch{
			ID:           gitBranch.ID,
			RepositoryID: gitBranch.RepositoryID,
			Name:         gitBranch.Name,
			CommitSHA:    gitBranch.CommitSHA,
			IsDefault:    gitBranch.IsDefault,
			IsProtected:  gitBranch.IsProtected,
			CreatedAt:    gitBranch.CreatedAt,
			UpdatedAt:    gitBranch.UpdatedAt,
		}
	}

	return &BranchListResponse{
		Branches: branches,
		Total:    int64(len(gitBranchList)),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// CreatePullRequest 创建合并请求 (预留接口，等待Git网关实现)
func (s *projectService) CreatePullRequest(ctx context.Context, repositoryID uuid.UUID, req *CreatePullRequestRequest, userID, tenantID uuid.UUID) (*models.PullRequest, error) {
	// TODO: 等待Git网关实现Pull Request功能
	return nil, fmt.Errorf("合并请求功能正在开发中")
}

// GetPullRequest 获取合并请求详情 (预留接口，等待Git网关实现)
func (s *projectService) GetPullRequest(ctx context.Context, repositoryID, pullRequestID uuid.UUID, userID, tenantID uuid.UUID) (*models.PullRequest, error) {
	// TODO: 等待Git网关实现Pull Request功能
	return nil, fmt.Errorf("合并请求功能正在开发中")
}

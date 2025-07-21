package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProjectRepository 项目存储接口
type ProjectRepository interface {
	// 项目CRUD操作
	Create(ctx context.Context, project *models.Project) error
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Project, error)
	GetByKey(ctx context.Context, key string, tenantID uuid.UUID) (*models.Project, error)
	Update(ctx context.Context, project *models.Project) error
	Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, filters map[string]interface{}) ([]models.Project, int64, error)
	
	// 项目成员管理
	AddMember(ctx context.Context, member *models.ProjectMember) error
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
	GetMembers(ctx context.Context, projectID uuid.UUID) ([]models.ProjectMember, error)
	GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (*models.Role, error)
	
	// 权限检查
	CheckUserAccess(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
	GetUserProjects(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) ([]models.Project, error)
}

// projectRepository 项目存储实现
type projectRepository struct {
	db *gorm.DB
}

// NewProjectRepository 创建项目存储实例
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{
		db: db,
	}
}

// Create 创建项目
func (r *projectRepository) Create(ctx context.Context, project *models.Project) error {
	// 检查项目key在租户内是否唯一
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Project{}).
		Where("tenant_id = ? AND key = ? AND deleted_at IS NULL", project.TenantID, project.Key).
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("检查项目key唯一性失败: %w", err)
	}
	if count > 0 {
		return errors.New("项目key已存在")
	}

	// 创建项目
	if err := r.db.WithContext(ctx).Create(project).Error; err != nil {
		return fmt.Errorf("创建项目失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取项目
func (r *projectRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := r.db.WithContext(ctx).
		Preload("Manager").
		Preload("Members").
		Preload("Members.User").
		Preload("Members.Role").
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		First(&project).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("项目不存在")
		}
		return nil, fmt.Errorf("获取项目失败: %w", err)
	}

	return &project, nil
}

// GetByKey 根据key获取项目
func (r *projectRepository) GetByKey(ctx context.Context, key string, tenantID uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := r.db.WithContext(ctx).
		Preload("Manager").
		Preload("Members").
		Preload("Members.User").
		Preload("Members.Role").
		Where("key = ? AND tenant_id = ? AND deleted_at IS NULL", key, tenantID).
		First(&project).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("项目不存在")
		}
		return nil, fmt.Errorf("获取项目失败: %w", err)
	}

	return &project, nil
}

// Update 更新项目
func (r *projectRepository) Update(ctx context.Context, project *models.Project) error {
	// 如果更新了key，检查唯一性
	if project.Key != "" {
		var count int64
		err := r.db.WithContext(ctx).Model(&models.Project{}).
			Where("tenant_id = ? AND key = ? AND id != ? AND deleted_at IS NULL", 
				project.TenantID, project.Key, project.ID).
			Count(&count).Error
		if err != nil {
			return fmt.Errorf("检查项目key唯一性失败: %w", err)
		}
		if count > 0 {
			return errors.New("项目key已存在")
		}
	}

	err := r.db.WithContext(ctx).Save(project).Error
	if err != nil {
		return fmt.Errorf("更新项目失败: %w", err)
	}

	return nil
}

// Delete 删除项目（软删除）
func (r *projectRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Delete(&models.Project{})
	
	if result.Error != nil {
		return fmt.Errorf("删除项目失败: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return errors.New("项目不存在或无权限删除")
	}

	return nil
}

// List 获取项目列表
func (r *projectRepository) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, filters map[string]interface{}) ([]models.Project, int64, error) {
	var projects []models.Project
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Project{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID)

	// 应用过滤条件
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if managerID, ok := filters["manager_id"].(uuid.UUID); ok && managerID != uuid.Nil {
		query = query.Where("manager_id = ?", managerID)
	}
	if search, ok := filters["search"].(string); ok && search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计项目数量失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Preload("Manager").
		Preload("Members").
		Preload("Members.User").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&projects).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取项目列表失败: %w", err)
	}

	return projects, total, nil
}

// AddMember 添加项目成员
func (r *projectRepository) AddMember(ctx context.Context, member *models.ProjectMember) error {
	// 检查成员是否已存在
	var count int64
	err := r.db.WithContext(ctx).Model(&models.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", member.ProjectID, member.UserID).
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("检查成员是否存在失败: %w", err)
	}
	if count > 0 {
		return errors.New("用户已是项目成员")
	}

	// 添加成员
	if err := r.db.WithContext(ctx).Create(member).Error; err != nil {
		return fmt.Errorf("添加项目成员失败: %w", err)
	}

	return nil
}

// RemoveMember 移除项目成员
func (r *projectRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Delete(&models.ProjectMember{})
	
	if result.Error != nil {
		return fmt.Errorf("移除项目成员失败: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return errors.New("成员不存在")
	}

	return nil
}

// GetMembers 获取项目成员列表
func (r *projectRepository) GetMembers(ctx context.Context, projectID uuid.UUID) ([]models.ProjectMember, error) {
	var members []models.ProjectMember
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Role").
		Preload("AddedByUser").
		Where("project_id = ?", projectID).
		Find(&members).Error

	if err != nil {
		return nil, fmt.Errorf("获取项目成员失败: %w", err)
	}

	return members, nil
}

// GetMemberRole 获取用户在项目中的角色
func (r *projectRepository) GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (*models.Role, error) {
	var member models.ProjectMember
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("project_id = ? AND user_id = ?", projectID, userID).
		First(&member).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不是项目成员")
		}
		return nil, fmt.Errorf("获取用户角色失败: %w", err)
	}

	return member.Role, nil
}

// CheckUserAccess 检查用户是否有项目访问权限
func (r *projectRepository) CheckUserAccess(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	// 首先检查用户是否是项目管理员
	var project models.Project
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", projectID).
		First(&project).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("检查项目权限失败: %w", err)
	}
	
	// 如果用户是项目管理员，直接返回true
	if project.ManagerID != nil && *project.ManagerID == userID {
		return true, nil
	}
	
	// 否则检查用户是否是项目成员
	var count int64
	err = r.db.WithContext(ctx).Model(&models.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("检查用户权限失败: %w", err)
	}

	return count > 0, nil
}

// GetUserProjects 获取用户参与的项目列表
func (r *projectRepository) GetUserProjects(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) ([]models.Project, error) {
	var projects []models.Project
	
	err := r.db.WithContext(ctx).
		Joins("JOIN project_members ON projects.id = project_members.project_id").
		Preload("Manager").
		Where("project_members.user_id = ? AND projects.tenant_id = ? AND projects.deleted_at IS NULL", userID, tenantID).
		Find(&projects).Error

	if err != nil {
		return nil, fmt.Errorf("获取用户项目列表失败: %w", err)
	}

	return projects, nil
}
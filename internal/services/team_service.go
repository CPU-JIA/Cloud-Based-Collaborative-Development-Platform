package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/internal/models"
)

// TeamService 团队服务
type TeamService struct {
	db                *gorm.DB
	permissionService *PermissionService
}

// NewTeamService 创建团队服务实例
func NewTeamService(db *gorm.DB, permissionService *PermissionService) *TeamService {
	return &TeamService{
		db:                db,
		permissionService: permissionService,
	}
}

// CreateTeam 创建团队
func (ts *TeamService) CreateTeam(tenantID string, projectID int, name, description string, createdBy int) (*models.Team, error) {
	// 验证团队名称
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("团队名称不能为空")
	}

	// 验证创建者用户是否存在
	var user models.User
	err := ts.db.Where("id = ? AND status = ?", createdBy, models.UserStatusActive).First(&user).Error
	if err != nil {
		return nil, errors.New("创建者用户不存在或状态无效")
	}

	team := &models.Team{
		TenantID:    tenantID,
		ProjectID:   projectID,
		Name:        strings.TrimSpace(name),
		Description: description,
		IsActive:    true,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := ts.db.Create(team).Error; err != nil {
		return nil, fmt.Errorf("创建团队失败: %w", err)
	}

	// 创建默认角色
	if err := ts.createDefaultRoles(tenantID, projectID, team.ID, createdBy); err != nil {
		return nil, fmt.Errorf("创建默认角色失败: %w", err)
	}

	// 将创建者添加为团队所有者
	ownerRole, err := ts.GetRoleByName(tenantID, projectID, models.RoleOwner)
	if err != nil {
		return nil, fmt.Errorf("获取所有者角色失败: %w", err)
	}

	if err := ts.AddTeamMember(tenantID, team.ID, createdBy, ownerRole.ID, createdBy); err != nil {
		return nil, fmt.Errorf("添加团队创建者失败: %w", err)
	}

	// 记录活动
	ts.logActivity(tenantID, team.ID, createdBy, models.ActivityJoin, "team", &team.ID, "创建团队并加入", "", "")

	return team, nil
}

// GetTeam 获取团队信息
func (ts *TeamService) GetTeam(tenantID string, teamID int) (*models.Team, error) {
	var team models.Team
	err := ts.db.Where("id = ? AND tenant_id = ?", teamID, tenantID).
		Preload("Members").
		Preload("Members.User").
		Preload("Members.Role").
		Preload("Roles").
		First(&team).Error

	if err != nil {
		return nil, fmt.Errorf("团队不存在")
	}

	return &team, nil
}

// AddTeamMember 添加团队成员
func (ts *TeamService) AddTeamMember(tenantID string, teamID int, userID int, roleID int, invitedBy int) error {
	// 验证用户是否存在且状态有效
	var user models.User
	err := ts.db.Where("id = ? AND status = ?", userID, models.UserStatusActive).First(&user).Error
	if err != nil {
		return errors.New("用户不存在或状态无效")
	}

	// 验证角色是否存在
	var role models.Role
	err = ts.db.Where("id = ?", roleID).First(&role).Error
	if err != nil {
		return errors.New("角色不存在")
	}

	// 检查用户是否已经是团队成员
	var existingMember models.TeamMember
	err = ts.db.Where("tenant_id = ? AND team_id = ? AND user_id = ?", tenantID, teamID, userID).
		First(&existingMember).Error

	if err == nil {
		if existingMember.Status == models.MemberStatusActive {
			return errors.New("用户已经是团队成员")
		}
		// 重新激活成员
		existingMember.Status = models.MemberStatusActive
		existingMember.RoleID = roleID
		existingMember.JoinedAt = time.Now()
		existingMember.UpdatedAt = time.Now()
		return ts.db.Save(&existingMember).Error
	}

	member := &models.TeamMember{
		TenantID:  tenantID,
		TeamID:    teamID,
		UserID:    userID,
		RoleID:    roleID,
		Status:    models.MemberStatusActive,
		JoinedAt:  time.Now(),
		InvitedBy: invitedBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := ts.db.Create(member).Error; err != nil {
		return fmt.Errorf("添加团队成员失败: %w", err)
	}

	// 记录活动
	ts.logActivity(tenantID, teamID, userID, models.ActivityJoin, "user", &userID, "加入团队", "", "")

	return nil
}

// RemoveTeamMember 移除团队成员
func (ts *TeamService) RemoveTeamMember(tenantID string, teamID int, userID int, operatorID int) error {
	var member models.TeamMember
	err := ts.db.Where("tenant_id = ? AND team_id = ? AND user_id = ?", tenantID, teamID, userID).
		Preload("Role").First(&member).Error

	if err != nil {
		return fmt.Errorf("成员不存在")
	}

	// 检查是否是最后一个所有者
	if member.Role.Name == models.RoleOwner {
		var ownerCount int64
		ts.db.Table("team_members").
			Joins("JOIN roles ON team_members.role_id = roles.id").
			Where("team_members.tenant_id = ? AND team_members.team_id = ? AND team_members.status = ? AND roles.name = ?",
				tenantID, teamID, models.MemberStatusActive, models.RoleOwner).
			Count(&ownerCount)

		if ownerCount <= 1 {
			return errors.New("不能移除最后一个团队所有者")
		}
	}

	member.Status = models.MemberStatusInactive
	member.UpdatedAt = time.Now()

	if err := ts.db.Save(&member).Error; err != nil {
		return fmt.Errorf("移除团队成员失败: %w", err)
	}

	// 记录活动
	ts.logActivity(tenantID, teamID, operatorID, models.ActivityLeave, "user", &userID, "移除团队成员", "", "")

	return nil
}

// ChangeUserRole 更改用户角色
func (ts *TeamService) ChangeUserRole(tenantID string, teamID int, userID int, newRoleID int, operatorID int) error {
	// 验证新角色是否存在
	var newRole models.Role
	err := ts.db.Where("id = ?", newRoleID).First(&newRole).Error
	if err != nil {
		return errors.New("角色不存在")
	}

	var member models.TeamMember
	err = ts.db.Where("tenant_id = ? AND team_id = ? AND user_id = ?", tenantID, teamID, userID).
		Preload("Role").First(&member).Error

	if err != nil {
		return fmt.Errorf("成员不存在")
	}

	if !member.IsActive() {
		return fmt.Errorf("成员状态无效")
	}

	oldRoleID := member.RoleID

	// 使用明确的更新操作，避免与预加载的关联对象冲突
	result := ts.db.Model(&models.TeamMember{}).Where("id = ?", member.ID).Updates(map[string]interface{}{
		"role_id":    newRoleID,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		return fmt.Errorf("更改用户角色失败: %w", result.Error)
	}

	// 记录活动
	details := fmt.Sprintf("角色从 %d 更改为 %d", oldRoleID, newRoleID)
	ts.logActivity(tenantID, teamID, operatorID, models.ActivityRoleChange, "user", &userID, details, "", "")

	return nil
}

// InviteUser 邀请用户
func (ts *TeamService) InviteUser(tenantID string, teamID int, projectID int, email string, roleID int, message string, invitedBy int) (*models.TeamInvitation, error) {
	// 验证邮箱格式
	if !isValidEmail(email) {
		return nil, errors.New("邮箱格式不正确")
	}

	// 验证角色是否存在
	var role models.Role
	err := ts.db.Where("id = ?", roleID).First(&role).Error
	if err != nil {
		return nil, errors.New("角色不存在")
	}

	// 检查是否已经存在未处理的邀请
	var existingInvitation models.TeamInvitation
	err = ts.db.Where("tenant_id = ? AND team_id = ? AND email = ? AND status = ?",
		tenantID, teamID, email, models.InvitationStatusPending).First(&existingInvitation).Error

	if err == nil {
		return nil, errors.New("该邮箱已有待处理的邀请")
	}

	// 生成邀请令牌
	token, err := ts.generateInvitationToken()
	if err != nil {
		return nil, fmt.Errorf("生成邀请令牌失败: %w", err)
	}

	invitation := &models.TeamInvitation{
		TenantID:  tenantID,
		TeamID:    teamID,
		ProjectID: projectID,
		Email:     email,
		RoleID:    roleID,
		Token:     token,
		Status:    models.InvitationStatusPending,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7天过期
		Message:   message,
		InvitedBy: invitedBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := ts.db.Create(invitation).Error; err != nil {
		return nil, fmt.Errorf("创建邀请失败: %w", err)
	}

	// 这里可以添加发送邮件的逻辑
	// ts.sendInvitationEmail(invitation)

	return invitation, nil
}

// AcceptInvitation 接受邀请
func (ts *TeamService) AcceptInvitation(token string, userID int) error {
	var invitation models.TeamInvitation
	err := ts.db.Where("token = ?", token).Preload("Team").First(&invitation).Error

	if err != nil {
		return fmt.Errorf("邀请不存在")
	}

	if !invitation.IsValid() {
		return fmt.Errorf("邀请已过期或无效")
	}

	// 检查用户邮箱是否匹配
	var user models.User
	err = ts.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return fmt.Errorf("用户不存在")
	}

	if user.Email != invitation.Email {
		return fmt.Errorf("邮箱不匹配")
	}

	// 添加为团队成员
	if err := ts.AddTeamMember(invitation.TenantID, invitation.TeamID, userID, invitation.RoleID, invitation.InvitedBy); err != nil {
		return fmt.Errorf("添加团队成员失败: %w", err)
	}

	// 更新邀请状态
	invitation.Status = models.InvitationStatusAccepted
	invitation.AcceptedBy = &userID
	now := time.Now()
	invitation.AcceptedAt = &now
	invitation.UpdatedAt = now

	return ts.db.Save(&invitation).Error
}

// CreateRole 创建角色
func (ts *TeamService) CreateRole(tenantID string, projectID int, name, description string, permissions []string, createdBy int) (*models.Role, error) {
	role := &models.Role{
		TenantID:    tenantID,
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		Permissions: permissions,
		IsSystem:    false,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := ts.db.Create(role).Error; err != nil {
		return nil, fmt.Errorf("创建角色失败: %w", err)
	}

	return role, nil
}

// UpdateRole 更新角色
func (ts *TeamService) UpdateRole(tenantID string, roleID int, name, description string, permissions []string) error {
	var role models.Role
	err := ts.db.Where("id = ? AND tenant_id = ?", roleID, tenantID).First(&role).Error
	if err != nil {
		return fmt.Errorf("角色不存在")
	}

	if role.IsSystem {
		return fmt.Errorf("不能修改系统角色")
	}

	role.Name = name
	role.Description = description
	role.Permissions = permissions
	role.UpdatedAt = time.Now()

	return ts.db.Save(&role).Error
}

// GetRoles 获取角色列表
func (ts *TeamService) GetRoles(tenantID string, projectID int) ([]*models.Role, error) {
	var roles []*models.Role
	err := ts.db.Where("tenant_id = ? AND project_id = ?", tenantID, projectID).
		Order("is_system DESC, name ASC").Find(&roles).Error

	return roles, err
}

// GetRoleByName 根据名称获取角色
func (ts *TeamService) GetRoleByName(tenantID string, projectID int, name string) (*models.Role, error) {
	var role models.Role
	err := ts.db.Where("tenant_id = ? AND project_id = ? AND name = ?", tenantID, projectID, name).
		First(&role).Error

	if err != nil {
		return nil, fmt.Errorf("角色不存在: %s", name)
	}

	return &role, nil
}

// CreatePermissionRequest 创建权限申请
func (ts *TeamService) CreatePermissionRequest(tenantID string, projectID int, userID int, requestType, permission, reason string, targetID *int) (*models.PermissionRequest, error) {
	// 验证申请理由不能为空
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("申请理由不能为空")
	}

	// 验证用户是否存在
	var user models.User
	err := ts.db.Where("id = ? AND status = ?", userID, models.UserStatusActive).First(&user).Error
	if err != nil {
		return nil, errors.New("用户不存在或状态无效")
	}

	request := &models.PermissionRequest{
		TenantID:    tenantID,
		ProjectID:   projectID,
		UserID:      userID,
		RequestType: requestType,
		TargetID:    targetID,
		Permission:  permission,
		Reason:      strings.TrimSpace(reason),
		Status:      models.RequestStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := ts.db.Create(request).Error; err != nil {
		return nil, fmt.Errorf("创建权限申请失败: %w", err)
	}

	return request, nil
}

// ReviewPermissionRequest 审批权限申请
func (ts *TeamService) ReviewPermissionRequest(tenantID string, requestID int, reviewerID int, approved bool, reviewReason string) error {
	var request models.PermissionRequest
	err := ts.db.Where("id = ? AND tenant_id = ?", requestID, tenantID).First(&request).Error
	if err != nil {
		return fmt.Errorf("权限申请不存在")
	}

	if request.Status != models.RequestStatusPending {
		return fmt.Errorf("权限申请已被处理")
	}

	// 更新申请状态
	if approved {
		request.Status = models.RequestStatusApproved
	} else {
		request.Status = models.RequestStatusRejected
	}

	request.ReviewedBy = &reviewerID
	now := time.Now()
	request.ReviewedAt = &now
	request.ReviewReason = reviewReason
	request.UpdatedAt = now

	if err := ts.db.Save(&request).Error; err != nil {
		return fmt.Errorf("更新权限申请失败: %w", err)
	}

	// 如果申请被批准，执行相应的权限操作
	if approved {
		switch request.RequestType {
		case models.RequestTypeRole:
			// 处理角色申请
			if request.TargetID != nil {
				// 这里需要实现角色分配逻辑
			}
		case models.RequestTypeFilePermission:
			// 处理文件权限申请
			if request.TargetID != nil {
				_, err := ts.permissionService.CreateFilePermission(
					tenantID, request.TargetID, nil, &request.UserID, nil,
					request.Permission, reviewerID, request.ExpiresAt)
				if err != nil {
					return fmt.Errorf("分配文件权限失败: %w", err)
				}
			}
		case models.RequestTypeFolderPermission:
			// 处理文件夹权限申请
			if request.TargetID != nil {
				_, err := ts.permissionService.CreateFilePermission(
					tenantID, nil, request.TargetID, &request.UserID, nil,
					request.Permission, reviewerID, request.ExpiresAt)
				if err != nil {
					return fmt.Errorf("分配文件夹权限失败: %w", err)
				}
			}
		}
	}

	return nil
}

// GetTeamMembers 获取团队成员列表
func (ts *TeamService) GetTeamMembers(tenantID string, teamID int, status string) ([]*models.TeamMember, error) {
	var members []*models.TeamMember
	query := ts.db.Where("tenant_id = ? AND team_id = ?", tenantID, teamID).
		Preload("User").Preload("Role")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Find(&members).Error
	return members, err
}

// GetUserPermissions 获取用户在项目中的所有权限
func (ts *TeamService) GetUserPermissions(tenantID string, projectID int, userID int) ([]string, error) {
	var permissions []string
	permissionMap := make(map[string]bool)

	// 获取用户的角色权限
	var roles []models.Role
	err := ts.db.Table("roles").
		Joins("JOIN team_members ON roles.id = team_members.role_id").
		Joins("JOIN teams ON team_members.team_id = teams.id").
		Where("teams.tenant_id = ? AND teams.project_id = ? AND team_members.user_id = ? AND team_members.status = ?",
			tenantID, projectID, userID, models.MemberStatusActive).
		Find(&roles).Error

	if err != nil {
		return nil, err
	}

	// 收集所有权限
	for _, role := range roles {
		for _, permission := range role.Permissions {
			permissionMap[permission] = true
		}
	}

	// 转换为切片
	for permission := range permissionMap {
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// SearchUsers 搜索用户
func (ts *TeamService) SearchUsers(tenantID string, query string, limit int) ([]*models.User, error) {
	var users []*models.User

	searchQuery := ts.db.Where("tenant_id = ? AND status = ?", tenantID, models.UserStatusActive)

	if query != "" {
		searchTerm := "%" + strings.ToLower(query) + "%"
		searchQuery = searchQuery.Where(
			"LOWER(username) LIKE ? OR LOWER(email) LIKE ? OR LOWER(display_name) LIKE ?",
			searchTerm, searchTerm, searchTerm)
	}

	if limit > 0 {
		searchQuery = searchQuery.Limit(limit)
	}

	err := searchQuery.Find(&users).Error
	return users, err
}

// createDefaultRoles 创建默认角色
func (ts *TeamService) createDefaultRoles(tenantID string, projectID int, teamID int, createdBy int) error {
	defaultRoles := []struct {
		Name        string
		Description string
		Permissions []string
	}{
		{
			Name:        models.RoleOwner,
			Description: "项目所有者，拥有所有权限",
			Permissions: models.GetDefaultRolePermissions(models.RoleOwner),
		},
		{
			Name:        models.RoleAdmin,
			Description: "项目管理员，拥有管理权限",
			Permissions: models.GetDefaultRolePermissions(models.RoleAdmin),
		},
		{
			Name:        models.RoleMember,
			Description: "项目成员，拥有基本操作权限",
			Permissions: models.GetDefaultRolePermissions(models.RoleMember),
		},
		{
			Name:        models.RoleViewer,
			Description: "项目查看者，只有查看权限",
			Permissions: models.GetDefaultRolePermissions(models.RoleViewer),
		},
	}

	for _, roleData := range defaultRoles {
		// 检查角色是否已存在
		var existingRole models.Role
		if err := ts.db.Where("tenant_id = ? AND project_id = ? AND name = ?",
			tenantID, projectID, roleData.Name).First(&existingRole).Error; err == nil {
			// 角色已存在，跳过
			continue
		}

		role := &models.Role{
			TenantID:    tenantID,
			ProjectID:   projectID,
			Name:        roleData.Name,
			Description: roleData.Description,
			Permissions: roleData.Permissions,
			IsSystem:    true,
			CreatedBy:   createdBy,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := ts.db.Create(role).Error; err != nil {
			return err
		}
	}

	return nil
}

// generateInvitationToken 生成邀请令牌
func (ts *TeamService) generateInvitationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// logActivity 记录团队活动
func (ts *TeamService) logActivity(tenantID string, teamID int, userID int, action, targetType string, targetID *int, details, ipAddress, userAgent string) {
	activity := &models.TeamActivity{
		TenantID:   tenantID,
		TeamID:     teamID,
		UserID:     userID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Details:    details,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		CreatedAt:  time.Now(),
	}

	ts.db.Create(activity)
}

// GetTeamActivities 获取团队活动记录
func (ts *TeamService) GetTeamActivities(tenantID string, teamID int, page, pageSize int) ([]*models.TeamActivity, int64, error) {
	var activities []*models.TeamActivity
	var total int64

	query := ts.db.Where("tenant_id = ? AND team_id = ?", tenantID, teamID)

	// 统计总数
	query.Model(&models.TeamActivity{}).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Preload("User").Preload("Team").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&activities).Error

	return activities, total, err
}

// CheckUserPermission 检查用户权限（委托给权限服务）
func (ts *TeamService) CheckUserPermission(userID int, projectID int, permission string) (bool, error) {
	return ts.permissionService.CheckUserPermission(userID, projectID, permission)
}

// GetTeamsByProject 获取项目的团队列表
func (ts *TeamService) GetTeamsByProject(tenantID string, projectID int) ([]*models.Team, error) {
	var teams []*models.Team
	err := ts.db.Where("tenant_id = ? AND project_id = ? AND is_active = ?", tenantID, projectID, true).
		Preload("Members").
		Preload("Members.User").
		Preload("Members.Role").
		Find(&teams).Error

	return teams, err
}

// GetRolesByProject 获取项目的角色列表
func (ts *TeamService) GetRolesByProject(tenantID string, projectID int) ([]*models.Role, error) {
	return ts.GetRoles(tenantID, projectID)
}

// AddTeamMemberCompat 兼容测试的添加团队成员方法
func (ts *TeamService) AddTeamMemberCompat(teamID int, userID int, roleID int, invitedBy int) (*models.TeamMember, error) {
	// 获取团队信息确定租户ID
	var team models.Team
	if err := ts.db.First(&team, teamID).Error; err != nil {
		return nil, fmt.Errorf("团队不存在")
	}

	// 调用原始方法
	err := ts.AddTeamMember(team.TenantID, teamID, userID, roleID, invitedBy)
	if err != nil {
		return nil, err
	}

	// 返回创建的成员
	var member models.TeamMember
	err = ts.db.Where("team_id = ? AND user_id = ?", teamID, userID).
		Preload("User").Preload("Role").First(&member).Error

	return &member, err
}

// UpdateMemberRoleCompat 兼容测试的更新成员角色方法
func (ts *TeamService) UpdateMemberRoleCompat(teamID int, userID int, roleID int) error {
	// 获取团队信息确定租户ID
	var team models.Team
	if err := ts.db.First(&team, teamID).Error; err != nil {
		return fmt.Errorf("团队不存在")
	}

	return ts.ChangeUserRole(team.TenantID, teamID, userID, roleID, 1) // 默认操作者为1
}

// RemoveTeamMemberCompat 兼容测试的移除团队成员方法
func (ts *TeamService) RemoveTeamMemberCompat(teamID int, userID int) error {
	// 获取团队信息确定租户ID
	var team models.Team
	if err := ts.db.First(&team, teamID).Error; err != nil {
		return fmt.Errorf("团队不存在")
	}

	return ts.RemoveTeamMember(team.TenantID, teamID, userID, 1) // 默认操作者为1
}

// InviteUserCompat 兼容测试的邀请用户方法
func (ts *TeamService) InviteUserCompat(teamID int, email string, roleID int, message string, invitedBy int) (*models.TeamInvitation, error) {
	// 获取团队信息确定租户ID和项目ID
	var team models.Team
	if err := ts.db.First(&team, teamID).Error; err != nil {
		return nil, fmt.Errorf("团队不存在")
	}

	return ts.InviteUser(team.TenantID, teamID, team.ProjectID, email, roleID, message, invitedBy)
}

// CreatePermissionRequestCompat 兼容测试的创建权限申请方法
func (ts *TeamService) CreatePermissionRequestCompat(projectID int, userID int, requestType, permission, reason string, targetID *int) (*models.PermissionRequest, error) {
	return ts.CreatePermissionRequest("default", projectID, userID, requestType, permission, reason, targetID)
}

// ReviewPermissionRequestCompat 兼容测试的审批权限申请方法
func (ts *TeamService) ReviewPermissionRequestCompat(requestID int, reviewerID int, approved bool, reviewReason string) error {
	return ts.ReviewPermissionRequest("default", requestID, reviewerID, approved, reviewReason)
}

// isValidEmail 验证邮箱格式
func isValidEmail(email string) bool {
	if email == "" {
		return false
	}
	// 简单的邮箱格式验证
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

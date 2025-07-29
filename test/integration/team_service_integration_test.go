package integration

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 团队管理相关数据模型
type Team struct {
	ID          int              `json:"id" gorm:"primary_key"`
	TenantID    string           `json:"tenant_id" gorm:"index"`
	ProjectID   int              `json:"project_id" gorm:"index"`
	Name        string           `json:"name" gorm:"not null"`
	Description string           `json:"description"`
	IsActive    bool             `json:"is_active" gorm:"default:true"`
	CreatedBy   int              `json:"created_by"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Members     []TeamMember     `json:"members" gorm:"foreignKey:TeamID"`
	Roles       []Role           `json:"roles" gorm:"foreignKey:ProjectID"`
	Activities  []TeamActivity   `json:"activities" gorm:"foreignKey:TeamID"`
	Invitations []TeamInvitation `json:"invitations" gorm:"foreignKey:TeamID"`
}

type TeamMember struct {
	ID        int       `json:"id" gorm:"primary_key"`
	TenantID  string    `json:"tenant_id" gorm:"index"`
	TeamID    int       `json:"team_id" gorm:"index"`
	UserID    int       `json:"user_id" gorm:"index"`
	RoleID    int       `json:"role_id"`
	Status    string    `json:"status" gorm:"default:'active'"`
	JoinedAt  time.Time `json:"joined_at"`
	InvitedBy int       `json:"invited_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Role      *Role     `json:"role,omitempty" gorm:"foreignKey:RoleID"`
}

type User struct {
	ID          int       `json:"id" gorm:"primary_key"`
	TenantID    string    `json:"tenant_id" gorm:"index"`
	Username    string    `json:"username" gorm:"unique;not null"`
	Email       string    `json:"email" gorm:"unique;not null"`
	DisplayName string    `json:"display_name"`
	Status      string    `json:"status" gorm:"default:'active'"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Role struct {
	ID          int      `json:"id" gorm:"primary_key"`
	TenantID    string   `json:"tenant_id" gorm:"index"`
	ProjectID   int      `json:"project_id" gorm:"index"`
	Name        string   `json:"name" gorm:"not null"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions" gorm:"type:json"`
	IsSystem    bool     `json:"is_system" gorm:"default:false"`
	CreatedBy   int      `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TeamInvitation struct {
	ID         int        `json:"id" gorm:"primary_key"`
	TenantID   string     `json:"tenant_id" gorm:"index"`
	TeamID     int        `json:"team_id" gorm:"index"`
	ProjectID  int        `json:"project_id"`
	Email      string     `json:"email" gorm:"not null"`
	RoleID     int        `json:"role_id"`
	Token      string     `json:"token" gorm:"unique;not null"`
	Status     string     `json:"status" gorm:"default:'pending'"`
	ExpiresAt  time.Time  `json:"expires_at"`
	Message    string     `json:"message"`
	InvitedBy  int        `json:"invited_by"`
	AcceptedBy *int       `json:"accepted_by,omitempty"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Team       *Team      `json:"team,omitempty" gorm:"foreignKey:TeamID"`
	Role       *Role      `json:"role,omitempty" gorm:"foreignKey:RoleID"`
}

type PermissionRequest struct {
	ID           int        `json:"id" gorm:"primary_key"`
	TenantID     string     `json:"tenant_id" gorm:"index"`
	ProjectID    int        `json:"project_id" gorm:"index"`
	UserID       int        `json:"user_id" gorm:"index"`
	RequestType  string     `json:"request_type" gorm:"not null"`
	TargetID     *int       `json:"target_id,omitempty"`
	Permission   string     `json:"permission" gorm:"not null"`
	Reason       string     `json:"reason" gorm:"not null"`
	Status       string     `json:"status" gorm:"default:'pending'"`
	ReviewedBy   *int       `json:"reviewed_by,omitempty"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	ReviewReason string     `json:"review_reason"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	User         *User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Reviewer     *User      `json:"reviewer,omitempty" gorm:"foreignKey:ReviewedBy"`
}

type TeamActivity struct {
	ID         int       `json:"id" gorm:"primary_key"`
	TenantID   string    `json:"tenant_id" gorm:"index"`
	TeamID     int       `json:"team_id" gorm:"index"`
	UserID     int       `json:"user_id" gorm:"index"`
	Action     string    `json:"action" gorm:"not null"`
	TargetType string    `json:"target_type"`
	TargetID   *int      `json:"target_id,omitempty"`
	Details    string    `json:"details"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
	User       *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Team       *Team     `json:"team,omitempty" gorm:"foreignKey:TeamID"`
}

// 常量定义
const (
	// 用户状态
	UserStatusActive   = "active"
	UserStatusInactive = "inactive"
	UserStatusSuspended = "suspended"

	// 成员状态
	MemberStatusActive   = "active"
	MemberStatusInactive = "inactive"

	// 邀请状态
	InvitationStatusPending  = "pending"
	InvitationStatusAccepted = "accepted"
	InvitationStatusRejected = "rejected"
	InvitationStatusExpired  = "expired"

	// 权限申请状态
	RequestStatusPending  = "pending"
	RequestStatusApproved = "approved"
	RequestStatusRejected = "rejected"

	// 角色名称
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"

	// 活动类型
	ActivityJoin       = "join"
	ActivityLeave      = "leave"
	ActivityRoleChange = "role_change"

	// 权限申请类型
	RequestTypeRole           = "role"
	RequestTypeFilePermission = "file_permission"
	RequestTypeFolderPermission = "folder_permission"
)

// 成员方法
func (m *TeamMember) IsActive() bool {
	return m.Status == MemberStatusActive
}

// 邀请方法
func (i *TeamInvitation) IsValid() bool {
	return i.Status == InvitationStatusPending && time.Now().Before(i.ExpiresAt)
}

// 角色权限定义
func GetDefaultRolePermissions(roleName string) []string {
	switch roleName {
	case RoleOwner:
		return []string{"*"}
	case RoleAdmin:
		return []string{"team:manage", "member:manage", "role:manage", "project:read", "project:write"}
	case RoleMember:
		return []string{"project:read", "project:write", "team:read"}
	case RoleViewer:
		return []string{"project:read", "team:read"}
	default:
		return []string{"project:read"}
	}
}

// TeamServiceIntegrationTestSuite 团队服务集成测试套件
type TeamServiceIntegrationTestSuite struct {
	suite.Suite
	db *gorm.DB
}

// SetupSuite 测试套件初始化
func (suite *TeamServiceIntegrationTestSuite) SetupSuite() {
	// 创建内存SQLite数据库
	var err error
	suite.db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(suite.T(), err)

	// 自动迁移
	err = suite.db.AutoMigrate(
		&User{}, &Team{}, &TeamMember{}, &Role{}, 
		&TeamInvitation{}, &PermissionRequest{}, &TeamActivity{},
	)
	require.NoError(suite.T(), err)
}

// SetupTest 每个测试前的初始化
func (suite *TeamServiceIntegrationTestSuite) SetupTest() {
	// 清理数据库
	suite.db.Exec("DELETE FROM team_activities")
	suite.db.Exec("DELETE FROM permission_requests")
	suite.db.Exec("DELETE FROM team_invitations")
	suite.db.Exec("DELETE FROM team_members")
	suite.db.Exec("DELETE FROM teams")
	suite.db.Exec("DELETE FROM roles")
	suite.db.Exec("DELETE FROM users")

	// 创建测试用户
	suite.createTestUsers()
	
	// 创建默认角色
	suite.createDefaultRoles()
}

// createTestUsers 创建测试用户
func (suite *TeamServiceIntegrationTestSuite) createTestUsers() {
	users := []User{
		{ID: 1, TenantID: "test-tenant", Username: "owner", Email: "owner@test.com", DisplayName: "Team Owner", Status: UserStatusActive},
		{ID: 2, TenantID: "test-tenant", Username: "admin", Email: "admin@test.com", DisplayName: "Team Admin", Status: UserStatusActive},
		{ID: 3, TenantID: "test-tenant", Username: "member1", Email: "member1@test.com", DisplayName: "Member One", Status: UserStatusActive},
		{ID: 4, TenantID: "test-tenant", Username: "member2", Email: "member2@test.com", DisplayName: "Member Two", Status: UserStatusActive},
		{ID: 5, TenantID: "test-tenant", Username: "viewer", Email: "viewer@test.com", DisplayName: "Team Viewer", Status: UserStatusActive},
		{ID: 6, TenantID: "test-tenant", Username: "inactive", Email: "inactive@test.com", DisplayName: "Inactive User", Status: UserStatusInactive},
	}

	for _, user := range users {
		suite.db.Create(&user)
	}
}

// createDefaultRoles 创建默认角色
func (suite *TeamServiceIntegrationTestSuite) createDefaultRoles() {
	roles := []Role{
		{ID: 1, TenantID: "test-tenant", ProjectID: 1, Name: RoleOwner, Description: "项目所有者", Permissions: GetDefaultRolePermissions(RoleOwner), IsSystem: true, CreatedBy: 1},
		{ID: 2, TenantID: "test-tenant", ProjectID: 1, Name: RoleAdmin, Description: "项目管理员", Permissions: GetDefaultRolePermissions(RoleAdmin), IsSystem: true, CreatedBy: 1},
		{ID: 3, TenantID: "test-tenant", ProjectID: 1, Name: RoleMember, Description: "项目成员", Permissions: GetDefaultRolePermissions(RoleMember), IsSystem: true, CreatedBy: 1},
		{ID: 4, TenantID: "test-tenant", ProjectID: 1, Name: RoleViewer, Description: "项目查看者", Permissions: GetDefaultRolePermissions(RoleViewer), IsSystem: true, CreatedBy: 1},
	}

	for _, role := range roles {
		suite.db.Create(&role)
	}
}

// TeamService 团队服务实现
type TeamService struct {
	db *gorm.DB
}

// NewTeamService 创建团队服务
func NewTeamService(db *gorm.DB) *TeamService {
	return &TeamService{db: db}
}

// CreateTeam 创建团队
func (ts *TeamService) CreateTeam(tenantID string, projectID int, name, description string, createdBy int) (*Team, error) {
	if err := suite.validateTeamName(name); err != nil {
		return nil, err
	}

	// 验证创建者用户是否存在
	var user User
	err := ts.db.Where("id = ? AND status = ?", createdBy, UserStatusActive).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("创建者用户不存在或状态无效")
	}

	team := &Team{
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

	// 将创建者添加为团队所有者
	ownerRole, err := ts.GetRoleByName(tenantID, projectID, RoleOwner)
	if err != nil {
		return nil, fmt.Errorf("获取所有者角色失败: %w", err)
	}

	if err := ts.AddTeamMember(tenantID, team.ID, createdBy, ownerRole.ID, createdBy); err != nil {
		return nil, fmt.Errorf("添加团队创建者失败: %w", err)
	}

	// 记录活动
	ts.logActivity(tenantID, team.ID, createdBy, ActivityJoin, "team", &team.ID, "创建团队并加入", "", "")

	return team, nil
}

// GetTeam 获取团队信息
func (ts *TeamService) GetTeam(tenantID string, teamID int) (*Team, error) {
	var team Team
	err := ts.db.Where("id = ? AND tenant_id = ?", teamID, tenantID).
		Preload("Members").
		Preload("Members.User").
		Preload("Members.Role").
		First(&team).Error

	if err != nil {
		return nil, fmt.Errorf("团队不存在")
	}

	return &team, nil
}

// AddTeamMember 添加团队成员
func (ts *TeamService) AddTeamMember(tenantID string, teamID int, userID int, roleID int, invitedBy int) error {
	// 验证用户是否存在且状态有效
	var user User
	err := ts.db.Where("id = ? AND status = ?", userID, UserStatusActive).First(&user).Error
	if err != nil {
		return fmt.Errorf("用户不存在或状态无效")
	}

	// 验证角色是否存在
	var role Role
	err = ts.db.Where("id = ?", roleID).First(&role).Error
	if err != nil {
		return fmt.Errorf("角色不存在")
	}

	// 检查用户是否已经是团队成员
	var existingMember TeamMember
	err = ts.db.Where("tenant_id = ? AND team_id = ? AND user_id = ?", tenantID, teamID, userID).
		First(&existingMember).Error

	if err == nil {
		if existingMember.Status == MemberStatusActive {
			return fmt.Errorf("用户已经是团队成员")
		}
		// 重新激活成员
		existingMember.Status = MemberStatusActive
		existingMember.RoleID = roleID
		existingMember.JoinedAt = time.Now()
		existingMember.UpdatedAt = time.Now()
		return ts.db.Save(&existingMember).Error
	}

	member := &TeamMember{
		TenantID:  tenantID,
		TeamID:    teamID,
		UserID:    userID,
		RoleID:    roleID,
		Status:    MemberStatusActive,
		JoinedAt:  time.Now(),
		InvitedBy: invitedBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := ts.db.Create(member).Error; err != nil {
		return fmt.Errorf("添加团队成员失败: %w", err)
	}

	// 记录活动
	ts.logActivity(tenantID, teamID, userID, ActivityJoin, "user", &userID, "加入团队", "", "")

	return nil
}

// RemoveTeamMember 移除团队成员
func (ts *TeamService) RemoveTeamMember(tenantID string, teamID int, userID int, operatorID int) error {
	var member TeamMember
	err := ts.db.Where("tenant_id = ? AND team_id = ? AND user_id = ?", tenantID, teamID, userID).
		Preload("Role").First(&member).Error

	if err != nil {
		return fmt.Errorf("成员不存在")
	}

	// 检查是否是最后一个所有者
	if member.Role.Name == RoleOwner {
		var ownerCount int64
		ts.db.Table("team_members").
			Joins("JOIN roles ON team_members.role_id = roles.id").
			Where("team_members.tenant_id = ? AND team_members.team_id = ? AND team_members.status = ? AND roles.name = ?",
				tenantID, teamID, MemberStatusActive, RoleOwner).
			Count(&ownerCount)

		if ownerCount <= 1 {
			return fmt.Errorf("不能移除最后一个团队所有者")
		}
	}

	member.Status = MemberStatusInactive
	member.UpdatedAt = time.Now()

	if err := ts.db.Save(&member).Error; err != nil {
		return fmt.Errorf("移除团队成员失败: %w", err)
	}

	// 记录活动
	ts.logActivity(tenantID, teamID, operatorID, ActivityLeave, "user", &userID, "移除团队成员", "", "")

	return nil
}

// ChangeUserRole 更改用户角色
func (ts *TeamService) ChangeUserRole(tenantID string, teamID int, userID int, newRoleID int, operatorID int) error {
	// 验证新角色是否存在
	var newRole Role
	err := ts.db.Where("id = ?", newRoleID).First(&newRole).Error
	if err != nil {
		return fmt.Errorf("角色不存在")
	}

	var member TeamMember
	err = ts.db.Where("tenant_id = ? AND team_id = ? AND user_id = ?", tenantID, teamID, userID).
		Preload("Role").First(&member).Error

	if err != nil {
		return fmt.Errorf("成员不存在")
	}

	if !member.IsActive() {
		return fmt.Errorf("成员状态无效")
	}

	oldRoleID := member.RoleID

	// 更新角色
	result := ts.db.Model(&TeamMember{}).Where("id = ?", member.ID).Updates(map[string]interface{}{
		"role_id":    newRoleID,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		return fmt.Errorf("更改用户角色失败: %w", result.Error)
	}

	// 记录活动
	details := fmt.Sprintf("角色从 %d 更改为 %d", oldRoleID, newRoleID)
	ts.logActivity(tenantID, teamID, operatorID, ActivityRoleChange, "user", &userID, details, "", "")

	return nil
}

// InviteUser 邀请用户
func (ts *TeamService) InviteUser(tenantID string, teamID int, projectID int, email string, roleID int, message string, invitedBy int) (*TeamInvitation, error) {
	// 验证邮箱格式
	if !isValidEmail(email) {
		return nil, fmt.Errorf("邮箱格式不正确")
	}

	// 验证角色是否存在
	var role Role
	err := ts.db.Where("id = ?", roleID).First(&role).Error
	if err != nil {
		return nil, fmt.Errorf("角色不存在")
	}

	// 检查是否已经存在未处理的邀请
	var existingInvitation TeamInvitation
	err = ts.db.Where("tenant_id = ? AND team_id = ? AND email = ? AND status = ?",
		tenantID, teamID, email, InvitationStatusPending).First(&existingInvitation).Error

	if err == nil {
		return nil, fmt.Errorf("该邮箱已有待处理的邀请")
	}

	// 生成邀请令牌
	token := suite.generateToken()

	invitation := &TeamInvitation{
		TenantID:  tenantID,
		TeamID:    teamID,
		ProjectID: projectID,
		Email:     email,
		RoleID:    roleID,
		Token:     token,
		Status:    InvitationStatusPending,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7天过期
		Message:   message,
		InvitedBy: invitedBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := ts.db.Create(invitation).Error; err != nil {
		return nil, fmt.Errorf("创建邀请失败: %w", err)
	}

	return invitation, nil
}

// AcceptInvitation 接受邀请
func (ts *TeamService) AcceptInvitation(token string, userID int) error {
	var invitation TeamInvitation
	err := ts.db.Where("token = ?", token).Preload("Team").First(&invitation).Error

	if err != nil {
		return fmt.Errorf("邀请不存在")
	}

	if !invitation.IsValid() {
		return fmt.Errorf("邀请已过期或无效")
	}

	// 检查用户邮箱是否匹配
	var user User
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
	invitation.Status = InvitationStatusAccepted
	invitation.AcceptedBy = &userID
	now := time.Now()
	invitation.AcceptedAt = &now
	invitation.UpdatedAt = now

	return ts.db.Save(&invitation).Error
}

// CreatePermissionRequest 创建权限申请
func (ts *TeamService) CreatePermissionRequest(tenantID string, projectID int, userID int, requestType, permission, reason string, targetID *int) (*PermissionRequest, error) {
	// 验证申请理由不能为空
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("申请理由不能为空")
	}

	// 验证用户是否存在
	var user User
	err := ts.db.Where("id = ? AND status = ?", userID, UserStatusActive).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("用户不存在或状态无效")
	}

	request := &PermissionRequest{
		TenantID:    tenantID,
		ProjectID:   projectID,
		UserID:      userID,
		RequestType: requestType,
		TargetID:    targetID,
		Permission:  permission,
		Reason:      strings.TrimSpace(reason),
		Status:      RequestStatusPending,
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
	var request PermissionRequest
	err := ts.db.Where("id = ? AND tenant_id = ?", requestID, tenantID).First(&request).Error
	if err != nil {
		return fmt.Errorf("权限申请不存在")
	}

	if request.Status != RequestStatusPending {
		return fmt.Errorf("权限申请已被处理")
	}

	// 更新申请状态
	if approved {
		request.Status = RequestStatusApproved
	} else {
		request.Status = RequestStatusRejected
	}

	request.ReviewedBy = &reviewerID
	now := time.Now()
	request.ReviewedAt = &now
	request.ReviewReason = reviewReason
	request.UpdatedAt = now

	return ts.db.Save(&request).Error
}

// GetTeamMembers 获取团队成员列表
func (ts *TeamService) GetTeamMembers(tenantID string, teamID int, status string) ([]*TeamMember, error) {
	var members []*TeamMember
	query := ts.db.Where("tenant_id = ? AND team_id = ?", tenantID, teamID).
		Preload("User").Preload("Role")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Find(&members).Error
	return members, err
}

// GetRoleByName 根据名称获取角色
func (ts *TeamService) GetRoleByName(tenantID string, projectID int, name string) (*Role, error) {
	var role Role
	err := ts.db.Where("tenant_id = ? AND project_id = ? AND name = ?", tenantID, projectID, name).
		First(&role).Error

	if err != nil {
		return nil, fmt.Errorf("角色不存在: %s", name)
	}

	return &role, nil
}

// GetRoles 获取角色列表
func (ts *TeamService) GetRoles(tenantID string, projectID int) ([]*Role, error) {
	var roles []*Role
	err := ts.db.Where("tenant_id = ? AND project_id = ?", tenantID, projectID).
		Order("is_system DESC, name ASC").Find(&roles).Error

	return roles, err
}

// logActivity 记录团队活动
func (ts *TeamService) logActivity(tenantID string, teamID int, userID int, action, targetType string, targetID *int, details, ipAddress, userAgent string) {
	activity := &TeamActivity{
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
func (ts *TeamService) GetTeamActivities(tenantID string, teamID int, page, pageSize int) ([]*TeamActivity, int64, error) {
	var activities []*TeamActivity
	var total int64

	query := ts.db.Where("tenant_id = ? AND team_id = ?", tenantID, teamID)

	// 统计总数
	query.Model(&TeamActivity{}).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Preload("User").Preload("Team").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&activities).Error

	return activities, total, err
}

// 工具函数

// validateTeamName 验证团队名称
func (suite *TeamServiceIntegrationTestSuite) validateTeamName(name string) error {
	if name == "" {
		return fmt.Errorf("团队名称不能为空")
	}
	if len(name) > 100 {
		return fmt.Errorf("团队名称不能超过100个字符")
	}
	// 团队名称格式检查：允许字母、数字、中文、空格、下划线、连字符
	namePattern := regexp.MustCompile(`^[\w\s\x{4e00}-\x{9fff}-]+$`)
	if !namePattern.MatchString(name) {
		return fmt.Errorf("团队名称格式无效")
	}
	return nil
}

// generateToken 生成令牌
func (suite *TeamServiceIntegrationTestSuite) generateToken() string {
	return fmt.Sprintf("token_%d", time.Now().UnixNano())
}

// isValidEmail 验证邮箱格式
func isValidEmail(email string) bool {
	if email == "" {
		return false
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// 测试用例

// TestCreateTeam 测试创建团队
func (suite *TeamServiceIntegrationTestSuite) TestCreateTeam() {
	ts := NewTeamService(suite.db)

	// 成功创建团队
	team, err := ts.CreateTeam("test-tenant", 1, "开发团队", "前端开发团队", 1)
	require.NoError(suite.T(), err)
	assert.NotZero(suite.T(), team.ID)
	assert.Equal(suite.T(), "开发团队", team.Name)
	assert.Equal(suite.T(), "前端开发团队", team.Description)
	assert.Equal(suite.T(), 1, team.CreatedBy)
	assert.True(suite.T(), team.IsActive)

	// 验证创建者成为团队所有者
	members, err := ts.GetTeamMembers("test-tenant", team.ID, MemberStatusActive)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), members, 1)
	assert.Equal(suite.T(), 1, members[0].UserID)
	assert.Equal(suite.T(), RoleOwner, members[0].Role.Name)

	// 验证活动记录
	activities, total, err := ts.GetTeamActivities("test-tenant", team.ID, 1, 10)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1), total)
	assert.Equal(suite.T(), ActivityJoin, activities[0].Action)
}

// TestCreateTeamValidation 测试创建团队验证
func (suite *TeamServiceIntegrationTestSuite) TestCreateTeamValidation() {
	ts := NewTeamService(suite.db)

	// 空团队名称
	_, err := ts.CreateTeam("test-tenant", 1, "", "描述", 1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "团队名称不能为空")

	// 团队名称过长
	longName := strings.Repeat("a", 101)
	_, err = ts.CreateTeam("test-tenant", 1, longName, "描述", 1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "团队名称不能超过100个字符")

	// 无效创建者
	_, err = ts.CreateTeam("test-tenant", 1, "团队名称", "描述", 999)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "创建者用户不存在或状态无效")

	// 非活跃用户创建团队
	_, err = ts.CreateTeam("test-tenant", 1, "团队名称", "描述", 6) // 用户6是非活跃用户
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "创建者用户不存在或状态无效")
}

// TestAddTeamMember 测试添加团队成员
func (suite *TeamServiceIntegrationTestSuite) TestAddTeamMember() {
	ts := NewTeamService(suite.db)

	// 创建测试团队
	team, err := ts.CreateTeam("test-tenant", 1, "测试团队", "描述", 1)
	require.NoError(suite.T(), err)

	// 成功添加成员
	err = ts.AddTeamMember("test-tenant", team.ID, 2, 3, 1) // 用户2，角色3（member）
	require.NoError(suite.T(), err)

	// 验证成员已添加
	members, err := ts.GetTeamMembers("test-tenant", team.ID, MemberStatusActive)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), members, 2) // owner + 新成员

	// 查找新添加的成员
	var newMember *TeamMember
	for _, member := range members {
		if member.UserID == 2 {
			newMember = member
			break
		}
	}
	require.NotNil(suite.T(), newMember)
	assert.Equal(suite.T(), 3, newMember.RoleID)
	assert.Equal(suite.T(), MemberStatusActive, newMember.Status)

	// 重复添加同一用户
	err = ts.AddTeamMember("test-tenant", team.ID, 2, 3, 1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "用户已经是团队成员")

	// 添加不存在的用户
	err = ts.AddTeamMember("test-tenant", team.ID, 999, 3, 1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "用户不存在或状态无效")

	// 添加不存在的角色
	err = ts.AddTeamMember("test-tenant", team.ID, 3, 999, 1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "角色不存在")
}

// TestRemoveTeamMember 测试移除团队成员
func (suite *TeamServiceIntegrationTestSuite) TestRemoveTeamMember() {
	ts := NewTeamService(suite.db)

	// 创建测试团队并添加成员
	team, err := ts.CreateTeam("test-tenant", 1, "测试团队", "描述", 1)
	require.NoError(suite.T(), err)

	err = ts.AddTeamMember("test-tenant", team.ID, 2, 3, 1) // 添加普通成员
	require.NoError(suite.T(), err)

	// 成功移除成员
	err = ts.RemoveTeamMember("test-tenant", team.ID, 2, 1)
	require.NoError(suite.T(), err)

	// 验证成员已被移除（状态变为非活跃）
	members, err := ts.GetTeamMembers("test-tenant", team.ID, MemberStatusActive)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), members, 1) // 只剩owner

	// 尝试移除最后一个所有者
	err = ts.RemoveTeamMember("test-tenant", team.ID, 1, 1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "不能移除最后一个团队所有者")

	// 移除不存在的成员
	err = ts.RemoveTeamMember("test-tenant", team.ID, 999, 1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "成员不存在")
}

// TestChangeUserRole 测试更改用户角色
func (suite *TeamServiceIntegrationTestSuite) TestChangeUserRole() {
	ts := NewTeamService(suite.db)

	// 创建测试团队并添加成员
	team, err := ts.CreateTeam("test-tenant", 1, "测试团队", "描述", 1)
	require.NoError(suite.T(), err)

	err = ts.AddTeamMember("test-tenant", team.ID, 2, 4, 1) // 添加viewer角色成员
	require.NoError(suite.T(), err)

	// 成功更改角色
	err = ts.ChangeUserRole("test-tenant", team.ID, 2, 3, 1) // 从viewer改为member
	require.NoError(suite.T(), err)

	// 验证角色已更改
	members, err := ts.GetTeamMembers("test-tenant", team.ID, MemberStatusActive)
	require.NoError(suite.T(), err)

	var targetMember *TeamMember
	for _, member := range members {
		if member.UserID == 2 {
			targetMember = member
			break
		}
	}
	require.NotNil(suite.T(), targetMember)
	assert.Equal(suite.T(), 3, targetMember.RoleID) // 角色ID应该是3（member）

	// 更改为不存在的角色
	err = ts.ChangeUserRole("test-tenant", team.ID, 2, 999, 1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "角色不存在")

	// 更改不存在的成员
	err = ts.ChangeUserRole("test-tenant", team.ID, 999, 3, 1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "成员不存在")
}

// TestInviteUser 测试邀请用户
func (suite *TeamServiceIntegrationTestSuite) TestInviteUser() {
	ts := NewTeamService(suite.db)

	// 创建测试团队
	team, err := ts.CreateTeam("test-tenant", 1, "测试团队", "描述", 1)
	require.NoError(suite.T(), err)

	// 成功创建邀请
	invitation, err := ts.InviteUser("test-tenant", team.ID, 1, "newuser@test.com", 3, "欢迎加入我们的团队", 1)
	require.NoError(suite.T(), err)
	assert.NotZero(suite.T(), invitation.ID)
	assert.Equal(suite.T(), "newuser@test.com", invitation.Email)
	assert.Equal(suite.T(), 3, invitation.RoleID)
	assert.Equal(suite.T(), InvitationStatusPending, invitation.Status)
	assert.True(suite.T(), invitation.ExpiresAt.After(time.Now()))
	assert.NotEmpty(suite.T(), invitation.Token)

	// 重复邀请同一邮箱
	_, err = ts.InviteUser("test-tenant", team.ID, 1, "newuser@test.com", 3, "重复邀请", 1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "该邮箱已有待处理的邀请")

	// 无效邮箱格式
	_, err = ts.InviteUser("test-tenant", team.ID, 1, "invalid_email", 3, "消息", 1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "邮箱格式不正确")

	// 不存在的角色
	_, err = ts.InviteUser("test-tenant", team.ID, 1, "another@test.com", 999, "消息", 1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "角色不存在")
}

// TestAcceptInvitation 测试接受邀请
func (suite *TeamServiceIntegrationTestSuite) TestAcceptInvitation() {
	ts := NewTeamService(suite.db)

	// 创建测试团队和邀请
	team, err := ts.CreateTeam("test-tenant", 1, "测试团队", "描述", 1)
	require.NoError(suite.T(), err)

	invitation, err := ts.InviteUser("test-tenant", team.ID, 1, "member1@test.com", 3, "邀请消息", 1)
	require.NoError(suite.T(), err)

	// 成功接受邀请
	err = ts.AcceptInvitation(invitation.Token, 3) // 用户3的邮箱是member1@test.com
	require.NoError(suite.T(), err)

	// 验证用户已加入团队
	members, err := ts.GetTeamMembers("test-tenant", team.ID, MemberStatusActive)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), members, 2) // owner + 新成员

	// 验证邀请状态已更新
	var updatedInvitation TeamInvitation
	err = suite.db.First(&updatedInvitation, invitation.ID).Error
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), InvitationStatusAccepted, updatedInvitation.Status)
	assert.NotNil(suite.T(), updatedInvitation.AcceptedBy)
	assert.Equal(suite.T(), 3, *updatedInvitation.AcceptedBy)

	// 使用无效令牌
	err = ts.AcceptInvitation("invalid_token", 4)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "邀请不存在")

	// 邮箱不匹配
	invitation2, err := ts.InviteUser("test-tenant", team.ID, 1, "another@test.com", 3, "邀请消息", 1)
	require.NoError(suite.T(), err)

	err = ts.AcceptInvitation(invitation2.Token, 4) // 用户4的邮箱是member2@test.com，不匹配
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "邮箱不匹配")
}

// TestCreatePermissionRequest 测试创建权限申请
func (suite *TeamServiceIntegrationTestSuite) TestCreatePermissionRequest() {
	ts := NewTeamService(suite.db)

	// 成功创建权限申请
	request, err := ts.CreatePermissionRequest("test-tenant", 1, 3, RequestTypeRole, "admin", "需要管理权限", nil)
	require.NoError(suite.T(), err)
	assert.NotZero(suite.T(), request.ID)
	assert.Equal(suite.T(), 3, request.UserID)
	assert.Equal(suite.T(), RequestTypeRole, request.RequestType)
	assert.Equal(suite.T(), "admin", request.Permission)
	assert.Equal(suite.T(), "需要管理权限", request.Reason)
	assert.Equal(suite.T(), RequestStatusPending, request.Status)

	// 空申请理由
	_, err = ts.CreatePermissionRequest("test-tenant", 1, 3, RequestTypeRole, "admin", "", nil)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "申请理由不能为空")

	// 不存在的用户
	_, err = ts.CreatePermissionRequest("test-tenant", 1, 999, RequestTypeRole, "admin", "理由", nil)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "用户不存在或状态无效")
}

// TestReviewPermissionRequest 测试审批权限申请
func (suite *TeamServiceIntegrationTestSuite) TestReviewPermissionRequest() {
	ts := NewTeamService(suite.db)

	// 创建权限申请
	request, err := ts.CreatePermissionRequest("test-tenant", 1, 3, RequestTypeRole, "admin", "需要管理权限", nil)
	require.NoError(suite.T(), err)

	// 批准申请
	err = ts.ReviewPermissionRequest("test-tenant", request.ID, 1, true, "申请合理，批准")
	require.NoError(suite.T(), err)

	// 验证申请状态已更新
	var updatedRequest PermissionRequest
	err = suite.db.First(&updatedRequest, request.ID).Error
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), RequestStatusApproved, updatedRequest.Status)
	assert.NotNil(suite.T(), updatedRequest.ReviewedBy)
	assert.Equal(suite.T(), 1, *updatedRequest.ReviewedBy)
	assert.Equal(suite.T(), "申请合理，批准", updatedRequest.ReviewReason)

	// 尝试再次审批已处理的申请
	err = ts.ReviewPermissionRequest("test-tenant", request.ID, 1, false, "再次审批")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "权限申请已被处理")

	// 审批不存在的申请
	err = ts.ReviewPermissionRequest("test-tenant", 999, 1, true, "理由")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "权限申请不存在")
}

// TestGetTeamActivities 测试获取团队活动记录
func (suite *TeamServiceIntegrationTestSuite) TestGetTeamActivities() {
	ts := NewTeamService(suite.db)

	// 创建测试团队和一些活动
	team, err := ts.CreateTeam("test-tenant", 1, "测试团队", "描述", 1)
	require.NoError(suite.T(), err)

	err = ts.AddTeamMember("test-tenant", team.ID, 2, 3, 1)
	require.NoError(suite.T(), err)

	err = ts.AddTeamMember("test-tenant", team.ID, 3, 4, 1)
	require.NoError(suite.T(), err)

	// 获取活动记录
	activities, total, err := ts.GetTeamActivities("test-tenant", team.ID, 1, 10)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), total) // 创建团队 + 添加2个成员
	assert.Len(suite.T(), activities, 3)

	// 验证活动按时间倒序排列
	assert.True(suite.T(), activities[0].CreatedAt.After(activities[1].CreatedAt) || activities[0].CreatedAt.Equal(activities[1].CreatedAt))

	// 分页测试
	activities, total, err = ts.GetTeamActivities("test-tenant", team.ID, 1, 2)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), total)
	assert.Len(suite.T(), activities, 2)
}

// TestComplexTeamWorkflow 测试复杂的团队管理工作流
func (suite *TeamServiceIntegrationTestSuite) TestComplexTeamWorkflow() {
	ts := NewTeamService(suite.db)

	// 1. 创建团队
	team, err := ts.CreateTeam("test-tenant", 1, "产品开发团队", "负责产品功能开发", 1)
	require.NoError(suite.T(), err)

	// 2. 添加不同角色的成员
	err = ts.AddTeamMember("test-tenant", team.ID, 2, 2, 1) // admin
	require.NoError(suite.T(), err)

	err = ts.AddTeamMember("test-tenant", team.ID, 3, 3, 1) // member
	require.NoError(suite.T(), err)

	err = ts.AddTeamMember("test-tenant", team.ID, 4, 4, 1) // viewer
	require.NoError(suite.T(), err)

	// 3. 创建邀请
	invitation, err := ts.InviteUser("test-tenant", team.ID, 1, "newdev@company.com", 3, "欢迎加入开发团队", 1)
	require.NoError(suite.T(), err)

	// 4. 创建权限申请
	request, err := ts.CreatePermissionRequest("test-tenant", 1, 4, RequestTypeRole, "member", "希望获得更多权限参与开发", nil)
	require.NoError(suite.T(), err)

	// 5. 审批权限申请
	err = ts.ReviewPermissionRequest("test-tenant", request.ID, 1, true, "申请合理，批准提升权限")
	require.NoError(suite.T(), err)

	// 6. 更改用户角色
	err = ts.ChangeUserRole("test-tenant", team.ID, 4, 3, 1) // viewer -> member
	require.NoError(suite.T(), err)

	// 7. 验证最终状态
	members, err := ts.GetTeamMembers("test-tenant", team.ID, MemberStatusActive)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), members, 4) // owner + admin + 2个member

	// 验证角色分布
	roleCount := make(map[string]int)
	for _, member := range members {
		roleCount[member.Role.Name]++
	}
	assert.Equal(suite.T(), 1, roleCount[RoleOwner])
	assert.Equal(suite.T(), 1, roleCount[RoleAdmin])
	assert.Equal(suite.T(), 2, roleCount[RoleMember])
	assert.Equal(suite.T(), 0, roleCount[RoleViewer])

	// 8. 验证活动记录
	activities, total, err := ts.GetTeamActivities("test-tenant", team.ID, 1, 20)
	require.NoError(suite.T(), err)
	assert.Greater(suite.T(), total, int64(5)) // 应该有多个活动记录

	// 9. 验证邀请状态
	var storedInvitation TeamInvitation
	err = suite.db.First(&storedInvitation, invitation.ID).Error
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), InvitationStatusPending, storedInvitation.Status)

	// 10. 验证权限申请状态
	var storedRequest PermissionRequest
	err = suite.db.First(&storedRequest, request.ID).Error
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), RequestStatusApproved, storedRequest.Status)
}

// TestTenantIsolation 测试租户隔离
func (suite *TeamServiceIntegrationTestSuite) TestTenantIsolation() {
	ts := NewTeamService(suite.db)

	// 为不同租户创建团队
	team1, err := ts.CreateTeam("tenant-1", 1, "团队1", "租户1的团队", 1)
	require.NoError(suite.T(), err)

	team2, err := ts.CreateTeam("tenant-2", 1, "团队2", "租户2的团队", 1)
	require.NoError(suite.T(), err)

	// 尝试跨租户访问团队（应该失败）
	_, err = ts.GetTeam("tenant-1", team2.ID)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "团队不存在")

	_, err = ts.GetTeam("tenant-2", team1.ID)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "团队不存在")

	// 正确的租户访问应该成功
	retrievedTeam1, err := ts.GetTeam("tenant-1", team1.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), team1.ID, retrievedTeam1.ID)

	retrievedTeam2, err := ts.GetTeam("tenant-2", team2.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), team2.ID, retrievedTeam2.ID)
}

// TestConcurrentOperations 测试并发操作
func (suite *TeamServiceIntegrationTestSuite) TestConcurrentOperations() {
	ts := NewTeamService(suite.db)

	// 创建测试团队
	team, err := ts.CreateTeam("test-tenant", 1, "并发测试团队", "描述", 1)
	require.NoError(suite.T(), err)

	// 并发添加多个成员
	const numMembers = 10
	results := make(chan error, numMembers)

	for i := 2; i <= numMembers+1; i++ {
		go func(userID int) {
			// 先创建用户
			user := User{
				ID:          userID,
				TenantID:    "test-tenant",
				Username:    fmt.Sprintf("concurrent_user_%d", userID),
				Email:       fmt.Sprintf("user%d@test.com", userID),
				DisplayName: fmt.Sprintf("User %d", userID),
				Status:      UserStatusActive,
			}
			suite.db.Create(&user)

			// 添加到团队
			err := ts.AddTeamMember("test-tenant", team.ID, userID, 3, 1)
			results <- err
		}(i)
	}

	// 等待所有操作完成
	successCount := 0
	for i := 0; i < numMembers; i++ {
		err := <-results
		if err == nil {
			successCount++
		}
	}

	// 验证成员数量
	members, err := ts.GetTeamMembers("test-tenant", team.ID, MemberStatusActive)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), successCount+1, len(members)) // +1 for owner
}

// TestEdgeCases 测试边界情况
func (suite *TeamServiceIntegrationTestSuite) TestEdgeCases() {
	ts := NewTeamService(suite.db)

	// 测试极长团队名称边界
	longName99 := strings.Repeat("a", 99)
	_, err := ts.CreateTeam("test-tenant", 1, longName99, "描述", 1)
	assert.NoError(suite.T(), err)

	longName100 := strings.Repeat("b", 100)
	_, err = ts.CreateTeam("test-tenant", 1, longName100, "描述", 2)
	assert.NoError(suite.T(), err)

	longName101 := strings.Repeat("c", 101)
	_, err = ts.CreateTeam("test-tenant", 1, longName101, "描述", 3)
	assert.Error(suite.T(), err)

	// 测试特殊字符团队名称
	validNames := []string{"开发Team", "My Team", "Dev_Team", "Dev-Team", "Team123", "开发团队"}
	for i, name := range validNames {
		_, err := ts.CreateTeam("test-tenant", 1, name, "描述", i+1)
		assert.NoError(suite.T(), err, "Valid name should pass: %s", name)
	}

	invalidNames := []string{"Team@", "Team#", "Team$", "Team%"}
	for i, name := range invalidNames {
		_, err := ts.CreateTeam("test-tenant", 1, name, "描述", i+10)
		assert.Error(suite.T(), err, "Invalid name should fail: %s", name)
	}

	// 测试邀请过期时间
	team, err := ts.CreateTeam("test-tenant", 1, "过期测试团队", "描述", 1)
	require.NoError(suite.T(), err)

	invitation, err := ts.InviteUser("test-tenant", team.ID, 1, "expire@test.com", 3, "测试过期", 1)
	require.NoError(suite.T(), err)

	// 验证过期时间设置正确（7天后）
	expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
	assert.True(suite.T(), invitation.ExpiresAt.Sub(expectedExpiry) < time.Minute)
	assert.True(suite.T(), invitation.ExpiresAt.After(time.Now()))
}

// TestPerformance 测试性能要求
func (suite *TeamServiceIntegrationTestSuite) TestPerformance() {
	ts := NewTeamService(suite.db)

	// 测试团队名称验证性能
	teamName := "开发团队_Development_Team_2024"
	start := time.Now()
	for i := 0; i < 1000; i++ {
		suite.validateTeamName(teamName)
	}
	duration := time.Since(start)
	assert.Less(suite.T(), duration, 10*time.Millisecond, "团队名称验证性能不达标")

	// 测试邮箱验证性能
	email := "user@company.example.com"
	start = time.Now()
	for i := 0; i < 1000; i++ {
		isValidEmail(email)
	}
	duration = time.Since(start)
	assert.Less(suite.T(), duration, 10*time.Millisecond, "邮箱验证性能不达标")

	// 测试批量添加成员性能
	team, err := ts.CreateTeam("test-tenant", 1, "性能测试团队", "描述", 1)
	require.NoError(suite.T(), err)

	// 预创建用户
	for i := 2; i <= 101; i++ {
		user := User{
			ID:          i,
			TenantID:    "test-tenant",
			Username:    fmt.Sprintf("perf_user_%d", i),
			Email:       fmt.Sprintf("perf%d@test.com", i),
			DisplayName: fmt.Sprintf("Performance User %d", i),
			Status:      UserStatusActive,
		}
		suite.db.Create(&user)
	}

	start = time.Now()
	for i := 2; i <= 101; i++ {
		ts.AddTeamMember("test-tenant", team.ID, i, 3, 1)
	}
	duration = time.Since(start)
	assert.Less(suite.T(), duration, 500*time.Millisecond, "批量添加成员性能不达标")
}

// 运行测试套件
func TestTeamServiceIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(TeamServiceIntegrationTestSuite))
}

// 性能基准测试
func BenchmarkCreateTeam(b *testing.B) {
	// 设置测试环境
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&User{}, &Team{}, &TeamMember{}, &Role{}, &TeamInvitation{}, &PermissionRequest{}, &TeamActivity{})

	// 创建测试用户和角色
	user := User{ID: 1, TenantID: "bench-tenant", Username: "bench_owner", Email: "owner@bench.com", Status: UserStatusActive}
	db.Create(&user)

	role := Role{ID: 1, TenantID: "bench-tenant", ProjectID: 1, Name: RoleOwner, Permissions: GetDefaultRolePermissions(RoleOwner), IsSystem: true, CreatedBy: 1}
	db.Create(&role)

	ts := NewTeamService(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts.CreateTeam("bench-tenant", 1, fmt.Sprintf("团队_%d", i), "基准测试团队", 1)
	}
}

func BenchmarkAddTeamMember(b *testing.B) {
	// 设置测试环境
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&User{}, &Team{}, &TeamMember{}, &Role{}, &TeamInvitation{}, &PermissionRequest{}, &TeamActivity{})

	// 创建测试数据
	owner := User{ID: 1, TenantID: "bench-tenant", Username: "bench_owner", Email: "owner@bench.com", Status: UserStatusActive}
	db.Create(&owner)

	ownerRole := Role{ID: 1, TenantID: "bench-tenant", ProjectID: 1, Name: RoleOwner, Permissions: GetDefaultRolePermissions(RoleOwner), IsSystem: true, CreatedBy: 1}
	memberRole := Role{ID: 2, TenantID: "bench-tenant", ProjectID: 1, Name: RoleMember, Permissions: GetDefaultRolePermissions(RoleMember), IsSystem: true, CreatedBy: 1}
	db.Create(&ownerRole)
	db.Create(&memberRole)

	ts := NewTeamService(db)
	team, _ := ts.CreateTeam("bench-tenant", 1, "基准测试团队", "描述", 1)

	// 预创建用户
	for i := 2; i <= b.N+1; i++ {
		user := User{
			ID:       i,
			TenantID: "bench-tenant",
			Username: fmt.Sprintf("bench_user_%d", i),
			Email:    fmt.Sprintf("user%d@bench.com", i),
			Status:   UserStatusActive,
		}
		db.Create(&user)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts.AddTeamMember("bench-tenant", team.ID, i+2, 2, 1)
	}
}
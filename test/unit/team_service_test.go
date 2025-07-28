package unit

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Team 团队结构
type Team struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	OwnerID     string            `json:"owner_id"`
	Members     map[string]Member `json:"members"`
	Settings    TeamSettings      `json:"settings"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Member 团队成员结构
type Member struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
	JoinedAt    time.Time `json:"joined_at"`
	Status      string    `json:"status"`
}

// TeamSettings 团队设置
type TeamSettings struct {
	IsPublic      bool     `json:"is_public"`
	AllowInvites  bool     `json:"allow_invites"`
	MaxMembers    int      `json:"max_members"`
	RequiredRoles []string `json:"required_roles"`
}

// Invitation 邀请结构
type Invitation struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	InviterID string    `json:"inviter_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// PermissionRequest 权限请求结构
type PermissionRequest struct {
	ID          string     `json:"id"`
	TeamID      string     `json:"team_id"`
	UserID      string     `json:"user_id"`
	RequestedBy string     `json:"requested_by"`
	Permission  string     `json:"permission"`
	Reason      string     `json:"reason"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy  string     `json:"reviewed_by,omitempty"`
}

// MockTeamService 模拟团队服务
type MockTeamService struct {
	teams              map[string]*Team
	invitations        map[string]*Invitation
	permissionRequests map[string]*PermissionRequest
}

// NewMockTeamService 创建模拟团队服务
func NewMockTeamService() *MockTeamService {
	return &MockTeamService{
		teams:              make(map[string]*Team),
		invitations:        make(map[string]*Invitation),
		permissionRequests: make(map[string]*PermissionRequest),
	}
}

// CreateTeam 创建团队
func (s *MockTeamService) CreateTeam(name, description, ownerID string) (*Team, error) {
	if err := validateTeamName(name); err != nil {
		return nil, err
	}
	if err := validateUserID(ownerID); err != nil {
		return nil, err
	}

	teamID := fmt.Sprintf("team_%d", time.Now().UnixNano())
	team := &Team{
		ID:          teamID,
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
		Members:     make(map[string]Member),
		Settings: TeamSettings{
			IsPublic:      false,
			AllowInvites:  true,
			MaxMembers:    50,
			RequiredRoles: []string{"member"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 添加创建者作为owner
	team.Members[ownerID] = Member{
		UserID:      ownerID,
		Username:    fmt.Sprintf("user_%s", ownerID),
		Email:       fmt.Sprintf("user_%s@example.com", ownerID),
		Role:        "owner",
		Permissions: []string{"all"},
		JoinedAt:    time.Now(),
		Status:      "active",
	}

	s.teams[teamID] = team
	return team, nil
}

// AddMember 添加团队成员
func (s *MockTeamService) AddMember(teamID, userID, role string) error {
	team, exists := s.teams[teamID]
	if !exists {
		return fmt.Errorf("团队不存在")
	}

	if _, exists := team.Members[userID]; exists {
		return fmt.Errorf("用户已是团队成员")
	}

	if err := validateRole(role); err != nil {
		return err
	}

	if len(team.Members) >= team.Settings.MaxMembers {
		return fmt.Errorf("团队成员已达上限")
	}

	team.Members[userID] = Member{
		UserID:      userID,
		Username:    fmt.Sprintf("user_%s", userID),
		Email:       fmt.Sprintf("user_%s@example.com", userID),
		Role:        role,
		Permissions: getRolePermissions(role),
		JoinedAt:    time.Now(),
		Status:      "active",
	}

	team.UpdatedAt = time.Now()
	return nil
}

// CreateInvitation 创建邀请
func (s *MockTeamService) CreateInvitation(teamID, inviterID, email, role string) (*Invitation, error) {
	team, exists := s.teams[teamID]
	if !exists {
		return nil, fmt.Errorf("团队不存在")
	}

	if !team.Settings.AllowInvites {
		return nil, fmt.Errorf("团队不允许邀请")
	}

	if err := teamValidateEmail(email); err != nil {
		return nil, err
	}

	if err := validateRole(role); err != nil {
		return nil, err
	}

	invitationID := fmt.Sprintf("inv_%d", time.Now().UnixNano())
	invitation := &Invitation{
		ID:        invitationID,
		TeamID:    teamID,
		InviterID: inviterID,
		Email:     email,
		Role:      role,
		Status:    "pending",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7天后过期
		CreatedAt: time.Now(),
	}

	s.invitations[invitationID] = invitation
	return invitation, nil
}

// RequestPermission 请求权限
func (s *MockTeamService) RequestPermission(teamID, userID, requestedBy, permission, reason string) (*PermissionRequest, error) {
	if _, exists := s.teams[teamID]; !exists {
		return nil, fmt.Errorf("团队不存在")
	}

	if err := teamValidatePermission(permission); err != nil {
		return nil, err
	}

	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	request := &PermissionRequest{
		ID:          requestID,
		TeamID:      teamID,
		UserID:      userID,
		RequestedBy: requestedBy,
		Permission:  permission,
		Reason:      reason,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	s.permissionRequests[requestID] = request
	return request, nil
}

// ================================
// 验证函数
// ================================

// validateTeamName 验证团队名称
func validateTeamName(name string) error {
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

// validateUserID 验证用户ID
func validateUserID(userID string) error {
	if userID == "" {
		return fmt.Errorf("用户ID不能为空")
	}
	if len(userID) < 1 || len(userID) > 50 {
		return fmt.Errorf("用户ID长度无效")
	}
	return nil
}

// validateRole 验证角色
func validateRole(role string) error {
	validRoles := []string{"owner", "admin", "developer", "viewer", "member"}
	for _, validRole := range validRoles {
		if role == validRole {
			return nil
		}
	}
	return fmt.Errorf("无效的角色: %s", role)
}

// validateEmail 验证邮箱
func teamValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}
	emailPattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailPattern.MatchString(email) {
		return fmt.Errorf("邮箱格式无效")
	}
	return nil
}

// validatePermission 验证权限
func teamValidatePermission(permission string) error {
	validPermissions := []string{"read", "write", "delete", "admin", "invite", "manage_members"}
	for _, validPerm := range validPermissions {
		if permission == validPerm {
			return nil
		}
	}
	return fmt.Errorf("无效的权限: %s", permission)
}

// getRolePermissions 获取角色权限
func getRolePermissions(role string) []string {
	switch role {
	case "owner":
		return []string{"all"}
	case "admin":
		return []string{"read", "write", "delete", "invite", "manage_members"}
	case "developer":
		return []string{"read", "write"}
	case "viewer":
		return []string{"read"}
	case "member":
		return []string{"read", "write"}
	default:
		return []string{"read"}
	}
}

// ================================
// 单元测试 - 验证函数测试
// ================================

func TestValidateTeamName(t *testing.T) {
	testCases := []struct {
		name        string
		teamName    string
		expectError bool
		errorMsg    string
	}{
		{"有效团队名称-英文", "Development Team", false, ""},
		{"有效团队名称-中文", "开发团队", false, ""},
		{"有效团队名称-混合", "开发Team_001", false, ""},
		{"空团队名称", "", true, "团队名称不能为空"},
		{"团队名称过长", strings.Repeat("a", 101), true, "团队名称不能超过100个字符"},
		{"无效字符", "Team@#$%", true, "团队名称格式无效"},
		{"边界值-最大长度", strings.Repeat("a", 100), false, ""},
		{"包含下划线", "Dev_Team", false, ""},
		{"包含连字符", "Dev-Team", false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTeamName(tc.teamName)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUserID(t *testing.T) {
	testCases := []struct {
		name        string
		userID      string
		expectError bool
		errorMsg    string
	}{
		{"有效用户ID", "user123", false, ""},
		{"空用户ID", "", true, "用户ID不能为空"},
		{"用户ID过长", strings.Repeat("a", 51), true, "用户ID长度无效"},
		{"边界值-最大长度", strings.Repeat("a", 50), false, ""},
		{"边界值-最小长度", "a", false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUserID(tc.userID)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRole(t *testing.T) {
	testCases := []struct {
		name        string
		role        string
		expectError bool
	}{
		{"有效角色-owner", "owner", false},
		{"有效角色-admin", "admin", false},
		{"有效角色-developer", "developer", false},
		{"有效角色-viewer", "viewer", false},
		{"有效角色-member", "member", false},
		{"无效角色", "invalid_role", true},
		{"空角色", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRole(tc.role)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	testCases := []struct {
		name        string
		email       string
		expectError bool
	}{
		{"有效邮箱", "user@example.com", false},
		{"有效邮箱-子域名", "user@sub.example.com", false},
		{"有效邮箱-数字域名", "user@example123.com", false},
		{"无效邮箱-缺少@", "userexample.com", true},
		{"无效邮箱-缺少域名", "user@", true},
		{"无效邮箱-缺少后缀", "user@example", true},
		{"空邮箱", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := teamValidateEmail(tc.email)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePermission(t *testing.T) {
	testCases := []struct {
		name        string
		permission  string
		expectError bool
	}{
		{"有效权限-read", "read", false},
		{"有效权限-write", "write", false},
		{"有效权限-delete", "delete", false},
		{"有效权限-admin", "admin", false},
		{"有效权限-invite", "invite", false},
		{"有效权限-manage_members", "manage_members", false},
		{"无效权限", "invalid_permission", true},
		{"空权限", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := teamValidatePermission(tc.permission)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetRolePermissions(t *testing.T) {
	testCases := []struct {
		role                string
		expectedPermissions []string
	}{
		{"owner", []string{"all"}},
		{"admin", []string{"read", "write", "delete", "invite", "manage_members"}},
		{"developer", []string{"read", "write"}},
		{"viewer", []string{"read"}},
		{"member", []string{"read", "write"}},
		{"unknown", []string{"read"}},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("角色_%s", tc.role), func(t *testing.T) {
			permissions := getRolePermissions(tc.role)
			assert.Equal(t, tc.expectedPermissions, permissions)
		})
	}
}

// ================================
// 单元测试 - 团队服务测试
// ================================

func TestCreateTeam(t *testing.T) {
	service := NewMockTeamService()

	t.Run("成功创建团队", func(t *testing.T) {
		team, err := service.CreateTeam("开发团队", "前端开发团队", "user123")

		require.NoError(t, err)
		assert.NotEmpty(t, team.ID)
		assert.Equal(t, "开发团队", team.Name)
		assert.Equal(t, "前端开发团队", team.Description)
		assert.Equal(t, "user123", team.OwnerID)
		assert.Len(t, team.Members, 1)

		// 验证创建者成为owner
		owner, exists := team.Members["user123"]
		assert.True(t, exists)
		assert.Equal(t, "owner", owner.Role)
		assert.Equal(t, []string{"all"}, owner.Permissions)
		assert.Equal(t, "active", owner.Status)
	})

	t.Run("无效团队名称", func(t *testing.T) {
		_, err := service.CreateTeam("", "描述", "user123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "团队名称不能为空")
	})

	t.Run("无效用户ID", func(t *testing.T) {
		_, err := service.CreateTeam("团队名称", "描述", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "用户ID不能为空")
	})
}

func TestAddMember(t *testing.T) {
	service := NewMockTeamService()
	team, _ := service.CreateTeam("测试团队", "描述", "owner123")

	t.Run("成功添加成员", func(t *testing.T) {
		err := service.AddMember(team.ID, "user456", "developer")

		require.NoError(t, err)
		assert.Len(t, team.Members, 2)

		member, exists := team.Members["user456"]
		assert.True(t, exists)
		assert.Equal(t, "developer", member.Role)
		assert.Equal(t, []string{"read", "write"}, member.Permissions)
		assert.Equal(t, "active", member.Status)
	})

	t.Run("团队不存在", func(t *testing.T) {
		err := service.AddMember("invalid_team", "user789", "member")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "团队不存在")
	})

	t.Run("用户已是成员", func(t *testing.T) {
		err := service.AddMember(team.ID, "user456", "admin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "用户已是团队成员")
	})

	t.Run("无效角色", func(t *testing.T) {
		err := service.AddMember(team.ID, "user789", "invalid_role")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无效的角色")
	})

	t.Run("团队成员达到上限", func(t *testing.T) {
		// 修改团队设置为最大2个成员
		team.Settings.MaxMembers = 2

		err := service.AddMember(team.ID, "user789", "member")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "团队成员已达上限")
	})
}

func TestCreateInvitation(t *testing.T) {
	service := NewMockTeamService()
	team, _ := service.CreateTeam("测试团队", "描述", "owner123")

	t.Run("成功创建邀请", func(t *testing.T) {
		invitation, err := service.CreateInvitation(team.ID, "owner123", "newuser@example.com", "developer")

		require.NoError(t, err)
		assert.NotEmpty(t, invitation.ID)
		assert.Equal(t, team.ID, invitation.TeamID)
		assert.Equal(t, "owner123", invitation.InviterID)
		assert.Equal(t, "newuser@example.com", invitation.Email)
		assert.Equal(t, "developer", invitation.Role)
		assert.Equal(t, "pending", invitation.Status)
		assert.True(t, invitation.ExpiresAt.After(time.Now()))
	})

	t.Run("团队不存在", func(t *testing.T) {
		_, err := service.CreateInvitation("invalid_team", "owner123", "user@example.com", "member")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "团队不存在")
	})

	t.Run("团队不允许邀请", func(t *testing.T) {
		team.Settings.AllowInvites = false

		_, err := service.CreateInvitation(team.ID, "owner123", "user@example.com", "member")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "团队不允许邀请")

		// 恢复设置
		team.Settings.AllowInvites = true
	})

	t.Run("无效邮箱", func(t *testing.T) {
		_, err := service.CreateInvitation(team.ID, "owner123", "invalid_email", "member")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "邮箱格式无效")
	})

	t.Run("无效角色", func(t *testing.T) {
		_, err := service.CreateInvitation(team.ID, "owner123", "user@example.com", "invalid_role")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无效的角色")
	})
}

func TestRequestPermission(t *testing.T) {
	service := NewMockTeamService()
	team, _ := service.CreateTeam("测试团队", "描述", "owner123")
	service.AddMember(team.ID, "user456", "viewer")

	t.Run("成功请求权限", func(t *testing.T) {
		request, err := service.RequestPermission(team.ID, "user456", "user456", "write", "需要编辑权限")

		require.NoError(t, err)
		assert.NotEmpty(t, request.ID)
		assert.Equal(t, team.ID, request.TeamID)
		assert.Equal(t, "user456", request.UserID)
		assert.Equal(t, "user456", request.RequestedBy)
		assert.Equal(t, "write", request.Permission)
		assert.Equal(t, "需要编辑权限", request.Reason)
		assert.Equal(t, "pending", request.Status)
		assert.Nil(t, request.ReviewedAt)
	})

	t.Run("团队不存在", func(t *testing.T) {
		_, err := service.RequestPermission("invalid_team", "user456", "user456", "write", "原因")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "团队不存在")
	})

	t.Run("无效权限", func(t *testing.T) {
		_, err := service.RequestPermission(team.ID, "user456", "user456", "invalid_permission", "原因")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无效的权限")
	})
}

// ================================
// 性能测试
// ================================

func BenchmarkValidateTeamName(b *testing.B) {
	teamName := "开发团队_Development_Team"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		validateTeamName(teamName)
	}
}

func BenchmarkValidateEmail(b *testing.B) {
	email := "user@example.com"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		teamValidateEmail(email)
	}
}

func BenchmarkCreateTeam(b *testing.B) {
	service := NewMockTeamService()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		service.CreateTeam(fmt.Sprintf("团队_%d", i), "描述", fmt.Sprintf("user_%d", i))
	}
}

func BenchmarkAddMember(b *testing.B) {
	service := NewMockTeamService()
	team, _ := service.CreateTeam("基准测试团队", "描述", "owner123")
	team.Settings.MaxMembers = 10000 // 增加上限避免测试失败

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		service.AddMember(team.ID, fmt.Sprintf("user_%d", i), "member")
	}
}

// ================================
// 边缘情况和错误场景测试
// ================================

func TestTeamEdgeCases(t *testing.T) {
	service := NewMockTeamService()

	t.Run("极长团队名称边界测试", func(t *testing.T) {
		// 99字符 - 应该成功
		longName99 := strings.Repeat("a", 99)
		_, err := service.CreateTeam(longName99, "描述", "user123")
		assert.NoError(t, err)

		// 100字符 - 应该成功
		longName100 := strings.Repeat("b", 100)
		_, err = service.CreateTeam(longName100, "描述", "user456")
		assert.NoError(t, err)

		// 101字符 - 应该失败
		longName101 := strings.Repeat("c", 101)
		_, err = service.CreateTeam(longName101, "描述", "user789")
		assert.Error(t, err)
	})

	t.Run("特殊字符团队名称测试", func(t *testing.T) {
		testCases := []struct {
			name        string
			teamName    string
			expectError bool
		}{
			{"中英混合", "开发Team", false},
			{"带空格", "My Team", false},
			{"带下划线", "Dev_Team", false},
			{"带连字符", "Dev-Team", false},
			{"带数字", "Team123", false},
			{"纯中文", "开发团队", false},
			{"特殊字符@", "Team@", true},
			{"特殊字符#", "Team#", true},
			{"特殊字符$", "Team$", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := service.CreateTeam(tc.teamName, "描述", fmt.Sprintf("user_%s", tc.name))
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("邀请过期时间测试", func(t *testing.T) {
		team, _ := service.CreateTeam("测试团队", "描述", "owner123")
		invitation, err := service.CreateInvitation(team.ID, "owner123", "test@example.com", "member")

		require.NoError(t, err)

		// 验证过期时间设置正确（7天后）
		expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
		assert.True(t, invitation.ExpiresAt.Sub(expectedExpiry) < time.Minute)
		assert.True(t, invitation.ExpiresAt.After(time.Now()))
	})

	t.Run("并发添加成员测试", func(t *testing.T) {
		team, _ := service.CreateTeam("并发测试团队", "描述", "owner123")
		team.Settings.MaxMembers = 10

		// 模拟并发添加成员，测试是否会超过限制
		for i := 0; i < 15; i++ {
			err := service.AddMember(team.ID, fmt.Sprintf("concurrent_user_%d", i), "member")
			if i < 9 { // 第一个是owner，所以还能添加9个
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "团队成员已达上限")
			}
		}
	})
}

// ================================
// 集成测试场景
// ================================

func TestTeamManagementWorkflow(t *testing.T) {
	service := NewMockTeamService()

	t.Run("完整团队管理流程", func(t *testing.T) {
		// 1. 创建团队
		team, err := service.CreateTeam("产品开发团队", "负责产品功能开发", "product_owner")
		require.NoError(t, err)

		// 2. 添加不同角色的成员
		err = service.AddMember(team.ID, "tech_lead", "admin")
		require.NoError(t, err)

		err = service.AddMember(team.ID, "developer1", "developer")
		require.NoError(t, err)

		err = service.AddMember(team.ID, "developer2", "developer")
		require.NoError(t, err)

		err = service.AddMember(team.ID, "designer", "member")
		require.NoError(t, err)

		// 3. 创建邀请
		invitation, err := service.CreateInvitation(team.ID, "product_owner", "newdev@company.com", "developer")
		require.NoError(t, err)
		assert.Equal(t, "pending", invitation.Status)

		// 4. 请求权限
		request, err := service.RequestPermission(team.ID, "designer", "designer", "admin", "需要管理权限来处理设计评审")
		require.NoError(t, err)
		assert.Equal(t, "pending", request.Status)

		// 5. 验证最终状态
		assert.Len(t, team.Members, 5) // owner + 4个成员
		assert.Len(t, service.invitations, 1)
		assert.Len(t, service.permissionRequests, 1)

		// 6. 验证不同角色的权限
		owner := team.Members["product_owner"]
		assert.Equal(t, []string{"all"}, owner.Permissions)

		admin := team.Members["tech_lead"]
		assert.Contains(t, admin.Permissions, "manage_members")

		developer := team.Members["developer1"]
		assert.Contains(t, developer.Permissions, "write")
		assert.NotContains(t, developer.Permissions, "delete")
	})
}

// ================================
// 性能基准测试
// ================================

func TestTeamPerformanceRequirements(t *testing.T) {
	service := NewMockTeamService()

	t.Run("团队名称验证性能", func(t *testing.T) {
		teamName := "开发团队_Development_Team_2024"

		start := time.Now()
		for i := 0; i < 1000; i++ {
			validateTeamName(teamName)
		}
		duration := time.Since(start)

		// 1000次验证应该在10ms内完成
		assert.Less(t, duration, 10*time.Millisecond, "团队名称验证性能不达标")
	})

	t.Run("邮箱验证性能", func(t *testing.T) {
		email := "user@company.example.com"

		start := time.Now()
		for i := 0; i < 1000; i++ {
			teamValidateEmail(email)
		}
		duration := time.Since(start)

		// 1000次验证应该在10ms内完成
		assert.Less(t, duration, 10*time.Millisecond, "邮箱验证性能不达标")
	})

	t.Run("大量成员添加性能", func(t *testing.T) {
		team, _ := service.CreateTeam("性能测试团队", "描述", "owner123")
		team.Settings.MaxMembers = 1000

		start := time.Now()
		for i := 0; i < 100; i++ {
			service.AddMember(team.ID, fmt.Sprintf("perf_user_%d", i), "member")
		}
		duration := time.Since(start)

		// 添加100个成员应该在100ms内完成
		assert.Less(t, duration, 100*time.Millisecond, "批量添加成员性能不达标")
	})
}

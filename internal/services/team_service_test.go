package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/internal/models"
)

// TeamServiceTestSuite 团队服务测试套件
type TeamServiceTestSuite struct {
	suite.Suite
	db          *gorm.DB
	teamService *TeamService
}

// SetupSuite 设置测试套件
func (suite *TeamServiceTestSuite) SetupSuite() {
	// 使用内存SQLite数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	// 自动迁移数据表
	err = db.AutoMigrate(
		&models.Team{},
		&models.TeamMember{},
		&models.TeamInvitation{},
		&models.PermissionRequest{},
		&models.User{},
		&models.Role{},
		&models.TeamActivity{},
	)
	suite.Require().NoError(err)

	suite.db = db
	permissionService := NewPermissionService(db)
	suite.teamService = NewTeamService(db, permissionService)

	// 初始化测试数据
	suite.seedTestData()
}

// TearDownSuite 清理测试套件
func (suite *TeamServiceTestSuite) TearDownSuite() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

// seedTestData 初始化测试数据
func (suite *TeamServiceTestSuite) seedTestData() {
	// 创建测试用户
	users := []models.User{
		{
			ID: 1, Username: "owner", Email: "owner@test.com", 
			DisplayName: "团队所有者", Status: models.UserStatusActive,
		},
		{
			ID: 2, Username: "admin", Email: "admin@test.com", 
			DisplayName: "团队管理员", Status: models.UserStatusActive,
		},
		{
			ID: 3, Username: "member", Email: "member@test.com", 
			DisplayName: "团队成员", Status: models.UserStatusActive,
		},
		{
			ID: 4, Username: "viewer", Email: "viewer@test.com", 
			DisplayName: "团队观察者", Status: models.UserStatusActive,
		},
		{
			ID: 5, Username: "inactive", Email: "inactive@test.com", 
			DisplayName: "非活跃用户", Status: models.UserStatusInactive,
		},
	}

	for _, user := range users {
		suite.db.Create(&user)
	}

	// 创建测试角色
	roles := []models.Role{
		{
			ID: 1, TenantID: "default", ProjectID: 1, 
			Name: models.RoleOwner, Description: "团队所有者",
			Permissions: models.GetDefaultRolePermissions(models.RoleOwner),
			IsSystem: true,
		},
		{
			ID: 2, TenantID: "default", ProjectID: 1, 
			Name: models.RoleAdmin, Description: "团队管理员",
			Permissions: models.GetDefaultRolePermissions(models.RoleAdmin),
			IsSystem: true,
		},
		{
			ID: 3, TenantID: "default", ProjectID: 1, 
			Name: models.RoleMember, Description: "团队成员",
			Permissions: models.GetDefaultRolePermissions(models.RoleMember),
			IsSystem: true,
		},
		{
			ID: 4, TenantID: "default", ProjectID: 1, 
			Name: models.RoleViewer, Description: "团队观察者",
			Permissions: models.GetDefaultRolePermissions(models.RoleViewer),
			IsSystem: true,
		},
	}

	for _, role := range roles {
		suite.db.Create(&role)
	}
}

// TestCreateTeam 测试创建团队
func (suite *TeamServiceTestSuite) TestCreateTeam() {
	testCases := []struct {
		name        string
		tenantID    string
		projectID   int
		teamName    string
		description string
		createdBy   int
		expectError bool
	}{
		{
			name: "成功创建团队",
			tenantID: "default", projectID: 1,
			teamName: "开发团队", description: "负责产品开发",
			createdBy: 1, expectError: false,
		},
		{
			name: "团队名称为空",
			tenantID: "default", projectID: 1,
			teamName: "", description: "描述",
			createdBy: 1, expectError: true,
		},
		{
			name: "创建者不存在",
			tenantID: "default", projectID: 1,
			teamName: "测试团队", description: "描述",
			createdBy: 999, expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			team, err := suite.teamService.CreateTeam(
				tc.tenantID, tc.projectID, tc.teamName, tc.description, tc.createdBy,
			)

			if tc.expectError {
				suite.Error(err)
				suite.Nil(team)
			} else {
				suite.NoError(err)
				suite.NotNil(team)
				suite.Equal(tc.teamName, team.Name)
				suite.Equal(tc.description, team.Description)
				suite.Equal(tc.createdBy, team.CreatedBy)
				suite.True(team.IsActive)

				// 验证创建者自动成为所有者
				var member models.TeamMember
				err = suite.db.Where("team_id = ? AND user_id = ?", team.ID, tc.createdBy).First(&member).Error
				suite.NoError(err)
				suite.Equal(1, member.RoleID) // 所有者角色ID
				suite.Equal(models.MemberStatusActive, member.Status)
			}
		})
	}
}

// TestAddTeamMember 测试添加团队成员
func (suite *TeamServiceTestSuite) TestAddTeamMember() {
	// 先创建一个测试团队
	team, err := suite.teamService.CreateTeam("default", 1, "测试团队", "测试描述", 1)
	suite.Require().NoError(err)

	testCases := []struct {
		name        string
		teamID      int
		userID      int
		roleID      int
		invitedBy   int
		expectError bool
	}{
		{
			name: "成功添加成员",
			teamID: team.ID, userID: 2, roleID: 2, invitedBy: 1,
			expectError: false,
		},
		{
			name: "添加不存在的用户",
			teamID: team.ID, userID: 999, roleID: 3, invitedBy: 1,
			expectError: true,
		},
		{
			name: "重复添加成员",
			teamID: team.ID, userID: 2, roleID: 3, invitedBy: 1,
			expectError: true,
		},
		{
			name: "添加非活跃用户",
			teamID: team.ID, userID: 5, roleID: 4, invitedBy: 1,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			member, err := suite.teamService.AddTeamMemberCompat(
				tc.teamID, tc.userID, tc.roleID, tc.invitedBy,
			)

			if tc.expectError {
				suite.Error(err)
				suite.Nil(member)
			} else {
				suite.NoError(err)
				suite.NotNil(member)
				suite.Equal(tc.teamID, member.TeamID)
				suite.Equal(tc.userID, member.UserID)
				suite.Equal(tc.roleID, member.RoleID)
				suite.Equal(models.MemberStatusActive, member.Status)
			}
		})
	}
}

// TestUpdateMemberRole 测试更新成员角色
func (suite *TeamServiceTestSuite) TestUpdateMemberRole() {
	testCases := []struct {
		name        string
		setupUserID int
		setupRoleID int
		updateUserID int
		newRoleID   int
		expectError bool
	}{
		{
			name: "成功更新角色",
			setupUserID: 2, setupRoleID: 3, updateUserID: 2, newRoleID: 2,
			expectError: false,
		},
		{
			name: "更新不存在的成员",
			setupUserID: 2, setupRoleID: 3, updateUserID: 999, newRoleID: 3,
			expectError: true,
		},
		{
			name: "角色ID不存在",
			setupUserID: 3, setupRoleID: 3, updateUserID: 3, newRoleID: 999,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 为每个测试用例创建独立的团队和成员
			team, err := suite.teamService.CreateTeam("default", 1, "测试团队", "描述", 1)
			suite.Require().NoError(err)
			
			_, err = suite.teamService.AddTeamMemberCompat(team.ID, tc.setupUserID, tc.setupRoleID, 1)
			suite.Require().NoError(err)
			
			err = suite.teamService.UpdateMemberRoleCompat(team.ID, tc.updateUserID, tc.newRoleID)

			if tc.expectError {
				suite.Error(err)
			} else {
				suite.NoError(err)
				
				// 验证角色已更新
				var updatedMember models.TeamMember
				err = suite.db.Where("team_id = ? AND user_id = ?", team.ID, tc.updateUserID).First(&updatedMember).Error
				suite.NoError(err)
				suite.Equal(tc.newRoleID, updatedMember.RoleID)
			}
		})
	}
}

// TestRemoveTeamMember 测试移除团队成员
func (suite *TeamServiceTestSuite) TestRemoveTeamMember() {
	// 创建测试团队和成员
	team, err := suite.teamService.CreateTeam("default", 1, "测试团队", "描述", 1)
	suite.Require().NoError(err)
	
	_, err = suite.teamService.AddTeamMemberCompat(team.ID, 2, 3, 1)
	suite.Require().NoError(err)

	testCases := []struct {
		name        string
		teamID      int
		userID      int
		expectError bool
	}{
		{
			name: "成功移除成员",
			teamID: team.ID, userID: 2,
			expectError: false,
		},
		{
			name: "移除不存在的成员",
			teamID: team.ID, userID: 999,
			expectError: true,
		},
		{
			name: "尝试移除团队所有者",
			teamID: team.ID, userID: 1,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := suite.teamService.RemoveTeamMemberCompat(tc.teamID, tc.userID)

			if tc.expectError {
				suite.Error(err)
			} else {
				suite.NoError(err)
				
				// 验证成员状态已变为inactive
				var member models.TeamMember
				err = suite.db.Where("team_id = ? AND user_id = ?", tc.teamID, tc.userID).First(&member).Error
				suite.NoError(err)
				suite.Equal(models.MemberStatusInactive, member.Status)
			}
		})
	}
}

// TestInviteUser 测试邀请用户
func (suite *TeamServiceTestSuite) TestInviteUser() {
	// 创建测试团队
	team, err := suite.teamService.CreateTeam("default", 1, "测试团队", "描述", 1)
	suite.Require().NoError(err)

	testCases := []struct {
		name        string
		teamID      int
		email       string
		roleID      int
		message     string
		invitedBy   int
		expectError bool
	}{
		{
			name: "成功邀请用户",
			teamID: team.ID, email: "newuser@test.com", roleID: 3,
			message: "欢迎加入我们的团队", invitedBy: 1,
			expectError: false,
		},
		{
			name: "邮箱格式错误",
			teamID: team.ID, email: "invalid-email", roleID: 3,
			message: "测试消息", invitedBy: 1,
			expectError: true,
		},
		{
			name: "角色ID不存在",
			teamID: team.ID, email: "test2@test.com", roleID: 999,
			message: "测试消息", invitedBy: 1,
			expectError: true,
		},
		{
			name: "重复邀请",
			teamID: team.ID, email: "newuser@test.com", roleID: 3,
			message: "重复邀请", invitedBy: 1,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			invitation, err := suite.teamService.InviteUserCompat(
				tc.teamID, tc.email, tc.roleID, tc.message, tc.invitedBy,
			)

			if tc.expectError {
				suite.Error(err)
				suite.Nil(invitation)
			} else {
				suite.NoError(err)
				suite.NotNil(invitation)
				suite.Equal(tc.teamID, invitation.TeamID)
				suite.Equal(tc.email, invitation.Email)
				suite.Equal(tc.roleID, invitation.RoleID)
				suite.Equal(models.InvitationStatusPending, invitation.Status)
				suite.NotEmpty(invitation.Token)
				suite.True(invitation.ExpiresAt.After(time.Now()))
			}
		})
	}
}

// TestAcceptInvitation 测试接受邀请
func (suite *TeamServiceTestSuite) TestAcceptInvitation() {
	// 创建测试团队和邀请
	team, err := suite.teamService.CreateTeam("default", 1, "测试团队", "描述", 1)
	suite.Require().NoError(err)
	
	invitation, err := suite.teamService.InviteUserCompat(team.ID, "newuser@test.com", 3, "欢迎", 1)
	suite.Require().NoError(err)

	// 创建对应的用户
	newUser := models.User{
		ID: 6, Username: "newuser", Email: "newuser@test.com",
		DisplayName: "新用户", Status: models.UserStatusActive,
	}
	suite.db.Create(&newUser)

	testCases := []struct {
		name        string
		token       string
		userID      int
		expectError bool
	}{
		{
			name: "成功接受邀请",
			token: invitation.Token, userID: 6,
			expectError: false,
		},
		{
			name: "Token不存在",
			token: "invalid-token", userID: 6,
			expectError: true,
		},
		{
			name: "用户邮箱不匹配",
			token: invitation.Token, userID: 2,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := suite.teamService.AcceptInvitation(tc.token, tc.userID)

			if tc.expectError {
				suite.Error(err)
			} else {
				suite.NoError(err)
				
				// 验证成员已创建
				var member models.TeamMember
				err = suite.db.Where("team_id = ? AND user_id = ?", team.ID, tc.userID).First(&member).Error
				suite.NoError(err)
				suite.Equal(team.ID, member.TeamID)
				suite.Equal(tc.userID, member.UserID)
				suite.Equal(models.MemberStatusActive, member.Status)

				// 验证邀请状态已更新
				var updatedInvitation models.TeamInvitation
				err = suite.db.Where("token = ?", tc.token).First(&updatedInvitation).Error
				suite.NoError(err)
				suite.Equal(models.InvitationStatusAccepted, updatedInvitation.Status)
			}
		})
	}
}

// TestCreatePermissionRequest 测试创建权限申请
func (suite *TeamServiceTestSuite) TestCreatePermissionRequest() {
	testCases := []struct {
		name        string
		projectID   int
		userID      int
		reqType     string
		permission  string
		reason      string
		expectError bool
	}{
		{
			name: "成功创建权限申请",
			projectID: 1, userID: 3, reqType: "role",
			permission: "admin", reason: "需要管理权限处理项目",
			expectError: false,
		},
		{
			name: "申请理由为空",
			projectID: 1, userID: 3, reqType: "role",
			permission: "admin", reason: "",
			expectError: true,
		},
		{
			name: "用户不存在",
			projectID: 1, userID: 999, reqType: "role",
			permission: "admin", reason: "测试理由",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			request, err := suite.teamService.CreatePermissionRequestCompat(
				tc.projectID, tc.userID, tc.reqType, tc.permission, tc.reason, nil,
			)

			if tc.expectError {
				suite.Error(err)
				suite.Nil(request)
			} else {
				suite.NoError(err)
				suite.NotNil(request)
				suite.Equal(tc.projectID, request.ProjectID)
				suite.Equal(tc.userID, request.UserID)
				suite.Equal(tc.reqType, request.RequestType)
				suite.Equal(tc.permission, request.Permission)
				suite.Equal(tc.reason, request.Reason)
				suite.Equal(models.RequestStatusPending, request.Status)
			}
		})
	}
}

// TestReviewPermissionRequest 测试审批权限申请
func (suite *TeamServiceTestSuite) TestReviewPermissionRequest() {
	// 创建测试权限申请
	request, err := suite.teamService.CreatePermissionRequestCompat(
		1, 3, "role", "admin", "需要管理权限", nil,
	)
	suite.Require().NoError(err)

	testCases := []struct {
		name         string
		requestID    int
		reviewerID   int
		approved     bool
		reviewReason string
		expectError  bool
	}{
		{
			name: "批准权限申请",
			requestID: request.ID, reviewerID: 1, approved: true,
			reviewReason: "申请合理，批准", expectError: false,
		},
		{
			name: "请求不存在",
			requestID: 999, reviewerID: 1, approved: true,
			reviewReason: "测试", expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := suite.teamService.ReviewPermissionRequestCompat(
				tc.requestID, tc.reviewerID, tc.approved, tc.reviewReason,
			)

			if tc.expectError {
				suite.Error(err)
			} else {
				suite.NoError(err)
				
				// 验证申请状态已更新
				var updatedRequest models.PermissionRequest
				err = suite.db.First(&updatedRequest, tc.requestID).Error
				suite.NoError(err)
				
				expectedStatus := models.RequestStatusApproved
				if !tc.approved {
					expectedStatus = models.RequestStatusRejected
				}
				suite.Equal(expectedStatus, updatedRequest.Status)
				suite.Equal(&tc.reviewerID, updatedRequest.ReviewedBy)
				suite.NotNil(updatedRequest.ReviewedAt)
			}
		})
	}
}

// TestGetTeamsByProject 测试按项目获取团队
func (suite *TeamServiceTestSuite) TestGetTeamsByProject() {
	// 创建多个测试团队
	team1, _ := suite.teamService.CreateTeam("default", 1, "开发团队", "开发", 1)
	team2, _ := suite.teamService.CreateTeam("default", 1, "测试团队", "测试", 1)
	_, _ = suite.teamService.CreateTeam("default", 2, "其他项目团队", "其他", 1)

	teams, err := suite.teamService.GetTeamsByProject("default", 1)
	suite.NoError(err)
	suite.Len(teams, 2)
	
	teamIDs := []int{teams[0].ID, teams[1].ID}
	suite.Contains(teamIDs, team1.ID)
	suite.Contains(teamIDs, team2.ID)
}

// TestCheckUserPermission 测试检查用户权限
func (suite *TeamServiceTestSuite) TestCheckUserPermission() {
	// 创建测试团队和成员
	team, err := suite.teamService.CreateTeam("default", 1, "测试团队", "描述", 1)
	suite.Require().NoError(err)
	
	// 添加不同角色的成员
	_, err = suite.teamService.AddTeamMemberCompat(team.ID, 2, 2, 1) // 管理员
	suite.Require().NoError(err)
	_, err = suite.teamService.AddTeamMemberCompat(team.ID, 3, 3, 1) // 普通成员
	suite.Require().NoError(err)

	testCases := []struct {
		name       string
		userID     int
		projectID  int
		permission string
		expected   bool
	}{
		{
			name: "所有者有所有权限",
			userID: 1, projectID: 1, permission: models.PermissionAdmin,
			expected: true,
		},
		{
			name: "管理员有管理权限",
			userID: 2, projectID: 1, permission: models.PermissionDelete,
			expected: true,
		},
		{
			name: "普通成员没有删除权限",
			userID: 3, projectID: 1, permission: models.PermissionDelete,
			expected: false,
		},
		{
			name: "普通成员有读取权限",
			userID: 3, projectID: 1, permission: models.PermissionRead,
			expected: true,
		},
		{
			name: "非团队成员无权限",
			userID: 4, projectID: 1, permission: models.PermissionRead,
			expected: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			hasPermission, err := suite.teamService.CheckUserPermission(tc.userID, tc.projectID, tc.permission)
			suite.NoError(err)
			suite.Equal(tc.expected, hasPermission)
		})
	}
}

// TestTeamServiceTestSuite 运行测试套件
func TestTeamServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TeamServiceTestSuite))
}

// 基准测试
func BenchmarkCreateTeam(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Team{}, &models.TeamMember{}, &models.User{}, &models.Role{})
	
	// 创建测试数据
	user := models.User{ID: 1, Username: "test", Email: "test@test.com", Status: models.UserStatusActive}
	db.Create(&user)
	
	permissionService := NewPermissionService(db)
	teamService := NewTeamService(db, permissionService)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		teamService.CreateTeam("default", 1, "测试团队", "描述", 1)
	}
}

func BenchmarkAddTeamMember(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Team{}, &models.TeamMember{}, &models.User{}, &models.Role{})
	
	// 创建测试数据
	users := []models.User{
		{ID: 1, Username: "owner", Email: "owner@test.com", Status: models.UserStatusActive},
		{ID: 2, Username: "member", Email: "member@test.com", Status: models.UserStatusActive},
	}
	for _, user := range users {
		db.Create(&user)
	}
	
	role := models.Role{ID: 1, TenantID: "default", ProjectID: 1, Name: "member"}
	db.Create(&role)
	
	permissionService := NewPermissionService(db)
	teamService := NewTeamService(db, permissionService)
	team, _ := teamService.CreateTeam("default", 1, "测试团队", "描述", 1)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		teamService.AddTeamMemberCompat(team.ID, 2, 1, 1)
		// 清理数据以便下次测试
		db.Delete(&models.TeamMember{}, "team_id = ? AND user_id = ?", team.ID, 2)
	}
}
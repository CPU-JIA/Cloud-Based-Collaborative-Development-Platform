package unit

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/cloud-platform/collaborative-dev/internal/models"
	"github.com/cloud-platform/collaborative-dev/internal/services"
)

// TeamServiceTestSuite 团队服务测试套件
type TeamServiceTestSuite struct {
	suite.Suite
	db             *gorm.DB
	teamSvc        *services.TeamService
	permissionSvc  *services.PermissionService
	testTenantID   string
	testProjectID  int
	testTeamID     int
	testUserID     int
	testAdminID    int
	testMemberID   int
	testRoleOwner  int
	testRoleAdmin  int
	testRoleMember int
}

func (suite *TeamServiceTestSuite) SetupSuite() {
	// 使用内存SQLite数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	suite.Require().NoError(err)

	// 创建表结构
	err = db.AutoMigrate(
		&models.Team{},
		&models.TeamMember{},
		&models.TeamInvitation{},
		&models.PermissionRequest{},
		&models.User{},
		&models.Role{},
		&models.UserRole{},
		&models.TeamActivity{},
		&models.FilePermission{},
		&models.ShareLink{},
		&models.AccessLog{},
	)
	suite.Require().NoError(err)

	suite.db = db
	suite.teamSvc = services.NewTeamService(db)
	suite.permissionSvc = services.NewPermissionService(db)
	
	// 初始化测试数据
	suite.testTenantID = "tenant-" + uuid.New().String()
	suite.testProjectID = 1
	suite.testTeamID = 1
	suite.testUserID = 1
	suite.testAdminID = 2
	suite.testMemberID = 3
	suite.testRoleOwner = 1
	suite.testRoleAdmin = 2
	suite.testRoleMember = 3
}

func (suite *TeamServiceTestSuite) SetupTest() {
	// 清理测试数据
	suite.db.Exec("DELETE FROM teams")
	suite.db.Exec("DELETE FROM team_members")
	suite.db.Exec("DELETE FROM team_invitations")
	suite.db.Exec("DELETE FROM permission_requests")
	suite.db.Exec("DELETE FROM users")
	suite.db.Exec("DELETE FROM roles")
	suite.db.Exec("DELETE FROM user_roles")
	suite.db.Exec("DELETE FROM team_activities")
	suite.db.Exec("DELETE FROM file_permissions")
	suite.db.Exec("DELETE FROM share_links")
	suite.db.Exec("DELETE FROM access_logs")

	// 创建测试基础数据
	suite.createTestData()
}

func (suite *TeamServiceTestSuite) createTestData() {
	// 创建测试用户
	users := []*models.User{
		{
			ID:          suite.testUserID,
			TenantID:    suite.testTenantID,
			Username:    "owner",
			Email:       "owner@example.com",
			DisplayName: "Project Owner",
			Status:      models.UserStatusActive,
			PasswordHash: "hashed_password",
		},
		{
			ID:          suite.testAdminID,
			TenantID:    suite.testTenantID,
			Username:    "admin",
			Email:       "admin@example.com",
			DisplayName: "Team Admin",
			Status:      models.UserStatusActive,
			PasswordHash: "hashed_password",
		},
		{
			ID:          suite.testMemberID,
			TenantID:    suite.testTenantID,
			Username:    "member",
			Email:       "member@example.com",
			DisplayName: "Team Member",
			Status:      models.UserStatusActive,
			PasswordHash: "hashed_password",
		},
	}

	for _, user := range users {
		suite.db.Create(user)
	}

	// 创建测试角色
	roles := []*models.Role{
		{
			ID:          suite.testRoleOwner,
			TenantID:    suite.testTenantID,
			ProjectID:   suite.testProjectID,
			Name:        models.RoleOwner,
			Description: "Project Owner",
			Permissions: models.GetDefaultRolePermissions(models.RoleOwner),
			IsSystem:    true,
			CreatedBy:   suite.testUserID,
		},
		{
			ID:          suite.testRoleAdmin,
			TenantID:    suite.testTenantID,
			ProjectID:   suite.testProjectID,
			Name:        models.RoleAdmin,
			Description: "Project Admin",
			Permissions: models.GetDefaultRolePermissions(models.RoleAdmin),
			IsSystem:    true,
			CreatedBy:   suite.testUserID,
		},
		{
			ID:          suite.testRoleMember,
			TenantID:    suite.testTenantID,
			ProjectID:   suite.testProjectID,
			Name:        models.RoleMember,
			Description: "Project Member",
			Permissions: models.GetDefaultRolePermissions(models.RoleMember),
			IsSystem:    true,
			CreatedBy:   suite.testUserID,
		},
	}

	for _, role := range roles {
		suite.db.Create(role)
	}

	// 创建测试团队
	team := &models.Team{
		ID:          suite.testTeamID,
		TenantID:    suite.testTenantID,
		ProjectID:   suite.testProjectID,
		Name:        "Development Team",
		Description: "Main development team",
		Avatar:      "https://example.com/avatar.png",
		Settings:    `{"notifications": true, "auto_assign": false}`,
		IsActive:    true,
		CreatedBy:   suite.testUserID,
	}
	suite.db.Create(team)

	// 创建团队成员
	members := []*models.TeamMember{
		{
			ID:        1,
			TenantID:  suite.testTenantID,
			TeamID:    suite.testTeamID,
			UserID:    suite.testUserID,
			RoleID:    suite.testRoleOwner,
			Status:    models.MemberStatusActive,
			JoinedAt:  time.Now(),
			InvitedBy: suite.testUserID,
		},
		{
			ID:        2,
			TenantID:  suite.testTenantID,
			TeamID:    suite.testTeamID,
			UserID:    suite.testAdminID,
			RoleID:    suite.testRoleAdmin,
			Status:    models.MemberStatusActive,
			JoinedAt:  time.Now(),
			InvitedBy: suite.testUserID,
		},
		{
			ID:        3,
			TenantID:  suite.testTenantID,
			TeamID:    suite.testTeamID,
			UserID:    suite.testMemberID,
			RoleID:    suite.testRoleMember,
			Status:    models.MemberStatusActive,
			JoinedAt:  time.Now(),
			InvitedBy: suite.testUserID,
		},
	}

	for _, member := range members {
		suite.db.Create(member)
	}
}

// TestTeamCreation 测试团队创建
func (suite *TeamServiceTestSuite) TestTeamCreation() {
	suite.Run("成功创建团队", func() {
		newTeam := &models.Team{
			TenantID:    suite.testTenantID,
			ProjectID:   suite.testProjectID,
			Name:        "New Team",
			Description: "A new team for testing",
			IsActive:    true,
			CreatedBy:   suite.testUserID,
		}

		err := suite.db.Create(newTeam).Error
		assert.NoError(suite.T(), err)
		assert.NotZero(suite.T(), newTeam.ID)
		assert.Equal(suite.T(), suite.testTenantID, newTeam.TenantID)
		assert.Equal(suite.T(), "New Team", newTeam.Name)
	})

	suite.Run("多租户隔离验证", func() {
		otherTenantID := "other-tenant-" + uuid.New().String()
		
		team1 := &models.Team{
			TenantID:  suite.testTenantID,
			ProjectID: suite.testProjectID,
			Name:      "Team A",
			CreatedBy: suite.testUserID,
		}
		
		team2 := &models.Team{
			TenantID:  otherTenantID,
			ProjectID: suite.testProjectID,
			Name:      "Team B", 
			CreatedBy: suite.testUserID,
		}

		suite.db.Create(team1)
		suite.db.Create(team2)

		// 验证租户隔离
		var count1, count2 int64
		suite.db.Model(&models.Team{}).Where("tenant_id = ?", suite.testTenantID).Count(&count1)
		suite.db.Model(&models.Team{}).Where("tenant_id = ?", otherTenantID).Count(&count2)

		assert.Equal(suite.T(), int64(2), count1) // 包含已存在的测试团队
		assert.Equal(suite.T(), int64(1), count2)
	})
}

// TestTeamMemberManagement 测试团队成员管理
func (suite *TeamServiceTestSuite) TestTeamMemberManagement() {
	suite.Run("添加团队成员", func() {
		newUserID := 100
		
		// 创建新用户
		newUser := &models.User{
			ID:          newUserID,
			TenantID:    suite.testTenantID,
			Username:    "newmember",
			Email:       "newmember@example.com",
			DisplayName: "New Member",
			Status:      models.UserStatusActive,
			PasswordHash: "hashed_password",
		}
		suite.db.Create(newUser)

		// 添加团队成员
		newMember := &models.TeamMember{
			TenantID:  suite.testTenantID,
			TeamID:    suite.testTeamID,
			UserID:    newUserID,
			RoleID:    suite.testRoleMember,
			Status:    models.MemberStatusActive,
			JoinedAt:  time.Now(),
			InvitedBy: suite.testUserID,
		}

		err := suite.db.Create(newMember).Error
		assert.NoError(suite.T(), err)
		assert.NotZero(suite.T(), newMember.ID)
		assert.Equal(suite.T(), models.MemberStatusActive, newMember.Status)
	})

	suite.Run("更新成员角色", func() {
		// 更新成员角色从member到admin
		err := suite.db.Model(&models.TeamMember{}).
			Where("team_id = ? AND user_id = ?", suite.testTeamID, suite.testMemberID).
			Update("role_id", suite.testRoleAdmin).Error

		assert.NoError(suite.T(), err)

		// 验证角色已更新
		var member models.TeamMember
		err = suite.db.Where("team_id = ? AND user_id = ?", suite.testTeamID, suite.testMemberID).
			First(&member).Error
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), suite.testRoleAdmin, member.RoleID)
	})

	suite.Run("移除团队成员", func() {
		// 移除成员（设置为inactive）
		err := suite.db.Model(&models.TeamMember{}).
			Where("team_id = ? AND user_id = ?", suite.testTeamID, suite.testMemberID).
			Update("status", models.MemberStatusInactive).Error

		assert.NoError(suite.T(), err)

		// 验证成员状态已更新
		var member models.TeamMember
		err = suite.db.Where("team_id = ? AND user_id = ?", suite.testTeamID, suite.testMemberID).
			First(&member).Error
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), models.MemberStatusInactive, member.Status)
	})

	suite.Run("获取团队成员列表", func() {
		var members []models.TeamMember
		err := suite.db.Preload("User").Preload("Role").
			Where("team_id = ? AND status = ?", suite.testTeamID, models.MemberStatusActive).
			Find(&members).Error

		assert.NoError(suite.T(), err)
		assert.Len(suite.T(), members, 2) // owner和admin（member已被设为inactive）

		// 验证预加载数据
		for _, member := range members {
			assert.NotEmpty(suite.T(), member.User.Username)
			assert.NotEmpty(suite.T(), member.Role.Name)
		}
	})
}

// TestTeamInvitations 测试团队邀请
func (suite *TeamServiceTestSuite) TestTeamInvitations() {
	suite.Run("创建团队邀请", func() {
		inviteEmail := "invite@example.com"
		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		
		invitation := &models.TeamInvitation{
			TenantID:  suite.testTenantID,
			TeamID:    suite.testTeamID,
			ProjectID: suite.testProjectID,
			Email:     inviteEmail,
			RoleID:    suite.testRoleMember,
			Token:     suite.generateInviteToken(),
			Status:    models.InvitationStatusPending,
			ExpiresAt: expiresAt,
			Message:   "Welcome to our team!",
			InvitedBy: suite.testUserID,
		}

		err := suite.db.Create(invitation).Error
		assert.NoError(suite.T(), err)
		assert.NotZero(suite.T(), invitation.ID)
		assert.Equal(suite.T(), models.InvitationStatusPending, invitation.Status)
		assert.NotEmpty(suite.T(), invitation.Token)
	})

	suite.Run("验证邀请状态", func() {
		// 创建有效邀请
		validInvitation := &models.TeamInvitation{
			TenantID:  suite.testTenantID,
			TeamID:    suite.testTeamID,
			ProjectID: suite.testProjectID,
			Email:     "valid@example.com",
			RoleID:    suite.testRoleMember,
			Token:     suite.generateInviteToken(),
			Status:    models.InvitationStatusPending,
			ExpiresAt: time.Now().Add(1 * time.Hour),
			InvitedBy: suite.testUserID,
		}
		suite.db.Create(validInvitation)

		// 创建过期邀请
		expiredInvitation := &models.TeamInvitation{
			TenantID:  suite.testTenantID,
			TeamID:    suite.testTeamID,
			ProjectID: suite.testProjectID,
			Email:     "expired@example.com",
			RoleID:    suite.testRoleMember,
			Token:     suite.generateInviteToken(),
			Status:    models.InvitationStatusPending,
			ExpiresAt: time.Now().Add(-1 * time.Hour),
			InvitedBy: suite.testUserID,
		}
		suite.db.Create(expiredInvitation)

		// 测试有效邀请
		assert.True(suite.T(), validInvitation.IsValid())
		assert.False(suite.T(), validInvitation.IsExpired())

		// 测试过期邀请
		assert.False(suite.T(), expiredInvitation.IsValid())
		assert.True(suite.T(), expiredInvitation.IsExpired())
	})

	suite.Run("接受邀请", func() {
		newUserID := 101
		acceptedAt := time.Now()
		
		// 创建邀请
		invitation := &models.TeamInvitation{
			TenantID:  suite.testTenantID,
			TeamID:    suite.testTeamID,
			ProjectID: suite.testProjectID,
			Email:     "accept@example.com",
			RoleID:    suite.testRoleMember,
			Token:     suite.generateInviteToken(),
			Status:    models.InvitationStatusPending,
			ExpiresAt: time.Now().Add(1 * time.Hour),
			InvitedBy: suite.testUserID,
		}
		suite.db.Create(invitation)

		// 接受邀请
		invitation.Status = models.InvitationStatusAccepted
		invitation.AcceptedBy = &newUserID
		invitation.AcceptedAt = &acceptedAt
		suite.db.Save(invitation)

		// 创建对应的团队成员
		member := &models.TeamMember{
			TenantID:  suite.testTenantID,
			TeamID:    suite.testTeamID,
			UserID:    newUserID,
			RoleID:    invitation.RoleID,
			Status:    models.MemberStatusActive,
			JoinedAt:  acceptedAt,
			InvitedBy: suite.testUserID,
		}
		suite.db.Create(member)

		// 验证邀请状态
		assert.Equal(suite.T(), models.InvitationStatusAccepted, invitation.Status)
		assert.Equal(suite.T(), &newUserID, invitation.AcceptedBy)
		assert.NotNil(suite.T(), invitation.AcceptedAt)
	})

	suite.Run("拒绝邀请", func() {
		invitation := &models.TeamInvitation{
			TenantID:  suite.testTenantID,
			TeamID:    suite.testTeamID,
			ProjectID: suite.testProjectID,
			Email:     "reject@example.com",
			RoleID:    suite.testRoleMember,
			Token:     suite.generateInviteToken(),
			Status:    models.InvitationStatusPending,
			ExpiresAt: time.Now().Add(1 * time.Hour),
			InvitedBy: suite.testUserID,
		}
		suite.db.Create(invitation)

		// 拒绝邀请
		invitation.Status = models.InvitationStatusRejected
		suite.db.Save(invitation)

		assert.Equal(suite.T(), models.InvitationStatusRejected, invitation.Status)
	})
}

// TestPermissionRequests 测试权限申请
func (suite *TeamServiceTestSuite) TestPermissionRequests() {
	suite.Run("创建权限申请", func() {
		targetRoleID := suite.testRoleAdmin
		
		request := &models.PermissionRequest{
			TenantID:    suite.testTenantID,
			ProjectID:   suite.testProjectID,
			UserID:      suite.testMemberID,
			RequestType: models.RequestTypeRole,
			TargetID:    &targetRoleID,
			Permission:  models.RoleAdmin,
			Reason:      "Need admin access to manage project settings",
			Status:      models.RequestStatusPending,
			ExpiresAt:   timePtr(time.Now().Add(7 * 24 * time.Hour)),
		}

		err := suite.db.Create(request).Error
		assert.NoError(suite.T(), err)
		assert.NotZero(suite.T(), request.ID)
		assert.Equal(suite.T(), models.RequestStatusPending, request.Status)
	})

	suite.Run("审批权限申请", func() {
		targetRoleID := suite.testRoleAdmin
		
		// 创建权限申请
		request := &models.PermissionRequest{
			TenantID:    suite.testTenantID,
			ProjectID:   suite.testProjectID,
			UserID:      suite.testMemberID,
			RequestType: models.RequestTypeRole,
			TargetID:    &targetRoleID,
			Permission:  models.RoleAdmin,
			Reason:      "Need admin access",
			Status:      models.RequestStatusPending,
		}
		suite.db.Create(request)

		// 模拟管理员审批
		reviewedAt := time.Now()
		request.Status = models.RequestStatusApproved
		request.ReviewedBy = &suite.testUserID
		request.ReviewedAt = &reviewedAt
		request.ReviewReason = "Approved based on project needs"
		suite.db.Save(request)

		assert.Equal(suite.T(), models.RequestStatusApproved, request.Status)
		assert.Equal(suite.T(), &suite.testUserID, request.ReviewedBy)
		assert.NotNil(suite.T(), request.ReviewedAt)
	})

	suite.Run("检查审批权限", func() {
		targetRoleID := suite.testRoleAdmin
		
		request := &models.PermissionRequest{
			TenantID:    suite.testTenantID,
			ProjectID:   suite.testProjectID,
			UserID:      suite.testMemberID,
			RequestType: models.RequestTypeRole,
			TargetID:    &targetRoleID,
			Permission:  models.RoleAdmin,
			Status:      models.RequestStatusPending,
		}

		// 获取用户角色
		var adminRole models.Role
		suite.db.Where("id = ?", suite.testRoleOwner).First(&adminRole)
		adminRoles := []models.Role{adminRole}

		// 拥有者应该可以审批
		canApprove := request.CanApprove(suite.testUserID, adminRoles)
		assert.True(suite.T(), canApprove)

		// 普通成员不应该能审批
		var memberRole models.Role
		suite.db.Where("id = ?", suite.testRoleMember).First(&memberRole)
		memberRoles := []models.Role{memberRole}
		
		canApprove = request.CanApprove(suite.testMemberID, memberRoles)
		assert.False(suite.T(), canApprove)
	})

	suite.Run("拒绝权限申请", func() {
		targetRoleID := suite.testRoleAdmin
		
		request := &models.PermissionRequest{
			TenantID:    suite.testTenantID,
			ProjectID:   suite.testProjectID,
			UserID:      suite.testMemberID,
			RequestType: models.RequestTypeRole,
			TargetID:    &targetRoleID,
			Permission:  models.RoleAdmin,
			Status:      models.RequestStatusPending,
		}
		suite.db.Create(request)

		// 拒绝申请
		reviewedAt := time.Now()
		request.Status = models.RequestStatusRejected
		request.ReviewedBy = &suite.testUserID
		request.ReviewedAt = &reviewedAt
		request.ReviewReason = "Insufficient experience for admin role"
		suite.db.Save(request)

		assert.Equal(suite.T(), models.RequestStatusRejected, request.Status)
		assert.NotEmpty(suite.T(), request.ReviewReason)
	})
}

// TestTeamActivities 测试团队活动日志
func (suite *TeamServiceTestSuite) TestTeamActivities() {
	suite.Run("记录团队活动", func() {
		activities := []*models.TeamActivity{
			{
				TenantID:   suite.testTenantID,
				TeamID:     suite.testTeamID,
				UserID:     suite.testUserID,
				Action:     models.ActivityJoin,
				TargetType: "user",
				TargetID:   &suite.testMemberID,
				Details:    `{"user_name": "member", "role": "member"}`,
				IPAddress:  "192.168.1.100",
				UserAgent:  "Test Browser",
			},
			{
				TenantID:   suite.testTenantID,
				TeamID:     suite.testTeamID,
				UserID:     suite.testAdminID,
				Action:     models.ActivityRoleChange,
				TargetType: "user",
				TargetID:   &suite.testMemberID,
				Details:    `{"from_role": "member", "to_role": "admin"}`,
				IPAddress:  "192.168.1.101",
				UserAgent:  "Admin Browser",
			},
			{
				TenantID:   suite.testTenantID,
				TeamID:     suite.testTeamID,
				UserID:     suite.testMemberID,
				Action:     models.ActivityFileUpload,
				TargetType: "file",
				TargetID:   intPtr(123),
				Details:    `{"file_name": "document.pdf", "size": 1024000}`,
				IPAddress:  "192.168.1.102",
				UserAgent:  "File Manager",
			},
		}

		for _, activity := range activities {
			err := suite.db.Create(activity).Error
			assert.NoError(suite.T(), err)
		}

		// 获取团队活动日志
		var logs []models.TeamActivity
		err := suite.db.Where("team_id = ?", suite.testTeamID).
			Order("created_at DESC").Find(&logs).Error

		assert.NoError(suite.T(), err)
		assert.Len(suite.T(), logs, 3)
		
		// 验证活动类型
		actionTypes := make(map[string]bool)
		for _, log := range logs {
			actionTypes[log.Action] = true
		}
		assert.True(suite.T(), actionTypes[models.ActivityJoin])
		assert.True(suite.T(), actionTypes[models.ActivityRoleChange])
		assert.True(suite.T(), actionTypes[models.ActivityFileUpload])
	})

	suite.Run("多租户活动隔离", func() {
		otherTenantID := "other-tenant-" + uuid.New().String()
		
		// 为其他租户创建活动
		otherActivity := &models.TeamActivity{
			TenantID:   otherTenantID,
			TeamID:     suite.testTeamID,
			UserID:     suite.testUserID,
			Action:     models.ActivityFileShare,
			TargetType: "file",
			IPAddress:  "10.0.0.1",
			UserAgent:  "Other Browser",
		}
		suite.db.Create(otherActivity)

		// 获取当前租户的活动
		var currentTenantLogs []models.TeamActivity
		err := suite.db.Where("tenant_id = ? AND team_id = ?", suite.testTenantID, suite.testTeamID).
			Find(&currentTenantLogs).Error

		assert.NoError(suite.T(), err)
		
		// 验证租户隔离
		for _, log := range currentTenantLogs {
			assert.Equal(suite.T(), suite.testTenantID, log.TenantID)
		}

		// 获取其他租户的活动
		var otherTenantLogs []models.TeamActivity
		err = suite.db.Where("tenant_id = ?", otherTenantID).Find(&otherTenantLogs).Error

		assert.NoError(suite.T(), err)
		assert.Len(suite.T(), otherTenantLogs, 1)
		assert.Equal(suite.T(), otherTenantID, otherTenantLogs[0].TenantID)
	})
}

// TestUserMethods 测试用户模型方法
func (suite *TeamServiceTestSuite) TestUserMethods() {
	suite.Run("获取用户团队角色", func() {
		var user models.User
		err := suite.db.Preload("TeamMemberships.Role").
			Where("id = ?", suite.testUserID).First(&user).Error
		assert.NoError(suite.T(), err)

		// 获取用户在团队中的角色
		role := user.GetTeamRole(suite.testTeamID)
		assert.NotNil(suite.T(), role)
		assert.Equal(suite.T(), models.RoleOwner, role.Name)
	})

	suite.Run("检查用户是否是团队成员", func() {
		var user models.User
		err := suite.db.Preload("TeamMemberships").
			Where("id = ?", suite.testUserID).First(&user).Error
		assert.NoError(suite.T(), err)

		// 检查用户是否是团队成员
		isMember := user.IsTeamMember(suite.testTeamID)
		assert.True(suite.T(), isMember)

		// 检查用户是否是其他团队成员
		isMemberOfOtherTeam := user.IsTeamMember(999)
		assert.False(suite.T(), isMemberOfOtherTeam)
	})
}

// TestTeamMethods 测试团队模型方法
func (suite *TeamServiceTestSuite) TestTeamMethods() {
	suite.Run("获取团队成员数量", func() {
		var team models.Team
		err := suite.db.Preload("Members").Where("id = ?", suite.testTeamID).First(&team).Error
		assert.NoError(suite.T(), err)

		memberCount := team.GetMemberCount()
		assert.Equal(suite.T(), 3, memberCount) // owner, admin, member
	})

	suite.Run("获取角色分布", func() {
		var team models.Team
		err := suite.db.Preload("Members.Role").Where("id = ?", suite.testTeamID).First(&team).Error
		assert.NoError(suite.T(), err)

		distribution := team.GetRoleDistribution()
		assert.Equal(suite.T(), 1, distribution[models.RoleOwner])
		assert.Equal(suite.T(), 1, distribution[models.RoleAdmin])
		assert.Equal(suite.T(), 1, distribution[models.RoleMember])
	})
}

// TestTeamMemberMethods 测试团队成员模型方法
func (suite *TeamServiceTestSuite) TestTeamMemberMethods() {
	suite.Run("检查成员是否活跃", func() {
		var member models.TeamMember
		err := suite.db.Where("team_id = ? AND user_id = ?", suite.testTeamID, suite.testUserID).
			First(&member).Error
		assert.NoError(suite.T(), err)

		assert.True(suite.T(), member.IsActive())
		assert.Equal(suite.T(), models.MemberStatusActive, member.Status)
	})

	suite.Run("检查成员权限", func() {
		var member models.TeamMember
		err := suite.db.Preload("Role").Where("team_id = ? AND user_id = ?", suite.testTeamID, suite.testUserID).
			First(&member).Error
		assert.NoError(suite.T(), err)

		// Owner应该有admin权限
		hasAdminPermission := member.HasPermission(models.PermissionAdmin)
		assert.True(suite.T(), hasAdminPermission)

		// Owner应该有read权限
		hasReadPermission := member.HasPermission(models.PermissionRead)
		assert.True(suite.T(), hasReadPermission)
	})
}

// TestRoleBasedAccessControl 测试基于角色的访问控制
func (suite *TeamServiceTestSuite) TestRoleBasedAccessControl() {
	suite.Run("角色权限层级验证", func() {
		// 获取不同角色的权限
		ownerPermissions := models.GetDefaultRolePermissions(models.RoleOwner)
		adminPermissions := models.GetDefaultRolePermissions(models.RoleAdmin)
		memberPermissions := models.GetDefaultRolePermissions(models.RoleMember)
		viewerPermissions := models.GetDefaultRolePermissions(models.RoleViewer)
		guestPermissions := models.GetDefaultRolePermissions(models.RoleGuest)

		// 验证权限层级
		assert.Contains(suite.T(), ownerPermissions, models.PermissionAdmin)
		assert.Contains(suite.T(), ownerPermissions, models.PermissionDelete)
		assert.Contains(suite.T(), ownerPermissions, models.PermissionWrite)
		assert.Contains(suite.T(), ownerPermissions, models.PermissionRead)

		assert.NotContains(suite.T(), adminPermissions, models.PermissionAdmin)
		assert.Contains(suite.T(), adminPermissions, models.PermissionDelete)
		assert.Contains(suite.T(), adminPermissions, models.PermissionWrite)
		assert.Contains(suite.T(), adminPermissions, models.PermissionRead)

		assert.NotContains(suite.T(), memberPermissions, models.PermissionDelete)
		assert.Contains(suite.T(), memberPermissions, models.PermissionWrite)
		assert.Contains(suite.T(), memberPermissions, models.PermissionRead)

		assert.NotContains(suite.T(), viewerPermissions, models.PermissionWrite)
		assert.Contains(suite.T(), viewerPermissions, models.PermissionRead)

		assert.Equal(suite.T(), []string{models.PermissionRead}, guestPermissions)
	})

	suite.Run("用户权限检查", func() {
		// 检查Owner权限
		hasOwnerPermission, err := suite.permissionSvc.CheckUserPermission(
			suite.testUserID,
			suite.testProjectID,
			models.PermissionAdmin,
		)
		assert.NoError(suite.T(), err)
		assert.True(suite.T(), hasOwnerPermission)

		// 检查Admin权限
		hasAdminPermission, err := suite.permissionSvc.CheckUserPermission(
			suite.testAdminID,
			suite.testProjectID,
			models.PermissionDelete,
		)
		assert.NoError(suite.T(), err)
		assert.True(suite.T(), hasAdminPermission)

		// 检查Member权限（应该没有delete权限）
		hasMemberDeletePermission, err := suite.permissionSvc.CheckUserPermission(
			suite.testMemberID,
			suite.testProjectID,
			models.PermissionDelete,
		)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), hasMemberDeletePermission)

		// 检查Member权限（应该有write权限）
		hasMemberWritePermission, err := suite.permissionSvc.CheckUserPermission(
			suite.testMemberID,
			suite.testProjectID,
			models.PermissionWrite,
		)
		assert.NoError(suite.T(), err)
		assert.True(suite.T(), hasMemberWritePermission)
	})
}

// TestMultiTenantRBAC 测试多租户RBAC
func (suite *TeamServiceTestSuite) TestMultiTenantRBAC() {
	suite.Run("跨租户权限隔离", func() {
		otherTenantID := "other-tenant-" + uuid.New().String()
		otherProjectID := 999
		
		// 为其他租户创建用户和角色
		otherUser := &models.User{
			ID:          999,
			TenantID:    otherTenantID,
			Username:    "otheruser",
			Email:       "other@example.com",
			Status:      models.UserStatusActive,
			PasswordHash: "hashed_password",
		}
		suite.db.Create(otherUser)

		otherRole := &models.Role{
			ID:          999,
			TenantID:    otherTenantID,
			ProjectID:   otherProjectID,
			Name:        models.RoleAdmin,
			Description: "Other Tenant Admin",
			Permissions: models.GetDefaultRolePermissions(models.RoleAdmin),
			IsSystem:    true,
			CreatedBy:   999,
		}
		suite.db.Create(otherRole)

		otherTeam := &models.Team{
			ID:        999,
			TenantID:  otherTenantID,
			ProjectID: otherProjectID,
			Name:      "Other Team",
			CreatedBy: 999,
		}
		suite.db.Create(otherTeam)

		otherMember := &models.TeamMember{
			TenantID:  otherTenantID,
			TeamID:    999,
			UserID:    999,
			RoleID:    999,
			Status:    models.MemberStatusActive,
			JoinedAt:  time.Now(),
			InvitedBy: 999,
		}
		suite.db.Create(otherMember)

		// 验证租户隔离 - 当前租户用户不能访问其他租户的项目
		hasPermission, err := suite.permissionSvc.CheckUserPermission(
			suite.testUserID,
			otherProjectID,
			models.PermissionRead,
		)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), hasPermission)

		// 验证其他租户用户不能访问当前租户项目
		hasPermission, err = suite.permissionSvc.CheckUserPermission(
			999,
			suite.testProjectID,
			models.PermissionRead,
		)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), hasPermission)
	})

	suite.Run("同租户跨项目权限隔离", func() {
		otherProjectID := 888
		
		// 为同租户创建另一个项目的角色和团队
		otherProjectRole := &models.Role{
			ID:          888,
			TenantID:    suite.testTenantID,
			ProjectID:   otherProjectID,
			Name:        models.RoleAdmin,
			Description: "Other Project Admin",
			Permissions: models.GetDefaultRolePermissions(models.RoleAdmin),
			IsSystem:    true,
			CreatedBy:   suite.testUserID,
		}
		suite.db.Create(otherProjectRole)

		// 当前用户在其他项目中没有权限
		hasPermission, err := suite.permissionSvc.CheckUserPermission(
			suite.testUserID,
			otherProjectID,
			models.PermissionRead,
		)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), hasPermission)
	})
}

// Helper functions
func (suite *TeamServiceTestSuite) generateInviteToken() string {
	return "invite-" + uuid.New().String()[:8]
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func intPtr(i int) *int {
	return &i
}

// 运行测试套件
func TestTeamServiceSuite(t *testing.T) {
	suite.Run(t, new(TeamServiceTestSuite))
}
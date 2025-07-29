package unit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/cloud-platform/collaborative-dev/internal/models"
	"github.com/cloud-platform/collaborative-dev/internal/services"
)

// MockFile Mock文件模型
type MockFile struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FolderID *int   `json:"folder_id"`
	TenantID string `json:"tenant_id"`
}

// MockFolder Mock文件夹模型
type MockFolder struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ParentID *int   `json:"parent_id"`
	TenantID string `json:"tenant_id"`
}

// PermissionServiceTestSuite 权限服务测试套件
type PermissionServiceTestSuite struct {
	suite.Suite
	db              *gorm.DB
	permissionSvc   *services.PermissionService
	testTenantID    string
	testProjectID   int
	testUserID      int
	testRoleID      int
	testFileID      int
	testFolderID    int
	testParentFolderID int
}

func (suite *PermissionServiceTestSuite) SetupSuite() {
	// 使用内存SQLite数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	suite.Require().NoError(err)

	// 创建表结构
	err = db.AutoMigrate(
		&models.FilePermission{},
		&models.ShareLink{},
		&models.AccessLog{},
		&models.Role{},
		&models.UserRole{},
		&models.Team{},
		&models.TeamMember{},
		&models.TeamInvitation{},
		&models.PermissionRequest{},
		&models.User{},
		&models.TeamActivity{},
		&MockFile{},
		&MockFolder{},
	)
	suite.Require().NoError(err)

	suite.db = db
	suite.permissionSvc = services.NewPermissionService(db)
	
	// 初始化测试数据
	suite.testTenantID = "tenant-" + uuid.New().String()
	suite.testProjectID = 1
	suite.testUserID = 1
	suite.testRoleID = 1
	suite.testFileID = 1
	suite.testFolderID = 1
	suite.testParentFolderID = 2
}

func (suite *PermissionServiceTestSuite) SetupTest() {
	// 清理测试数据
	suite.db.Exec("DELETE FROM file_permissions")
	suite.db.Exec("DELETE FROM share_links")
	suite.db.Exec("DELETE FROM access_logs")
	suite.db.Exec("DELETE FROM roles")
	suite.db.Exec("DELETE FROM user_roles")
	suite.db.Exec("DELETE FROM teams")
	suite.db.Exec("DELETE FROM team_members")
	suite.db.Exec("DELETE FROM team_invitations")
	suite.db.Exec("DELETE FROM permission_requests")
	suite.db.Exec("DELETE FROM users")
	suite.db.Exec("DELETE FROM team_activities")
	suite.db.Exec("DELETE FROM mock_files")
	suite.db.Exec("DELETE FROM mock_folders")

	// 创建测试基础数据
	suite.createTestData()
}

func (suite *PermissionServiceTestSuite) createTestData() {
	// 创建测试用户
	user := &models.User{
		ID:          suite.testUserID,
		TenantID:    suite.testTenantID,
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Status:      models.UserStatusActive,
		PasswordHash: "hashed_password",
	}
	suite.db.Create(user)

	// 创建测试角色
	role := &models.Role{
		ID:          suite.testRoleID,
		TenantID:    suite.testTenantID,
		ProjectID:   suite.testProjectID,
		Name:        models.RoleMember,
		Description: "Project Member Role",
		Permissions: []string{models.PermissionRead, models.PermissionWrite},
		IsSystem:    true,
		CreatedBy:   suite.testUserID,
	}
	suite.db.Create(role)

	// 创建用户角色关联
	userRole := &models.UserRole{
		TenantID:  suite.testTenantID,
		UserID:    suite.testUserID,
		RoleID:    suite.testRoleID,
		ProjectID: suite.testProjectID,
		GrantedBy: suite.testUserID,
	}
	suite.db.Create(userRole)

	// 创建测试文件夹（父文件夹）
	parentFolder := &MockFolder{
		ID:       suite.testParentFolderID,
		Name:     "parent-folder",
		ParentID: nil,
		TenantID: suite.testTenantID,
	}
	suite.db.Create(parentFolder)

	// 创建测试文件夹（子文件夹）
	folder := &MockFolder{
		ID:       suite.testFolderID,
		Name:     "test-folder",
		ParentID: &suite.testParentFolderID,
		TenantID: suite.testTenantID,
	}
	suite.db.Create(folder)

	// 创建测试文件
	file := &MockFile{
		ID:       suite.testFileID,
		Name:     "test-file.txt",
		FolderID: &suite.testFolderID,
		TenantID: suite.testTenantID,
	}
	suite.db.Create(file)
}

// TestCreateFilePermission 测试创建文件权限
func (suite *PermissionServiceTestSuite) TestCreateFilePermission() {
	suite.Run("创建用户文件权限", func() {
		permission, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			&suite.testUserID,
			nil,
			models.PermissionRead,
			suite.testUserID,
			nil,
		)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), permission)
		assert.Equal(suite.T(), suite.testTenantID, permission.TenantID)
		assert.Equal(suite.T(), &suite.testFileID, permission.FileID)
		assert.Equal(suite.T(), &suite.testUserID, permission.UserID)
		assert.Equal(suite.T(), models.PermissionRead, permission.Permission)
		assert.True(suite.T(), permission.IsActive)
	})

	suite.Run("创建角色文件权限", func() {
		permission, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			nil,
			&suite.testRoleID,
			models.PermissionWrite,
			suite.testUserID,
			nil,
		)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), permission)
		assert.Equal(suite.T(), &suite.testRoleID, permission.RoleID)
		assert.Equal(suite.T(), models.PermissionWrite, permission.Permission)
	})

	suite.Run("创建文件夹权限", func() {
		permission, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			nil,
			&suite.testFolderID,
			&suite.testUserID,
			nil,
			models.PermissionAdmin,
			suite.testUserID,
			nil,
		)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), permission)
		assert.Equal(suite.T(), &suite.testFolderID, permission.FolderID)
		assert.Equal(suite.T(), models.PermissionAdmin, permission.Permission)
	})

	suite.Run("创建带过期时间的权限", func() {
		expiresAt := time.Now().Add(24 * time.Hour)
		permission, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			&suite.testUserID,
			nil,
			models.PermissionRead,
			suite.testUserID,
			&expiresAt,
		)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), permission)
		assert.NotNil(suite.T(), permission.ExpiresAt)
		assert.WithinDuration(suite.T(), expiresAt, *permission.ExpiresAt, time.Second)
	})

	suite.Run("参数验证失败 - 文件ID和文件夹ID都为空", func() {
		permission, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			nil,
			nil,
			&suite.testUserID,
			nil,
			models.PermissionRead,
			suite.testUserID,
			nil,
		)

		assert.Error(suite.T(), err)
		assert.Nil(suite.T(), permission)
		assert.Contains(suite.T(), err.Error(), "文件ID或文件夹ID不能同时为空")
	})

	suite.Run("参数验证失败 - 用户ID和角色ID都为空", func() {
		permission, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			nil,
			nil,
			models.PermissionRead,
			suite.testUserID,
			nil,
		)

		assert.Error(suite.T(), err)
		assert.Nil(suite.T(), permission)
		assert.Contains(suite.T(), err.Error(), "用户ID或角色ID不能同时为空")
	})
}

// TestCheckFilePermission 测试检查文件权限
func (suite *PermissionServiceTestSuite) TestCheckFilePermission() {
	suite.Run("检查直接文件权限", func() {
		// 创建直接文件权限
		_, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			&suite.testUserID,
			nil,
			models.PermissionRead,
			suite.testUserID,
			nil,
		)
		suite.Require().NoError(err)

		// 检查读权限
		hasPermission, err := suite.permissionSvc.CheckFilePermission(
			suite.testTenantID,
			suite.testFileID,
			suite.testUserID,
			models.ActionView,
		)

		assert.NoError(suite.T(), err)
		assert.True(suite.T(), hasPermission)
	})

	suite.Run("检查角色文件权限", func() {
		// 创建角色文件权限
		_, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			nil,
			&suite.testRoleID,
			models.PermissionWrite,
			suite.testUserID,
			nil,
		)
		suite.Require().NoError(err)

		// 检查写权限
		hasPermission, err := suite.permissionSvc.CheckFilePermission(
			suite.testTenantID,
			suite.testFileID,
			suite.testUserID,
			models.ActionEdit,
		)

		assert.NoError(suite.T(), err)
		assert.True(suite.T(), hasPermission)
	})

	suite.Run("检查继承的文件夹权限", func() {
		// 为父文件夹创建权限
		_, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			nil,
			&suite.testParentFolderID,
			&suite.testUserID,
			nil,
			models.PermissionAdmin,
			suite.testUserID,
			nil,
		)
		suite.Require().NoError(err)

		// 检查子文件夹中文件的权限（应该继承）
		hasPermission, err := suite.permissionSvc.CheckFilePermission(
			suite.testTenantID,
			suite.testFileID,
			suite.testUserID,
			models.ActionDelete,
		)

		assert.NoError(suite.T(), err)
		assert.True(suite.T(), hasPermission)
	})

	suite.Run("权限不足", func() {
		// 创建只读权限
		_, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			&suite.testUserID,
			nil,
			models.PermissionRead,
			suite.testUserID,
			nil,
		)
		suite.Require().NoError(err)

		// 检查删除权限（应该没有）
		hasPermission, err := suite.permissionSvc.CheckFilePermission(
			suite.testTenantID,
			suite.testFileID,
			suite.testUserID,
			models.ActionDelete,
		)

		assert.NoError(suite.T(), err)
		assert.False(suite.T(), hasPermission)
	})

	suite.Run("多租户隔离", func() {
		otherTenantID := "other-tenant-" + uuid.New().String()
		
		// 为当前租户创建权限
		_, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			&suite.testUserID,
			nil,
			models.PermissionAdmin,
			suite.testUserID,
			nil,
		)
		suite.Require().NoError(err)

		// 检查其他租户的权限（应该没有）
		hasPermission, err := suite.permissionSvc.CheckFilePermission(
			otherTenantID,
			suite.testFileID,
			suite.testUserID,
			models.ActionView,
		)

		assert.NoError(suite.T(), err)
		assert.False(suite.T(), hasPermission)
	})
}

// TestShareLinkManagement 测试分享链接管理
func (suite *PermissionServiceTestSuite) TestShareLinkManagement() {
	suite.Run("创建文件分享链接", func() {
		password := "test123"
		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		maxDownloads := 10

		shareLink, err := suite.permissionSvc.CreateShareLink(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			suite.testUserID,
			models.PermissionRead,
			password,
			&expiresAt,
			&maxDownloads,
		)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), shareLink)
		assert.Equal(suite.T(), suite.testTenantID, shareLink.TenantID)
		assert.Equal(suite.T(), &suite.testFileID, shareLink.FileID)
		assert.Equal(suite.T(), password, shareLink.Password)
		assert.Equal(suite.T(), models.PermissionRead, shareLink.Permission)
		assert.NotEmpty(suite.T(), shareLink.ShareToken)
		assert.Equal(suite.T(), 0, shareLink.Downloads)
		assert.True(suite.T(), shareLink.IsActive)
		assert.WithinDuration(suite.T(), expiresAt, *shareLink.ExpiresAt, time.Second)
		assert.Equal(suite.T(), &maxDownloads, shareLink.MaxDownloads)
	})

	suite.Run("创建文件夹分享链接", func() {
		shareLink, err := suite.permissionSvc.CreateShareLink(
			suite.testTenantID,
			nil,
			&suite.testFolderID,
			suite.testUserID,
			models.PermissionWrite,
			"",
			nil,
			nil,
		)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), shareLink)
		assert.Equal(suite.T(), &suite.testFolderID, shareLink.FolderID)
		assert.Empty(suite.T(), shareLink.Password)
		assert.Nil(suite.T(), shareLink.ExpiresAt)
		assert.Nil(suite.T(), shareLink.MaxDownloads)
	})

	suite.Run("获取分享链接", func() {
		// 创建分享链接
		originalLink, err := suite.permissionSvc.CreateShareLink(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			suite.testUserID,
			models.PermissionRead,
			"",
			nil,
			nil,
		)
		suite.Require().NoError(err)

		// 获取分享链接
		retrievedLink, err := suite.permissionSvc.GetShareLink(originalLink.ShareToken)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), retrievedLink)
		assert.Equal(suite.T(), originalLink.ID, retrievedLink.ID)
		assert.Equal(suite.T(), originalLink.ShareToken, retrievedLink.ShareToken)
	})

	suite.Run("验证分享访问权限", func() {
		password := "secret123"
		shareLink, err := suite.permissionSvc.CreateShareLink(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			suite.testUserID,
			models.PermissionRead,
			password,
			nil,
			nil,
		)
		suite.Require().NoError(err)

		// 正确密码验证
		validatedLink, err := suite.permissionSvc.ValidateShareAccess(
			shareLink.ShareToken,
			password,
		)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), validatedLink)
		assert.Equal(suite.T(), shareLink.ID, validatedLink.ID)

		// 错误密码验证
		invalidLink, err := suite.permissionSvc.ValidateShareAccess(
			shareLink.ShareToken,
			"wrong-password",
		)

		assert.Error(suite.T(), err)
		assert.Nil(suite.T(), invalidLink)
		assert.Contains(suite.T(), err.Error(), "密码错误")
	})

	suite.Run("增加分享下载次数", func() {
		shareLink, err := suite.permissionSvc.CreateShareLink(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			suite.testUserID,
			models.PermissionRead,
			"",
			nil,
			nil,
		)
		suite.Require().NoError(err)

		// 增加下载次数
		err = suite.permissionSvc.IncrementShareDownload(shareLink.ShareToken)
		assert.NoError(suite.T(), err)

		// 验证下载次数增加
		updatedLink, err := suite.permissionSvc.GetShareLink(shareLink.ShareToken)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 1, updatedLink.Downloads)
	})

	suite.Run("撤销分享链接", func() {
		shareLink, err := suite.permissionSvc.CreateShareLink(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			suite.testUserID,
			models.PermissionRead,
			"",
			nil,
			nil,
		)
		suite.Require().NoError(err)

		// 撤销分享链接
		err = suite.permissionSvc.RevokeShareLink(shareLink.ShareToken, suite.testUserID)
		assert.NoError(suite.T(), err)

		// 验证链接已被撤销
		revokedLink, err := suite.permissionSvc.GetShareLink(shareLink.ShareToken)
		assert.Error(suite.T(), err)
		assert.Nil(suite.T(), revokedLink)
	})

	suite.Run("无权限撤销他人的分享链接", func() {
		otherUserID := 999
		shareLink, err := suite.permissionSvc.CreateShareLink(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			suite.testUserID,
			models.PermissionRead,
			"",
			nil,
			nil,
		)
		suite.Require().NoError(err)

		// 尝试用其他用户撤销
		err = suite.permissionSvc.RevokeShareLink(shareLink.ShareToken, otherUserID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "无权限撤销此分享链接")
	})

	suite.Run("列出用户的分享链接", func() {
		// 创建多个分享链接
		for i := 0; i < 3; i++ {
			_, err := suite.permissionSvc.CreateShareLink(
				suite.testTenantID,
				&suite.testFileID,
				nil,
				suite.testUserID,
				models.PermissionRead,
				"",
				nil,
				nil,
			)
			suite.Require().NoError(err)
		}

		// 列出分享链接
		shareLinks, total, err := suite.permissionSvc.ListUserShareLinks(
			suite.testTenantID,
			suite.testUserID,
			1,
			10,
		)

		assert.NoError(suite.T(), err)
		assert.Len(suite.T(), shareLinks, 3)
		assert.Equal(suite.T(), int64(3), total)
	})
}

// TestAccessLogging 测试访问日志
func (suite *PermissionServiceTestSuite) TestAccessLogging() {
	suite.Run("记录文件访问日志", func() {
		ipAddress := "192.168.1.100"
		userAgent := "Mozilla/5.0 Test Browser"
		action := models.ActionView

		err := suite.permissionSvc.LogAccess(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			nil,
			&suite.testUserID,
			action,
			ipAddress,
			userAgent,
			true,
			"",
		)

		assert.NoError(suite.T(), err)

		// 验证日志记录
		logs, total, err := suite.permissionSvc.GetAccessLogs(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			1,
			10,
		)

		assert.NoError(suite.T(), err)
		assert.Len(suite.T(), logs, 1)
		assert.Equal(suite.T(), int64(1), total)
		
		log := logs[0]
		assert.Equal(suite.T(), suite.testTenantID, log.TenantID)
		assert.Equal(suite.T(), &suite.testFileID, log.FileID)
		assert.Equal(suite.T(), &suite.testUserID, log.UserID)
		assert.Equal(suite.T(), action, log.Action)
		assert.Equal(suite.T(), ipAddress, log.IPAddress)
		assert.Equal(suite.T(), userAgent, log.UserAgent)
		assert.True(suite.T(), log.Success)
		assert.Empty(suite.T(), log.ErrorReason)
	})

	suite.Run("记录分享链接访问日志", func() {
		// 创建分享链接
		shareLink, err := suite.permissionSvc.CreateShareLink(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			suite.testUserID,
			models.PermissionRead,
			"",
			nil,
			nil,
		)
		suite.Require().NoError(err)

		// 记录匿名访问日志
		err = suite.permissionSvc.LogAccess(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			&shareLink.ShareToken,
			nil, // 匿名访问
			models.ActionDownload,
			"203.0.113.1",
			"wget/1.0",
			true,
			"",
		)

		assert.NoError(suite.T(), err)
	})

	suite.Run("记录失败的访问日志", func() {
		errorReason := "权限不足"
		
		err := suite.permissionSvc.LogAccess(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			nil,
			&suite.testUserID,
			models.ActionDelete,
			"192.168.1.100",
			"Test Client",
			false,
			errorReason,
		)

		assert.NoError(suite.T(), err)

		// 验证失败日志
		logs, _, err := suite.permissionSvc.GetAccessLogs(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			1,
			10,
		)

		assert.NoError(suite.T(), err)
		assert.Len(suite.T(), logs, 1)
		
		log := logs[0]
		assert.False(suite.T(), log.Success)
		assert.Equal(suite.T(), errorReason, log.ErrorReason)
	})

	suite.Run("获取文件夹访问日志", func() {
		// 记录文件夹访问
		err := suite.permissionSvc.LogAccess(
			suite.testTenantID,
			nil,
			&suite.testFolderID,
			nil,
			&suite.testUserID,
			models.ActionView,
			"192.168.1.100",
			"Test Browser",
			true,
			"",
		)
		suite.Require().NoError(err)

		// 获取文件夹日志
		logs, total, err := suite.permissionSvc.GetAccessLogs(
			suite.testTenantID,
			nil,
			&suite.testFolderID,
			1,
			10,
		)

		assert.NoError(suite.T(), err)
		assert.Len(suite.T(), logs, 1)
		assert.Equal(suite.T(), int64(1), total)
		assert.Equal(suite.T(), &suite.testFolderID, logs[0].FolderID)
	})
}

// TestRoleManagement 测试角色管理
func (suite *PermissionServiceTestSuite) TestRoleManagement() {
	suite.Run("创建角色", func() {
		roleName := "Custom Role"
		description := "Custom role for testing"
		permissions := []string{models.PermissionRead, models.PermissionWrite, models.PermissionShare}

		role, err := suite.permissionSvc.CreateRole(
			suite.testTenantID,
			suite.testProjectID,
			roleName,
			description,
			permissions,
			suite.testUserID,
		)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), role)
		assert.Equal(suite.T(), suite.testTenantID, role.TenantID)
		assert.Equal(suite.T(), suite.testProjectID, role.ProjectID)
		assert.Equal(suite.T(), roleName, role.Name)
		assert.Equal(suite.T(), description, role.Description)
		assert.Equal(suite.T(), permissions, role.Permissions)
		assert.False(suite.T(), role.IsSystem)
		assert.Equal(suite.T(), suite.testUserID, role.CreatedBy)
	})

	suite.Run("分配用户角色", func() {
		newUserID := 2
		
		err := suite.permissionSvc.AssignUserRole(
			suite.testTenantID,
			newUserID,
			suite.testRoleID,
			suite.testProjectID,
			suite.testUserID,
		)

		assert.NoError(suite.T(), err)

		// 验证角色分配
		var userRole models.UserRole
		err = suite.db.Where(
			"tenant_id = ? AND user_id = ? AND role_id = ? AND project_id = ?",
			suite.testTenantID, newUserID, suite.testRoleID, suite.testProjectID,
		).First(&userRole).Error

		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), newUserID, userRole.UserID)
		assert.Equal(suite.T(), suite.testRoleID, userRole.RoleID)
		assert.Equal(suite.T(), suite.testUserID, userRole.GrantedBy)
	})

	suite.Run("移除用户角色", func() {
		// 先分配角色
		newUserID := 3
		err := suite.permissionSvc.AssignUserRole(
			suite.testTenantID,
			newUserID,
			suite.testRoleID,
			suite.testProjectID,
			suite.testUserID,
		)
		suite.Require().NoError(err)

		// 然后移除角色
		err = suite.permissionSvc.RemoveUserRole(
			suite.testTenantID,
			newUserID,
			suite.testRoleID,
			suite.testProjectID,
		)

		assert.NoError(suite.T(), err)

		// 验证角色已移除
		var count int64
		suite.db.Model(&models.UserRole{}).Where(
			"tenant_id = ? AND user_id = ? AND role_id = ? AND project_id = ?",
			suite.testTenantID, newUserID, suite.testRoleID, suite.testProjectID,
		).Count(&count)

		assert.Equal(suite.T(), int64(0), count)
	})

	suite.Run("获取用户角色", func() {
		roles, err := suite.permissionSvc.GetUserRoles(suite.testTenantID, suite.testUserID)

		assert.NoError(suite.T(), err)
		assert.Len(suite.T(), roles, 1)
		assert.Equal(suite.T(), suite.testRoleID, roles[0].ID)
		assert.Equal(suite.T(), models.RoleMember, roles[0].Name)
	})
}

// TestMultiTenantIsolation 测试多租户隔离
func (suite *PermissionServiceTestSuite) TestMultiTenantIsolation() {
	otherTenantID := "other-tenant-" + uuid.New().String()
	otherUserID := 100
	otherRoleID := 100
	otherProjectID := 100

	suite.Run("多租户权限隔离", func() {
		// 为当前租户创建权限
		permission1, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			&suite.testUserID,
			nil,
			models.PermissionAdmin,
			suite.testUserID,
			nil,
		)
		suite.Require().NoError(err)

		// 为其他租户创建权限
		permission2, err := suite.permissionSvc.CreateFilePermission(
			otherTenantID,
			&suite.testFileID,
			nil,
			&otherUserID,
			nil,
			models.PermissionAdmin,
			otherUserID,
			nil,
		)
		suite.Require().NoError(err)

		// 验证权限隔离
		assert.Equal(suite.T(), suite.testTenantID, permission1.TenantID)
		assert.Equal(suite.T(), otherTenantID, permission2.TenantID)

		// 当前租户用户无法访问其他租户的文件
		hasPermission, err := suite.permissionSvc.CheckFilePermission(
			otherTenantID,
			suite.testFileID,
			suite.testUserID,
			models.ActionView,
		)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), hasPermission)
	})

	suite.Run("多租户角色隔离", func() {
		// 为其他租户创建角色
		role, err := suite.permissionSvc.CreateRole(
			otherTenantID,
			otherProjectID,
			"Other Tenant Role",
			"Role for other tenant",
			[]string{models.PermissionRead},
			otherUserID,
		)
		suite.Require().NoError(err)

		// 验证角色租户隔离
		assert.Equal(suite.T(), otherTenantID, role.TenantID)

		// 获取当前租户用户的角色（不应包含其他租户的角色）
		roles, err := suite.permissionSvc.GetUserRoles(suite.testTenantID, suite.testUserID)
		assert.NoError(suite.T(), err)
		
		for _, r := range roles {
			assert.Equal(suite.T(), suite.testTenantID, r.TenantID)
			assert.NotEqual(suite.T(), role.ID, r.ID)
		}
	})

	suite.Run("多租户分享链接隔离", func() {
		// 为当前租户创建分享链接
		shareLink1, err := suite.permissionSvc.CreateShareLink(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			suite.testUserID,
			models.PermissionRead,
			"",
			nil,
			nil,
		)
		suite.Require().NoError(err)

		// 为其他租户创建分享链接
		shareLink2, err := suite.permissionSvc.CreateShareLink(
			otherTenantID,
			&suite.testFileID,
			nil,
			otherUserID,
			models.PermissionRead,
			"",
			nil,
			nil,
		)
		suite.Require().NoError(err)

		// 验证分享链接租户隔离
		assert.Equal(suite.T(), suite.testTenantID, shareLink1.TenantID)
		assert.Equal(suite.T(), otherTenantID, shareLink2.TenantID)

		// 列出当前租户用户的分享链接
		shareLinks, total, err := suite.permissionSvc.ListUserShareLinks(
			suite.testTenantID,
			suite.testUserID,
			1,
			10,
		)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), int64(1), total)
		assert.Len(suite.T(), shareLinks, 1)
		assert.Equal(suite.T(), suite.testTenantID, shareLinks[0].TenantID)
	})

	suite.Run("多租户访问日志隔离", func() {
		// 为当前租户记录访问日志
		err := suite.permissionSvc.LogAccess(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			nil,
			&suite.testUserID,
			models.ActionView,
			"192.168.1.100",
			"Test Browser",
			true,
			"",
		)
		suite.Require().NoError(err)

		// 为其他租户记录访问日志
		err = suite.permissionSvc.LogAccess(
			otherTenantID,
			&suite.testFileID,
			nil,
			nil,
			&otherUserID,
			models.ActionView,
			"192.168.1.101",
			"Other Browser",
			true,
			"",
		)
		suite.Require().NoError(err)

		// 获取当前租户的访问日志
		logs, total, err := suite.permissionSvc.GetAccessLogs(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			1,
			10,
		)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), int64(1), total)
		assert.Len(suite.T(), logs, 1)
		assert.Equal(suite.T(), suite.testTenantID, logs[0].TenantID)

		// 获取其他租户的访问日志
		otherLogs, otherTotal, err := suite.permissionSvc.GetAccessLogs(
			otherTenantID,
			&suite.testFileID,
			nil,
			1,
			10,
		)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), int64(1), otherTotal)
		assert.Len(suite.T(), otherLogs, 1)
		assert.Equal(suite.T(), otherTenantID, otherLogs[0].TenantID)
	})
}

// TestPermissionInheritance 测试权限继承
func (suite *PermissionServiceTestSuite) TestPermissionInheritance() {
	suite.Run("文件夹权限继承", func() {
		// 为父文件夹设置权限
		_, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			nil,
			&suite.testParentFolderID,
			&suite.testUserID,
			nil,
			models.PermissionWrite,
			suite.testUserID,
			nil,
		)
		suite.Require().NoError(err)

		// 检查子文件夹权限（应该继承）
		hasPermission, err := suite.permissionSvc.CheckFolderPermission(
			suite.testTenantID,
			suite.testFolderID,
			suite.testUserID,
			models.ActionUpload,
		)

		assert.NoError(suite.T(), err)
		assert.True(suite.T(), hasPermission)

		// 检查子文件夹中文件的权限（应该继承）
		hasFilePermission, err := suite.permissionSvc.CheckFilePermission(
			suite.testTenantID,
			suite.testFileID,
			suite.testUserID,
			models.ActionEdit,
		)

		assert.NoError(suite.T(), err)
		assert.True(suite.T(), hasFilePermission)
	})

	suite.Run("直接权限优先于继承权限", func() {
		// 为父文件夹设置只读权限
		_, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			nil,
			&suite.testParentFolderID,
			&suite.testUserID,
			nil,
			models.PermissionRead,
			suite.testUserID,
			nil,
		)
		suite.Require().NoError(err)

		// 为子文件夹设置管理员权限（直接权限）
		_, err = suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			nil,
			&suite.testFolderID,
			&suite.testUserID,
			nil,
			models.PermissionAdmin,
			suite.testUserID,
			nil,
		)
		suite.Require().NoError(err)

		// 检查子文件夹权限（应该使用直接权限）
		hasPermission, err := suite.permissionSvc.CheckFolderPermission(
			suite.testTenantID,
			suite.testFolderID,
			suite.testUserID,
			models.ActionDelete,
		)

		assert.NoError(suite.T(), err)
		assert.True(suite.T(), hasPermission)
	})

	suite.Run("角色权限继承", func() {
		// 为父文件夹设置角色权限
		_, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			nil,
			&suite.testParentFolderID,
			nil,
			&suite.testRoleID,
			models.PermissionShare,
			suite.testUserID,
			nil,
		)
		suite.Require().NoError(err)

		// 检查用户通过角色继承的权限
		hasPermission, err := suite.permissionSvc.CheckFolderPermission(
			suite.testTenantID,
			suite.testFolderID,
			suite.testUserID,
			models.ActionShare,
		)

		assert.NoError(suite.T(), err)
		assert.True(suite.T(), hasPermission)
	})
}

// TestPermissionExpiration 测试权限过期
func (suite *PermissionServiceTestSuite) TestPermissionExpiration() {
	suite.Run("过期权限无效", func() {
		// 创建已过期的权限
		expiredTime := time.Now().Add(-1 * time.Hour)
		_, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			&suite.testUserID,
			nil,
			models.PermissionAdmin,
			suite.testUserID,
			&expiredTime,
		)
		suite.Require().NoError(err)

		// 检查权限（应该无效）
		hasPermission, err := suite.permissionSvc.CheckFilePermission(
			suite.testTenantID,
			suite.testFileID,
			suite.testUserID,
			models.ActionDelete,
		)

		assert.NoError(suite.T(), err)
		assert.False(suite.T(), hasPermission)
	})

	suite.Run("未过期权限有效", func() {
		// 创建未过期的权限
		futureTime := time.Now().Add(1 * time.Hour)
		_, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			&suite.testUserID,
			nil,
			models.PermissionAdmin,
			suite.testUserID,
			&futureTime,
		)
		suite.Require().NoError(err)

		// 检查权限（应该有效）
		hasPermission, err := suite.permissionSvc.CheckFilePermission(
			suite.testTenantID,
			suite.testFileID,
			suite.testUserID,
			models.ActionDelete,
		)

		assert.NoError(suite.T(), err)
		assert.True(suite.T(), hasPermission)
	})

	suite.Run("过期分享链接无效", func() {
		// 创建已过期的分享链接
		expiredTime := time.Now().Add(-1 * time.Hour)
		shareLink, err := suite.permissionSvc.CreateShareLink(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			suite.testUserID,
			models.PermissionRead,
			"",
			&expiredTime,
			nil,
		)
		suite.Require().NoError(err)

		// 尝试获取过期的分享链接
		retrievedLink, err := suite.permissionSvc.GetShareLink(shareLink.ShareToken)

		assert.Error(suite.T(), err)
		assert.Nil(suite.T(), retrievedLink)
		assert.Contains(suite.T(), err.Error(), "已过期或失效")
	})

	suite.Run("达到下载限制的分享链接无效", func() {
		maxDownloads := 1
		shareLink, err := suite.permissionSvc.CreateShareLink(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			suite.testUserID,
			models.PermissionRead,
			"",
			nil,
			&maxDownloads,
		)
		suite.Require().NoError(err)

		// 增加下载次数到达限制
		err = suite.permissionSvc.IncrementShareDownload(shareLink.ShareToken)
		suite.Require().NoError(err)

		// 尝试获取已达下载限制的分享链接
		retrievedLink, err := suite.permissionSvc.GetShareLink(shareLink.ShareToken)

		assert.Error(suite.T(), err)
		assert.Nil(suite.T(), retrievedLink)
		assert.Contains(suite.T(), err.Error(), "已过期或失效")
	})
}

// TestConcurrentAccess 测试并发访问
func (suite *PermissionServiceTestSuite) TestConcurrentAccess() {
	suite.Run("并发权限检查", func() {
		// 创建权限
		_, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			&suite.testFileID,
			nil,
			&suite.testUserID,
			nil,
			models.PermissionRead,
			suite.testUserID,
			nil,
		)
		suite.Require().NoError(err)

		// 并发检查权限
		goroutineCount := 10
		results := make(chan bool, goroutineCount)
		errors := make(chan error, goroutineCount)

		for i := 0; i < goroutineCount; i++ {
			go func() {
				hasPermission, err := suite.permissionSvc.CheckFilePermission(
					suite.testTenantID,
					suite.testFileID,
					suite.testUserID,
					models.ActionView,
				)
				results <- hasPermission
				errors <- err
			}()
		}

		// 收集结果
		for i := 0; i < goroutineCount; i++ {
			hasPermission := <-results
			err := <-errors
			assert.NoError(suite.T(), err)
			assert.True(suite.T(), hasPermission)
		}
	})

	suite.Run("并发分享链接创建", func() {
		goroutineCount := 5
		tokens := make(chan string, goroutineCount)
		errors := make(chan error, goroutineCount)

		for i := 0; i < goroutineCount; i++ {
			go func() {
				shareLink, err := suite.permissionSvc.CreateShareLink(
					suite.testTenantID,
					&suite.testFileID,
					nil,
					suite.testUserID,
					models.PermissionRead,
					"",
					nil,
					nil,
				)
				if shareLink != nil {
					tokens <- shareLink.ShareToken
				} else {
					tokens <- ""
				}
				errors <- err
			}()
		}

		// 收集结果
		uniqueTokens := make(map[string]bool)
		for i := 0; i < goroutineCount; i++ {
			token := <-tokens
			err := <-errors
			assert.NoError(suite.T(), err)
			assert.NotEmpty(suite.T(), token)
			
			// 验证令牌唯一性
			assert.False(suite.T(), uniqueTokens[token], "分享令牌应该是唯一的")
			uniqueTokens[token] = true
		}
	})
}

// TestEdgeCases 测试边缘情况
func (suite *PermissionServiceTestSuite) TestEdgeCases() {
	suite.Run("检查不存在文件的权限", func() {
		nonExistentFileID := 999999
		
		hasPermission, err := suite.permissionSvc.CheckFilePermission(
			suite.testTenantID,
			nonExistentFileID,
			suite.testUserID,
			models.ActionView,
		)

		// 根据实现，这可能返回错误或false
		// 这里假设返回false表示没有权限
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), hasPermission)
	})

	suite.Run("使用无效的分享令牌", func() {
		invalidToken := "invalid-token-12345"
		
		shareLink, err := suite.permissionSvc.GetShareLink(invalidToken)

		assert.Error(suite.T(), err)
		assert.Nil(suite.T(), shareLink)
	})

	suite.Run("空租户ID处理", func() {
		// 尝试用空租户ID创建权限
		permission, err := suite.permissionSvc.CreateFilePermission(
			"",
			&suite.testFileID,
			nil,
			&suite.testUserID,
			nil,
			models.PermissionRead,
			suite.testUserID,
			nil,
		)

		// 应该成功创建但租户ID为空
		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), permission)
		assert.Empty(suite.T(), permission.TenantID)
	})

	suite.Run("同时指定文件ID和文件夹ID", func() {
		// 这应该是有效的，创建的权限同时关联文件和文件夹
		permission, err := suite.permissionSvc.CreateFilePermission(
			suite.testTenantID,
			&suite.testFileID,
			&suite.testFolderID,
			&suite.testUserID,
			nil,
			models.PermissionRead,
			suite.testUserID,
			nil,
		)

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), permission)
		assert.Equal(suite.T(), &suite.testFileID, permission.FileID)
		assert.Equal(suite.T(), &suite.testFolderID, permission.FolderID)
	})
}

// 运行测试套件
func TestPermissionServiceSuite(t *testing.T) {
	suite.Run(t, new(PermissionServiceTestSuite))
}
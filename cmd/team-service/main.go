package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// 简化的团队管理服务（内存存储）
type User struct {
	ID          int       `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Avatar      string    `json:"avatar"`
	Department  string    `json:"department"`
	Position    string    `json:"position"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Role struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	IsSystem    bool     `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TeamMember struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	RoleID    int       `json:"role_id"`
	Status    string    `json:"status"`
	JoinedAt  time.Time `json:"joined_at"`
	InvitedBy int       `json:"invited_by"`
	User      User      `json:"user"`
	Role      Role      `json:"role"`
}

type Team struct {
	ID          int          `json:"id"`
	ProjectID   int          `json:"project_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Members     []TeamMember `json:"members"`
	IsActive    bool         `json:"is_active"`
	CreatedBy   int          `json:"created_by"`
	CreatedAt   time.Time    `json:"created_at"`
}

type Invitation struct {
	ID        int       `json:"id"`
	TeamID    int       `json:"team_id"`
	Email     string    `json:"email"`
	RoleID    int       `json:"role_id"`
	Token     string    `json:"token"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message"`
	InvitedBy int       `json:"invited_by"`
	CreatedAt time.Time `json:"created_at"`
}

type PermissionRequest struct {
	ID          int       `json:"id"`
	ProjectID   int       `json:"project_id"`
	UserID      int       `json:"user_id"`
	RequestType string    `json:"request_type"`
	TargetID    *int      `json:"target_id"`
	Permission  string    `json:"permission"`
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
	ReviewedBy  *int      `json:"reviewed_by"`
	ReviewedAt  *time.Time `json:"reviewed_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// 内存存储
var users = []User{
	{1, "admin", "admin@example.com", "系统管理员", "", "技术部", "管理员", "active", time.Now(), time.Now()},
	{2, "alice", "alice@example.com", "Alice Wang", "", "开发部", "高级开发工程师", "active", time.Now(), time.Now()},
	{3, "bob", "bob@example.com", "Bob Chen", "", "设计部", "UI设计师", "active", time.Now(), time.Now()},
	{4, "charlie", "charlie@example.com", "Charlie Li", "", "测试部", "测试工程师", "active", time.Now(), time.Now()},
}

var roles = []Role{
	{1, "owner", "项目所有者", []string{"read", "write", "delete", "share", "admin"}, true, time.Now(), time.Now()},
	{2, "admin", "项目管理员", []string{"read", "write", "delete", "share"}, true, time.Now(), time.Now()},
	{3, "member", "项目成员", []string{"read", "write", "share"}, true, time.Now(), time.Now()},
	{4, "viewer", "项目查看者", []string{"read"}, true, time.Now(), time.Now()},
}

var teams = []Team{
	{
		ID: 1, ProjectID: 1, Name: "开发团队", Description: "负责项目开发工作",
		Members: []TeamMember{
			{1, 1, 1, "active", time.Now(), 1, users[0], roles[0]},
			{2, 2, 3, "active", time.Now(), 1, users[1], roles[2]},
		},
		IsActive: true, CreatedBy: 1, CreatedAt: time.Now(),
	},
}

var invitations []Invitation
var permissionRequests []PermissionRequest
var teamIDCounter = 2
var memberIDCounter = 3
var invitationIDCounter = 1
var requestIDCounter = 1

func main() {
	r := gin.Default()
	
	// CORS配置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:3002"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Tenant-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	
	// API路由组
	api := r.Group("/api/v1")
	{
		// 健康检查
		api.GET("/health", healthCheck)
		
		// 团队管理
		teams := api.Group("/teams")
		{
			teams.POST("", createTeam)
			teams.GET("/project/:projectId", getProjectTeams)
			teams.GET("/:id", getTeam)
			teams.PUT("/:id", updateTeam)
			teams.DELETE("/:id", deleteTeam)
			
			// 团队成员管理
			teams.GET("/:id/members", getTeamMembers)
			teams.POST("/:id/members", addTeamMember)
			teams.PUT("/:id/members/:userId", updateMemberRole)
			teams.DELETE("/:id/members/:userId", removeTeamMember)
		}
		
		// 用户管理
		users := api.Group("/users")
		{
			users.GET("", searchUsers)
			users.GET("/:id", getUser)
			users.GET("/:id/permissions", getUserPermissions)
		}
		
		// 角色管理
		roles := api.Group("/roles")
		{
			roles.GET("/project/:projectId", getRoles)
			roles.POST("", createRole)
			roles.PUT("/:id", updateRole)
			roles.DELETE("/:id", deleteRole)
		}
		
		// 邀请管理
		invitations := api.Group("/invitations")
		{
			invitations.POST("", createInvitation)
			invitations.GET("/team/:teamId", getTeamInvitations)
			invitations.POST("/:token/accept", acceptInvitation)
			invitations.POST("/:token/reject", rejectInvitation)
		}
		
		// 权限申请
		requests := api.Group("/permission-requests")
		{
			requests.POST("", createPermissionRequest)
			requests.GET("/project/:projectId", getPermissionRequests)
			requests.POST("/:id/review", reviewPermissionRequest)
		}
	}
	
	log.Println("🚀 团队管理服务启动成功！")
	log.Println("🌐 服务地址: http://localhost:8086")
	log.Println("🔍 健康检查: http://localhost:8086/api/v1/health")
	
	r.Run(":8086")
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"service": "团队管理服务",
		"version": "1.0.0",
		"status":  "healthy",
		"teams_count": len(teams),
		"users_count": len(users),
		"roles_count": len(roles),
	})
}

func createTeam(c *gin.Context) {
	var req struct {
		ProjectID   int    `json:"project_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	
	team := Team{
		ID:          teamIDCounter,
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Description: req.Description,
		Members:     []TeamMember{},
		IsActive:    true,
		CreatedBy:   1, // 临时硬编码
		CreatedAt:   time.Now(),
	}
	
	teams = append(teams, team)
	teamIDCounter++
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"team":    team,
	})
}

func getProjectTeams(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("projectId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目ID无效"})
		return
	}
	
	var projectTeams []Team
	for _, team := range teams {
		if team.ProjectID == projectID && team.IsActive {
			projectTeams = append(projectTeams, team)
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"teams":   projectTeams,
		"total":   len(projectTeams),
	})
}

func getTeam(c *gin.Context) {
	teamID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队ID无效"})
		return
	}
	
	for _, team := range teams {
		if team.ID == teamID {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"team":    team,
			})
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "团队不存在"})
}

func updateTeam(c *gin.Context) {
	teamID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队ID无效"})
		return
	}
	
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	
	for i, team := range teams {
		if team.ID == teamID {
			teams[i].Name = req.Name
			teams[i].Description = req.Description
			
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"team":    teams[i],
			})
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "团队不存在"})
}

func deleteTeam(c *gin.Context) {
	teamID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队ID无效"})
		return
	}
	
	for i, team := range teams {
		if team.ID == teamID {
			teams[i].IsActive = false
			
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "团队已删除",
			})
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "团队不存在"})
}

func getTeamMembers(c *gin.Context) {
	teamID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队ID无效"})
		return
	}
	
	for _, team := range teams {
		if team.ID == teamID {
			var activeMembers []TeamMember
			for _, member := range team.Members {
				if member.Status == "active" {
					activeMembers = append(activeMembers, member)
				}
			}
			
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"members": activeMembers,
				"total":   len(activeMembers),
			})
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "团队不存在"})
}

func addTeamMember(c *gin.Context) {
	teamID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队ID无效"})
		return
	}
	
	var req struct {
		UserID int `json:"user_id" binding:"required"`
		RoleID int `json:"role_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	
	// 查找用户和角色
	var user *User
	var role *Role
	
	for _, u := range users {
		if u.ID == req.UserID {
			user = &u
			break
		}
	}
	
	for _, r := range roles {
		if r.ID == req.RoleID {
			role = &r
			break
		}
	}
	
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	
	if role == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}
	
	// 添加成员到团队
	for i, team := range teams {
		if team.ID == teamID {
			// 检查用户是否已是成员
			for _, member := range team.Members {
				if member.UserID == req.UserID && member.Status == "active" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "用户已是团队成员"})
					return
				}
			}
			
			newMember := TeamMember{
				ID:        memberIDCounter,
				UserID:    req.UserID,
				RoleID:    req.RoleID,
				Status:    "active",
				JoinedAt:  time.Now(),
				InvitedBy: 1, // 临时硬编码
				User:      *user,
				Role:      *role,
			}
			
			teams[i].Members = append(teams[i].Members, newMember)
			memberIDCounter++
			
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"member":  newMember,
			})
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "团队不存在"})
}

func updateMemberRole(c *gin.Context) {
	teamID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队ID无效"})
		return
	}
	
	userID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID无效"})
		return
	}
	
	var req struct {
		RoleID int `json:"role_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	
	// 查找角色
	var role *Role
	for _, r := range roles {
		if r.ID == req.RoleID {
			role = &r
			break
		}
	}
	
	if role == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}
	
	// 更新成员角色
	for i, team := range teams {
		if team.ID == teamID {
			for j, member := range team.Members {
				if member.UserID == userID && member.Status == "active" {
					teams[i].Members[j].RoleID = req.RoleID
					teams[i].Members[j].Role = *role
					
					c.JSON(http.StatusOK, gin.H{
						"success": true,
						"member":  teams[i].Members[j],
					})
					return
				}
			}
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "团队成员不存在"})
}

func removeTeamMember(c *gin.Context) {
	teamID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队ID无效"})
		return
	}
	
	userID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID无效"})
		return
	}
	
	// 移除团队成员
	for i, team := range teams {
		if team.ID == teamID {
			for j, member := range team.Members {
				if member.UserID == userID && member.Status == "active" {
					teams[i].Members[j].Status = "inactive"
					
					c.JSON(http.StatusOK, gin.H{
						"success": true,
						"message": "成员已移除",
					})
					return
				}
			}
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "团队成员不存在"})
}

func searchUsers(c *gin.Context) {
	query := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	
	var results []User
	for _, user := range users {
		if user.Status == "active" {
			if query == "" || 
				contains(user.Username, query) ||
				contains(user.Email, query) ||
				contains(user.DisplayName, query) {
				results = append(results, user)
				if len(results) >= limit {
					break
				}
			}
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"users":   results,
		"total":   len(results),
	})
}

func getUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID无效"})
		return
	}
	
	for _, user := range users {
		if user.ID == userID {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"user":    user,
			})
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
}

func getUserPermissions(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID无效"})
		return
	}
	
	projectID, err := strconv.Atoi(c.Query("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目ID无效"})
		return
	}
	
	var permissions []string
	permissionMap := make(map[string]bool)
	
	// 查找用户在项目中的所有角色
	for _, team := range teams {
		if team.ProjectID == projectID && team.IsActive {
			for _, member := range team.Members {
				if member.UserID == userID && member.Status == "active" {
					for _, permission := range member.Role.Permissions {
						permissionMap[permission] = true
					}
				}
			}
		}
	}
	
	// 转换为切片
	for permission := range permissionMap {
		permissions = append(permissions, permission)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"permissions": permissions,
	})
}

func getRoles(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"roles":   roles,
		"total":   len(roles),
	})
}

func createRole(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	
	role := Role{
		ID:          len(roles) + 1,
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
		IsSystem:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	roles = append(roles, role)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"role":    role,
	})
}

func updateRole(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色ID无效"})
		return
	}
	
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	
	for i, role := range roles {
		if role.ID == roleID {
			if role.IsSystem {
				c.JSON(http.StatusBadRequest, gin.H{"error": "不能修改系统角色"})
				return
			}
			
			roles[i].Name = req.Name
			roles[i].Description = req.Description
			roles[i].Permissions = req.Permissions
			roles[i].UpdatedAt = time.Now()
			
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"role":    roles[i],
			})
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
}

func deleteRole(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色ID无效"})
		return
	}
	
	for i, role := range roles {
		if role.ID == roleID {
			if role.IsSystem {
				c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除系统角色"})
				return
			}
			
			// 移除角色
			roles = append(roles[:i], roles[i+1:]...)
			
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "角色已删除",
			})
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
}

func createInvitation(c *gin.Context) {
	var req struct {
		TeamID  int    `json:"team_id" binding:"required"`
		Email   string `json:"email" binding:"required"`
		RoleID  int    `json:"role_id" binding:"required"`
		Message string `json:"message"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	
	invitation := Invitation{
		ID:        invitationIDCounter,
		TeamID:    req.TeamID,
		Email:     req.Email,
		RoleID:    req.RoleID,
		Token:     generateToken(),
		Status:    "pending",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Message:   req.Message,
		InvitedBy: 1, // 临时硬编码
		CreatedAt: time.Now(),
	}
	
	invitations = append(invitations, invitation)
	invitationIDCounter++
	
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"invitation": invitation,
	})
}

func getTeamInvitations(c *gin.Context) {
	teamID, err := strconv.Atoi(c.Param("teamId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队ID无效"})
		return
	}
	
	var teamInvitations []Invitation
	for _, inv := range invitations {
		if inv.TeamID == teamID {
			teamInvitations = append(teamInvitations, inv)
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"invitations": teamInvitations,
		"total":       len(teamInvitations),
	})
}

func acceptInvitation(c *gin.Context) {
	token := c.Param("token")
	
	for i, inv := range invitations {
		if inv.Token == token && inv.Status == "pending" && time.Now().Before(inv.ExpiresAt) {
			invitations[i].Status = "accepted"
			
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "邀请已接受",
			})
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "邀请不存在或已过期"})
}

func rejectInvitation(c *gin.Context) {
	token := c.Param("token")
	
	for i, inv := range invitations {
		if inv.Token == token && inv.Status == "pending" {
			invitations[i].Status = "rejected"
			
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "邀请已拒绝",
			})
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "邀请不存在"})
}

func createPermissionRequest(c *gin.Context) {
	var req struct {
		ProjectID   int    `json:"project_id" binding:"required"`
		UserID      int    `json:"user_id" binding:"required"`
		RequestType string `json:"request_type" binding:"required"`
		TargetID    *int   `json:"target_id"`
		Permission  string `json:"permission" binding:"required"`
		Reason      string `json:"reason"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	
	request := PermissionRequest{
		ID:          requestIDCounter,
		ProjectID:   req.ProjectID,
		UserID:      req.UserID,
		RequestType: req.RequestType,
		TargetID:    req.TargetID,
		Permission:  req.Permission,
		Reason:      req.Reason,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}
	
	permissionRequests = append(permissionRequests, request)
	requestIDCounter++
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"request": request,
	})
}

func getPermissionRequests(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("projectId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目ID无效"})
		return
	}
	
	var projectRequests []PermissionRequest
	for _, req := range permissionRequests {
		if req.ProjectID == projectID {
			projectRequests = append(projectRequests, req)
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"requests": projectRequests,
		"total":    len(projectRequests),
	})
}

func reviewPermissionRequest(c *gin.Context) {
	requestID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "申请ID无效"})
		return
	}
	
	var req struct {
		Approved     bool   `json:"approved"`
		ReviewReason string `json:"review_reason"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	
	for i, request := range permissionRequests {
		if request.ID == requestID && request.Status == "pending" {
			if req.Approved {
				permissionRequests[i].Status = "approved"
			} else {
				permissionRequests[i].Status = "rejected"
			}
			
			reviewerID := 1 // 临时硬编码
			now := time.Now()
			permissionRequests[i].ReviewedBy = &reviewerID
			permissionRequests[i].ReviewedAt = &now
			
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"request": permissionRequests[i],
			})
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "权限申请不存在"})
}

// 工具函数
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		(s == substr || 
		 len(s) >= len(substr) && 
		 (s[:len(substr)] == substr || 
		  s[len(s)-len(substr):] == substr ||
		  len(s) > len(substr) && 
		  func() bool {
		  	for i := 1; i <= len(s)-len(substr); i++ {
		  		if s[i:i+len(substr)] == substr {
		  			return true
		  		}
		  	}
		  	return false
		  }()))
}

func generateToken() string {
	return fmt.Sprintf("inv_%d_%d", time.Now().Unix(), invitationIDCounter)
}
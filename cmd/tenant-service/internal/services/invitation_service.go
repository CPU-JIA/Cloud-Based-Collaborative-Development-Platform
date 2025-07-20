package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cloud-platform/collaborative-dev/cmd/tenant-service/internal/repository"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// InvitationService 邀请服务接口
type InvitationService interface {
	// 邀请管理
	SendInvitation(ctx context.Context, req *SendInvitationRequest) (*models.TenantInvitation, error)
	AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) (*models.TenantMember, error)
	DeclineInvitation(ctx context.Context, token string) error
	CancelInvitation(ctx context.Context, invitationID uuid.UUID, userID uuid.UUID) error

	// 邀请查询
	GetInvitation(ctx context.Context, id uuid.UUID) (*models.TenantInvitation, error)
	GetInvitationByToken(ctx context.Context, token string) (*models.TenantInvitation, error)
	ListInvitations(ctx context.Context, tenantID uuid.UUID, req *ListInvitationsRequest) (*ListInvitationsResponse, error)
	ListPendingInvitations(ctx context.Context, email string) ([]*models.TenantInvitation, error)

	// 成员管理
	AddMember(ctx context.Context, req *AddMemberRequest) (*models.TenantMember, error)
	RemoveMember(ctx context.Context, tenantID, userID uuid.UUID) error
	UpdateMemberRole(ctx context.Context, tenantID, userID uuid.UUID, role models.InvitationRole) error
	UpdateMemberPermissions(ctx context.Context, tenantID, userID uuid.UUID, permissions []string) error

	// 成员查询
	GetMember(ctx context.Context, tenantID, userID uuid.UUID) (*models.TenantMember, error)
	ListMembers(ctx context.Context, tenantID uuid.UUID, req *ListMembersRequest) (*ListMembersResponse, error)
	IsMember(ctx context.Context, tenantID, userID uuid.UUID) (bool, error)
	HasPermission(ctx context.Context, tenantID, userID uuid.UUID, permission string) (bool, error)

	// 批量操作
	BulkInvite(ctx context.Context, req *BulkInviteRequest) (*BulkInviteResponse, error)
	ResendInvitation(ctx context.Context, invitationID uuid.UUID) error
	CleanupExpiredInvitations(ctx context.Context) (int64, error)
}

// 请求和响应结构

// SendInvitationRequest 发送邀请请求
type SendInvitationRequest struct {
	TenantID    uuid.UUID             `json:"tenant_id" binding:"required"`
	InviterID   uuid.UUID             `json:"inviter_id" binding:"required"`
	Email       string                `json:"email" binding:"required,email"`
	Role        models.InvitationRole `json:"role" binding:"required"`
	Message     string                `json:"message"`
	Permissions []string              `json:"permissions"`
	ExpiryDays  int                   `json:"expiry_days,omitempty"` // 默认7天
}

// AddMemberRequest 直接添加成员请求
type AddMemberRequest struct {
	TenantID    uuid.UUID             `json:"tenant_id" binding:"required"`
	UserID      uuid.UUID             `json:"user_id" binding:"required"`
	Role        models.InvitationRole `json:"role" binding:"required"`
	Permissions []string              `json:"permissions"`
	InvitedBy   *uuid.UUID            `json:"invited_by"`
}

// ListInvitationsRequest 邀请列表请求
type ListInvitationsRequest struct {
	Status models.InvitationStatus `json:"status,omitempty"`
	Role   models.InvitationRole   `json:"role,omitempty"`
	Email  string                  `json:"email,omitempty"`
	Offset int                     `json:"offset"`
	Limit  int                     `json:"limit" binding:"max=100"`
}

// ListInvitationsResponse 邀请列表响应
type ListInvitationsResponse struct {
	Invitations []*models.TenantInvitation `json:"invitations"`
	Total       int64                      `json:"total"`
	Offset      int                        `json:"offset"`
	Limit       int                        `json:"limit"`
}

// ListMembersRequest 成员列表请求
type ListMembersRequest struct {
	Role            models.InvitationRole `json:"role,omitempty"`
	Status          string                `json:"status,omitempty"`
	Search          string                `json:"search,omitempty"` // 搜索邮箱或姓名
	Offset          int                   `json:"offset"`
	Limit           int                   `json:"limit" binding:"max=100"`
	IncludeInactive bool                  `json:"include_inactive,omitempty"`
}

// ListMembersResponse 成员列表响应
type ListMembersResponse struct {
	Members []*models.TenantMember `json:"members"`
	Total   int64                  `json:"total"`
	Offset  int                    `json:"offset"`
	Limit   int                    `json:"limit"`
}

// BulkInviteRequest 批量邀请请求
type BulkInviteRequest struct {
	TenantID    uuid.UUID             `json:"tenant_id" binding:"required"`
	InviterID   uuid.UUID             `json:"inviter_id" binding:"required"`
	Emails      []string              `json:"emails" binding:"required,min=1,max=50"`
	Role        models.InvitationRole `json:"role" binding:"required"`
	Message     string                `json:"message"`
	Permissions []string              `json:"permissions"`
	ExpiryDays  int                   `json:"expiry_days,omitempty"`
}

// BulkInviteResponse 批量邀请响应
type BulkInviteResponse struct {
	Successful []*models.TenantInvitation `json:"successful"`
	Failed     []BulkInviteError          `json:"failed"`
	Total      int                        `json:"total"`
	Success    int                        `json:"success"`
	Failure    int                        `json:"failure"`
}

// BulkInviteError 批量邀请错误
type BulkInviteError struct {
	Email string `json:"email"`
	Error string `json:"error"`
}

// invitationService 邀请服务实现
type invitationService struct {
	invitationRepo InvitationRepository
	memberRepo     MemberRepository
	tenantRepo     repository.TenantRepository
	userRepo       UserRepository // 假设存在用户仓储
}

// NewInvitationService 创建邀请服务实例
func NewInvitationService(
	invitationRepo InvitationRepository,
	memberRepo MemberRepository,
	tenantRepo repository.TenantRepository,
	userRepo UserRepository,
) InvitationService {
	return &invitationService{
		invitationRepo: invitationRepo,
		memberRepo:     memberRepo,
		tenantRepo:     tenantRepo,
		userRepo:       userRepo,
	}
}

// SendInvitation 发送邀请
func (s *invitationService) SendInvitation(ctx context.Context, req *SendInvitationRequest) (*models.TenantInvitation, error) {
	// 验证租户是否存在
	tenant, err := s.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("租户不存在: %w", err)
	}

	if !tenant.IsActive() {
		return nil, fmt.Errorf("租户未激活，无法发送邀请")
	}

	// 验证邀请人权限
	canInvite, err := s.HasPermission(ctx, req.TenantID, req.InviterID, "user.invite")
	if err != nil {
		return nil, fmt.Errorf("检查邀请权限失败: %w", err)
	}
	if !canInvite {
		return nil, fmt.Errorf("您没有邀请用户的权限")
	}

	// 检查是否已存在待处理的邀请
	existing, err := s.invitationRepo.GetByEmailAndTenant(ctx, req.Email, req.TenantID)
	if err == nil && existing.IsPending() {
		return nil, fmt.Errorf("该邮箱已有待处理的邀请")
	}

	// 检查用户是否已是成员
	if user, err := s.userRepo.GetByEmail(ctx, req.Email); err == nil {
		isMember, _ := s.IsMember(ctx, req.TenantID, user.ID)
		if isMember {
			return nil, fmt.Errorf("该用户已是租户成员")
		}
	}

	// 生成邀请令牌
	token, err := s.generateInvitationToken()
	if err != nil {
		return nil, fmt.Errorf("生成邀请令牌失败: %w", err)
	}

	// 设置过期时间
	expiryDays := req.ExpiryDays
	if expiryDays <= 0 {
		expiryDays = 7 // 默认7天
	}

	// 创建邀请
	invitation := &models.TenantInvitation{
		ID:          uuid.New(),
		TenantID:    req.TenantID,
		InviterID:   req.InviterID,
		Email:       strings.ToLower(req.Email),
		Role:        req.Role,
		Status:      models.InvitationStatusPending,
		Message:     req.Message,
		Permissions: req.Permissions,
		Token:       token,
		ExpiresAt:   time.Now().AddDate(0, 0, expiryDays),
	}

	if err := s.invitationRepo.Create(ctx, invitation); err != nil {
		return nil, fmt.Errorf("创建邀请失败: %w", err)
	}

	// TODO: 发送邀请邮件
	// s.emailService.SendInvitationEmail(ctx, invitation)

	return invitation, nil
}

// AcceptInvitation 接受邀请
func (s *invitationService) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) (*models.TenantMember, error) {
	// 获取邀请
	invitation, err := s.invitationRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("无效的邀请链接: %w", err)
	}

	// 检查邀请是否可以接受
	if !invitation.CanAccept() {
		return nil, fmt.Errorf("邀请已过期或无效")
	}

	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 验证邮箱匹配
	if strings.ToLower(user.Email) != strings.ToLower(invitation.Email) {
		return nil, fmt.Errorf("邮箱不匹配，无法接受邀请")
	}

	// 检查是否已是成员
	isMember, err := s.IsMember(ctx, invitation.TenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("检查成员状态失败: %w", err)
	}
	if isMember {
		return nil, fmt.Errorf("您已是该租户的成员")
	}

	// 开始事务（这里简化处理，实际应该用数据库事务）

	// 接受邀请
	if err := invitation.Accept(userID); err != nil {
		return nil, fmt.Errorf("接受邀请失败: %w", err)
	}

	if err := s.invitationRepo.Update(ctx, invitation); err != nil {
		return nil, fmt.Errorf("更新邀请状态失败: %w", err)
	}

	// 添加为成员
	member := &models.TenantMember{
		ID:           uuid.New(),
		TenantID:     invitation.TenantID,
		UserID:       userID,
		Role:         invitation.Role,
		Permissions:  invitation.Permissions,
		Status:       "active",
		JoinedAt:     time.Now(),
		InvitedBy:    &invitation.InviterID,
		InvitationID: &invitation.ID,
	}

	if err := s.memberRepo.Create(ctx, member); err != nil {
		return nil, fmt.Errorf("添加成员失败: %w", err)
	}

	return member, nil
}

// DeclineInvitation 拒绝邀请
func (s *invitationService) DeclineInvitation(ctx context.Context, token string) error {
	invitation, err := s.invitationRepo.GetByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("无效的邀请链接: %w", err)
	}

	if err := invitation.Decline(); err != nil {
		return err
	}

	return s.invitationRepo.Update(ctx, invitation)
}

// CancelInvitation 取消邀请
func (s *invitationService) CancelInvitation(ctx context.Context, invitationID uuid.UUID, userID uuid.UUID) error {
	invitation, err := s.invitationRepo.GetByID(ctx, invitationID)
	if err != nil {
		return fmt.Errorf("邀请不存在: %w", err)
	}

	// 检查权限（只有邀请人或管理员可以取消）
	if invitation.InviterID != userID {
		hasPermission, err := s.HasPermission(ctx, invitation.TenantID, userID, "user.manage")
		if err != nil {
			return fmt.Errorf("检查权限失败: %w", err)
		}
		if !hasPermission {
			return fmt.Errorf("您没有权限取消此邀请")
		}
	}

	if err := invitation.Cancel(); err != nil {
		return err
	}

	return s.invitationRepo.Update(ctx, invitation)
}

// GetInvitation 获取邀请详情
func (s *invitationService) GetInvitation(ctx context.Context, id uuid.UUID) (*models.TenantInvitation, error) {
	return s.invitationRepo.GetByID(ctx, id)
}

// GetInvitationByToken 根据令牌获取邀请
func (s *invitationService) GetInvitationByToken(ctx context.Context, token string) (*models.TenantInvitation, error) {
	return s.invitationRepo.GetByToken(ctx, token)
}

// ListInvitations 获取邀请列表
func (s *invitationService) ListInvitations(ctx context.Context, tenantID uuid.UUID, req *ListInvitationsRequest) (*ListInvitationsResponse, error) {
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	invitations, err := s.invitationRepo.ListByTenant(ctx, tenantID, req)
	if err != nil {
		return nil, err
	}

	total, err := s.invitationRepo.CountByTenant(ctx, tenantID, req)
	if err != nil {
		return nil, err
	}

	return &ListInvitationsResponse{
		Invitations: invitations,
		Total:       total,
		Offset:      req.Offset,
		Limit:       req.Limit,
	}, nil
}

// ListPendingInvitations 获取用户的待处理邀请
func (s *invitationService) ListPendingInvitations(ctx context.Context, email string) ([]*models.TenantInvitation, error) {
	return s.invitationRepo.GetPendingByEmail(ctx, strings.ToLower(email))
}

// AddMember 直接添加成员
func (s *invitationService) AddMember(ctx context.Context, req *AddMemberRequest) (*models.TenantMember, error) {
	// 验证租户和用户是否存在
	if _, err := s.tenantRepo.GetByID(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("租户不存在: %w", err)
	}

	if _, err := s.userRepo.GetByID(ctx, req.UserID); err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 检查是否已是成员
	isMember, err := s.IsMember(ctx, req.TenantID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("检查成员状态失败: %w", err)
	}
	if isMember {
		return nil, fmt.Errorf("用户已是租户成员")
	}

	// 创建成员
	member := &models.TenantMember{
		ID:          uuid.New(),
		TenantID:    req.TenantID,
		UserID:      req.UserID,
		Role:        req.Role,
		Permissions: req.Permissions,
		Status:      "active",
		JoinedAt:    time.Now(),
		InvitedBy:   req.InvitedBy,
	}

	if err := s.memberRepo.Create(ctx, member); err != nil {
		return nil, fmt.Errorf("添加成员失败: %w", err)
	}

	return member, nil
}

// IsMember 检查是否为成员
func (s *invitationService) IsMember(ctx context.Context, tenantID, userID uuid.UUID) (bool, error) {
	_, err := s.memberRepo.GetByTenantAndUser(ctx, tenantID, userID)
	if err != nil {
		return false, nil // 不存在返回false，而不是错误
	}
	return true, nil
}

// HasPermission 检查权限
func (s *invitationService) HasPermission(ctx context.Context, tenantID, userID uuid.UUID, permission string) (bool, error) {
	member, err := s.memberRepo.GetByTenantAndUser(ctx, tenantID, userID)
	if err != nil {
		return false, nil // 不是成员，没有权限
	}

	return member.HasPermission(permission), nil
}

// generateInvitationToken 生成邀请令牌
func (s *invitationService) generateInvitationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// BulkInvite 批量邀请
func (s *invitationService) BulkInvite(ctx context.Context, req *BulkInviteRequest) (*BulkInviteResponse, error) {
	response := &BulkInviteResponse{
		Total: len(req.Emails),
	}

	for _, email := range req.Emails {
		inviteReq := &SendInvitationRequest{
			TenantID:    req.TenantID,
			InviterID:   req.InviterID,
			Email:       email,
			Role:        req.Role,
			Message:     req.Message,
			Permissions: req.Permissions,
			ExpiryDays:  req.ExpiryDays,
		}

		invitation, err := s.SendInvitation(ctx, inviteReq)
		if err != nil {
			response.Failed = append(response.Failed, BulkInviteError{
				Email: email,
				Error: err.Error(),
			})
			response.Failure++
		} else {
			response.Successful = append(response.Successful, invitation)
			response.Success++
		}
	}

	return response, nil
}

// CleanupExpiredInvitations 清理过期邀请
func (s *invitationService) CleanupExpiredInvitations(ctx context.Context) (int64, error) {
	return s.invitationRepo.MarkExpiredInvitations(ctx)
}

// 其他接口实现...
func (s *invitationService) RemoveMember(ctx context.Context, tenantID, userID uuid.UUID) error {
	return s.memberRepo.Delete(ctx, tenantID, userID)
}

func (s *invitationService) UpdateMemberRole(ctx context.Context, tenantID, userID uuid.UUID, role models.InvitationRole) error {
	member, err := s.memberRepo.GetByTenantAndUser(ctx, tenantID, userID)
	if err != nil {
		return err
	}

	member.Role = role
	return s.memberRepo.Update(ctx, member)
}

func (s *invitationService) UpdateMemberPermissions(ctx context.Context, tenantID, userID uuid.UUID, permissions []string) error {
	member, err := s.memberRepo.GetByTenantAndUser(ctx, tenantID, userID)
	if err != nil {
		return err
	}

	member.Permissions = permissions
	return s.memberRepo.Update(ctx, member)
}

func (s *invitationService) GetMember(ctx context.Context, tenantID, userID uuid.UUID) (*models.TenantMember, error) {
	return s.memberRepo.GetByTenantAndUser(ctx, tenantID, userID)
}

func (s *invitationService) ListMembers(ctx context.Context, tenantID uuid.UUID, req *ListMembersRequest) (*ListMembersResponse, error) {
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	members, err := s.memberRepo.ListByTenant(ctx, tenantID, req)
	if err != nil {
		return nil, err
	}

	total, err := s.memberRepo.CountByTenant(ctx, tenantID, req)
	if err != nil {
		return nil, err
	}

	return &ListMembersResponse{
		Members: members,
		Total:   total,
		Offset:  req.Offset,
		Limit:   req.Limit,
	}, nil
}

func (s *invitationService) ResendInvitation(ctx context.Context, invitationID uuid.UUID) error {
	invitation, err := s.invitationRepo.GetByID(ctx, invitationID)
	if err != nil {
		return err
	}

	if invitation.Status != models.InvitationStatusPending {
		return fmt.Errorf("只能重发待处理的邀请")
	}

	// TODO: 重新发送邀请邮件
	// return s.emailService.SendInvitationEmail(ctx, invitation)

	return nil
}

// Repository接口定义（这些应该在repository包中实现）
type InvitationRepository interface {
	Create(ctx context.Context, invitation *models.TenantInvitation) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.TenantInvitation, error)
	GetByToken(ctx context.Context, token string) (*models.TenantInvitation, error)
	GetByEmailAndTenant(ctx context.Context, email string, tenantID uuid.UUID) (*models.TenantInvitation, error)
	Update(ctx context.Context, invitation *models.TenantInvitation) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, req *ListInvitationsRequest) ([]*models.TenantInvitation, error)
	CountByTenant(ctx context.Context, tenantID uuid.UUID, req *ListInvitationsRequest) (int64, error)
	GetPendingByEmail(ctx context.Context, email string) ([]*models.TenantInvitation, error)
	MarkExpiredInvitations(ctx context.Context) (int64, error)
}

type MemberRepository interface {
	Create(ctx context.Context, member *models.TenantMember) error
	GetByTenantAndUser(ctx context.Context, tenantID, userID uuid.UUID) (*models.TenantMember, error)
	Update(ctx context.Context, member *models.TenantMember) error
	Delete(ctx context.Context, tenantID, userID uuid.UUID) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, req *ListMembersRequest) ([]*models.TenantMember, error)
	CountByTenant(ctx context.Context, tenantID uuid.UUID, req *ListMembersRequest) (int64, error)
}

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
}

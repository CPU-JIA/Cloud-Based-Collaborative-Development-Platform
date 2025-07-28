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

// Document 文档结构
type Document struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	AuthorID    int       `json:"author_id"`
	AuthorEmail string    `json:"author_email"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int       `json:"version"`
	IsPublic    bool      `json:"is_public"`
	Tags        []string  `json:"tags"`
	Category    string    `json:"category"`
}

// DocumentVersion 文档版本历史
type DocumentVersion struct {
	ID         int       `json:"id"`
	DocumentID int       `json:"document_id"`
	Version    int       `json:"version"`
	Content    string    `json:"content"`
	AuthorID   int       `json:"author_id"`
	CreatedAt  time.Time `json:"created_at"`
	ChangeLog  string    `json:"change_log"`
}

// CursorPosition 光标位置
type CursorPosition struct {
	UserID int `json:"user_id"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

// EditOperation 实时编辑操作
type EditOperation struct {
	Type       string    `json:"type"` // insert, delete, replace
	Position   int       `json:"position"`
	Content    string    `json:"content"`
	Length     int       `json:"length"`
	UserID     int       `json:"user_id"`
	Timestamp  time.Time `json:"timestamp"`
	DocumentID int       `json:"document_id"`
}

// MockKnowledgeBaseService 模拟知识库服务
type MockKnowledgeBaseService struct {
	documents map[int]*Document
	versions  map[int][]*DocumentVersion
	nextID    int
}

// NewMockKnowledgeBaseService 创建模拟知识库服务
func NewMockKnowledgeBaseService() *MockKnowledgeBaseService {
	return &MockKnowledgeBaseService{
		documents: make(map[int]*Document),
		versions:  make(map[int][]*DocumentVersion),
		nextID:    1,
	}
}

// CreateDocument 创建文档
func (s *MockKnowledgeBaseService) CreateDocument(title, content, category string, tags []string, isPublic bool, authorID int, authorEmail string) (*Document, error) {
	if err := validateDocumentTitle(title); err != nil {
		return nil, err
	}
	if err := validateDocumentContent(content); err != nil {
		return nil, err
	}
	if err := validateDocumentCategory(category); err != nil {
		return nil, err
	}
	if err := validateDocumentTags(tags); err != nil {
		return nil, err
	}
	if err := validateAuthorID(authorID); err != nil {
		return nil, err
	}
	if err := kbValidateEmail(authorEmail); err != nil {
		return nil, err
	}

	doc := &Document{
		ID:          s.nextID,
		Title:       title,
		Content:     content,
		AuthorID:    authorID,
		AuthorEmail: authorEmail,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
		IsPublic:    isPublic,
		Tags:        tags,
		Category:    category,
	}

	s.documents[s.nextID] = doc

	// 创建初始版本
	version := &DocumentVersion{
		ID:         s.nextID*100 + 1,
		DocumentID: s.nextID,
		Version:    1,
		Content:    content,
		AuthorID:   authorID,
		CreatedAt:  time.Now(),
		ChangeLog:  "初始版本",
	}
	s.versions[s.nextID] = []*DocumentVersion{version}

	s.nextID++
	return doc, nil
}

// GetDocument 获取文档详情
func (s *MockKnowledgeBaseService) GetDocument(id, userID int) (*Document, error) {
	doc, exists := s.documents[id]
	if !exists {
		return nil, fmt.Errorf("文档不存在")
	}

	// 权限检查
	if !doc.IsPublic && doc.AuthorID != userID {
		return nil, fmt.Errorf("无权限访问此文档")
	}

	return doc, nil
}

// UpdateDocument 更新文档
func (s *MockKnowledgeBaseService) UpdateDocument(id int, title, content, changeLog string, userID int) (*Document, error) {
	doc, exists := s.documents[id]
	if !exists {
		return nil, fmt.Errorf("文档不存在")
	}

	// 权限检查
	if doc.AuthorID != userID {
		return nil, fmt.Errorf("无权限修改此文档")
	}

	if err := validateDocumentTitle(title); err != nil {
		return nil, err
	}
	if err := validateDocumentContent(content); err != nil {
		return nil, err
	}

	// 创建新版本
	doc.Version++
	doc.Title = title
	doc.Content = content
	doc.UpdatedAt = time.Now()

	// 保存版本历史
	version := &DocumentVersion{
		ID:         id*100 + doc.Version,
		DocumentID: id,
		Version:    doc.Version,
		Content:    content,
		AuthorID:   userID,
		CreatedAt:  time.Now(),
		ChangeLog:  changeLog,
	}
	s.versions[id] = append(s.versions[id], version)

	return doc, nil
}

// GetDocumentVersions 获取文档版本历史
func (s *MockKnowledgeBaseService) GetDocumentVersions(docID, userID int) ([]*DocumentVersion, error) {
	doc, exists := s.documents[docID]
	if !exists {
		return nil, fmt.Errorf("文档不存在")
	}

	// 权限检查
	if !doc.IsPublic && doc.AuthorID != userID {
		return nil, fmt.Errorf("无权限访问此文档")
	}

	versions, exists := s.versions[docID]
	if !exists {
		return []*DocumentVersion{}, nil
	}

	return versions, nil
}

// DeleteDocument 删除文档
func (s *MockKnowledgeBaseService) DeleteDocument(id, userID int) error {
	doc, exists := s.documents[id]
	if !exists {
		return fmt.Errorf("文档不存在")
	}

	if doc.AuthorID != userID {
		return fmt.Errorf("无权限删除此文档")
	}

	delete(s.documents, id)
	delete(s.versions, id)
	return nil
}

// GetDocuments 获取文档列表
func (s *MockKnowledgeBaseService) GetDocuments(userID int, page, pageSize int, category, tag string) ([]*Document, int) {
	var docs []*Document
	for _, doc := range s.documents {
		// 只返回公开文档或用户自己的文档
		if doc.IsPublic || doc.AuthorID == userID {
			// 过滤条件
			if category != "" && doc.Category != category {
				continue
			}
			if tag != "" {
				hasTag := false
				for _, t := range doc.Tags {
					if t == tag {
						hasTag = true
						break
					}
				}
				if !hasTag {
					continue
				}
			}
			docs = append(docs, doc)
		}
	}

	// 简单分页
	total := len(docs)
	start := (page - 1) * pageSize
	end := start + pageSize

	if start > total {
		return []*Document{}, total
	}
	if end > total {
		end = total
	}

	return docs[start:end], total
}

// ================================
// 验证函数
// ================================

// validateDocumentTitle 验证文档标题
func validateDocumentTitle(title string) error {
	if title == "" {
		return fmt.Errorf("文档标题不能为空")
	}
	if len(title) > 200 {
		return fmt.Errorf("文档标题不能超过200个字符")
	}
	// 标题不能包含特殊控制字符
	if strings.ContainsAny(title, "\r\n\t") {
		return fmt.Errorf("文档标题不能包含控制字符")
	}
	return nil
}

// validateDocumentContent 验证文档内容
func validateDocumentContent(content string) error {
	if content == "" {
		return fmt.Errorf("文档内容不能为空")
	}
	if len(content) > 1000000 { // 1MB限制
		return fmt.Errorf("文档内容不能超过1MB")
	}
	return nil
}

// validateDocumentCategory 验证文档分类
func validateDocumentCategory(category string) error {
	if category == "" {
		return nil // 分类可选
	}
	if len(category) > 50 {
		return fmt.Errorf("文档分类不能超过50个字符")
	}
	// 分类名称格式检查
	categoryPattern := regexp.MustCompile(`^[\w\s\x{4e00}-\x{9fff}-]+$`)
	if !categoryPattern.MatchString(category) {
		return fmt.Errorf("文档分类格式无效")
	}
	return nil
}

// validateDocumentTags 验证文档标签
func validateDocumentTags(tags []string) error {
	if len(tags) > 10 {
		return fmt.Errorf("文档标签不能超过10个")
	}
	for _, tag := range tags {
		if len(tag) == 0 {
			return fmt.Errorf("标签不能为空")
		}
		if len(tag) > 30 {
			return fmt.Errorf("单个标签不能超过30个字符")
		}
		// 标签格式检查
		tagPattern := regexp.MustCompile(`^[\w\x{4e00}-\x{9fff}-]+$`)
		if !tagPattern.MatchString(tag) {
			return fmt.Errorf("标签格式无效: %s", tag)
		}
	}
	return nil
}

// validateAuthorID 验证作者ID
func validateAuthorID(authorID int) error {
	if authorID <= 0 {
		return fmt.Errorf("无效的作者ID")
	}
	return nil
}

// validateEmail 验证邮箱
func kbValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}
	emailPattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailPattern.MatchString(email) {
		return fmt.Errorf("邮箱格式无效")
	}
	return nil
}

// validateEditOperation 验证编辑操作
func validateEditOperation(op *EditOperation) error {
	validTypes := []string{"insert", "delete", "replace"}
	typeValid := false
	for _, validType := range validTypes {
		if op.Type == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return fmt.Errorf("无效的编辑操作类型: %s", op.Type)
	}

	if op.Position < 0 {
		return fmt.Errorf("编辑位置不能为负数")
	}

	if op.Type == "delete" && op.Length <= 0 {
		return fmt.Errorf("删除操作必须指定长度")
	}

	if op.UserID <= 0 {
		return fmt.Errorf("无效的用户ID")
	}

	return nil
}

// validateCursorPosition 验证光标位置
func validateCursorPosition(cursor *CursorPosition) error {
	if cursor.UserID <= 0 {
		return fmt.Errorf("无效的用户ID")
	}
	if cursor.Line < 0 {
		return fmt.Errorf("行号不能为负数")
	}
	if cursor.Column < 0 {
		return fmt.Errorf("列号不能为负数")
	}
	return nil
}

// ================================
// 单元测试 - 验证函数测试
// ================================

func TestValidateDocumentTitle(t *testing.T) {
	testCases := []struct {
		name        string
		title       string
		expectError bool
		errorMsg    string
	}{
		{"有效标题", "项目文档", false, ""},
		{"有效长标题", "企业协作开发平台技术文档v1.0", false, ""},
		{"空标题", "", true, "文档标题不能为空"},
		{"超长标题", strings.Repeat("a", 201), true, "文档标题不能超过200个字符"},
		{"包含换行符", "标题\n换行", true, "文档标题不能包含控制字符"},
		{"包含制表符", "标题\t制表", true, "文档标题不能包含控制字符"},
		{"包含回车符", "标题\r回车", true, "文档标题不能包含控制字符"},
		{"边界值-最大长度", strings.Repeat("a", 200), false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDocumentTitle(tc.title)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDocumentContent(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{"有效内容", "# 标题\n\n这是文档内容。", false, ""},
		{"空内容", "", true, "文档内容不能为空"},
		{"超大内容", strings.Repeat("a", 1000001), true, "文档内容不能超过1MB"},
		{"边界值-最大长度", strings.Repeat("a", 1000000), false, ""},
		{"Markdown内容", "# 标题\n\n## 子标题\n\n- 列表项1\n- 列表项2", false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDocumentContent(tc.content)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDocumentCategory(t *testing.T) {
	testCases := []struct {
		name        string
		category    string
		expectError bool
	}{
		{"有效分类", "技术文档", false},
		{"空分类", "", false}, // 分类可选
		{"英文分类", "Technical-Docs", false},
		{"超长分类", strings.Repeat("a", 51), true},
		{"特殊字符", "分类@#$", true},
		{"边界值-最大长度", strings.Repeat("a", 50), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDocumentCategory(tc.category)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDocumentTags(t *testing.T) {
	testCases := []struct {
		name        string
		tags        []string
		expectError bool
		errorMsg    string
	}{
		{"有效标签", []string{"golang", "web开发"}, false, ""},
		{"空标签列表", []string{}, false, ""},
		{"单个标签", []string{"技术"}, false, ""},
		{"超过10个标签", []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}, true, "文档标签不能超过10个"},
		{"空标签项", []string{"valid", ""}, true, "标签不能为空"},
		{"超长标签", []string{strings.Repeat("a", 31)}, true, "单个标签不能超过30个字符"},
		{"特殊字符标签", []string{"tag@#"}, true, "标签格式无效"},
		{"边界值-10个标签", []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}, false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDocumentTags(tc.tags)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAuthorID(t *testing.T) {
	testCases := []struct {
		name        string
		authorID    int
		expectError bool
	}{
		{"有效作者ID", 123, false},
		{"零值作者ID", 0, true},
		{"负数作者ID", -1, true},
		{"大数值作者ID", 999999, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAuthorID(tc.authorID)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEditOperation(t *testing.T) {
	testCases := []struct {
		name        string
		operation   EditOperation
		expectError bool
		errorMsg    string
	}{
		{
			"有效插入操作",
			EditOperation{Type: "insert", Position: 10, Content: "text", UserID: 1},
			false, "",
		},
		{
			"有效删除操作",
			EditOperation{Type: "delete", Position: 5, Length: 3, UserID: 1},
			false, "",
		},
		{
			"有效替换操作",
			EditOperation{Type: "replace", Position: 0, Content: "new", Length: 3, UserID: 1},
			false, "",
		},
		{
			"无效操作类型",
			EditOperation{Type: "invalid", Position: 0, UserID: 1},
			true, "无效的编辑操作类型",
		},
		{
			"负数位置",
			EditOperation{Type: "insert", Position: -1, UserID: 1},
			true, "编辑位置不能为负数",
		},
		{
			"删除操作无长度",
			EditOperation{Type: "delete", Position: 0, Length: 0, UserID: 1},
			true, "删除操作必须指定长度",
		},
		{
			"无效用户ID",
			EditOperation{Type: "insert", Position: 0, UserID: 0},
			true, "无效的用户ID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEditOperation(&tc.operation)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCursorPosition(t *testing.T) {
	testCases := []struct {
		name        string
		cursor      CursorPosition
		expectError bool
		errorMsg    string
	}{
		{"有效光标位置", CursorPosition{UserID: 1, Line: 10, Column: 5}, false, ""},
		{"起始位置", CursorPosition{UserID: 1, Line: 0, Column: 0}, false, ""},
		{"无效用户ID", CursorPosition{UserID: 0, Line: 1, Column: 1}, true, "无效的用户ID"},
		{"负数行号", CursorPosition{UserID: 1, Line: -1, Column: 0}, true, "行号不能为负数"},
		{"负数列号", CursorPosition{UserID: 1, Line: 0, Column: -1}, true, "列号不能为负数"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCursorPosition(&tc.cursor)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ================================
// 单元测试 - 知识库服务测试
// ================================

func TestCreateDocument(t *testing.T) {
	service := NewMockKnowledgeBaseService()

	t.Run("成功创建文档", func(t *testing.T) {
		doc, err := service.CreateDocument(
			"测试文档",
			"# 测试内容\n\n这是一个测试文档。",
			"技术文档",
			[]string{"测试", "文档"},
			true,
			1,
			"test@example.com",
		)

		require.NoError(t, err)
		assert.Equal(t, 1, doc.ID)
		assert.Equal(t, "测试文档", doc.Title)
		assert.Equal(t, 1, doc.Version)
		assert.True(t, doc.IsPublic)
		assert.Equal(t, []string{"测试", "文档"}, doc.Tags)
		assert.Equal(t, "技术文档", doc.Category)

		// 验证版本历史
		versions := service.versions[doc.ID]
		assert.Len(t, versions, 1)
		assert.Equal(t, "初始版本", versions[0].ChangeLog)
	})

	t.Run("无效标题", func(t *testing.T) {
		_, err := service.CreateDocument("", "content", "category", []string{}, true, 1, "test@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文档标题不能为空")
	})

	t.Run("无效内容", func(t *testing.T) {
		_, err := service.CreateDocument("title", "", "category", []string{}, true, 1, "test@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文档内容不能为空")
	})

	t.Run("无效标签", func(t *testing.T) {
		invalidTags := make([]string, 11) // 超过10个标签
		for i := range invalidTags {
			invalidTags[i] = fmt.Sprintf("tag%d", i)
		}
		_, err := service.CreateDocument("title", "content", "category", invalidTags, true, 1, "test@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文档标签不能超过10个")
	})
}

func TestGetDocument(t *testing.T) {
	service := NewMockKnowledgeBaseService()

	// 创建测试文档
	doc, _ := service.CreateDocument("测试文档", "内容", "分类", []string{"标签"}, true, 1, "author@example.com")
	privateDoc, _ := service.CreateDocument("私有文档", "私有内容", "分类", []string{}, false, 1, "author@example.com")

	t.Run("成功获取公开文档", func(t *testing.T) {
		retrieved, err := service.GetDocument(doc.ID, 2) // 其他用户
		require.NoError(t, err)
		assert.Equal(t, doc.Title, retrieved.Title)
	})

	t.Run("作者获取私有文档", func(t *testing.T) {
		retrieved, err := service.GetDocument(privateDoc.ID, 1) // 作者本人
		require.NoError(t, err)
		assert.Equal(t, privateDoc.Title, retrieved.Title)
	})

	t.Run("无权限访问私有文档", func(t *testing.T) {
		_, err := service.GetDocument(privateDoc.ID, 2) // 其他用户
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无权限访问此文档")
	})

	t.Run("文档不存在", func(t *testing.T) {
		_, err := service.GetDocument(999, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文档不存在")
	})
}

func TestUpdateDocument(t *testing.T) {
	service := NewMockKnowledgeBaseService()

	// 创建测试文档
	doc, _ := service.CreateDocument("原始标题", "原始内容", "分类", []string{"标签"}, true, 1, "author@example.com")

	t.Run("成功更新文档", func(t *testing.T) {
		updated, err := service.UpdateDocument(doc.ID, "更新标题", "更新内容", "更新了标题和内容", 1)

		require.NoError(t, err)
		assert.Equal(t, "更新标题", updated.Title)
		assert.Equal(t, "更新内容", updated.Content)
		assert.Equal(t, 2, updated.Version)
		assert.True(t, updated.UpdatedAt.After(updated.CreatedAt))

		// 验证版本历史
		versions := service.versions[doc.ID]
		assert.Len(t, versions, 2)
		assert.Equal(t, "更新了标题和内容", versions[1].ChangeLog)
	})

	t.Run("无权限更新文档", func(t *testing.T) {
		_, err := service.UpdateDocument(doc.ID, "新标题", "新内容", "无权限更新", 2) // 其他用户
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无权限修改此文档")
	})

	t.Run("文档不存在", func(t *testing.T) {
		_, err := service.UpdateDocument(999, "标题", "内容", "更新日志", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文档不存在")
	})
}

func TestGetDocumentVersions(t *testing.T) {
	service := NewMockKnowledgeBaseService()

	// 创建测试文档并更新几次
	doc, _ := service.CreateDocument("文档", "初始内容", "分类", []string{}, true, 1, "author@example.com")
	service.UpdateDocument(doc.ID, "文档", "第二版内容", "更新1", 1)
	service.UpdateDocument(doc.ID, "文档", "第三版内容", "更新2", 1)

	t.Run("成功获取版本历史", func(t *testing.T) {
		versions, err := service.GetDocumentVersions(doc.ID, 1)

		require.NoError(t, err)
		assert.Len(t, versions, 3)
		assert.Equal(t, "初始版本", versions[0].ChangeLog)
		assert.Equal(t, "更新1", versions[1].ChangeLog)
		assert.Equal(t, "更新2", versions[2].ChangeLog)
	})

	t.Run("无权限访问版本历史", func(t *testing.T) {
		// 创建私有文档
		privateDoc, _ := service.CreateDocument("私有", "内容", "分类", []string{}, false, 1, "author@example.com")

		_, err := service.GetDocumentVersions(privateDoc.ID, 2) // 其他用户
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无权限访问此文档")
	})
}

func TestDeleteDocument(t *testing.T) {
	service := NewMockKnowledgeBaseService()

	// 创建测试文档
	doc, _ := service.CreateDocument("待删除文档", "内容", "分类", []string{}, true, 1, "author@example.com")

	t.Run("成功删除文档", func(t *testing.T) {
		err := service.DeleteDocument(doc.ID, 1)
		require.NoError(t, err)

		// 验证文档已删除
		_, err = service.GetDocument(doc.ID, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文档不存在")
	})

	t.Run("无权限删除文档", func(t *testing.T) {
		doc2, _ := service.CreateDocument("文档2", "内容", "分类", []string{}, true, 1, "author@example.com")

		err := service.DeleteDocument(doc2.ID, 2) // 其他用户
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无权限删除此文档")
	})

	t.Run("删除不存在文档", func(t *testing.T) {
		err := service.DeleteDocument(999, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文档不存在")
	})
}

func TestGetDocuments(t *testing.T) {
	service := NewMockKnowledgeBaseService()

	// 创建多个测试文档
	service.CreateDocument("公开文档1", "内容1", "技术文档", []string{"golang", "web"}, true, 1, "user1@example.com")
	service.CreateDocument("公开文档2", "内容2", "技术文档", []string{"react", "web"}, true, 2, "user2@example.com")
	service.CreateDocument("私有文档", "私有内容", "个人笔记", []string{"私人"}, false, 1, "user1@example.com")
	service.CreateDocument("产品文档", "产品说明", "产品文档", []string{"产品", "说明"}, true, 3, "user3@example.com")

	t.Run("获取所有可访问文档", func(t *testing.T) {
		docs, total := service.GetDocuments(1, 1, 10, "", "")

		assert.Equal(t, 4, total) // user1能看到所有公开文档+自己的私有文档
		assert.Len(t, docs, 4)
	})

	t.Run("其他用户只能看到公开文档", func(t *testing.T) {
		docs, total := service.GetDocuments(2, 1, 10, "", "")

		assert.Equal(t, 3, total) // user2只能看到公开文档
		assert.Len(t, docs, 3)
	})

	t.Run("按分类过滤", func(t *testing.T) {
		docs, total := service.GetDocuments(1, 1, 10, "技术文档", "")

		assert.Equal(t, 2, total)
		assert.Len(t, docs, 2)
	})

	t.Run("按标签过滤", func(t *testing.T) {
		docs, total := service.GetDocuments(1, 1, 10, "", "web")

		assert.Equal(t, 2, total)
		assert.Len(t, docs, 2)
	})

	t.Run("分页测试", func(t *testing.T) {
		docs, total := service.GetDocuments(1, 1, 2, "", "")

		assert.Equal(t, 4, total)
		assert.Len(t, docs, 2)

		docs2, _ := service.GetDocuments(1, 2, 2, "", "")
		assert.Len(t, docs2, 2)
	})
}

// ================================
// 性能测试
// ================================

func BenchmarkValidateDocumentTitle(b *testing.B) {
	title := "企业协作开发平台技术文档v1.0"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		validateDocumentTitle(title)
	}
}

func BenchmarkValidateDocumentTags(b *testing.B) {
	tags := []string{"golang", "web开发", "微服务", "API", "文档"}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		validateDocumentTags(tags)
	}
}

func BenchmarkCreateDocument(b *testing.B) {
	service := NewMockKnowledgeBaseService()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		service.CreateDocument(
			fmt.Sprintf("文档_%d", i),
			"# 标题\n\n这是测试内容。",
			"技术文档",
			[]string{"测试"},
			true,
			1,
			"test@example.com",
		)
	}
}

func BenchmarkUpdateDocument(b *testing.B) {
	service := NewMockKnowledgeBaseService()
	doc, _ := service.CreateDocument("基准测试文档", "初始内容", "测试", []string{}, true, 1, "test@example.com")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		service.UpdateDocument(doc.ID, "更新文档", fmt.Sprintf("更新内容_%d", i), fmt.Sprintf("更新_%d", i), 1)
	}
}

// ================================
// 边缘情况和错误场景测试
// ================================

func TestKBEdgeCases(t *testing.T) {
	service := NewMockKnowledgeBaseService()

	t.Run("极长文档内容边界测试", func(t *testing.T) {
		// 接近1MB的内容
		longContent := strings.Repeat("a", 999999)
		_, err := service.CreateDocument("大文档", longContent, "测试", []string{}, true, 1, "test@example.com")
		assert.NoError(t, err)

		// 超过1MB的内容
		tooLongContent := strings.Repeat("a", 1000001)
		_, err = service.CreateDocument("超大文档", tooLongContent, "测试", []string{}, true, 1, "test@example.com")
		assert.Error(t, err)
	})

	t.Run("特殊Markdown内容测试", func(t *testing.T) {
		markdownContent := `# 主标题

## 二级标题

### 代码块
` + "```go" + `
func main() {
    fmt.Println("Hello, World!")
}
` + "```" + `

### 表格
| 列1 | 列2 | 列3 |
|-----|-----|-----|
| 值1 | 值2 | 值3 |

### 链接和图片
[链接](https://example.com)
![图片](https://example.com/image.png)

### 列表
- 项目1
- 项目2
  - 子项目1
  - 子项目2

1. 有序列表1
2. 有序列表2

> 引用内容
> 多行引用

**粗体** 和 *斜体* 文本

---

分割线上方
分割线下方`

		_, err := service.CreateDocument("Markdown测试", markdownContent, "测试", []string{"markdown"}, true, 1, "test@example.com")
		assert.NoError(t, err)
	})

	t.Run("Unicode字符处理", func(t *testing.T) {
		unicodeTitle := "📚 知识库文档 🚀"
		unicodeContent := "包含各种Unicode字符：😀 🎉 🔥 ⭐ 💻\n\n中文、English、日本語、한국어 混合内容"
		unicodeTags := []string{"emoji", "多语言", "unicode"}

		_, err := service.CreateDocument(unicodeTitle, unicodeContent, "国际化", unicodeTags, true, 1, "test@example.com")
		assert.NoError(t, err)
	})

	t.Run("并发文档操作", func(t *testing.T) {
		// 创建基础文档
		doc, _ := service.CreateDocument("并发测试", "初始内容", "测试", []string{}, true, 1, "test@example.com")

		// 模拟并发更新
		for i := 0; i < 10; i++ {
			_, err := service.UpdateDocument(doc.ID, "并发更新", fmt.Sprintf("更新内容_%d", i), fmt.Sprintf("并发更新_%d", i), 1)
			assert.NoError(t, err)
		}

		// 验证版本数量
		versions, _ := service.GetDocumentVersions(doc.ID, 1)
		assert.Equal(t, 11, len(versions)) // 初始版本 + 10次更新
	})
}

// ================================
// 集成测试场景
// ================================

func TestKnowledgeBaseWorkflow(t *testing.T) {
	service := NewMockKnowledgeBaseService()

	t.Run("完整知识库管理流程", func(t *testing.T) {
		// 1. 创建文档
		doc, err := service.CreateDocument(
			"API设计指南",
			"# API设计指南\n\n## RESTful设计原则\n\n1. 使用名词表示资源\n2. 使用HTTP动词表示操作",
			"技术文档",
			[]string{"API", "设计", "RESTful"},
			true,
			1,
			"architect@company.com",
		)
		require.NoError(t, err)

		// 2. 其他用户查看文档
		retrieved, err := service.GetDocument(doc.ID, 2)
		require.NoError(t, err)
		assert.Equal(t, doc.Title, retrieved.Title)

		// 3. 作者更新文档
		updated, err := service.UpdateDocument(
			doc.ID,
			"API设计指南v2.0",
			"# API设计指南v2.0\n\n## RESTful设计原则\n\n1. 使用名词表示资源\n2. 使用HTTP动词表示操作\n3. 使用HTTP状态码表示结果",
			"添加了HTTP状态码部分",
			1,
		)
		require.NoError(t, err)
		assert.Equal(t, 2, updated.Version)

		// 4. 查看版本历史
		versions, err := service.GetDocumentVersions(doc.ID, 1)
		require.NoError(t, err)
		assert.Len(t, versions, 2)

		// 5. 再次更新
		service.UpdateDocument(doc.ID, "API设计指南v3.0", "添加了更多内容...", "major update", 1)

		// 6. 获取文档列表
		docs, total := service.GetDocuments(2, 1, 10, "", "")
		assert.Equal(t, 1, total)
		assert.Equal(t, "API设计指南v3.0", docs[0].Title)

		// 7. 按标签搜索
		docs, total = service.GetDocuments(2, 1, 10, "", "API")
		assert.Equal(t, 1, total)
		assert.Contains(t, docs[0].Tags, "API")
	})
}

// ================================
// 性能基准测试
// ================================

func TestKBPerformanceRequirements(t *testing.T) {
	service := NewMockKnowledgeBaseService()

	t.Run("文档标题验证性能", func(t *testing.T) {
		title := "企业协作开发平台知识库管理系统技术文档v1.0"

		start := time.Now()
		for i := 0; i < 1000; i++ {
			validateDocumentTitle(title)
		}
		duration := time.Since(start)

		// 1000次验证应该在10ms内完成
		assert.Less(t, duration, 10*time.Millisecond, "文档标题验证性能不达标")
	})

	t.Run("文档标签验证性能", func(t *testing.T) {
		tags := []string{"API", "文档", "设计", "RESTful", "微服务", "架构", "开发", "指南"}

		start := time.Now()
		for i := 0; i < 1000; i++ {
			validateDocumentTags(tags)
		}
		duration := time.Since(start)

		// 1000次验证应该在20ms内完成
		assert.Less(t, duration, 20*time.Millisecond, "文档标签验证性能不达标")
	})

	t.Run("大量文档创建性能", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < 100; i++ {
			service.CreateDocument(
				fmt.Sprintf("性能测试文档_%d", i),
				"# 标题\n\n这是性能测试的内容。",
				"性能测试",
				[]string{"测试"},
				true,
				1,
				"perf@example.com",
			)
		}
		duration := time.Since(start)

		// 创建100个文档应该在100ms内完成
		assert.Less(t, duration, 100*time.Millisecond, "批量创建文档性能不达标")
	})

	t.Run("文档检索性能", func(t *testing.T) {
		// 先创建一些文档
		for i := 0; i < 50; i++ {
			service.CreateDocument(
				fmt.Sprintf("检索测试_%d", i),
				"内容",
				"测试",
				[]string{"检索"},
				true,
				1,
				"search@example.com",
			)
		}

		start := time.Now()
		for i := 0; i < 100; i++ {
			service.GetDocuments(1, 1, 10, "", "")
		}
		duration := time.Since(start)

		// 100次检索应该在50ms内完成
		assert.Less(t, duration, 50*time.Millisecond, "文档检索性能不达标")
	})
}

package unit

import (
	"crypto/md5"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SimpleFile 简化的文件模型用于测试
type SimpleFile struct {
	ID            int       `json:"id"`
	TenantID      string    `json:"tenant_id"`
	ProjectID     int       `json:"project_id"`
	Name          string    `json:"name"`
	OriginalName  string    `json:"original_name"`
	Path          string    `json:"path"`
	Size          int64     `json:"size"`
	MimeType      string    `json:"mime_type"`
	Extension     string    `json:"extension"`
	Hash          string    `json:"hash"`
	FolderID      *int      `json:"folder_id"`
	Tags          []string  `json:"tags"`
	Description   string    `json:"description"`
	Version       int       `json:"version"`
	IsLatest      bool      `json:"is_latest"`
	IsPublic      bool      `json:"is_public"`
	ShareToken    string    `json:"share_token"`
	Status        string    `json:"status"`
	UploadedBy    int       `json:"uploaded_by"`
	DownloadCount int       `json:"download_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SimpleFolder 简化的文件夹模型用于测试
type SimpleFolder struct {
	ID          int       `json:"id"`
	TenantID    string    `json:"tenant_id"`
	ProjectID   int       `json:"project_id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	ParentID    *int      `json:"parent_id"`
	Level       int       `json:"level"`
	IsPublic    bool      `json:"is_public"`
	CreatedBy   int       `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SimpleFileActivity 简化的文件活动模型用于测试
type SimpleFileActivity struct {
	ID        int       `json:"id"`
	TenantID  string    `json:"tenant_id"`
	FileID    int       `json:"file_id"`
	UserID    int       `json:"user_id"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// SimpleUploadRequest 简化的上传请求用于测试
type SimpleUploadRequest struct {
	ProjectID   int    `json:"project_id"`
	FolderID    *int   `json:"folder_id"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
}

// SimpleCreateFolderRequest 简化的创建文件夹请求用于测试
type SimpleCreateFolderRequest struct {
	ProjectID   int    `json:"project_id"`
	Name        string `json:"name"`
	ParentID    *int   `json:"parent_id"`
	Description string `json:"description"`
}

// File Service 验证函数

// validateFileName 验证文件名
func validateFileName(name string) error {
	if name == "" {
		return fmt.Errorf("文件名不能为空")
	}

	if len(name) > 255 {
		return fmt.Errorf("文件名长度不能超过255字符")
	}

	// 检查不允许的字符
	invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*", "/", "\\"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("文件名不能包含字符: %s", char)
		}
	}

	// 检查保留名称
	reservedNames := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	upperName := strings.ToUpper(name)
	for _, reserved := range reservedNames {
		if upperName == reserved || strings.HasPrefix(upperName, reserved+".") {
			return fmt.Errorf("文件名不能使用保留名称: %s", reserved)
		}
	}

	return nil
}

// validateMimeType 验证MIME类型
func validateMimeType(mimeType string) error {
	if mimeType == "" {
		return fmt.Errorf("MIME类型不能为空")
	}

	// MIME类型格式检查 - 更宽松的模式支持更多标准MIME类型
	mimePattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9!#$&\-\^\.]*\/[a-zA-Z0-9][a-zA-Z0-9!#$&\-\^\.]*$`)
	if !mimePattern.MatchString(mimeType) {
		return fmt.Errorf("无效的MIME类型格式")
	}

	return nil
}

// validateFileSize 验证文件大小
func validateFileSize(size int64, maxSize int64) error {
	if size <= 0 {
		return fmt.Errorf("文件大小必须大于0")
	}

	if size > maxSize {
		return fmt.Errorf("文件大小超过限制: %d bytes", maxSize)
	}

	return nil
}

// validateFileExtension 验证文件扩展名
func validateFileExtension(extension string) error {
	if extension == "" {
		return nil // 扩展名可以为空
	}

	if !strings.HasPrefix(extension, ".") {
		return fmt.Errorf("文件扩展名必须以点号开头")
	}

	if len(extension) > 10 {
		return fmt.Errorf("文件扩展名长度不能超过10字符")
	}

	// 只允许字母和数字
	extPattern := regexp.MustCompile(`^\.[a-zA-Z0-9]+$`)
	if !extPattern.MatchString(extension) {
		return fmt.Errorf("文件扩展名只能包含字母和数字")
	}

	return nil
}

// validateFileHash 验证文件哈希
func validateFileHash(hash string) error {
	if hash == "" {
		return fmt.Errorf("文件哈希不能为空")
	}

	// MD5哈希长度为32字符
	if len(hash) != 32 {
		return fmt.Errorf("MD5哈希长度必须为32字符")
	}

	// 哈希只能包含十六进制字符
	hashPattern := regexp.MustCompile(`^[a-fA-F0-9]+$`)
	if !hashPattern.MatchString(hash) {
		return fmt.Errorf("文件哈希只能包含十六进制字符")
	}

	return nil
}

// validateFolderName 验证文件夹名称
func validateFolderName(name string) error {
	if name == "" {
		return fmt.Errorf("文件夹名称不能为空")
	}

	if len(name) < 1 || len(name) > 100 {
		return fmt.Errorf("文件夹名称长度必须在1-100字符之间")
	}

	// 检查不允许的字符
	invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*", "/", "\\"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("文件夹名称不能包含字符: %s", char)
		}
	}

	// 不能以点开头或结尾
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return fmt.Errorf("文件夹名称不能以点开头或结尾")
	}

	return nil
}

// validateFileStatus 验证文件状态
func validateFileStatus(status string) error {
	if status == "" {
		return fmt.Errorf("文件状态不能为空")
	}

	validStatuses := []string{"active", "deleted", "archived", "processing"}
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return nil
		}
	}

	return fmt.Errorf("无效的文件状态: %s", status)
}

// validateActivityAction 验证活动动作
func validateActivityAction(action string) error {
	if action == "" {
		return fmt.Errorf("活动动作不能为空")
	}

	validActions := []string{"upload", "download", "delete", "share", "view", "edit", "rename", "move", "copy"}
	for _, validAction := range validActions {
		if action == validAction {
			return nil
		}
	}

	return fmt.Errorf("无效的活动动作: %s", action)
}

// validateUploadRequest 验证上传请求
func validateUploadRequest(req *SimpleUploadRequest) error {
	if req == nil {
		return fmt.Errorf("上传请求不能为nil")
	}

	if req.ProjectID <= 0 {
		return fmt.Errorf("项目ID必须大于0")
	}

	if req.FolderID != nil && *req.FolderID <= 0 {
		return fmt.Errorf("文件夹ID必须大于0")
	}

	if len(req.Description) > 500 {
		return fmt.Errorf("描述长度不能超过500字符")
	}

	// 验证标签格式
	if req.Tags != "" {
		tags := strings.Split(req.Tags, ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if len(tag) > 50 {
				return fmt.Errorf("标签长度不能超过50字符")
			}
		}
	}

	return nil
}

// validateCreateFolderRequest 验证创建文件夹请求
func validateCreateFolderRequest(req *SimpleCreateFolderRequest) error {
	if req == nil {
		return fmt.Errorf("创建文件夹请求不能为nil")
	}

	if req.ProjectID <= 0 {
		return fmt.Errorf("项目ID必须大于0")
	}

	if err := validateFolderName(req.Name); err != nil {
		return fmt.Errorf("文件夹名称: %v", err)
	}

	if req.ParentID != nil && *req.ParentID <= 0 {
		return fmt.Errorf("父文件夹ID必须大于0")
	}

	if len(req.Description) > 500 {
		return fmt.Errorf("描述长度不能超过500字符")
	}

	return nil
}

// calculateMD5Hash 计算MD5哈希
func calculateMD5Hash(data []byte) string {
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// getFileType 获取文件类型
func getFileType(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	if strings.Contains(mimeType, "pdf") || strings.Contains(mimeType, "document") || strings.Contains(mimeType, "text") {
		return "document"
	}
	if strings.Contains(mimeType, "video/") {
		return "video"
	}
	if strings.Contains(mimeType, "audio/") {
		return "audio"
	}
	return "other"
}

// canPreview 检查文件是否可预览
func canPreview(mimeType string) bool {
	previewableTypes := []string{
		"image/jpeg", "image/png", "image/gif", "image/webp",
		"text/plain", "text/html", "text/css", "text/javascript",
		"application/pdf", "application/json",
	}

	for _, pType := range previewableTypes {
		if mimeType == pType {
			return true
		}
	}

	return strings.HasPrefix(mimeType, "text/") || strings.HasPrefix(mimeType, "image/")
}

// formatSize 格式化文件大小
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// TestFileNameValidation 测试文件名验证
func TestFileNameValidation(t *testing.T) {
	tests := []struct {
		name          string
		fileName      string
		expectedError string
	}{
		{"有效的文件名", "document.pdf", ""},
		{"有效的图片文件", "image.jpg", ""},
		{"有效的长文件名", "这是一个很长的文件名包含中文字符.txt", ""},
		{"有效的文件名带空格", "my document.pdf", ""},
		{"空文件名", "", "文件名不能为空"},
		{"文件名过长", strings.Repeat("a", 256), "文件名长度不能超过255字符"},
		{"包含非法字符1", "file<name.txt", "文件名不能包含字符: <"},
		{"包含非法字符2", "file>name.txt", "文件名不能包含字符: >"},
		{"包含非法字符3", "file:name.txt", "文件名不能包含字符: :"},
		{"包含非法字符4", "file\"name.txt", "文件名不能包含字符: \""},
		{"包含非法字符5", "file|name.txt", "文件名不能包含字符: |"},
		{"包含非法字符6", "file?name.txt", "文件名不能包含字符: ?"},
		{"包含非法字符7", "file*name.txt", "文件名不能包含字符: *"},
		{"包含斜杠", "file/name.txt", "文件名不能包含字符: /"},
		{"包含反斜杠", "file\\name.txt", "文件名不能包含字符: \\"},
		{"保留名称CON", "CON", "文件名不能使用保留名称: CON"},
		{"保留名称PRN", "PRN.txt", "文件名不能使用保留名称: PRN"},
		{"保留名称AUX", "aux", "文件名不能使用保留名称: AUX"},
		{"保留名称COM1", "COM1.log", "文件名不能使用保留名称: COM1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileName(tt.fileName)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestMimeTypeValidation 测试MIME类型验证
func TestMimeTypeValidation(t *testing.T) {
	tests := []struct {
		name          string
		mimeType      string
		expectedError string
	}{
		{"有效的图片类型", "image/jpeg", ""},
		{"有效的文档类型", "application/pdf", ""},
		{"有效的文本类型", "text/plain", ""},
		{"有效的音频类型", "audio/mpeg", ""},
		{"有效的视频类型", "video/mp4", ""},
		{"有效的复杂类型", "application/vnd.ms-excel", ""},
		{"空MIME类型", "", "MIME类型不能为空"},
		{"无效格式1", "image", "无效的MIME类型格式"},
		{"无效格式2", "/jpeg", "无效的MIME类型格式"},
		{"无效格式3", "image/", "无效的MIME类型格式"},
		{"包含非法字符", "image/jpe@g", "无效的MIME类型格式"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMimeType(tt.mimeType)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestFileSizeValidation 测试文件大小验证
func TestFileSizeValidation(t *testing.T) {
	maxSize := int64(100 * 1024 * 1024) // 100MB

	tests := []struct {
		name          string
		size          int64
		expectedError string
	}{
		{"有效的小文件", 1024, ""},
		{"有效的中等文件", 10 * 1024 * 1024, ""},
		{"有效的大文件", 99 * 1024 * 1024, ""},
		{"边界大小", maxSize, ""},
		{"零大小", 0, "文件大小必须大于0"},
		{"负数大小", -1, "文件大小必须大于0"},
		{"超出限制", maxSize + 1, "文件大小超过限制"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileSize(tt.size, maxSize)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestFileExtensionValidation 测试文件扩展名验证
func TestFileExtensionValidation(t *testing.T) {
	tests := []struct {
		name          string
		extension     string
		expectedError string
	}{
		{"有效的扩展名", ".txt", ""},
		{"有效的图片扩展名", ".jpg", ""},
		{"有效的文档扩展名", ".pdf", ""},
		{"有效的代码扩展名", ".go", ""},
		{"有效的长扩展名", ".jpeg", ""},
		{"空扩展名", "", ""},
		{"无点号开头", "txt", "文件扩展名必须以点号开头"},
		{"扩展名过长", ".verylongext", "文件扩展名长度不能超过10字符"},
		{"包含特殊字符", ".tx-t", "文件扩展名只能包含字母和数字"},
		{"包含空格", ".tx t", "文件扩展名只能包含字母和数字"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileExtension(tt.extension)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestFileHashValidation 测试文件哈希验证
func TestFileHashValidation(t *testing.T) {
	tests := []struct {
		name          string
		hash          string
		expectedError string
	}{
		{"有效的MD5哈希", "d41d8cd98f00b204e9800998ecf8427e", ""},
		{"有效的大写哈希", "D41D8CD98F00B204E9800998ECF8427E", ""},
		{"有效的混合大小写", "d41D8Cd98f00B204e9800998ECf8427E", ""},
		{"空哈希", "", "文件哈希不能为空"},
		{"哈希过短", "d41d8cd98f00b204e9800998ecf8427", "MD5哈希长度必须为32字符"},
		{"哈希过长", "d41d8cd98f00b204e9800998ecf8427e1", "MD5哈希长度必须为32字符"},
		{"包含非十六进制字符", "d41d8cd98f00g204e9800998ecf8427e", "文件哈希只能包含十六进制字符"},
		{"包含特殊字符", "d41d8cd98f00-204e9800998ecf8427e", "文件哈希只能包含十六进制字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileHash(tt.hash)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestFolderNameValidation 测试文件夹名称验证
func TestFolderNameValidation(t *testing.T) {
	tests := []struct {
		name          string
		folderName    string
		expectedError string
	}{
		{"有效的文件夹名", "documents", ""},
		{"有效的中文文件夹名", "我的文档", ""},
		{"有效的带数字文件夹名", "folder123", ""},
		{"有效的带空格文件夹名", "my documents", ""},
		{"空文件夹名", "", "文件夹名称不能为空"},
		{"文件夹名过长", strings.Repeat("a", 101), "文件夹名称长度必须在1-100字符之间"},
		{"以点开头", ".hidden", "文件夹名称不能以点开头或结尾"},
		{"以点结尾", "folder.", "文件夹名称不能以点开头或结尾"},
		{"包含非法字符", "folder<name", "文件夹名称不能包含字符: <"},
		{"包含斜杠", "folder/name", "文件夹名称不能包含字符: /"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFolderName(tt.folderName)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestFileStatusValidation 测试文件状态验证
func TestFileStatusValidation(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		expectedError string
	}{
		{"活跃状态", "active", ""},
		{"删除状态", "deleted", ""},
		{"归档状态", "archived", ""},
		{"处理中状态", "processing", ""},
		{"空状态", "", "文件状态不能为空"},
		{"无效状态", "unknown", "无效的文件状态"},
		{"大写状态", "ACTIVE", "无效的文件状态"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileStatus(tt.status)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestUploadRequestValidation 测试上传请求验证
func TestUploadRequestValidation(t *testing.T) {
	folderID := 1

	tests := []struct {
		name          string
		req           *SimpleUploadRequest
		expectedError string
	}{
		{
			name: "有效的上传请求",
			req: &SimpleUploadRequest{
				ProjectID:   1,
				FolderID:    &folderID,
				Description: "测试文件上传",
				Tags:        "test,upload",
			},
			expectedError: "",
		},
		{
			name: "无文件夹的上传请求",
			req: &SimpleUploadRequest{
				ProjectID:   1,
				Description: "根目录上传",
				Tags:        "test",
			},
			expectedError: "",
		},
		{
			name:          "nil请求",
			req:           nil,
			expectedError: "上传请求不能为nil",
		},
		{
			name: "项目ID为0",
			req: &SimpleUploadRequest{
				ProjectID: 0,
			},
			expectedError: "项目ID必须大于0",
		},
		{
			name: "项目ID为负数",
			req: &SimpleUploadRequest{
				ProjectID: -1,
			},
			expectedError: "项目ID必须大于0",
		},
		{
			name: "文件夹ID为0",
			req: &SimpleUploadRequest{
				ProjectID: 1,
				FolderID:  &[]int{0}[0],
			},
			expectedError: "文件夹ID必须大于0",
		},
		{
			name: "描述过长",
			req: &SimpleUploadRequest{
				ProjectID:   1,
				Description: strings.Repeat("a", 501),
			},
			expectedError: "描述长度不能超过500字符",
		},
		{
			name: "标签过长",
			req: &SimpleUploadRequest{
				ProjectID: 1,
				Tags:      strings.Repeat("a", 51),
			},
			expectedError: "标签长度不能超过50字符",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUploadRequest(tt.req)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestFileUtilityFunctions 测试文件工具函数
func TestFileUtilityFunctions(t *testing.T) {
	t.Run("MD5哈希计算", func(t *testing.T) {
		testData := []byte("hello world")
		expectedHash := "5eb63bbbe01eeed093cb22bb8f5acdc3"
		actualHash := calculateMD5Hash(testData)
		assert.Equal(t, expectedHash, actualHash)
	})

	t.Run("文件类型检测", func(t *testing.T) {
		tests := []struct {
			mimeType     string
			expectedType string
		}{
			{"image/jpeg", "image"},
			{"image/png", "image"},
			{"application/pdf", "document"},
			{"text/plain", "document"},
			{"video/mp4", "video"},
			{"audio/mp3", "audio"},
			{"application/zip", "other"},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.expectedType, getFileType(tt.mimeType))
		}
	})

	t.Run("文件预览检测", func(t *testing.T) {
		tests := []struct {
			mimeType   string
			canPreview bool
		}{
			{"image/jpeg", true},
			{"text/plain", true},
			{"application/pdf", true},
			{"video/mp4", false},
			{"application/zip", false},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.canPreview, canPreview(tt.mimeType))
		}
	})

	t.Run("文件大小格式化", func(t *testing.T) {
		tests := []struct {
			size           int64
			expectedFormat string
		}{
			{512, "512 B"},
			{1024, "1.0 KB"},
			{1536, "1.5 KB"},
			{1048576, "1.0 MB"},
			{1073741824, "1.0 GB"},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.expectedFormat, formatSize(tt.size))
		}
	})
}

// TestFileServiceEdgeCases 测试文件服务边界情况
func TestFileServiceEdgeCases(t *testing.T) {
	t.Run("极长的描述文本", func(t *testing.T) {
		longDescription := strings.Repeat("这是一个很长的描述", 50)
		req := &SimpleUploadRequest{
			ProjectID:   1,
			Description: longDescription,
		}

		if len(longDescription) > 500 {
			err := validateUploadRequest(req)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "描述长度不能超过500字符")
		}
	})

	t.Run("Unicode文件名", func(t *testing.T) {
		unicodeNames := []string{
			"测试文件.txt",
			"файл.pdf",
			"ファイル.jpg",
			"الملف.doc",
		}

		for _, name := range unicodeNames {
			err := validateFileName(name)
			assert.NoError(t, err, "Unicode文件名应该被允许: %s", name)
		}
	})

	t.Run("边界大小文件", func(t *testing.T) {
		maxSize := int64(100 * 1024 * 1024)

		// 测试恰好等于限制的文件
		err := validateFileSize(maxSize, maxSize)
		assert.NoError(t, err)

		// 测试超出1字节的文件
		err = validateFileSize(maxSize+1, maxSize)
		assert.Error(t, err)
	})
}

// TestFileServicePerformance 测试文件服务性能
func TestFileServicePerformance(t *testing.T) {
	t.Run("批量文件名验证", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			fileName := fmt.Sprintf("file_%d.txt", i)
			err := validateFileName(fileName)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个文件名耗时: %v", duration)

		// 验证应该在5毫秒内完成
		assert.Less(t, duration, 5*time.Millisecond)
	})

	t.Run("批量MIME类型验证", func(t *testing.T) {
		start := time.Now()

		mimeTypes := []string{"image/jpeg", "text/plain", "application/pdf", "video/mp4", "audio/mpeg"}
		for i := 0; i < 1000; i++ {
			mimeType := mimeTypes[i%len(mimeTypes)]
			err := validateMimeType(mimeType)
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("批量验证1000个MIME类型耗时: %v", duration)

		// 验证应该在10毫秒内完成
		assert.Less(t, duration, 10*time.Millisecond)
	})

	t.Run("批量哈希计算", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 100; i++ {
			data := []byte(fmt.Sprintf("test data %d", i))
			hash := calculateMD5Hash(data)
			assert.Len(t, hash, 32)
		}

		duration := time.Since(start)
		t.Logf("批量计算100个MD5哈希耗时: %v", duration)

		// 哈希计算应该在10毫秒内完成
		assert.Less(t, duration, 10*time.Millisecond)
	})
}

// MockFileService 文件服务模拟实现
type MockFileService struct {
	files           map[int]*SimpleFile
	folders         map[int]*SimpleFolder
	activities      map[int]*SimpleFileActivity
	fileCounter     int
	folderCounter   int
	activityCounter int
}

// NewMockFileService 创建文件服务模拟
func NewMockFileService() *MockFileService {
	return &MockFileService{
		files:           make(map[int]*SimpleFile),
		folders:         make(map[int]*SimpleFolder),
		activities:      make(map[int]*SimpleFileActivity),
		fileCounter:     1,
		folderCounter:   1,
		activityCounter: 1,
	}
}

// UploadFile 模拟文件上传
func (m *MockFileService) UploadFile(req *SimpleUploadRequest, originalName, mimeType string, size int64, data []byte) (*SimpleFile, error) {
	if err := validateUploadRequest(req); err != nil {
		return nil, err
	}

	if err := validateFileName(originalName); err != nil {
		return nil, fmt.Errorf("文件名: %v", err)
	}

	if err := validateMimeType(mimeType); err != nil {
		return nil, fmt.Errorf("MIME类型: %v", err)
	}

	if err := validateFileSize(size, 100*1024*1024); err != nil {
		return nil, err
	}

	// 生成文件哈希
	hash := calculateMD5Hash(data)

	// 检查重复文件
	for _, file := range m.files {
		if file.Hash == hash && file.ProjectID == req.ProjectID {
			return nil, fmt.Errorf("文件已存在: %s", file.OriginalName)
		}
	}

	// 生成唯一文件名
	ext := filepath.Ext(originalName)
	uniqueName := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)

	// 解析标签
	var tags []string
	if req.Tags != "" {
		for _, tag := range strings.Split(req.Tags, ",") {
			tags = append(tags, strings.TrimSpace(tag))
		}
	}

	file := &SimpleFile{
		ID:            m.fileCounter,
		TenantID:      "default",
		ProjectID:     req.ProjectID,
		Name:          uniqueName,
		OriginalName:  originalName,
		Path:          "/uploads/" + uniqueName,
		Size:          size,
		MimeType:      mimeType,
		Extension:     ext,
		Hash:          hash,
		FolderID:      req.FolderID,
		Tags:          tags,
		Description:   req.Description,
		Version:       1,
		IsLatest:      true,
		IsPublic:      false,
		Status:        "active",
		UploadedBy:    1,
		DownloadCount: 0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	m.files[file.ID] = file
	m.fileCounter++

	// 记录活动
	m.logActivity(file.ID, 1, "upload", fmt.Sprintf("上传文件: %s", originalName), "127.0.0.1", "test-agent")

	return file, nil
}

// CreateFolder 模拟创建文件夹
func (m *MockFileService) CreateFolder(req *SimpleCreateFolderRequest) (*SimpleFolder, error) {
	if err := validateCreateFolderRequest(req); err != nil {
		return nil, err
	}

	// 检查同名文件夹
	for _, folder := range m.folders {
		if folder.Name == req.Name && folder.ProjectID == req.ProjectID && folder.ParentID == req.ParentID {
			return nil, fmt.Errorf("文件夹名称已存在: %s", req.Name)
		}
	}

	// 构建路径
	path := req.Name
	level := 0
	if req.ParentID != nil {
		parent, exists := m.folders[*req.ParentID]
		if !exists {
			return nil, fmt.Errorf("父文件夹不存在")
		}
		path = parent.Path + "/" + req.Name
		level = parent.Level + 1
	}

	folder := &SimpleFolder{
		ID:          m.folderCounter,
		TenantID:    "default",
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Path:        path,
		Description: req.Description,
		ParentID:    req.ParentID,
		Level:       level,
		IsPublic:    false,
		CreatedBy:   1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.folders[folder.ID] = folder
	m.folderCounter++

	return folder, nil
}

// logActivity 记录文件活动
func (m *MockFileService) logActivity(fileID, userID int, action, details, ipAddress, userAgent string) {
	activity := &SimpleFileActivity{
		ID:        m.activityCounter,
		TenantID:  "default",
		FileID:    fileID,
		UserID:    userID,
		Action:    action,
		Details:   details,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
	}

	m.activities[activity.ID] = activity
	m.activityCounter++
}

// TestMockFileService 测试文件服务模拟
func TestMockFileService(t *testing.T) {
	mockService := NewMockFileService()

	t.Run("文件上传成功", func(t *testing.T) {
		req := &SimpleUploadRequest{
			ProjectID:   1,
			Description: "测试文件",
			Tags:        "test,upload",
		}

		testData := []byte("test file content")
		file, err := mockService.UploadFile(req, "test.txt", "text/plain", int64(len(testData)), testData)

		require.NoError(t, err)
		require.NotNil(t, file)

		assert.Equal(t, 1, file.ID)
		assert.Equal(t, req.ProjectID, file.ProjectID)
		assert.Equal(t, "test.txt", file.OriginalName)
		assert.Equal(t, "text/plain", file.MimeType)
		assert.Equal(t, int64(len(testData)), file.Size)
		assert.Equal(t, ".txt", file.Extension)
		assert.Equal(t, "active", file.Status)
		assert.Equal(t, []string{"test", "upload"}, file.Tags)
		assert.Len(t, file.Hash, 32)
	})

	t.Run("创建文件夹成功", func(t *testing.T) {
		req := &SimpleCreateFolderRequest{
			ProjectID:   1,
			Name:        "documents",
			Description: "文档文件夹",
		}

		folder, err := mockService.CreateFolder(req)

		require.NoError(t, err)
		require.NotNil(t, folder)

		assert.Equal(t, 1, folder.ID)
		assert.Equal(t, req.ProjectID, folder.ProjectID)
		assert.Equal(t, req.Name, folder.Name)
		assert.Equal(t, req.Name, folder.Path)
		assert.Equal(t, 0, folder.Level)
		assert.Nil(t, folder.ParentID)
	})

	t.Run("创建子文件夹成功", func(t *testing.T) {
		// 先创建父文件夹
		parentReq := &SimpleCreateFolderRequest{
			ProjectID: 1,
			Name:      "parent",
		}
		parent, err := mockService.CreateFolder(parentReq)
		require.NoError(t, err)

		// 创建子文件夹
		childReq := &SimpleCreateFolderRequest{
			ProjectID: 1,
			Name:      "child",
			ParentID:  &parent.ID,
		}

		child, err := mockService.CreateFolder(childReq)

		require.NoError(t, err)
		require.NotNil(t, child)

		assert.Equal(t, parent.ID, *child.ParentID)
		assert.Equal(t, "parent/child", child.Path)
		assert.Equal(t, 1, child.Level)
	})

	t.Run("重复文件上传失败", func(t *testing.T) {
		req := &SimpleUploadRequest{
			ProjectID: 1,
		}

		testData := []byte("duplicate content")

		// 第一次上传成功
		_, err := mockService.UploadFile(req, "duplicate.txt", "text/plain", int64(len(testData)), testData)
		require.NoError(t, err)

		// 第二次上传同样内容失败
		file, err := mockService.UploadFile(req, "duplicate2.txt", "text/plain", int64(len(testData)), testData)

		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "文件已存在")
	})

	t.Run("同名文件夹创建失败", func(t *testing.T) {
		req := &SimpleCreateFolderRequest{
			ProjectID: 1,
			Name:      "samename",
		}

		// 第一次创建成功
		_, err := mockService.CreateFolder(req)
		require.NoError(t, err)

		// 第二次创建失败
		folder, err := mockService.CreateFolder(req)

		assert.Error(t, err)
		assert.Nil(t, folder)
		assert.Contains(t, err.Error(), "文件夹名称已存在")
	})

	t.Run("父文件夹不存在创建失败", func(t *testing.T) {
		nonExistentParentID := 9999
		req := &SimpleCreateFolderRequest{
			ProjectID: 1,
			Name:      "orphan",
			ParentID:  &nonExistentParentID,
		}

		folder, err := mockService.CreateFolder(req)

		assert.Error(t, err)
		assert.Nil(t, folder)
		assert.Contains(t, err.Error(), "父文件夹不存在")
	})

	t.Run("验证活动记录", func(t *testing.T) {
		// 上传文件会自动记录活动
		req := &SimpleUploadRequest{
			ProjectID: 1,
		}

		testData := []byte("activity test")
		file, err := mockService.UploadFile(req, "activity.txt", "text/plain", int64(len(testData)), testData)
		require.NoError(t, err)

		// 检查活动是否记录
		assert.Greater(t, len(mockService.activities), 0)

		// 找到对应的活动
		var foundActivity *SimpleFileActivity
		for _, activity := range mockService.activities {
			if activity.FileID == file.ID {
				foundActivity = activity
				break
			}
		}

		require.NotNil(t, foundActivity)
		assert.Equal(t, "upload", foundActivity.Action)
		assert.Contains(t, foundActivity.Details, "activity.txt")
	})
}

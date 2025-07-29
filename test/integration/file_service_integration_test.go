package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// FileService数据模型
type File struct {
	ID            int       `json:"id" gorm:"primary_key"`
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
	Tags          []string  `json:"tags" gorm:"type:json"`
	Description   string    `json:"description"`
	Version       int       `json:"version" gorm:"default:1"`
	IsLatest      bool      `json:"is_latest" gorm:"default:true"`
	IsPublic      bool      `json:"is_public" gorm:"default:false"`
	ShareToken    string    `json:"share_token"`
	Status        string    `json:"status" gorm:"default:'active'"`
	UploadedBy    int       `json:"uploaded_by"`
	DownloadCount int       `json:"download_count" gorm:"default:0"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Folder struct {
	ID          int       `json:"id" gorm:"primary_key"`
	TenantID    string    `json:"tenant_id"`
	ProjectID   int       `json:"project_id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	ParentID    *int      `json:"parent_id"`
	Level       int       `json:"level" gorm:"default:0"`
	IsPublic    bool      `json:"is_public" gorm:"default:false"`
	CreatedBy   int       `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type FileActivity struct {
	ID        int       `json:"id" gorm:"primary_key"`
	TenantID  string    `json:"tenant_id"`
	FileID    int       `json:"file_id"`
	UserID    int       `json:"user_id"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// 请求响应结构
type UploadRequest struct {
	ProjectID   int    `form:"project_id" binding:"required"`
	FolderID    *int   `form:"folder_id"`
	Description string `form:"description"`
	Tags        string `form:"tags"`
}

type CreateFolderRequest struct {
	ProjectID   int    `json:"project_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	ParentID    *int   `json:"parent_id"`
	Description string `json:"description"`
}

type FileResponse struct {
	File
	FileType      string `json:"file_type"`
	FormattedSize string `json:"formatted_size"`
	CanPreview    bool   `json:"can_preview"`
	PreviewURL    string `json:"preview_url,omitempty"`
	DownloadURL   string `json:"download_url"`
}

// FileServiceIntegrationTestSuite 文件服务集成测试套件
type FileServiceIntegrationTestSuite struct {
	suite.Suite
	db        *gorm.DB
	router    *gin.Engine
	uploadDir string
}

// SetupSuite 测试套件初始化
func (suite *FileServiceIntegrationTestSuite) SetupSuite() {
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 创建内存SQLite数据库
	var err error
	suite.db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(suite.T(), err)

	// 自动迁移
	err = suite.db.AutoMigrate(&File{}, &Folder{}, &FileActivity{})
	require.NoError(suite.T(), err)

	// 创建临时上传目录
	suite.uploadDir = "./test_uploads"
	err = os.MkdirAll(suite.uploadDir, 0755)
	require.NoError(suite.T(), err)

	// 初始化路由
	suite.setupRouter()
}

// TearDownSuite 测试套件清理
func (suite *FileServiceIntegrationTestSuite) TearDownSuite() {
	// 清理上传目录
	os.RemoveAll(suite.uploadDir)
}

// SetupTest 每个测试前的初始化
func (suite *FileServiceIntegrationTestSuite) SetupTest() {
	// 清理数据库
	suite.db.Exec("DELETE FROM file_activities")
	suite.db.Exec("DELETE FROM files")
	suite.db.Exec("DELETE FROM folders")

	// 清理上传文件
	files, _ := filepath.Glob(filepath.Join(suite.uploadDir, "*", "*"))
	for _, file := range files {
		os.RemoveAll(file)
	}
}

// setupRouter 设置路由
func (suite *FileServiceIntegrationTestSuite) setupRouter() {
	suite.router = gin.New()

	// API路由组
	api := suite.router.Group("/api/v1")
	{
		// 健康检查
		api.GET("/health", suite.healthCheck)

		// 文件管理
		files := api.Group("/files")
		{
			files.POST("/upload", suite.uploadFiles)
			files.GET("/project/:projectId", suite.listFiles)
			files.GET("/:id", suite.getFile)
			files.GET("/:id/download", suite.downloadFile)
			files.GET("/:id/preview", suite.previewFile)
			files.PUT("/:id", suite.updateFile)
			files.DELETE("/:id", suite.deleteFile)
		}

		// 文件夹管理
		folders := api.Group("/folders")
		{
			folders.POST("", suite.createFolder)
			folders.GET("/project/:projectId", suite.listFolders)
			folders.GET("/:id", suite.getFolder)
			folders.PUT("/:id", suite.updateFolder)
			folders.DELETE("/:id", suite.deleteFolder)
		}

		// 文件活动
		api.GET("/files/:id/activities", suite.getFileActivities)
	}
}

// API处理函数实现
func (suite *FileServiceIntegrationTestSuite) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"service":    "文件管理服务",
		"version":    "1.0.0",
		"status":     "healthy",
		"upload_dir": suite.uploadDir,
	})
}

func (suite *FileServiceIntegrationTestSuite) uploadFiles(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 获取上传的文件
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取文件失败: " + err.Error()})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未选择文件"})
		return
	}

	var uploadedFiles []FileResponse
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	// 创建项目上传目录
	projectUploadDir := filepath.Join(suite.uploadDir, tenantID, fmt.Sprintf("project_%d", req.ProjectID))
	if err := os.MkdirAll(projectUploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目录失败"})
		return
	}

	for _, fileHeader := range files {
		file, err := suite.processFileUpload(fileHeader, req, tenantID, projectUploadDir)
		if err != nil {
			continue
		}
		uploadedFiles = append(uploadedFiles, *file)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"files":   uploadedFiles,
		"count":   len(uploadedFiles),
	})
}

func (suite *FileServiceIntegrationTestSuite) processFileUpload(fileHeader *multipart.FileHeader, req UploadRequest, tenantID, uploadDir string) (*FileResponse, error) {
	// 打开上传的文件
	src, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// 读取文件内容并计算哈希
	content, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}

	hash := fmt.Sprintf("%x", content) // 简化的哈希计算

	// 生成唯一文件名
	ext := filepath.Ext(fileHeader.Filename)
	fileName := fmt.Sprintf("test_%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadDir, fileName)

	// 保存文件
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	if _, err := dst.Write(content); err != nil {
		return nil, err
	}

	// 解析标签
	var tags []string
	if req.Tags != "" {
		for _, tag := range strings.Split(req.Tags, ",") {
			tags = append(tags, strings.TrimSpace(tag))
		}
	}

	// 保存到数据库
	file := File{
		TenantID:     tenantID,
		ProjectID:    req.ProjectID,
		Name:         fileName,
		OriginalName: fileHeader.Filename,
		Path:         filePath,
		Size:         fileHeader.Size,
		MimeType:     fileHeader.Header.Get("Content-Type"),
		Extension:    ext,
		Hash:         hash,
		FolderID:     req.FolderID,
		Tags:         tags,
		Description:  req.Description,
		UploadedBy:   1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := suite.db.Create(&file).Error; err != nil {
		os.Remove(filePath)
		return nil, err
	}

	return suite.buildFileResponse(&file), nil
}

func (suite *FileServiceIntegrationTestSuite) listFiles(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("projectId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目ID无效"})
		return
	}

	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var files []File
	query := suite.db.Where("tenant_id = ? AND project_id = ? AND status = ?", tenantID, projectID, "active")

	// 文件夹过滤
	if folderID := c.Query("folder_id"); folderID != "" {
		if folderID == "null" || folderID == "0" {
			query = query.Where("folder_id IS NULL")
		} else {
			query = query.Where("folder_id = ?", folderID)
		}
	}

	if err := query.Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询文件失败"})
		return
	}

	var fileResponses []FileResponse
	for _, file := range files {
		fileResponses = append(fileResponses, *suite.buildFileResponse(&file))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"files":   fileResponses,
		"total":   len(fileResponses),
	})
}

func (suite *FileServiceIntegrationTestSuite) getFile(c *gin.Context) {
	fileID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件ID无效"})
		return
	}

	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var file File
	if err := suite.db.Where("id = ? AND tenant_id = ? AND status = ?", fileID, tenantID, "active").First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"file":    suite.buildFileResponse(&file),
	})
}

func (suite *FileServiceIntegrationTestSuite) downloadFile(c *gin.Context) {
	fileID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件ID无效"})
		return
	}

	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var file File
	if err := suite.db.Where("id = ? AND tenant_id = ? AND status = ?", fileID, tenantID, "active").First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 更新下载计数
	suite.db.Model(&file).Update("download_count", file.DownloadCount+1)

	// 记录活动
	suite.logFileActivity(file.ID, 1, "download", fmt.Sprintf("下载文件: %s", file.OriginalName), c.ClientIP(), c.GetHeader("User-Agent"), tenantID)

	// 设置响应头
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.OriginalName))
	c.Header("Content-Type", file.MimeType)
	c.File(file.Path)
}

func (suite *FileServiceIntegrationTestSuite) previewFile(c *gin.Context) {
	fileID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件ID无效"})
		return
	}

	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var file File
	if err := suite.db.Where("id = ? AND tenant_id = ? AND status = ?", fileID, tenantID, "active").First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 检查是否可预览
	if !suite.canPreview(&file) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该文件类型不支持预览"})
		return
	}

	c.Header("Content-Type", file.MimeType)
	c.File(file.Path)
}

func (suite *FileServiceIntegrationTestSuite) updateFile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "更新文件"})
}

func (suite *FileServiceIntegrationTestSuite) deleteFile(c *gin.Context) {
	fileID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件ID无效"})
		return
	}

	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var file File
	if err := suite.db.Where("id = ? AND tenant_id = ? AND status = ?", fileID, tenantID, "active").First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 软删除
	if err := suite.db.Model(&file).Update("status", "deleted").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除文件失败"})
		return
	}

	// 记录活动
	suite.logFileActivity(file.ID, 1, "delete", fmt.Sprintf("删除文件: %s", file.OriginalName), c.ClientIP(), c.GetHeader("User-Agent"), tenantID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "文件删除成功",
	})
}

func (suite *FileServiceIntegrationTestSuite) createFolder(c *gin.Context) {
	var req CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	// 构建路径
	path := req.Name
	level := 0
	if req.ParentID != nil {
		var parent Folder
		if err := suite.db.Where("id = ? AND tenant_id = ?", *req.ParentID, tenantID).First(&parent).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "父文件夹不存在"})
			return
		}
		path = parent.Path + "/" + req.Name
		level = parent.Level + 1
	}

	folder := Folder{
		TenantID:    tenantID,
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Path:        path,
		Description: req.Description,
		ParentID:    req.ParentID,
		Level:       level,
		CreatedBy:   1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := suite.db.Create(&folder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建文件夹失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"folder":  folder,
	})
}

func (suite *FileServiceIntegrationTestSuite) listFolders(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("projectId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目ID无效"})
		return
	}

	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var folders []Folder
	if err := suite.db.Where("tenant_id = ? AND project_id = ?", tenantID, projectID).Find(&folders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询文件夹失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"folders": folders,
	})
}

func (suite *FileServiceIntegrationTestSuite) getFolder(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "获取文件夹"})
}

func (suite *FileServiceIntegrationTestSuite) updateFolder(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "更新文件夹"})
}

func (suite *FileServiceIntegrationTestSuite) deleteFolder(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "删除文件夹"})
}

func (suite *FileServiceIntegrationTestSuite) getFileActivities(c *gin.Context) {
	fileID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件ID无效"})
		return
	}

	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var activities []FileActivity
	if err := suite.db.Where("file_id = ? AND tenant_id = ?", fileID, tenantID).Order("created_at DESC").Find(&activities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询活动失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"activities": activities,
	})
}

// 工具函数
func (suite *FileServiceIntegrationTestSuite) buildFileResponse(file *File) *FileResponse {
	response := &FileResponse{
		File:          *file,
		FileType:      suite.getFileType(file),
		FormattedSize: suite.formatSize(file.Size),
		CanPreview:    suite.canPreview(file),
		DownloadURL:   fmt.Sprintf("/api/v1/files/%d/download", file.ID),
	}

	if response.CanPreview {
		response.PreviewURL = fmt.Sprintf("/api/v1/files/%d/preview", file.ID)
	}

	return response
}

func (suite *FileServiceIntegrationTestSuite) getFileType(file *File) string {
	if strings.HasPrefix(file.MimeType, "image/") {
		return "image"
	}
	if strings.Contains(file.MimeType, "pdf") || strings.Contains(file.MimeType, "document") || strings.Contains(file.MimeType, "text") {
		return "document"
	}
	return "other"
}

func (suite *FileServiceIntegrationTestSuite) canPreview(file *File) bool {
	return strings.HasPrefix(file.MimeType, "image/") ||
		strings.HasPrefix(file.MimeType, "text/") ||
		file.MimeType == "application/pdf"
}

func (suite *FileServiceIntegrationTestSuite) formatSize(size int64) string {
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

func (suite *FileServiceIntegrationTestSuite) logFileActivity(fileID, userID int, action, details, ipAddress, userAgent, tenantID string) {
	activity := FileActivity{
		TenantID:  tenantID,
		FileID:    fileID,
		UserID:    userID,
		Action:    action,
		Details:   details,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
	}
	suite.db.Create(&activity)
}

// 测试用例

// TestHealthCheck 测试健康检查
func (suite *FileServiceIntegrationTestSuite) TestHealthCheck() {
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), response["success"].(bool))
	assert.Equal(suite.T(), "文件管理服务", response["service"])
	assert.Equal(suite.T(), "healthy", response["status"])
}

// TestFileUpload 测试文件上传
func (suite *FileServiceIntegrationTestSuite) TestFileUpload() {
	// 创建测试文件内容
	fileContent := "This is a test file content"
	
	// 创建multipart表单
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	// 添加表单字段
	writer.WriteField("project_id", "1")
	writer.WriteField("description", "测试文件上传")
	writer.WriteField("tags", "test,upload")
	
	// 添加文件
	part, err := writer.CreateFormFile("files", "test.txt")
	require.NoError(suite.T(), err)
	part.Write([]byte(fileContent))
	
	writer.Close()

	// 创建请求
	req, _ := http.NewRequest("POST", "/api/v1/files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Tenant-ID", "test-tenant")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), response["success"].(bool))
	assert.Equal(suite.T(), float64(1), response["count"])

	files := response["files"].([]interface{})
	assert.Len(suite.T(), files, 1)

	file := files[0].(map[string]interface{})
	assert.Equal(suite.T(), "test.txt", file["original_name"])
	assert.Equal(suite.T(), float64(1), file["project_id"])
	assert.Equal(suite.T(), "测试文件上传", file["description"])
}

// TestFileUploadValidation 测试文件上传验证
func (suite *FileServiceIntegrationTestSuite) TestFileUploadValidation() {
	// 测试缺少必需参数
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("description", "missing project_id")
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/v1/files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), response["error"], "参数错误")
}

// TestFileList 测试文件列表
func (suite *FileServiceIntegrationTestSuite) TestFileList() {
	// 先上传一个文件
	file := File{
		TenantID:     "test-tenant",
		ProjectID:    1,
		Name:         "test_file.txt",
		OriginalName: "test.txt",
		Path:         "/test/path",
		Size:         100,
		MimeType:     "text/plain",
		Extension:    ".txt",
		Hash:         "testhash",
		Status:       "active",
		UploadedBy:   1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	suite.db.Create(&file)

	// 请求文件列表
	req, _ := http.NewRequest("GET", "/api/v1/files/project/1", nil)
	req.Header.Set("X-Tenant-ID", "test-tenant")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), response["success"].(bool))
	files := response["files"].([]interface{})
	assert.Len(suite.T(), files, 1)

	returnedFile := files[0].(map[string]interface{})
	assert.Equal(suite.T(), "test.txt", returnedFile["original_name"])
}

// TestFileDownload 测试文件下载
func (suite *FileServiceIntegrationTestSuite) TestFileDownload() {
	// 创建测试文件
	testContent := "test file content for download"
	testFilePath := filepath.Join(suite.uploadDir, "download_test.txt")
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	require.NoError(suite.T(), err)

	// 创建数据库记录
	file := File{
		TenantID:     "test-tenant",
		ProjectID:    1,
		Name:         "download_test.txt",
		OriginalName: "download.txt",
		Path:         testFilePath,
		Size:         int64(len(testContent)),
		MimeType:     "text/plain",
		Extension:    ".txt",
		Status:       "active",
		UploadedBy:   1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	suite.db.Create(&file)

	// 请求下载
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/files/%d/download", file.ID), nil)
	req.Header.Set("X-Tenant-ID", "test-tenant")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Equal(suite.T(), testContent, w.Body.String())
	assert.Contains(suite.T(), w.Header().Get("Content-Disposition"), "download.txt")
}

// TestFolderCreation 测试文件夹创建
func (suite *FileServiceIntegrationTestSuite) TestFolderCreation() {
	reqBody := CreateFolderRequest{
		ProjectID:   1,
		Name:        "test-folder",
		Description: "测试文件夹",
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/folders", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "test-tenant")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), response["success"].(bool))
	
	folder := response["folder"].(map[string]interface{})
	assert.Equal(suite.T(), "test-folder", folder["name"])
	assert.Equal(suite.T(), "test-folder", folder["path"])
	assert.Equal(suite.T(), float64(0), folder["level"])
}

// TestSubfolderCreation 测试子文件夹创建
func (suite *FileServiceIntegrationTestSuite) TestSubfolderCreation() {
	// 先创建父文件夹
	parentFolder := Folder{
		TenantID:    "test-tenant",
		ProjectID:   1,
		Name:        "parent",
		Path:        "parent",
		Level:       0,
		CreatedBy:   1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	suite.db.Create(&parentFolder)

	// 创建子文件夹
	reqBody := CreateFolderRequest{
		ProjectID:   1,
		Name:        "child",
		ParentID:    &parentFolder.ID,
		Description: "子文件夹",
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/folders", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "test-tenant")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	folder := response["folder"].(map[string]interface{})
	assert.Equal(suite.T(), "child", folder["name"])
	assert.Equal(suite.T(), "parent/child", folder["path"])
	assert.Equal(suite.T(), float64(1), folder["level"])
}

// TestFileActivities 测试文件活动记录
func (suite *FileServiceIntegrationTestSuite) TestFileActivities() {
	// 创建测试文件和活动记录
	file := File{
		TenantID:     "test-tenant",
		ProjectID:    1,
		Name:         "activity_test.txt",
		OriginalName: "activity.txt",
		Status:       "active",
		UploadedBy:   1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	suite.db.Create(&file)

	// 创建活动记录
	activity := FileActivity{
		TenantID:  "test-tenant",
		FileID:    file.ID,
		UserID:    1,
		Action:    "upload",
		Details:   "上传文件: activity.txt",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		CreatedAt: time.Now(),
	}
	suite.db.Create(&activity)

	// 请求活动列表
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/files/%d/activities", file.ID), nil)
	req.Header.Set("X-Tenant-ID", "test-tenant")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), response["success"].(bool))
	activities := response["activities"].([]interface{})
	assert.Len(suite.T(), activities, 1)

	returnedActivity := activities[0].(map[string]interface{})
	assert.Equal(suite.T(), "upload", returnedActivity["action"])
	assert.Contains(suite.T(), returnedActivity["details"], "activity.txt")
}

// TestFileDelete 测试文件删除
func (suite *FileServiceIntegrationTestSuite) TestFileDelete() {
	// 创建测试文件
	file := File{
		TenantID:     "test-tenant",
		ProjectID:    1,
		Name:         "delete_test.txt",
		OriginalName: "delete.txt",
		Status:       "active",
		UploadedBy:   1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	suite.db.Create(&file)

	// 删除文件
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/files/%d", file.ID), nil)
	req.Header.Set("X-Tenant-ID", "test-tenant")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), response["success"].(bool))

	// 验证文件状态已更新
	var updatedFile File
	suite.db.First(&updatedFile, file.ID)
	assert.Equal(suite.T(), "deleted", updatedFile.Status)
}

// TestConcurrentFileOperations 测试并发文件操作
func (suite *FileServiceIntegrationTestSuite) TestConcurrentFileOperations() {
	const numConcurrent = 10
	results := make(chan error, numConcurrent)

	// 并发上传文件
	for i := 0; i < numConcurrent; i++ {
		go func(index int) {
			fileContent := fmt.Sprintf("Content for file %d", index)
			
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			writer.WriteField("project_id", "1")
			writer.WriteField("description", fmt.Sprintf("并发测试文件 %d", index))
			
			part, _ := writer.CreateFormFile("files", fmt.Sprintf("concurrent_%d.txt", index))
			part.Write([]byte(fileContent))
			writer.Close()

			req, _ := http.NewRequest("POST", "/api/v1/files/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("X-Tenant-ID", "concurrent-test")

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				results <- fmt.Errorf("upload %d failed with status %d", index, w.Code)
			} else {
				results <- nil
			}
		}(i)
	}

	// 等待所有操作完成
	for i := 0; i < numConcurrent; i++ {
		err := <-results
		assert.NoError(suite.T(), err)
	}

	// 验证所有文件都已上传
	var count int64
	suite.db.Model(&File{}).Where("tenant_id = ?", "concurrent-test").Count(&count)
	assert.Equal(suite.T(), int64(numConcurrent), count)
}

// TestTenantIsolation 测试租户隔离
func (suite *FileServiceIntegrationTestSuite) TestTenantIsolation() {
	// 为不同租户创建文件
	file1 := File{
		TenantID:     "tenant-1",
		ProjectID:    1,
		Name:         "tenant1_file.txt",
		OriginalName: "file1.txt",
		Status:       "active",
		UploadedBy:   1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	suite.db.Create(&file1)

	file2 := File{
		TenantID:     "tenant-2",
		ProjectID:    1,
		Name:         "tenant2_file.txt",
		OriginalName: "file2.txt",
		Status:       "active",
		UploadedBy:   1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	suite.db.Create(&file2)

	// tenant-1 只能看到自己的文件
	req, _ := http.NewRequest("GET", "/api/v1/files/project/1", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	files := response["files"].([]interface{})
	assert.Len(suite.T(), files, 1)
	assert.Equal(suite.T(), "file1.txt", files[0].(map[string]interface{})["original_name"])
}

// TestErrorHandling 测试错误处理
func (suite *FileServiceIntegrationTestSuite) TestErrorHandling() {
	// 测试不存在的文件下载
	req, _ := http.NewRequest("GET", "/api/v1/files/99999/download", nil)
	req.Header.Set("X-Tenant-ID", "test-tenant")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), response["error"], "文件不存在")
}

// 运行测试套件
func TestFileServiceIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(FileServiceIntegrationTestSuite))
}

// 性能测试
func BenchmarkFileUpload(b *testing.B) {
	// 设置测试环境
	gin.SetMode(gin.TestMode)
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&File{}, &Folder{}, &FileActivity{})

	uploadDir := "./bench_uploads"
	os.MkdirAll(uploadDir, 0755)
	defer os.RemoveAll(uploadDir)

	suite := &FileServiceIntegrationTestSuite{
		db:        db,
		uploadDir: uploadDir,
	}
	suite.setupRouter()

	fileContent := "Benchmark test file content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("project_id", "1")
		writer.WriteField("description", "性能测试文件")
		
		part, _ := writer.CreateFormFile("files", fmt.Sprintf("bench_%d.txt", i))
		part.Write([]byte(fileContent))
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/v1/files/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-Tenant-ID", "benchmark")

		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Upload failed with status %d", w.Code)
		}
	}
}
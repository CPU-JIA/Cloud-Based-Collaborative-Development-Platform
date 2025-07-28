package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 简化的数据模型（用于文件服务）
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

// 请求和响应结构
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

var db *gorm.DB
var uploadDir = "./uploads"

func main() {
	// 初始化数据库
	initDB()

	// 创建上传目录
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatal("无法创建上传目录:", err)
	}

	// 初始化Gin路由
	r := gin.Default()

	// CORS配置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:3002"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Tenant-ID"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 设置静态文件服务
	r.Static("/uploads", uploadDir)

	// API路由组
	api := r.Group("/api/v1")
	{
		// 健康检查
		api.GET("/health", healthCheck)

		// 文件管理
		files := api.Group("/files")
		{
			files.POST("/upload", uploadFiles)
			files.GET("/project/:projectId", listFiles)
			files.GET("/:id", getFile)
			files.GET("/:id/download", downloadFile)
			files.GET("/:id/preview", previewFile)
			files.PUT("/:id", updateFile)
			files.DELETE("/:id", deleteFile)
			files.POST("/:id/share", shareFile)
			files.GET("/share/:token", getSharedFile)
		}

		// 文件夹管理
		folders := api.Group("/folders")
		{
			folders.POST("", createFolder)
			folders.GET("/project/:projectId", listFolders)
			folders.GET("/:id", getFolder)
			folders.PUT("/:id", updateFolder)
			folders.DELETE("/:id", deleteFolder)
		}

		// 文件活动
		api.GET("/files/:id/activities", getFileActivities)
		api.GET("/projects/:projectId/activities", getProjectFileActivities)
	}

	log.Println("🚀 文件管理服务启动成功！")
	log.Println("📁 文件上传目录:", uploadDir)
	log.Println("🌐 服务地址: http://localhost:8085")
	log.Println("🔍 健康检查: http://localhost:8085/api/v1/health")

	r.Run(":8085")
}

func initDB() {
	dsn := "host=localhost user=postgres password=123456 dbname=postgres port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 自动迁移
	db.AutoMigrate(&File{}, &Folder{}, &FileActivity{})
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"service":    "文件管理服务",
		"version":    "1.0.0",
		"status":     "healthy",
		"uptime":     time.Since(time.Now().Add(-time.Hour)).String(),
		"upload_dir": uploadDir,
	})
}

func uploadFiles(c *gin.Context) {
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
	projectUploadDir := filepath.Join(uploadDir, tenantID, fmt.Sprintf("project_%d", req.ProjectID))
	if err := os.MkdirAll(projectUploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目录失败"})
		return
	}

	for _, fileHeader := range files {
		file, err := processFileUpload(fileHeader, req, tenantID, projectUploadDir)
		if err != nil {
			log.Printf("文件上传失败 %s: %v", fileHeader.Filename, err)
			continue
		}

		uploadedFiles = append(uploadedFiles, *file)

		// 记录活动
		logFileActivity(file.ID, 1, "upload", fmt.Sprintf("上传文件: %s", file.OriginalName), c.ClientIP(), c.GetHeader("User-Agent"), tenantID)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"files":   uploadedFiles,
		"count":   len(uploadedFiles),
	})
}

func processFileUpload(fileHeader *multipart.FileHeader, req UploadRequest, tenantID, uploadDir string) (*FileResponse, error) {
	// 打开上传的文件
	src, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// 计算文件哈希
	hash := md5.New()
	if _, err := io.Copy(hash, src); err != nil {
		return nil, err
	}
	fileHash := fmt.Sprintf("%x", hash.Sum(nil))

	// 重新定位到文件开头
	src.Seek(0, 0)

	// 生成唯一文件名
	ext := filepath.Ext(fileHeader.Filename)
	fileName := fmt.Sprintf("%s_%s%s", uuid.New().String(), strconv.FormatInt(time.Now().Unix(), 10), ext)
	filePath := filepath.Join(uploadDir, fileName)

	// 保存文件
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, err
	}

	// 解析标签
	var tags []string
	if req.Tags != "" {
		tags = strings.Split(req.Tags, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
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
		Hash:         fileHash,
		FolderID:     req.FolderID,
		Tags:         tags,
		Description:  req.Description,
		UploadedBy:   1, // 临时硬编码用户ID
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.Create(&file).Error; err != nil {
		os.Remove(filePath) // 清理文件
		return nil, err
	}

	return buildFileResponse(&file), nil
}

func listFiles(c *gin.Context) {
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
	query := db.Where("tenant_id = ? AND project_id = ? AND status = ?", tenantID, projectID, "active")

	// 文件夹过滤
	if folderID := c.Query("folder_id"); folderID != "" {
		if folderID == "null" || folderID == "0" {
			query = query.Where("folder_id IS NULL")
		} else {
			query = query.Where("folder_id = ?", folderID)
		}
	}

	// 搜索过滤
	if search := c.Query("search"); search != "" {
		query = query.Where("original_name ILIKE ?", "%"+search+"%")
	}

	// 文件类型过滤
	if fileType := c.Query("type"); fileType != "" {
		switch fileType {
		case "image":
			query = query.Where("mime_type LIKE 'image/%'")
		case "document":
			query = query.Where("mime_type IN (?)", []string{"application/pdf", "application/msword", "text/plain"})
		case "code":
			query = query.Where("extension IN (?)", []string{".go", ".js", ".ts", ".py", ".java"})
		}
	}

	// 排序
	order := c.DefaultQuery("order", "created_at DESC")
	query = query.Order(order)

	// 分页
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	offset := (page - 1) * pageSize

	var total int64
	query.Count(&total)

	if err := query.Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询文件失败"})
		return
	}

	var fileResponses []FileResponse
	for _, file := range files {
		fileResponses = append(fileResponses, *buildFileResponse(&file))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"files":     fileResponses,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func downloadFile(c *gin.Context) {
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
	if err := db.Where("id = ? AND tenant_id = ? AND status = ?", fileID, tenantID, "active").First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 更新下载计数
	db.Model(&file).Update("download_count", file.DownloadCount+1)

	// 记录活动
	logFileActivity(file.ID, 1, "download", fmt.Sprintf("下载文件: %s", file.OriginalName), c.ClientIP(), c.GetHeader("User-Agent"), tenantID)

	// 设置响应头
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.OriginalName))
	c.Header("Content-Type", file.MimeType)
	c.File(file.Path)
}

func previewFile(c *gin.Context) {
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
	if err := db.Where("id = ? AND tenant_id = ? AND status = ?", fileID, tenantID, "active").First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 检查是否可预览
	if !canPreview(&file) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该文件类型不支持预览"})
		return
	}

	// 设置适当的Content-Type
	c.Header("Content-Type", file.MimeType)
	c.File(file.Path)
}

func createFolder(c *gin.Context) {
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
		if err := db.Where("id = ? AND tenant_id = ?", *req.ParentID, tenantID).First(&parent).Error; err != nil {
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
		CreatedBy:   1, // 临时硬编码
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := db.Create(&folder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建文件夹失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"folder":  folder,
	})
}

func listFolders(c *gin.Context) {
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
	query := db.Where("tenant_id = ? AND project_id = ?", tenantID, projectID)

	// 父文件夹过滤
	if parentID := c.Query("parent_id"); parentID != "" {
		if parentID == "null" || parentID == "0" {
			query = query.Where("parent_id IS NULL")
		} else {
			query = query.Where("parent_id = ?", parentID)
		}
	}

	if err := query.Order("name ASC").Find(&folders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询文件夹失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"folders": folders,
	})
}

// 工具函数
func buildFileResponse(file *File) *FileResponse {
	response := &FileResponse{
		File:          *file,
		FileType:      getFileType(file),
		FormattedSize: formatSize(file.Size),
		CanPreview:    canPreview(file),
		DownloadURL:   fmt.Sprintf("/api/v1/files/%d/download", file.ID),
	}

	if response.CanPreview {
		response.PreviewURL = fmt.Sprintf("/api/v1/files/%d/preview", file.ID)
	}

	return response
}

func getFileType(file *File) string {
	if strings.HasPrefix(file.MimeType, "image/") {
		return "image"
	}
	if strings.Contains(file.MimeType, "pdf") || strings.Contains(file.MimeType, "document") || strings.Contains(file.MimeType, "text") {
		return "document"
	}
	codeExts := []string{".go", ".js", ".ts", ".py", ".java", ".cpp", ".c", ".h", ".css", ".html", ".json", ".xml", ".yaml", ".yml"}
	for _, ext := range codeExts {
		if file.Extension == ext {
			return "code"
		}
	}
	return "other"
}

func canPreview(file *File) bool {
	// 图片文件可预览
	if strings.HasPrefix(file.MimeType, "image/") {
		return true
	}
	// 文本文件可预览
	if strings.HasPrefix(file.MimeType, "text/") {
		return true
	}
	// PDF文件可预览
	if file.MimeType == "application/pdf" {
		return true
	}
	return false
}

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

func logFileActivity(fileID, userID int, action, details, ipAddress, userAgent, tenantID string) {
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
	db.Create(&activity)
}

// 其他API处理函数的占位符
func getFile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "获取文件详情"})
}

func updateFile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "更新文件"})
}

func deleteFile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "删除文件"})
}

func shareFile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "分享文件"})
}

func getSharedFile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "获取分享文件"})
}

func getFolder(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "获取文件夹"})
}

func updateFolder(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "更新文件夹"})
}

func deleteFolder(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "删除文件夹"})
}

func getFileActivities(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "获取文件活动"})
}

func getProjectFileActivities(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "获取项目文件活动"})
}

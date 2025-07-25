package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 简化的文件模型（内存存储）
type File struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	OriginalName string    `json:"original_name"`
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	MimeType     string    `json:"mime_type"`
	Extension    string    `json:"extension"`
	FileType     string    `json:"file_type"`
	FormattedSize string   `json:"formatted_size"`
	CanPreview   bool      `json:"can_preview"`
	PreviewURL   string    `json:"preview_url,omitempty"`
	DownloadURL  string    `json:"download_url"`
	ShareURL     string    `json:"share_url,omitempty"`
	FolderID     *int      `json:"folder_id"`
	Tags         []string  `json:"tags"`
	Description  string    `json:"description"`
	UploadedBy   int       `json:"uploaded_by"`
	DownloadCount int      `json:"download_count"`
	IsShared     bool      `json:"is_shared"`
	ShareToken   string    `json:"share_token,omitempty"`
	SharePassword string   `json:"share_password,omitempty"`
	ShareExpires *time.Time `json:"share_expires,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Folder struct {
	ID          int       `json:"id"`
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

// 内存存储
var files []File
var folders []Folder
var shareLinks = make(map[string]*File) // 分享链接映射
var fileIDCounter = 1
var folderIDCounter = 1
var uploadDir = "./uploads"

func main() {
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
			files.POST("/:id/share", createShareLink)
			files.DELETE("/:id/share", revokeShareLink)
		}
		
		// 公共分享访问
		api.GET("/share/:token", getSharedFile)
		api.GET("/share/:token/download", downloadSharedFile)
		api.POST("/share/:token/verify", verifySharePassword)
		
		// 文件夹管理
		api.POST("/folders", createFolder)
		api.GET("/folders/project/:projectId", listFolders)
	}
	
	log.Println("🚀 模拟文件管理服务启动成功！")
	log.Println("📁 文件上传目录:", uploadDir)
	log.Println("🌐 服务地址: http://localhost:8085")
	log.Println("🔍 健康检查: http://localhost:8085/api/v1/health")
	
	r.Run(":8085")
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"service":    "模拟文件管理服务",
		"version":    "1.0.0",
		"status":     "healthy",
		"upload_dir": uploadDir,
		"files_count": len(files),
		"folders_count": len(folders),
	})
}

func uploadFiles(c *gin.Context) {
	projectIDStr := c.PostForm("project_id")
	folderIDStr := c.PostForm("folder_id")
	description := c.PostForm("description")
	tags := c.PostForm("tags")
	
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目ID无效"})
		return
	}
	
	var folderID *int
	if folderIDStr != "" && folderIDStr != "null" {
		if fid, err := strconv.Atoi(folderIDStr); err == nil {
			folderID = &fid
		}
	}
	
	// 获取上传的文件
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取文件失败: " + err.Error()})
		return
	}
	
	uploadedFiles := form.File["files"]
	if len(uploadedFiles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未选择文件"})
		return
	}
	
	var savedFiles []File
	
	// 创建项目上传目录
	projectUploadDir := filepath.Join(uploadDir, fmt.Sprintf("project_%d", projectID))
	if err := os.MkdirAll(projectUploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目录失败"})
		return
	}
	
	for _, fileHeader := range uploadedFiles {
		// 打开上传的文件
		src, err := fileHeader.Open()
		if err != nil {
			log.Printf("打开文件失败 %s: %v", fileHeader.Filename, err)
			continue
		}
		defer src.Close()
		
		// 生成唯一文件名
		ext := filepath.Ext(fileHeader.Filename)
		fileName := fmt.Sprintf("%s_%s%s", uuid.New().String(), strconv.FormatInt(time.Now().Unix(), 10), ext)
		filePath := filepath.Join(projectUploadDir, fileName)
		
		// 保存文件
		dst, err := os.Create(filePath)
		if err != nil {
			log.Printf("创建文件失败 %s: %v", filePath, err)
			continue
		}
		
		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			log.Printf("复制文件失败 %s: %v", filePath, err)
			continue
		}
		dst.Close()
		
		// 解析标签
		var fileTags []string
		if tags != "" {
			fileTags = strings.Split(tags, ",")
			for i, tag := range fileTags {
				fileTags[i] = strings.TrimSpace(tag)
			}
		}
		
		// 创建文件记录
		file := File{
			ID:           fileIDCounter,
			Name:         fileName,
			OriginalName: fileHeader.Filename,
			Path:         filePath,
			Size:         fileHeader.Size,
			MimeType:     fileHeader.Header.Get("Content-Type"),
			Extension:    ext,
			FileType:     getFileType(fileHeader.Header.Get("Content-Type"), ext),
			FormattedSize: formatSize(fileHeader.Size),
			CanPreview:   canPreview(fileHeader.Header.Get("Content-Type")),
			DownloadURL:  fmt.Sprintf("/api/v1/files/%d/download", fileIDCounter),
			FolderID:     folderID,
			Tags:         fileTags,
			Description:  description,
			UploadedBy:   1,
			DownloadCount: 0,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		
		if file.CanPreview {
			file.PreviewURL = fmt.Sprintf("/api/v1/files/%d/preview", fileIDCounter)
		}
		
		files = append(files, file)
		savedFiles = append(savedFiles, file)
		fileIDCounter++
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"files":   savedFiles,
		"count":   len(savedFiles),
	})
}

func listFiles(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("projectId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目ID无效"})
		return
	}
	
	folderIDStr := c.Query("folder_id")
	search := c.Query("search")
	fileType := c.Query("type")
	
	var filteredFiles []File
	for _, file := range files {
		// 项目过滤（这里简化处理，实际应该根据项目ID过滤）
		_ = projectID
		
		// 文件夹过滤
		if folderIDStr != "" {
			if folderIDStr == "null" {
				if file.FolderID != nil {
					continue
				}
			} else if fid, err := strconv.Atoi(folderIDStr); err == nil {
				if file.FolderID == nil || *file.FolderID != fid {
					continue
				}
			}
		}
		
		// 搜索过滤
		if search != "" && !strings.Contains(strings.ToLower(file.OriginalName), strings.ToLower(search)) {
			continue
		}
		
		// 类型过滤
		if fileType != "" && fileType != "all" && file.FileType != fileType {
			continue
		}
		
		filteredFiles = append(filteredFiles, file)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"files":     filteredFiles,
		"total":     len(filteredFiles),
		"page":      1,
		"page_size": len(filteredFiles),
	})
}

func downloadFile(c *gin.Context) {
	fileID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件ID无效"})
		return
	}
	
	// 查找文件
	var file *File
	for i := range files {
		if files[i].ID == fileID {
			file = &files[i]
			break
		}
	}
	
	if file == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	
	// 更新下载计数
	file.DownloadCount++
	
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
	
	// 查找文件
	var file *File
	for i := range files {
		if files[i].ID == fileID {
			file = &files[i]
			break
		}
	}
	
	if file == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	
	if !file.CanPreview {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该文件类型不支持预览"})
		return
	}
	
	// 设置适当的Content-Type
	c.Header("Content-Type", file.MimeType)
	c.File(file.Path)
}

func createFolder(c *gin.Context) {
	var req struct {
		ProjectID   int    `json:"project_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		ParentID    *int   `json:"parent_id"`
		Description string `json:"description"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	
	// 构建路径
	path := req.Name
	level := 0
	if req.ParentID != nil {
		// 查找父文件夹
		for _, folder := range folders {
			if folder.ID == *req.ParentID {
				path = folder.Path + "/" + req.Name
				level = folder.Level + 1
				break
			}
		}
	}
	
	folder := Folder{
		ID:          folderIDCounter,
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
	
	folders = append(folders, folder)
	folderIDCounter++
	
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
	
	parentIDStr := c.Query("parent_id")
	
	var filteredFolders []Folder
	for _, folder := range folders {
		// 项目过滤（这里简化处理）
		_ = projectID
		
		// 父文件夹过滤
		if parentIDStr != "" {
			if parentIDStr == "null" {
				if folder.ParentID != nil {
					continue
				}
			} else if pid, err := strconv.Atoi(parentIDStr); err == nil {
				if folder.ParentID == nil || *folder.ParentID != pid {
					continue
				}
			}
		}
		
		filteredFolders = append(filteredFolders, folder)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"folders": filteredFolders,
	})
}

func getFile(c *gin.Context) {
	fileID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件ID无效"})
		return
	}
	
	// 查找文件
	for _, file := range files {
		if file.ID == fileID {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"file":    file,
			})
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
}

// 工具函数
func getFileType(mimeType, extension string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	if strings.Contains(mimeType, "pdf") || strings.Contains(mimeType, "document") || strings.Contains(mimeType, "text") || mimeType == "application/json" {
		return "document"
	}
	codeExts := []string{".go", ".js", ".ts", ".py", ".java", ".cpp", ".c", ".h", ".css", ".html", ".json", ".xml", ".yaml", ".yml", ".sql", ".sh", ".php", ".rb", ".rs", ".swift"}
	for _, ext := range codeExts {
		if extension == ext {
			return "code"
		}
	}
	return "other"
}

func canPreview(mimeType string) bool {
	// 图片文件可预览
	if strings.HasPrefix(mimeType, "image/") {
		return true
	}
	// 文本文件可预览
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	// PDF文件可预览
	if mimeType == "application/pdf" {
		return true
	}
	// JSON文件可预览
	if mimeType == "application/json" {
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

// 分享相关功能
func createShareLink(c *gin.Context) {
	fileID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件ID无效"})
		return
	}
	
	// 解析请求参数
	var req struct {
		Password   string    `json:"password"`
		ExpiresAt  *string   `json:"expires_at"`
		Permission string    `json:"permission"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	
	// 查找文件
	var file *File
	for i := range files {
		if files[i].ID == fileID {
			file = &files[i]
			break
		}
	}
	
	if file == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	
	// 生成分享令牌
	shareToken := uuid.New().String()
	
	// 处理过期时间
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, *req.ExpiresAt); err == nil {
			expiresAt = &t
		}
	}
	
	// 更新文件分享信息
	file.IsShared = true
	file.ShareToken = shareToken
	file.SharePassword = req.Password
	file.ShareExpires = expiresAt
	file.ShareURL = fmt.Sprintf("/api/v1/share/%s", shareToken)
	file.UpdatedAt = time.Now()
	
	// 添加到分享链接映射
	shareLinks[shareToken] = file
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"share": gin.H{
			"token":       shareToken,
			"share_url":   fmt.Sprintf("/api/v1/share/%s", shareToken),
			"download_url": fmt.Sprintf("/api/v1/share/%s/download", shareToken),
			"password":    req.Password != "",
			"expires_at":  expiresAt,
			"permission":  req.Permission,
		},
	})
}

func revokeShareLink(c *gin.Context) {
	fileID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件ID无效"})
		return
	}
	
	// 查找文件
	var file *File
	for i := range files {
		if files[i].ID == fileID {
			file = &files[i]
			break
		}
	}
	
	if file == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	
	// 移除分享
	if file.ShareToken != "" {
		delete(shareLinks, file.ShareToken)
	}
	
	file.IsShared = false
	file.ShareToken = ""
	file.SharePassword = ""
	file.ShareExpires = nil
	file.ShareURL = ""
	file.UpdatedAt = time.Now()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "分享已撤销",
	})
}

func getSharedFile(c *gin.Context) {
	token := c.Param("token")
	
	file, exists := shareLinks[token]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享链接不存在或已失效"})
		return
	}
	
	// 检查过期时间
	if file.ShareExpires != nil && time.Now().After(*file.ShareExpires) {
		c.JSON(http.StatusGone, gin.H{"error": "分享链接已过期"})
		return
	}
	
	// 如果有密码保护，需要先验证密码
	if file.SharePassword != "" {
		password := c.Query("password")
		if password != file.SharePassword {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "需要密码验证",
				"password_required": true,
			})
			return
		}
	}
	
	// 返回文件信息（不包含敏感信息）
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"file": gin.H{
			"id":            file.ID,
			"original_name": file.OriginalName,
			"size":         file.Size,
			"formatted_size": file.FormattedSize,
			"mime_type":    file.MimeType,
			"file_type":    file.FileType,
			"can_preview":  file.CanPreview,
			"preview_url":  fmt.Sprintf("/api/v1/share/%s/preview", token),
			"download_url": fmt.Sprintf("/api/v1/share/%s/download", token),
			"created_at":   file.CreatedAt,
		},
		"share": gin.H{
			"token":        token,
			"password_required": file.SharePassword != "",
			"expires_at":   file.ShareExpires,
		},
	})
}

func downloadSharedFile(c *gin.Context) {
	token := c.Param("token")
	
	file, exists := shareLinks[token]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享链接不存在或已失效"})
		return
	}
	
	// 检查过期时间
	if file.ShareExpires != nil && time.Now().After(*file.ShareExpires) {
		c.JSON(http.StatusGone, gin.H{"error": "分享链接已过期"})
		return
	}
	
	// 检查密码
	if file.SharePassword != "" {
		password := c.Query("password")
		if password != file.SharePassword {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
			return
		}
	}
	
	// 更新下载计数
	file.DownloadCount++
	
	// 设置响应头
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.OriginalName))
	c.Header("Content-Type", file.MimeType)
	c.File(file.Path)
}

func verifySharePassword(c *gin.Context) {
	token := c.Param("token")
	
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码不能为空"})
		return
	}
	
	file, exists := shareLinks[token]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享链接不存在或已失效"})
		return
	}
	
	// 检查过期时间
	if file.ShareExpires != nil && time.Now().After(*file.ShareExpires) {
		c.JSON(http.StatusGone, gin.H{"error": "分享链接已过期"})
		return
	}
	
	// 验证密码
	if file.SharePassword != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": "密码错误",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "密码验证成功",
	})
}
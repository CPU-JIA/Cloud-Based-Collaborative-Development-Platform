package models

import (
	"fmt"
	"time"
)

// File 文件模型
type File struct {
	ID          int       `json:"id" gorm:"primary_key"`
	TenantID    string    `json:"tenant_id" gorm:"not null;index"`
	ProjectID   int       `json:"project_id" gorm:"not null;index"`
	Name        string    `json:"name" gorm:"not null"`
	OriginalName string   `json:"original_name" gorm:"not null"`
	Path        string    `json:"path" gorm:"not null"`
	Size        int64     `json:"size" gorm:"not null"`
	MimeType    string    `json:"mime_type" gorm:"not null"`
	Extension   string    `json:"extension"`
	Hash        string    `json:"hash" gorm:"unique_index"`
	
	// 文件分类和组织
	FolderID    *int      `json:"folder_id" gorm:"index"`
	Tags        []string  `json:"tags" gorm:"type:json"`
	Description string    `json:"description"`
	
	// 版本控制
	Version     int       `json:"version" gorm:"default:1"`
	ParentFileID *int     `json:"parent_file_id" gorm:"index"`
	IsLatest    bool      `json:"is_latest" gorm:"default:true"`
	
	// 权限和访问控制
	IsPublic    bool      `json:"is_public" gorm:"default:false"`
	ShareToken  string    `json:"share_token" gorm:"unique_index"`
	ShareExpiry *time.Time `json:"share_expiry"`
	
	// 状态信息
	Status      string    `json:"status" gorm:"default:'active'"` // active, deleted, archived
	UploadedBy  int       `json:"uploaded_by" gorm:"not null;index"`
	DownloadCount int     `json:"download_count" gorm:"default:0"`
	
	// 时间戳
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at" gorm:"index"`
	
	// 关联
	// Project关联暂时省略，避免循环依赖
	Folder      *Folder   `json:"folder,omitempty" gorm:"foreignkey:FolderID"`
	Uploader    User      `json:"uploader,omitempty" gorm:"foreignkey:UploadedBy"`
	ParentFile  *File     `json:"parent_file,omitempty" gorm:"foreignkey:ParentFileID"`
	Versions    []File    `json:"versions,omitempty" gorm:"foreignkey:ParentFileID"`
}

// Folder 文件夹模型
type Folder struct {
	ID          int       `json:"id" gorm:"primary_key"`
	TenantID    string    `json:"tenant_id" gorm:"not null;index"`
	ProjectID   int       `json:"project_id" gorm:"not null;index"`
	Name        string    `json:"name" gorm:"not null"`
	Path        string    `json:"path" gorm:"not null"`
	Description string    `json:"description"`
	
	// 层级结构
	ParentID    *int      `json:"parent_id" gorm:"index"`
	Level       int       `json:"level" gorm:"default:0"`
	SortOrder   int       `json:"sort_order" gorm:"default:0"`
	
	// 权限
	IsPublic    bool      `json:"is_public" gorm:"default:false"`
	CreatedBy   int       `json:"created_by" gorm:"not null;index"`
	
	// 时间戳
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at" gorm:"index"`
	
	// 关联
	// Project关联暂时省略，避免循环依赖
	Parent      *Folder   `json:"parent,omitempty" gorm:"foreignkey:ParentID"`
	Children    []Folder  `json:"children,omitempty" gorm:"foreignkey:ParentID"`
	Files       []File    `json:"files,omitempty" gorm:"foreignkey:FolderID"`
	Creator     User      `json:"creator,omitempty" gorm:"foreignkey:CreatedBy"`
}

// 注意：FilePermission模型已在permission.go中定义

// FileActivity 文件活动记录
type FileActivity struct {
	ID          int       `json:"id" gorm:"primary_key"`
	TenantID    string    `json:"tenant_id" gorm:"not null;index"`
	FileID      int       `json:"file_id" gorm:"not null;index"`
	UserID      int       `json:"user_id" gorm:"not null;index"`
	Action      string    `json:"action" gorm:"not null"` // upload, download, delete, share, rename, move
	Details     string    `json:"details"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	CreatedAt   time.Time `json:"created_at"`
	
	// 关联
	File        File      `json:"file,omitempty" gorm:"foreignkey:FileID"`
	User        User      `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// TableName 设置表名
func (File) TableName() string {
	return "files"
}

func (Folder) TableName() string {
	return "folders"
}

func (FilePermission) TableName() string {
	return "file_permissions"
}

func (FileActivity) TableName() string {
	return "file_activities"
}

// 文件扩展方法

// IsImage 判断是否为图片文件
func (f *File) IsImage() bool {
	imageTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp", "image/svg+xml"}
	for _, t := range imageTypes {
		if f.MimeType == t {
			return true
		}
	}
	return false
}

// IsDocument 判断是否为文档文件
func (f *File) IsDocument() bool {
	docTypes := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"text/plain",
		"text/markdown",
	}
	for _, t := range docTypes {
		if f.MimeType == t {
			return true
		}
	}
	return false
}

// IsCode 判断是否为代码文件
func (f *File) IsCode() bool {
	codeExts := []string{".go", ".js", ".ts", ".py", ".java", ".cpp", ".c", ".h", ".css", ".html", ".json", ".xml", ".yaml", ".yml", ".sql", ".sh", ".php", ".rb", ".rs", ".swift"}
	for _, ext := range codeExts {
		if f.Extension == ext {
			return true
		}
	}
	return false
}

// GetFileType 获取文件类型
func (f *File) GetFileType() string {
	if f.IsImage() {
		return "image"
	}
	if f.IsDocument() {
		return "document"
	}
	if f.IsCode() {
		return "code"
	}
	return "other"
}

// FormatSize 格式化文件大小
func (f *File) FormatSize() string {
	size := float64(f.Size)
	units := []string{"B", "KB", "MB", "GB", "TB"}
	
	for i, unit := range units {
		if size < 1024 || i == len(units)-1 {
			if i == 0 {
				return fmt.Sprintf("%.0f %s", size, unit)
			}
			return fmt.Sprintf("%.1f %s", size, unit)
		}
		size /= 1024
	}
	return fmt.Sprintf("%.1f %s", size, units[len(units)-1])
}

// GetFullPath 获取完整路径
func (f *Folder) GetFullPath() string {
	if f.Parent != nil {
		return f.Parent.GetFullPath() + "/" + f.Name
	}
	return f.Name
}
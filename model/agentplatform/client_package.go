package agentplatform

import "time"

const (
	ClientPackagePlatformWindowsX64 = "windows_x64"
	ClientPackagePlatformMacArm64   = "mac_arm64"
	ClientPackagePlatformMacX64     = "mac_x64"

	ClientPackageStatusDraft     = "draft"
	ClientPackageStatusPublished = "published"
	ClientPackageStatusDisabled  = "disabled"

	ClientPackageRolloutModeGlobal   = "global"
	ClientPackageRolloutModeTargeted = "targeted"
)

type ClientPackage struct {
	Id                     int        `json:"id" gorm:"primaryKey"`
	Platform               string     `json:"platform" gorm:"type:varchar(32);not null;index:idx_agent_platform_client_package_platform_status"`
	Version                string     `json:"version" gorm:"type:varchar(32);not null;index:idx_agent_platform_client_package_platform_version"`
	Status                 string     `json:"status" gorm:"type:varchar(32);not null;index:idx_agent_platform_client_package_platform_status"`
	RolloutMode            string     `json:"rollout_mode" gorm:"type:varchar(16);not null;default:'global';index:idx_agent_platform_client_package_rollout_mode"`
	FileName               string     `json:"file_name" gorm:"type:varchar(255);not null"`
	FilePath               string     `json:"file_path" gorm:"type:text;not null"`
	FileSha256             string     `json:"file_sha256" gorm:"type:char(64);not null"`
	FileSha512             string     `json:"file_sha512" gorm:"type:varchar(128);not null"`
	FileSize               int64      `json:"file_size" gorm:"type:bigint;not null"`
	ContentType            string     `json:"content_type" gorm:"type:varchar(128);not null"`
	UpdateFileName         string     `json:"update_file_name" gorm:"type:varchar(255);not null;default:''"`
	UpdateFilePath         string     `json:"update_file_path" gorm:"type:text"`
	UpdateFileSha256       string     `json:"update_file_sha256" gorm:"type:char(64);not null;default:''"`
	UpdateFileSha512       string     `json:"update_file_sha512" gorm:"type:varchar(128);not null;default:''"`
	UpdateFileSize         int64      `json:"update_file_size" gorm:"type:bigint;not null;default:0"`
	UpdateContentType      string     `json:"update_content_type" gorm:"type:varchar(128);not null;default:''"`
	UpdateMetadataFileName string     `json:"update_metadata_file_name" gorm:"type:varchar(255);not null;default:''"`
	UpdateMetadataFilePath string     `json:"update_metadata_file_path" gorm:"type:text"`
	UpdateMetadataSha256   string     `json:"update_metadata_sha256" gorm:"type:char(64);not null;default:''"`
	ReleaseNote            string     `json:"release_note" gorm:"type:text"`
	CreatedBy              int        `json:"created_by" gorm:"type:int;not null;index:idx_agent_platform_client_package_created_by"`
	PublishedAt            *time.Time `json:"published_at"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

func (ClientPackage) TableName() string {
	return "agent_platform_client_packages"
}

type ClientPackageScope struct {
	Id              int       `json:"id" gorm:"primaryKey"`
	ClientPackageId int       `json:"client_package_id" gorm:"not null;index:idx_agent_platform_client_package_scope_package_subject,priority:1;uniqueIndex:uq_agent_platform_client_package_scope"`
	SubjectType     string    `json:"subject_type" gorm:"type:varchar(32);not null;index:idx_agent_platform_client_package_scope_package_subject,priority:2;uniqueIndex:uq_agent_platform_client_package_scope"`
	SubjectId       string    `json:"subject_id" gorm:"type:varchar(128);not null;index:idx_agent_platform_client_package_scope_package_subject,priority:3;uniqueIndex:uq_agent_platform_client_package_scope"`
	CreatedBy       int       `json:"created_by" gorm:"not null;index"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (ClientPackageScope) TableName() string {
	return "agent_platform_client_package_scopes"
}
